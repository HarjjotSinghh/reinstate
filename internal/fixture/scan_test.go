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
