// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package palette

import (
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// maxVisible bounds the overlay height so it never covers the whole screen.
// Seeing some of what is underneath is what makes it read as an overlay rather
// than as a new screen.
const maxVisible = 8

// Lines renders the overlay as a block of lines, each at most width cells.
func (m *Model) Lines(width int) []string {
	if !m.Open() {
		return nil
	}
	if width <= 0 {
		width = m.width
	}
	inner := width - 4
	if inner < 16 {
		inner = 16
	}

	lines := make([]string, 0, maxVisible+4)
	lines = append(lines, m.border(width, true))

	prompt := m.theme.Accent.Render(m.theme.Glyphs.Search) + " "
	text := m.filter
	if text == "" {
		text = m.theme.Muted.Render("run a command")
	}
	lines = append(lines, m.row(prompt+text, width))
	lines = append(lines, m.border(width, false))

	if len(m.filtered) == 0 {
		lines = append(lines, m.row(m.theme.Muted.Render("no command matches "+
			ui.Truncate(m.filter, 24, m.theme.Glyphs.Ellipsis)), width))
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	for offset := start; offset < len(m.filtered) && offset-start < maxVisible; offset++ {
		command := m.filtered[offset]
		marker := "  "
		// Entry text is sanitized on the way to the screen. The table is ours
		// today, but the view is the last thing between a command's words and
		// the terminal, and it must not be the component that assumes an
		// escape sequence or a newline cannot reach it.
		title := ui.Fit(ui.Sanitize(command.Title), 20, m.theme.Glyphs.Ellipsis)
		if offset == m.cursor {
			marker = m.theme.Selected.Render(m.theme.Glyphs.Cursor) + " "
			title = m.theme.Selected.Render(title)
		}
		detail := m.theme.Muted.Render(ui.Truncate(ui.Sanitize(command.Detail), inner-24, m.theme.Glyphs.Ellipsis))
		lines = append(lines, m.row(marker+title+"  "+detail, width))
	}
	// Count everything off-screen, above the window as well as below it. A count
	// of only what is below reads as "scroll down for the rest" and is wrong the
	// moment the list has scrolled at all.
	shown := len(m.filtered) - start
	if shown > maxVisible {
		shown = maxVisible
	}
	if hidden := len(m.filtered) - shown; hidden > 0 {
		lines = append(lines, m.row(m.theme.Muted.Render(plural(hidden, "more match", "more matches")), width))
	}
	lines = append(lines, m.border(width, true))
	return lines
}

// row draws one line of the overlay. The overlay is delimited by rules rather
// than boxed, because a box needs corner glyphs that the ASCII fallback cannot
// draw convincingly, and a rule reads the same in both glyph sets.
func (m *Model) row(content string, width int) string {
	return ui.Truncate(" "+content, width, m.theme.Glyphs.Ellipsis)
}

func (m *Model) border(width int, _ bool) string {
	if width < 1 {
		return ""
	}
	return m.theme.Border.Render(strings.Repeat(m.theme.Glyphs.HorizontalBr, width))
}

func plural(count int, singular, many string) string {
	word := many
	if count == 1 {
		word = singular
	}
	return itoa(count) + " " + word
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
