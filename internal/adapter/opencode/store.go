package opencode

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
	"runtime"
	"sort"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/vendorsqlite"

	_ "modernc.org/sqlite"
)

// sessionDocument is the portable, deterministic representation of one OpenCode
// session: exactly the session, project, message and part rows needed to
// resume it, with every absolute path normalised to a portable token. It never
// carries a row from the credential or account tables.
//
// Field order and the sorted rows below make json.Marshal deterministic, which
// the sync engine relies on for content-addressed change detection.
type sessionDocument struct {
	Schema string `json:"schema"`
	// ProjectKey is Reinstate's portable project identity for the session —
	// the same value Discover reports as adapter.Session.ProjectID — so the
	// document names the project the sync envelope was filed under. It is
	// derived from the session's working directory (a configured project id,
	// else a ${HOME}-relative token) and so survives a cross-device path remap;
	// it is deliberately not the vendor's own project-table id, which lives in
	// Session.ProjectID and is only meaningful inside one store.
	ProjectKey string       `json:"project_key,omitempty"`
	Session    sessionRow   `json:"session"`
	Project    *projectRow  `json:"project,omitempty"`
	Messages   []messageRow `json:"messages"`
	Parts      []partRow    `json:"parts"`
}

// sessionRow carries the columns of the vendor's session table that a resume
// depends on. Columns are always named in SQL, never positional, so a store
// with more columns than these (1.18.21 has 29) reads and writes the same way
// as the synthetic seeds.
type sessionRow struct {
	ID string `json:"id"`
	// ProjectID is the vendor's project-table id ("global" for a session
	// outside any project, a 40-hex digest for a repository). It is the
	// session's foreign key inside its own store, not a portable identity.
	ProjectID   string          `json:"project_id"`
	Slug        string          `json:"slug"`
	Directory   string          `json:"directory"`
	Path        string          `json:"path,omitempty"`
	Title       string          `json:"title"`
	Version     string          `json:"version"`
	Agent       string          `json:"agent,omitempty"`
	Model       string          `json:"model,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	TimeCreated int64           `json:"time_created"`
	TimeUpdated int64           `json:"time_updated"`
}

type projectRow struct {
	ID          string `json:"id"`
	Worktree    string `json:"worktree"`
	VCS         string `json:"vcs,omitempty"`
	Name        string `json:"name,omitempty"`
	Sandboxes   string `json:"sandboxes"`
	TimeCreated int64  `json:"time_created"`
	TimeUpdated int64  `json:"time_updated"`
}

type messageRow struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"session_id"`
	TimeCreated int64           `json:"time_created"`
	TimeUpdated int64           `json:"time_updated"`
	Data        json.RawMessage `json:"data"`
}

type partRow struct {
	ID          string          `json:"id"`
	MessageID   string          `json:"message_id"`
	SessionID   string          `json:"session_id"`
	TimeCreated int64           `json:"time_created"`
	TimeUpdated int64           `json:"time_updated"`
	Data        json.RawMessage `json:"data"`
}

// readSessionDocument reads one session from a store read-only and returns its
// normalised portable document.
func (a *Adapter) readSessionDocument(ctx context.Context, dbPath, sessionID string) (sessionDocument, error) {
	if strings.TrimSpace(sessionID) == "" {
		return sessionDocument{}, fmt.Errorf("opencode export requires a session id")
	}
	handle, err := vendorsqlite.Open(dbPath)
	if err != nil {
		return sessionDocument{}, fmt.Errorf("open opencode store: %w", err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.DB

	mapper := a.mapper()
	doc := sessionDocument{Schema: exportSchema}

	var (
		sess         sessionRow
		path         sql.NullString
		agent, model sql.NullString
		metadata     sql.NullString
		projectID    string
	)
	err = db.QueryRowContext(ctx, `
SELECT id, project_id, slug, directory, path, title, version, agent, model, metadata,
       COALESCE(time_created, 0), COALESCE(time_updated, 0)
  FROM session WHERE id = ?`, sessionID).Scan(
		&sess.ID, &projectID, &sess.Slug, &sess.Directory, &path, &sess.Title, &sess.Version,
		&agent, &model, &metadata, &sess.TimeCreated, &sess.TimeUpdated)
	if err == sql.ErrNoRows {
		return sessionDocument{}, fmt.Errorf("opencode session %q not found", sessionID)
	}
	if err != nil {
		return sessionDocument{}, err
	}
	sess.ProjectID = projectID
	doc.ProjectKey = a.projectKey(sess.Directory)
	sess.Directory = mapper.Normalize(sess.Directory)
	if path.Valid && path.String != "" {
		sess.Path = mapper.Normalize(path.String)
	}
	sess.Agent = agent.String
	sess.Model = model.String
	if metadata.Valid && metadata.String != "" {
		sess.Metadata = json.RawMessage(metadata.String)
	}
	doc.Session = sess

	if strings.TrimSpace(projectID) != "" {
		project, ok, perr := readProject(ctx, db, projectID, mapper.Normalize)
		if perr != nil {
			return sessionDocument{}, perr
		}
		if ok {
			doc.Project = &project
		}
	}

	messages, err := readMessages(ctx, db, sessionID, mapper.Normalize)
	if err != nil {
		return sessionDocument{}, err
	}
	doc.Messages = messages

	parts, err := readParts(ctx, db, sessionID, mapper.Normalize)
	if err != nil {
		return sessionDocument{}, err
	}
	doc.Parts = parts
	return doc, nil
}

func readProject(ctx context.Context, db *sql.DB, id string, mapPath func(string) string) (projectRow, bool, error) {
	var (
		p         projectRow
		vcs       sql.NullString
		name      sql.NullString
		sandboxes sql.NullString
	)
	err := db.QueryRowContext(ctx, `
SELECT id, worktree, vcs, name, sandboxes, COALESCE(time_created, 0), COALESCE(time_updated, 0)
  FROM project WHERE id = ?`, id).Scan(&p.ID, &p.Worktree, &vcs, &name, &sandboxes, &p.TimeCreated, &p.TimeUpdated)
	if err == sql.ErrNoRows {
		return projectRow{}, false, nil
	}
	if err != nil {
		return projectRow{}, false, err
	}
	p.Worktree = mapPath(p.Worktree)
	p.VCS = vcs.String
	p.Name = name.String
	p.Sandboxes = sandboxes.String
	if p.Sandboxes == "" {
		p.Sandboxes = "[]"
	}
	return p, true, nil
}

func readMessages(ctx context.Context, db *sql.DB, sessionID string, mapPath func(string) string) ([]messageRow, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, COALESCE(time_created, 0), COALESCE(time_updated, 0), data
  FROM message WHERE session_id = ? ORDER BY id LIMIT ?`, sessionID, maxExportMessages+1)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []messageRow
	for rows.Next() {
		var m messageRow
		var data []byte
		if err := rows.Scan(&m.ID, &m.TimeCreated, &m.TimeUpdated, &data); err != nil {
			return nil, err
		}
		m.SessionID = sessionID
		m.Data = rewriteJSONPaths(data, mapPath)
		out = append(out, m)
		if len(out) > maxExportMessages {
			return nil, fmt.Errorf("opencode: session %s has more than %d messages; refusing a truncated export", sessionID, maxExportMessages)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func readParts(ctx context.Context, db *sql.DB, sessionID string, mapPath func(string) string) ([]partRow, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, message_id, COALESCE(time_created, 0), COALESCE(time_updated, 0), data
  FROM part WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []partRow
	for rows.Next() {
		var p partRow
		var data []byte
		if err := rows.Scan(&p.ID, &p.MessageID, &p.TimeCreated, &p.TimeUpdated, &data); err != nil {
			return nil, err
		}
		p.SessionID = sessionID
		p.Data = rewriteJSONPaths(data, mapPath)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// applyDocument writes one session document into a writable store copy. When
// forkID differs from sourceID the session identity and every message/part id
// derived from it are remapped so a fork lands alongside the original instead
// of colliding with it.
func (a *Adapter) applyDocument(ctx context.Context, dbPath string, doc sessionDocument, sourceID, forkID string) error {
	denorm := a.denormalizer()

	// Denormalize structured path columns onto the destination platform.
	doc.Session.Directory = denorm(doc.Session.Directory)
	if doc.Session.Path != "" {
		doc.Session.Path = denorm(doc.Session.Path)
	}
	if doc.Project != nil {
		doc.Project.Worktree = denorm(doc.Project.Worktree)
	}
	// Denormalize path tokens embedded in message and part bodies too.
	for i := range doc.Messages {
		doc.Messages[i].Data = rewriteJSONPaths(doc.Messages[i].Data, denorm)
	}
	for i := range doc.Parts {
		doc.Parts[i].Data = rewriteJSONPaths(doc.Parts[i].Data, denorm)
	}

	forking := forkID != "" && forkID != sourceID
	targetID := doc.Session.ID
	if forking {
		targetID = forkID
	}
	if strings.TrimSpace(targetID) == "" {
		return fmt.Errorf("opencode restore requires a session id")
	}

	messageIDs := map[string]string{}
	for i := range doc.Messages {
		newID := doc.Messages[i].ID
		if forking {
			newID = derivedID("msg", forkID, doc.Messages[i].ID)
		}
		messageIDs[doc.Messages[i].ID] = newID
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// The session row's project_id is a foreign key into the project table
	// (ON DELETE CASCADE on 1.18.21), so the project row must exist before the
	// session is written. A project the destination already knows keeps its
	// own worktree: the destination created that row from its own filesystem,
	// and the source's remapped guess must not move it. A project the
	// destination has never seen is created from the document, remapped onto
	// this device; a document with no project row at all (the source store was
	// missing it) still gets a minimal row so the session can be written.
	if doc.Project != nil {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO project (id, worktree, vcs, name, sandboxes, time_created, time_updated)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET name=COALESCE(project.name, excluded.name)`,
			doc.Project.ID, doc.Project.Worktree, nullIfEmpty(doc.Project.VCS), nullIfEmpty(doc.Project.Name),
			doc.Project.Sandboxes, doc.Project.TimeCreated, doc.Project.TimeUpdated); err != nil {
			return fmt.Errorf("restore opencode project: %w", err)
		}
	} else if strings.TrimSpace(doc.Session.ProjectID) != "" {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO project (id, worktree, sandboxes, time_created, time_updated)
VALUES (?, ?, '[]', ?, ?)
ON CONFLICT(id) DO NOTHING`,
			doc.Session.ProjectID, doc.Session.Directory, doc.Session.TimeCreated, doc.Session.TimeUpdated); err != nil {
			return fmt.Errorf("restore opencode project placeholder: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO session (id, project_id, slug, directory, path, title, version, agent, model, metadata, time_created, time_updated)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  project_id=excluded.project_id, slug=excluded.slug, directory=excluded.directory,
  path=excluded.path, title=excluded.title, version=excluded.version,
  agent=excluded.agent, model=excluded.model, metadata=excluded.metadata,
  time_updated=excluded.time_updated`,
		targetID, doc.Session.ProjectID, doc.Session.Slug, doc.Session.Directory,
		nullIfEmpty(doc.Session.Path), doc.Session.Title, doc.Session.Version,
		nullIfEmpty(doc.Session.Agent), nullIfEmpty(doc.Session.Model),
		rawOrNull(doc.Session.Metadata), doc.Session.TimeCreated, doc.Session.TimeUpdated); err != nil {
		return fmt.Errorf("restore opencode session: %w", err)
	}

	for _, m := range doc.Messages {
		newID := messageIDs[m.ID]
		data := rewriteSessionRef(m.Data, sourceID, targetID)
		if forking {
			// A forked message must point at the fork's own copies of its
			// neighbours (its parent user message, its own id), never back
			// into the original session.
			data = rewriteIDRefs(data, messageIDs, messageIDKeys)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET session_id=excluded.session_id, time_updated=excluded.time_updated, data=excluded.data`,
			newID, targetID, m.TimeCreated, m.TimeUpdated, string(data)); err != nil {
			return fmt.Errorf("restore opencode message %s: %w", m.ID, err)
		}
	}

	for _, p := range doc.Parts {
		newID := p.ID
		newMsgID := messageIDs[p.MessageID]
		if newMsgID == "" {
			newMsgID = p.MessageID
		}
		data := rewriteSessionRef(p.Data, sourceID, targetID)
		if forking {
			newID = derivedID("prt", forkID, p.ID)
			data = rewriteIDRefs(data, messageIDs, messageIDKeys)
			data = rewriteIDRefs(data, map[string]string{p.ID: newID}, partIDKeys)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET message_id=excluded.message_id, session_id=excluded.session_id, time_updated=excluded.time_updated, data=excluded.data`,
			newID, newMsgID, targetID, p.TimeCreated, p.TimeUpdated, string(data)); err != nil {
			return fmt.Errorf("restore opencode part %s: %w", p.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// denormalizer returns a path mapper that converts portable tokens back to the
// destination platform's absolute paths and forces the destination separator.
//
// pathmap.Denormalize joins with the host's filepath separator, which is wrong
// when a Windows session is restored from a build that was cross-compiled or
// when a test emulates the other platform. Forcing the separator by the
// adapter's declared GOOS keeps a restored path platform-correct regardless of
// the host that ran the restore — the platform-free normalization the project
// requires of workspace paths.
func (a *Adapter) denormalizer() func(string) string {
	m := a.mapper()
	goos := a.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	return func(p string) string {
		if !strings.HasPrefix(p, "${") {
			// A path that was never tokenised (no ${HOME} or ${REPO:} prefix)
			// belongs to no known root on this device, so swapping its
			// separators would only manufacture a path that exists nowhere.
			return p
		}
		out := m.Denormalize(p)
		if goos == "windows" {
			return strings.ReplaceAll(out, "/", `\`)
		}
		return strings.ReplaceAll(out, `\`, "/")
	}
}

func derivedID(prefix, forkID, sourceID string) string {
	sum := sha256.Sum256([]byte(forkID + "\x00" + sourceID))
	return prefix + "_fork" + hex.EncodeToString(sum[:8])
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func rawOrNull(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

// checkpointedCopy makes a private, write-ahead-merged copy of a store so
// edits happen off to the side and the live store is only replaced by an atomic
// rename. The copy lives in a hidden directory beside the store, on the same
// volume, because os.Rename does not cross filesystems: a copy under the
// system temp directory would fail the final rename with EXDEV on hosts where
// /tmp is tmpfs, or on Windows when %TEMP% is on another drive, after the
// backup had already been taken. The Claude and Codex adapters stage their
// restores beside the destination for the same reason.
func checkpointedCopy(src string) (string, func(), error) {
	tempDir, err := os.MkdirTemp(filepath.Dir(src), ".reinstate-opencode-restore-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	copyPath := filepath.Join(tempDir, DatabaseName)
	// The working copy replaces the vendor's file, so it keeps the vendor's mode.
	if err := copyFile(src, copyPath); err != nil {
		cleanup()
		return "", func() {}, err
	}
	for _, suffix := range storeSidecars {
		if _, err := os.Stat(src + suffix); err != nil {
			continue
		}
		if err := copyFile(src+suffix, copyPath+suffix); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	// Fold any write-ahead log into the copy's main file, then drop the sidecars
	// so the renamed file is a self-contained database.
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(copyPath)+"?_pragma=busy_timeout(5000)")
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		_ = db.Close()
		cleanup()
		return "", func() {}, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=DELETE`); err != nil {
		_ = db.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := db.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	_ = os.Remove(copyPath + "-wal")
	_ = os.Remove(copyPath + "-shm")
	return copyPath, cleanup, nil
}

// storeSidecars are the write-ahead-log files SQLite keeps beside a WAL-mode
// database. Rows the vendor has committed but not yet checkpointed live only
// in the -wal file, so any faithful copy of the store must carry them.
var storeSidecars = []string{"-wal", "-shm"}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if info, err := in.Stat(); err == nil {
		return os.Chmod(destination, info.Mode().Perm())
	}
	return nil
}
