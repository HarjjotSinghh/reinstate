package conformance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestIsolationFSRejectsWritesAndOutsideRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys, err := newIsolationFS(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Open("ok.jsonl"); err != nil {
		t.Fatal(err)
	}
	if len(fsys.Opens()) != 1 {
		t.Fatalf("opens = %v", fsys.Opens())
	}
	if err := probeIsolationFS(fsys); err != nil {
		t.Fatal(err)
	}
}

func TestBrokenEvidencePathFails(t *testing.T) {
	t.Parallel()
	d := agents.Descriptor{
		Key:         "broken",
		DisplayName: "Broken",
		Vendor:      "Test",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Evidence: agents.Evidence{
			StoragePage:  "docs/session-storage/kimi.md",
			ProbeReports: []string{"testdata/does-not-exist-probe.json"},
			Fixtures:     []string{"testdata/sessionindex/claude/macos"},
		},
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	err = checkEvidence(d, root)
	if err == nil {
		t.Fatal("evidence check passed for a missing probe path")
	}
	if !strings.Contains(err.Error(), "testdata/does-not-exist-probe.json") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunFailsBrokenEvidence(t *testing.T) {
	d := agents.Descriptor{
		Key:         "broken",
		DisplayName: "Broken",
		Vendor:      "Test",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/missing-agent.md",
			Fixtures:    []string{"testdata/sessionindex/claude/macos"},
		},
		NewIndexSource: func(agents.Env) (sessionindex.Source, error) {
			return emptySource{}, nil
		},
	}
	recorder := &recordT{T: t}
	Run(recorder, d, Fixtures{})
	if !recorder.failed {
		t.Fatal("Run did not fail a descriptor with a broken evidence path")
	}
}

type recordT struct {
	*testing.T
	failed bool
}

func (r *recordT) Errorf(format string, args ...any) {
	r.failed = true
	r.Logf(format, args...)
}

func (r *recordT) Helper() {}

type emptySource struct{}

func (emptySource) Name() string { return "broken" }
func (emptySource) Scan(context.Context) (sessionindex.ScanResult, error) {
	return sessionindex.ScanResult{}, nil
}

func TestIsolationFSOpenFileWriteRejected(t *testing.T) {
	t.Parallel()
	fsys, err := newIsolationFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = fsys.OpenFile("x", os.O_RDWR|os.O_CREATE, 0o600)
	if !errors.Is(err, errWriteAttempt) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckStructureT0Reason(t *testing.T) {
	t.Parallel()
	if err := checkStructure(agents.Descriptor{
		Key: "x", DisplayName: "X", Vendor: "V",
		Tier: agents.TierKnown, Family: agents.FamilyRemote,
	}); err == nil {
		t.Fatal("T0 without reason passed")
	}
	if err := checkStructure(agents.Descriptor{
		Key: "x", DisplayName: "X", Vendor: "V",
		Tier: agents.TierDiscover, Family: agents.FamilyHomeTree,
		T0Reason: agents.T0LayoutUnverified,
	}); err == nil {
		t.Fatal("T1 with T0Reason passed")
	}
}
