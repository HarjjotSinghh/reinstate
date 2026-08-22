package vendorsqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// stage builds a database whose newest row exists only in the write-ahead log,
// by copying the pair out while the writing connection is still open. Closing
// the last connection checkpoints the log and deletes it.
func stage(t *testing.T, withLog bool) string {
	t.Helper()
	staging := t.TempDir()
	target := t.TempDir()
	path := filepath.Join(staging, "store.db")

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+
		"?_pragma=journal_mode(WAL)&_pragma=wal_autocheckpoint(0)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE t (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t VALUES ('checkpointed')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(FULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t VALUES ('in-the-log')`); err != nil {
		t.Fatal(err)
	}

	suffixes := []string{""}
	if withLog {
		suffixes = append(suffixes, walSuffix)
	}
	for _, suffix := range suffixes {
		body, readErr := os.ReadFile(path + suffix)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if err := os.WriteFile(filepath.Join(target, "store.db"+suffix), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(target, "store.db")
}

func rowsIn(t *testing.T, h *Handle) []string {
	t.Helper()
	rows, err := h.DB.Query(`SELECT id FROM t ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func listing(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// TestLogContentsAreVisible is the whole point: an immutable in-place handle
// ignores the write-ahead log by definition, so the newest rows read as absent.
func TestLogContentsAreVisible(t *testing.T) {
	t.Parallel()
	path := stage(t, true)
	handle, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = handle.Close() }()

	if !handle.SawWAL {
		t.Fatal("a store with a write-ahead log did not report one")
	}
	got := rowsIn(t, handle)
	if len(got) != 2 {
		t.Fatalf("rows = %v, want both the checkpointed row and the one still in the log", got)
	}
}

// TestNothingIsWrittenBesideTheStore is the constraint the fix must not break.
func TestNothingIsWrittenBesideTheStore(t *testing.T) {
	t.Parallel()
	for _, withLog := range []bool{false, true} {
		name := "without a log"
		if withLog {
			name = "with a log"
		}
		t.Run(name, func(t *testing.T) {
			path := stage(t, withLog)
			dir := filepath.Dir(path)
			before := listing(t, dir)

			handle, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			rowsIn(t, handle)
			during := listing(t, dir)
			if err := handle.Close(); err != nil {
				t.Fatal(err)
			}
			after := listing(t, dir)

			for label, got := range map[string][]string{"during": during, "after": after} {
				if len(got) != len(before) {
					t.Fatalf("%s the read the directory became %v, was %v", label, got, before)
				}
				for i := range before {
					if before[i] != got[i] {
						t.Fatalf("%s the read the directory became %v, was %v", label, got, before)
					}
				}
			}
		})
	}
}

// TestPrivateCopyIsRemoved keeps the temporary copy from outliving the handle.
func TestPrivateCopyIsRemoved(t *testing.T) {
	t.Parallel()
	path := stage(t, true)
	handle, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	temp := handle.tempDir
	if temp == "" {
		t.Fatal("a store with a log was not read through a private copy")
	}
	if _, err := os.Stat(temp); err != nil {
		t.Fatalf("private copy is not on disk while the handle is open: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(temp); !os.IsNotExist(err) {
		t.Fatalf("private copy outlived the handle: %v", err)
	}
}

// TestNoLogTakesTheInPlacePath keeps the cheap case cheap: a checkpointed store
// is read where it lies, with no copy at all.
func TestNoLogTakesTheInPlacePath(t *testing.T) {
	t.Parallel()
	path := stage(t, false)
	handle, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = handle.Close() }()
	if handle.tempDir != "" {
		t.Fatal("a store with no write-ahead log was copied anyway")
	}
	if handle.SawWAL || handle.Incomplete {
		t.Fatalf("SawWAL=%t Incomplete=%t for a store with no log", handle.SawWAL, handle.Incomplete)
	}
}

func TestMissingStoreIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := Open(filepath.Join(t.TempDir(), "absent.db")); err == nil {
		t.Fatal("opening a nonexistent store succeeded")
	}
}
