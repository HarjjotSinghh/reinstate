package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

// TestGrokVersionParsesShippedOutput pins the parser to bytes actually printed
// by the shipped CLI on both acceptance hosts, rather than to a transcription
// of them. The release channel suffix was missing from the first reading, and
// with it absent the pattern matched neither platform: every Grok resume failed
// closed as UNTESTED and exited 5, on macOS as well as Windows.
func TestGrokVersionParsesShippedOutput(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name, line, want string
	}{
		{"macos 1.0.5", "grok 1.0.5 (5115b46bc909) [stable]", "1.0.5"},
		{"windows 1.0.5", "grok 1.0.5 (5115b46bc9) [stable]", "1.0.5"},
		{"no channel", "grok 1.0.5 (5115b46bc909)", "1.0.5"},
		{"no build id", "grok 1.0.5", "1.0.5"},
		{"channel without build id", "grok 1.0.5 [beta]", "1.0.5"},
		// Measured on the acceptance host, 2026-09-08, after it self-updated
		// past 1.0.5 mid-cycle: `grok --version` =
		// "grok 1.0.13 (5e9a58528b76) [stable]". Same shape as 1.0.5 — a hex
		// build id and a [stable] channel — so no pattern change was needed
		// to widen the range. See
		// docs/testing/results/2026-09-08-windows-range-widening-grok-v060.md.
		{"windows 1.0.13", "grok 1.0.13 (5e9a58528b76) [stable]", "1.0.13"},
		{"no channel 1.0.13", "grok 1.0.13 (5e9a58528b76)", "1.0.13"},
		{"no build id 1.0.13", "grok 1.0.13", "1.0.13"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := parseGrokVersion(agents.VersionOutput{Stdout: testCase.line})
			if !ok {
				t.Fatalf("parseGrokVersion(%q) failed; agentcheck would report UNTESTED", testCase.line)
			}
			if got != testCase.want {
				t.Fatalf("parseGrokVersion(%q) = %q, want %q", testCase.line, got, testCase.want)
			}
		})
	}
}

// TestGrokVersionStillFailsClosed keeps the suffix from becoming a hole: only
// metadata in the shapes the vendor prints is tolerated, and anything else is
// still UNTESTED rather than guessed at.
func TestGrokVersionStillFailsClosed(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"grok",
		"grok 1.0",
		"grok 1.0.5 extra",
		"grok 1.0.5 (build) [stable] trailing",
		"not-grok 1.0.5 [stable]",
		"grok 1.0.5 [stable] (build)",
	} {
		if got, ok := parseGrokVersion(agents.VersionOutput{Stdout: line}); ok {
			t.Errorf("parseGrokVersion(%q) = %q, want a closed failure", line, got)
		}
	}
}
