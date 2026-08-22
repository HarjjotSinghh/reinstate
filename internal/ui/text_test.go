// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"math/rand"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

// Column-aligned output is a correctness property, not a cosmetic one: the
// switcher subtracts fixed column widths from the row width, so a helper that
// returns one cell more than it promised shifts every column after it.

func TestWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: 0},
		{name: "ascii", in: "reinstate", want: 9},
		{name: "spaces count", in: "a b", want: 3},
		{name: "cjk is two cells per glyph", in: "日本語テキスト", want: 14},
		{name: "mixed script", in: "a日b", want: 4},
		{name: "emoji is two cells", in: "\U0001F600", want: 2},
		{name: "combining mark adds no cell", in: "e\u0301", want: 1},
		{name: "zero width space", in: "a\u200bb", want: 2},
		{name: "byte order mark", in: "\ufeff", want: 0},
		{name: "ansi sequences are not counted", in: "\x1b[31mred\x1b[0m", want: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Width(tc.in); got != tc.want {
				t.Fatalf("Width(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPad(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{name: "pads to width", in: "ab", width: 5, want: "ab   "},
		{name: "already exact", in: "abcde", width: 5, want: "abcde"},
		{name: "longer is returned unchanged", in: "abcdef", width: 3, want: "abcdef"},
		{name: "empty", in: "", width: 3, want: "   "},
		{name: "zero width", in: "ab", width: 0, want: "ab"},
		{name: "negative width", in: "ab", width: -4, want: "ab"},
		{name: "cjk pads by cells not runes", in: "日本", width: 6, want: "日本  "},
		{name: "emoji pads by cells", in: "\U0001F600", width: 4, want: "\U0001F600  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Pad(tc.in, tc.width)
			if got != tc.want {
				t.Fatalf("Pad(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		width    int
		ellipsis string
		want     string
	}{
		{name: "zero width", in: "abc", width: 0, ellipsis: "…", want: ""},
		{name: "negative width", in: "abc", width: -2, ellipsis: "…", want: ""},
		{name: "fits exactly", in: "abcde", width: 5, ellipsis: "…", want: "abcde"},
		{name: "fits with room", in: "abc", width: 9, ellipsis: "…", want: "abc"},
		{name: "cuts and marks", in: "abcdefgh", width: 5, ellipsis: "…", want: "abcd…"},
		{name: "ascii ellipsis reserves three cells", in: "abcdefgh", width: 5, ellipsis: "...", want: "ab..."},
		{name: "empty ellipsis cuts hard", in: "abcdefgh", width: 4, ellipsis: "", want: "abcd"},
		{
			name: "an ellipsis as wide as the column cuts hard",
			in:   "abcdefgh", width: 3, ellipsis: "...", want: "abc",
		},
		{
			name: "an ellipsis wider than the column cuts hard",
			in:   "abcdefgh", width: 2, ellipsis: "...", want: "ab",
		},
		{
			name: "cjk cuts on cell boundaries",
			in:   "日本語テキスト", width: 5, ellipsis: "…", want: "日本…",
		},
		{
			name: "cjk never splits into a half cell",
			in:   "日本語テキスト", width: 4, ellipsis: "…", want: "日…",
		},
		{
			name: "cjk with no room for content keeps only the marker",
			in:   "日本語テキスト", width: 2, ellipsis: "…", want: "…",
		},
		{
			name: "emoji cuts on cell boundaries",
			in:   "\U0001F600\U0001F600\U0001F600", width: 5, ellipsis: "…", want: "\U0001F600\U0001F600…",
		},
		{
			// A variation selector is zero cells on its own but widens the
			// cluster it joins, so the cut has to measure the cluster.
			name: "an emoji presentation selector cannot overflow the column",
			in:   "⚠\ufe0f warning", width: 3, ellipsis: "", want: "⚠\ufe0f ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Truncate(tc.in, tc.width, tc.ellipsis)
			if got != tc.want {
				t.Fatalf("Truncate(%q, %d, %q) = %q, want %q", tc.in, tc.width, tc.ellipsis, got, tc.want)
			}
			if tc.width > 0 && Width(got) > tc.width {
				t.Fatalf("Truncate(%q, %d, %q) = %q, width %d exceeds the column",
					tc.in, tc.width, tc.ellipsis, got, Width(got))
			}
		})
	}
}

func TestFit(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		width    int
		ellipsis string
		want     string
	}{
		{name: "pads a short string", in: "ab", width: 5, ellipsis: "…", want: "ab   "},
		{name: "leaves an exact string", in: "abcde", width: 5, ellipsis: "…", want: "abcde"},
		{name: "truncates a long string", in: "abcdefgh", width: 5, ellipsis: "…", want: "abcd…"},
		{name: "zero width", in: "abcdefgh", width: 0, ellipsis: "…", want: ""},
		{
			name: "cjk pads the cell the marker could not use",
			in:   "日本語テキスト", width: 6, ellipsis: "…", want: "日本… ",
		},
		{
			name: "cjk that cannot use its last cell is padded",
			in:   "日本語", width: 5, ellipsis: "", want: "日本 ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Fit(tc.in, tc.width, tc.ellipsis)
			if got != tc.want {
				t.Fatalf("Fit(%q, %d, %q) = %q, want %q", tc.in, tc.width, tc.ellipsis, got, tc.want)
			}
			if Width(got) != tc.width {
				t.Fatalf("Fit(%q, %d, %q) = %q, width %d, want exactly %d",
					tc.in, tc.width, tc.ellipsis, got, Width(got), tc.width)
			}
		})
	}
}

// fitCorpus is the fixed part of the width property corpus. Every entry is a
// shape a real session title has taken: ASCII, CJK, emoji with and without a
// presentation selector, combining marks, and mixed scripts.
var fitCorpus = []string{
	"",
	"a",
	"ab",
	"hello world",
	"a very long session title that will certainly not fit",
	"日本語テキスト",
	"日本語 mixed テキスト",
	"\U0001F600\U0001F600\U0001F600",
	"\U0001F468\u200d\U0001F469\u200d\U0001F467 family",
	"⚠\ufe0f warning: the build failed",
	"✅ done",
	"cafe\u0301 crème brûlée",
	"┌──────┐",
	"    ",
	"\u200b\u200b\u200b",
}

// randomDisplayString builds a deterministic pseudo-random string from the same
// character classes the corpus covers.
func randomDisplayString(rng *rand.Rand) string {
	alphabet := []rune{
		'a', 'b', 'Z', '7', ' ', '-', '_', '/', '.',
		'é', 'ñ', 'ü',
		'日', '本', '語', '漢', '字',
		'\U0001F600', '\U0001F680', '⚠', '\ufe0f', '\u200d',
		'\u0301', '\u200b', '▸', '…', '─',
	}
	length := rng.Intn(24)
	var builder strings.Builder
	for i := 0; i < length; i++ {
		builder.WriteRune(alphabet[rng.Intn(len(alphabet))])
	}
	return builder.String()
}

// TestFitIsExactlyTheRequestedWidth is the property the column layout depends
// on: whatever goes in, exactly width cells come out.
func TestFitIsExactlyTheRequestedWidth(t *testing.T) {
	ellipses := []string{"", "…", "...", "▸"}
	inputs := append([]string(nil), fitCorpus...)
	rng := rand.New(rand.NewSource(20260822))
	for i := 0; i < 300; i++ {
		inputs = append(inputs, randomDisplayString(rng))
	}

	for _, ellipsis := range ellipses {
		for _, in := range inputs {
			for width := 0; width <= 24; width++ {
				got := Fit(in, width, ellipsis)
				if Width(got) != width {
					t.Fatalf("Width(Fit(%q, %d, %q)) = %d (%q), want %d",
						in, width, ellipsis, Width(got), got, width)
				}
			}
		}
	}
}

// TestTruncateNeverExceedsWidth is the half of the property that Fit cannot
// paper over: padding can widen a short cut, but nothing shortens a long one.
func TestTruncateNeverExceedsWidth(t *testing.T) {
	ellipses := []string{"", "…", "...", "▸"}
	inputs := append([]string(nil), fitCorpus...)
	rng := rand.New(rand.NewSource(11235813))
	for i := 0; i < 300; i++ {
		inputs = append(inputs, randomDisplayString(rng))
	}

	for _, ellipsis := range ellipses {
		for _, in := range inputs {
			for width := 0; width <= 24; width++ {
				got := Truncate(in, width, ellipsis)
				if Width(got) > width {
					t.Fatalf("Width(Truncate(%q, %d, %q)) = %d (%q), want at most %d",
						in, width, ellipsis, Width(got), got, width)
				}
			}
		}
	}
}

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain text is unchanged", in: "resume the session", want: "resume the session"},
		{name: "leading and trailing space is trimmed", in: "   padded   ", want: "padded"},
		{name: "whitespace runs collapse", in: "a \t\n\r  b", want: "a b"},
		{name: "newlines become one space", in: "line one\nline two", want: "line one line two"},
		{
			name: "an ansi colour sequence loses its escape byte",
			in:   "\x1b[31mred\x1b[0m",
			want: "[31mred [0m",
		},
		{
			name: "a cursor-moving sequence cannot survive",
			in:   "before\x1b[2J\x1b[Hafter",
			want: "before [2J [Hafter",
		},
		{name: "a bare escape is dropped at the start", in: "\x1bok", want: "ok"},
		{name: "nul and other control bytes go", in: "a\x00\x01b", want: "a b"},
		{name: "the bell is not rung", in: "wake\x07up", want: "wake up"},
		{name: "byte order mark is removed", in: "\ufeffhello", want: "hello"},
		{name: "an interior byte order mark is removed", in: "he\ufeffllo", want: "hello"},
		{name: "zero width space is removed", in: "a\u200bb", want: "ab"},
		{name: "zero width joiner is removed", in: "a\u200db", want: "ab"},
		{
			name: "a right-to-left override cannot reorder the line",
			in:   "safe\u202egnp.exe",
			want: "safegnp.exe",
		},
		{name: "a left-to-right mark is removed", in: "a\u200eb", want: "ab"},
		{name: "a non-breaking space collapses like a space", in: "a\u00a0b", want: "a b"},
		{name: "cjk survives", in: "日本語 テキスト", want: "日本語 テキスト"},
		{name: "emoji survives", in: "ship it \U0001F680", want: "ship it \U0001F680"},
		{name: "whitespace only", in: " \t\n ", want: ""},
		{name: "control characters only", in: "\x00\x01\x1b", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Sanitize(tc.in)
			if got != tc.want {
				t.Fatalf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if strings.ContainsRune(got, 0x1b) {
				t.Fatalf("Sanitize(%q) = %q, still carries an escape byte", tc.in, got)
			}
		})
	}
}

// TestSanitizeInvariants states the guarantees the rest of the UI relies on
// rather than the exact output of one input.
func TestSanitizeInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(31415926))
	inputs := append([]string(nil), fitCorpus...)
	inputs = append(inputs,
		"\x1b[31m\ufeff bad \u202e input \x00\n",
		"\t\t tabs \t and \n newlines \r\n",
	)
	for i := 0; i < 200; i++ {
		inputs = append(inputs, randomDisplayString(rng)+"\x1b[1m\u202e\x00")
	}

	for _, in := range inputs {
		got := Sanitize(in)
		for _, r := range got {
			if unicode.IsControl(r) {
				t.Fatalf("Sanitize(%q) = %q, kept control rune %U", in, got, r)
			}
			if unicode.Is(unicode.Cf, r) {
				t.Fatalf("Sanitize(%q) = %q, kept format rune %U", in, got, r)
			}
			if unicode.IsSpace(r) && r != ' ' {
				t.Fatalf("Sanitize(%q) = %q, kept non-space whitespace %U", in, got, r)
			}
		}
		if strings.Contains(got, "  ") {
			t.Fatalf("Sanitize(%q) = %q, kept a whitespace run", in, got)
		}
		if got != strings.TrimSpace(got) {
			t.Fatalf("Sanitize(%q) = %q, is not trimmed", in, got)
		}
	}
}

func TestPreview(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "short text passes through", in: "fix the flaky test", want: "fix the flaky test"},
		{name: "sanitizes before bounding", in: "\x1b[31mred\x1b[0m", want: "[31mred [0m"},
		{
			name: "exactly at the limit is untouched",
			in:   strings.Repeat("a", PreviewLimit),
			want: strings.Repeat("a", PreviewLimit),
		},
		{
			name: "one past the limit is cut and marked",
			in:   strings.Repeat("a", PreviewLimit+1),
			want: strings.Repeat("a", PreviewLimit) + "...",
		},
		{
			name: "a trailing space is trimmed before the marker",
			in:   strings.Repeat("a", PreviewLimit-1) + " bbb",
			want: strings.Repeat("a", PreviewLimit-1) + "...",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Preview(tc.in); got != tc.want {
				t.Fatalf("Preview(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestPreviewCountsCodePoints proves the bound is not a byte bound: a CJK
// prompt is three bytes per code point, so a byte-counting implementation would
// cut it at a third of the allowance.
func TestPreviewCountsCodePoints(t *testing.T) {
	in := strings.Repeat("日", 200)
	got := Preview(in)

	kept := strings.TrimSuffix(got, "...")
	if kept == got {
		t.Fatalf("Preview(200 cjk runes) = %q, want a trailing marker", got)
	}
	if runes := utf8.RuneCountInString(kept); runes != PreviewLimit {
		t.Fatalf("Preview kept %d code points, want %d", runes, PreviewLimit)
	}
	if bytes := len(kept); bytes != PreviewLimit*3 {
		t.Fatalf("Preview kept %d bytes, want %d: the bound must count code points", bytes, PreviewLimit*3)
	}
	if kept != strings.Repeat("日", PreviewLimit) {
		t.Fatal("Preview cut a multi-byte rune in half")
	}
}

func TestWrap(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{name: "zero width", in: "anything", width: 0, want: nil},
		{name: "negative width", in: "anything", width: -3, want: nil},
		{name: "empty input", in: "", width: 10, want: nil},
		{name: "whitespace only", in: "   \t\n ", width: 10, want: nil},
		{name: "fits on one line", in: "one two", width: 10, want: []string{"one two"}},
		{
			name: "breaks on spaces",
			in:   "the quick brown fox", width: 9,
			want: []string{"the quick", "brown fox"},
		},
		{
			name: "collapses the whitespace it splits on",
			in:   "  the   quick  ", width: 9,
			want: []string{"the quick"},
		},
		{
			name: "hard-cuts a word longer than the column",
			in:   "supercalifragilistic word", width: 5,
			want: []string{"super", "calif", "ragil", "istic", "word"},
		},
		{
			name: "a long word after a short one flushes the short one first",
			in:   "hi supercalifragilistic", width: 5,
			want: []string{"hi", "super", "calif", "ragil", "istic"},
		},
		{
			name: "hard-cuts cjk on cell boundaries",
			in:   "日本語テキスト", width: 4,
			want: []string{"日本", "語テ", "キス", "ト"},
		},
		{
			name: "an exactly-fitting word is not cut",
			in:   "abcde fg", width: 5,
			want: []string{"abcde", "fg"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Wrap(tc.in, tc.width)
			if len(got) != len(tc.want) {
				t.Fatalf("Wrap(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("Wrap(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
				}
			}
		})
	}
}

// TestWrapNeverExceedsWidth is the property a pane depends on: a line wider
// than the pane wraps in the terminal and desynchronises every row below it.
func TestWrapNeverExceedsWidth(t *testing.T) {
	rng := rand.New(rand.NewSource(27182818))
	inputs := append([]string(nil), fitCorpus...)
	inputs = append(inputs,
		"supercalifragilisticexpialidocious",
		"日本語テキストの長い行がここにあります",
		"https://example.invalid/a/very/long/path/that/never/breaks",
		"\U0001F600\U0001F600\U0001F600\U0001F600\U0001F600\U0001F600",
	)
	for i := 0; i < 200; i++ {
		inputs = append(inputs, randomDisplayString(rng))
	}

	// Two cells is the narrowest column that can hold any single glyph the
	// corpus contains, so below it the bound is not achievable at all; that
	// degenerate case is covered by TestWrapMakesProgressOnUnfittableGlyphs.
	for _, in := range inputs {
		for width := 2; width <= 24; width++ {
			for _, line := range Wrap(in, width) {
				if Width(line) > width {
					t.Fatalf("Wrap(%q, %d) produced %q of width %d", in, width, line, Width(line))
				}
				if line == "" {
					t.Fatalf("Wrap(%q, %d) produced an empty line", in, width)
				}
			}
		}
	}
}

// TestWrapMakesProgressOnUnfittableGlyphs pins the one case the width bound
// cannot honour: a glyph wider than the whole column. It must be emitted alone
// and the loop must terminate; the earlier implementation cut zero bytes and
// span forever, appending empty lines until the process died.
func TestWrapMakesProgressOnUnfittableGlyphs(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{name: "cjk in a one-cell column", in: "日本", width: 1, want: []string{"日", "本"}},
		{name: "emoji in a one-cell column", in: "\U0001F600", width: 1, want: []string{"\U0001F600"}},
		{
			name: "mixed content in a one-cell column",
			in:   "a日", width: 1, want: []string{"a", "日"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan []string, 1)
			go func() { done <- Wrap(tc.in, tc.width) }()
			select {
			case got := <-done:
				if len(got) != len(tc.want) {
					t.Fatalf("Wrap(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
				}
				for i := range got {
					if got[i] != tc.want[i] {
						t.Fatalf("Wrap(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
					}
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("Wrap(%q, %d) did not terminate", tc.in, tc.width)
			}
		})
	}
}

// TestWrapKeepsEveryGlyph checks that wrapping is a layout decision and never a
// content decision: nothing is dropped on the way to the screen.
func TestWrapKeepsEveryGlyph(t *testing.T) {
	rng := rand.New(rand.NewSource(16180339))
	inputs := []string{
		"the quick brown fox jumps over the lazy dog",
		"supercalifragilisticexpialidocious rules",
		"日本語テキストの長い行",
		"⚠\ufe0f warning warning warning",
	}
	for i := 0; i < 100; i++ {
		inputs = append(inputs, randomDisplayString(rng))
	}

	for _, in := range inputs {
		for width := 2; width <= 12; width++ {
			want := strings.Join(strings.Fields(in), "")
			got := strings.Join(Wrap(in, width), "")
			got = strings.ReplaceAll(got, " ", "")
			if got != want {
				t.Fatalf("Wrap(%q, %d) lost or invented content: joined %q, want %q", in, width, got, want)
			}
		}
	}
}
