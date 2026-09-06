package cursor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

// TestCursorConfigDirOverridesDiscovery is the regression for F-CURSOR-ROOTENV:
// config() used to omit hometree.Config.RootEnv, so CURSOR_CONFIG_DIR isolated
// only the doctor --agents probe (which reads the catalog descriptor directly)
// and never session discovery, search, inspect, resume, or fork, which all go
// through Scan via this Source's config(). A seeded override must be honoured
// and an empty override must yield nothing, both without ever touching a real
// home directory.
func TestCursorConfigDirOverridesDiscovery(t *testing.T) {
	t.Parallel()
	seeded := fixture(t, "macos")
	nonexistentHome := filepath.Join(t.TempDir(), "no-such-home")

	lookupSeeded := func(key string) string {
		if key == "CURSOR_CONFIG_DIR" {
			return seeded
		}
		return ""
	}
	source, err := New(agents.Env{Home: nonexistentHome, LookupEnv: lookupSeeded})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 || result.Records[0].ID != "01987654-3210-7890-abcd-ef0123456789" {
		t.Fatalf("seeded CURSOR_CONFIG_DIR: records = %+v", result.Records)
	}

	empty := t.TempDir()
	lookupEmpty := func(key string) string {
		if key == "CURSOR_CONFIG_DIR" {
			return empty
		}
		return ""
	}
	source, err = New(agents.Env{Home: nonexistentHome, LookupEnv: lookupEmpty})
	if err != nil {
		t.Fatal(err)
	}
	result, err = source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 0 {
		t.Fatalf("isolated CURSOR_CONFIG_DIR: records = %+v", result.Records)
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

// TestSearchIndexesMessageBody covers Phase 5 Matrix C3 for Cursor: search
// must find text from the message body, not only id/title/project/workspace.
// It also proves assistant-only text is excluded, matching the Claude
// reader's policy, and that PromptPreview falls back to the first user
// message (meta.json carries no vendor title for Cursor CLI sessions).
func TestSearchIndexesMessageBody(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	const userToken = "fixture-search-token-cursor"
	const assistantOnlyToken = "assistant-only-reply-marker-cursor"
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT, ts INTEGER)`)
	execSQL(t, storePath, fmt.Sprintf(
		`INSERT INTO messages (role, text, ts) VALUES ('user', 'Investigate %s in the retry loop', 1), ('assistant', '%s', 2)`,
		userToken, assistantOnlyToken,
	))

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, userToken) {
		t.Fatalf("search text does not contain the user message body: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, assistantOnlyToken) {
		t.Fatalf("search text leaked assistant-only content: %q", record.SearchText)
	}
	if !strings.Contains(record.PromptPreview, userToken) {
		t.Fatalf("prompt preview did not fall back to the first user message: %q", record.PromptPreview)
	}
}

// TestSearchTextUnrecognizedColumnsLeavesTextEmpty proves a recognized
// message table with no recognized text or role column still yields its row
// count, exactly as before this reader read content, without guessing at an
// unrecognized column's meaning.
func TestSearchTextUnrecognizedColumnsLeavesTextEmpty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, payload BLOB)`)
	execSQL(t, storePath, `INSERT INTO messages (payload) VALUES (x'00')`)

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if record.MessageCount != 1 {
		t.Fatalf("message_count = %d, want 1", record.MessageCount)
	}
	if strings.Contains(record.SearchText, "\x00") {
		t.Fatalf("search text carries raw column bytes: %q", record.SearchText)
	}
}

// TestSearchTextBoundHolds proves a message far larger than
// sessionindex.MaxSearchTextBytes is truncated in the final SearchText
// rather than reported whole.
func TestSearchTextBoundHolds(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	const marker = "fixture-search-token-cursor-bound"
	oversized := marker + " " + strings.Repeat("padding ", 100000)
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT, ts INTEGER)`)
	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO messages (role, text, ts) VALUES ('user', ?, 1)`, oversized); err != nil {
		t.Fatal(err)
	}

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, marker) {
		t.Fatalf("search text lost the leading marker: %q", record.SearchText[:min(200, len(record.SearchText))])
	}
	if len(record.SearchText) > sessionindex.MaxSearchTextBytes {
		t.Fatalf("search text = %d bytes, want at most %d", len(record.SearchText), sessionindex.MaxSearchTextBytes)
	}
}

// TestSearchTextRedactsControlSequences proves message-body text goes
// through the same SafeText sanitization the Claude reader applies.
func TestSearchTextRedactsControlSequences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	const marker = "fixture-search-token-cursor-escape"
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT, ts INTEGER)`)
	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO messages (role, text, ts) VALUES ('user', ?, 1)`,
		"\x1b[31m"+marker+"\x1b[0m\n more text"); err != nil {
		t.Fatal(err)
	}

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if strings.Contains(record.SearchText, "\x1b") {
		t.Fatalf("search text carries a raw terminal escape: %q", record.SearchText)
	}
	if !strings.Contains(record.SearchText, marker) {
		t.Fatalf("search text lost the sanitized marker: %q", record.SearchText)
	}
}

// TestSearchTextPerRowBoundLimitsMemory proves a single pathologically large
// message row (a pasted log or file dump saved as one message, not even a
// corrupted store) is never pulled into process memory whole: the SQL-level
// substr(...) bound (maxRowTextBytes) must keep the allocation growth from
// reading one such row within a small multiple of maxRowTextBytes, not the
// row's own multi-hundred-megabyte size. This is the same empirical
// methodology used to demonstrate the gap this test now closes: measuring
// runtime.MemStats.TotalAlloc growth around the read, not just asserting on
// the final (separately bounded) SearchText output.
func TestSearchTextPerRowBoundLimitsMemory(t *testing.T) {
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT, ts INTEGER)`)

	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	const marker = "fixture-search-token-cursor-oversized-row"
	const rowSize = 60 << 20 // 60 MiB: far larger than maxRowTextBytes (4 MiB).
	oversized := marker + " " + strings.Repeat("x", rowSize)
	if _, err := db.Exec(`INSERT INTO messages (role, text, ts) VALUES ('user', ?, 1)`, oversized); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	searchText, _ := readMessageText(context.Background(), db, "messages")

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	grew := after.TotalAlloc - before.TotalAlloc
	t.Logf("TotalAlloc grew by %d bytes", grew)

	if !strings.Contains(searchText, marker) {
		t.Fatalf("search text lost the leading marker: %q", searchText[:min(200, len(searchText))])
	}
	// A generous ceiling: several multiples of maxRowTextBytes to absorb
	// driver/runtime overhead and the Go-side copies (Scan, ToValidUTF8,
	// SafeText: measured ~35 MB for a 4 MiB value), yet well below the
	// 60 MiB row — proving the row was not materialized whole.
	const ceiling = 12 * maxRowTextBytes
	if grew > ceiling {
		t.Fatalf("reading one oversized row grew TotalAlloc by %d bytes, want at most %d (row was %d bytes) — the row was not bounded before materializing", grew, ceiling, rowSize)
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

// TestSearchTextPerRowBoundCountsBytesNotRunes pins the CAST(... AS BLOB)
// in readMessageText's substr(): SQLite counts characters on TEXT input, so
// without the cast a 60 MiB row of 4-byte runes would come back as 16 MiB.
// The ceiling here is tight enough that a character-counted bound fails it.
func TestSearchTextPerRowBoundCountsBytesNotRunes(t *testing.T) {
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	execSQL(t, storePath, `CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT, ts INTEGER)`)

	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	const marker = "fixture-search-token-cursor-multibyte-row"
	const rowSize = 60 << 20 // 60 MiB of 4-byte runes: 15 Mi characters.
	oversized := marker + " " + strings.Repeat("😀", rowSize/4)
	if _, err := db.Exec(`INSERT INTO messages (role, text, ts) VALUES ('user', ?, 1)`, oversized); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	searchText, firstUser := readMessageText(context.Background(), db, "messages")

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	grew := after.TotalAlloc - before.TotalAlloc
	t.Logf("TotalAlloc grew by %d bytes", grew)

	if !strings.Contains(searchText, marker) {
		t.Fatalf("search text lost the leading marker: %q", searchText[:min(200, len(searchText))])
	}
	if !utf8.ValidString(firstUser) {
		t.Fatal("first user text is not valid UTF-8 after the byte-bounded read")
	}
	if len(firstUser) > maxRowTextBytes {
		t.Fatalf("first user text is %d bytes, want at most %d", len(firstUser), maxRowTextBytes)
	}
	// Measured: ~39 MB with the byte-counted bound, ~137 MB when substr
	// counts runes instead; the ceiling sits between the two.
	const ceiling = 12 * maxRowTextBytes
	if grew > ceiling {
		t.Fatalf("reading one multibyte row grew TotalAlloc by %d bytes, want at most %d — substr counted runes, not bytes", grew, ceiling)
	}
}
