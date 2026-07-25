package fixture

import (
	"os"
	"path/filepath"
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
