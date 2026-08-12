package handoff

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

	changed, omitted := portableChangedFiles(mapper, report.Workspace.Git)

	return capsule.Workspace{
		ProjectID:           projectID,
		Root:                root,
		Branch:              report.Workspace.Git.Branch,
		Head:                report.Workspace.Git.Head,
		Dirty:               report.Workspace.Git.WorkingTree.State == workspace.WorkingTreeModified,
		WorkingTreeDigest:   report.Workspace.Git.WorkingTree.Digest,
		ChangedFiles:        changed,
		ChangedFilesOmitted: omitted,
		Path:                abs,
	}, nil
}

// portableChangedFiles rewrites the live Git porcelain paths observed by the
// workspace probe into pathmap tokens.
//
// Porcelain reports paths relative to the repository root, so each one is
// re-anchored there before normalization; the result is a ${REPO:<id>}/… token
// the destination device can resolve. A path that somehow escapes every
// configured root becomes an external token rather than an absolute path,
// because a capsule may not carry one. Anything that still looks absolute is
// dropped and counted, never emitted.
func portableChangedFiles(mapper pathmap.Mapper, git workspace.GitFingerprint) ([]string, int) {
	omitted := git.WorkingTree.ChangedOmitted
	root := strings.TrimSpace(git.Root)
	if root == "" || len(git.WorkingTree.Changed) == 0 {
		return nil, omitted
	}
	out := make([]string, 0, len(git.WorkingTree.Changed))
	seen := make(map[string]struct{}, len(git.WorkingTree.Changed))
	for _, relative := range git.WorkingTree.Changed {
		token := mapper.NormalizePortable(filepath.Join(root, filepath.FromSlash(relative)))
		if token == "" || capsule.AbsolutePathForbidden(token) {
			omitted++
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out, omitted
}
