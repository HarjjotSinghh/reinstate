package safetext

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func FuzzTextRenderer(f *testing.F) {
	for _, seed := range []struct {
		value string
		limit uint16
	}{
		{value: "plain metadata", limit: 128},
		{value: " safe\t\n spaced\r text ", limit: 32},
		{value: "safe \x1b[31mred\x1b[0m \x1b]0;title\a end", limit: 128},
		{value: "left\u202eright\u2066isolated\u2069", limit: 64},
		{value: string([]byte{'a', 0xff, 'b', 0xfe, 'c'}), limit: 8},
		{value: strings.Repeat("界", 300), limit: 1},
	} {
		f.Add(seed.value, seed.limit)
	}

	f.Fuzz(func(t *testing.T, value string, rawLimit uint16) {
		limit := int(rawLimit%512) + 1
		got := Text(value, limit)
		if !utf8.ValidString(got) {
			t.Fatalf("Text returned invalid UTF-8: %q", got)
		}
		if count := utf8.RuneCountInString(got); count > limit {
			t.Fatalf("Text returned %d runes with limit %d: %q", count, limit, got)
		}
		if got != strings.TrimSpace(got) || strings.Contains(got, "  ") {
			t.Fatalf("Text returned non-canonical whitespace: %q", got)
		}
		for _, current := range got {
			if unicode.IsControl(current) || unicode.In(current, unicode.Cf) {
				t.Fatalf("forbidden control %U survived in %q", current, got)
			}
			if unicode.IsSpace(current) && current != ' ' {
				t.Fatalf("non-canonical whitespace %U survived in %q", current, got)
			}
		}
		if second := Text(got, limit); second != got {
			t.Fatalf("Text is not idempotent: first=%q second=%q", got, second)
		}
	})
}
