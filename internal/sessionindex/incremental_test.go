package sessionindex

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// fingerprintSource is a source that can summarise itself, so an unchanged
// refresh may skip it.
type fingerprintSource struct {
	fakeSource
	digest string
	usable bool
	// fingerprintErr is the error Fingerprint returns. It is deliberately not
	// named err: fakeSource.err is the error Scan returns, and an embedded
	// field of the same name silently shadows it.
	fingerprintErr error
	printCall      int
}

func (s *fingerprintSource) Fingerprint(context.Context) (string, bool, error) {
	s.printCall++
	return s.digest, s.usable, s.fingerprintErr
}

func newFingerprintSource(name, digest string, records ...Record) *fingerprintSource {
	return &fingerprintSource{
		fakeSource: fakeSource{name: name, result: ScanResult{Records: records}},
		digest:     digest,
		usable:     true,
	}
}

func openIndex(t *testing.T, sources ...Source) (*Index, *Store) {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	index, err := NewIndex(store, sources...)
	if err != nil {
		t.Fatal(err)
	}
	return index, store
}

func TestRefreshSkipsSourceWhoseFingerprintIsUnchanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src := newFingerprintSource(AgentClaude, "digest-a",
		testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1))
	index, _ := openIndex(t, src)

	first, err := index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if src.scanCall != 1 {
		t.Fatalf("cold refresh scan calls = %d, want 1", src.scanCall)
	}
	if got := sourceStatus(t, first, AgentClaude).Records; got != 1 {
		t.Fatalf("cold refresh records = %d, want 1", got)
	}

	second, err := index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if src.scanCall != 1 {
		t.Fatalf("unchanged refresh rescanned: scan calls = %d, want 1", src.scanCall)
	}
	// A skipped source must still report the rows it owns, otherwise the
	// refresh summary would read as though its sessions had disappeared.
	status := sourceStatus(t, second, AgentClaude)
	if status.Records != 1 || status.Unchanged != 1 {
		t.Fatalf("skipped source reported records=%d unchanged=%d, want 1 and 1",
			status.Records, status.Unchanged)
	}

	// The rows themselves must survive the skip.
	records, err := index.Store().Search(ctx, Filter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("rows after skipped refresh = %d, want 1", len(records))
	}
}

func TestRefreshRescansWhenFingerprintChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src := newFingerprintSource(AgentClaude, "digest-a",
		testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1))
	index, _ := openIndex(t, src)

	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}

	// The source changed on disk: a new digest and a second session.
	src.digest = "digest-b"
	src.result = ScanResult{Records: []Record{
		testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1),
		testRecord(AgentClaude, "two", time.Unix(2, 0), "/two", 1),
	}}

	result, err := index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if src.scanCall != 2 {
		t.Fatalf("changed source scan calls = %d, want 2", src.scanCall)
	}
	if got := sourceStatus(t, result, AgentClaude).Records; got != 2 {
		t.Fatalf("changed refresh records = %d, want 2", got)
	}

	// And a third refresh with the new digest settles back to skipping.
	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if src.scanCall != 2 {
		t.Fatalf("settled refresh rescanned: scan calls = %d, want 2", src.scanCall)
	}
}

func TestRefreshAlwaysScansSourceWithoutUsableFingerprint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	unusable := newFingerprintSource(AgentCodex, "digest-a",
		testRecord(AgentCodex, "one", time.Unix(1, 0), "/one", 1))
	unusable.usable = false
	failing := newFingerprintSource(AgentClaude, "digest-b",
		testRecord(AgentClaude, "two", time.Unix(2, 0), "/two", 1))
	failing.fingerprintErr = errors.New("cannot stat root")
	index, _ := openIndex(t, unusable, failing)

	for range 3 {
		if _, err := index.Refresh(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if unusable.scanCall != 3 {
		t.Fatalf("source with no usable fingerprint: scan calls = %d, want 3", unusable.scanCall)
	}
	if failing.scanCall != 3 {
		t.Fatalf("source whose fingerprint errored: scan calls = %d, want 3", failing.scanCall)
	}
}

// A scan that fails must never leave the source marked as up to date, or the
// next refresh would skip it and the failure would become permanent.
func TestFailedScanDoesNotRecordFingerprint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src := newFingerprintSource(AgentClaude, "digest-a",
		testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1))
	src.err = errors.New("transient read failure") // fakeSource.err: the scan fails
	index, store := openIndex(t, src)

	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	stored, err := store.SourceFingerprint(ctx, AgentClaude)
	if err != nil {
		t.Fatal(err)
	}
	if stored != "" {
		t.Fatalf("failed scan stored fingerprint %q, want none", stored)
	}

	// Recovery: the next refresh scans again and now succeeds.
	src.err = nil
	result, err := index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if src.scanCall != 2 {
		t.Fatalf("scan calls after failure = %d, want 2", src.scanCall)
	}
	if got := sourceStatus(t, result, AgentClaude).Records; got != 1 {
		t.Fatalf("records after recovery = %d, want 1", got)
	}
}

func sourceStatus(t *testing.T, result RefreshResult, name string) SourceRefresh {
	t.Helper()
	for _, s := range result.Sources {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("refresh result has no source %q", name)
	return SourceRefresh{}
}

// TestUpgradedReaderRescansUnchangedSource covers what happens when Reinstate
// itself changes but the agent's files do not.
//
// A source fingerprint answers "have these files changed". It cannot answer
// "would this build read them the same way", and the answer differs every time
// a reader is fixed. Binding the stored fingerprint to the reader is what makes
// an upgrade re-read; without it an existing index stays frozen and a reader
// fix is invisible until the user's agent happens to write a new session.
//
// This is not hypothetical: a released build indexed Gemini sessions with no
// workspace, the reader was fixed, and every existing index kept serving the
// old rows because the files on disk had not moved.
func TestUpgradedReaderRescansUnchangedSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "index.sqlite")

	// The old build reads one session and records it without a workspace.
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := testRecord(AgentGemini, "one", time.Unix(1, 0), "/one", 1)
	stale.Workspace = "" // what the old reader could not resolve
	old := newFingerprintSource(AgentGemini, "unchanged-tree", stale)
	index, err := NewIndex(store, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// The new build reads the very same tree — identical fingerprint — but now
	// resolves the workspace. Reopen the same index, as an upgrade would.
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	fixed := testRecord(AgentGemini, "one", time.Unix(1, 0), "/one", 1)
	fixed.Workspace = "/work/demo" // what the fixed reader now resolves
	upgraded := newFingerprintSource(AgentGemini, "unchanged-tree", fixed)
	index, err = NewIndex(store, upgraded)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the upgrade: a different binary means a different reader.
	index.readerID = "a-different-build"

	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if upgraded.scanCall != 1 {
		t.Fatal("the upgraded reader did not rescan: the stale rows would be served forever")
	}
	records, err := index.Store().Search(ctx, Filter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	if records[0].Workspace != "/work/demo" {
		t.Fatalf("Workspace = %q, want %q: the upgrade served the row the old reader wrote",
			records[0].Workspace, "/work/demo")
	}

	// And the upgraded reader still skips on its own second run.
	if _, err := index.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if upgraded.scanCall != 1 {
		t.Fatalf("scan calls = %d, want 1: the reader identity must be stable within a build",
			upgraded.scanCall)
	}
}
