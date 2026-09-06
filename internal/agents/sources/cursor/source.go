// Package cursor discovers Cursor CLI sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 dual-platform probes:
//
//	~/.cursor/chats/<32-hex>/<uuid-v4>/meta.json
//	~/.cursor/chats/<32-hex>/<uuid-v4>/store.db
//
// First-line keys on both platforms: createdAtMs, cwd, hasConversation,
// schemaVersion, updatedAtMs. The editor tree under projects/ is excluded.
//
// meta.json is a small index sidecar; the session's own content lives in the
// sibling store.db (SQLite). size_bytes is the two files combined, and
// message_count is read from store.db read-only through vendorsqlite, table
// name unverified (see doc/session-storage/cursor.md): a store whose schema
// does not match the recognized candidate table names degrades to the
// message_count 0 this reader always reported before it existed, rather than
// guessing.
//
// Search text is read the same way, from the same recognized table, and only
// when that table also has a recognized author/role column and a recognized
// body/text column (see messageRoleColumnCandidates and
// messageTextColumnCandidates below) — a table with a row count but no
// recognized author column contributes no text, since there is no way to
// exclude assistant turns from a store this reader cannot identify authorship
// in. Where text is available, only user-authored rows are indexed, matching
// the policy internal/sessionindex/claude.go applies to Claude Code
// transcripts.
package cursor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/vendorsqlite"
)

// StoreDatabaseName is the per-session SQLite store beside meta.json.
const StoreDatabaseName = "store.db"

// messageTableCandidates are the table names this reader recognizes as
// holding one row per message. The real schema is undocumented and
// unverified (docs/session-storage/cursor.md); a store using none of these
// names yields message_count 0, the same as before this reader existed. If
// more than one candidate table is present, the larger count wins, on the
// same reasoning OpenCode's reader uses for its own migrated table pair: one
// name is the live one and the rest are remnants, and summing would double
// count a store mid-migration.
var messageTableCandidates = []string{"messages", "message", "bubbles"}

// messageTextColumnCandidates are the column names this reader recognizes as
// holding one message row's body text. Like the table names above, the real
// column names are unverified; a recognized table using none of these
// column names still yields its row count, just no search text — the same
// as this reader's behavior before content was ever read.
var messageTextColumnCandidates = []string{"text", "content", "body", "message"}

// messageRoleColumnCandidates are the column names this reader recognizes as
// naming a message row's author, so only user-authored text is indexed. A
// recognized table with a recognized text column but no recognized role
// column still contributes no text: there is no way to exclude assistant
// turns from it, and guessing every row is a user turn risks indexing the
// agent's own replies as if the user had typed them.
var messageRoleColumnCandidates = []string{"role", "author", "sender", "type"}

// userRoleValues are the values this reader recognizes in a role column as
// naming the user, tried in order.
var userRoleValues = []string{"user", "human"}

// maxTextRows bounds how many rows of one session's message table are read
// for search text. Reading stops earlier, as soon as the shared
// sessionindex.MaxSearchTextBytes budget is spent; this is a backstop against
// a pathological store with a huge number of tiny rows.
const maxTextRows = 20000

// SessionGlob matches one CLI session metadata file.
const SessionGlob = "chats/**/meta.json"

var requiredKeys = []string{"createdAtMs", "cwd", "hasConversation", "schemaVersion", "updatedAtMs"}

// Excluded keeps the editor agent, extensions, and skills out of the CLI walk.
var Excluded = []string{
	"projects",
	"extensions",
	"plugins",
	"skills",
	"skills-cursor",
	"plans",
	"agents",
	"rules",
	"ai-tracking",
	"sandbox-policies",
	"worktrees",
	"cli-config.json",
	"mcp.json",
	"**/mcp.json",
	"ide_state.json",
	"argv.json",
	"hooks.json",
	"hooks.json.bak",
	"agent-cli-state.json",
	"statsig-cache.json",
}

// Source discovers Cursor CLI sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Cursor index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentCursor }

// Scan maps every readable CLI meta.json to one record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		record, parseErr := parseSession(ctx, file)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentCursor,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Cursor CLI session could not be read; other sessions remain available",
			})
			continue
		}
		if record.ID == "" {
			continue
		}
		result.Records = append(result.Records, record)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

// Fingerprint summarises the source without opening any vendor file for its
// content, so an unchanged refresh can skip parsing entirely.
//
// hometree.Fingerprint alone is not enough: its walk only matches
// meta.json (SessionGlob), so a message recorded in a session's sibling
// store.db would leave the meta.json-only hash unchanged and an incremental
// refresh would keep serving a stale message_count and size_bytes forever.
// Each meta.json's sibling store.db is stat-ed (never opened) and folded in.
func (s *Source) Fingerprint(ctx context.Context) (string, bool, error) {
	base, ok, err := hometree.Fingerprint(ctx, s.config())
	if err != nil || !ok {
		return base, ok, err
	}
	_, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return "", false, err
	}
	sum := sha256.New()
	_, _ = sum.Write([]byte(base))
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		info, statErr := os.Stat(storeDatabasePath(file.Path))
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(info.Size(), 10)))
	}
	return hex.EncodeToString(sum.Sum(nil)), true, nil
}

func (s *Source) config() hometree.Config {
	cfg := hometree.Config{
		Explicit:    s.env.FixtureRoot,
		RootEnv:     "CURSOR_CONFIG_DIR",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "chats",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".cursor")}
	}
	return cfg
}

type meta struct {
	CreatedAtMs     json.Number `json:"createdAtMs"`
	UpdatedAtMs     json.Number `json:"updatedAtMs"`
	CWD             string      `json:"cwd"`
	HasConversation bool        `json:"hasConversation"`
	SchemaVersion   *int        `json:"schemaVersion"`
}

func parseSession(ctx context.Context, file hometree.File) (sessionindex.Record, error) {
	item, err := readMeta(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	if item.SchemaVersion == nil {
		return sessionindex.Record{}, fmt.Errorf("unknown cursor layout: missing schemaVersion")
	}
	if !item.HasConversation {
		return sessionindex.Record{}, nil
	}
	id := filepath.Base(filepath.Dir(file.Path))
	workspace := strings.TrimSpace(item.CWD)
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}
	updated := unixMs(item.UpdatedAtMs)
	if updated.IsZero() {
		updated = unixMs(item.CreatedAtMs)
	}
	if updated.IsZero() {
		updated = file.ModTime.UTC()
	}
	title := project
	if title == "unknown" {
		title = id
	}

	// meta.json is a small index sidecar; the session's actual content lives
	// in the sibling store.db. size_bytes, message_count, and search text all
	// come from the store the session lives in, not from the sidecar alone.
	sizeBytes := file.Size
	sourceModTime := file.ModTime.UnixNano()
	messageCount := 0
	var messageText, firstUserText string
	storePath := storeDatabasePath(file.Path)
	if storeInfo, statErr := os.Stat(storePath); statErr == nil && storeInfo.Mode().IsRegular() {
		sizeBytes += storeInfo.Size()
		if storeModTime := storeInfo.ModTime().UnixNano(); storeModTime > sourceModTime {
			sourceModTime = storeModTime
		}
		messageCount, messageText, firstUserText = readStoreMessages(ctx, storePath)
	}

	safeTitle := sessionindex.SafePreview(title)
	// title above is always derived from the project name or the bare
	// session id (Cursor's meta.json carries no vendor title), so the first
	// user message, when one exists, is always a better preview.
	preview := safeTitle
	if fallback := sessionindex.SafePreview(firstUserText); fallback != "" {
		preview = fallback
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentCursor, id),
		ID:             id,
		Agent:          sessionindex.AgentCursor,
		Title:          safeTitle,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      sizeBytes,
		MessageCount:   messageCount,
		PromptPreview:  preview,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.CursorReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  sourceModTime,
		SourceSize:     sizeBytes,
		SearchText:     sessionindex.BuildSearchText(id, title, project, workspace, messageText),
	}, nil
}

// storeDatabasePath returns the store.db path beside a session's meta.json.
func storeDatabasePath(metaPath string) string {
	return filepath.Join(filepath.Dir(metaPath), StoreDatabaseName)
}

// countStoreMessages is a thin wrapper over readStoreMessages for callers
// (and existing tests) that only need the count.
func countStoreMessages(ctx context.Context, path string) int {
	count, _, _ := readStoreMessages(ctx, path)
	return count
}

// readStoreMessages opens a session's store.db read-only through
// vendorsqlite (never writing under the vendor's root) exactly once, and
// returns the row count of whichever recognized message table the store
// actually has, that same table's bounded and sanitized user-authored
// search text, and the first user row's raw text for use as a prompt
// preview fallback. Any failure to open, query, or recognize the schema
// yields a zero count and no text, not an error: the session still gets a
// record from meta.json, just without them.
func readStoreMessages(ctx context.Context, path string) (count int, searchText string, firstUserText string) {
	if err := ctx.Err(); err != nil {
		return 0, "", ""
	}
	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return 0, "", ""
	}
	defer func() { _ = handle.Close() }()
	table, count := bestMessageTable(ctx, handle.DB)
	searchText, firstUserText = readMessageText(ctx, handle.DB, table)
	return count, searchText, firstUserText
}

// bestMessageTable reports which recognized message table has the largest
// row count, and that count. If more than one candidate table is present,
// the larger count wins, on the same reasoning OpenCode's reader uses for
// its own migrated table pair: one name is the live one and the rest are
// remnants, and summing would double count a store mid-migration.
func bestMessageTable(ctx context.Context, db *sql.DB) (table string, count int) {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('messages','message','bubbles')`)
	if err != nil {
		return "", 0
	}
	present := map[string]bool{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			present[name] = true
		}
	}
	closeErr := rows.Close()
	if err := rows.Err(); err != nil {
		return "", 0
	}
	if closeErr != nil {
		return "", 0
	}

	for _, candidate := range messageTableCandidates {
		if !present[candidate] {
			continue
		}
		if err := ctx.Err(); err != nil {
			return table, count
		}
		var query string
		switch candidate {
		case "messages":
			query = `SELECT COUNT(*) FROM messages`
		case "message":
			query = `SELECT COUNT(*) FROM message`
		case "bubbles":
			query = `SELECT COUNT(*) FROM bubbles`
		}
		var candidateCount int
		if scanErr := db.QueryRowContext(ctx, query).Scan(&candidateCount); scanErr != nil {
			continue
		}
		if candidateCount > count {
			count = candidateCount
			table = candidate
		}
	}
	return table, count
}

// readMessageText reads the bounded, sanitized text of every user-authored
// row in table, and returns the first such row's raw text separately for use
// as a prompt preview fallback. table must be empty or one of
// messageTableCandidates; any other value is refused rather than built into
// a query. A table with no recognized text column, or no recognized role
// column, contributes no text — see messageTextColumnCandidates and
// messageRoleColumnCandidates.
func readMessageText(ctx context.Context, db *sql.DB, table string) (searchText string, firstUserText string) {
	if table == "" || !isRecognizedTable(table) {
		return "", ""
	}
	columns, err := tableColumns(ctx, db, table)
	if err != nil {
		return "", ""
	}
	textColumn := pickColumn(columns, messageTextColumnCandidates)
	roleColumn := pickColumn(columns, messageRoleColumnCandidates)
	if textColumn == "" || roleColumn == "" {
		return "", ""
	}

	placeholders := make([]string, len(userRoleValues))
	args := make([]any, 0, len(userRoleValues)+1)
	for index, value := range userRoleValues {
		placeholders[index] = "?"
		args = append(args, value)
	}
	// table, textColumn, and roleColumn are only ever one of a small number
	// of fixed, hardcoded identifiers verified above (isRecognizedTable,
	// messageTextColumnCandidates, messageRoleColumnCandidates), never a
	// value read from the vendor's own data, so building the query by string
	// concatenation carries no injection risk here.
	query := `SELECT ` + textColumn + ` FROM ` + table +
		` WHERE ` + roleColumn + ` IN (` + strings.Join(placeholders, ",") + `)` +
		` ORDER BY rowid LIMIT ?`
	args = append(args, maxTextRows)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", ""
	}
	defer func() { _ = rows.Close() }()
	var text sources.BoundedText
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			break
		}
		var value sql.NullString
		if scanErr := rows.Scan(&value); scanErr != nil {
			continue
		}
		if !value.Valid || value.String == "" {
			continue
		}
		if firstUserText == "" {
			firstUserText = value.String
		}
		text.Add(value.String)
	}
	return text.String(), firstUserText
}

// isRecognizedTable reports whether table is one of messageTableCandidates,
// the only names ever interpolated into a query in this file.
func isRecognizedTable(table string) bool {
	for _, candidate := range messageTableCandidates {
		if table == candidate {
			return true
		}
	}
	return false
}

// tableColumns reads a table's column names via PRAGMA table_info. table
// must already be verified by isRecognizedTable.
func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid          int
			name, ctype  string
			notNull, pk  int
			defaultValue sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		columns[strings.ToLower(name)] = true
	}
	return columns, rows.Err()
}

// pickColumn returns the first candidate present in columns, or "".
func pickColumn(columns map[string]bool, candidates []string) string {
	for _, candidate := range candidates {
		if columns[candidate] {
			return candidate
		}
	}
	return ""
}

func readMeta(path string) (meta, error) {
	file, err := os.Open(path)
	if err != nil {
		return meta{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return meta{}, err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return meta{}, fmt.Errorf("cursor meta.json exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return meta{}, err
	}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			return meta{}, fmt.Errorf("unknown cursor layout: missing %s", key)
		}
	}
	var item meta
	if err := json.Unmarshal(data, &item); err != nil {
		return meta{}, err
	}
	return item, nil
}

func unixMs(value json.Number) time.Time {
	n, err := value.Int64()
	if err != nil || n <= 0 {
		return time.Time{}
	}
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}
