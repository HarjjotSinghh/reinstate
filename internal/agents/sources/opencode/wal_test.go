package opencode

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

// stageUncheckpointedStore produces the pair of files OpenCode leaves behind:
// a main database missing its newest session, and a write-ahead log holding it.
//
// OpenCode journals in WAL mode and does not checkpoint on exit, so a session
// the user has just worked in lives entirely in the -wal sidecar until later
// vendor activity happens to cross SQLite's checkpoint threshold — roughly four
// megabytes of writes. On a lightly used install that is a long time; on a new
// install the main database is a 4 KB header and the whole store is the log.
//
// The pair is copied out while the writing connection is still open, because
// closing the last connection checkpoints the log and deletes it. Copying first
// is what leaves an uncheckpointed log behind with no live process holding it,
// which is the state a scan actually meets.
func stageUncheckpointedStore(t *testing.T, target string) {
	t.Helper()
	staging := t.TempDir()
	path := writeStore(t, staging)

	db, err := sql.Open("sqlite",
		"file:"+filepath.ToSlash(path)+"?_pragma=journal_mode(WAL)&_pragma=wal_autocheckpoint(0)")
	if err != nil {
		t.Fatal(err)
	}
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("fixture is journalling in %q, not wal; the test would prove nothing", mode)
	}
	if _, err := db.Exec(`INSERT INTO session VALUES
		('ses_wal','p1','Session still in the log','/work/alpha',1787000030000,1787000040000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO message VALUES ('m_wal','ses_wal','{}')`); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal"} {
		body, readErr := os.ReadFile(path + suffix)
		if readErr != nil {
			t.Fatalf("staging %s: %v", DatabaseName+suffix, readErr)
		}
		if writeErr := os.WriteFile(filepath.Join(target, DatabaseName+suffix), body, 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(target, DatabaseName+"-wal"))
	if err != nil || info.Size() == 0 {
		t.Fatalf("staged store has no write-ahead log; the test would prove nothing")
	}
	// The staged main database must genuinely lack the session, or the log is
	// not load-bearing and every assertion below passes for the wrong reason.
	immutable, err := sql.Open("sqlite",
		"file:"+filepath.ToSlash(filepath.Join(target, DatabaseName))+"?mode=ro&immutable=1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = immutable.Close() }()
	var staged int
	if err := immutable.QueryRow(`SELECT count(*) FROM session WHERE id = 'ses_wal'`).Scan(&staged); err == nil && staged != 0 {
		t.Fatal("the staged main database already contains the session; it was checkpointed")
	}
}

// TestUncheckpointedSessionIsVisible is the regression.
//
// The store was opened with immutable=1, which promises SQLite the file cannot
// change and, as a direct consequence, makes it ignore the write-ahead log. The
// flag was there to stop the reader creating -wal and -shm sidecars under the
// agent root. It did that, and it also hid every session the user had most
// recently worked in — they could not be listed, searched, or resumed.
func TestUncheckpointedSessionIsVisible(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stageUncheckpointedStore(t, root)

	got := scanStore(t, root)
	sort.Strings(got)
	if len(got) != 3 {
		t.Fatalf("scan found %d sessions, want 3 including the one still in the log: %v", len(got), got)
	}
	var found bool
	for _, row := range got {
		if strings.Contains(row, "Session still in the log") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the session in the write-ahead log was not indexed: %v", got)
	}
}

// TestScanWithWriteAheadLogLeavesNoSidecar is the guarantee that constrains the
// fix. TestScanLeavesNoSidecar removes the log before scanning, so it never
// exercised the case where a sidecar could actually be created: opening a WAL
// database in place makes SQLite build a -shm beside it, and writing under a
// vendor's root is not something this project does.
func TestScanWithWriteAheadLogLeavesNoSidecar(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stageUncheckpointedStore(t, root)
	// The vendor process has exited, so no shared-memory file is present. This
	// is the state in which a careless reader creates one.
	_ = os.Remove(filepath.Join(root, DatabaseName+"-shm"))

	before := dirListing(t, root)
	if _, err := os.Stat(filepath.Join(root, DatabaseName+"-shm")); err == nil {
		t.Fatal("fixture still has a -shm; the test cannot detect a new one")
	}

	scanStore(t, root)

	after := dirListing(t, root)
	if len(after) != len(before) {
		t.Fatalf("scan changed the agent root\n before: %v\n after:  %v", before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("scan changed the agent root\n before: %v\n after:  %v", before, after)
		}
	}
}

// TestFingerprintNoticesAWriteAheadLogOnlyChange keeps the read fix reachable.
//
// The fingerprint summarised the main database's path, timestamp and size. A
// session written since the last checkpoint changes only the log, so the
// fingerprint was identical and the incremental refresh skipped the scan
// entirely — the session stayed invisible however well the reader worked.
func TestFingerprintNoticesAWriteAheadLogOnlyChange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	stageUncheckpointedStore(t, root)

	logPath := filepath.Join(root, DatabaseName+"-wal")
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, DatabaseName)
	mainBefore, err := os.Stat(mainPath)
	if err != nil {
		t.Fatal(err)
	}

	// Constructed directly: Fingerprint is how the incremental refresh decides
	// whether to scan at all, and it is not part of the narrower Source
	// interface NewSQLite returns.
	source := &SQLiteSource{env: agents.Env{FixtureRoot: root}}

	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	before, ok, err := source.Fingerprint(context.Background())
	if err != nil || !ok {
		t.Fatalf("fingerprint without the log: ok=%t err=%v", ok, err)
	}

	if err := os.WriteFile(logPath, log, 0o600); err != nil {
		t.Fatal(err)
	}
	after, ok, err := source.Fingerprint(context.Background())
	if err != nil || !ok {
		t.Fatalf("fingerprint with the log: ok=%t err=%v", ok, err)
	}

	mainAfter, err := os.Stat(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if mainAfter.Size() != mainBefore.Size() || !mainAfter.ModTime().Equal(mainBefore.ModTime()) {
		t.Fatal("the main database changed; this no longer tests the log-only case")
	}
	if before == after {
		t.Fatal("a session written only to the write-ahead log left the fingerprint unchanged; " +
			"an incremental refresh would skip the scan that would have found it")
	}
}

func dirListing(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}
