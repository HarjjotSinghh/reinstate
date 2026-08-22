// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package wizard

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

	add(m.theme.Title.Render("rein init") + "  " + m.theme.Muted.Render(m.progress()))
	add("")
	add(m.theme.Title.Render(m.step.title()))
	add("")
	lines = append(lines, m.stepBody(inner)...)

	if m.status != "" {
		add("")
		add(m.theme.Warn.Render(ui.Truncate(m.status, inner, m.theme.Glyphs.Ellipsis)))
	}
	add("")
	add(m.keyBar())
	return strings.Join(lines, "\n")
}

// progress shows which of the applicable steps this is. Steps that do not apply
// to the current answers are not counted, so the total does not jump around.
func (m *Model) progress() string {
	total, position := 0, 0
	for candidate := stepProvider; candidate < stepCount; candidate++ {
		if m.skip(candidate) {
			continue
		}
		total++
		if candidate == m.step {
			position = total
		}
	}
	return fmt.Sprintf("step %d of %d", position, total)
}

func (m *Model) stepBody(inner int) []string {
	var lines []string
	add := func(text string) { lines = append(lines, " "+text) }

	switch m.step {
	case stepProvider:
		for index, provider := range Providers {
			marker := "  "
			name := provider.Name
			if index == m.providerIndex {
				marker = m.theme.Selected.Render(m.theme.Glyphs.Cursor) + " "
				name = m.theme.Selected.Render(name)
			}
			add(marker + ui.Fit(name, 26, m.theme.Glyphs.Ellipsis) + " " +
				m.theme.Muted.Render(ui.Truncate(provider.Note, inner-30, m.theme.Glyphs.Ellipsis)))
		}

	case stepEndpoint, stepBucket, stepRegion, stepPrefix, stepProfileID:
		add(m.inputs[m.step].Render(m.theme, true, inner-2))
		add("")
		for _, hint := range ui.Wrap(m.hint(), inner) {
			add(m.theme.Muted.Render(hint))
		}

	case stepProfile:
		options := []struct {
			join  bool
			label string
			note  string
		}{
			{false, "This is my first device", "creates a new profile and a new encryption identity"},
			{true, "Join a profile from another device", "needs the profile ID shown when that device was set up"},
		}
		for _, option := range options {
			marker := "  "
			label := option.label
			if option.join == m.joinExisting {
				marker = m.theme.Selected.Render(m.theme.Glyphs.Cursor) + " "
				label = m.theme.Selected.Render(label)
			}
			add(marker + label)
			add("    " + m.theme.Muted.Render(ui.Truncate(option.note, inner-4, m.theme.Glyphs.Ellipsis)))
		}

	case stepReview:
		result := m.Result()
		for _, row := range [][2]string{
			{"provider", providerName(result.Provider)},
			{"endpoint", result.Endpoint},
			{"bucket", result.Bucket},
			{"region", result.Region},
			{"prefix", orDefault(result.Prefix, "profiles/<profile-id> (generated)")},
			{"device", deviceDescription(result)},
		} {
			add(m.theme.Muted.Render(ui.Fit(row[0], 10, m.theme.Glyphs.Ellipsis)) + " " +
				ui.Truncate(row[1], inner-11, m.theme.Glyphs.Ellipsis))
		}
		add("")
		// The passphrase is explained here rather than prompted for, because it
		// is not stored anywhere and losing it means losing the synced data.
		// Saying so once, plainly, before anything is written is the whole job.
		for _, wrapped := range ui.Wrap(
			"Next you will enter your storage keys, then a passphrase. "+
				"The passphrase is never written to disk or sent anywhere: it is "+
				"asked for again on every push and pull. If you lose it, the "+
				"synced sessions cannot be recovered by anyone, including you.",
			inner,
		) {
			add(m.theme.Muted.Render(wrapped))
		}
	}
	return lines
}

// hint is the guidance shown under a text field.
func (m *Model) hint() string {
	switch m.step {
	case stepEndpoint:
		return "Template: " + Providers[m.providerIndex].Endpoint +
			". Replace anything in angle brackets."
	case stepBucket:
		return "The bucket must already exist. Reinstate never creates one."
	case stepRegion:
		return "Leave blank to use " + Providers[m.providerIndex].Region + "."
	case stepPrefix:
		return "Leave blank to derive it from the profile ID. Every device on a " +
			"profile must use the same prefix."
	case stepProfileID:
		return "Run `rein init --link` on the first device to print this."
	default:
		return ""
	}
}

func (m *Model) keyBar() string {
	pairs := [][2]string{}
	switch m.step {
	case stepProvider, stepProfile:
		pairs = append(pairs, [2]string{m.theme.Glyphs.Cursor, "choose"})
	case stepReview:
		// Nothing on the review screen is editable, so offering an edit key
		// there would be a lie. Going back is the way to change an answer.
	default:
		pairs = append(pairs, [2]string{"type", "edit"})
	}
	next := "next"
	if m.step == stepReview {
		next = "start setup"
	}
	pairs = append(pairs,
		[2]string{m.theme.Glyphs.Enter, next},
		[2]string{"shift+tab", "back"},
		[2]string{"esc", "cancel"},
	)
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, m.theme.KeyCap.Render(pair[0])+" "+m.theme.KeyBar.Render(pair[1]))
	}
	return ui.Truncate(strings.Join(parts, m.theme.KeyBar.Render("   ")), m.width-1, m.theme.Glyphs.Ellipsis)
}

func providerName(key string) string {
	for _, provider := range Providers {
		if provider.Key == key {
			return provider.Name
		}
	}
	return key
}

func deviceDescription(result Result) string {
	if result.JoinExisting {
		return "joining profile " + result.ProfileID
	}
	return "first device; a new profile ID will be generated"
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
