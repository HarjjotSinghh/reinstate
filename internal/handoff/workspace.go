package handoff

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/project"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// BlockedError is returned when BindWorkspace receives a DecisionBlocked
// preflight report. ExitCode matches Phase 3 blocked-environment semantics.
type BlockedError struct {
	Report   preflight.Report
	ExitCode int
}

func (e *BlockedError) Error() string {
	if e == nil {
		return "handoff: environment preflight is blocked"
	}
	return "handoff: environment preflight is blocked"
}

// BindWorkspace runs the Phase 3 verifier for the source record and converts
// the result into capsule workspace truth with portable path tokens.
func BindWorkspace(ctx context.Context, v preflight.Verifier, rec sessionindex.Record) (capsule.Workspace, preflight.Report, error) {
	if v == nil {
		return capsule.Workspace{}, preflight.Report{}, errors.New("handoff: verifier is required")
	}

	report, err := v.Verify(ctx, preflight.Input{
		SessionRef:  rec.Reference(),
		Agent:       rec.Agent,
		Workspace:   rec.Workspace,
		AgentRoot:   sessionindex.AgentRoot(rec),
		Recorded:    rec.RecordedEnvironment,
		SourceFresh: true,
	})
	if err != nil {
		return capsule.Workspace{}, preflight.Report{}, err
	}

	if report.Decision == preflight.DecisionBlocked {
		code := report.BlockExitCode
		if code == 0 {
			code = exitcode.Runtime
		}
		return capsule.Workspace{}, report, &BlockedError{Report: report, ExitCode: code}
	}

	bound, err := bindCapsuleWorkspace(report, rec)
	if err != nil {
		return capsule.Workspace{}, report, err
	}
	return bound, report, nil
}

func bindCapsuleWorkspace(report preflight.Report, rec sessionindex.Record) (capsule.Workspace, error) {
	abs := strings.TrimSpace(report.Workspace.Git.Root)
	if abs == "" {
		abs = strings.TrimSpace(report.Workspace.Workspace.Path)
	}
	if abs == "" {
		abs = strings.TrimSpace(rec.Workspace)
	}
	if abs == "" {
		return capsule.Workspace{}, errors.New("handoff: workspace root is unavailable")
	}

	projectID := strings.TrimSpace(rec.Project)
	if projectID == "" {
		projectID = project.OpaqueID(abs)
	}
	if projectID == "" {
		return capsule.Workspace{}, errors.New("handoff: canonical project id is unavailable")
	}

	mapper := pathmap.Mapper{Projects: map[string]string{projectID: abs}}
	root := mapper.Normalize(abs)
	if !strings.HasPrefix(root, "${REPO:") {
		return capsule.Workspace{}, fmt.Errorf("handoff: failed to emit portable workspace root for %q", projectID)
	}

	return capsule.Workspace{
		ProjectID:         projectID,
		Root:              root,
		Branch:            report.Workspace.Git.Branch,
		Head:              report.Workspace.Git.Head,
		Dirty:             report.Workspace.Git.WorkingTree.State == workspace.WorkingTreeModified,
		WorkingTreeDigest: report.Workspace.Git.WorkingTree.Digest,
		Path:              abs,
	}, nil
}
