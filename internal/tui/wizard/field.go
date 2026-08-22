// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package wizard

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// field is a single-line text input.
//
// It is hand-written rather than taken from bubbles/textinput because that
// package depends on a native clipboard binding. Reinstate deliberately copies
// through OSC 52 instead, so that the terminal the human is actually looking at
// performs the copy — which is the only thing that works over SSH. Pulling in a
// host-side clipboard for one text field would contradict that and add a
// platform dependency to a tool whose flagship case is Windows to macOS.
//
// What a setup form needs is insert, delete, and horizontal movement. That is
// what this is.
type field struct {
	value       []rune
	cursor      int
	placeholder string
	// secret is defensive only. The wizard never collects credentials; see the
	// package comment. If that ever changes, this at least stops the value
	// being rendered.
	secret bool
}

func newField(value, placeholder string) field {
	runes := filterControl([]rune(value))
	return field{value: runes, cursor: len(runes), placeholder: placeholder}
}

// Value returns the current text.
func (f field) Value() string { return string(f.value) }

// SetValue replaces the text and puts the cursor at the end.
func (f *field) SetValue(value string) {
	f.value = filterControl([]rune(value))
	f.cursor = len(f.value)
}

// Empty reports whether the field has no text.
func (f field) Empty() bool { return len(f.value) == 0 }

// Update applies one key press.
func (f *field) Update(key tea.KeyMsg) {
	switch key.Type {
	case tea.KeyRunes:
		f.insert(key.Runes)
	case tea.KeySpace:
		f.insert([]rune{' '})
	case tea.KeyBackspace:
		if f.cursor > 0 {
			f.value = append(f.value[:f.cursor-1], f.value[f.cursor:]...)
			f.cursor--
		}
	case tea.KeyDelete:
		if f.cursor < len(f.value) {
			f.value = append(f.value[:f.cursor], f.value[f.cursor+1:]...)
		}
	case tea.KeyLeft:
		if f.cursor > 0 {
			f.cursor--
		}
	case tea.KeyRight:
		if f.cursor < len(f.value) {
			f.cursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		f.cursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		f.cursor = len(f.value)
	case tea.KeyCtrlU:
		f.value = f.value[:0]
		f.cursor = 0
	case tea.KeyCtrlW:
		f.deleteWord()
	}
}

func (f *field) insert(runes []rune) {
	filtered := filterControl(runes)
	if len(filtered) == 0 {
		return
	}
	tail := append([]rune(nil), f.value[f.cursor:]...)
	f.value = append(f.value[:f.cursor], filtered...)
	f.value = append(f.value, tail...)
	f.cursor += len(filtered)
}

// filterControl drops every rune that must not enter a single-line value.
//
// Control characters would corrupt the line and, in a value later written to
// config, could smuggle a newline into a field that must stay one line. Seeded
// values are filtered by the same rule as typed ones, because a default can
// arrive from a pairing code someone pasted out of a chat window: `rein init
// --paste` hands the decoded endpoint straight to this field, and an escape
// introducer in it would reach the screen unrendered.
func filterControl(runes []rune) []rune {
	filtered := make([]rune, 0, len(runes))
	for _, r := range runes {
		if unicode.IsControl(r) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func (f *field) deleteWord() {
	index := f.cursor
	for index > 0 && f.value[index-1] == ' ' {
		index--
	}
	for index > 0 && f.value[index-1] != ' ' {
		index--
	}
	f.value = append(f.value[:index], f.value[f.cursor:]...)
	f.cursor = index
}

// Render draws the field, showing the placeholder when empty and a block
// cursor at the insertion point when focused.
func (f field) Render(theme ui.Theme, focused bool, width int) string {
	if width <= 0 {
		return ""
	}
	if len(f.value) == 0 {
		if !focused {
			return theme.Muted.Render(ui.Truncate(f.placeholder, width, theme.Glyphs.Ellipsis))
		}
		// The cursor block takes the first cell, so the placeholder gets one
		// cell less than the field.
		return theme.Selected.Render(" ") +
			theme.Muted.Render(ui.Truncate(f.placeholder, width-1, theme.Glyphs.Ellipsis))
	}
	text := string(f.value)
	if f.secret {
		text = strings.Repeat("*", len(f.value))
	}
	if !focused {
		return ui.Truncate(text, width, theme.Glyphs.Ellipsis)
	}

	// Scroll the window so the cursor is on screen, and stop drawing at the
	// field's last cell. Both bounds are measured in display cells rather than
	// runes: a wide glyph occupies two cells, and a field that drew its whole
	// value would push the line — and so the frame — past the width of the
	// terminal it was given.
	runes := []rune(text)
	cursorCells := 1 // the block drawn past the last rune
	if f.cursor < len(runes) {
		cursorCells = ui.Width(string(runes[f.cursor]))
	}
	start := 0
	for start < f.cursor && ui.Width(string(runes[start:f.cursor]))+cursorCells > width {
		start++
	}

	var builder strings.Builder
	cells := 0
	for index := start; index < len(runes); index++ {
		glyph := string(runes[index])
		glyphCells := ui.Width(glyph)
		if cells+glyphCells > width {
			break
		}
		cells += glyphCells
		if index == f.cursor {
			builder.WriteString(theme.Selected.Render(glyph))
			continue
		}
		builder.WriteString(glyph)
	}
	if f.cursor >= len(runes) && cells < width {
		builder.WriteString(theme.Selected.Render(" "))
	}
	return builder.String()
}
