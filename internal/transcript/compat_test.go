package transcript

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// claudeInstallLayout writes a Claude Code layout that matches a real
// installation: projects/<encoded-project>/<session>.jsonl and deliberately no
// <root>/version file, because Claude Code never writes one.
func claudeInstallLayout(t *testing.T) (root, session string) {
	t.Helper()
	root = t.TempDir()
	projects := filepath.Join(root, "projects", "-Users-fixture-user-code-demo")
	if err := os.MkdirAll(projects, 0o700); err != nil {
		t.Fatal(err)
	}
	session = filepath.Join(projects, "00000000-0000-4000-8000-0000000000ab.jsonl")
	body := `{"type":"user","uuid":"u1","cwd":"/Users/fixture-user/code/demo","message":{"role":"user","content":"hello"}}` + "\n"
	if err := os.WriteFile(session, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, session
}

// codexInstallLayout writes a Codex CLI layout: sessions/<rollout>.jsonl.
func codexInstallLayout(t *testing.T) (root, session string) {
	t.Helper()
	root = t.TempDir()
	sessions := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	session = filepath.Join(sessions, "rollout-2026-08-01T16-00-00-00000000-0000-4000-8000-00000000ab01.jsonl")
	body := `{"timestamp":"2026-08-01T16:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}` + "\n"
	if err := os.WriteFile(session, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, session
}

// emptyExecutablePath scopes PATH to an empty directory so no agent executable
// is resolvable. This models the supported case where the source agent is not
// installed on this device at all.
func emptyExecutablePath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// fakeAgentExecutable puts a deterministic <name> on PATH whose `--version`
// stdout is exactly line. It keeps version probes hermetic: no test may depend
// on the contributor's installed agents.
func fakeAgentExecutable(t *testing.T, name, line string) {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		script := filepath.Join(dir, name+".cmd")
		body := "@echo off\r\necho " + line + "\r\n"
		if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	} else {
		script := filepath.Join(dir, name)
		body := "#!/bin/sh\nprintf '%s\\n' " + shellQuote(line) + "\n"
		if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
}

func shellQuote(value string) string {
	return "'" + value + "'"
}

// TestClaudeProbeWithoutVendorVersionFileStaysUsable reproduces the rc.1
// blocker: a real Claude Code installation has no <agent-root>/version file, so
// the probe used to read "unknown", fall outside the verified range, and refuse
// the handoff with UNTESTED. A recognizable layout whose version cannot be
// determined must remain usable as a handoff source.
func TestClaudeProbeWithoutVendorVersionFileStaysUsable(t *testing.T) {
	emptyExecutablePath(t)
	_, session := claudeInstallLayout(t)

	compat, err := (&ClaudeReader{}).Probe(context.Background(), sessionindex.Record{
		Agent: "claude", ID: "00000000-0000-4000-8000-0000000000ab", SourcePath: session,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("probe without a version file = %q, want %q", compat, CompatibilitySupported)
	}
}

// TestCodexProbeWithoutInstalledAgentStaysUsable is the Codex half of the same
// contract.
func TestCodexProbeWithoutInstalledAgentStaysUsable(t *testing.T) {
	emptyExecutablePath(t)
	_, session := codexInstallLayout(t)

	compat, err := (&CodexReader{}).Probe(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: session,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("probe without an installed agent = %q, want %q", compat, CompatibilitySupported)
	}
}

// TestProbeUsesInstalledAgentVersion proves both readers resolve the version
// from the installed executable — the same mechanism internal/agentcheck uses
// for `rein inspect` — and that a determinable out-of-range version is still
// UNTESTED for both agents.
func TestProbeUsesInstalledAgentVersion(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		versionLine string
		want        Compatibility
	}{
		{name: "claude in range", agent: "claude", versionLine: "2.1.228 (Claude Code)", want: CompatibilitySupported},
		{name: "claude above range", agent: "claude", versionLine: "9.9.9 (Claude Code)", want: CompatibilityUntested},
		{name: "claude below range", agent: "claude", versionLine: "2.1.100 (Claude Code)", want: CompatibilityUntested},
		{name: "claude unparseable", agent: "claude", versionLine: "claude code, nightly", want: CompatibilitySupported},
		{name: "codex in range", agent: "codex", versionLine: "codex-cli 0.147.0", want: CompatibilitySupported},
		{name: "codex above range", agent: "codex", versionLine: "codex-cli 0.199.0", want: CompatibilityUntested},
		{name: "codex unparseable", agent: "codex", versionLine: "codex nightly", want: CompatibilitySupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				reader  Reader
				session string
			)
			if test.agent == "claude" {
				reader = &ClaudeReader{}
				_, session = claudeInstallLayout(t)
			} else {
				reader = &CodexReader{}
				_, session = codexInstallLayout(t)
			}
			fakeAgentExecutable(t, test.agent, test.versionLine)

			compat, err := reader.Probe(context.Background(), sessionindex.Record{
				Agent: test.agent, SourcePath: session,
			})
			if err != nil {
				t.Fatal(err)
			}
			if compat != test.want {
				t.Fatalf("probe with %q = %q, want %q", test.versionLine, compat, test.want)
			}
		})
	}
}

// TestProbeUnrecognizedLayoutIsUnsupported keeps layout authoritative for both
// readers even when the installed version is inside the verified range.
func TestProbeUnrecognizedLayoutIsUnsupported(t *testing.T) {
	fakeAgentExecutable(t, "claude", "2.1.228 (Claude Code)")
	stray := filepath.Join(t.TempDir(), "not-projects", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(stray), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stray, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	compat, err := (&ClaudeReader{}).Probe(context.Background(), sessionindex.Record{
		Agent: "claude", SourcePath: stray,
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("stray claude layout = %q, want %q", compat, CompatibilityUnsupported)
	}

	compat, err = (&CodexReader{}).Probe(context.Background(), sessionindex.Record{
		Agent: sessionindex.AgentCodex, SourcePath: filepath.Join(t.TempDir(), "rollout.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if compat != CompatibilityUnsupported {
		t.Fatalf("non-jsonl codex layout = %q, want %q", compat, CompatibilityUnsupported)
	}
}
