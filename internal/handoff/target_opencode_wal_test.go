package handoff

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/transcript"

	_ "modernc.org/sqlite"
)

// TestOpenCodeVerifyResolvesAnUncheckpointedSession pins the behaviour that
// makes this destination advertisable as a tier at all.
//
// OpenCode keeps its writes in a SQLite write-ahead log and does not checkpoint
// them into the database file when it exits. Measured on macOS against OpenCode
// 1.18.21: after a session was created and the vendor quit through its own UI,
// `opencode.db` was 4096 bytes with no `session` table and 543 KB sat in
// `opencode.db-wal`. The `immutable=1` guard that stops Reinstate creating
// sidecars beside a store it does not own is exactly what makes those rows
// invisible.
//
// This test used to assert the honest-but-undesirable outcome — unresolved —
// and instructed a reviewer who made the reader see write-ahead log content to
// check what that change did to the no-write invariant before updating it.
// That change is the private-copy reader: the database and its log are copied
// to a directory this process owns and the copy is opened there, so the log is
// visible and nothing is created beside the vendor's store. Both halves are
// asserted below, because seeing the log would not be worth having if it cost
// the invariant.
func TestOpenCodeVerifyResolvesAnUncheckpointedSession(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	root := t.TempDir()
	path := filepath.Join(root, transcript.OpenCodeDatabaseName)

	// Reproduce the vendor's own on-disk state: WAL journalling with automatic
	// checkpointing switched off, and the connection left open, so every row
	// written below stays in the log exactly as OpenCode leaves it.
	writer, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	for _, statement := range []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA wal_autocheckpoint=0`,
		`CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			directory TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL,
			session_id TEXT NOT NULL, data TEXT NOT NULL)`,
	} {
		if _, err := writer.Exec(statement); err != nil {
			t.Skipf("this SQLite build cannot reproduce the vendor's journalling: %v", err)
		}
	}

	target := NewOpenCodeTarget(&OpenCodeTarget{Root: root})
	plan, _, err := target.Plan(testOpenCodeCapsule(workspace, "goal", "intent"), PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UnixMilli()
	if _, err := writer.Exec(`INSERT INTO session VALUES (?,?,?,?,?,?)`,
		"ses_inwal", "p1", "in the log", workspace, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec(`INSERT INTO message VALUES (?,?,?,?)`,
		"msg_inwal", "ses_inwal", stamp, `{"role":"user"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec(`INSERT INTO part VALUES (?,?,?,?)`,
		"prt_inwal", "msg_inwal", "ses_inwal",
		`{"type":"text","text":`+jsonQuote(string(plan.Bootstrap))+`}`); err != nil {
		t.Fatal(err)
	}

	if info, statErr := os.Stat(path + "-wal"); statErr != nil || info.Size() == 0 {
		t.Skip("this SQLite build checkpointed anyway; the limitation cannot be reproduced here")
	}

	before := vendorDirListing(t, root)

	id, state, err := target.Verify(context.Background(), plan, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("Verify returned an error: %v", err)
	}
	if state != VerifyResolved || id != "ses_inwal" {
		t.Fatalf("Verify = %q/%q, want ses_inwal/resolved; a session that exists only in "+
			"the write-ahead log is exactly the one a handoff has just caused to exist",
			id, state)
	}

	after := vendorDirListing(t, root)
	if len(after) != len(before) {
		t.Fatalf("Verify changed the vendor directory\n before: %v\n after:  %v", before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("Verify changed the vendor directory\n before: %v\n after:  %v", before, after)
		}
	}
}

func vendorDirListing(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		info, statErr := e.Info()
		if statErr != nil {
			t.Fatal(statErr)
		}
		out = append(out, e.Name()+"|"+strconv.FormatInt(info.Size(), 10))
	}
	sort.Strings(out)
	return out
}
