package sessionindex

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type fakeSource struct {
	name     string
	result   ScanResult
	err      error
	scanCall int
}

func (s *fakeSource) Name() string {
	return s.name
}

func (s *fakeSource) Scan(context.Context) (ScanResult, error) {
	s.scanCall++
	return s.result, s.err
}

func TestIndexRefreshContinuesAfterSourceFailureWithoutDeletingOldRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	claude := &fakeSource{
		name: AgentClaude,
		result: ScanResult{
			Records:  []Record{testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1)},
			Warnings: []Warning{{Code: "fixture_warning", Message: "\x1b[31msafe\x1b[0m"}},
		},
	}
	codex := &fakeSource{
		name:   AgentCodex,
		result: ScanResult{Records: []Record{testRecord(AgentCodex, "two", time.Unix(2, 0), "/two", 1)}},
	}
	index, err := NewIndex(store, codex, claude)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Failed() || len(refreshed.Sources) != 2 {
		t.Fatalf("refresh = %+v", refreshed)
	}
	if refreshed.Sources[0].Name != AgentClaude || refreshed.Sources[1].Name != AgentCodex {
		t.Fatalf("source order = %+v", refreshed.Sources)
	}
	if len(refreshed.Warnings) != 1 ||
		refreshed.Warnings[0].Agent != AgentClaude ||
		refreshed.Warnings[0].Message != "safe" {
		t.Fatalf("warnings = %+v", refreshed.Warnings)
	}

	claude.err = errors.New("\x1b[31mtemporary failure\x1b[0m")
	claude.result.Records = nil
	codex.result.Records = nil
	refreshed, err = index.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed.Failed() {
		t.Fatalf("failed refresh = %+v", refreshed)
	}
	records, err := index.Search(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if got := joinReferences(records); got != "claude:one" {
		t.Fatalf("records after partial failure = %s", got)
	}
	if refreshed.Sources[0].Error != "temporary failure" {
		t.Fatalf("sanitized source error = %q", refreshed.Sources[0].Error)
	}
}

func TestIndexRejectsDuplicateSourcesAndHonorsCancellation(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := NewIndex(store, &fakeSource{name: AgentClaude}, &fakeSource{name: AgentClaude}); err == nil {
		t.Fatal("duplicate sources unexpectedly accepted")
	}

	source := &fakeSource{name: AgentClaude}
	index, err := NewIndex(store, source)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := index.Refresh(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled refresh error = %v", err)
	}
	if source.scanCall != 0 {
		t.Fatalf("cancelled source scans = %d", source.scanCall)
	}
}

func TestRefreshAgentAndRefreshAndResolveBindSelectedFreshness(t *testing.T) {
	t.Parallel()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	claude := &fakeSource{name: AgentClaude, result: ScanResult{Records: []Record{
		testRecord(AgentClaude, "one", time.Unix(1, 0), "/one", 1),
	}}}
	codex := &fakeSource{name: AgentCodex, result: ScanResult{Records: []Record{
		testRecord(AgentCodex, "two", time.Unix(2, 0), "/two", 1),
	}}}
	index, err := NewIndex(store, claude, codex)
	if err != nil {
		t.Fatal(err)
	}
	record, refresh, fresh, err := index.RefreshAndResolve(context.Background(), "claude:one")
	if err != nil {
		t.Fatal(err)
	}
	if record.Reference() != "claude:one" || !fresh || !refresh.SourceFresh(AgentClaude) || refresh.SourceFresh(AgentCodex) {
		t.Fatalf("targeted resolve = record:%s fresh:%t refresh:%+v", record.Reference(), fresh, refresh)
	}
	if claude.scanCall != 1 || codex.scanCall != 0 {
		t.Fatalf("targeted scan calls claude=%d codex=%d", claude.scanCall, codex.scanCall)
	}

	claude.err = errors.New("controlled source failure")
	record, refresh, fresh, err = index.RefreshAndResolve(context.Background(), "claude:one")
	if err != nil {
		t.Fatal(err)
	}
	if record.Reference() != "claude:one" || fresh || refresh.SourceFresh(AgentClaude) {
		t.Fatalf("stale resolve = record:%s fresh:%t refresh:%+v", record.Reference(), fresh, refresh)
	}
	if _, err := index.RefreshAgent(context.Background(), "missing-agent"); err == nil {
		t.Fatal("missing target source unexpectedly refreshed")
	}
}
