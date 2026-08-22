package opencode

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"

	_ "modernc.org/sqlite"
)

// writeResumeStore builds a store with one ordinary session and one whose
// working directory the vendor never recorded.
func writeResumeStore(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, DatabaseName))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	stmts := []string{
		`CREATE TABLE project (id TEXT PRIMARY KEY, worktree TEXT NOT NULL, name TEXT)`,
		`CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			directory TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO project VALUES ('p1','/work/alpha','alpha'), ('p2','','')`,
		`INSERT INTO session VALUES
			('ses_launchable','p1','Launchable','/work/alpha',1787000000000,1787000005000),
			('ses_rootless','p2','Rootless','',1787000010000,1787000020000)`,
	}
	for _, statement := range stmts {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

func recordsByID(t *testing.T, root string) map[string]sessionindex.Record {
	t.Helper()
	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]sessionindex.Record, len(result.Records))
	for _, record := range result.Records {
		out[record.ID] = record
	}
	return out
}

// TestStoreSessionsAreResumableAtT3 pins the capability the tier promotion
// exists to deliver. OpenCode continues a session by id, so a store row with a
// recorded working directory is resumable and forkable, and carries no
// read-only reason.
func TestStoreSessionsAreResumableAtT3(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeResumeStore(t, root)

	record, ok := recordsByID(t, root)["ses_launchable"]
	if !ok {
		t.Fatal("the launchable session was not indexed")
	}
	if !record.CanResume || !record.CanFork {
		t.Fatalf("capabilities = resume:%v fork:%v, want both true", record.CanResume, record.CanFork)
	}
	if record.ReadOnlyReason != "" {
		t.Fatalf("read-only reason = %q, want empty for a resumable session", record.ReadOnlyReason)
	}
}

// TestSessionWithoutWorkingDirectoryStaysReadOnly keeps the promotion honest at
// its edge. OpenCode is launched into a directory, so a row the vendor recorded
// without one has nowhere to go. Offering it and refusing at the launch
// boundary would be a worse answer than saying so in the index.
func TestSessionWithoutWorkingDirectoryStaysReadOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeResumeStore(t, root)

	record, ok := recordsByID(t, root)["ses_rootless"]
	if !ok {
		t.Fatal("the rootless session was not indexed")
	}
	if record.CanResume || record.CanFork {
		t.Fatalf("capabilities = resume:%v fork:%v, want both false", record.CanResume, record.CanFork)
	}
	if record.ReadOnlyReason != readOnlyReasonNoWorkspace {
		t.Fatalf("read-only reason = %q, want %q", record.ReadOnlyReason, readOnlyReasonNoWorkspace)
	}
}

// TestResumePlanningLeavesNoSidecar extends the read-only guarantee across the
// whole T3 path. Resume now reaches this store through the index, and any open
// that is not both read-only and immutable creates -wal and -shm beside the
// vendor's database — a write under an agent root, performed by a tool whose
// contract is that it never writes there.
func TestResumePlanningLeavesNoSidecar(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeResumeStore(t, root)
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(filepath.Join(root, DatabaseName+suffix))
	}

	if record := recordsByID(t, root)["ses_launchable"]; !record.CanResume {
		t.Fatal("the launchable session is not resumable; this test would prove nothing")
	}
	// A refresh before launch scans the store again, so scan twice.
	recordsByID(t, root)

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != DatabaseName {
			t.Fatalf("resume planning left %q beside the vendor store", entry.Name())
		}
	}
}
