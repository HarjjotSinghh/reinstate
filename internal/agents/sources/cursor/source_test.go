package cursor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

// --- Fixture / blob store construction helpers -----------------------------
//
// These are the "small generator" the design doc for this fix calls for: the
// only way any blobs/meta store.db (temp, per-test, or the two committed
// testdata fixtures) is ever built, so the shape stays reproducible in one
// place. See TestGenerateCommittedFixtures below for regenerating the
// committed fixtures.

// blobRow is one row of the real schema's blobs(id TEXT, data BLOB) table.
type blobRow struct {
	id   string
	data []byte
}

// blobID derives a deterministic 64-char hex id (matching the real schema's
// observed id shape) from a short human label, so test fixtures stay
// reproducible without a random source.
func blobID(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

// jsonBlob builds one JSON blobs.data value with "role" always encoded
// before "content" — the key order the real store was observed to use, and
// the order decodeBlobRole's early-stop scan is optimized for. content must
// already be JSON (a quoted string, or a JSON array of parts).
func jsonBlob(label, role string, content []byte, extra string) blobRow {
	roleJSON, err := json.Marshal(role)
	if err != nil {
		panic(err)
	}
	var buf strings.Builder
	buf.WriteString(`{"role":`)
	buf.Write(roleJSON)
	buf.WriteString(`,"content":`)
	buf.Write(content)
	if extra != "" {
		buf.WriteString(",")
		buf.WriteString(extra)
	}
	buf.WriteString("}")
	return blobRow{id: blobID(label), data: []byte(buf.String())}
}

// userStringBlob is a user-role blob whose content is a plain JSON string,
// one of the two content shapes the real store uses.
func userStringBlob(label, text string) blobRow {
	content, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return jsonBlob(label, "user", content, "")
}

// userPartsBlob is a user-role blob whose content is an array of
// {"type":"text","text":…} parts, the other real content shape.
func userPartsBlob(label, text string) blobRow {
	part := struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "text", Text: text}
	content, err := json.Marshal([]any{part})
	if err != nil {
		panic(err)
	}
	return jsonBlob(label, "user", content, "")
}

// assistantPartsBlob is an assistant-role blob with a reasoning part
// followed by a text part, matching the real store's observed assistant
// shape ({type, text, signature, providerOptions} parts, type reasoning and
// text).
func assistantPartsBlob(label, reasoning, text string) blobRow {
	content, err := json.Marshal([]map[string]string{
		{"type": "reasoning", "text": reasoning},
		{"type": "text", "text": text},
	})
	if err != nil {
		panic(err)
	}
	return jsonBlob(label, "assistant", content, `"providerOptions":{}`)
}

// systemBlob is a system-role blob; system rows are excluded from
// message_count and search text.
func systemBlob(label, text string) blobRow {
	content, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return jsonBlob(label, "system", content, "")
}

// binaryBlob is a non-JSON binary blob, matching the real store's observed
// protobuf-like rows (first bytes 0x0a 0x20…). It must never be decoded as
// JSON.
func binaryBlob(label string, n int) blobRow {
	data := make([]byte, n)
	data[0] = 0x0a
	if n > 1 {
		data[1] = 0x20
	}
	for i := 2; i < n; i++ {
		data[i] = byte(i % 256)
	}
	return blobRow{id: blobID(label), data: data}
}

// malformedBlob looks like a JSON object (starts with '{') but is not valid
// JSON — the malformed-blob-skipped case.
func malformedBlob(label string) blobRow {
	return blobRow{id: blobID(label), data: []byte(`{"role":"user, not valid json`)}
}

// writeBlobStore creates path as a SQLite database with the real schema's
// blobs and meta tables and inserts rows into blobs. meta gets the single
// observed row (key "0"); its value is never read by this reader.
func writeBlobStore(t *testing.T, path string, rows []blobRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE blobs (id TEXT, data BLOB)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE meta (key TEXT, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO meta (key, value) VALUES ('0', '1')`); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if _, err := db.Exec(`INSERT INTO blobs (id, data) VALUES (?, ?)`, row.id, row.data); err != nil {
			t.Fatal(err)
		}
	}
}

// --- Fixture generation ------------------------------------------------

// TestGenerateCommittedFixtures (re)writes the committed testdata store.db
// fixtures under testdata/sessionindex/cursor/{macos,windows} in the real
// blobs/meta shape (see the package doc comment and
// docs/session-storage/cursor.md). It intentionally writes outside
// t.TempDir(), so it never runs as part of the normal suite — only when
// REINSTATE_GENERATE_CURSOR_FIXTURES=1 is set. Regenerate with:
//
//	REINSTATE_GENERATE_CURSOR_FIXTURES=1 go test ./internal/agents/sources/cursor/ -run TestGenerateCommittedFixtures -v
func TestGenerateCommittedFixtures(t *testing.T) {
	if os.Getenv("REINSTATE_GENERATE_CURSOR_FIXTURES") != "1" {
		t.Skip("set REINSTATE_GENERATE_CURSOR_FIXTURES=1 to regenerate the committed testdata store.db fixtures")
	}

	macosRows := []blobRow{
		systemBlob("macos-system", "You are a careful pair programmer."),
		userStringBlob("macos-user-string", "Where is the retry loop defined?"),
		binaryBlob("macos-binary-1", 24),
		userPartsBlob("macos-user-parts", "Investigate fixture-search-token-cursor-macos in the retry loop"),
		binaryBlob("macos-binary-2", 40),
		assistantPartsBlob("macos-assistant", "Let me check the retry loop.", "It lives in internal/sync/retry.go."),
		binaryBlob("macos-binary-3", 16),
	}
	writeBlobStore(t, filepath.Join(fixture(t, "macos"), "chats",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "01987654-3210-7890-abcd-ef0123456789", "store.db"), macosRows)

	windowsRows := []blobRow{
		systemBlob("windows-system", "You are a careful pair programmer."),
		userStringBlob("windows-user-string-1", "What does the sync manifest track?"),
		binaryBlob("windows-binary-1", 24),
		userPartsBlob("windows-user-parts", "Investigate fixture-search-token-cursor-windows in the retry loop"),
		assistantPartsBlob("windows-assistant-1", "Checking the manifest.", "It tracks per-file hashes and mtimes."),
		userStringBlob("windows-user-string-2", "And the retry backoff?"),
		binaryBlob("windows-binary-2", 40),
		assistantPartsBlob("windows-assistant-2", "Checking the backoff.", "Exponential, capped at five attempts."),
		binaryBlob("windows-binary-3", 16),
	}
	writeBlobStore(t, filepath.Join(fixture(t, "windows"), "chats",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "01912345-6789-7abc-def0-123456789abc", "store.db"), windowsRows)

	t.Log("committed cursor store.db fixtures regenerated")
}

// --- Scan-level tests --------------------------------------------------

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace, wantToken string
		wantMessageCount                         int
		wantMinSizeBytes                         int64 // meta.json alone, before store.db is added
	}{
		{"macos", "01987654-3210-7890-abcd-ef0123456789", "/Users/fixture-user/code/demo", "fixture-search-token-cursor-macos", 3, 127},
		{"windows", "01912345-6789-7abc-def0-123456789abc", `C:\Users\fixture-user\code\demo`, "fixture-search-token-cursor-windows", 5, 127},
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
			// counting real-schema blobs rows, not a placeholder.
			if record.MessageCount != tt.wantMessageCount {
				t.Fatalf("message_count = %d, want %d", record.MessageCount, tt.wantMessageCount)
			}
			// size_bytes must reflect the store the session actually lives
			// in (meta.json plus its sibling store.db), not the tiny
			// meta.json sidecar alone.
			if record.SizeBytes <= tt.wantMinSizeBytes {
				t.Fatalf("size_bytes = %d, want more than the meta.json-only size %d", record.SizeBytes, tt.wantMinSizeBytes)
			}
			if !strings.Contains(record.SearchText, tt.wantToken) {
				t.Fatalf("search text does not contain the planted user token %q: %q", tt.wantToken, record.SearchText)
			}
			if record.PromptPreview == "" {
				t.Fatalf("prompt preview is empty")
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

// --- countStoreMessages / scanBlobs tests -------------------------------

func TestCountStoreMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows []blobRow
		want int
	}{
		{
			name: "user and assistant blobs count, system does not",
			rows: []blobRow{
				systemBlob("s", "system prompt"),
				userStringBlob("u1", "hello"),
				assistantPartsBlob("a1", "thinking", "reply"),
			},
			want: 2,
		},
		{
			name: "binary blobs never counted",
			rows: []blobRow{
				userStringBlob("u1", "hello"),
				binaryBlob("b1", 32),
				binaryBlob("b2", 8),
			},
			want: 1,
		},
		{
			name: "empty blobs table",
			rows: nil,
			want: 0,
		},
		{
			name: "malformed JSON blob is skipped, not counted",
			rows: []blobRow{
				userStringBlob("u1", "hello"),
				malformedBlob("bad"),
			},
			want: 1,
		},
		{
			name: "several user and assistant blobs",
			rows: []blobRow{
				userStringBlob("u1", "one"),
				userPartsBlob("u2", "two"),
				assistantPartsBlob("a1", "r1", "reply one"),
				assistantPartsBlob("a2", "r2", "reply two"),
				systemBlob("s", "sys"),
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "store.db")
			writeBlobStore(t, path, tt.rows)
			if got := countStoreMessages(context.Background(), path); got != tt.want {
				t.Fatalf("countStoreMessages() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestOldSchemaFixtureYieldsZeroNoError proves a store using the
// v0.6.0-rc.2 reader's guessed messages/message/bubbles shape — which no
// real Cursor CLI store ever had — degrades to message_count 0 and no
// error, the same as any other unrecognized shape, rather than being
// misread as the real schema.
func TestOldSchemaFixtureYieldsZeroNoError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE messages (id INTEGER PRIMARY KEY, role TEXT, text TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO messages (role, text) VALUES ('user', 'hello'), ('assistant', 'hi')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	result := scan(t, root)
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none", result.Warnings)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.MessageCount != 0 {
		t.Fatalf("message_count = %d, want 0 for the old, never-real schema", record.MessageCount)
	}
	if record.SearchText != "" && strings.Contains(record.SearchText, "hello") {
		t.Fatalf("search text leaked the old schema's content: %q", record.SearchText)
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
	writeBlobStore(t, storePath, []blobRow{userStringBlob("u1", "hello")})

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
	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	row := userStringBlob("u2", "another message")
	if _, err := db.Exec(`INSERT INTO blobs (id, data) VALUES (?, ?)`, row.id, row.data); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
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

// --- Search text tests ---------------------------------------------------

// TestSearchIndexesUserBlobsOnly covers Phase 5 Matrix C3 for Cursor: search
// must find text from a user-role blob's content, not only id/title/
// project/workspace, and must never leak assistant- or system-only text.
// PromptPreview falls back to the first user blob's text (meta.json carries
// no vendor title for Cursor CLI sessions). Both real content shapes (a
// plain string, and a parts array) are covered.
func TestSearchIndexesUserBlobsOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	const stringToken = "fixture-search-token-cursor-string"
	const partsToken = "fixture-search-token-cursor-parts"
	const assistantOnlyToken = "assistant-only-reply-marker-cursor"
	const systemOnlyToken = "system-only-prompt-marker-cursor"
	writeBlobStore(t, storePath, []blobRow{
		systemBlob("s", "You are a system prompt with "+systemOnlyToken),
		userStringBlob("u1", "Investigate "+stringToken+" in the retry loop"),
		userPartsBlob("u2", "Also check "+partsToken),
		assistantPartsBlob("a1", "reasoning", assistantOnlyToken),
	})

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, stringToken) {
		t.Fatalf("search text does not contain the string-content user message: %q", record.SearchText)
	}
	if !strings.Contains(record.SearchText, partsToken) {
		t.Fatalf("search text does not contain the parts-content user message: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, assistantOnlyToken) {
		t.Fatalf("search text leaked assistant-only content: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, systemOnlyToken) {
		t.Fatalf("search text leaked system-only content: %q", record.SearchText)
	}
	if !strings.Contains(record.PromptPreview, stringToken) {
		t.Fatalf("prompt preview did not fall back to the first user message: %q", record.PromptPreview)
	}
}

// TestSearchTextUnrecognizedShapeLeavesTextEmpty proves a JSON blob that is
// an object but has neither "role" nor a shape decodeBlobRole can read
// contributes no count and no text, without guessing at an unrecognized
// shape's meaning.
func TestSearchTextUnrecognizedShapeLeavesTextEmpty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	writeBlobStore(t, storePath, []blobRow{
		{id: blobID("no-role"), data: []byte(`{"foo":"bar","baz":1}`)},
	})

	result := scan(t, root)
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if record.MessageCount != 0 {
		t.Fatalf("message_count = %d, want 0", record.MessageCount)
	}
	if strings.Contains(record.SearchText, "bar") {
		t.Fatalf("search text carries an unrecognized field's value: %q", record.SearchText)
	}
}

// TestSearchTextBoundHolds proves a user message far larger than
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
	writeBlobStore(t, storePath, []blobRow{userStringBlob("u1", oversized)})

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
	writeBlobStore(t, storePath, []blobRow{
		userStringBlob("u1", "\x1b[31m"+marker+"\x1b[0m\n more text"),
	})

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
// user message (a pasted log or file dump saved as one blob's content, not
// even a corrupted store) is never pulled into process memory whole: the
// SQL-level substr(...) bound (maxRowTextBytes) must keep the allocation
// growth from reading one such row within a small multiple of
// maxRowTextBytes, not the row's own multi-hundred-megabyte size. This is
// the same empirical methodology used against the old messages-table shape:
// measuring runtime.MemStats.TotalAlloc growth around the read, not just
// asserting on the final (separately bounded) SearchText output.
func TestSearchTextPerRowBoundLimitsMemory(t *testing.T) {
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	writeBlobStore(t, storePath, nil)

	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	const marker = "fixture-search-token-cursor-oversized-row"
	const rowSize = 60 << 20 // 60 MiB: far larger than maxRowTextBytes (4 MiB).
	oversized := marker + " " + strings.Repeat("x", rowSize)
	row := userStringBlob("oversized", oversized)
	if _, err := db.Exec(`INSERT INTO blobs (id, data) VALUES (?, ?)`, row.id, row.data); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	count, searchText, _ := scanBlobs(context.Background(), db)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	grew := after.TotalAlloc - before.TotalAlloc
	t.Logf("TotalAlloc grew by %d bytes", grew)

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if !strings.Contains(searchText, marker) {
		t.Fatalf("search text lost the leading marker: %q", searchText[:min(200, len(searchText))])
	}
	// A generous ceiling: several multiples of maxRowTextBytes to absorb
	// driver/runtime overhead and the Go-side copies (Scan, json.Unmarshal
	// fast-path failure, the raw fallback scan, SafeText), yet well below
	// the 60 MiB row — proving the row was not materialized whole.
	const ceiling = 16 * maxRowTextBytes
	if grew > ceiling {
		t.Fatalf("reading one oversized row grew TotalAlloc by %d bytes, want at most %d (row was %d bytes) — the row was not bounded before materializing", grew, ceiling, rowSize)
	}
}

// TestSearchTextPerRowBoundCountsBytesNotRunes pins the CAST(... AS BLOB)
// in scanBlobs' substr(): SQLite counts characters on TEXT input, so
// without the cast a 60 MiB row of 4-byte runes would come back as 16 MiB.
// The ceiling here is tight enough that a character-counted bound fails it.
func TestSearchTextPerRowBoundCountsBytesNotRunes(t *testing.T) {
	root := t.TempDir()
	metaPath := filepath.Join(root, "chats", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sess-1", "meta.json")
	writeMeta(t, metaPath, true)
	storePath := storeDatabasePath(metaPath)
	writeBlobStore(t, storePath, nil)

	db, err := sql.Open("sqlite", "file:"+storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	const marker = "fixture-search-token-cursor-multibyte-row"
	const rowSize = 60 << 20 // 60 MiB of 4-byte runes: 15 Mi characters.
	oversized := marker + " " + strings.Repeat("😀", rowSize/4)
	row := userStringBlob("multibyte", oversized)
	if _, err := db.Exec(`INSERT INTO blobs (id, data) VALUES (?, ?)`, row.id, row.data); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	_, searchText, firstUser := scanBlobs(context.Background(), db)

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
	const ceiling = 16 * maxRowTextBytes
	if grew > ceiling {
		t.Fatalf("reading one multibyte row grew TotalAlloc by %d bytes, want at most %d — substr counted runes, not bytes", grew, ceiling)
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
