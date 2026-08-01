package adapter_test

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
)

// TestStableVersionFromOutput covers the vendor `--version` shapes the adapters
// have to survive. Reading a fixed field index made a supported vendor report
// as untested whenever the wording or the platform package differed, which
// blocks sync writes and changes setup-check exit codes.
func TestStableVersionFromOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "claude trailing product name", output: "2.1.220 (Claude Code)\n", want: "2.1.220"},
		{name: "codex leading package name", output: "codex-cli 0.145.0\n", want: "0.145.0"},
		{name: "bare version only", output: "0.145.0\n", want: "0.145.0"},
		{name: "windows carriage return", output: "codex-cli 0.145.0\r\n", want: "0.145.0"},
		{name: "v prefix", output: "codex v0.145.0", want: "0.145.0"},
		{name: "parenthesised version", output: "codex (0.145.0)", want: "0.145.0"},
		{name: "extra leading words", output: "OpenAI Codex CLI version 0.146.0", want: "0.146.0"},
		{name: "no version present", output: "not logged in", want: ""},
		{name: "empty output", output: "", want: ""},
		{name: "two component version is not stable", output: "codex 0.145", want: ""},
		{name: "prerelease is not stable", output: "codex 0.145.0-beta.1", want: ""},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := adapter.StableVersionFromOutput(test.output); got != test.want {
				t.Fatalf("StableVersionFromOutput(%q) = %q, want %q", test.output, got, test.want)
			}
		})
	}
}
