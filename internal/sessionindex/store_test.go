package sessionindex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
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
	_, err = incompatible.db.Exec("PRAGMA user_version = 99")
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
