package catalog

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestShippedAgentsRegisterAtDeclaredTiers(t *testing.T) {
	want := []struct {
		key    string
		tier   agents.Tier
		family agents.StorageFamily
		min    string
		max    string
	}{
		{sessionindex.AgentClaude, agents.TierSync, agents.FamilyHomeTree, "2.1.219", "2.1.238"},
		{sessionindex.AgentCodex, agents.TierSync, agents.FamilyHomeTree, "0.133.0", "0.149.0"},
		{sessionindex.AgentGemini, agents.TierHandoffFrom, agents.FamilyHomeTree, "0.55.1", "0.55.1"},
		{sessionindex.AgentOpenCode, agents.TierHandoffTo, agents.FamilyEmbeddedDB, "1.18.21", "1.18.21"},
		{sessionindex.AgentGrok, agents.TierHandoffTo, agents.FamilyHomeTree, "1.0.5", "1.0.5"},
	}
	keys := agents.Keys()
	for _, tt := range want {
		if !contains(keys, tt.key) {
			t.Fatalf("Keys() missing shipped agent %q: %v", tt.key, keys)
		}
	}
	for _, tt := range want {
		got, ok := agents.Get(tt.key)
		if !ok {
			t.Fatalf("Get(%q) missing", tt.key)
		}
		if got.Tier != tt.tier || got.Family != tt.family {
			t.Fatalf("%s tier/family = %s/%s, want %s/%s", tt.key, got.Tier, got.Family, tt.tier, tt.family)
		}
		if tt.min != "" {
			if got.Version == nil || got.Version.Min != tt.min || got.Version.Max != tt.max {
				t.Fatalf("%s version = %+v, want %s–%s", tt.key, got.Version, tt.min, tt.max)
			}
		} else if got.Native != nil {
			t.Fatalf("%s T2 descriptor must not set NativeSpec: %+v", tt.key, got.Native)
		}
		if got.NewIndexSource == nil || got.NewReader == nil {
			t.Fatalf("%s missing T2 constructors", tt.key)
		}
		if tt.tier == agents.TierSync && (got.NewTarget == nil || got.NewSyncAdapter == nil) {
			t.Fatalf("%s missing T5 constructors", tt.key)
		}
		assertEvidenceExists(t, got)
	}
}

func TestVersionParsersMatchAgentcheckShape(t *testing.T) {
	tests := []struct {
		name   string
		parse  func(agents.VersionOutput) (string, bool)
		output agents.VersionOutput
		want   string
		ok     bool
	}{
		{name: "claude canonical", parse: parseClaudeVersion, output: agents.VersionOutput{Stdout: "2.1.220 (Claude Code)\n"}, want: "2.1.220", ok: true},
		{name: "claude stderr", parse: parseClaudeVersion, output: agents.VersionOutput{Stdout: "2.1.220 (Claude Code)\n", Stderr: "warn\n"}},
		{name: "codex canonical", parse: parseCodexVersion, output: agents.VersionOutput{Stdout: "codex-cli 0.147.0\n"}, want: "0.147.0", ok: true},
		{name: "codex reject suffix", parse: parseCodexVersion, output: agents.VersionOutput{Stdout: "codex-cli 0.147.0-beta.1\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.parse(tt.output)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parse() = %q, %t, want %q, %t", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestClaudeAndCodexExcludedComeFromAdapters(t *testing.T) {
	claude, _ := agents.Get(sessionindex.AgentClaude)
	if !contains(claude.Storage.Excluded, "subagents") || !contains(claude.Storage.Excluded, "**/auth.json") {
		t.Fatalf("claude excluded = %v", claude.Storage.Excluded)
	}
	codex, _ := agents.Get(sessionindex.AgentCodex)
	if !contains(codex.Storage.Excluded, "**/auth.json") || !contains(codex.Storage.Excluded, "**/.codex/auth.json") {
		t.Fatalf("codex excluded = %v", codex.Storage.Excluded)
	}
}

func assertEvidenceExists(t *testing.T, d agents.Descriptor) {
	t.Helper()
	root := repoRoot(t)
	for _, path := range append(append([]string{}, d.Evidence.Fixtures...), d.Evidence.DeviceReports...) {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("%s evidence %s: %v", d.Key, path, err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
