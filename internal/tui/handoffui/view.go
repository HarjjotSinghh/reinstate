// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package handoffui

import (
	"fmt"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	inner := width - 2
	if inner < 24 {
		inner = 24
	}

	var lines []string
	add := func(text string) { lines = append(lines, " "+text) }

	// Header: source, arrow, destination selector.
	add(m.theme.Title.Render("hand off") + "  " +
		m.theme.Agent(m.sourceAgent, 0) + " " +
		m.theme.Muted.Render(ui.Truncate(m.reference, inner-30, m.theme.Glyphs.Ellipsis)))
	add(m.destinationLine())
	add("")

	// Policy selector plus the live measurement.
	add(m.policyLine(inner))
	add("   " + m.theme.Muted.Render(ui.Truncate(policyBlurb[m.Policy()], inner-3, m.theme.Glyphs.Ellipsis)))
	add("")

	preview, ready := m.current()
	switch {
	case !ready:
		add(m.theme.Pending.Render(m.theme.Glyphs.PendingMark + " measuring this projection"))
	case preview.Err != nil:
		add(m.theme.Blocked.Render(m.theme.Glyphs.BlockedMark + " this handoff cannot be planned"))
		for _, wrapped := range ui.Wrap(ui.Sanitize(preview.Err.Error()), inner-2) {
			add("  " + m.theme.Muted.Render(wrapped))
		}
	default:
		lines = append(lines, m.previewLines(preview, inner)...)
	}

	// The reminder that this is not native resume. It is stated on every frame
	// because conflating the two is the single most damaging misunderstanding
	// a user of this command can carry away from it.
	add("")
	add(m.theme.Muted.Render(ui.Truncate(
		"starts a NEW "+m.Destination()+" session from a briefing; not a native resume",
		inner, m.theme.Glyphs.Ellipsis,
	)))

	add("")
	add(m.theme.Muted.Render("equivalent command"))
	for _, wrapped := range ui.Wrap(m.EquivalentCommand(), inner) {
		add(m.theme.Command.Render(wrapped))
	}

	if m.status != "" {
		add("")
		add(m.theme.Warn.Render(ui.Truncate(ui.Sanitize(m.status), inner, m.theme.Glyphs.Ellipsis)))
	}

	add("")
	add(m.keyBar())
	return strings.Join(lines, "\n")
}

// destinationLine renders the destination chooser as a spinner between arrows.
func (m *Model) destinationLine() string {
	if len(m.destinations) == 0 {
		return m.theme.Blocked.Render("no destination agent available")
	}
	name := m.theme.Agent(m.Destination(), 0)
	arrows := ""
	if len(m.destinations) > 1 {
		arrows = m.theme.Accent.Render(m.theme.Glyphs.LeftArrow) + " " + name + " " +
			m.theme.Accent.Render(m.theme.Glyphs.RightArrow)
	} else {
		arrows = name
	}
	return m.theme.Muted.Render("to  ") + arrows
}

// policyLine renders the policy chooser and the measured size beside it.
func (m *Model) policyLine(inner int) string {
	left := m.theme.Muted.Render("policy  ") +
		m.theme.Accent.Render(m.theme.Glyphs.LeftArrow) + " " +
		m.theme.Title.Render(ui.Fit(m.Policy(), 10, m.theme.Glyphs.Ellipsis)) + " " +
		m.theme.Accent.Render(m.theme.Glyphs.RightArrow)

	// A plan that failed is not still being measured, and saying so would
	// promise a number that is never going to arrive.
	right := m.theme.Pending.Render("measuring")
	if preview, ready := m.current(); ready {
		switch {
		case preview.Err != nil:
			right = m.theme.Blocked.Render("no measurement")
		default:
			right = m.theme.Muted.Render(fmt.Sprintf(
				"%s · %s · ~%s tokens",
				humanBytes(preview.Bytes),
				plural(preview.Events, "event", "events"),
				humanCount(preview.Tokens),
			))
		}
	}
	gap := inner - ui.Width(left) - ui.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// previewLines renders what the capsule carries and what it leaves behind.
func (m *Model) previewLines(preview Preview, inner int) []string {
	var lines []string
	add := func(text string) { lines = append(lines, " "+text) }

	included := make([]Component, 0, len(preview.Components))
	excluded := make([]Component, 0, len(preview.Components))
	for _, component := range preview.Components {
		if component.Included() {
			included = append(included, component)
			continue
		}
		excluded = append(excluded, component)
	}

	add(m.theme.Muted.Render("carried across"))
	if len(included) == 0 {
		add("  " + m.theme.Muted.Render("nothing beyond the task boundary"))
	}
	for _, component := range included {
		add("  " + m.theme.Ready.Render(m.theme.Glyphs.Included) + " " +
			ui.Fit(ui.Sanitize(component.Name), componentNameWidth, m.theme.Glyphs.Ellipsis) + " " +
			m.theme.Muted.Render(m.componentDetailLine(component, inner)))
	}

	if len(excluded) > 0 {
		add("")
		add(m.theme.Muted.Render("left behind"))
		for _, component := range excluded {
			add("  " + m.theme.Blocked.Render(m.theme.Glyphs.Excluded) + " " +
				ui.Fit(ui.Sanitize(component.Name), componentNameWidth, m.theme.Glyphs.Ellipsis) + " " +
				m.theme.Muted.Render(m.componentDetailLine(component, inner)))
		}
	}

	if total := preview.RedactionTotal(); total > 0 {
		add("")
		add(m.theme.Warn.Render(fmt.Sprintf(
			"%d %s hidden by redaction",
			total,
			pluralWord(total, "value", "values"),
		)))
		// Category names only. A redacted value must never reach the screen,
		// which is the entire reason it was redacted.
		add("  " + m.theme.Muted.Render(ui.Truncate(
			strings.Join(preview.RedactionCategories(), ", "), inner-2, m.theme.Glyphs.Ellipsis)))
	}

	if len(preview.Warnings) > 0 {
		add("")
		box := m.theme.Glyphs.CheckOff
		if m.acknowledged {
			box = m.theme.Ready.Render(m.theme.Glyphs.CheckOn)
		}
		add(box + " " + m.theme.Warn.Render(fmt.Sprintf(
			"%d %s to acknowledge",
			len(preview.Warnings),
			pluralWord(len(preview.Warnings), "warning", "warnings"),
		)))
		// The identifiers are listed, not just counted. Accepting a warning
		// you cannot name is not consent, and these are the exact strings the
		// equivalent command passes to --allow-warning.
		for _, warning := range preview.Warnings {
			add("  " + m.theme.Muted.Render(ui.Truncate(ui.Sanitize(warning), inner-2, m.theme.Glyphs.Ellipsis)))
		}
	}
	return lines
}

// componentNameWidth is the fixed column the component name occupies, so the
// details after it line up down the list.
const componentNameWidth = 22

// componentDetailLine bounds a component's detail to the room left on the row.
// The reason text comes from the capsule, which means from a vendor transcript,
// so its length is not the view's to assume: an unbounded one would run past
// the right edge and wrap, breaking every row after it.
func (m *Model) componentDetailLine(component Component, inner int) string {
	// The row is a leading space, a two-space indent, a one-cell glyph, a
	// space, the fitted name, and a space.
	budget := inner - componentNameWidth - 5
	return ui.Truncate(componentDetail(component), budget, m.theme.Glyphs.Ellipsis)
}

func componentDetail(component Component) string {
	detail := component.Portability
	if component.Count > 0 {
		detail += fmt.Sprintf(" · %d", component.Count)
	}
	if component.Reason != "" {
		detail += " · " + ui.Sanitize(component.Reason)
	}
	return detail
}

func (m *Model) keyBar() string {
	pairs := [][2]string{
		{m.theme.Glyphs.LeftArrow + m.theme.Glyphs.RightArrow, "policy"},
		{"tab", "destination"},
	}
	if len(m.Warnings()) > 0 {
		pairs = append(pairs, [2]string{"a", "acknowledge"})
	}
	pairs = append(pairs,
		[2]string{m.theme.Glyphs.Enter, "send"},
		[2]string{"e", "export only"},
		[2]string{"c", "copy command"},
		[2]string{"esc", "cancel"},
	)
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, m.theme.KeyCap.Render(pair[0])+" "+m.theme.KeyBar.Render(pair[1]))
	}
	return ui.Truncate(strings.Join(parts, m.theme.KeyBar.Render("   ")), m.width-1, m.theme.Glyphs.Ellipsis)
}

// humanBytes renders a byte count at one decimal place, which is as much
// precision as a size comparison between policies ever needs.
func humanBytes(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

func humanCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	return fmt.Sprintf("%.1fk", float64(count)/1000)
}

func plural(count int, singular, many string) string {
	return fmt.Sprintf("%d %s", count, pluralWord(count, singular, many))
}

func pluralWord(count int, singular, many string) string {
	if count == 1 {
		return singular
	}
	return many
}
