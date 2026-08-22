// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package tuitest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite TUI golden frames instead of comparing")

// AssertGolden compares a rendered frame against testdata/golden/<name>.txt.
//
// Run `go test ./internal/tui/... -update-golden` to rewrite the files after a
// deliberate layout change. Review the diff: a golden frame is the contract for
// what a user sees, so an unexplained change to one is a regression.
func AssertGolden(t *testing.T, name, frame string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".txt")
	normalized := Normalize(frame)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(normalized+"\n"), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\nrerun with -update-golden to create it\n\ngot frame:\n%s", path, err, normalized)
	}
	expected := Normalize(string(want))
	if normalized == expected {
		return
	}
	t.Errorf("frame %s does not match golden\n%s", name, diff(expected, normalized))
}

// diff renders a compact line-oriented comparison. A full-frame dump on both
// sides is unreadable at 40 lines; pointing at the first differing lines is not.
func diff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	var builder strings.Builder
	limit := len(wantLines)
	if len(gotLines) > limit {
		limit = len(gotLines)
	}
	shown := 0
	for i := 0; i < limit; i++ {
		wantLine := ""
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		gotLine := ""
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if wantLine == gotLine {
			continue
		}
		builder.WriteString("line ")
		builder.WriteString(itoa(i + 1))
		builder.WriteString("\n  want: ")
		builder.WriteString(quote(wantLine))
		builder.WriteString("\n  got:  ")
		builder.WriteString(quote(gotLine))
		builder.WriteString("\n")
		shown++
		if shown >= 12 {
			builder.WriteString("... further differences omitted\n")
			break
		}
	}
	return builder.String()
}

func quote(s string) string {
	replacer := strings.NewReplacer("\x1b", "\\x1b", "\t", "\\t")
	return "\"" + replacer.Replace(s) + "\""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
