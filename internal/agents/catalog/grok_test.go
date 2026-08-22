package catalog

import (
	"reflect"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestGrokExcludesInstallNoise(t *testing.T) {
	d := Grok()
	if d.Key != sessionindex.AgentGrok {
		t.Fatalf("Key = %q", d.Key)
	}
	if d.Tier != agents.TierResume {
		t.Fatalf("Tier = %s, want T3", d.Tier)
	}
	for _, want := range []string{
		"bundled",
		"marketplace-cache",
		"bin",
		"downloads",
		"docs",
		"skills",
		"auth.json",
		"auth.json.lock",
		"mcp_credentials.json",
		"subagents",
	} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %q", d.Storage.Excluded, want)
		}
	}
}

// TestGrokNativeArgvMatchesMeasuredHelp pins the argv against the surface
// measured from `grok --help` on Grok Build 1.0.5, not against vendor prose.
func TestGrokNativeArgvMatchesMeasuredHelp(t *testing.T) {
	native := Grok().Native
	if native == nil {
		t.Fatal("Grok T3 descriptor has no NativeSpec")
	}
	if native.Executable != "grok" {
		t.Fatalf("Executable = %q", native.Executable)
	}
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{"resume", native.Resume, []string{"--resume", "{{.SessionID}}"}},
		{"fork", native.Fork, []string{"--resume", "{{.SessionID}}", "--fork-session"}},
		{"continue", native.Continue, []string{"--continue"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !reflect.DeepEqual(test.got, test.want) {
				t.Fatalf("%s argv = %v, want %v", test.name, test.got, test.want)
			}
		})
	}
	if native.InitialPrompt != agents.PromptArgv {
		t.Fatalf("InitialPrompt = %q, want argv", native.InitialPrompt)
	}
}

// TestGrokSessionIDPatternRejectsTitles is the safety property behind the argv.
// `grok --resume [<SESSION_ID_OR_TITLE>]` resolves any value that is not
// UUID-shaped as a session title, and two sessions in one directory can share
// a title, so a title in that position addresses a session Reinstate never
// resolved.
func TestGrokSessionIDPatternRejectsTitles(t *testing.T) {
	native := Grok().Native
	tests := []struct {
		name      string
		sessionID string
		allowed   bool
	}{
		{"uuid", "01987654-3210-7890-abcd-ef0123456789", true},
		{"uuid_uppercase", "01987654-3210-7890-ABCD-EF0123456789", true},
		{"plain_title", "fix the parser", false},
		{"single_word_title", "refactor", false},
		{"non_hex_group", "01987654-basic-0000-0000-000000000001", false},
		{"empty", "", false},
		{"whitespace", "   ", false},
		{"flag_injection", "--fork-session", false},
		{"uuid_with_suffix", "01987654-3210-7890-abcd-ef0123456789x", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := native.SessionIDAllowed(test.sessionID); got != test.allowed {
				t.Fatalf("SessionIDAllowed(%q) = %t, want %t", test.sessionID, got, test.allowed)
			}
		})
	}
}

func TestGrokVersionParser(t *testing.T) {
	tests := []struct {
		name   string
		output agents.VersionOutput
		want   string
		ok     bool
	}{
		{
			// Measured on the macOS acceptance host, 2026-08-22.
			name:   "measured",
			output: agents.VersionOutput{Stdout: "grok 1.0.5 (5115b46bc909)\n"},
			want:   "1.0.5",
			ok:     true,
		},
		{name: "no_build_id", output: agents.VersionOutput{Stdout: "grok 1.0.5\n"}, want: "1.0.5", ok: true},
		{name: "crlf", output: agents.VersionOutput{Stdout: "grok 1.0.5 (5115b46bc909)\r\n"}, want: "1.0.5", ok: true},
		{name: "stderr_present", output: agents.VersionOutput{Stdout: "grok 1.0.5\n", Stderr: "warn\n"}},
		{name: "unparseable", output: agents.VersionOutput{Stdout: "not-a-version\n"}},
		{name: "wrong_product", output: agents.VersionOutput{Stdout: "grok-cli 1.0.5\n"}},
		{name: "prerelease_suffix", output: agents.VersionOutput{Stdout: "grok 1.0.5-beta.1\n"}},
		{name: "two_lines", output: agents.VersionOutput{Stdout: "grok 1.0.5\ngrok 1.0.6\n"}},
		{name: "empty", output: agents.VersionOutput{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := parseGrokVersion(test.output)
			if ok != test.ok || got != test.want {
				t.Fatalf("parseGrokVersion() = %q, %t, want %q, %t", got, ok, test.want, test.ok)
			}
		})
	}
}
