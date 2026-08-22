// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package readiness

import (
	"fmt"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// View implements tea.Model.
func (c *Checklist) View() string {
	if c.quitting {
		return ""
	}
	width := c.width
	if width <= 0 {
		width = 80
	}
	inner := width - 2
	if inner < 20 {
		inner = 20
	}

	var lines []string
	add := func(text string) { lines = append(lines, " "+text) }

	add(c.theme.Title.Render(ui.Truncate(c.reference, inner, c.theme.Glyphs.Ellipsis)))
	add(c.theme.Warn.Render(fmt.Sprintf(
		"%s  %s",
		c.theme.Glyphs.WarnMark,
		plural(len(c.items), "environment warning", "environment warnings"),
	)))
	add("")

	for index, current := range c.items {
		lines = append(lines, c.itemLines(index, current, inner)...)
	}
	if len(c.items) == 0 {
		add(c.theme.Muted.Render("No warnings to acknowledge."))
	}

	add("")
	add(c.theme.Muted.Render("equivalent command"))
	for _, wrapped := range ui.Wrap(c.EquivalentCommand(), inner) {
		add(c.theme.Command.Render(wrapped))
	}

	if c.status != "" {
		add("")
		add(c.theme.Warn.Render(ui.Truncate(c.status, inner, c.theme.Glyphs.Ellipsis)))
	}

	add("")
	add(c.keyBar())
	return strings.Join(lines, "\n")
}

// itemLines renders one warning: the checkbox, the message, and the repair
// hint when the verifier supplied one.
func (c *Checklist) itemLines(index int, current item, inner int) []string {
	cursor := "  "
	if index == c.cursor {
		cursor = c.theme.Selected.Render(c.theme.Glyphs.Cursor) + " "
	}
	box := c.theme.Glyphs.CheckOff
	if current.accepted {
		box = c.theme.Ready.Render(c.theme.Glyphs.CheckOn)
	}

	identifier := c.theme.Accent.Render(current.check.ID)
	head := " " + cursor + box + " " + identifier
	lines := []string{head}

	// The message is indented under the identifier so the checkbox column stays
	// scannable when a message wraps.
	indent := "       "
	for _, wrapped := range ui.Wrap(ui.Sanitize(current.check.Message), inner-len(indent)) {
		lines = append(lines, " "+indent+c.theme.Muted.Render(wrapped))
	}
	if repair := ui.Sanitize(current.check.Repair); repair != "" {
		prefix := indent + c.theme.Glyphs.TrailLink + " "
		for offset, wrapped := range ui.Wrap("repair: "+repair, inner-len(indent)-3) {
			if offset == 0 {
				lines = append(lines, " "+prefix+c.theme.Muted.Render(wrapped))
				continue
			}
			lines = append(lines, " "+indent+"  "+c.theme.Muted.Render(wrapped))
		}
	}
	return lines
}

func (c *Checklist) keyBar() string {
	pairs := [][2]string{
		{"space", "acknowledge"},
		{"a", "all"},
		{c.theme.Glyphs.Enter, "continue"},
		{"c", "copy command"},
		{"esc", "cancel"},
	}
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, c.theme.KeyCap.Render(pair[0])+" "+c.theme.KeyBar.Render(pair[1]))
	}
	return ui.Truncate(strings.Join(parts, c.theme.KeyBar.Render("   ")), c.width-1, c.theme.Glyphs.Ellipsis)
}

func plural(count int, singular, pluralWord string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, pluralWord)
}

// Summary renders a plain, non-interactive description of the report for the
// degraded path. It deliberately mirrors what the interactive view shows so the
// two surfaces cannot drift into describing the environment differently.
func Summary(theme ui.Theme, report preflight.Report) []string {
	var lines []string
	for _, check := range report.Checks {
		if check.Severity != preflight.SeverityWarning {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s  %s", check.ID, ui.Sanitize(check.Message)))
	}
	return lines
}
