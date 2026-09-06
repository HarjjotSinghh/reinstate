package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDeviceHomeDistinguishable(t *testing.T) {
	a := BuildDeviceHome(`D:\lab`, "device-a")
	b := BuildDeviceHome(`D:\lab`, "device-b")
	if a.Project == b.Project {
		t.Fatalf("device-a and device-b share a project path: %q", a.Project)
	}
	if a.ReinstateHome == b.ReinstateHome || a.ClaudeConfigDir == b.ClaudeConfigDir ||
		a.CodexHome == b.CodexHome || a.XDGDataHome == b.XDGDataHome {
		t.Fatalf("device-a and device-b share a home path:\na=%+v\nb=%+v", a, b)
	}
}

func TestClaudeProjectSlug(t *testing.T) {
	got := claudeProjectSlug(`C:\Users\fixture-user\code\device-a`)
	want := "C--Users-fixture-user-code-device-a"
	if got != want {
		t.Fatalf("claudeProjectSlug = %q, want %q", got, want)
	}
}

func TestRewriteFixtureDistinctPerDevice(t *testing.T) {
	raw := `{"type":"meta","sessionId":"session-syn-001","cwd":"C:\\Users\\fixture-user\\code\\demo"}`
	a := rewriteFixture(raw, "a")
	b := rewriteFixture(raw, "b")
	if a == b {
		t.Fatalf("rewriteFixture produced the same content for both devices: %q", a)
	}
	if !strings.Contains(a, "session-syn-001-a") || !strings.Contains(a, `device-a`) {
		t.Fatalf("device a rewrite = %q", a)
	}
	if !strings.Contains(b, "session-syn-001-b") || !strings.Contains(b, `device-b`) {
		t.Fatalf("device b rewrite = %q", b)
	}
	if strings.Contains(a, "demo") || strings.Contains(b, "demo") {
		t.Fatalf("the original fixture project name leaked through: a=%q b=%q", a, b)
	}
}

func TestDeviceHomeSeed(t *testing.T) {
	repoRoot, err := findRepoRoot("")
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	lab := t.TempDir()
	a := BuildDeviceHome(lab, "device-a")
	b := BuildDeviceHome(lab, "device-b")
	if err := a.Seed(repoRoot); err != nil {
		t.Fatalf("seed device-a: %v", err)
	}
	if err := b.Seed(repoRoot); err != nil {
		t.Fatalf("seed device-b: %v", err)
	}

	// Claude: one session file each, under a slug matching this device's
	// own project path, none of them referencing the other device.
	aClaudeDir := filepath.Join(a.ClaudeConfigDir, "projects", claudeProjectSlug(a.Project))
	aClaude, err := os.ReadFile(filepath.Join(aClaudeDir, "session-syn-001-a.jsonl"))
	if err != nil {
		t.Fatalf("device-a claude session: %v", err)
	}
	if strings.Contains(string(aClaude), "device-b") {
		t.Fatalf("device-a's claude session mentions device-b: %q", aClaude)
	}

	// Codex: same shape.
	aCodex, err := os.ReadFile(filepath.Join(a.CodexHome, "sessions", "rollout-syn-001-a.jsonl"))
	if err != nil {
		t.Fatalf("device-a codex session: %v", err)
	}
	if !strings.Contains(string(aCodex), "device-a") {
		t.Fatalf("device-a's codex session does not reference its own project: %q", aCodex)
	}

	// OpenCode: a real, queryable sqlite database with one session whose
	// directory is this device's project path.
	dbPath := filepath.Join(a.XDGDataHome, "opencode", "opencode.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatalf("open device-a opencode.db: %v", err)
	}
	defer func() { _ = db.Close() }()
	var directory string
	if err := db.QueryRow("SELECT directory FROM session WHERE id = ?", "ses_fixture001a").Scan(&directory); err != nil {
		t.Fatalf("query device-a session: %v", err)
	}
	if directory != a.Project {
		t.Fatalf("device-a opencode session directory = %q, want %q", directory, a.Project)
	}
}
