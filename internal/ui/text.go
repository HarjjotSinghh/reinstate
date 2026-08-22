// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Width returns the rendered cell width of s, counting wide glyphs as two
// cells and ignoring ANSI sequences.
func Width(s string) int { return lipgloss.Width(s) }

// Pad right-pads s with spaces to exactly width cells. A string already at or
// beyond width is returned unchanged; use Truncate first when the column is
// hard-bounded.
func Pad(s string, width int) string {
	current := Width(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
}

// Fit truncates s to width cells and then pads it, so the result occupies
// exactly width cells.
func Fit(s string, width int, ellipsis string) string {
	return Pad(Truncate(s, width, ellipsis), width)
}

// Truncate shortens s to at most width cells, appending ellipsis when it had to
// cut. It is width-aware rather than rune-aware, so CJK titles do not overflow
// their column.
func Truncate(s string, width int, ellipsis string) string {
	if width <= 0 {
		return ""
	}
	if Width(s) <= width {
		return s
	}
	markerWidth := Width(ellipsis)
	if markerWidth >= width {
		// No room for content plus a marker; cut hard.
		return cutToWidth(s, width)
	}
	return cutToWidth(s, width-markerWidth) + ellipsis
}

func cutToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	var builder strings.Builder
	for _, r := range s {
		// Measure the candidate prefix rather than the rune on its own. A
		// variation selector or a combining mark is zero cells alone but widens
		// the cluster it joins, so per-rune arithmetic lets a string such as
		// "⚠️" overflow its column by a cell and shift every column after it.
		if lipgloss.Width(builder.String()+string(r)) > width {
			break
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// Sanitize makes untrusted session text safe to draw: control characters are
// dropped, every run of whitespace collapses to one space, and the result is
// trimmed.
//
// Session titles and prompt previews reach the screen from vendor files. A raw
// escape sequence in one of those would let a session file repaint the
// terminal, so nothing renders without passing through here first.
func Sanitize(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	lastWasSpace := true // leading whitespace is dropped
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Cf, r):
			// Zero-width and format characters, including the byte-order mark
			// and the bidi overrides that can reorder a rendered line.
			continue
		case unicode.IsControl(r) || unicode.IsSpace(r):
			if !lastWasSpace {
				builder.WriteByte(' ')
				lastWasSpace = true
			}
		default:
			builder.WriteRune(r)
			lastWasSpace = false
		}
	}
	return strings.TrimRight(builder.String(), " ")
}

// PreviewLimit is the maximum number of Unicode code points of user-authored
// prompt text any surface may show. It matches the bound the session index
// already enforces; the UI restates it so a widened index cannot silently
// widen the screen.
const PreviewLimit = 160

// Preview sanitizes and bounds untrusted prompt text for display.
func Preview(s string) string {
	cleaned := Sanitize(s)
	runes := []rune(cleaned)
	if len(runes) <= PreviewLimit {
		return cleaned
	}
	return strings.TrimRight(string(runes[:PreviewLimit]), " ") + "..."
}

// Wrap breaks s into lines of at most width cells, splitting on spaces and
// hard-cutting words that cannot fit.
func Wrap(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var (
		lines   []string
		current strings.Builder
	)
	flush := func() {
		if current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
		}
	}
	for _, word := range words {
		for Width(word) > width {
			flush()
			head := cutToWidth(word, width)
			if head == "" {
				// One glyph is wider than the whole column, so no cut can honour
				// the bound. Emit it alone: looping on a zero-length cut would
				// spin forever and grow lines without limit, and untrusted
				// session text can reach here.
				_, size := utf8.DecodeRuneInString(word)
				head = word[:size]
			}
			lines = append(lines, head)
			word = word[len(head):]
		}
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if Width(current.String())+1+Width(word) > width {
			flush()
			current.WriteString(word)
			continue
		}
		current.WriteByte(' ')
		current.WriteString(word)
	}
	flush()
	return lines
}
