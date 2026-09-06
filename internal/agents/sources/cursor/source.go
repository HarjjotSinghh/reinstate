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
	// in the sibling store.db. size_bytes and message_count both come from
	// the store the session lives in, not from the sidecar alone.
	sizeBytes := file.Size
	sourceModTime := file.ModTime.UnixNano()
	messageCount := 0
	storePath := storeDatabasePath(file.Path)
	if storeInfo, statErr := os.Stat(storePath); statErr == nil && storeInfo.Mode().IsRegular() {
		sizeBytes += storeInfo.Size()
		if storeModTime := storeInfo.ModTime().UnixNano(); storeModTime > sourceModTime {
			sourceModTime = storeModTime
		}
		messageCount = countStoreMessages(ctx, storePath)
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentCursor, id),
		ID:             id,
		Agent:          sessionindex.AgentCursor,
		Title:          sessionindex.SafePreview(title),
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      sizeBytes,
		MessageCount:   messageCount,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.CursorReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  sourceModTime,
		SourceSize:     sizeBytes,
		SearchText:     sessionindex.BuildSearchText(id, title, project, workspace),
	}, nil
}

// storeDatabasePath returns the store.db path beside a session's meta.json.
func storeDatabasePath(metaPath string) string {
	return filepath.Join(filepath.Dir(metaPath), StoreDatabaseName)
}

// countStoreMessages opens a session's store.db read-only through
// vendorsqlite (never writing under the vendor's root) and counts rows in
// whichever recognized message table the store actually has. Any failure to
// open, query, or recognize the schema yields 0, not an error: the session
// still gets a record from meta.json, just without a message count. Content
// is never read — only COUNT(*), computed by SQLite itself.
func countStoreMessages(ctx context.Context, path string) int {
	if err := ctx.Err(); err != nil {
		return 0
	}
	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = handle.Close() }()
	return countMessageRows(ctx, handle.DB)
}

func countMessageRows(ctx context.Context, db *sql.DB) int {
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('messages','message','bubbles')`)
	if err != nil {
		return 0
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
		return 0
	}
	if closeErr != nil {
		return 0
	}

	best := 0
	for _, table := range messageTableCandidates {
		if !present[table] {
			continue
		}
		if err := ctx.Err(); err != nil {
			return best
		}
		var query string
		switch table {
		case "messages":
			query = `SELECT COUNT(*) FROM messages`
		case "message":
			query = `SELECT COUNT(*) FROM message`
		case "bubbles":
			query = `SELECT COUNT(*) FROM bubbles`
		}
		var count int
		if scanErr := db.QueryRowContext(ctx, query).Scan(&count); scanErr != nil {
			continue
		}
		if count > best {
			best = count
		}
	}
	return best
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
