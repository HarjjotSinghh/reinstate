package opencode

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"

	_ "modernc.org/sqlite"
)

// writeStore builds a store shaped like OpenCode's, including the credential
// and account tables the reader must never touch.
func writeStore(t *testing.T, root string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, DatabaseName)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	stmts := []string{
		`CREATE TABLE project (id TEXT PRIMARY KEY, worktree TEXT NOT NULL, name TEXT)`,
		`CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			directory TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE credential (id TEXT PRIMARY KEY, token TEXT NOT NULL)`,
		`CREATE TABLE account (id TEXT PRIMARY KEY, email TEXT NOT NULL)`,
		`INSERT INTO project VALUES ('p1','/work/alpha','alpha'), ('p2','/work/beta',NULL)`,
		`INSERT INTO session VALUES
			('ses_1','p1','First session','/work/alpha',1787000000000,1787000005000),
			('ses_2','p2','Second session','/work/beta',1787000010000,1787000020000)`,
		`INSERT INTO message VALUES ('m1','ses_1','{}'), ('m2','ses_1','{}'), ('m3','ses_2','{}')`,
		`INSERT INTO credential VALUES ('c1','sensitive-value-never-read')`,
		`INSERT INTO account VALUES ('a1','person@example.test')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	return path
}

func scanStore(t *testing.T, root string) []string {
	t.Helper()
	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(result.Records))
	for _, r := range result.Records {
		out = append(out, r.Project+"|"+r.Workspace+"|"+r.Title+"|"+itoa(r.MessageCount))
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

// TestSessionsSpanEveryProject covers Matrix C1. The CLI-query source answered
// only for the directory it ran in, so a single scan could never observe two
// distinct projects no matter how many existed.
func TestSessionsSpanEveryProject(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStore(t, root)
	got := scanStore(t, root)
	if len(got) != 2 {
		t.Fatalf("records = %v, want both sessions", got)
	}
	projects := map[string]bool{}
	for _, r := range got {
		projects[r[:index(r, '|')]] = true
	}
	if len(projects) != 2 {
		t.Fatalf("distinct projects = %v, want two", projects)
	}
	// A project with no name falls back to its directory, never an opaque id.
	if !projects["alpha"] || !projects["beta"] {
		t.Fatalf("projects = %v, want alpha and beta", projects)
	}
}

// TestCredentialTablesAreNeverRead pins the read surface to session, project
// and message. The same store holds credential and account tables.
func TestCredentialTablesAreNeverRead(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStore(t, root)
	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range result.Records {
		blob := r.Title + r.Project + r.Workspace + r.SearchText + r.PromptPreview
		for _, secret := range []string{"sensitive-value-never-read", "person@example.test"} {
			if contains(blob, secret) {
				t.Fatalf("record carries a value from a credential or account table: %q", blob)
			}
		}
	}
}

// TestScanLeavesNoSidecar covers Matrix A10. Opening the store read-write, or
// even read-only without immutable, creates -wal and -shm beside the vendor's
// database, which is a write under an agent root.
func TestScanLeavesNoSidecar(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStore(t, root)
	// Drop anything the fixture writer left so the scan is measured alone.
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(filepath.Join(root, DatabaseName+suffix))
	}
	before, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	scanStore(t, root)
	after, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		var names []string
		for _, e := range after {
			names = append(names, e.Name())
		}
		t.Fatalf("scan changed the agent root: %v", names)
	}
}

// TestMissingStoreIsAbsenceNotFailure keeps an uninstalled agent quiet.
func TestMissingStoreIsAbsenceNotFailure(t *testing.T) {
	t.Parallel()
	source, err := NewSQLite(agents.Env{FixtureRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatalf("absent store returned an error: %v", err)
	}
	if len(result.Records) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("absent store produced %d records and %d warnings",
			len(result.Records), len(result.Warnings))
	}
}

func index(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return len(s)
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return len(needle) == 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
