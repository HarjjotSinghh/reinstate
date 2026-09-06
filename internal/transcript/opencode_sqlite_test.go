package transcript

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"

	_ "modernc.org/sqlite"
)

// buildOpenCodeStore creates a synthetic, multi-session embedded OpenCode
// store under t.TempDir() and returns its data root (the "opencode" directory
// holding opencode.db, matching OpenCodeReader.DataRoot's contract). It seeds
// two sessions ("ses_target" and "ses_other") plus the credential table real
// stores carry, so a test can mutate rows this reader must never let leak into
// a session's boundary.
func buildOpenCodeStore(t *testing.T) string {
	t.Helper()
	dataRoot := filepath.Join(t.TempDir(), "opencode")
	dbPath := filepath.Join(dataRoot, OpenCodeDatabaseName)
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	stmts := []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY, version TEXT)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER, time_updated INTEGER, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER, time_updated INTEGER, data TEXT NOT NULL)`,
		// A table this reader never reads (credentials), and a second session,
		// so a test can churn both without touching ses_target's own rows.
		`CREATE TABLE credential (id TEXT PRIMARY KEY, note TEXT NOT NULL)`,
		`INSERT INTO session (id, version) VALUES ('ses_target', '1')`,
		`INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES
			('msg_target_1', 'ses_target', 1, 1, '{"role":"user","time":{"created":1}}')`,
		`INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES
			('prt_target_1', 'msg_target_1', 'ses_target', 1, 1, '{"type":"text","text":"hello"}')`,
		`INSERT INTO session (id, version) VALUES ('ses_other', '1')`,
		`INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES
			('msg_other_1', 'ses_other', 1, 1, '{"role":"user","time":{"created":1}}')`,
		`INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES
			('prt_other_1', 'msg_other_1', 'ses_other', 1, 1, '{"type":"text","text":"unrelated"}')`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
	return dataRoot
}

// TestOpenCodeDatabaseBoundaryIgnoresUnrelatedStoreChurn is the regression
// test for PD-B1/D5: a real `rein handoff opencode:<id> --to claude --dry-run
// --json` run twice over an unchanged source produced a different
// handoff_id/lineage_root AND a different destination session id, because the
// boundary digest hashed the *entire* opencode.db file. That file is shared by
// every session in the store, and OpenCode's own CLI rewrites parts of it
// (bookkeeping tables this reader never reads) as a side effect of ordinary,
// read-only-looking commands — so two Snapshot() calls over a session whose
// own rows never changed still produced two different digests. This asserts
// the boundary is scoped to exactly the target session's own rows: churn in
// another session, or a table this reader never reads, must never move it.
func TestOpenCodeDatabaseBoundaryIgnoresUnrelatedStoreChurn(t *testing.T) {
	t.Parallel()

	dataRoot := buildOpenCodeStore(t)
	reader := &OpenCodeReader{DataRoot: dataRoot}
	record := sessionindex.Record{ID: "ses_target", Agent: sessionindex.AgentOpenCode}

	before, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot (before): %v", err)
	}
	if isOpenCodeMetadataBoundary(before) {
		t.Fatal("expected a database boundary, got the metadata fallback")
	}

	// Simulate the store-wide churn OpenCode's own CLI performs: write an
	// unrelated table, and edit a *different* session's message. Neither
	// touches ses_target's own session, message, or part rows.
	dbPath := filepath.Join(dataRoot, OpenCodeDatabaseName)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO credential (id, note) VALUES ('cred_new', 'vendor housekeeping write')`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE message SET time_updated = 999, data = '{"role":"user","time":{"created":1,"completed":999}}' WHERE id = 'msg_other_1'`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	after, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot (after): %v", err)
	}
	if before.SHA256 != after.SHA256 {
		t.Fatalf("boundary digest moved from unrelated store churn: before=%s after=%s", before.SHA256, after.SHA256)
	}
	if before.SizeBytes != after.SizeBytes {
		t.Fatalf("boundary size moved from unrelated store churn: before=%d after=%d", before.SizeBytes, after.SizeBytes)
	}

	// Sanity: a real edit to ses_target's own message *does* move the digest,
	// so this is a scoping fix, not an accidental constant.
	db2, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db2.Exec(`UPDATE message SET data = '{"role":"user","time":{"created":1,"completed":2}}' WHERE id = 'msg_target_1'`); err != nil {
		_ = db2.Close()
		t.Fatal(err)
	}
	if err := db2.Close(); err != nil {
		t.Fatal(err)
	}
	changed, err := reader.Snapshot(context.Background(), record)
	if err != nil {
		t.Fatalf("Snapshot (changed): %v", err)
	}
	if changed.SHA256 == after.SHA256 {
		t.Fatal("boundary digest did not move after a real edit to the target session's own message")
	}
}

// TestOpenCodeProbeAnswersFromDatabaseWithoutShellingOut asserts Probe never
// invokes the vendor CLI when the embedded store already contains the
// session. Shelling out to `opencode session list` is exactly the read-only
// vendor command that was observed rewriting opencode.db during a `rein
// handoff --dry-run`; a compatibility check the database itself can answer
// must never trigger it.
func TestOpenCodeProbeAnswersFromDatabaseWithoutShellingOut(t *testing.T) {
	t.Parallel()

	dataRoot := buildOpenCodeStore(t)
	called := false
	runner := sessionindex.CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		called = true
		return nil, errors.New("the vendor CLI must not be invoked when the store answers directly")
	})
	reader := &OpenCodeReader{DataRoot: dataRoot, Runner: runner}
	record := sessionindex.Record{ID: "ses_target", Agent: sessionindex.AgentOpenCode}

	compat, err := reader.Probe(context.Background(), record)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if compat != CompatibilitySupported {
		t.Fatalf("Probe = %q, want SUPPORTED", compat)
	}
	if called {
		t.Fatal("Probe shelled out to the vendor CLI although the database already had the session")
	}
}
