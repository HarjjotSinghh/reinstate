package sessionindex

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/filelock"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

const (
	indexFileName              = "session-index-v2.sqlite"
	maxRecordedEnvironmentJSON = 1 << 20
	maxBaselineInventoryJSON   = 2 << 20
	maxFilesJSON               = 4 << 20
	maxStoredSourcePathRunes   = 32 << 10
	maximumBaselineFutureSkew  = 24 * time.Hour
	indexOpenLockTimeout       = 10 * time.Second
)

var errIncompatibleSchema = errors.New("incompatible session-index schema")

// ErrPrelaunchBaselineNotFound means no private Reinstate observation exists
// for the requested session reference.
var ErrPrelaunchBaselineNotFound = errors.New("prelaunch baseline not found")

var (
	ErrPrelaunchBaselineOlder    = errors.New("prelaunch baseline is older than the stored observation")
	ErrPrelaunchBaselineConflict = errors.New("prelaunch baseline conflicts at the same observation time")
	ErrIndexDataCorrupt          = errors.New("session-index derived data is invalid")
)

// ReplaceResult summarizes one atomic source replacement.
type ReplaceResult struct {
	Upserted  int `json:"upserted"`
	Unchanged int `json:"unchanged"`
	Deleted   int `json:"deleted"`
}

// Store owns the private derived SQLite index.
type Store struct {
	path string

	mu           sync.RWMutex
	db           *sql.DB
	lifetimeLock *filelock.Lock
}

// IndexPath returns the current private derived-index location below a
// Reinstate home. v2 is separate from the Phase 2 v1 file so downgrades cannot
// destroy non-reconstructible verified-resume baselines.
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
	lockContext, cancel := context.WithTimeout(context.Background(), indexOpenLockTimeout)
	defer cancel()
	lock, err := filelock.AcquireShared(lockContext, path+".lock")
	if err != nil {
		return nil, fmt.Errorf("lock session index: %w", err)
	}

	db, err := openDatabase(path)
	if err != nil {
		if !isRebuildableDatabaseError(err) {
			_ = lock.Close()
			return nil, err
		}
		_ = lock.Close()
		writer, writerErr := filelock.Acquire(lockContext, path+".write.lock")
		if writerErr != nil {
			return nil, fmt.Errorf("lock session index writer: %w", writerErr)
		}
		defer func() { _ = writer.Close() }()
		// Another process may have repaired the database while this opener
		// waited for the writer lock. Rejoin the lifetime shared lock and retry
		// before requesting destructive recovery.
		lock, err = filelock.AcquireShared(lockContext, path+".lock")
		if err != nil {
			return nil, fmt.Errorf("lock repaired session index: %w", err)
		}
		db, err = openDatabase(path)
		if err == nil {
			return &Store{path: path, db: db, lifetimeLock: lock}, nil
		}
		if !isRebuildableDatabaseError(err) {
			_ = lock.Close()
			return nil, err
		}
		_ = lock.Close()
		lock, err = filelock.Acquire(lockContext, path+".lock")
		if err != nil {
			return nil, fmt.Errorf("lock session index rebuild: %w", err)
		}
		// Another process may have rebuilt the file while this opener waited.
		db, err = openDatabase(path)
		if err == nil {
			shared, shareErr := replaceExclusiveWithShared(lockContext, path, lock)
			if shareErr != nil {
				_ = db.Close()
				return nil, shareErr
			}
			return &Store{path: path, db: db, lifetimeLock: shared}, nil
		}
		if !isRebuildableDatabaseError(err) {
			_ = lock.Close()
			return nil, err
		}
		if removeErr := removeDatabaseFiles(path); removeErr != nil {
			_ = lock.Close()
			return nil, errors.Join(err, removeErr)
		}
		db, err = openDatabase(path)
		if err != nil {
			_ = lock.Close()
			return nil, fmt.Errorf("rebuild session index: %w", err)
		}
		shared, shareErr := replaceExclusiveWithShared(lockContext, path, lock)
		if shareErr != nil {
			_ = db.Close()
			return nil, shareErr
		}
		lock = shared
	}
	return &Store{path: path, db: db, lifetimeLock: lock}, nil
}

func replaceExclusiveWithShared(ctx context.Context, path string, exclusive *filelock.Lock) (*filelock.Lock, error) {
	if err := exclusive.Close(); err != nil {
		return nil, fmt.Errorf("release exclusive session index lock: %w", err)
	}
	shared, err := filelock.AcquireShared(ctx, path+".lock")
	if err != nil {
		return nil, fmt.Errorf("restore shared session index lock: %w", err)
	}
	return shared, nil
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
		if s.lifetimeLock == nil {
			return nil
		}
		err := s.lifetimeLock.Close()
		s.lifetimeLock = nil
		return err
	}
	err := s.db.Close()
	s.db = nil
	if s.lifetimeLock != nil {
		err = errors.Join(err, s.lifetimeLock.Close())
		s.lifetimeLock = nil
	}
	return err
}

// Rebuild deletes only the derived index and creates an empty current schema.
func (s *Store) Rebuild(ctx context.Context) error {
	if s == nil {
		return errors.New("session-index store is nil")
	}
	writeLock, err := filelock.Acquire(ctx, s.path+".write.lock")
	if err != nil {
		return fmt.Errorf("lock session index writer: %w", err)
	}
	defer func() { _ = writeLock.Close() }()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.db == nil {
		return errors.New("session-index store is closed")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Baselines are linked to sessions with ON DELETE CASCADE. Clearing the
	// derived rows transactionally preserves the database identity, so other
	// live Store handles never become split-brain readers of an unlinked file.
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return err
	}
	return tx.Commit()
}

// ReplaceSource atomically upserts a complete successful scan and deletes
// records that disappeared from that source. Callers must not invoke it after a
// failed scan.
// ReplaceOption adjusts how a source's rows are replaced.
type ReplaceOption func(*replaceOptions)

type replaceOptions struct{ rewriteEveryRow bool }

// RewriteEveryRow disables the per-record shortcut that leaves a row alone when
// its source file has not moved.
//
// That shortcut compares only the source path, modification time and size, so
// it answers "has this file changed" — never "would this build read it the same
// way". When Reinstate's own readers change, the same untouched file parses
// into a different record, and without this the stored row survives the
// upgrade unchanged. A released build indexed Gemini sessions with no
// workspace; the reader was fixed; every existing index kept serving the old
// rows because the files on disk had not moved.
func RewriteEveryRow() ReplaceOption {
	return func(o *replaceOptions) { o.rewriteEveryRow = true }
}

func (s *Store) ReplaceSource(
	ctx context.Context,
	source string,
	records []Record,
	options ...ReplaceOption,
) (ReplaceResult, error) {
	var settings replaceOptions
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
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

	lock, err := filelock.Acquire(ctx, s.path+".write.lock")
	if err != nil {
		return result, fmt.Errorf("lock session index update: %w", err)
	}
	defer func() { _ = lock.Close() }()
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
		if exists && !settings.rewriteEveryRow &&
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
	if len(capabilitiesJSON) > maxBaselineInventoryJSON {
		return errors.New("prelaunch capability inventory exceeds safe encoded size")
	}
	runtimesJSON, err := json.Marshal(baseline.Runtimes)
	if err != nil {
		return fmt.Errorf("encode prelaunch runtimes: %w", err)
	}
	if len(runtimesJSON) > maxBaselineInventoryJSON {
		return errors.New("prelaunch runtime inventory exceeds safe encoded size")
	}
	if baseline.ObservedAt.Year() < 1970 || baseline.ObservedAt.Year() > 2262 {
		return errors.New("prelaunch observation time is outside the safe storage range")
	}
	if baseline.ObservedAt.After(time.Now().UTC().Add(maximumBaselineFutureSkew)) {
		return errors.New("prelaunch observation time is too far in the future")
	}
	observedAt := timeToDatabase(baseline.ObservedAt)
	canonicalObservedAt := timeFromDatabase(observedAt)
	if !canonicalObservedAt.Equal(baseline.ObservedAt) {
		return errors.New("prelaunch observation time is outside the lossless storage range")
	}
	// Compare the same representation that SQLite persists. This strips any
	// process-local monotonic reading and makes an identical retry idempotent.
	baseline.ObservedAt = canonicalObservedAt
	lock, err := filelock.Acquire(ctx, s.path+".write.lock")
	if err != nil {
		return fmt.Errorf("lock prelaunch baseline update: %w", err)
	}
	defer func() { _ = lock.Close() }()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("session-index store is closed")
	}
	var sessionExists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sessions WHERE key = ?`, baseline.SessionRef).Scan(&sessionExists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: baseline session is absent", ErrNotFound)
		}
		return err
	}
	existing, existingErr := getPrelaunchBaseline(s.db.QueryRowContext(ctx, baselineSelect+` WHERE session_ref = ?`, baseline.SessionRef))
	if existingErr == nil {
		switch {
		case baseline.ObservedAt.Before(existing.ObservedAt):
			return ErrPrelaunchBaselineOlder
		case baseline.ObservedAt.Equal(existing.ObservedAt):
			if reflect.DeepEqual(baseline, existing) {
				return nil
			}
			return ErrPrelaunchBaselineConflict
		}
	} else if !errors.Is(existingErr, sql.ErrNoRows) {
		return existingErr
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO prelaunch_baselines (
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
		runtimes_json = excluded.runtimes_json
	WHERE excluded.observed_at > prelaunch_baselines.observed_at`,
		baseline.SessionRef,
		baseline.RepositoryID,
		baseline.Branch,
		baseline.GitHead,
		baseline.WorkingTreeDigest,
		string(baseline.WorkingTreeState),
		observedAt,
		baseline.Provenance,
		baseline.SourceSessionRef,
		string(capabilitiesJSON),
		string(runtimesJSON),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrPrelaunchBaselineConflict
	}
	return nil
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
	baseline, err := getPrelaunchBaseline(s.db.QueryRowContext(ctx, baselineSelect+` WHERE session_ref = ?`, sessionRef))
	if errors.Is(err, sql.ErrNoRows) {
		return environment.PrelaunchBaseline{}, fmt.Errorf("%w: %s", ErrPrelaunchBaselineNotFound, sessionRef)
	}
	if err != nil {
		return environment.PrelaunchBaseline{}, err
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
	lock, err := filelock.Acquire(ctx, s.path+".write.lock")
	if err != nil {
		return fmt.Errorf("lock prelaunch baseline deletion: %w", err)
	}
	defer func() { _ = lock.Close() }()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("session-index store is closed")
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM prelaunch_baselines WHERE session_ref = ?`, sessionRef)
	return err
}

const baselineSelect = `SELECT
	session_ref, repository_id, branch, git_head, working_tree_digest,
	working_tree_state, observed_at, provenance, source_session_ref,
	substr(capabilities_json, 1, 2097153), substr(runtimes_json, 1, 2097153)
	FROM prelaunch_baselines`

func getPrelaunchBaseline(scanner rowScanner) (environment.PrelaunchBaseline, error) {
	var baseline environment.PrelaunchBaseline
	var workingTreeState string
	var observedAt int64
	var capabilitiesJSON, runtimesJSON string
	if err := scanner.Scan(
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
	); err != nil {
		return environment.PrelaunchBaseline{}, err
	}
	if len(capabilitiesJSON) > maxBaselineInventoryJSON || len(runtimesJSON) > maxBaselineInventoryJSON {
		return environment.PrelaunchBaseline{}, ErrIndexDataCorrupt
	}
	baseline.WorkingTreeState = environment.WorkingTreeState(workingTreeState)
	baseline.ObservedAt = timeFromDatabase(observedAt)
	if json.Unmarshal([]byte(capabilitiesJSON), &baseline.Capabilities) != nil ||
		json.Unmarshal([]byte(runtimesJSON), &baseline.Runtimes) != nil {
		return environment.PrelaunchBaseline{}, ErrIndexDataCorrupt
	}
	normalized, err := environment.NormalizePrelaunchBaseline(baseline)
	if err != nil {
		return environment.PrelaunchBaseline{}, ErrIndexDataCorrupt
	}
	return normalized, nil
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
	if len(recordedEnvironmentJSON) > maxRecordedEnvironmentJSON {
		return errors.New("recorded environment exceeds safe encoded size")
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
		return Record{}, ErrIndexDataCorrupt
	}
	record.UpdatedAt = timeFromDatabase(updatedAt)
	record.CanResume = canResume != 0
	record.CanFork = canFork != 0
	if len(filesJSON) > maxFilesJSON {
		return Record{}, ErrIndexDataCorrupt
	}
	if err := json.Unmarshal([]byte(filesJSON), &record.Files); err != nil {
		return Record{}, ErrIndexDataCorrupt
	}
	if len(recordedEnvironmentJSON) > maxRecordedEnvironmentJSON {
		return Record{}, ErrIndexDataCorrupt
	}
	if err := json.Unmarshal([]byte(recordedEnvironmentJSON), &record.RecordedEnvironment); err != nil {
		return Record{}, ErrIndexDataCorrupt
	}
	record.RecordedEnvironment, err = environment.NormalizeRecordedEnvironment(record.RecordedEnvironment)
	if err != nil {
		return Record{}, ErrIndexDataCorrupt
	}
	if err := validateStoredRecord(record); err != nil {
		return Record{}, ErrIndexDataCorrupt
	}
	return record, nil
}

func validateStoredRecord(record Record) error {
	if utf8.RuneCountInString(record.SourcePath) > maxStoredSourcePathRunes ||
		len(record.SearchText) > MaxSearchTextBytes || !validStoredSearchText(record.SearchText) {
		return ErrIndexDataCorrupt
	}
	searchText := record.SearchText
	record.SearchText = ""
	normalized, err := NormalizeRecord(record)
	if err != nil {
		return err
	}
	// Search text includes bounded prompt metadata not otherwise retained, so
	// validate it independently above. Every other stored public field must
	// already be canonical; silently repairing attacker-controlled derived rows
	// would make later safety comparisons ambiguous.
	normalized.SearchText = searchText
	original := recordWithSearch(record, searchText)
	if normalized.Key != original.Key || normalized.ID != original.ID || normalized.Agent != original.Agent {
		return errors.New("non-canonical identity")
	}
	if normalized.Title != original.Title || normalized.Project != original.Project ||
		normalized.Workspace != original.Workspace || normalized.Branch != original.Branch ||
		normalized.PromptPreview != original.PromptPreview || normalized.ReadOnlyReason != original.ReadOnlyReason {
		return errors.New("non-canonical metadata")
	}
	if !normalized.UpdatedAt.Equal(original.UpdatedAt) || normalized.SizeBytes != original.SizeBytes ||
		normalized.MessageCount != original.MessageCount || normalized.CanResume != original.CanResume ||
		normalized.CanFork != original.CanFork {
		return errors.New("non-canonical numeric state")
	}
	if !reflect.DeepEqual(normalized.Files, original.Files) {
		return errors.New("non-canonical file references")
	}
	if !reflect.DeepEqual(normalized.RecordedEnvironment, original.RecordedEnvironment) {
		return errors.New("non-canonical recorded environment")
	}
	if normalized.SourcePath != original.SourcePath || normalized.SourceModTime != original.SourceModTime ||
		normalized.SourceSize != original.SourceSize || normalized.SearchText != original.SearchText {
		return errors.New("non-canonical source state")
	}
	return nil
}

func validStoredSearchText(value string) bool {
	for _, line := range strings.Split(value, "\n") {
		if SafeText(line, 0) != line {
			return false
		}
	}
	return true
}

func recordWithSearch(record Record, searchText string) Record {
	record.SearchText = searchText
	return record
}

const recordSelect = `SELECT
	substr(key, 1, 4162), substr(id, 1, 4097), substr(agent, 1, 65),
	substr(title, 1, 513), substr(project, 1, 4097), substr(workspace, 1, 4097),
	substr(branch, 1, 4097), updated_at, size_bytes, message_count,
	substr(prompt_preview, 1, 161), substr(files_json, 1, 4194305), can_resume, can_fork,
	substr(read_only_reason, 1, 513), substr(recorded_environment_json, 1, 1048577),
	substr(source_path, 1, 32769), source_mod_time, source_size,
	substr(search_text, 1, 262145)
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
	session_ref TEXT PRIMARY KEY NOT NULL REFERENCES sessions(key) ON DELETE CASCADE,
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
);
CREATE TABLE IF NOT EXISTS source_state (
	source TEXT PRIMARY KEY NOT NULL,
	fingerprint TEXT NOT NULL,
	observed_at INTEGER NOT NULL
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
	foreignKeyRows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return closeOnError(err)
	}
	hasForeignKeyViolation := foreignKeyRows.Next()
	foreignKeyErr := foreignKeyRows.Err()
	closeForeignKeyErr := foreignKeyRows.Close()
	if foreignKeyErr != nil || closeForeignKeyErr != nil {
		return closeOnError(errors.Join(foreignKeyErr, closeForeignKeyErr))
	}
	if hasForeignKeyViolation {
		return closeOnError(fmt.Errorf("%w: foreign key violation", errIncompatibleSchema))
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
	if err := fsx.ProtectOwnerOnly(path, false); err != nil {
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
	required := map[string]string{
		"key": "TEXT", "id": "TEXT", "agent": "TEXT", "source": "TEXT",
		"source_path": "TEXT", "source_mod_time": "INTEGER", "source_size": "INTEGER",
		"search_text": "TEXT", "project_fold": "TEXT", "workspace_fold": "TEXT",
		"branch_fold": "TEXT", "recorded_environment_json": "TEXT",
	}
	foundRequired := make(map[string]bool, len(required))
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
		if expectedType, exists := required[name]; exists {
			if strings.ToUpper(columnType) != expectedType || notNull != 1 || name == "key" && primaryKey != 1 {
				return fmt.Errorf("%w: invalid column contract", errIncompatibleSchema)
			}
			foundRequired[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for name := range required {
		found := foundRequired[name]
		if !found {
			return fmt.Errorf("%w: missing column %s", errIncompatibleSchema, name)
		}
	}
	baselineRows, err := db.Query("PRAGMA table_info(prelaunch_baselines)")
	if err != nil {
		return err
	}
	defer func() { _ = baselineRows.Close() }()
	baselineRequired := map[string]string{
		"session_ref": "TEXT", "repository_id": "TEXT", "branch": "TEXT",
		"git_head": "TEXT", "working_tree_digest": "TEXT",
		"working_tree_state": "TEXT", "observed_at": "INTEGER",
		"provenance": "TEXT", "source_session_ref": "TEXT",
		"capabilities_json": "TEXT", "runtimes_json": "TEXT",
	}
	baselineFound := make(map[string]bool, len(baselineRequired))
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
		if expectedType, exists := baselineRequired[name]; exists {
			if strings.ToUpper(columnType) != expectedType || notNull != 1 || name == "session_ref" && primaryKey != 1 {
				return fmt.Errorf("%w: invalid prelaunch baseline column contract", errIncompatibleSchema)
			}
			baselineFound[name] = true
		}
	}
	if err := baselineRows.Err(); err != nil {
		return err
	}
	if err := baselineRows.Close(); err != nil {
		return err
	}
	for name := range baselineRequired {
		found := baselineFound[name]
		if !found {
			return fmt.Errorf("%w: missing prelaunch baseline column %s", errIncompatibleSchema, name)
		}
	}
	foreignRows, err := db.Query("PRAGMA foreign_key_list(prelaunch_baselines)")
	if err != nil {
		return err
	}
	defer func() { _ = foreignRows.Close() }()
	validForeignKey := false
	for foreignRows.Next() {
		var id, sequence int
		var table, from, to, onUpdate, onDelete, match string
		if err := foreignRows.Scan(&id, &sequence, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return err
		}
		if table == "sessions" && from == "session_ref" && to == "key" && strings.EqualFold(onDelete, "CASCADE") {
			validForeignKey = true
		}
	}
	if err := foreignRows.Err(); err != nil {
		return err
	}
	if !validForeignKey {
		return fmt.Errorf("%w: missing prelaunch baseline cascade", errIncompatibleSchema)
	}
	return nil
}

func ensurePrivateParent(path string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := fsx.ProtectOwnerOnly(parent, true); err != nil {
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

// SourceFingerprint returns the fingerprint recorded the last time this source
// was scanned, or "" when the source has never been scanned.
func (s *Store) SourceFingerprint(ctx context.Context, source string) (string, error) {
	var fingerprint string
	err := s.db.QueryRowContext(ctx,
		`SELECT fingerprint FROM source_state WHERE source = ?`, source).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return fingerprint, nil
}

// SetSourceFingerprint records the fingerprint that produced the rows now
// stored for this source. It is written only after a scan has been persisted,
// so a failed scan can never mark a source as up to date.
func (s *Store) SetSourceFingerprint(ctx context.Context, source, fingerprint string, observedAt int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO source_state (source, fingerprint, observed_at) VALUES (?, ?, ?)
		 ON CONFLICT(source) DO UPDATE SET fingerprint = excluded.fingerprint,
		                                   observed_at = excluded.observed_at`,
		source, fingerprint, observedAt)
	return err
}

// CountSource reports how many rows this source currently has stored.
func (s *Store) CountSource(ctx context.Context, source string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE source = ?`, source).Scan(&count)
	return count, err
}
