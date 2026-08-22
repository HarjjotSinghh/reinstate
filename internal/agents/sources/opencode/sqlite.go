package opencode

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/vendorsqlite"

	_ "modernc.org/sqlite"
)

// walSuffix is the write-ahead log SQLite keeps beside a journalled database.
const walSuffix = "-wal"

// DatabaseName is the OpenCode session store inside the data root.
const DatabaseName = "opencode.db"

// maxSessions bounds a single scan so a very large store cannot make
// `rein sessions` unbounded.
const maxSessions = 10000

// SQLiteSource reads OpenCode sessions straight from its embedded database.
//
// The vendor CLI is deliberately not invoked. `opencode session list` answers
// only for the directory it runs in, so a single query could never see a second
// project, and running it opened the vendor's database and left WAL and shared
// memory files behind under the agent root — a scan must not write there.
//
// The store is opened through vendorsqlite, which keeps that promise without
// paying for it in coverage. OpenCode journals in write-ahead mode and does not
// checkpoint on exit, so the sessions a user has just worked in sit in a -wal
// sidecar — often the entire store on a new install. An immutable in-place
// handle ignores that file by definition and reported those sessions as absent
// until unrelated vendor activity happened to cross SQLite's checkpoint
// threshold. Reading a private copy of the database and its log sees them, and
// still writes nothing under the agent root.
//
// Only the session, project and session_message tables are read. The same
// database also holds credential and account tables, and those are never
// opened.
type SQLiteSource struct {
	env agents.Env
}

// NewSQLite constructs the embedded-database source.
func NewSQLite(env agents.Env) (sessionindex.Source, error) {
	return &SQLiteSource{env: env}, nil
}

// Name returns the stable agent key.
func (s *SQLiteSource) Name() string { return sessionindex.AgentOpenCode }

// DatabasePath resolves the store inside the resolved data root.
func DatabasePath(env agents.Env) (string, error) {
	root, err := DataRoot(env)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", nil
	}
	return filepath.Join(root, DatabaseName), nil
}

// Fingerprint summarises the store by its path, modification time and size,
// without opening it. An unchanged store cannot yield different records, so an
// unchanged refresh skips the query entirely.
func (s *SQLiteSource) Fingerprint(ctx context.Context) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	path, err := DatabasePath(s.env)
	if err != nil || strings.TrimSpace(path) == "" {
		return "", false, nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false, nil
	}
	sum := sha256.New()
	_, _ = sum.Write([]byte(path))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write([]byte(strconv.FormatInt(info.Size(), 10)))
	// The write-ahead log is part of the store's state, so it is part of the
	// store's identity. A session written since the last checkpoint changes only
	// the log, leaving the main file's size and timestamp untouched — summarising
	// the main file alone reports "unchanged" and the refresh skips a scan that
	// would have found it.
	if walInfo, walErr := os.Stat(path + walSuffix); walErr == nil {
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(walInfo.ModTime().UnixNano(), 10)))
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(walInfo.Size(), 10)))
	}
	return hex.EncodeToString(sum.Sum(nil)), true, nil
}

// Scan reads every session the store knows about, across every project.
func (s *SQLiteSource) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	var result sessionindex.ScanResult

	path, err := DatabasePath(s.env)
	if err != nil || strings.TrimSpace(path) == "" {
		return result, nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		// An absent store is an absent agent, not a failure.
		return result, nil
	}

	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return result, warnOnly(&result, path, err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.DB
	if handle.Incomplete {
		// A short list presented as the whole store is worse than a short list
		// the operator knows is short.
		result.Warnings = append(result.Warnings, sessionindex.Warning{
			Agent:  sessionindex.AgentOpenCode,
			Source: path,
			Code:   "session_store_busy",
			Message: "OpenCode was writing to its store while it was read; sessions created " +
				"since it last checkpointed are missing from this listing",
		})
	}

	// OpenCode has carried messages in more than one table across schema
	// versions, so the count comes from whichever ones this store actually has
	// rather than from a hard-coded name.
	countExpr := messageCountExpression(ctx, db)

	rows, err := db.QueryContext(ctx, `
SELECT s.id,
       COALESCE(s.title, ''),
       COALESCE(s.directory, ''),
       COALESCE(p.name, ''),
       COALESCE(p.worktree, ''),
       COALESCE(s.time_updated, 0),
       COALESCE(s.time_created, 0),
       `+countExpr+`
  FROM session s
  LEFT JOIN project p ON p.id = s.project_id
 ORDER BY s.id
 LIMIT ?`, maxSessions)
	if err != nil {
		return result, warnOnly(&result, path, err)
	}
	defer func() { _ = rows.Close() }()

	modTime := info.ModTime().UnixNano()
	size := info.Size()
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		var (
			id, title, directory, projectName, worktree string
			updated, created                            int64
			messages                                    int
		)
		if err := rows.Scan(&id, &title, &directory, &projectName, &worktree,
			&updated, &created, &messages); err != nil {
			return result, warnOnly(&result, path, err)
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		result.Records = append(result.Records,
			recordFromRow(id, title, directory, projectName, worktree, updated, created, messages, path, modTime, size))
	}
	if err := rows.Err(); err != nil {
		return result, warnOnly(&result, path, err)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

// warnOnly degrades a store-level failure into a warning so one unreadable
// agent never fails the whole refresh.
func warnOnly(result *sessionindex.ScanResult, path string, cause error) error {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}
	result.Warnings = append(result.Warnings, sessionindex.Warning{
		Agent:   sessionindex.AgentOpenCode,
		Source:  path,
		Code:    "session_read_failed",
		Message: "OpenCode session store could not be read; other agents remain available",
	})
	return nil
}

// messageCountExpression picks the message tables present in this store. A
// store that has neither yields a constant zero rather than a failed scan.
func messageCountExpression(ctx context.Context, db *sql.DB) string {
	present := map[string]bool{}
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('message','session_message')`)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var name string
			if rows.Scan(&name) == nil {
				present[name] = true
			}
		}
	}
	var terms []string
	if present["message"] {
		terms = append(terms, "(SELECT COUNT(*) FROM message m WHERE m.session_id = s.id)")
	}
	if present["session_message"] {
		terms = append(terms, "(SELECT COUNT(*) FROM session_message sm WHERE sm.session_id = s.id)")
	}
	switch len(terms) {
	case 0:
		return "0"
	case 1:
		return terms[0]
	default:
		// One table is the live one and the other a migrated remnant; the
		// larger count is the real conversation length, and summing would
		// double count a store mid-migration.
		return "MAX(" + terms[0] + ", " + terms[1] + ")"
	}
}

func recordFromRow(
	id, title, directory, projectName, worktree string,
	updated, created int64, messages int,
	path string, modTime, size int64,
) sessionindex.Record {
	workspace := strings.TrimSpace(directory)
	if workspace == "" {
		workspace = strings.TrimSpace(worktree)
	}

	// A human project name first, then the directory it lives in. The vendor's
	// opaque project id is never used as a display name.
	project := strings.TrimSpace(projectName)
	if project == "" && workspace != "" {
		project = sources.PortableBase(workspace)
	}
	if project == "" {
		project = "unknown"
	}

	stamp := updated
	if stamp == 0 {
		stamp = created
	}

	safeTitle := sessionindex.SafePreview(title)
	if safeTitle == "" {
		safeTitle = id
	}

	// OpenCode continues a session by id — `opencode --session <id>`, plus
	// `--fork` for a branch — and it starts that session in a working
	// directory. A row whose directory the vendor never recorded has nowhere to
	// be launched, so it stays read-only with that stated as the reason rather
	// than being offered and then refused at the launch boundary.
	resumable := workspace != ""
	reason := ""
	if !resumable {
		reason = readOnlyReasonNoWorkspace
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentOpenCode, id),
		ID:             id,
		Agent:          sessionindex.AgentOpenCode,
		Title:          safeTitle,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      unixMillisOrSeconds(stamp),
		MessageCount:   messages,
		CanResume:      resumable,
		CanFork:        resumable,
		ReadOnlyReason: reason,
		SourcePath:     path,
		SourceModTime:  modTime,
		SourceSize:     size,
		SearchText:     sessionindex.BuildSearchText(id, safeTitle, project, workspace),
	}
}

// unixMillisOrSeconds accepts either encoding; OpenCode records milliseconds,
// but a store written by an older build may carry seconds.
func unixMillisOrSeconds(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 1e12 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}
