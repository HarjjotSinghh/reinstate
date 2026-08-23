package catalog

import (
	"os"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// TestParseOpenCodeVersion pins the grammar against what the real binary
// prints. Measured on macOS from opencode 1.18.21: a bare stable version on
// stdout, nothing on stderr. A bare number is the least distinctive shape any
// vendor uses, so the pattern is anchored at both ends and every near-miss
// below must stay rejected rather than be read as a version.
func TestParseOpenCodeVersion(t *testing.T) {
	tests := []struct {
		name   string
		output agents.VersionOutput
		want   string
		ok     bool
	}{
		{name: "measured", output: agents.VersionOutput{Stdout: "1.18.21\n"}, want: "1.18.21", ok: true},
		{name: "windows newline", output: agents.VersionOutput{Stdout: "1.18.21\r\n"}, want: "1.18.21", ok: true},
		{name: "no trailing newline", output: agents.VersionOutput{Stdout: "1.18.21"}, want: "1.18.21", ok: true},
		{name: "stderr present", output: agents.VersionOutput{Stdout: "1.18.21\n", Stderr: "warn\n"}},
		{name: "prerelease suffix", output: agents.VersionOutput{Stdout: "1.19.0-beta.1\n"}},
		{name: "v prefix", output: agents.VersionOutput{Stdout: "v1.18.21\n"}},
		{name: "name prefix", output: agents.VersionOutput{Stdout: "opencode 1.18.21\n"}},
		{name: "trailing text", output: agents.VersionOutput{Stdout: "1.18.21 (opencode)\n"}},
		{name: "two components", output: agents.VersionOutput{Stdout: "1.18\n"}},
		{name: "four components", output: agents.VersionOutput{Stdout: "1.18.21.3\n"}},
		{name: "leading zeros", output: agents.VersionOutput{Stdout: "01.18.21\n"}},
		{name: "leading whitespace", output: agents.VersionOutput{Stdout: " 1.18.21\n"}},
		{name: "multiple lines", output: agents.VersionOutput{Stdout: "1.18.21\n1.18.20\n"}},
		{name: "ansi", output: agents.VersionOutput{Stdout: "\x1b[32m1.18.21\x1b[0m\n"}},
		{name: "empty", output: agents.VersionOutput{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := parseOpenCodeVersion(test.output)
			if ok != test.ok || got != test.want {
				t.Fatalf("parseOpenCodeVersion() = %q, %t, want %q, %t", got, ok, test.want, test.ok)
			}
		})
	}
}

// TestOpenCodeResumeArgvIsOptionShaped is the shape check that matters for this
// vendor.
//
// OpenCode has no `resume` or `fork` verb. Continuation is an option on its
// default command — `--session <id>`, with `--fork` as a modifier — so an argv
// copied from Codex's `codex fork <id>` would be a positional project path and
// would silently start a new session in a directory named after the id.
func TestOpenCodeResumeArgvIsOptionShaped(t *testing.T) {
	native := OpenCode().Native
	if native == nil {
		t.Fatal("T3 OpenCode must declare a NativeSpec")
	}
	if native.Executable != "opencode" {
		t.Fatalf("Executable = %q", native.Executable)
	}
	tests := []struct {
		name     string
		template []string
		want     []string
	}{
		{name: "resume", template: native.Resume, want: []string{"--session", "ses_abc123"}},
		{name: "fork", template: native.Fork, want: []string{"--session", "ses_abc123", "--fork"}},
		{name: "continue", template: native.Continue, want: []string{"--continue"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if len(test.template) == 0 {
				t.Fatalf("%s argv is not declared", test.name)
			}
			got := sessionindex.ApplyArgvTemplate(test.template, "ses_abc123")
			if strings.Join(got, " ") != strings.Join(test.want, " ") {
				t.Fatalf("argv = %v, want %v", got, test.want)
			}
			for _, arg := range got {
				if !strings.HasPrefix(arg, "-") && arg != "ses_abc123" {
					t.Fatalf("argv %v carries a bare positional %q; OpenCode reads that as a project path", got, arg)
				}
			}
		})
	}
	// Fork is resume plus a modifier, never its own verb.
	if len(native.Fork) <= len(native.Resume) ||
		strings.Join(native.Fork[:len(native.Resume)], " ") != strings.Join(native.Resume, " ") {
		t.Fatalf("Fork %v is not Resume %v plus a modifier", native.Fork, native.Resume)
	}
}

// TestOpenCodeDescriptorIsT4 keeps tier, capability constructors and the
// measured version range together, so a later edit cannot move one alone.
// The sync adapter exists but is not wired until the native Windows T5
// journey is recorded; until then the descriptor must not claim T5.
func TestOpenCodeDescriptorIsT4(t *testing.T) {
	got := OpenCode()
	if got.Key != sessionindex.AgentOpenCode {
		t.Fatalf("Key = %q", got.Key)
	}
	if got.Tier != agents.TierHandoffTo {
		t.Fatalf("Tier = %s, want T4", got.Tier)
	}
	if got.Family != agents.FamilyEmbeddedDB {
		t.Fatalf("Family = %s, want F3", got.Family)
	}
	if got.Version == nil || got.Version.Min != "1.18.21" || got.Version.Max != "1.18.21" {
		t.Fatalf("Version = %+v, want the single measured build 1.18.21", got.Version)
	}
	if got.NewIndexSource == nil || got.NewReader == nil || got.NewTarget == nil {
		t.Fatal("missing T1/T2/T4 constructors")
	}
	if got.NewSyncAdapter != nil {
		t.Fatal("constructors above the declared T4: T5 is not advertised until the Windows journey is recorded")
	}
	if len(got.Process.Images) == 0 {
		t.Fatal("T3 needs a ProcessSpec so a running OpenCode is recognized")
	}
	if len(got.Evidence.DeviceReports) == 0 {
		t.Fatal("T3 requires a device journey")
	}
}

// TestOpenCodeNewSessionArgvMatchesTheDestination keeps the descriptor's
// declared new-session argv and the destination target's launch argv from
// drifting. It carries no {{.SessionID}} on purpose: `opencode --session
// <unknown-id>` refuses with "Session not found" and creates nothing, so the
// destination id is only knowable after launch.
func TestOpenCodeNewSessionArgvMatchesTheDestination(t *testing.T) {
	native := OpenCode().Native
	if native == nil || len(native.NewSession) != 1 {
		t.Fatalf("NewSession = %v, want the vendor's single new-session flag", native.NewSession)
	}
	if native.NewSession[0] != handoff.OpenCodeNewSessionFlag {
		t.Fatalf("NewSession = %v, want %q", native.NewSession, handoff.OpenCodeNewSessionFlag)
	}
	for _, arg := range native.NewSession {
		if strings.Contains(arg, "{{.SessionID}}") {
			t.Fatalf("NewSession %v pins a session id OpenCode will not accept", native.NewSession)
		}
	}
	if native.InitialPrompt != agents.PromptArgv {
		t.Fatalf("InitialPrompt = %q, want argv", native.InitialPrompt)
	}
}

// TestOpenCodeTargetNeverWritesVendorFiles states the design conclusion the T4
// promotion rests on in the place a future contributor will look first.
func TestOpenCodeTargetNeverWritesVendorFiles(t *testing.T) {
	root := t.TempDir()
	target, err := OpenCode().NewTarget(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if target.Name() != sessionindex.AgentOpenCode {
		t.Fatalf("Name = %q", target.Name())
	}
	if target.Capabilities().SupportsPinnedID {
		t.Fatal("SupportsPinnedID = true; OpenCode assigns the session id")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("constructing the destination wrote under the agent root: %d entries", len(entries))
	}
}

// TestOpenCodeRootEnvironmentReachesTheStore keeps the root variable usable for
// the thing it exists for: pointing a probe or a resume at a sanitized root.
// The variable names the parent, so the suffix is not decoration.
func TestOpenCodeRootEnvironmentReachesTheStore(t *testing.T) {
	storage := OpenCode().Storage
	if storage.RootEnv != "XDG_DATA_HOME" || storage.RootEnvSuffix != "opencode" {
		t.Fatalf("root variable = %q + %q, want XDG_DATA_HOME + opencode",
			storage.RootEnv, storage.RootEnvSuffix)
	}
	if storage.Marker == "" || !strings.HasSuffix(storage.Marker, ".db") {
		t.Fatalf("Marker = %q, want the embedded store file", storage.Marker)
	}
}
