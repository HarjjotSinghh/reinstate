package handoff

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

type fakeVerifier struct {
	report preflight.Report
	err    error
	input  preflight.Input
	calls  int
}

func (f *fakeVerifier) Verify(_ context.Context, input preflight.Input) (preflight.Report, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return preflight.Report{}, f.err
	}
	report := f.report
	if report.SessionRef == "" {
		report.SessionRef = input.SessionRef
	}
	return report, nil
}

func TestBindWorkspaceBlockedPropagatesExitCode(t *testing.T) {
	t.Parallel()

	abs := filepath.Join(t.TempDir(), "repo")
	rec := sessionindex.Record{
		Key: "claude:session-one", Agent: "claude", Project: "github.com/example/demo", Workspace: abs,
	}
	verifier := &fakeVerifier{report: preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		Decision:      preflight.DecisionBlocked,
		BlockExitCode: exitcode.Safety,
		Checks: []preflight.Check{{
			ID: "git.repository", Status: preflight.StatusChanged, Severity: preflight.SeverityBlock,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "repository identity changed",
			ExitCode: exitcode.Safety,
		}},
		Workspace: fingerprint(abs, "main", "abc123", workspace.WorkingTreeClean, ""),
	}}

	bound, report, err := BindWorkspace(context.Background(), verifier, rec)
	if err == nil {
		t.Fatal("BindWorkspace returned nil error for blocked report")
	}
	var blocked *BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("error type = %T (%v), want *BlockedError", err, err)
	}
	if blocked.ExitCode != exitcode.Safety {
		t.Fatalf("ExitCode = %d, want %d", blocked.ExitCode, exitcode.Safety)
	}
	if report.Decision != preflight.DecisionBlocked {
		t.Fatalf("report.Decision = %s, want blocked", report.Decision)
	}
	if bound.Root != "" || bound.ProjectID != "" || bound.Path != "" {
		t.Fatalf("blocked bind returned workspace %+v", bound)
	}
	if verifier.calls != 1 {
		t.Fatalf("Verify calls = %d, want 1", verifier.calls)
	}
	if verifier.input.SessionRef != rec.Key || verifier.input.Agent != rec.Agent || verifier.input.Workspace != abs {
		t.Fatalf("Verify input = %+v", verifier.input)
	}
	if !verifier.input.SourceFresh {
		t.Fatal("Verify input SourceFresh = false, want true")
	}
}

func TestBindWorkspaceWarningRequiresAcknowledgement(t *testing.T) {
	t.Parallel()

	abs := filepath.Join(t.TempDir(), "repo")
	rec := sessionindex.Record{
		Key: "codex:session-two", Agent: "codex", Project: "github.com/example/demo", Workspace: abs,
	}
	verifier := &fakeVerifier{report: preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		Decision:      preflight.DecisionConfirmationRequired,
		Checks: []preflight.Check{
			{
				ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
				Provenance: workspace.ProvenanceCurrentObservation, Message: "source is fresh",
			},
			{
				ID: "baseline.unavailable", Status: preflight.StatusUnknown, Severity: preflight.SeverityWarning,
				Provenance: workspace.ProvenanceUnavailable, Message: "no previous baseline",
				ExitCode: exitcode.Safety,
			},
			{
				ID: "git.branch", Status: preflight.StatusChanged, Severity: preflight.SeverityWarning,
				Provenance: workspace.ProvenanceCurrentObservation, Message: "branch changed",
				ExitCode: exitcode.Safety,
			},
		},
		Workspace: fingerprint(abs, "feature", "def456", workspace.WorkingTreeModified, "digest-one"),
	}}

	bound, report, err := BindWorkspace(context.Background(), verifier, rec)
	if err != nil {
		t.Fatalf("BindWorkspace: %v", err)
	}
	if report.Decision != preflight.DecisionConfirmationRequired {
		t.Fatalf("Decision = %s, want confirmation_required", report.Decision)
	}

	auth, authErr := preflight.Authorize(report, nil)
	if authErr == nil || auth.Allowed {
		t.Fatal("Authorize(nil) allowed a warning report; acknowledgement should be required")
	}
	if auth.ExitCode != exitcode.Safety {
		t.Fatalf("Authorize exit = %d, want %d", auth.ExitCode, exitcode.Safety)
	}
	warnings := preflight.WarningIDs(report)
	if len(warnings) != 2 {
		t.Fatalf("WarningIDs = %v, want two warning IDs", warnings)
	}
	auth, authErr = preflight.Authorize(report, warnings)
	if authErr != nil || !auth.Allowed {
		t.Fatalf("Authorize(exact warnings) = %+v, %v", auth, authErr)
	}

	if bound.Root != "${REPO:github.com/example/demo}" {
		t.Fatalf("Root = %q", bound.Root)
	}
	if !bound.Dirty || bound.Branch != "feature" || bound.Head != "def456" {
		t.Fatalf("bound workspace = %+v", bound)
	}
}

func TestBindWorkspaceRejectsAbsolutePathsInCapsule(t *testing.T) {
	t.Parallel()

	abs := filepath.Join(t.TempDir(), "demo")
	rec := sessionindex.Record{
		Key: "claude:session-three", Agent: "claude", Project: "github.com/example/demo", Workspace: abs,
	}
	verifier := &fakeVerifier{report: preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		Decision:      preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "source is fresh",
		}},
		Workspace: fingerprint(abs, "main", "abc123", workspace.WorkingTreeClean, "digest-two"),
	}}

	bound, _, err := BindWorkspace(context.Background(), verifier, rec)
	if err != nil {
		t.Fatalf("BindWorkspace: %v", err)
	}
	if bound.Path != abs {
		t.Fatalf("private Path = %q, want %q", bound.Path, abs)
	}
	if bound.Root != "${REPO:github.com/example/demo}" {
		t.Fatalf("Root = %q, want portable token", bound.Root)
	}
	if bound.ProjectID != "github.com/example/demo" {
		t.Fatalf("ProjectID = %q", bound.ProjectID)
	}
	if filepath.IsAbs(bound.Root) || strings.Contains(bound.Root, abs) {
		t.Fatalf("absolute path leaked into Root: %q", bound.Root)
	}

	doc := minimalCapsule(bound)
	raw, err := capsule.CanonicalBytes(doc)
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	if strings.Contains(string(raw), abs) {
		t.Fatalf("absolute path leaked into canonical capsule bytes: %s", raw)
	}
	if !strings.Contains(string(raw), "${REPO:github.com/example/demo}") {
		t.Fatalf("portable root missing from canonical bytes: %s", raw)
	}
}

func TestBindWorkspaceOpaqueProjectIDWhenUnset(t *testing.T) {
	t.Parallel()

	abs := filepath.Join(t.TempDir(), "orphan")
	rec := sessionindex.Record{
		Key: "claude:session-four", Agent: "claude", Workspace: abs,
	}
	verifier := &fakeVerifier{report: preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		Decision:      preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Provenance: workspace.ProvenanceCurrentObservation, Message: "source is fresh",
		}},
		Workspace: fingerprint(abs, "main", "abc123", workspace.WorkingTreeClean, ""),
	}}

	bound, _, err := BindWorkspace(context.Background(), verifier, rec)
	if err != nil {
		t.Fatalf("BindWorkspace: %v", err)
	}
	if !strings.HasPrefix(bound.ProjectID, "local/") {
		t.Fatalf("ProjectID = %q, want opaque local/ id", bound.ProjectID)
	}
	wantRoot := "${REPO:" + bound.ProjectID + "}"
	if bound.Root != wantRoot {
		t.Fatalf("Root = %q, want %q", bound.Root, wantRoot)
	}
}

func fingerprint(abs, branch, head string, state workspace.WorkingTreeState, digest string) workspace.Fingerprint {
	return workspace.Fingerprint{
		Workspace: workspace.WorkspaceFingerprint{Path: abs, Exists: true, Directory: true},
		Git: workspace.GitFingerprint{
			Available: true, Repository: true, Root: abs,
			Branch: branch, Head: head,
			WorkingTree: workspace.WorkingTreeFingerprint{State: state, Digest: digest},
		},
	}
}

func minimalCapsule(ws capsule.Workspace) capsule.Capsule {
	src := capsule.SourcePointer{
		Agent: "claude", SessionID: "session-three", RecordKey: "msg-1", ByteOffset: 0, Index: 0,
	}
	ev := capsule.Event{
		ID: capsule.EventID(src), Order: 0, Actor: capsule.ActorUser, Kind: capsule.KindMessage,
		Portability: capsule.PortabilityExact, ContentHash: "abc123", Source: src,
		Blocks: []capsule.Block{{Type: capsule.BlockTypeText, Text: "continue"}},
	}
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			SchemaVer: capsule.SchemaVersion,
			Parent: capsule.Parent{
				Agent: "claude", SessionID: "session-three",
				ArtifactSHA256: "deadbeef", AdapterVersion: "test",
			},
		},
		RawSource: capsule.RawSource{
			Agent: "claude", SessionID: "session-three",
			ArtifactSHA256: "deadbeef", AdapterVersion: "test",
			ByteOffset: 0, SizeBytes: 1,
		},
		Task: capsule.Task{
			Goal:             capsule.TextField{Text: "bind workspace", Portability: capsule.PortabilityNormalized},
			LatestUserIntent: capsule.TextField{Text: "continue", Portability: capsule.PortabilityExact},
			RecentUserMessages: capsule.ListField{
				Items: []string{"continue"}, Portability: capsule.PortabilityExact,
			},
			Constraints:        capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Decisions:          capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			RejectedApproaches: capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Completed:          capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "none"},
			Pending:            capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "none"},
			ChangedFiles:       capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "none"},
			FilesTouchedPerTranscript: capsule.ListField{
				Portability: capsule.PortabilityOmitted, Reason: "none",
			},
			Tests:      capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "none"},
			NextAction: capsule.TextField{Text: "continue", Portability: capsule.PortabilityNormalized},
		},
		Workspace:    ws,
		Conversation: capsule.Conversation{Events: []capsule.Event{ev}},
		Capabilities: capsule.CapabilityDiff{
			Source: map[string]any{}, Destination: map[string]any{},
		},
		Security: capsule.Security{SourceInstructionsAreUntrustedHistory: true},
		Fidelity: capsule.Fidelity{
			Overall: capsule.PortabilityExact, Mode: capsule.FidelityModeStructuredHandoff,
			Components: []capsule.Component{{
				Name: "conversation", Portability: capsule.PortabilityExact, Count: 1, Bytes: 1,
			}},
		},
		Projection: capsule.Projection{Policy: "balanced", IncludedEventIDs: []string{ev.ID}},
	}
}
