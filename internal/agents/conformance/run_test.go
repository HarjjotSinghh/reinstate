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
	file, err := fsys.Open("ok.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
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

// A single-platform probe used to satisfy the T1 evidence check even though
// docs/agent-support-tiers.md requires macOS and native Windows.
func TestProbePlatformGap(t *testing.T) {
	tests := []struct {
		name    string
		reports []string
		want    string
	}{
		{"none", nil, "macos and windows"},
		{"macos only", []string{"docs/testing/results/agent-probes/2026-08-16-macos-kimi.json"}, "windows"},
		{"windows only", []string{"docs/testing/results/agent-probes/2026-08-16-windows-kimi.json"}, "macos"},
		{
			name: "both",
			reports: []string{
				"docs/testing/results/agent-probes/2026-08-16-macos-kimi.json",
				"docs/testing/results/agent-probes/2026-08-20-windows-kimi.json",
			},
			want: "",
		},
		{
			name: "wsl does not substitute for native windows",
			reports: []string{
				"docs/testing/results/agent-probes/2026-08-16-macos-kimi.json",
				"docs/testing/results/agent-probes/2026-08-16-wsl-kimi.json",
			},
			want: "windows",
		},
	}
	for _, tt := range tests {
		if got := probePlatformGap(tt.reports); got != tt.want {
			t.Errorf("%s: probePlatformGap = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestNewSessionNeedsNoSessionID pins the asymmetry between the templates that
// address an existing session and the one that does not.
//
// Resume and Fork must substitute the resolved identifier or they silently
// resume the wrong session. NewSession must not be held to that rule: a vendor
// that assigns its own identifier takes a bare "start fresh" flag, and
// requiring the placeholder would reject a shipped T4 destination.
func TestNewSessionNeedsNoSessionID(t *testing.T) {
	t.Parallel()

	base := func() agents.NativeSpec {
		return agents.NativeSpec{
			Resume: []string{"--resume", "{{.SessionID}}"},
			Fork:   []string{"--resume", "{{.SessionID}}", "--fork-session"},
		}
	}

	for _, tc := range []struct {
		name       string
		newSession []string
		wantErr    string
	}{
		{name: "vendor assigns the id", newSession: []string{"--new-session"}},
		{name: "caller pins the id", newSession: []string{"--session-id", "{{.SessionID}}"}},
		{name: "absent entirely", newSession: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := base()
			spec.NewSession = tc.newSession
			if err := checkNativeArgv(agents.Descriptor{Native: &spec}); err != nil {
				t.Fatalf("NewSession %q rejected: %v", tc.newSession, err)
			}
		})
	}

	t.Run("resume still enforced", func(t *testing.T) {
		t.Parallel()

		spec := base()
		spec.Resume = []string{"--resume"}
		err := checkNativeArgv(agents.Descriptor{Native: &spec})
		if err == nil {
			t.Fatal("a Resume template with no {{.SessionID}} was accepted")
		}
		if !strings.Contains(err.Error(), "Resume") {
			t.Fatalf("error does not name the offending template: %v", err)
		}
	})

	t.Run("fork still enforced", func(t *testing.T) {
		t.Parallel()

		spec := base()
		spec.Fork = []string{"--fork-session"}
		err := checkNativeArgv(agents.Descriptor{Native: &spec})
		if err == nil {
			t.Fatal("a Fork template with no {{.SessionID}} was accepted")
		}
		if !strings.Contains(err.Error(), "Fork") {
			t.Fatalf("error does not name the offending template: %v", err)
		}
	})
}
