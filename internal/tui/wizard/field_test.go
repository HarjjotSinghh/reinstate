// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// key builds one key press. The names match the spellings tuitest uses, plus
// the editing keys this field implements that the shared harness does not name.
// Anything else is sent as a rune press, which is how a paste arrives.
func key(t *testing.T, name string) tea.KeyMsg {
	t.Helper()
	switch name {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "space", " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+a":
		return tea.KeyMsg{Type: tea.KeyCtrlA}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+e":
		return tea.KeyMsg{Type: tea.KeyCtrlE}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+w":
		return tea.KeyMsg{Type: tea.KeyCtrlW}
	case "":
		t.Fatal("empty key name")
		return tea.KeyMsg{}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
	}
}

// press applies a sequence of named keys. A name that is not a key name is
// inserted as text, in one press, the way a paste does.
func press(t *testing.T, f *field, names ...string) {
	t.Helper()
	for _, name := range names {
		f.Update(key(t, name))
	}
}

// typeRunes sends each rune as its own press, the way a person typing does.
func typeRunes(t *testing.T, f *field, text string) {
	t.Helper()
	for _, r := range text {
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// monoTheme is the theme every render assertion here uses: no colour, so the
// rendered string is exactly the characters a reader sees.
func monoTheme() ui.Theme {
	return ui.NewTheme(ui.Capability{Mode: ui.ModeFull, Color: ui.ColorNone, Unicode: true})
}

func TestFieldEditing(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		keys       []string
		want       string
		wantCursor int
	}{
		{
			name:  "a new field puts the cursor after the value",
			start: "reinstate", keys: nil,
			want: "reinstate", wantCursor: 9,
		},
		{
			name:  "insert at the end",
			start: "rein", keys: []string{"state"},
			want: "reinstate", wantCursor: 9,
		},
		{
			name:  "insert at the start",
			start: "state", keys: []string{"home", "rein"},
			want: "reinstate", wantCursor: 4,
		},
		{
			name:  "insert at the cursor",
			start: "rente", keys: []string{"left", "left", "insta"},
			want: "reninstate", wantCursor: 8,
		},
		{
			name:  "space is inserted as text",
			start: "two", keys: []string{"home", "one", "space"},
			want: "one two", wantCursor: 4,
		},
		{
			name:  "backspace removes the rune before the cursor",
			start: "bucket", keys: []string{"backspace"},
			want: "bucke", wantCursor: 5,
		},
		{
			name:  "backspace in the middle",
			start: "bucket", keys: []string{"home", "right", "right", "backspace"},
			want: "bcket", wantCursor: 1,
		},
		{
			name:  "backspace at the start does nothing",
			start: "bucket", keys: []string{"home", "backspace", "backspace"},
			want: "bucket", wantCursor: 0,
		},
		{
			name:  "backspace on an empty field does nothing",
			start: "", keys: []string{"backspace"},
			want: "", wantCursor: 0,
		},
		{
			name:  "delete removes the rune under the cursor",
			start: "bucket", keys: []string{"home", "delete"},
			want: "ucket", wantCursor: 0,
		},
		{
			name:  "delete in the middle",
			start: "bucket", keys: []string{"home", "right", "right", "delete"},
			want: "buket", wantCursor: 2,
		},
		{
			name:  "delete at the end does nothing",
			start: "bucket", keys: []string{"delete", "delete"},
			want: "bucket", wantCursor: 6,
		},
		{
			name:  "left stops at the start",
			start: "abc", keys: []string{"left", "left", "left", "left", "left"},
			want: "abc", wantCursor: 0,
		},
		{
			name:  "right stops at the end",
			start: "abc", keys: []string{"home", "right", "right", "right", "right", "right"},
			want: "abc", wantCursor: 3,
		},
		{
			name:  "home and end",
			start: "abcdef", keys: []string{"home"},
			want: "abcdef", wantCursor: 0,
		},
		{
			name:  "end returns to the end",
			start: "abcdef", keys: []string{"home", "end"},
			want: "abcdef", wantCursor: 6,
		},
		{
			name:  "ctrl+a is home",
			start: "abcdef", keys: []string{"ctrl+a"},
			want: "abcdef", wantCursor: 0,
		},
		{
			name:  "ctrl+e is end",
			start: "abcdef", keys: []string{"ctrl+a", "ctrl+e"},
			want: "abcdef", wantCursor: 6,
		},
		{
			name:  "ctrl+a then insert prepends",
			start: "example.com", keys: []string{"ctrl+a", "https://"},
			want: "https://example.com", wantCursor: 8,
		},
		{
			name:  "ctrl+u clears the field",
			start: "https://s3.us-east-1.amazonaws.com", keys: []string{"ctrl+u"},
			want: "", wantCursor: 0,
		},
		{
			name:  "ctrl+u from the middle clears everything",
			start: "abcdef", keys: []string{"left", "left", "ctrl+u"},
			want: "", wantCursor: 0,
		},
		{
			name:  "ctrl+u then typing starts over",
			start: "wrong", keys: []string{"ctrl+u", "right", "new"},
			want: "new", wantCursor: 3,
		},
		{
			name:  "ctrl+w deletes the previous word",
			start: "one two three", keys: []string{"ctrl+w"},
			want: "one two ", wantCursor: 8,
		},
		{
			name:  "ctrl+w twice deletes two words",
			start: "one two three", keys: []string{"ctrl+w", "ctrl+w"},
			want: "one ", wantCursor: 4,
		},
		{
			name:  "ctrl+w skips trailing spaces first",
			start: "one two   ", keys: []string{"ctrl+w"},
			want: "one ", wantCursor: 4,
		},
		{
			name:  "ctrl+w on spaces alone empties the field",
			start: "   ", keys: []string{"ctrl+w"},
			want: "", wantCursor: 0,
		},
		{
			name:  "ctrl+w at the start does nothing",
			start: "one two", keys: []string{"home", "ctrl+w"},
			want: "one two", wantCursor: 0,
		},
		{
			name:  "ctrl+w on an empty field does nothing",
			start: "", keys: []string{"ctrl+w"},
			want: "", wantCursor: 0,
		},
		{
			// Readline leaves the separator that followed the deleted word,
			// so the text after the cursor is untouched.
			name:  "ctrl+w keeps the text after the cursor",
			start: "one two three", keys: []string{"home", "right", "right", "right", "right", "right", "right", "right", "ctrl+w"},
			want: "one  three", wantCursor: 4,
		},
		{
			name:  "ctrl+w on a single word empties the field",
			start: "bucket", keys: []string{"ctrl+w"},
			want: "", wantCursor: 0,
		},
		{
			name:  "an unbound key changes nothing",
			start: "bucket", keys: []string{"up", "down", "enter", "esc"},
			want: "bucket", wantCursor: 6,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := newField(test.start, "placeholder")
			press(t, &field, test.keys...)

			if got := field.Value(); got != test.want {
				t.Fatalf("Value = %q, want %q", got, test.want)
			}
			if field.cursor != test.wantCursor {
				t.Fatalf("cursor = %d, want %d", field.cursor, test.wantCursor)
			}
			if got := field.Empty(); got != (test.want == "") {
				t.Fatalf("Empty = %v with value %q", got, test.want)
			}
			if field.cursor < 0 || field.cursor > len([]rune(test.want)) {
				t.Fatalf("cursor %d is outside the value %q", field.cursor, test.want)
			}
		})
	}
}

func TestFieldSetValue(t *testing.T) {
	field := newField("original", "placeholder")
	press(t, &field, "home")

	field.SetValue("replaced")

	if got := field.Value(); got != "replaced" {
		t.Fatalf("Value = %q, want %q", got, "replaced")
	}
	if field.cursor != len("replaced") {
		t.Fatalf("cursor = %d, want it moved to the end at %d", field.cursor, len("replaced"))
	}
	field.SetValue("")
	if !field.Empty() || field.cursor != 0 {
		t.Fatalf("after clearing: value %q cursor %d, want empty at 0", field.Value(), field.cursor)
	}
	// Typing after a clear starts from scratch rather than from a stale cursor.
	typeRunes(t, &field, "new")
	if got := field.Value(); got != "new" {
		t.Fatalf("Value = %q, want %q", got, "new")
	}
}

// TestControlCharactersNeverEnterTheValue is a correctness invariant for the
// config file, not a cosmetic one. These values are written to config and used
// to build storage requests: a newline reaching an endpoint would corrupt the
// file, and an escape introducer would repaint the terminal on the next render.
func TestControlCharactersNeverEnterTheValue(t *testing.T) {
	const forbidden = "\x1b\n\r\t\x00\x07\x7f\v\f"

	tests := []struct {
		name  string
		runes string
		want  string
	}{
		{name: "escape introducer", runes: "\x1b[31mred\x1b[0m", want: "[31mred[0m"},
		{name: "newline", runes: "one\ntwo", want: "onetwo"},
		{name: "carriage return", runes: "one\r\ntwo", want: "onetwo"},
		{name: "tab", runes: "one\ttwo", want: "onetwo"},
		{name: "nul", runes: "one\x00two", want: "onetwo"},
		{name: "bell and backspace", runes: "one\a\btwo", want: "onetwo"},
		{name: "delete", runes: "one\x7ftwo", want: "onetwo"},
		{name: "vertical tab and form feed", runes: "one\v\ftwo", want: "onetwo"},
		{name: "control characters only", runes: "\x1b\n\r\t\x00", want: ""},
		{name: "an OSC title sequence", runes: "\x1b]0;pwned\a", want: "]0;pwned"},
		{name: "a printable value survives intact", runes: "https://s3.example.com", want: "https://s3.example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Sent as one press, because that is how a paste of hostile text
			// arrives: a single KeyRunes carrying every rune at once.
			field := newField("", "placeholder")
			field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(test.runes)})

			if got := field.Value(); got != test.want {
				t.Fatalf("Value = %q, want %q", got, test.want)
			}
			if index := strings.IndexAny(field.Value(), forbidden); index >= 0 {
				t.Fatalf("value %q kept the control character %q", field.Value(), field.Value()[index])
			}
			if field.cursor != len([]rune(test.want)) {
				t.Fatalf("cursor = %d, want %d", field.cursor, len([]rune(test.want)))
			}

			// Rune by rune has to reach the same place: the filter cannot depend
			// on a whole paste arriving at once.
			byRune := newField("", "placeholder")
			typeRunes(t, &byRune, test.runes)
			if got := byRune.Value(); got != test.want {
				t.Fatalf("typed rune by rune: Value = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("a press of only control characters does not move the cursor", func(t *testing.T) {
		field := newField("abc", "placeholder")
		press(t, &field, "home")
		field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\x1b\n")})
		if field.Value() != "abc" || field.cursor != 0 {
			t.Fatalf("value %q cursor %d, want %q at 0", field.Value(), field.cursor, "abc")
		}
	})

	// A seeded value is the other way in, and the one a pairing code takes:
	// `rein init --paste` decodes a code someone pasted from a chat window and
	// hands the endpoint straight to the wizard as a default.
	t.Run("a seeded value is filtered too", func(t *testing.T) {
		field := newField("https://\x1b[31mevil\x1b[0m.example.com\n", "placeholder")
		if strings.ContainsAny(field.Value(), forbidden) {
			t.Fatalf("a seeded value kept control characters: %q", field.Value())
		}
		if got := field.Value(); got != "https://[31mevil[0m.example.com" {
			t.Fatalf("Value = %q, want the escape introducers stripped and the text kept", got)
		}
		if field.cursor != len([]rune(field.Value())) {
			t.Fatalf("cursor = %d, want the end of the filtered value %d",
				field.cursor, len([]rune(field.Value())))
		}
	})

	t.Run("SetValue is filtered too", func(t *testing.T) {
		field := newField("", "placeholder")
		field.SetValue("bucket\x1b[2J")
		if strings.ContainsAny(field.Value(), forbidden) {
			t.Fatalf("SetValue kept control characters: %q", field.Value())
		}
	})
}

func TestFieldRender(t *testing.T) {
	theme := monoTheme()

	t.Run("an empty unfocused field shows the placeholder", func(t *testing.T) {
		field := newField("", "reinstate")
		if got := field.Render(theme, false, 20); got != "reinstate" {
			t.Fatalf("Render = %q, want the placeholder", got)
		}
	})

	t.Run("a long placeholder is truncated to the width", func(t *testing.T) {
		field := newField("", "paste the ID from your first device")
		got := field.Render(theme, false, 12)
		if ui.Width(got) > 12 {
			t.Fatalf("Render = %q, %d cells, want at most 12", got, ui.Width(got))
		}
		if !strings.HasSuffix(got, theme.Glyphs.Ellipsis) {
			t.Fatalf("Render = %q, want it to end with the ellipsis", got)
		}
	})

	t.Run("a value hides the placeholder", func(t *testing.T) {
		field := newField("reinstate-sessions", "reinstate")
		if got := field.Render(theme, false, 40); got != "reinstate-sessions" {
			t.Fatalf("Render = %q, want the value", got)
		}
	})

	// Without colour the cursor cannot be a highlight, so it is an extra cell:
	// a focused field is one cell wider than the same text unfocused. That is
	// the only thing a monochrome terminal can show, and the only thing a
	// golden frame can pin.
	t.Run("focus adds a cursor cell", func(t *testing.T) {
		field := newField("bucket", "reinstate")
		unfocused := field.Render(theme, false, 40)
		focused := field.Render(theme, true, 40)

		if unfocused != "bucket" {
			t.Fatalf("unfocused = %q, want %q", unfocused, "bucket")
		}
		if focused != "bucket " {
			t.Fatalf("focused = %q, want the value plus a cursor cell", focused)
		}
		if ui.Width(focused) != ui.Width(unfocused)+1 {
			t.Fatalf("focused is %d cells and unfocused is %d; the cursor is invisible",
				ui.Width(focused), ui.Width(unfocused))
		}
	})

	t.Run("an empty focused field shows a cursor before the placeholder", func(t *testing.T) {
		field := newField("", "reinstate")
		got := field.Render(theme, true, 20)
		if got != " reinstate" {
			t.Fatalf("Render = %q, want a cursor cell then the placeholder", got)
		}
		if ui.Width(got) > 20 {
			t.Fatalf("Render = %q, %d cells, want at most 20", got, ui.Width(got))
		}
	})

	t.Run("the window scrolls to keep the cursor visible", func(t *testing.T) {
		const value = "https://s3.us-east-1.amazonaws.com"
		field := newField(value, "endpoint")

		// Cursor at the end: the tail is what matters.
		atEnd := field.Render(theme, true, 12)
		if ui.Width(atEnd) > 12 {
			t.Fatalf("Render = %q, %d cells, want at most 12", atEnd, ui.Width(atEnd))
		}
		if !strings.Contains(atEnd, "naws.com") {
			t.Fatalf("Render = %q, want the end of %q to be visible", atEnd, value)
		}
		if strings.Contains(atEnd, "https") {
			t.Fatalf("Render = %q, want the start scrolled out of view", atEnd)
		}

		// Cursor back at the start: the head is what matters, and the tail must
		// not be drawn past the width of the field.
		press(t, &field, "home")
		atStart := field.Render(theme, true, 12)
		if ui.Width(atStart) > 12 {
			t.Fatalf("Render = %q, %d cells, want at most 12", atStart, ui.Width(atStart))
		}
		if !strings.HasPrefix(atStart, "https://") {
			t.Fatalf("Render = %q, want the start of %q to be visible", atStart, value)
		}
		if strings.Contains(atStart, "amazonaws") {
			t.Fatalf("Render = %q, want the end scrolled out of view", atStart)
		}

		// And somewhere in the middle.
		press(t, &field, "right", "right", "right", "right", "right",
			"right", "right", "right", "right", "right",
			"right", "right", "right", "right", "right")
		middle := field.Render(theme, true, 12)
		if ui.Width(middle) > 12 {
			t.Fatalf("Render = %q, %d cells, want at most 12", middle, ui.Width(middle))
		}
		if !strings.ContainsRune(middle, []rune(value)[15]) {
			t.Fatalf("Render = %q, want the rune under the cursor (%q) visible",
				middle, string([]rune(value)[15]))
		}
	})

	// The width bound is the layout contract of every surface that embeds a
	// field: one overflowing line pushes the whole frame past the terminal.
	t.Run("the width bound holds for every length and width", func(t *testing.T) {
		const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"
		for length := 0; length <= len(alphabet); length += 3 {
			value := alphabet[:length]
			for width := 1; width <= 24; width++ {
				for _, cursor := range []int{0, length / 3, length / 2, length} {
					field := newField(value, "placeholder text that is quite long")
					field.cursor = cursor
					for _, focused := range []bool{true, false} {
						got := field.Render(theme, focused, width)
						if cells := ui.Width(got); cells > width {
							t.Fatalf(
								"len %d width %d cursor %d focused %v: Render = %q is %d cells",
								length, width, cursor, focused, got, cells)
						}
						if strings.ContainsRune(got, 0x1b) {
							t.Fatalf("Render = %q contains an escape byte", got)
						}
					}
					// The cursor must stay on screen, or the reader cannot see
					// what they are editing.
					if cursor < length {
						got := field.Render(theme, true, width)
						if !strings.ContainsRune(got, rune(value[cursor])) {
							t.Fatalf("len %d width %d cursor %d: Render = %q does not show %q",
								length, width, cursor, got, string(value[cursor]))
						}
					}
				}
			}
		}
	})

	t.Run("a non-positive width renders nothing", func(t *testing.T) {
		field := newField("bucket", "placeholder")
		for _, width := range []int{0, -1} {
			for _, focused := range []bool{true, false} {
				if got := field.Render(theme, focused, width); got != "" {
					t.Fatalf("width %d focused %v: Render = %q, want empty", width, focused, got)
				}
			}
		}
	})
}

// TestFieldUnicode covers the values this form actually receives: an endpoint
// can be an internationalised host, and a bucket name pasted from a browser can
// carry anything. Editing is per rune, and the width bound is per cell.
func TestFieldUnicode(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		keys       []string
		want       string
		wantCursor int
	}{
		{
			name:  "CJK is inserted per rune",
			start: "", keys: []string{"日本語"},
			want: "日本語", wantCursor: 3,
		},
		{
			name:  "backspace removes one CJK rune",
			start: "日本語", keys: []string{"backspace"},
			want: "日本", wantCursor: 2,
		},
		{
			name:  "delete removes one CJK rune",
			start: "日本語", keys: []string{"home", "delete"},
			want: "本語", wantCursor: 0,
		},
		{
			name:  "left and right move by rune, not by byte",
			start: "日本語", keys: []string{"home", "right", "insert"},
			want: "日insert本語", wantCursor: 7,
		},
		{
			name:  "emoji are one rune each",
			start: "🚀🛰", keys: []string{"backspace"},
			want: "🚀", wantCursor: 1,
		},
		{
			name:  "an emoji is inserted whole",
			start: "ship ", keys: []string{"🚀"},
			want: "ship 🚀", wantCursor: 6,
		},
		{
			name:  "mixed scripts",
			start: "bucket-日本-🚀", keys: []string{"backspace", "backspace"},
			want: "bucket-日本", wantCursor: 9,
		},
		{
			name:  "ctrl+w deletes a CJK word",
			start: "one 日本語", keys: []string{"ctrl+w"},
			want: "one ", wantCursor: 4,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := newField(test.start, "placeholder")
			press(t, &field, test.keys...)

			if got := field.Value(); got != test.want {
				t.Fatalf("Value = %q, want %q", got, test.want)
			}
			if field.cursor != test.wantCursor {
				t.Fatalf("cursor = %d, want %d", field.cursor, test.wantCursor)
			}
		})
	}

	t.Run("the width bound holds for wide glyphs", func(t *testing.T) {
		theme := monoTheme()
		values := []string{
			"日本語のバケット",
			"🚀🛰🌍🌎🌏",
			"bucket-日本語-🚀-name",
			strings.Repeat("語", 30),
		}
		for _, value := range values {
			runes := []rune(value)
			for width := 1; width <= 20; width++ {
				for cursor := 0; cursor <= len(runes); cursor++ {
					field := newField(value, "placeholder")
					field.cursor = cursor
					for _, focused := range []bool{true, false} {
						got := field.Render(theme, focused, width)
						if cells := ui.Width(got); cells > width {
							t.Fatalf("%q width %d cursor %d focused %v: Render = %q is %d cells",
								value, width, cursor, focused, got, cells)
						}
					}
				}
			}
		}
	})
}

// TestFieldRenderIsStable checks Render has no side effects: it takes a value
// receiver, and a scroll window computed in it must not leak back into state.
func TestFieldRenderIsStable(t *testing.T) {
	theme := monoTheme()
	field := newField("https://s3.us-east-1.amazonaws.com", "endpoint")
	before, beforeCursor := field.Value(), field.cursor

	for i := 0; i < 3; i++ {
		first := field.Render(theme, true, 10)
		second := field.Render(theme, true, 10)
		if first != second {
			t.Fatalf("Render is not stable: %q then %q", first, second)
		}
	}
	if field.Value() != before || field.cursor != beforeCursor {
		t.Fatalf("Render changed the field: %q at %d, was %q at %d",
			field.Value(), field.cursor, before, beforeCursor)
	}
}
