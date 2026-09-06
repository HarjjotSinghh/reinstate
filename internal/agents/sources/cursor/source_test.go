package cursor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"

	_ "modernc.org/sqlite"
)

func scan(t *testing.T, root string) sessionindex.ScanResult {
	t.Helper()
	source, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func fixture(t *testing.T, osName string) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "cursor", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace string
		wantMessageCount              int
		wantMinSizeBytes              int64 // meta.json alone, before store.db is added
	}{
		{"macos", "01987654-3210-7890-abcd-ef0123456789", "/Users/fixture-user/code/demo", 3, 127},
		{"windows", "01912345-6789-7abc-def0-123456789abc", `C:\Users\fixture-user\code\demo`, 5, 127},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentCursor || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.CursorReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
			}
			// The committed store.db fixture pins this count; it comes from
			// the vendor's own record table, not a placeholder.
			if record.MessageCount != tt.wantMessageCount {
				t.Fatalf("message_count = %d, want %d", record.MessageCount, tt.wantMessageCount)
			}
			// size_bytes must reflect the store the session actually lives
			// in (meta.json plus its sibling store.db), not the tiny
			// meta.json sidecar alone.
			if record.SizeBytes <= tt.wantMinSizeBytes {
				t.Fatalf("size_bytes = %d, want more than the meta.json-only size %d", record.SizeBytes, tt.wantMinSizeBytes)
			}
		})
	}
}

func TestScanIsDeterministic(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	if !reflect.DeepEqual(scan(t, root).Records, scan(t, root).Records) {
		t.Fatal("two scans of one fixture disagreed")
	}
}

func TestEditorTreeIsNotIndexed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMeta(t, filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json"), true)
	writeMeta(t, filepath.Join(root, "projects", "demo", "agent-transcripts", "sess-2", "meta.json"), true)
	result := scan(t, root)
	if len(result.Records) != 1 || result.Records[0].ID != "sess-1" {
		t.Fatalf("records = %+v", result.Records)
	}
}

func TestEmptyConversationIsSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMeta(t, filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "empty", "meta.json"), false)
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %+v", result.Records)
	}
}

func TestCountStoreMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
		want  int
	}{
		{
			name: "messages table",
			setup: func(t *testing.T, path string) {
				execSQL(t, path, `CREATE TABLE messages (id INTEGER PRIMARY KEY)`)
				execSQLTimes(t, path, `INSERT INTO messages DEFAULT VALUES`, 4)
			},
			want: 4,
		},
		{
			name: "singular message table",
			setup: func(t *testing.T, path string) {
				execSQL(t, path, `CREATE TABLE message (id INTEGER PRIMARY KEY)`)
				execSQLTimes(t, path, `INSERT INTO message DEFAULT VALUES`, 2)
			},
			want: 2,
		},
		{
			name: "empty messages table",
			setup: func(t *testing.T, path string) {
				execSQL(t, path, `CREATE TABLE messages (id INTEGER PRIMARY KEY)`)
			},
			want: 0,
		},
		{
			name: "unrecognized schema degrades to zero",
			setup: func(t *testing.T, path string) {
				execSQL(t, path, `CREATE TABLE blobs (id INTEGER PRIMARY KEY)`)
				execSQLTimes(t, path, `INSERT INTO blobs DEFAULT VALUES`, 9)
			},
			want: 0,
		},
		{
			name: "two candidate tables: larger wins",
			setup: func(t *testing.T, path string) {
				execSQL(t, path, `CREATE TABLE messages (id INTEGER PRIMARY KEY)`)
				execSQLTimes(t, path, `INSERT INTO messages DEFAULT VALUES`, 3)
				execSQL(t, path, `CREATE TABLE message (id INTEGER PRIMARY KEY)`)
				execSQLTimes(t, path, `INSERT INTO message DEFAULT VALUES`, 7)
			},
			want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "store.db")
			tt.setup(t, path)
			if got := countStoreMessages(context.Background(), path); got != tt.want {
				t.Fatalf("countStoreMessages() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountStoreMessagesMissingOrCorruptDegradesToZero(t *testing.T) {
	t.Parallel()
	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		if got := countStoreMessages(context.Background(), filepath.Join(t.TempDir(), "absent.db")); got != 0 {
			t.Fatalf("countStoreMessages() = %d, want 0", got)
		}
	})
	t.Run("not a database", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "store.db")
		if err := os.WriteFile(path, []byte("not sqlite"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := countStoreMessages(context.Background(), path); got != 0 {
			t.Fatalf("countStoreMessages() = %d, want 0", got)
		}
	})
}

// TestFingerprintSeesStoreDBChanges guards against the incremental-refresh
// staleness this reader would otherwise have: SessionGlob only matches
// meta.json, so a session whose store.db changes without meta.json changing
// must still produce a different fingerprint.
func TestFingerprintSeesStoreDBChanges(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY)`)
	execSQLTimes(t, storePath, `INSERT INTO messages DEFAULT VALUES`, 1)

	src, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	source, ok := src.(sessionindex.Fingerprinter)
	if !ok {
		t.Fatal("cursor source does not implement sessionindex.Fingerprinter")
	}
	first, ok, err := source.Fingerprint(context.Background())
	if err != nil || !ok {
		t.Fatalf("first fingerprint: ok=%t err=%v", ok, err)
	}

	// Touch only store.db, matching a vendor write that adds a message
	// without rewriting meta.json's own createdAtMs/updatedAtMs.
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(storePath, later, later); err != nil {
		t.Fatal(err)
	}
	execSQLTimes(t, storePath, `INSERT INTO messages DEFAULT VALUES`, 1)
	if err := os.Chtimes(storePath, later, later); err != nil {
		t.Fatal(err)
	}

	second, ok, err := source.Fingerprint(context.Background())
	if err != nil || !ok {
		t.Fatalf("second fingerprint: ok=%t err=%v", ok, err)
	}
	if first == second {
		t.Fatal("fingerprint did not change when store.db changed")
	}
}

func execSQL(t *testing.T, path, stmt string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(stmt); err != nil {
		t.Fatal(err)
	}
}

func execSQLTimes(t *testing.T, path, stmt string, n int) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for i := 0; i < n; i++ {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
}

func writeMeta(t *testing.T, path string, hasConversation bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(
		`{"createdAtMs":1,"cwd":"/tmp/demo","hasConversation":%t,"schemaVersion":1,"updatedAtMs":2}`,
		hasConversation,
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
