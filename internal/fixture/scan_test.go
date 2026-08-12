package fixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanRejectsAPIKey(t *testing.T) {
	if err := ScanBytes("x", []byte(`token sk-abcdefghijklmnopqrstuvwxyz`)); err == nil {
		t.Fatal("expected reject")
	}
}

func TestScanTreeFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no testdata")
	}
	if err := ScanTree(root); err != nil {
		t.Fatal(err)
	}
}

func TestScanAllowsSynthetic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "session.jsonl")
	content := `{"type":"user","text":"hello from fixture-user Synthetic session"}`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ScanTree(dir); err != nil {
		t.Fatal(err)
	}
}

func TestScanRejectsRealIdentityMarkers(t *testing.T) {
	for _, content := range []string{
		`{"cwd":"/Users/alice/private"}`,
		`{"cwd":"C:\\Users\\bob\\private"}`,
		`{"email":"person@example.net"}`,
		`{"repo":"github.com/private-owner/project"}`,
	} {
		if err := ScanBytes("fixture.jsonl", []byte(content)); err == nil {
			t.Fatalf("accepted non-synthetic marker %q", content)
		}
	}
}

func TestGenerateMatchesCommittedFixtures(t *testing.T) {
	generated := t.TempDir()
	if err := Generate(generated); err != nil {
		t.Fatal(err)
	}
	committedRoot := filepath.Join("..", "..", "testdata", "adapters")
	for relative, expected := range SyntheticFiles {
		generatedBody, err := os.ReadFile(filepath.Join(generated, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		committedBody, err := os.ReadFile(filepath.Join(committedRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("missing committed fixture %s: %v", relative, err)
		}
		if string(generatedBody) != expected || string(committedBody) != expected {
			t.Errorf("fixture drift: %s", relative)
		}
	}
}

func TestGenerateHandoffMatchesCommitted(t *testing.T) {
	committedRoot := filepath.Join("..", "..", "testdata", "handoff")
	if os.Getenv("UPDATE_HANDOFF_FIXTURES") == "1" {
		if err := GenerateHandoff(committedRoot); err != nil {
			t.Fatal(err)
		}
	}
	generated := t.TempDir()
	if err := GenerateHandoff(generated); err != nil {
		t.Fatal(err)
	}
	for _, dir := range HandoffTreeDirs {
		info, err := os.Stat(filepath.Join(committedRoot, filepath.FromSlash(dir)))
		if err != nil || !info.IsDir() {
			t.Fatalf("missing §10 tree dir %s: %v", dir, err)
		}
	}
	check := func(relative string) {
		t.Helper()
		want, err := os.ReadFile(filepath.Join(generated, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(committedRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("missing committed handoff fixture %s: %v", relative, err)
		}
		if string(got) != string(want) {
			t.Errorf("handoff fixture drift: %s", relative)
		}
	}
	for relative := range HandoffSyntheticFiles {
		check(relative)
	}
	check("claude/long-history/projects/-Users-fixture-user-code-demo/session-syn-001.jsonl")
	check("codex/long-history/rollout-2026-08-01T10-00-00-00000000-0000-4000-8000-00000000aa01.jsonl")
}

func TestLongHistoryTurnCounts(t *testing.T) {
	claudeLines := strings.Count(ClaudeLongHistoryJSONL(), "\n")
	if claudeLines != 400 {
		t.Fatalf("claude long-history lines = %d, want 400", claudeLines)
	}
	codex := CodexLongHistoryJSONL()
	users := strings.Count(codex, `"type":"user_message"`)
	if users != 200 {
		t.Fatalf("codex user_message count = %d, want 200", users)
	}
}
