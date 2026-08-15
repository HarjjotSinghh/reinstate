package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestAcceptClaudeWorkspaceTrustMergesExistingConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	existing := []byte(`{"oauthAccount":{"accountUuid":"keep-me"},"projects":{"/other":{"hasTrustDialogAccepted":false}}}` + "\n")
	if err := os.WriteFile(filepath.Join(root, claudeDestConfigName), existing, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := acceptClaudeWorkspaceTrust(root, workspace); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, claudeDestConfigName))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	oauth, _ := doc["oauthAccount"].(map[string]any)
	if oauth["accountUuid"] != "keep-me" {
		t.Fatalf("oauthAccount mutated: %#v", doc["oauthAccount"])
	}
	projects, _ := doc["projects"].(map[string]any)
	proj, _ := projects[workspace].(map[string]any)
	if proj["hasTrustDialogAccepted"] != true {
		t.Fatalf("dest workspace trust = %#v", proj)
	}
	other, _ := projects["/other"].(map[string]any)
	if other["hasTrustDialogAccepted"] != false {
		t.Fatalf("unrelated project mutated: %#v", other)
	}
}

func TestAcceptCodexProjectTrustMergesExistingConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	existing := []byte("[mcp_servers.browser]\ncommand = \"mcp-browser\"\n")
	if err := os.WriteFile(filepath.Join(root, codexDestConfigName), existing, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := acceptCodexProjectTrust(root, workspace); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, codexDestConfigName))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mcp, _ := doc["mcp_servers"].(map[string]any)
	if mcp == nil {
		t.Fatalf("mcp_servers dropped: %s", raw)
	}
	projects, _ := doc["projects"].(map[string]any)
	proj, _ := projects[workspace].(map[string]any)
	if proj["trust_level"] != "trusted" {
		t.Fatalf("dest project trust = %#v", proj)
	}
}

func TestTOMLTableKeyWindowsPathUsesLiteralQuotes(t *testing.T) {
	t.Parallel()
	got := tomlTableKey(`C:\Users\admin\repo`)
	if got != `'C:\Users\admin\repo'` {
		t.Fatalf("tomlTableKey = %s, want literal-quoted windows path", got)
	}
	if strings.Contains(got, `"C:\`) {
		t.Fatal("double-quoted windows path would unicode-escape \\U")
	}
}

func TestWorkspaceTrustKeysWindowsAliases(t *testing.T) {
	t.Parallel()
	got := workspaceTrustKeys("windows", `C:\Users\admin\repo`)
	want := []string{
		`C:\Users\admin\repo`,
		`C:/Users/admin/repo`,
		`c:\users\admin\repo`,
		`c:/users/admin/repo`,
	}
	if len(got) != len(want) {
		t.Fatalf("keys = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("keys = %#v, want %#v", got, want)
		}
	}
}

func TestClaudeTargetMaterializeSkipsDefaultHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	target := &ClaudeTarget{}
	plan := DestinationPlan{
		Executable: "claude",
		Args:       []string{"--session-id", "00000000-0000-4000-8000-000000000011", "x"},
		Dir:        t.TempDir(),
		SessionID:  "00000000-0000-4000-8000-000000000011",
	}
	if err := target.Materialize(context.TODO(), plan); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	for _, rel := range []string{claudeDestConfigName, filepath.Join(".claude", claudeDestConfigName)} {
		if _, err := os.Stat(filepath.Join(home, rel)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Materialize wrote default home %s: %v", rel, err)
		}
	}
}

func TestCodexTargetMaterializeSkipsDefaultHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CODEX_HOME", "")
	target := NewCodexTarget(nil)
	plan := DestinationPlan{
		Executable: "codex",
		Args:       []string{"prompt"},
		Dir:        t.TempDir(),
	}
	if err := target.Materialize(context.TODO(), plan); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", codexDestConfigName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Materialize wrote default ~/.codex: %v", err)
	}
}

func TestCodexTargetMaterializeTrustsExplicitRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := t.TempDir()
	target := NewCodexTarget(&CodexTarget{Root: root})
	plan := DestinationPlan{
		Executable: "codex",
		Args:       []string{"prompt"},
		Dir:        workspace,
	}
	if err := target.Materialize(context.TODO(), plan); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, codexDestConfigName))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	projects, _ := doc["projects"].(map[string]any)
	proj, _ := projects[workspace].(map[string]any)
	if proj["trust_level"] != "trusted" {
		t.Fatalf("trust not recorded: %s", raw)
	}
}

func TestFirstReplyAckOneLineHasNoCRLF(t *testing.T) {
	t.Parallel()
	got := firstReplyAckOneLine()
	if got == "" || strings.ContainsAny(got, "\r\n") {
		t.Fatalf("one-line ack = %q", got)
	}
	for _, needle := range []string{"(1)", "(2)", "(3)", "(4)", "(5)", "First reply"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("one-line ack missing %q: %s", needle, got)
		}
	}
}
