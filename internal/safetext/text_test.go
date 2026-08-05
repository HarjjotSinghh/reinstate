package safetext

import (
	"strings"
	"testing"
)

func TestTextRemovesTerminalAndFormatInjection(t *testing.T) {
	t.Parallel()
	secret := "controlled-sentinel"
	input := "safe \x1b[31m" + secret + "\x1b[0m \x1b]0;hidden\a end\u202e"
	got := Text(input, 128)
	if got != "safe "+secret+" end" || strings.ContainsAny(got, "\x1b\u202e") || strings.Contains(got, "[31m") {
		t.Fatalf("Text() = %q", got)
	}
}
