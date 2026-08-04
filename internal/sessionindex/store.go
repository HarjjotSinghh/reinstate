package sessionindex

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

const indexFileName = "session-index-v1.sqlite"

var errIncompatibleSchema = errors.New("incompatible session-index schema")

// ErrPrelaunchBaselineNotFound means no private Reinstate observation exists
// for the requested session reference.
var ErrPrelaunchBaselineNotFound = errors.New("prelaunch baseline not found")

// ReplaceResult summarizes one atomic source replacement.
type ReplaceResult struct {
	Upserted  int `json:"upserted"`
	Unchanged int `json:"unchanged"`
	Deleted   int `json:"deleted"`
}

// Store owns the private derived SQLite index.
type Store struct {
	path string

	mu sync.RWMutex
	db *sql.DB
}

// IndexPath returns the Phase 2 index location below a Reinstate home.
func IndexPath(reinstateHome string) string {
	return filepath.Join(reinstateHome, "cache", indexFileName)
}

// OpenDefault opens the derived index below reinstateHome.
func OpenDefault(reinstateHome string) (*Store, error) {
	if strings.TrimSpace(reinstateHome) == "" {
		return nil, errors.New("reinstate home must not be empty")
	}
	return Open(IndexPath(reinstateHome))
}

// Open opens a private derived index. Corrupt or incompatible derived state is
// removed and rebuilt without touching vendor session files.
func Open(path string) (*Store, error) {
	path = filepath.Clean(path)
	if path == "." || !filepath.IsAbs(path) {
		return nil, errors.New("session-index path must be absolute")
	}
	if err := ensurePrivateParent(path); err != nil {
		return nil, err
	}

	db, err := openDatabase(path)
	if err != nil {
		if !isRebuildableDatabaseError(err) {
			return nil, err
		}
		if removeErr := removeDatabaseFiles(path); removeErr != nil {
			return nil, errors.Join(err, removeErr)
		}
		db, err = openDatabase(path)
		if err != nil {
			return nil, fmt.Errorf("rebuild session index: %w", err)
		}
	}
	return &Store{path: path, db: db}, nil
}

// Path returns the SQLite file path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close releases the SQLite connection.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Rebuild deletes only the derived index and creates an empty current schema.
func (s *Store) Rebuild(ctx context.Context) error {
	if s == nil {
		return errors.New("session-index store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	if err := removeDatabaseFiles(s.path); err != nil {
		return err
	}
	db, err := openDatabase(s.path)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}

// ReplaceSource atomically upserts a complete successful scan and deletes
// records that disappeared from that source. Callers must not invoke it after a
// failed scan.
func (s *Store) ReplaceSource(
	ctx context.Context,
	source string,
	records []Record,
) (ReplaceResult, error) {
	var result ReplaceResult
	source = strings.ToLower(SafeText(strings.TrimSpace(source), 64))
	if source == "" {
		return result, errors.New("source name must not be empty")
	}

	normalized := make([]Record, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		var err error
		record, err = NormalizeRecord(record)
		if err != nil {
			return result, fmt.Errorf("normalize %s record: %w", source, err)
		}
		if record.Agent != source {
			return result, fmt.Errorf(
				"source %q returned record for agent %q",
				source,
				record.Agent,
			)
		}
		if _, duplicate := seen[record.Key]; duplicate {
			return result, fmt.Errorf("source %q returned duplicate session %q", source, record.Key)
		}
		seen[record.Key] = struct{}{}
		normalized = append(normalized, record)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return result, errors.New("session-index store is closed")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()

	existingRows, err := tx.QueryContext(
		ctx,
		`SELECT key, source_path, source_mod_time, source_size
		 FROM sessions WHERE source = ?`,
		source,
	)
	if err != nil {
		return result, err
	}
	type fingerprint struct {
		path    string
		modTime int64
		size    int64
	}
	existing := make(map[string]fingerprint)
	for existingRows.Next() {
		var key string
		var value fingerprint
		if err := existingRows.Scan(&key, &value.path, &value.modTime, &value.size); err != nil {
			_ = existingRows.Close()
			return result, err
		}
		existing[key] = value
	}
	if err := existingRows.Close(); err != nil {
		return result, err
	}
	if err := existingRows.Err(); err != nil {
		return result, err
	}

	for _, record := range normalized {
		previous, exists := existing[record.Key]
		if exists &&
			previous.path == record.SourcePath &&
			previous.modTime == record.SourceModTime &&
			previous.size == record.SourceSize {
			result.Unchanged++
			delete(existing, record.Key)
			continue
		}
		if err := upsertRecord(ctx, tx, source, record); err != nil {
			return result, err
		}
		result.Upserted++
		delete(existing, record.Key)
	}

	for key := range existing {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE key = ? AND source = ?`, key, source); err != nil {
			return result, err
		}
		result.Deleted++
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

// Search returns deterministic literal, case-insensitive matches.
func (s *Store) Search(ctx context.Context, filter Filter) ([]Record, error) {
	conditions := []string{"1 = 1"}
	var args []any

	agent := strings.ToLower(SafeText(strings.TrimSpace(filter.Agent), 64))
	if agent != "" && agent != "all" {
		conditions = append(conditions, "agent = ?")
		args = append(args, agent)
	}
	for _, term := range strings.Fields(SafeText(filter.Query, 0)) {
		conditions = append(conditions, "instr(search_text, ?) > 0")
		args = append(args, strings.ToLower(term))
	}
	if project := SafeText(filter.Project, maxMetadataRunes); project != "" {
		conditions = append(
			conditions,
			"(instr(project_fold, ?) > 0 OR instr(workspace_fold, ?) > 0)",
		)
		project = strings.ToLower(project)
		args = append(args, project, project)
	}
	if branch := SafeText(filter.Branch, maxMetadataRunes); branch != "" {
		conditions = append(conditions, "instr(branch_fold, ?) > 0")
		args = append(args, strings.ToLower(branch))
	}
	if file := SafeText(filter.File, MaxFileReferenceRunes); file != "" {
		conditions = append(conditions, "instr(file_text, ?) > 0")
		args = append(args, strings.ToLower(file))
	}
	if filter.ResumableOnly {
		conditions = append(conditions, "can_resume = 1")
	}

	query := recordSelect + `
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY updated_at DESC, agent ASC, id ASC
		LIMIT ?`
	args = append(args, filter.EffectiveLimit())
	return s.queryRecords(ctx, query, args...)
}

// All returns all records with the deterministic Phase 2 ordering.
func (s *Store) All(ctx context.Context, limit int) ([]Record, error) {
	return s.Search(ctx, Filter{Limit: limit})
}

// Resolve accepts an exact qualified reference or an unambiguous bare native ID.
func (s *Store) Resolve(ctx context.Context, reference string) (Record, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return Record{}, fmt.Errorf("%w: empty reference", ErrNotFound)
	}
	if agent, nativeID, qualified := ParseCompositeReference(reference); qualified {
		records, err := s.queryRecords(
			ctx,
			recordSelect+` WHERE agent = ? AND id = ? LIMIT 2`,
			strings.ToLower(agent),
			nativeID,
		)
		if err != nil {
			return Record{}, err
		}
		if len(records) == 0 {
			return Record{}, fmt.Errorf("%w: %s", ErrNotFound, reference)
		}
		return records[0], nil
	}

	records, err := s.queryRecords(
		ctx,
		recordSelect+` WHERE id = ? ORDER BY agent ASC, id ASC`,
		reference,
	)
	if err != nil {
		return Record{}, err
	}
	switch len(records) {
	case 0:
		return Record{}, fmt.Errorf("%w: %s", ErrNotFound, reference)
	case 1:
		return records[0], nil
	default:
		matches := make([]string, 0, len(records))
		for _, record := range records {
			matches = append(matches, record.Reference())
		}
		sort.Strings(matches)
		return Record{}, &AmbiguousReferenceError{Reference: reference, Matches: matches}
	}
}

// Last returns the newest record matching a filter.
func (s *Store) Last(ctx context.Context, filter Filter) (Record, error) {
	filter.Limit = 1
	records, err := s.Search(ctx, filter)
	if err != nil {
		return Record{}, err
	}
	if len(records) == 0 {
		return Record{}, ErrNotFound
	}
	return records[0], nil
}

// PutPrelaunchBaseline stores one private, derived observation independently
// from vendor source fingerprints. A later native append may update the session
// record without invalidating the last prelaunch truth Reinstate observed.
func (s *Store) PutPrelaunchBaseline(ctx context.Context, baseline environment.PrelaunchBaseline) error {
	baseline, err := environment.NormalizePrelaunchBaseline(baseline)
	if err != nil {
		return fmt.Errorf("normalize prelaunch baseline: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(baseline.Capabilities)
	if err != nil {
		return fmt.Errorf("encode prelaunch capabilities: %w", err)
	}
	runtimesJSON, err := json.Marshal(baseline.Runtimes)
	if err != nil {
		return fmt.Errorf("encode prelaunch runtimes: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("session-index store is closed")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO prelaunch_baselines (
		session_ref, repository_id, branch, git_head, working_tree_digest,
		working_tree_state, observed_at, provenance, source_session_ref,
		capabilities_json, runtimes_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_ref) DO UPDATE SET
		repository_id = excluded.repository_id,
		branch = excluded.branch,
		git_head = excluded.git_head,
		working_tree_digest = excluded.working_tree_digest,
		working_tree_state = excluded.working_tree_state,
		observed_at = excluded.observed_at,
		provenance = excluded.provenance,
		source_session_ref = excluded.source_session_ref,
		capabilities_json = excluded.capabilities_json,
		runtimes_json = excluded.runtimes_json`,
		baseline.SessionRef,
		baseline.RepositoryID,
		baseline.Branch,
		baseline.GitHead,
		baseline.WorkingTreeDigest,
		string(baseline.WorkingTreeState),
		timeToDatabase(baseline.ObservedAt),
		baseline.Provenance,
		baseline.SourceSessionRef,
		string(capabilitiesJSON),
		string(runtimesJSON),
	)
	return err
}

// GetPrelaunchBaseline returns the last private Reinstate observation for a
// session. It never reads or refreshes vendor state.
func (s *Store) GetPrelaunchBaseline(ctx context.Context, sessionRef string) (environment.PrelaunchBaseline, error) {
	sessionRef = SafeText(strings.TrimSpace(sessionRef), environment.MaxSessionReferenceRunes)
	if sessionRef == "" {
		return environment.PrelaunchBaseline{}, fmt.Errorf("%w: empty session reference", ErrPrelaunchBaselineNotFound)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return environment.PrelaunchBaseline{}, errors.New("session-index store is closed")
	}
	var baseline environment.PrelaunchBaseline
	var workingTreeState string
	var observedAt int64
	var capabilitiesJSON, runtimesJSON string
	err := s.db.QueryRowContext(ctx, `SELECT
		session_ref, repository_id, branch, git_head, working_tree_digest,
		working_tree_state, observed_at, provenance, source_session_ref,
		capabilities_json, runtimes_json
		FROM prelaunch_baselines WHERE session_ref = ?`, sessionRef).Scan(
		&baseline.SessionRef,
		&baseline.RepositoryID,
		&baseline.Branch,
		&baseline.GitHead,
		&baseline.WorkingTreeDigest,
		&workingTreeState,
		&observedAt,
		&baseline.Provenance,
		&baseline.SourceSessionRef,
		&capabilitiesJSON,
		&runtimesJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return environment.PrelaunchBaseline{}, fmt.Errorf("%w: %s", ErrPrelaunchBaselineNotFound, sessionRef)
	}
	if err != nil {
		return environment.PrelaunchBaseline{}, err
	}
	baseline.WorkingTreeState = environment.WorkingTreeState(workingTreeState)
	baseline.ObservedAt = timeFromDatabase(observedAt)
	if err := json.Unmarshal([]byte(capabilitiesJSON), &baseline.Capabilities); err != nil {
		return environment.PrelaunchBaseline{}, fmt.Errorf("decode prelaunch capabilities: %w", err)
	}
	if err := json.Unmarshal([]byte(runtimesJSON), &baseline.Runtimes); err != nil {
		return environment.PrelaunchBaseline{}, fmt.Errorf("decode prelaunch runtimes: %w", err)
	}
	baseline, err = environment.NormalizePrelaunchBaseline(baseline)
	if err != nil {
		return environment.PrelaunchBaseline{}, fmt.Errorf("decode prelaunch baseline: %w", err)
	}
	return baseline, nil
}

// DeletePrelaunchBaseline removes one private observation without touching the
// vendor session or its indexed record.
func (s *Store) DeletePrelaunchBaseline(ctx context.Context, sessionRef string) error {
	sessionRef = SafeText(strings.TrimSpace(sessionRef), environment.MaxSessionReferenceRunes)
	if sessionRef == "" {
		return errors.New("prelaunch baseline session reference must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("session-index store is closed")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM prelaunch_baselines WHERE session_ref = ?`, sessionRef)
	return err
}

func (s *Store) queryRecords(ctx context.Context, query string, args ...any) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return nil, errors.New("session-index store is closed")
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []Record
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func upsertRecord(ctx context.Context, tx *sql.Tx, source string, record Record) error {
	filesJSON, err := json.Marshal(record.Files)
	if err != nil {
		return err
	}
	recordedEnvironmentJSON, err := json.Marshal(record.RecordedEnvironment)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO sessions (
			key, id, agent, source, title, project, workspace, branch,
			project_fold, workspace_fold, branch_fold, updated_at, size_bytes,
			message_count, prompt_preview, files_json, file_text, can_resume,
				can_fork, read_only_reason, recorded_environment_json, source_path,
				source_mod_time, source_size, search_text
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			id = excluded.id,
			agent = excluded.agent,
			source = excluded.source,
			title = excluded.title,
			project = excluded.project,
			workspace = excluded.workspace,
			branch = excluded.branch,
			project_fold = excluded.project_fold,
			workspace_fold = excluded.workspace_fold,
			branch_fold = excluded.branch_fold,
			updated_at = excluded.updated_at,
			size_bytes = excluded.size_bytes,
			message_count = excluded.message_count,
			prompt_preview = excluded.prompt_preview,
			files_json = excluded.files_json,
			file_text = excluded.file_text,
			can_resume = excluded.can_resume,
				can_fork = excluded.can_fork,
				read_only_reason = excluded.read_only_reason,
				recorded_environment_json = excluded.recorded_environment_json,
				source_path = excluded.source_path,
			source_mod_time = excluded.source_mod_time,
			source_size = excluded.source_size,
			search_text = excluded.search_text`,
		record.Key,
		record.ID,
		record.Agent,
		source,
		record.Title,
		record.Project,
		record.Workspace,
		record.Branch,
		strings.ToLower(record.Project),
		strings.ToLower(record.Workspace),
		strings.ToLower(record.Branch),
		timeToDatabase(record.UpdatedAt),
		record.SizeBytes,
		record.MessageCount,
		record.PromptPreview,
		string(filesJSON),
		strings.ToLower(strings.Join(record.Files, "\n")),
		boolToDatabase(record.CanResume),
		boolToDatabase(record.CanFork),
		record.ReadOnlyReason,
		string(recordedEnvironmentJSON),
		record.SourcePath,
		record.SourceModTime,
		record.SourceSize,
		strings.ToLower(record.SearchText),
	)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(scanner rowScanner) (Record, error) {
	var record Record
	var updatedAt int64
	var filesJSON string
	var recordedEnvironmentJSON string
	var canResume, canFork int
	err := scanner.Scan(
		&record.Key,
		&record.ID,
		&record.Agent,
		&record.Title,
		&record.Project,
		&record.Workspace,
		&record.Branch,
		&updatedAt,
		&record.SizeBytes,
		&record.MessageCount,
		&record.PromptPreview,
		&filesJSON,
		&canResume,
		&canFork,
		&record.ReadOnlyReason,
		&recordedEnvironmentJSON,
		&record.SourcePath,
		&record.SourceModTime,
		&record.SourceSize,
		&record.SearchText,
	)
	if err != nil {
		return Record{}, err
	}
	record.UpdatedAt = timeFromDatabase(updatedAt)
	record.CanResume = canResume != 0
	record.CanFork = canFork != 0
	if err := json.Unmarshal([]byte(filesJSON), &record.Files); err != nil {
		return Record{}, fmt.Errorf("decode indexed files for %s: %w", record.Key, err)
	}
	if err := json.Unmarshal([]byte(recordedEnvironmentJSON), &record.RecordedEnvironment); err != nil {
		return Record{}, fmt.Errorf("decode indexed environment for %s: %w", record.Key, err)
	}
	record.RecordedEnvironment, err = environment.NormalizeRecordedEnvironment(record.RecordedEnvironment)
	if err != nil {
		return Record{}, fmt.Errorf("normalize indexed environment for %s: %w", record.Key, err)
	}
	return record, nil
}

const recordSelect = `SELECT
	key, id, agent, title, project, workspace, branch, updated_at, size_bytes,
	message_count, prompt_preview, files_json, can_resume, can_fork,
	read_only_reason, recorded_environment_json, source_path, source_mod_time,
	source_size, search_text
	FROM sessions`

const createSchema = `CREATE TABLE IF NOT EXISTS sessions (
	key TEXT PRIMARY KEY NOT NULL,
	id TEXT NOT NULL,
	agent TEXT NOT NULL,
	source TEXT NOT NULL,
	title TEXT NOT NULL,
	project TEXT NOT NULL,
	workspace TEXT NOT NULL,
	branch TEXT NOT NULL,
	project_fold TEXT NOT NULL,
	workspace_fold TEXT NOT NULL,
	branch_fold TEXT NOT NULL,
	updated_at INTEGER NOT NULL,
	size_bytes INTEGER NOT NULL,
	message_count INTEGER NOT NULL,
	prompt_preview TEXT NOT NULL,
	files_json TEXT NOT NULL,
	file_text TEXT NOT NULL,
	can_resume INTEGER NOT NULL,
	can_fork INTEGER NOT NULL,
	read_only_reason TEXT NOT NULL,
	recorded_environment_json TEXT NOT NULL,
	source_path TEXT NOT NULL,
	source_mod_time INTEGER NOT NULL,
	source_size INTEGER NOT NULL,
	search_text TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_source ON sessions(source);
CREATE INDEX IF NOT EXISTS sessions_native_id ON sessions(id);
CREATE INDEX IF NOT EXISTS sessions_order ON sessions(updated_at DESC, agent, id);
CREATE TABLE IF NOT EXISTS prelaunch_baselines (
	session_ref TEXT PRIMARY KEY NOT NULL,
	repository_id TEXT NOT NULL,
	branch TEXT NOT NULL,
	git_head TEXT NOT NULL,
	working_tree_digest TEXT NOT NULL,
	working_tree_state TEXT NOT NULL,
	observed_at INTEGER NOT NULL,
	provenance TEXT NOT NULL,
	source_session_ref TEXT NOT NULL,
	capabilities_json TEXT NOT NULL,
	runtimes_json TEXT NOT NULL
);`

func openDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	closeOnError := func(err error) (*sql.DB, error) {
		_ = db.Close()
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return closeOnError(err)
	}
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = DELETE",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return closeOnError(err)
		}
	}
	var integrity string
	if err := db.QueryRow("PRAGMA quick_check").Scan(&integrity); err != nil {
		return closeOnError(err)
	}
	if integrity != "ok" {
		return closeOnError(fmt.Errorf("session-index integrity check failed: %s", integrity))
	}
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return closeOnError(err)
	}
	switch version {
	case 0:
		if _, err := db.Exec(createSchema); err != nil {
			return closeOnError(err)
		}
		if err := verifySchema(db); err != nil {
			return closeOnError(err)
		}
		if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)); err != nil {
			return closeOnError(err)
		}
	case SchemaVersion:
		if _, err := db.Exec(createSchema); err != nil {
			return closeOnError(err)
		}
		if err := verifySchema(db); err != nil {
			return closeOnError(err)
		}
	default:
		return closeOnError(fmt.Errorf("%w: found %d, expected %d", errIncompatibleSchema, version, SchemaVersion))
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return closeOnError(fmt.Errorf("protect session index: %w", err))
	}
	return db, nil
}

func verifySchema(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	required := map[string]bool{
		"key": false, "id": false, "agent": false, "source": false,
		"source_path": false, "source_mod_time": false, "source_size": false,
		"search_text": false, "project_fold": false, "workspace_fold": false,
		"branch_fold": false, "recorded_environment_json": false,
	}
	for rows.Next() {
		var columnID, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(
			&columnID,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			return err
		}
		if _, exists := required[name]; exists {
			required[name] = true
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("%w: missing column %s", errIncompatibleSchema, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	baselineRows, err := db.Query("PRAGMA table_info(prelaunch_baselines)")
	if err != nil {
		return err
	}
	defer func() { _ = baselineRows.Close() }()
	baselineRequired := map[string]bool{
		"session_ref": false, "repository_id": false, "branch": false,
		"git_head": false, "working_tree_digest": false,
		"working_tree_state": false, "observed_at": false,
		"provenance": false, "source_session_ref": false,
		"capabilities_json": false, "runtimes_json": false,
	}
	for baselineRows.Next() {
		var columnID, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := baselineRows.Scan(
			&columnID,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			return err
		}
		if _, exists := baselineRequired[name]; exists {
			baselineRequired[name] = true
		}
	}
	for name, found := range baselineRequired {
		if !found {
			return fmt.Errorf("%w: missing prelaunch baseline column %s", errIncompatibleSchema, name)
		}
	}
	return baselineRows.Err()
}

func ensurePrivateParent(path string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		return fmt.Errorf("protect session-index directory: %w", err)
	}
	return nil
}

func removeDatabaseFiles(path string) error {
	for _, candidate := range []string{path, path + "-journal", path + "-wal", path + "-shm"} {
		err := os.Remove(candidate)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove derived session index %s: %w", filepath.Base(candidate), err)
		}
	}
	return nil
}

func isRebuildableDatabaseError(err error) bool {
	if errors.Is(err, errIncompatibleSchema) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"database disk image is malformed",
		"file is not a database",
		"integrity check failed",
		"sqlite_corrupt",
		"sqlite_notadb",
		"incompatible session-index schema",
		"missing column",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func boolToDatabase(value bool) int {
	if value {
		return 1
	}
	return 0
}

func timeToDatabase(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixNano()
}

func timeFromDatabase(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}
