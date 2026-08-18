package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestParseGeminiVersion(t *testing.T) {
	tests := []struct {
		name   string
		output agents.VersionOutput
		want   string
		ok     bool
	}{
		{name: "canonical", output: agents.VersionOutput{Stdout: "0.23.0\n"}, want: "0.23.0", ok: true},
		{name: "windows newline", output: agents.VersionOutput{Stdout: "0.23.0\r\n"}, want: "0.23.0", ok: true},
		{name: "no trailing newline", output: agents.VersionOutput{Stdout: "0.23.0"}, want: "0.23.0", ok: true},
		{name: "stderr", output: agents.VersionOutput{Stdout: "0.23.0\n", Stderr: "warn\n"}},
		{name: "preview suffix", output: agents.VersionOutput{Stdout: "0.35.0-preview.2\n"}},
		{name: "nightly suffix", output: agents.VersionOutput{Stdout: "0.19.0-nightly\n"}},
		{name: "vendor unknown", output: agents.VersionOutput{Stdout: "unknown\n"}},
		{name: "empty", output: agents.VersionOutput{}},
		{name: "leading whitespace", output: agents.VersionOutput{Stdout: " 0.23.0\n"}},
		{name: "multiple lines", output: agents.VersionOutput{Stdout: "0.23.0\n0.22.0\n"}},
		{name: "script name prefix", output: agents.VersionOutput{Stdout: "gemini 0.23.0\n"}},
		{name: "leading zeros", output: agents.VersionOutput{Stdout: "00.23.0\n"}},
		{name: "ansi", output: agents.VersionOutput{Stdout: "\x1b[32m0.23.0\x1b[0m\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseGeminiVersion(tt.output)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parseGeminiVersion() = %q, %t, want %q, %t", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestGeminiDescriptorStaysT2WithLatestStableRange(t *testing.T) {
	got := Gemini()
	if got.Key != sessionindex.AgentGemini {
		t.Fatalf("Key = %q", got.Key)
	}
	if got.Tier != agents.TierHandoffFrom {
		t.Fatalf("Tier = %s, want T2", got.Tier)
	}
	if got.Native != nil {
		t.Fatalf("T2 Gemini must not set NativeSpec: %+v", got.Native)
	}
	if got.Version == nil || got.Version.Min != "0.55.1" || got.Version.Max != "0.55.1" {
		t.Fatalf("Version = %+v, want 0.55.1–0.55.1", got.Version)
	}
	if got.NewIndexSource == nil || got.NewReader == nil {
		t.Fatal("missing T2 constructors")
	}
	if got.NewTarget != nil || got.NewSyncAdapter != nil {
		t.Fatal("constructors above T2")
	}
}

func TestGeminiExcludesAntigravityProductTrees(t *testing.T) {
	d := Gemini()
	for _, want := range []string{
		"antigravity",
		"antigravity-browser-profile",
		"antigravity-cli",
		"config",
		"history",
		"skills",
		"oauth_creds.json",
		"google_accounts.json",
		"subagents",
	} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %q", d.Storage.Excluded, want)
		}
	}
}
