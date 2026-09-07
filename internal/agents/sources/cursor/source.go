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
// sibling store.db (SQLite). Its real schema — established 2026-09-07 by a
// schema-only, read-only inspection of two real Cursor CLI 2026.08.11
// store.db files (table/column names via sqlite_master and
// pragma table_info, JSON key names and blob magic bytes only; see
// docs/session-storage/cursor.md) — is two tables:
//
//	blobs(id TEXT, data BLOB)   -- id is 64-char hex; one row per turn
//	meta(key, value)            -- observed one row, key "0"; never read
//
// There is no messages/message/bubbles table; the v0.6.0-rc.2 reader that
// looked for one never matched a real store, so real sessions always
// reported message_count 0. blobs.data is either a JSON object beginning
// with the byte '{' — keys seen: role (system/user/assistant), content (a
// string, or an array of {type, text, …} parts with type text or
// reasoning), providerOptions — or a non-JSON binary blob (protobuf-like,
// first bytes observed as 0x0a 0x20…). message_count counts blobs rows
// whose data is a JSON object with role user or assistant; system rows are
// excluded from the count, matching the user/assistant-only search policy
// below. blobs carries no ordering column, so rows are read by rowid, which
// is best-effort only.
//
// Search text is the sanitized, bounded content of every user-role blob
// (never assistant or system), matching the policy
// internal/sessionindex/claude.go applies to Claude Code transcripts.
// PromptPreview falls back to the first such user blob's raw content, since
// meta.json carries no vendor session title. A store using neither this
// shape nor any recognized shape (including the old rc.2 messages/message/
// bubbles guess, which was never real) degrades to message_count 0 and no
// text — the same value every Cursor session got before either reader
// existed — rather than guessing.
package cursor

import (
	"bytes"
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

// blobsTable and metaTable are the only two tables the real Cursor CLI
// store.db schema has (see the package doc comment). metaTable is never
// queried; it is named here only so its absence never gets confused for an
// unrecognized store shape by a future reader.
const (
	blobsTable = "blobs"
	metaTable  = "meta"
)

// maxBlobRows bounds how many rows of one session's blobs table are ever
// scanned, for message_count as well as search text. Real sessions this
// reader was built against held 11 blobs; this is a backstop against a
// pathological store with a huge number of rows, the same role maxTextRows
// played in the v0.6.0-rc.2 reader. A store past this bound undercounts
// rather than scanning unbounded.
const maxBlobRows = 20000

// maxRowTextBytes bounds how much of any single blobs row's data column
// this reader ever pulls out of SQLite, via substr() over CAST(data AS
// BLOB) in the SELECT list itself rather than a Go-side check after the
// value is already in hand. The CAST matters: SQLite's substr() counts
// characters on TEXT input and bytes on BLOB input, so without it a row of
// 4-byte runes (emoji, CJK, box-drawing terminal output) would come back
// four times the intended size. This is not a query-planner hint: it
// changes what the scanned []byte actually holds by the time rows.Scan
// runs, so a session with one pathologically large message (a pasted log or
// file dump saved as a single row's content — not even a corrupted store)
// never has that row's full bytes pulled into process memory, matching the
// same bound sessionindex.MaxJSONLineBytes applies to one Claude Code JSONL
// event. The bound is large enough to always reach role (a few bytes into
// every observed blob) and the start of content, but never the whole blob
// when it is huge: confirmed empirically against modernc.org/sqlite (the
// driver vendorsqlite opens), scanning a substr(col, 1, N)-bounded column
// off a 60 MiB row grows runtime.MemStats.TotalAlloc by only ~N bytes, not
// the row's full size, where scanning the unbounded column grows it by the
// full ~60 MiB.
const maxRowTextBytes = sessionindex.MaxJSONLineBytes

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
// returns the number of user/assistant blobs in it, the bounded and
// sanitized text of the user-role ones, and the first such row's raw text
// for use as a prompt preview fallback. Any failure to open, query, or
// recognize the schema yields a zero count and no text, not an error: the
// session still gets a record from meta.json, just without them.
func readStoreMessages(ctx context.Context, path string) (count int, searchText string, firstUserText string) {
	if err := ctx.Err(); err != nil {
		return 0, "", ""
	}
	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return 0, "", ""
	}
	defer func() { _ = handle.Close() }()
	if !hasBlobsTable(ctx, handle.DB) {
		// Defensive fallback: a store using any other shape — including the
		// v0.6.0-rc.2 reader's messages/message/bubbles guess, which no real
		// store ever had — yields message_count 0 and no text, the same as
		// every Cursor session reported before either reader existed.
		return 0, "", ""
	}
	return scanBlobs(ctx, handle.DB)
}

// hasBlobsTable reports whether the store has the real schema's blobs
// table.
func hasBlobsTable(ctx context.Context, db *sql.DB) bool {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, blobsTable).Scan(&name)
	return err == nil
}

// scanBlobs reads every blobs row, bounded per row (maxRowTextBytes, via
// substr(CAST(data AS BLOB), 1, ?) in the query itself) and in total
// (maxBlobRows), and returns the count of user/assistant-role rows, the
// bounded and sanitized search text of the user-role ones, and the first
// such row's raw text. blobs has no ordering column; rowid is the only
// order available, and is treated as best-effort.
func scanBlobs(ctx context.Context, db *sql.DB) (count int, searchText string, firstUserText string) {
	rows, err := db.QueryContext(ctx,
		`SELECT substr(CAST(data AS BLOB), 1, ?) FROM `+blobsTable+` ORDER BY rowid LIMIT ?`,
		maxRowTextBytes, maxBlobRows)
	if err != nil {
		return 0, "", ""
	}
	defer func() { _ = rows.Close() }()
	var text sources.BoundedText
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			break
		}
		var prefix []byte
		if scanErr := rows.Scan(&prefix); scanErr != nil {
			continue
		}
		if len(prefix) == 0 || prefix[0] != '{' {
			// Not a JSON object: one of the non-JSON binary blobs (observed
			// first bytes 0x0a 0x20…, protobuf-like). Skipped by its first
			// byte, never decoded.
			continue
		}
		role, ok := decodeBlobRole(prefix)
		if !ok {
			// Malformed JSON, or a "role" field this bounded prefix could
			// not reach: not counted, matching this reader's policy of
			// never guessing at a shape it cannot confirm.
			continue
		}
		switch role {
		case "user":
			count++
			if userText := decodeBlobUserText(prefix); userText != "" {
				if firstUserText == "" {
					firstUserText = userText
				}
				text.Add(userText)
			}
		case "assistant":
			count++
			// "system" (and any other role value) is excluded from the
			// count, matching the user/assistant-only policy every other
			// reader in this package applies to search text.
		}
	}
	return count, text.String(), firstUserText
}

// decodeBlobRole reads the "role" field from a bounded JSON prefix of one
// blobs.data value, stopping as soon as it has that field so a "content"
// value many times larger than the prefix bound (maxRowTextBytes) is never
// decoded to reach it. Every key encountered before "role" is skipped by
// discarding its raw value rather than materializing it, which only
// fails — safely, as ok=false — when that earlier value itself runs past
// the prefix bound. The two real stores this reader was built against
// always wrote "role" first (observed key order role, content,
// providerOptions), so that is not the common case.
func decodeBlobRole(prefix []byte) (role string, ok bool) {
	dec := json.NewDecoder(bytes.NewReader(prefix))
	tok, err := dec.Token()
	if err != nil {
		return "", false
	}
	if delim, isDelim := tok.(json.Delim); !isDelim || delim != '{' {
		return "", false
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return "", false
		}
		key, isString := keyTok.(string)
		if !isString {
			return "", false
		}
		if key == "role" {
			var value string
			if err := dec.Decode(&value); err != nil {
				return "", false
			}
			return value, true
		}
		var discard json.RawMessage
		if err := dec.Decode(&discard); err != nil {
			return "", false
		}
	}
	return "", false
}

// decodeBlobUserText reads the "content" field of a bounded JSON prefix as
// plain text: a string, or the joined text of its type:"text" parts (a
// type:"reasoning" part is the model's own chain of thought, never user
// text, and is skipped the same way an assistant reply is — there are none
// in a user-role blob, but the shape is shared). When content runs past the
// prefix bound the object is no longer valid JSON as a whole, so this falls
// back to a raw scan for the value instead of contributing nothing.
func decodeBlobUserText(prefix []byte) string {
	var envelope struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(prefix, &envelope) == nil {
		return blobContentText(envelope.Content)
	}
	return rawBlobContentText(prefix)
}

// blobContentText normalizes a decoded "content" value to plain text.
func blobContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if json.Unmarshal(raw, &asString) == nil {
		return asString
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var joined strings.Builder
		for _, part := range parts {
			if part.Type != "text" || part.Text == "" {
				continue
			}
			if joined.Len() > 0 {
				joined.WriteByte('\n')
			}
			joined.WriteString(part.Text)
		}
		return joined.String()
	}
	return ""
}

// rawBlobContentText is the fallback for a "content" string value long
// enough to run past the bounded prefix (maxRowTextBytes): the surrounding
// object is not valid JSON once truncated, so this looks for the literal
// `"content":"` marker instead of decoding, and returns everything the
// prefix captured after it, with any torn trailing byte sequence dropped.
// The JSON backslash-escaping in that tail is deliberately left undone —
// this path exists to keep a pathologically large single row's leading text
// inside the bound (see TestSearchTextPerRowBoundLimitsMemory), not to
// reproduce an oversized value in full; no reader should pull one into
// memory whole. A parts-array content value that runs past the bound is not
// recovered here — the marker only matches a string value — and
// contributes no text, the same conservative default as an unrecognized
// shape.
func rawBlobContentText(prefix []byte) string {
	const marker = `"content":"`
	index := bytes.Index(prefix, []byte(marker))
	if index < 0 {
		return ""
	}
	return strings.ToValidUTF8(string(prefix[index+len(marker):]), "")
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
