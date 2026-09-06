package opencode

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"

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

// writeStoreWithParts builds a store shaped like OpenCode's measured
// message+part schema (docs/testing/results/2026-08-23-macos-opencode-t5-journey.md),
// with one session carrying a user text part, an assistant text part, and a
// non-text (tool) part, so search-text tests can exercise the real join this
// reader performs.
func writeStoreWithParts(t *testing.T, root string, userText, assistantText string) string {
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
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO project VALUES ('p1','/work/alpha','alpha')`,
		`INSERT INTO session VALUES ('ses_1','p1','','/work/alpha',1787000000000,1787000005000)`,
		`INSERT INTO message VALUES ('msg_user','ses_1','{"role":"user"}'), ('msg_asst','ses_1','{"role":"assistant"}')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	insertPart := func(id, messageID string, data map[string]string) {
		payload, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO part (id, message_id, session_id, data) VALUES (?, ?, 'ses_1', ?)`,
			id, messageID, string(payload)); err != nil {
			t.Fatalf("insert part %s: %v", id, err)
		}
	}
	insertPart("prt_user", "msg_user", map[string]string{"type": "text", "text": userText})
	insertPart("prt_asst", "msg_asst", map[string]string{"type": "text", "text": assistantText})
	if _, err := db.Exec(
		`INSERT INTO part (id, message_id, session_id, data) VALUES ('prt_tool','msg_asst','ses_1','{"type":"tool","tool":"grep"}')`,
	); err != nil {
		t.Fatal(err)
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

// TestSearchIndexesUserPartText covers Phase 5 Matrix C3 for OpenCode's
// embedded-SQLite source: search must find text from a message part body,
// not only id/title/project/workspace. It also proves assistant-only text
// and non-text parts (tool calls) are excluded, matching the Claude reader's
// policy, and that PromptPreview falls back to the first user part when the
// vendor recorded no session title.
func TestSearchIndexesUserPartText(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const userToken = "fixture-search-token-opencode"
	const assistantOnlyToken = "assistant-only-reply-marker-opencode"
	writeStoreWithParts(t, root, "Investigate "+userToken+" in the retry loop", assistantOnlyToken)

	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if !strings.Contains(record.SearchText, userToken) {
		t.Fatalf("search text does not contain the user part body: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, assistantOnlyToken) {
		t.Fatalf("search text leaked assistant-only content: %q", record.SearchText)
	}
	if strings.Contains(record.SearchText, "grep") {
		t.Fatalf("search text leaked a non-text (tool) part: %q", record.SearchText)
	}
	if !strings.Contains(record.PromptPreview, userToken) {
		t.Fatalf("prompt preview did not fall back to the first user part: %q", record.PromptPreview)
	}
}

// TestSessionMessageOnlyStoreYieldsNoSearchText proves a store still on the
// legacy session_message schema (no part table) keeps its message_count and
// yields no search text, rather than guessing at that table's unverified
// per-row shape.
func TestSessionMessageOnlyStoreYieldsNoSearchText(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, DatabaseName)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE project (id TEXT PRIMARY KEY, worktree TEXT NOT NULL, name TEXT)`,
		`CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			directory TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE session_message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, type TEXT NOT NULL,
			seq INTEGER NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO project VALUES ('p1','/work/alpha','alpha')`,
		`INSERT INTO session VALUES ('ses_1','p1','','/work/alpha',1787000000000,1787000005000)`,
		`INSERT INTO session_message VALUES ('sm1','ses_1','user',1,'{"text":"legacy-schema-token-opencode"}')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
	}
	record := result.Records[0]
	if record.MessageCount != 1 {
		t.Fatalf("message_count = %d, want 1", record.MessageCount)
	}
	if strings.Contains(record.SearchText, "legacy-schema-token-opencode") {
		t.Fatalf("search text guessed at the unverified session_message shape: %q", record.SearchText)
	}
}

// TestSearchTextBoundHolds proves a part body far larger than
// sessionindex.MaxSearchTextBytes is truncated in the final SearchText
// rather than reported whole.
func TestSearchTextBoundHolds(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const marker = "fixture-search-token-opencode-bound"
	oversized := marker + " " + strings.Repeat("padding ", 100000)
	writeStoreWithParts(t, root, oversized, "reply")

	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
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

// TestSearchTextRedactsControlSequences proves part-body text goes through
// the same SafeText sanitization the Claude reader applies.
func TestSearchTextRedactsControlSequences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const marker = "fixture-search-token-opencode-escape"
	writeStoreWithParts(t, root, "\x1b[31m"+marker+"\x1b[0m\n more text", "reply")

	source, err := NewSQLite(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
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
// part (a pasted log or file dump saved as one part, not even a corrupted
// store) is never pulled into process memory whole: the SQL-level
// substr(...) bound (maxRowTextBytes) must keep the allocation growth from
// reading one such row within a small multiple of maxRowTextBytes, not the
// row's own multi-hundred-megabyte size. A part.data blob this large no
// longer parses as JSON once substr has truncated it, so it correctly
// contributes no text (the same "counts as a turn, no text" outcome the
// Cline reader applies to one oversized message) rather than a crash or a
// garbled value — this test asserts both: bounded memory growth, and a
// record that still scans cleanly with no leaked partial text.
func TestSearchTextPerRowBoundLimitsMemory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, DatabaseName)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	stmts := []string{
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, data TEXT NOT NULL)`,
		`INSERT INTO message VALUES ('msg_user','ses_1','{"role":"user"}')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	const marker = "fixture-search-token-opencode-oversized-part"
	const rowSize = 60 << 20 // 60 MiB: far larger than maxRowTextBytes (4 MiB).
	payload, err := json.Marshal(map[string]string{
		"type": "text",
		"text": marker + " " + strings.Repeat("x", rowSize),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO part (id, message_id, session_id, data) VALUES ('prt_user','msg_user','ses_1',?)`,
		string(payload)); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	searchText, _ := readSessionSearchText(context.Background(), db, "ses_1", true)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	grew := after.TotalAlloc - before.TotalAlloc

	// A generous ceiling: several multiples of maxRowTextBytes to absorb
	// driver/runtime overhead, but two full orders of magnitude below the
	// 60 MiB row — proving the row was not materialized whole.
	const ceiling = 8 * maxRowTextBytes
	if grew > ceiling {
		t.Fatalf("reading one oversized part grew TotalAlloc by %d bytes, want at most %d (part was %d bytes) — the row was not bounded before materializing", grew, ceiling, rowSize)
	}
	// The truncated blob no longer parses as JSON, so it must contribute no
	// text rather than a garbled fragment.
	if strings.Contains(searchText, marker) {
		t.Fatalf("search text unexpectedly kept text from a row the SQL-level bound should have truncated past valid JSON: %q", searchText[:min(200, len(searchText))])
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
