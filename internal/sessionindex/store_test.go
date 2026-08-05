package sessionindex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

func TestStoreReplaceSearchResolveDeleteAndPermissions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "private", indexFileName)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	store.mu.RLock()
	var schemaVersion int
	err = store.db.QueryRow("PRAGMA user_version").Scan(&schemaVersion)
	store.mu.RUnlock()
	if err != nil {
		t.Fatal(err)
	}
	if schemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", schemaVersion, SchemaVersion)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("index mode = %o", got)
		}
		parentInfo, err := os.Stat(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		if got := parentInfo.Mode().Perm(); got != 0o700 {
			t.Fatalf("index directory mode = %o", got)
		}
	}

	base := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	claude := testRecord(AgentClaude, "same", base.Add(time.Minute), "/work/a", 1)
	claude.SearchText = "Stripe %_ retry webhook CAFÉ"
	claude.Project = "PAYMÉNTS"
	claude.Branch = "FÉATURE/retry"
	claude.Files = []string{"internal/ÜBER-webhook.go"}
	claude.RecordedEnvironment = environment.RecordedEnvironment{
		RepositoryID: environment.RecordedField{
			Value:      "https://fixture:secret@github.com/example/payments.git?token=secret",
			Provenance: "codex.session_meta.git.repository_url",
		},
		Branch: environment.RecordedField{
			Value:      "FÉATURE/retry",
			Provenance: "codex.session_meta.git.branch",
		},
		GitHead: environment.RecordedField{
			Value:      "0123456789abcdef0123456789abcdef01234567",
			Provenance: "codex.session_meta.git.commit_hash",
		},
	}
	codex := testRecord(AgentCodex, "same", base, "/work/b", 1)
	second := testRecord(AgentClaude, "second", base.Add(2*time.Minute), "/work/a", 2)
	second.CanResume = false
	second.CanFork = false
	second.ReadOnlyReason = "fixture read only"

	replaced, err := store.ReplaceSource(ctx, AgentClaude, []Record{claude, second})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Upserted != 2 || replaced.Unchanged != 0 || replaced.Deleted != 0 {
		t.Fatalf("replace result = %+v", replaced)
	}
	if _, err := store.ReplaceSource(ctx, AgentCodex, []Record{codex}); err != nil {
		t.Fatal(err)
	}

	results, err := store.Search(ctx, Filter{Query: "stripe %_", Project: "pay", Branch: "RETRY", File: "webhook"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Reference() != "claude:same" {
		t.Fatalf("filtered results = %+v", results)
	}
	results, err = store.Search(ctx, Filter{
		Query:   "café",
		Project: "payménts",
		Branch:  "féature",
		File:    "über",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Reference() != "claude:same" {
		t.Fatalf("Unicode case-insensitive results = %+v", results)
	}
	results, err = store.Search(ctx, Filter{Query: "deliberate-zero-match"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("query-only zero match returned %+v", results)
	}

	all, err := store.All(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := joinReferences(all); got != "claude:second,claude:same,codex:same" {
		t.Fatalf("order = %s", got)
	}

	resolved, err := store.Resolve(ctx, "claude:same")
	if err != nil || resolved.Agent != AgentClaude {
		t.Fatalf("qualified resolve = %+v, %v", resolved, err)
	}
	if resolved.RecordedEnvironment.RepositoryID.Value != environment.NormalizeRepositoryID("https://github.com/example/payments.git") ||
		resolved.RecordedEnvironment.GitHead.Value != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("recorded environment round trip = %+v", resolved.RecordedEnvironment)
	}
	_, err = store.Resolve(ctx, "same")
	var ambiguous *AmbiguousReferenceError
	if !errors.As(err, &ambiguous) || !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("bare resolve error = %T %v", err, err)
	}
	if got := strings.Join(ambiguous.Matches, ","); got != "claude:same,codex:same" {
		t.Fatalf("ambiguous matches = %q", got)
	}
	if _, err := store.Resolve(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing resolve error = %v", err)
	}

	last, err := store.Last(ctx, Filter{ResumableOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if last.Reference() != "claude:same" {
		t.Fatalf("last resumable = %s", last.Reference())
	}

	unchangedRecord := claude
	unchangedRecord.Title = "parser output must be reused for same fingerprint"
	replaced, err = store.ReplaceSource(ctx, AgentClaude, []Record{unchangedRecord})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Unchanged != 1 || replaced.Deleted != 1 || replaced.Upserted != 0 {
		t.Fatalf("unchanged replacement = %+v", replaced)
	}
	resolved, err = store.Resolve(ctx, "claude:same")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Title == unchangedRecord.Title {
		t.Fatal("same source fingerprint unexpectedly replaced derived record")
	}

	unchangedRecord.SourceModTime++
	replaced, err = store.ReplaceSource(ctx, AgentClaude, []Record{unchangedRecord})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Upserted != 1 || replaced.Unchanged != 0 {
		t.Fatalf("changed replacement = %+v", replaced)
	}
	resolved, err = store.Resolve(ctx, "claude:same")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Title != unchangedRecord.Title {
		t.Fatalf("updated title = %q", resolved.Title)
	}
}

func TestStoreRebuildsCorruptAndIncompatibleDerivedState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	corruptPath := filepath.Join(root, "corrupt", indexFileName)
	if err := os.MkdirAll(filepath.Dir(corruptPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corruptPath, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := Open(corruptPath)
	if err != nil {
		t.Fatal(err)
	}
	records, err := corrupt.All(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("rebuilt corrupt store has %d records", len(records))
	}
	if err := corrupt.Close(); err != nil {
		t.Fatal(err)
	}

	incompatiblePath := filepath.Join(root, "incompatible", indexFileName)
	incompatible, err := Open(incompatiblePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := incompatible.ReplaceSource(ctx, AgentClaude, []Record{
		testRecord(AgentClaude, "old", time.Now(), "/old", 1),
	}); err != nil {
		t.Fatal(err)
	}
	incompatible.mu.RLock()
	// Schema v1 is disposable derived state. Opening it under v2 must rebuild
	// rather than attempt an in-place migration or retain stale records.
	_, err = incompatible.db.Exec("PRAGMA user_version = 1")
	incompatible.mu.RUnlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := incompatible.Close(); err != nil {
		t.Fatal(err)
	}
	incompatible, err = Open(incompatiblePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = incompatible.Close() })
	records, err = incompatible.All(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("rebuilt incompatible store has %d records", len(records))
	}
}

func TestConcurrentOpenersConvergeOnOneRepairedCorruptIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.sqlite")
	if err := os.WriteFile(path, []byte("controlled corrupt derived index"), 0o600); err != nil {
		t.Fatal(err)
	}
	type openResult struct {
		store *Store
		err   error
	}
	start := make(chan struct{})
	results := make(chan openResult, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			store, err := Open(path)
			results <- openResult{store: store, err: err}
		}()
	}
	close(start)
	stores := make([]*Store, 0, 2)
	var openErrors []error
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			openErrors = append(openErrors, result.err)
			continue
		}
		stores = append(stores, result.store)
	}
	if len(openErrors) != 0 {
		for _, store := range stores {
			_ = store.Close()
		}
		t.Fatalf("concurrent Open() errors = %v", openErrors)
	}
	for _, store := range stores {
		if records, err := store.All(context.Background(), 10); err != nil {
			t.Fatalf("repaired store unusable: %v", err)
		} else if len(records) != 0 {
			t.Fatalf("repaired store records = %d", len(records))
		}
	}

	record := testRecord(AgentClaude, "shared-repair", time.Now(), "/session.jsonl", 1)
	if _, err := stores[0].ReplaceSource(context.Background(), AgentClaude, []Record{record}); err != nil {
		t.Fatalf("write through first repaired store: %v", err)
	}
	if got, err := stores[1].Resolve(context.Background(), record.Reference()); err != nil || got.Reference() != record.Reference() {
		t.Fatalf("second repaired store did not share current state: %+v, %v", got, err)
	}
	for _, store := range stores {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen repaired index: %v", err)
	}
	defer func() { _ = reopened.Close() }()
	if got, err := reopened.Resolve(context.Background(), record.Reference()); err != nil || got.Reference() != record.Reference() {
		t.Fatalf("reopened repaired index lost shared state: %+v, %v", got, err)
	}
}

func TestPrelaunchBaselinePersistsAcrossVendorSourceAppend(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	record := testRecord(AgentCodex, "prelaunch", time.Now(), "/session.jsonl", 1)
	if _, err := store.ReplaceSource(ctx, AgentCodex, []Record{record}); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 5, 1, 2, 3, 4, time.UTC)
	baseline := environment.PrelaunchBaseline{
		SessionRef:        record.Reference(),
		RepositoryID:      "git@github.com:example/reinstate.git",
		Branch:            "phase-three",
		GitHead:           "0123456789abcdef0123456789abcdef01234567",
		WorkingTreeDigest: strings.Repeat("a", 64),
		WorkingTreeState:  environment.WorkingTreeModified,
		ObservedAt:        observedAt,
		Provenance:        environment.PrelaunchObservedProvenance,
		SourceSessionRef:  "codex:source-session",
		Capabilities: []environment.Capability{
			{Agent: "codex", Kind: "mcp", Name: "github", Scope: "project", State: "enabled", Provenance: environment.PrelaunchObservedProvenance},
		},
		Runtimes: []environment.Runtime{
			{Name: "go", Declared: "1.25.12", Version: "1.25.12", SourceKind: "go_mod", Provenance: environment.PrelaunchObservedProvenance},
		},
	}
	if err := store.PutPrelaunchBaseline(ctx, baseline); err != nil {
		t.Fatal(err)
	}

	// A successful native run appends to the vendor source and therefore changes
	// its fingerprint. That update must not erase the independently observed
	// prelaunch baseline.
	record.SourceModTime++
	record.SourceSize++
	record.SizeBytes++
	if _, err := store.ReplaceSource(ctx, AgentCodex, []Record{record}); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetPrelaunchBaseline(ctx, record.Reference())
	if err != nil {
		t.Fatal(err)
	}
	if got.RepositoryID != environment.NormalizeRepositoryID("git@github.com:example/reinstate.git") ||
		got.WorkingTreeState != environment.WorkingTreeModified ||
		got.WorkingTreeDigest != "sha256:"+strings.Repeat("a", 64) ||
		!got.ObservedAt.Equal(observedAt) ||
		got.SourceSessionRef != "codex:source-session" ||
		len(got.Capabilities) != 1 || got.Capabilities[0].Name != "github" ||
		len(got.Runtimes) != 1 || got.Runtimes[0].Name != "go" || got.Runtimes[0].Declared != "1.25.12" {
		t.Fatalf("prelaunch baseline = %+v", got)
	}

	if err := store.DeletePrelaunchBaseline(ctx, record.Reference()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPrelaunchBaseline(ctx, record.Reference()); !errors.Is(err, ErrPrelaunchBaselineNotFound) {
		t.Fatalf("deleted baseline error = %v", err)
	}
	baseline.Provenance = "vendor_claim"
	if err := store.PutPrelaunchBaseline(ctx, baseline); err == nil {
		t.Fatal("untrusted prelaunch provenance unexpectedly accepted")
	}
}

func TestPrelaunchBaselineIsMonotonicAndCascadesWithSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	record := testRecord(AgentClaude, "monotonic", time.Now(), "/session.jsonl", 1)
	if _, err := store.ReplaceSource(ctx, AgentClaude, []Record{record}); err != nil {
		t.Fatal(err)
	}
	base := environment.PrelaunchBaseline{
		SessionRef: record.Reference(), WorkingTreeState: environment.WorkingTreeUnavailable,
		ObservedAt: time.Date(2026, 8, 5, 1, 2, 3, 0, time.UTC),
		Provenance: environment.PrelaunchObservedProvenance, SourceSessionRef: record.Reference(),
	}
	newer := base
	newer.ObservedAt = base.ObservedAt.Add(time.Minute)
	newer.Branch = "newer"
	if err := store.PutPrelaunchBaseline(ctx, newer); err != nil {
		t.Fatal(err)
	}
	older := base
	older.Branch = "older"
	if err := store.PutPrelaunchBaseline(ctx, older); !errors.Is(err, ErrPrelaunchBaselineOlder) {
		t.Fatalf("older baseline error = %v", err)
	}
	equalConflict := newer
	equalConflict.Branch = "conflict"
	if err := store.PutPrelaunchBaseline(ctx, equalConflict); !errors.Is(err, ErrPrelaunchBaselineConflict) {
		t.Fatalf("equal conflicting baseline error = %v", err)
	}
	if err := store.PutPrelaunchBaseline(ctx, newer); err != nil {
		t.Fatalf("idempotent baseline error = %v", err)
	}
	got, err := store.GetPrelaunchBaseline(ctx, record.Reference())
	if err != nil || got.Branch != "newer" {
		t.Fatalf("stored baseline = %+v, %v", got, err)
	}
	if _, err := store.ReplaceSource(ctx, AgentClaude, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPrelaunchBaseline(ctx, record.Reference()); !errors.Is(err, ErrPrelaunchBaselineNotFound) {
		t.Fatalf("orphan baseline survived deletion: %v", err)
	}
	if err := store.PutPrelaunchBaseline(ctx, newer); !errors.Is(err, ErrNotFound) {
		t.Fatalf("baseline for absent session error = %v", err)
	}
}

func TestIndependentStoresSerializeConcurrentWriters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "shared.sqlite")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	stores := []*Store{first, second}
	var wait sync.WaitGroup
	errorsFound := make(chan error, 40)
	for index := 0; index < 40; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			agent := AgentClaude
			if index%2 == 1 {
				agent = AgentCodex
			}
			record := testRecord(agent, fmt.Sprintf("concurrent-%d", index), time.Unix(int64(index+1), 0), fmt.Sprintf("/%d", index), int64(index+1))
			_, writeErr := stores[index%len(stores)].ReplaceSource(ctx, agent, []Record{record})
			if writeErr != nil {
				errorsFound <- writeErr
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for writeErr := range errorsFound {
		t.Fatalf("concurrent store write failed: %v", writeErr)
	}
}

func TestCorruptBaselineJSONIsBoundedAndPathFree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	record := testRecord(AgentCodex, "corrupt", time.Now(), "/session.jsonl", 1)
	if _, err := store.ReplaceSource(ctx, AgentCodex, []Record{record}); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	_, err = store.db.ExecContext(ctx, `INSERT INTO prelaunch_baselines (
		session_ref, repository_id, branch, git_head, working_tree_digest,
		working_tree_state, observed_at, provenance, source_session_ref,
		capabilities_json, runtimes_json
	) VALUES (?, '', '', '', '', 'unavailable', ?, ?, ?, ?, '[]')`,
		record.Reference(), time.Now().UnixNano(), environment.PrelaunchObservedProvenance,
		record.Reference(), strings.Repeat("x", maxBaselineInventoryJSON+100))
	store.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPrelaunchBaseline(ctx, record.Reference()); !errors.Is(err, ErrIndexDataCorrupt) || strings.Contains(err.Error(), "xxx") {
		t.Fatalf("corrupt baseline error = %v", err)
	}
}

func TestOpenRebuildsDerivedIndexWithOrphanedPrelaunchBaseline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "index.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	record := testRecord(AgentCodex, "orphan", time.Now(), "/session.jsonl", 1)
	if _, err := store.ReplaceSource(ctx, AgentCodex, []Record{record}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutPrelaunchBaseline(ctx, environment.PrelaunchBaseline{
		SessionRef: record.Reference(), SourceSessionRef: record.Reference(),
		WorkingTreeState: environment.WorkingTreeUnavailable,
		ObservedAt:       time.Now(),
		Provenance:       environment.PrelaunchObservedProvenance,
	}); err != nil {
		t.Fatal(err)
	}

	store.mu.Lock()
	_, disableErr := store.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	_, deleteErr := store.db.ExecContext(ctx, `DELETE FROM sessions WHERE key = ?`, record.Reference())
	var orphanCount int
	countErr := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prelaunch_baselines`).Scan(&orphanCount)
	store.mu.Unlock()
	if disableErr != nil || deleteErr != nil || countErr != nil {
		t.Fatalf("manufacture orphan: disable=%v delete=%v count=%v", disableErr, deleteErr, countErr)
	}
	if orphanCount != 1 {
		t.Fatalf("orphan baseline count = %d, want 1", orphanCount)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	rebuilt, err := Open(path)
	if err != nil {
		t.Fatalf("Open did not rebuild orphaned derived state: %v", err)
	}
	defer func() { _ = rebuilt.Close() }()
	if records, err := rebuilt.All(ctx, 10); err != nil {
		t.Fatal(err)
	} else if len(records) != 0 {
		t.Fatalf("records after orphan rebuild = %d", len(records))
	}
	if _, err := rebuilt.GetPrelaunchBaseline(ctx, record.Reference()); !errors.Is(err, ErrPrelaunchBaselineNotFound) {
		t.Fatalf("baseline survived orphan rebuild: %v", err)
	}
}

func TestStoreTamperedRowsReturnFixedCorruptionErrorWithoutLeaking(t *testing.T) {
	t.Parallel()

	sentinel := "PRIVATE-CONTROLLED-ROW-SENTINEL"
	tests := []struct {
		name   string
		column string
		value  func() string
	}{
		{
			name: "oversized files JSON", column: "files_json",
			value: func() string { return strings.Repeat(sentinel, maxFilesJSON/len(sentinel)+2) },
		},
		{
			name: "malformed files JSON", column: "files_json",
			value: func() string { return `{"` + sentinel },
		},
		{
			name: "oversized scalar", column: "title",
			value: func() string { return sentinel + strings.Repeat("x", maxTitleRunes+2) },
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			record := testRecord(AgentClaude, "tampered", time.Now(), "/session.jsonl", 1)
			if _, err := store.ReplaceSource(ctx, AgentClaude, []Record{record}); err != nil {
				t.Fatal(err)
			}

			store.mu.Lock()
			_, updateErr := store.db.ExecContext(ctx, `UPDATE sessions SET `+test.column+` = ? WHERE key = ?`, test.value(), record.Reference())
			store.mu.Unlock()
			if updateErr != nil {
				t.Fatal(updateErr)
			}
			_, err = store.All(ctx, 10)
			if !errors.Is(err, ErrIndexDataCorrupt) || err.Error() != ErrIndexDataCorrupt.Error() {
				t.Fatalf("tampered row error = %v", err)
			}
			if strings.Contains(err.Error(), sentinel) {
				t.Fatalf("tampered row error leaked injected text: %v", err)
			}
		})
	}
}

func TestStoreExplicitRebuildAndClosedErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReplaceSource(ctx, AgentClaude, []Record{
		testRecord(AgentClaude, "one", time.Now(), "/one", 1),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	records, err := store.All(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("records after rebuild = %d", len(records))
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.All(ctx, 10); err == nil {
		t.Fatal("query on closed store unexpectedly succeeded")
	}
}

func TestTwoIndependentStoresCanRebuildConcurrentlyAndRemainUsable(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "index.sqlite")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, store := range []*Store{first, second} {
		store := store
		go func() {
			<-start
			results <- store.Rebuild(ctx)
		}()
	}
	close(start)
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Errorf("concurrent Rebuild() error = %v", err)
		}
	}

	for name, store := range map[string]*Store{"first": first, "second": second} {
		records, err := store.All(context.Background(), 10)
		if err != nil {
			t.Errorf("%s store unusable after concurrent rebuild: %v", name, err)
			continue
		}
		if len(records) != 0 {
			t.Errorf("%s store records after concurrent rebuild = %d", name, len(records))
		}
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen after concurrent rebuild: %v", err)
	}
	defer func() { _ = reopened.Close() }()
	if records, err := reopened.All(context.Background(), 10); err != nil {
		t.Fatalf("reopened store unusable: %v", err)
	} else if len(records) != 0 {
		t.Fatalf("reopened store records after concurrent rebuild = %d", len(records))
	}
}

func TestStoreSerializesConcurrentRefreshAndSearch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var wait sync.WaitGroup
	errs := make(chan error, 16)
	for worker := 0; worker < 4; worker++ {
		worker := worker
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := 0; iteration < 10; iteration++ {
				record := testRecord(
					AgentClaude,
					"shared",
					time.Unix(int64(iteration), 0),
					"/shared",
					int64(worker*100+iteration),
				)
				if _, err := store.ReplaceSource(ctx, AgentClaude, []Record{record}); err != nil {
					errs <- err
					return
				}
				if _, err := store.Search(ctx, Filter{Query: "shared"}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func testRecord(agent, id string, updated time.Time, sourcePath string, fingerprint int64) Record {
	canWrite := agent == AgentClaude || agent == AgentCodex
	return Record{
		ID:             id,
		Agent:          agent,
		Title:          id,
		Project:        "project",
		Workspace:      "/workspace",
		UpdatedAt:      updated,
		SizeBytes:      10,
		MessageCount:   1,
		PromptPreview:  id + " preview",
		CanResume:      canWrite,
		CanFork:        canWrite,
		ReadOnlyReason: "",
		SourcePath:     sourcePath,
		SourceModTime:  fingerprint,
		SourceSize:     10,
		SearchText:     id + " prompt",
	}
}

func joinReferences(records []Record) string {
	refs := make([]string, 0, len(records))
	for _, record := range records {
		refs = append(refs, record.Reference())
	}
	return strings.Join(refs, ",")
}
