package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const secretSentinel = "PHASE3-SECRET-SENTINEL-DO-NOT-LEAK"

func TestDiscoverDarwinNameOnlyInventory(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	workdir := filepath.Join(project, "sub")
	managed := filepath.Join(root, "managed")
	codexHome := filepath.Join(home, ".codex")

	writeTestFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "claude-user", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(home, ".claude", "commands", "legacy.md"), secretSentinel)
	writeTestFile(t, filepath.Join(home, ".agents", "skills", "codex-user", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(codexHome, "AGENTS.md"), secretSentinel)
	writeTestFile(t, filepath.Join(codexHome, "config.toml"), `
[mcp_servers.codex-user-mcp]
command = "`+secretSentinel+`"
args = ["https://secret.example/`+secretSentinel+`"]
[mcp_servers.codex-user-mcp.env]
TOKEN = "`+secretSentinel+`"
`)

	writeTestFile(t, filepath.Join(project, "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, ".claude", "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, "CLAUDE.local.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, ".claude", "rules", "security.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, ".claude", "skills", "claude-project", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, ".agents", "skills", "codex-project", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, "AGENTS.md"), secretSentinel)
	writeTestFile(t, filepath.Join(workdir, "AGENTS.override.md"), secretSentinel)
	writeTestFile(t, filepath.Join(project, ".mcp.json"), `{
  "mcpServers": {
    "claude-project-mcp": {
      "command": "`+secretSentinel+`",
      "args": ["--token", "`+secretSentinel+`"],
      "env": {"SECRET": "`+secretSentinel+`"},
      "url": "https://secret.example/`+secretSentinel+`",
      "headers": {"Authorization": "Bearer `+secretSentinel+`"}
    }
  }
}`)
	writeTestFile(t, filepath.Join(project, ".codex", "config.toml"), `
[mcp_servers.codex-project-mcp]
url = "https://secret.example/`+secretSentinel+`"
bearer_token_env_var = "`+secretSentinel+`"
`)
	writeTestFile(t, filepath.Join(home, ".claude.json"), fmt.Sprintf(`{
  "mcpServers": {"claude-user-mcp": {"command": %q}},
  "projects": {
    %q: {"mcpServers": {"claude-local-mcp": {"env": {"TOKEN": %q}}}},
    "/unrelated/private/project": {"mcpServers": {%q: {"command": %q}}}
  },
  "oauthAccount": {"accessToken": %q}
}`, secretSentinel, project, secretSentinel, secretSentinel, secretSentinel, secretSentinel))

	claudeManaged := filepath.Join(managed, "Library", "Application Support", "ClaudeCode")
	writeTestFile(t, filepath.Join(claudeManaged, "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(claudeManaged, "managed-mcp.json"), `{"mcpServers":{"claude-managed-mcp":{"url":"https://secret.example/`+secretSentinel+`"}}}`)
	writeTestFile(t, filepath.Join(managed, "etc", "codex", "skills", "codex-managed", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(managed, "etc", "codex", "config.toml"), `
[mcp_servers.codex-managed-mcp]
command = "`+secretSentinel+`"
`)

	got := Discover(Options{
		GOOS:        "darwin",
		UserHome:    home,
		CodexHome:   codexHome,
		ProjectRoot: project,
		WorkingDir:  workdir,
		ManagedRoot: managed,
	})

	for _, want := range []Item{
		{Agent: AgentClaude, Kind: KindInstruction, Name: "CLAUDE.md", Scope: ScopeManaged, State: StateCandidate, SourceKind: SourceClaudeMemory},
		{Agent: AgentClaude, Kind: KindInstruction, Name: "security", Scope: ScopeProject, State: StateCandidate, SourceKind: SourceClaudeRule, Lazy: true},
		{Agent: AgentClaude, Kind: KindInstruction, Name: "CLAUDE.local.md", Scope: ScopeLocal, State: StateCandidate, SourceKind: SourceClaudeMemory},
		{Agent: AgentClaude, Kind: KindSkill, Name: "claude-user", Scope: ScopeUser, State: StateCandidate, SourceKind: SourceClaudeSkill},
		{Agent: AgentClaude, Kind: KindSkill, Name: "legacy", Scope: ScopeUser, State: StateCandidate, SourceKind: SourceClaudeLegacyCmd},
		{Agent: AgentCodex, Kind: KindSkill, Name: "codex-managed", Scope: ScopeManaged, State: StateCandidate, SourceKind: SourceCodexSkill},
		{Agent: AgentCodex, Kind: KindInstruction, Name: "AGENTS.override.md", Scope: ScopeProject, State: StateCandidate, SourceKind: SourceCodexInstruction},
		{Agent: AgentClaude, Kind: KindMCP, Name: "claude-user-mcp", Scope: ScopeUser, State: StateDeclared, SourceKind: SourceClaudeStateJSON, Transport: TransportStdio},
		{Agent: AgentClaude, Kind: KindMCP, Name: "claude-local-mcp", Scope: ScopeLocal, State: StateDeclared, SourceKind: SourceClaudeStateJSON, Transport: TransportUnknown},
		{Agent: AgentClaude, Kind: KindMCP, Name: "claude-project-mcp", Scope: ScopeProject, State: StateDeclared, SourceKind: SourceClaudeMCPJSON, Transport: TransportUnknown},
		{Agent: AgentClaude, Kind: KindMCP, Name: "claude-managed-mcp", Scope: ScopeManaged, State: StateDeclared, SourceKind: SourceClaudeManagedMCP, Transport: TransportUnknown},
		{Agent: AgentCodex, Kind: KindMCP, Name: "codex-user-mcp", Scope: ScopeUser, State: StateDeclared, SourceKind: SourceCodexMCPConfigTOML, Transport: TransportStdio},
		{Agent: AgentCodex, Kind: KindMCP, Name: "codex-project-mcp", Scope: ScopeProject, State: StateDeclared, SourceKind: SourceCodexMCPConfigTOML, Transport: TransportHTTP},
		{Agent: AgentCodex, Kind: KindMCP, Name: "codex-managed-mcp", Scope: ScopeManaged, State: StateDeclared, SourceKind: SourceCodexMCPConfigTOML, Transport: TransportStdio},
	} {
		if !containsItem(got.Items, want) {
			t.Errorf("missing item %+v\nall: %+v", want, got.Items)
		}
	}

	assertInventoryPrivate(t, got, root, secretSentinel, "secret.example", "Authorization", "accessToken", "/unrelated/private/project")
	assertSorted(t, got)
	if got2 := Discover(Options{GOOS: "darwin", UserHome: home, CodexHome: codexHome, ProjectRoot: project, WorkingDir: workdir, ManagedRoot: managed}); !reflect.DeepEqual(got, got2) {
		t.Fatal("inventory is not deterministic across identical scans")
	}
	if got2 := DiscoverContext(context.Background(), Options{GOOS: "darwin", UserHome: home, CodexHome: codexHome, ProjectRoot: project, WorkingDir: workdir, ManagedRoot: managed}); !reflect.DeepEqual(got, got2) {
		t.Fatal("Discover compatibility wrapper differs from DiscoverContext")
	}
}

func TestDiscoverWindowsManagedLocationsOnly(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	managed := filepath.Join(root, "volume")
	writeTestFile(t, filepath.Join(managed, "Program Files", "ClaudeCode", "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(managed, "Program Files", "ClaudeCode", "managed-mcp.json"), `{"mcpServers":{"windows-managed":{"command":"`+secretSentinel+`"}}}`)
	// /etc/codex is documented as a Unix administrator layer. A Windows scan
	// must not invent it even if a similarly shaped test fixture exists.
	writeTestFile(t, filepath.Join(managed, "etc", "codex", "config.toml"), `[mcp_servers.invented-windows-managed]`)

	got := Discover(Options{GOOS: "windows", UserHome: home, ProjectRoot: project, WorkingDir: project, ManagedRoot: managed})
	if !containsName(got.Items, AgentClaude, KindMCP, ScopeManaged, "windows-managed") {
		t.Fatalf("missing Windows managed Claude MCP: %+v", got.Items)
	}
	if containsName(got.Items, AgentCodex, KindMCP, ScopeManaged, "invented-windows-managed") {
		t.Fatal("invented an undocumented native-Windows Codex managed path")
	}
	assertInventoryPrivate(t, got, root, secretSentinel)
}

func TestSanitizesNamesAndDedupesNormalizedEntries(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	long := strings.Repeat("界", maxNameRunes+20)
	writeTestFile(t, filepath.Join(project, ".mcp.json"), fmt.Sprintf(`{
  "mcpServers": {
    "Alpha": {},
    "alpha": {},
    "\u001b[31mred\u001b[0m\u202eserver": {},
    %q: {}
  }
}`, long))

	got := Discover(Options{GOOS: "darwin", ProjectRoot: project, WorkingDir: project})
	if countNameFold(got.Items, AgentClaude, KindMCP, ScopeProject, "alpha") != 1 {
		t.Fatalf("normalized duplicate was not rejected: %+v", got.Items)
	}
	if !containsName(got.Items, AgentClaude, KindMCP, ScopeProject, "redserver") {
		t.Fatalf("ANSI/bidi name not sanitized: %+v", got.Items)
	}
	for _, item := range got.Items {
		if utf8.RuneCountInString(item.Name) > maxNameRunes {
			t.Fatalf("name exceeds %d runes: %q", maxNameRunes, item.Name)
		}
		if strings.ContainsRune(item.Name, '\x1b') || strings.ContainsRune(item.Name, '\u202e') {
			t.Fatalf("unsafe control survived: %q", item.Name)
		}
	}
}

func TestSymlinksNeverEscapeOrLeakTarget(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	outside := filepath.Join(root, "outside")
	project := filepath.Join(root, "project")
	writeTestFile(t, filepath.Join(outside, secretSentinel, "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(outside, "state.json"), `{"mcpServers":{"`+secretSentinel+`":{"command":"`+secretSentinel+`"}}}`)
	writeTestFile(t, filepath.Join(outside, "codex", "config.toml"), `[mcp_servers.`+secretSentinel+`]`)
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, secretSentinel), filepath.Join(home, ".claude", "skills", "external")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "state.json"), filepath.Join(home, ".claude.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "codex"), filepath.Join(home, ".codex")); err != nil {
		t.Fatal(err)
	}

	got := Discover(Options{GOOS: "darwin", UserHome: home, ProjectRoot: project, WorkingDir: project})
	want := Item{Agent: AgentClaude, Kind: KindSkill, Name: "external", Scope: ScopeUser, State: StateUnverified, SourceKind: SourceClaudeSkill}
	if !containsItem(got.Items, want) {
		t.Fatalf("symlink declaration should be name-only and unverified: %+v", got.Items)
	}
	for _, item := range got.Items {
		if item.State == StateUnverified && item.VerifiedPresence() {
			t.Fatalf("unverified declaration satisfied verified-presence preflight: %+v", item)
		}
	}
	if !hasDiagnostic(got.Diagnostics, AgentClaude, KindSkill, ScopeUser, DiagnosticSymlink) || !hasDiagnostic(got.Diagnostics, AgentClaude, KindMCP, ScopeUser, DiagnosticSymlink) {
		t.Fatalf("missing bounded symlink diagnostics: %+v", got.Diagnostics)
	}
	if !hasDiagnostic(got.Diagnostics, AgentCodex, KindMCP, ScopeUser, DiagnosticSymlink) {
		t.Fatalf("missing Codex-home symlink diagnostic: %+v", got.Diagnostics)
	}
	assertInventoryPrivate(t, got, root, secretSentinel)
}

func TestOversizedAndMalformedConfigsDoNotLeak(t *testing.T) {
	tests := []struct {
		name string
		set  func(t *testing.T, home, project string)
		want DiagnosticCode
	}{
		{
			name: "oversized JSON",
			set: func(t *testing.T, _, project string) {
				body := `{"mcpServers":{"safe":{"command":"` + secretSentinel + `"}},"padding":"` + strings.Repeat("x", int(maxConfigBytes)) + `"}`
				writeTestFile(t, filepath.Join(project, ".mcp.json"), body)
			},
			want: DiagnosticOversized,
		},
		{
			name: "malformed JSON",
			set: func(t *testing.T, home, _ string) {
				writeTestFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"safe":{"command":"`+secretSentinel)
			},
			want: DiagnosticMalformed,
		},
		{
			name: "malformed TOML",
			set: func(t *testing.T, home, _ string) {
				writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `[mcp_servers.safe`+secretSentinel)
			},
			want: DiagnosticMalformed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			project := filepath.Join(root, "project")
			tt.set(t, home, project)
			got := Discover(Options{GOOS: "darwin", UserHome: home, ProjectRoot: project, WorkingDir: project})
			if !containsDiagnosticCode(got.Diagnostics, tt.want) {
				t.Fatalf("missing %s diagnostic: %+v", tt.want, got.Diagnostics)
			}
			assertInventoryPrivate(t, got, root, secretSentinel)
		})
	}
}

func TestEntryAndDepthBounds(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	for i := 0; i < maxEntries+40; i++ {
		writeTestFile(t, filepath.Join(home, ".claude", "skills", fmt.Sprintf("skill-%03d", i), "SKILL.md"), secretSentinel)
	}
	deep := filepath.Join(home, ".agents", "skills")
	for i := 0; i < maxDepth+2; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("level-%d", i))
	}
	writeTestFile(t, filepath.Join(deep, "SKILL.md"), secretSentinel)

	got := Discover(Options{GOOS: "darwin", UserHome: home, ProjectRoot: project, WorkingDir: project})
	if countKind(got.Items, AgentClaude, KindSkill) != maxEntries {
		t.Fatalf("Claude skill count = %d, want %d", countKind(got.Items, AgentClaude, KindSkill), maxEntries)
	}
	if containsName(got.Items, AgentCodex, KindSkill, ScopeUser, "level-9") {
		t.Fatal("skill discovery exceeded the depth bound")
	}
	if !containsDiagnosticCode(got.Diagnostics, DiagnosticLimitReached) {
		t.Fatalf("missing limit diagnostic: %+v", got.Diagnostics)
	}
	assertInventoryPrivate(t, got, root, secretSentinel)
}

func TestDenseTOMLNamesAreBoundedAndDeterministic(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	var body strings.Builder
	for i := maxEntries + 80; i >= 0; i-- {
		fmt.Fprintf(&body, "[mcp_servers.server-%03d]\ncommand = %q\n", i, secretSentinel)
	}
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), body.String())
	opts := Options{GOOS: "darwin", UserHome: home}
	first := Discover(opts)
	second := Discover(opts)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("dense TOML selection is not deterministic")
	}
	if got := countKind(first.Items, AgentCodex, KindMCP); got != maxEntries {
		t.Fatalf("Codex MCP count = %d, want %d", got, maxEntries)
	}
	if !containsName(first.Items, AgentCodex, KindMCP, ScopeUser, "server-000") || containsName(first.Items, AgentCodex, KindMCP, ScopeUser, "server-336") {
		t.Fatalf("bounded TOML selection did not retain the stable lowest names: %+v", first.Items)
	}
	if !hasDiagnostic(first.Diagnostics, AgentCodex, KindMCP, ScopeUser, DiagnosticLimitReached) {
		t.Fatalf("missing TOML limit diagnostic: %+v", first.Diagnostics)
	}
	assertInventoryPrivate(t, first, root, secretSentinel)
}

func TestCustomClaudeHomeOverridesDefaultConfigRoot(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	custom := filepath.Join(root, "custom-claude")
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "default-only", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"default-state-only":{"command":"`+secretSentinel+`"}}}`)
	writeTestFile(t, filepath.Join(custom, "CLAUDE.md"), secretSentinel)
	writeTestFile(t, filepath.Join(custom, "skills", "custom-skill", "SKILL.md"), secretSentinel)
	writeTestFile(t, filepath.Join(custom, "commands", "custom-command.md"), secretSentinel)
	writeTestFile(t, filepath.Join(custom, ".claude.json"), `{"mcpServers":{"custom-state":{"command":"`+secretSentinel+`"}}}`)

	got := Discover(Options{GOOS: "darwin", UserHome: home, ClaudeHome: custom})
	if !containsItem(got.Items, Item{Agent: AgentClaude, Kind: KindInstruction, Name: "CLAUDE.md", Scope: ScopeUser, State: StateCandidate, SourceKind: SourceClaudeMemory}) {
		t.Fatalf("custom Claude instruction root was not scanned: %+v", got.Items)
	}
	if !containsName(got.Items, AgentClaude, KindSkill, ScopeUser, "custom-skill") ||
		!containsName(got.Items, AgentClaude, KindSkill, ScopeUser, "custom-command") {
		t.Fatalf("custom Claude root was not scanned: %+v", got.Items)
	}
	if containsName(got.Items, AgentClaude, KindSkill, ScopeUser, "default-only") {
		t.Fatalf("default Claude root was scanned despite explicit override: %+v", got.Items)
	}
	if !containsName(got.Items, AgentClaude, KindMCP, ScopeUser, "custom-state") ||
		containsName(got.Items, AgentClaude, KindMCP, ScopeUser, "default-state-only") {
		t.Fatalf("custom Claude state root was not isolated: %+v", got.Items)
	}
	assertInventoryPrivate(t, got, root, secretSentinel)
}

func TestDiscoverContextCancellationIsTransactionalAndPrivate(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	writeTestFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), secretSentinel)
	for i := 0; i < 80; i++ {
		writeTestFile(t, filepath.Join(home, ".claude", "skills", fmt.Sprintf("skill-%03d", i), "SKILL.md"), secretSentinel)
	}

	want := Inventory{Items: []Item{}, Diagnostics: []Diagnostic{{Code: DiagnosticCancelled}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := DiscoverContext(ctx, Options{GOOS: "darwin", UserHome: home}); !reflect.DeepEqual(got, want) {
		t.Fatalf("immediate cancellation returned partial inventory: %+v", got)
	}

	firstContext := newStepCancelContext(140)
	first := DiscoverContext(firstContext, Options{GOOS: "darwin", UserHome: home})
	if !firstContext.cancelled || !reflect.DeepEqual(first, want) {
		t.Fatalf("mid-scan cancellation was not transactional: cancelled=%v inventory=%+v", firstContext.cancelled, first)
	}
	secondContext := newStepCancelContext(140)
	second := DiscoverContext(secondContext, Options{GOOS: "darwin", UserHome: home})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cancellation result is not deterministic: first=%+v second=%+v", first, second)
	}
	raw, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if got, wantJSON := string(raw), `{"items":[],"diagnostics":[{"code":"cancelled"}]}`; got != wantJSON {
		t.Fatalf("cancellation JSON changed or leaked context: got %s want %s", got, wantJSON)
	}
	assertInventoryPrivate(t, first, root, secretSentinel)
}

func TestMCPTransportInferenceIsConservativeAndPrivate(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	writeTestFile(t, filepath.Join(project, ".mcp.json"), `{"mcpServers":{
"json-stdio":{"type":"stdio","command":"`+secretSentinel+`"},
"json-http":{"type":"http","url":"https://`+secretSentinel+`.invalid"},
"json-sse":{"type":"sse","url":"https://`+secretSentinel+`.invalid"},
"json-command":{"command":"`+secretSentinel+`"},
"json-url-ambiguous":{"url":"https://`+secretSentinel+`.invalid"},
"json-conflict":{"type":"http","command":"`+secretSentinel+`"},
"json-hostile":{"type":"`+secretSentinel+`","headers":{"Authorization":"`+secretSentinel+`"}}
}}`)
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), `
[mcp_servers.codex-stdio]
command = "`+secretSentinel+`"
[mcp_servers.codex-http]
url = "https://`+secretSentinel+`.invalid"
[mcp_servers.codex-sse]
type = "sse"
url = "https://`+secretSentinel+`.invalid"
[mcp_servers.codex-conflict]
type = "http"
command = "`+secretSentinel+`"
[mcp_servers.codex-hostile]
transport = "`+secretSentinel+`"
command = "`+secretSentinel+`"
`)

	got := Discover(Options{GOOS: "darwin", UserHome: home, ProjectRoot: project, WorkingDir: project})
	for name, want := range map[string]Transport{
		"json-stdio":         TransportStdio,
		"json-http":          TransportHTTP,
		"json-sse":           TransportSSE,
		"json-command":       TransportStdio,
		"json-url-ambiguous": TransportUnknown,
		"json-conflict":      TransportUnknown,
		"json-hostile":       TransportUnknown,
		"codex-stdio":        TransportStdio,
		"codex-http":         TransportHTTP,
		"codex-sse":          TransportSSE,
		"codex-conflict":     TransportUnknown,
		"codex-hostile":      TransportUnknown,
	} {
		if item, ok := findMCPItem(got.Items, name); !ok || item.Transport != want {
			t.Errorf("transport %q = %q (found=%v), want %q", name, item.Transport, ok, want)
		}
	}
	assertInventoryPrivate(t, got, root, secretSentinel, "Authorization", "https://")
}

func TestJSONShapeIsStableAndOmitsRoots(t *testing.T) {
	root := t.TempDir()
	item := Item{Agent: AgentClaude, Kind: KindSkill, Name: "safe", Scope: ScopeUser, State: StateCandidate, SourceKind: SourceClaudeSkill}
	raw, err := json.Marshal(struct {
		Inventory Inventory `json:"inventory"`
		Options   Options   `json:"options"`
	}{
		Inventory: Inventory{Items: []Item{item}},
		Options:   Options{GOOS: "darwin", UserHome: root, ClaudeHome: root, CodexHome: root, ProjectRoot: root, WorkingDir: root, ManagedRoot: root},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"inventory":{"items":[{"agent":"claude","kind":"skill","name":"safe","scope":"user","state":"candidate","source_kind":"claude_skill","lazy":false}]},"options":{"goos":"darwin"}}`
	if string(raw) != want {
		t.Fatalf("JSON shape changed\ngot:  %s\nwant: %s", raw, want)
	}
	if strings.Contains(string(raw), root) {
		t.Fatalf("JSON leaked a root path: %s", raw)
	}
}

func TestInvalidProjectRootsDoNotEscape(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	outside := filepath.Join(root, "outside")
	writeTestFile(t, filepath.Join(outside, "AGENTS.md"), secretSentinel)
	got := Discover(Options{GOOS: "darwin", ProjectRoot: project, WorkingDir: outside})
	if len(got.Items) != 0 {
		t.Fatalf("escaped invalid project root: %+v", got.Items)
	}
	if !containsDiagnosticCode(got.Diagnostics, DiagnosticInvalidRoot) {
		t.Fatalf("missing invalid-root diagnostic: %+v", got.Diagnostics)
	}
	assertInventoryPrivate(t, got, root, secretSentinel)
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsItem(items []Item, want Item) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsName(items []Item, agent Agent, kind Kind, scope Scope, name string) bool {
	for _, item := range items {
		if item.Agent == agent && item.Kind == kind && item.Scope == scope && item.Name == name {
			return true
		}
	}
	return false
}

func countNameFold(items []Item, agent Agent, kind Kind, scope Scope, name string) int {
	count := 0
	for _, item := range items {
		if item.Agent == agent && item.Kind == kind && item.Scope == scope && strings.EqualFold(item.Name, name) {
			count++
		}
	}
	return count
}

func countKind(items []Item, agent Agent, kind Kind) int {
	count := 0
	for _, item := range items {
		if item.Agent == agent && item.Kind == kind {
			count++
		}
	}
	return count
}

func findMCPItem(items []Item, name string) (Item, bool) {
	for _, item := range items {
		if item.Kind == KindMCP && item.Name == name {
			return item, true
		}
	}
	return Item{}, false
}

type stepCancelContext struct {
	done      chan struct{}
	remaining int
	cancelled bool
}

func newStepCancelContext(checks int) *stepCancelContext {
	return &stepCancelContext{done: make(chan struct{}), remaining: checks}
}

func (c *stepCancelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *stepCancelContext) Done() <-chan struct{}       { return c.done }
func (c *stepCancelContext) Value(any) any               { return nil }
func (c *stepCancelContext) Err() error {
	if c.cancelled {
		return context.Canceled
	}
	c.remaining--
	if c.remaining <= 0 {
		c.cancelled = true
		close(c.done)
		return context.Canceled
	}
	return nil
}

func hasDiagnostic(diagnostics []Diagnostic, agent Agent, kind Kind, scope Scope, code DiagnosticCode) bool {
	for _, d := range diagnostics {
		if d.Agent == agent && d.Kind == kind && d.Scope == scope && d.Code == code {
			return true
		}
	}
	return false
}

func containsDiagnosticCode(diagnostics []Diagnostic, code DiagnosticCode) bool {
	for _, d := range diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func assertInventoryPrivate(t *testing.T, inventory Inventory, forbidden ...string) {
	t.Helper()
	raw, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range forbidden {
		if value != "" && strings.Contains(string(raw), value) {
			t.Fatalf("inventory leaked forbidden value %q: %s", value, raw)
		}
	}
}

func assertSorted(t *testing.T, inventory Inventory) {
	t.Helper()
	again := newCollector()
	for _, item := range inventory.Items {
		again.add(item)
	}
	for _, d := range inventory.Diagnostics {
		again.addDiagnostic(d)
	}
	want := again.inventory()
	if !reflect.DeepEqual(inventory, want) {
		t.Fatalf("inventory is not sorted/deduped\ngot:  %+v\nwant: %+v", inventory, want)
	}
}
