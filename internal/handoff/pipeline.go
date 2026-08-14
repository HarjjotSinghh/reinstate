package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/secretscan"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

// ErrWarningAck is returned when handoff/preflight warnings remain unacknowledged.
var ErrWarningAck = errors.New("handoff: warnings require acknowledgement")

// ErrUsage is a pipeline usage/validation failure (unknown agent, bad flags, etc.).
var ErrUsage = errors.New("handoff: usage")

// ErrCompatibility is returned for untested/unsupported source or destination.
var ErrCompatibility = errors.New("handoff: compatibility")

// ErrSafety is returned for active-session refusal and similar safety stops.
var ErrSafety = errors.New("handoff: safety")

// SessionBusyFunc reports whether the source agent holds the session artifact.
type SessionBusyFunc func(ctx context.Context, agent string, target processcheck.Target) (busy bool, scoped bool, err error)

// ResolveSourceFunc refreshes and resolves the selected source record. Fresh
// must describe the source scan that produced the returned record.
type ResolveSourceFunc func(ctx context.Context, rec sessionindex.Record) (resolved sessionindex.Record, fresh bool, err error)

// Options configures Plan and Execute. Callers inject fakes in unit tests;
// production wires preflight, processcheck, and sessionindex.LaunchRunner.
type Options struct {
	// ToAgent is the destination agent key (required): claude or codex in rc.1.
	ToAgent string
	// Policy defaults to PolicyBalanced when empty.
	Policy Policy
	// Verifier is required. Production uses preflight.DefaultService().
	Verifier preflight.Verifier
	// Reader overrides the registered source transcript reader. Tests use this
	// to keep all reads inside synthetic temporary fixtures.
	Reader transcript.Reader
	// ResolveSource refreshes and resolves the source boundary before any read.
	// WP-21b wires this to sessionindex.Index.RefreshAndResolve.
	ResolveSource ResolveSourceFunc
	// SessionBusy is required. Production wires processcheck.SessionBusy.
	SessionBusy SessionBusyFunc
	// LaunchRunner is required for Execute launches. Tests must pass a fake.
	LaunchRunner sessionindex.LaunchRunner
	// ReinstateHome is $REINSTATE_HOME (absolute). Required for Execute store writes.
	ReinstateHome string
	// AllowActive takes a boundary while the source agent is running.
	AllowActive bool
	// AllowUntested proceeds on CompatibilityUntested source or destination.
	AllowUntested bool
	// AllowWarnings are exact preflight/handoff warning IDs (no wildcards).
	AllowWarnings []string
	// NoRedact skips secretscan. Refused for Grok sources.
	NoRedact bool
	// ChangedFiles overrides the live Git porcelain paths for DeriveCheckpoint.
	// Production leaves it empty: BindWorkspace observes the working tree and
	// supplies the tokenized list. Tests set it to pin an exact list without a
	// real repository. It never invents paths — an empty override falls back to
	// the observation, and an unavailable observation stays empty.
	ChangedFiles []string
	// Capability configures DiscoverAgentContext. When UserHome is empty,
	// discovery skips real home roots (tests must not point at ~/.claude).
	Capability capability.Options
	// Target overrides the registered HandoffTarget (tests).
	Target HandoffTarget
	// SessionExists wires Claude UUID collision checks and is required for the
	// registered production Claude target.
	SessionExists ClaudeSessionExists
	// NewSessionID overrides Claude deterministic UUID generation in tests.
	NewSessionID func() (string, error)
	// Now overrides time.Now for lineage timestamps (tests).
	Now func() time.Time
	// MkdirTemp overrides os.MkdirTemp (tests).
	MkdirTemp func(dir, pattern string) (string, error)
	// WorkingDir is the operator process cwd. The CLI always sets it from
	// os.Getwd(). When empty (unit tests), Plan skips the cwd check. A
	// different git repository than the source session is refused.
	WorkingDir string
}

// PlanResult is the dry-run-pure handoff plan. Plan never writes outside TempDir.
type PlanResult struct {
	Capsule               capsule.Capsule
	Destination           DestinationPlan
	Preflight             preflight.Report
	Parse                 transcript.ParseReport
	WarningIDs            []string
	Artifacts             Artifacts
	TempDir               string
	PlannedFiles          []string // permanent paths under handoffs/<id>/
	EstimatedBytes        int64
	EstimatedTokens       int
	RedactionCounts       map[string]int
	SourceMayHaveAdvanced bool
	HandoffID             string
	LineageRoot           string
}

// ExecuteResult is the post-side-effect outcome of Execute.
type ExecuteResult struct {
	Plan                 PlanResult
	HandoffID            string
	Launched             bool
	DestinationSessionID string
	DestinationState     string
	Lineage              LineageEntry
}

// PipelineError carries a stable exit code for CLI mapping.
type PipelineError struct {
	Code int
	Err  error
}

func (e *PipelineError) Error() string {
	if e == nil || e.Err == nil {
		return "handoff: pipeline error"
	}
	return e.Err.Error()
}

func (e *PipelineError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func pipelineErrorf(code int, format string, args ...any) error {
	return &PipelineError{Code: code, Err: fmt.Errorf(format, args...)}
}

func pipelineWrap(code int, err error) error {
	if err == nil {
		return nil
	}
	var pe *PipelineError
	if errors.As(err, &pe) {
		return err
	}
	return &PipelineError{Code: code, Err: err}
}

// Plan builds a structured handoff plan without durable side effects.
// It may write preview artifacts under a temp directory only.
func Plan(ctx context.Context, rec sessionindex.Record, opts Options) (PlanResult, error) {
	if err := ctx.Err(); err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	if opts.ResolveSource == nil {
		return PlanResult{}, pipelineErrorf(exitcode.Runtime, "handoff: source resolver is required")
	}
	if opts.SessionBusy == nil {
		return PlanResult{}, pipelineErrorf(exitcode.Runtime, "handoff: session busy check is required")
	}
	resolved, fresh, err := opts.ResolveSource(ctx, rec)
	if err != nil {
		code := exitcode.Runtime
		switch {
		case errors.Is(err, sessionindex.ErrAmbiguous):
			code = exitcode.Conflict
		case errors.Is(err, sessionindex.ErrNotFound):
			code = exitcode.Usage
		}
		return PlanResult{}, pipelineWrap(code, fmt.Errorf("handoff: resolve source: %w", err))
	}
	if !fresh {
		return PlanResult{}, pipelineErrorf(exitcode.Compatibility, "%w: source session index is not fresh", ErrCompatibility)
	}
	rec = resolved
	to := normalizeAgent(opts.ToAgent)
	if to == "" {
		return PlanResult{}, pipelineErrorf(exitcode.Usage, "%w: --to AGENT is required", ErrUsage)
	}
	if to == normalizeAgent(rec.Agent) {
		return PlanResult{}, pipelineErrorf(exitcode.Usage, "%w: source and destination agents must differ", ErrUsage)
	}
	if opts.Verifier == nil {
		return PlanResult{}, pipelineErrorf(exitcode.Runtime, "handoff: verifier is required")
	}
	if opts.Policy != "" && normalizePolicy(opts.Policy) != opts.Policy {
		return PlanResult{}, pipelineErrorf(exitcode.Usage, "%w: unknown handoff policy %q", ErrUsage, opts.Policy)
	}
	policy := normalizePolicy(opts.Policy)

	if opts.NoRedact {
		if err := transcript.RefuseNoRedact(rec.Agent); err != nil {
			return PlanResult{}, pipelineWrap(exitcode.Usage, err)
		}
	}

	reader := opts.Reader
	if reader == nil {
		var ok bool
		reader, ok = transcript.Get(normalizeAgent(rec.Agent))
		if !ok {
			return PlanResult{}, pipelineErrorf(exitcode.Compatibility, "%w: no transcript reader for source agent %q", ErrCompatibility, rec.Agent)
		}
	}

	sourceMayAdvance := false
	busy, _, err := opts.SessionBusy(ctx, rec.Agent, processcheck.Target{
		SessionID:   rec.ID,
		Path:        rec.SourcePath,
		ProjectRoot: rec.Workspace,
	})
	if err != nil {
		// A failed host process listing is not evidence the source is busy.
		// Blocking Plan here turned a WMI/tasklist error into a 5-minute
		// runtime failure that skipped every dry-run row on Windows.
		busy = false
	}
	if busy && !opts.AllowActive {
		return PlanResult{}, pipelineErrorf(exitcode.Safety, "%w: source session is active; close it or pass --allow-active", ErrSafety)
	}
	if busy && opts.AllowActive {
		sourceMayAdvance = true
	}

	compat, err := reader.Probe(ctx, rec)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	if !adapter.CanRestore(compat, opts.AllowUntested) {
		return PlanResult{}, pipelineErrorf(exitcode.Compatibility, "%w: source agent %q is %s", ErrCompatibility, rec.Agent, compat)
	}

	boundary, err := reader.Snapshot(ctx, rec)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: snapshot source: %w", err))
	}
	events, parseReport, err := reader.Parse(ctx, boundary)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: parse source: %w", err))
	}
	events = transcript.LinkToolResults(events)

	if cwd := strings.TrimSpace(opts.WorkingDir); cwd != "" {
		if err := refuseForeignWorkspaceOnDifferentRepository(cwd, rec.Workspace); err != nil {
			return PlanResult{}, err
		}
	}
	if local := remapForeignWorkspace(rec.Workspace, opts.WorkingDir); local != rec.Workspace {
		rec.Workspace = local
	}
	ws, report, err := BindWorkspace(ctx, opts.Verifier, rec)
	if err != nil {
		var blocked *BlockedError
		if errors.As(err, &blocked) {
			return PlanResult{Preflight: report}, pipelineWrap(blocked.ExitCode, blocked)
		}
		return PlanResult{Preflight: report}, pipelineWrap(exitcode.Runtime, err)
	}
	if cwd := strings.TrimSpace(opts.WorkingDir); cwd != "" {
		if err := refuseMismatchedRepository(cwd, firstNonEmpty(ws.Path, rec.Workspace)); err != nil {
			return PlanResult{Preflight: report}, err
		}
	}

	target, err := resolveTarget(opts, to)
	if err != nil {
		return PlanResult{}, err
	}
	dstCompat, err := target.Compatible(ctx)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	if !adapter.CanRestore(dstCompat, opts.AllowUntested) {
		return PlanResult{}, pipelineErrorf(exitcode.Compatibility, "%w: destination agent %q is %s", ErrCompatibility, to, dstCompat)
	}

	capOpts := opts.Capability
	capOpts.ProjectRoot = firstNonEmpty(capOpts.ProjectRoot, ws.Path, rec.Workspace)
	capOpts.WorkingDir = firstNonEmpty(capOpts.WorkingDir, capOpts.ProjectRoot)
	srcInv := discoverInventory(ctx, rec.Agent, capOpts)
	dstInv := discoverInventory(ctx, to, capOpts)
	capDiff := DiffCapabilities(srcInv, dstInv, rec.Agent, to)

	redactedEvents, redactions, redactionCounts, err := redactEvents(events, opts.NoRedact)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	// The changed-file list is live workspace truth, so it comes from the
	// workspace binding rather than from the transcript. An explicit override
	// wins only when a caller supplied one.
	changedFiles := opts.ChangedFiles
	if len(changedFiles) == 0 {
		changedFiles = ws.ChangedFiles
	}
	task := DeriveCheckpoint(CheckpointInput{
		Events:           redactedEvents,
		Workspace:        report.Workspace,
		Changed:          append([]string(nil), changedFiles...),
		ChangedTruncated: ws.ChangedFilesOmitted > 0,
	})
	ws.ChangedFiles = append([]string(nil), task.ChangedFiles.Items...)
	ws.Tests = append([]string(nil), task.Tests.Items...)

	included, sidecar, fidelity := Apply(policy, redactedEvents)
	fidelity = capsule.AggregateFidelity(nil, append(append(capsule.Components(nil), fidelity.Components...), taskFidelityComponents(task)...))
	if fidelity.Mode == "" {
		fidelity.Mode = capsule.FidelityModeStructuredHandoff
	}

	adapterVersion := strings.TrimSpace(report.Agent.Version)
	if adapterVersion == "" {
		adapterVersion = "unknown"
	}
	raw := capsule.RawSource{
		Agent:          normalizeAgent(rec.Agent),
		SessionID:      rec.ID,
		ArtifactSHA256: boundary.SHA256,
		AdapterVersion: adapterVersion,
		ByteOffset:     boundary.ByteOffset,
		SizeBytes:      boundary.SizeBytes,
		Partial:        boundary.Partial || sourceMayAdvance,
		Path:           boundary.Path(),
	}

	c := capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			SchemaVer: capsule.SchemaVersion,
			Parent: capsule.Parent{
				Agent:          raw.Agent,
				SessionID:      raw.SessionID,
				ArtifactSHA256: raw.ArtifactSHA256,
				AdapterVersion: raw.AdapterVersion,
			},
		},
		RawSource:    raw,
		Task:         task,
		Workspace:    ws,
		Conversation: capsule.Conversation{Events: included},
		Capabilities: capDiff,
		Security: capsule.Security{
			Redactions:                            redactions,
			SourceInstructionsAreUntrustedHistory: true,
		},
		Fidelity: fidelity,
		Projection: capsule.Projection{
			Policy: string(policy),
		},
	}
	if grok, ok := reader.(interface{ ForcedSecurity() capsule.Security }); ok {
		forced := grok.ForcedSecurity()
		c.Security.DestinationWarning = forced.DestinationWarning
		c.Security.RedactionForced = forced.RedactionForced || c.Security.RedactionForced
		c.Security.SourceInstructionsAreUntrustedHistory = true
	}
	if len(sidecar) > 0 {
		c.Conversation.FullHistoryRef = "sidecar/events.jsonl"
		c.Projection.SidecarRef = "sidecar/events.jsonl"
	}
	ids := make([]string, 0, len(included))
	for _, e := range included {
		ids = append(ids, e.ID)
	}
	c.Projection.IncludedEventIDs = ids

	// Compute content ID before projection hashes (hashes depend on the id path).
	c.Identity.ID = ""
	c.Identity.LineageRoot, err = sourceLineageRoot(opts.ReinstateHome, rec)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: read source lineage: %w", err))
	}
	c.Projection.BootstrapSHA256 = ""
	c.Projection.MarkdownSHA256 = ""
	id, err := capsule.ComputeID(c)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	c.Identity.ID = id
	if c.Identity.LineageRoot == "" {
		c.Identity.LineageRoot = id
	}

	handoffDir := permanentHandoffDir(opts.ReinstateHome, id)
	projectionMD, err := RenderProjection(c)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	bootstrap, err := RenderBootstrap(c, handoffDir)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	fidelityJSON, err := json.Marshal(c.Fidelity)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	sidecarEvents, err := encodeSidecarEvents(redactedEvents, sidecar, policy != PolicyCheckpoint)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}

	c.Projection.EstimatedBytes = int64(len(projectionMD))
	c.Projection.EstimatedTokens = int64(EstimateTokens(projectionMD))
	c.Projection.MarkdownSHA256 = sha256Hex(projectionMD)

	destPlan, _, err := planDestination(ctx, target, c, policy, opts, bootstrap)
	if err != nil {
		return PlanResult{}, err
	}
	c.Projection.BootstrapSHA256 = sha256Hex(destPlan.Bootstrap)

	if err := capsule.Validate(c); err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: capsule validate: %w", err))
	}

	warningIDs := mergeWarningIDs(preflight.WarningIDs(report), CapabilityWarningIDs(capDiff))
	if err := validateWarningAcks(warningIDs, opts.AllowWarnings, false); err != nil {
		return PlanResult{}, err
	}

	mkdirTemp := opts.MkdirTemp
	if mkdirTemp == nil {
		mkdirTemp = os.MkdirTemp
	}
	tempDir, err := mkdirTemp("", "reinstate-handoff-*")
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: temp dir: %w", err))
	}
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()
	previewDir := filepath.Join(tempDir, id)
	if err := os.MkdirAll(previewDir, 0o700); err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}

	arts := Artifacts{
		ProjectionMD:  projectionMD,
		Bootstrap:     destPlan.Bootstrap,
		FidelityJSON:  fidelityJSON,
		SidecarEvents: sidecarEvents,
	}
	if err := writePreviewArtifacts(previewDir, c, arts); err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}

	planned := permanentPlannedFiles(opts.ReinstateHome, id, arts)
	keepTemp = true
	return PlanResult{
		Capsule:               c,
		Destination:           destPlan,
		Preflight:             report,
		Parse:                 parseReport,
		WarningIDs:            warningIDs,
		Artifacts:             arts,
		TempDir:               tempDir,
		PlannedFiles:          planned,
		EstimatedBytes:        c.Projection.EstimatedBytes,
		EstimatedTokens:       int(c.Projection.EstimatedTokens),
		RedactionCounts:       redactionCounts,
		SourceMayHaveAdvanced: sourceMayAdvance,
		HandoffID:             id,
		LineageRoot:           c.Identity.LineageRoot,
	}, nil
}

// Execute runs Plan, then persists artifacts, optionally launches, and records lineage.
func Execute(ctx context.Context, rec sessionindex.Record, opts Options, launch bool) (ExecuteResult, error) {
	plan, err := Plan(ctx, rec, opts)
	if err != nil {
		return ExecuteResult{}, err
	}
	defer func() {
		if plan.TempDir != "" {
			_ = os.RemoveAll(plan.TempDir)
		}
	}()

	if launch {
		if err := validateWarningAcks(plan.WarningIDs, opts.AllowWarnings, true); err != nil {
			return ExecuteResult{Plan: plan}, err
		}
		if auth, err := preflight.Authorize(plan.Preflight, filterPreflightWarnings(opts.AllowWarnings, plan.Preflight)); err != nil || !auth.Allowed {
			code := exitcode.Safety
			if auth.ExitCode != 0 {
				code = auth.ExitCode
			}
			if err == nil {
				err = ErrWarningAck
			}
			return ExecuteResult{Plan: plan}, pipelineWrap(code, err)
		}
	} else if err := validateWarningAcks(plan.WarningIDs, opts.AllowWarnings, false); err != nil {
		return ExecuteResult{Plan: plan}, err
	}
	if launch && opts.LaunchRunner == nil {
		return ExecuteResult{Plan: plan}, pipelineErrorf(exitcode.Runtime, "handoff: LaunchRunner is required")
	}

	home := strings.TrimSpace(opts.ReinstateHome)
	if home == "" {
		return ExecuteResult{Plan: plan}, pipelineErrorf(exitcode.Config, "handoff: reinstate home is required for execute")
	}
	store, err := OpenStore(home)
	if err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Config, err)
	}

	target, err := resolveTarget(opts, normalizeAgent(opts.ToAgent))
	if err != nil {
		return ExecuteResult{Plan: plan}, err
	}
	if claudeTarget, ok := target.(interface {
		claudeSessionCollisionCheck() ClaudeSessionExists
	}); launch && ok {
		sessionID := strings.TrimSpace(plan.Destination.SessionID)
		collisionCheck := claudeTarget.claudeSessionCollisionCheck()
		if sessionID == "" || collisionCheck == nil {
			return ExecuteResult{Plan: plan}, pipelineErrorf(exitcode.Runtime, "handoff: Claude launch collision guard is incomplete")
		}
		lock, err := store.acquireLock(claudeSessionLockName(sessionID))
		if err != nil {
			return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
		}
		defer func() { _ = lock.Close() }()
		// The callback must refresh live session state. The lock serializes
		// concurrent Reinstate executions using this deterministic session ID.
		exists, err := collisionCheck(ctx, sessionID)
		if err != nil {
			return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: final Claude collision check: %w", err))
		}
		if exists {
			return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Safety, ErrClaudeSessionIDCollision)
		}
	}

	id, err := store.Put(plan.Capsule, plan.Artifacts)
	if err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
	}
	if err := target.Materialize(ctx, plan.Destination); err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
	}

	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}

	destID := plan.Destination.SessionID
	destState := VerifyUnresolved
	launched := false
	var launchErr error
	var verifyErr error
	if launch {
		launchStart := now
		launchErr = target.Launch(ctx, plan.Destination, opts.LaunchRunner)
		childStarted := launchErr == nil || errors.Is(launchErr, sessionindex.ErrChildStarted)
		if !childStarted {
			// An untyped launch error provides no proof of spawn. Record the
			// attempt as unresolved and keep Launched false.
		} else {
			launched = true
			verifiedID, state, err := target.Verify(ctx, plan.Destination, launchStart)
			if verifiedID != "" {
				destID = verifiedID
			}
			if err != nil {
				verifyErr = err
				destState = VerifyUnresolved
			} else {
				destState = state
			}
		}
	}

	entry := LineageEntry{
		HandoffID:   id,
		LineageRoot: plan.LineageRoot,
		CreatedAt:   now,
		Source: LineageEndpoint{
			Agent:          plan.Capsule.RawSource.Agent,
			SessionID:      plan.Capsule.RawSource.SessionID,
			ArtifactSHA256: plan.Capsule.RawSource.ArtifactSHA256,
		},
		Destination: LineageEndpoint{
			Agent:     normalizeAgent(opts.ToAgent),
			SessionID: destID,
			State:     destState,
		},
		Policy:           string(normalizePolicy(opts.Policy)),
		CapsuleSHA256:    sha256Hex(mustCanonical(plan.Capsule)),
		ProjectionSHA256: plan.Capsule.Projection.MarkdownSHA256,
		FidelityOverall:  string(plan.Capsule.Fidelity.Overall),
		Launched:         launched,
		Acknowledged:     nil,
	}
	if err := store.AppendLineage(entry); err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
	}

	result := ExecuteResult{
		Plan:                 plan,
		HandoffID:            id,
		Launched:             launched,
		DestinationSessionID: destID,
		DestinationState:     destState,
		Lineage:              entry,
	}
	if launchErr != nil {
		return result, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: destination launch failed; outcome unresolved: %w", launchErr))
	}
	if verifyErr != nil {
		return result, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: verify destination after launch: %w", verifyErr))
	}
	return result, nil
}

func resolveTarget(opts Options, to string) (HandoffTarget, error) {
	if opts.Target != nil {
		return opts.Target, nil
	}
	base, ok := Target(to)
	if !ok {
		return nil, pipelineErrorf(exitcode.Usage, "%w: unknown destination agent %q", ErrUsage, to)
	}
	switch t := base.(type) {
	case *ClaudeTarget:
		if opts.SessionExists == nil {
			return nil, pipelineErrorf(exitcode.Runtime, "handoff: Claude destination session collision check is required")
		}
		cp := *t
		cp.SessionExists = opts.SessionExists
		cp.NewSessionID = opts.NewSessionID
		cp.Bootstrap = func(c capsule.Capsule, _ Policy) ([]byte, error) {
			return RenderBootstrap(c, permanentHandoffDir(opts.ReinstateHome, c.Identity.ID))
		}
		return &cp, nil
	case *CodexTarget:
		cp := *t
		return &cp, nil
	default:
		return base, nil
	}
}

func planDestination(
	ctx context.Context,
	target HandoffTarget,
	c capsule.Capsule,
	policy Policy,
	opts Options,
	renderedBootstrap []byte,
) (DestinationPlan, capsule.Fidelity, error) {
	_ = ctx
	plan, fidelity, err := target.Plan(c, policy)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, pipelineWrap(exitcode.Runtime, err)
	}
	if len(renderedBootstrap) > 0 {
		plan.Bootstrap = append([]byte(nil), renderedBootstrap...)
		plan = rewriteBootstrapArgs(plan, renderedBootstrap)
		caps := target.Capabilities()
		if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
			// Codex-style short fallback when full bootstrap exceeds argv budget.
			projectionPath := filepath.Join(permanentHandoffDir(opts.ReinstateHome, c.Identity.ID), projectionFile)
			if !filepath.IsAbs(projectionPath) {
				return DestinationPlan{}, fidelity, pipelineErrorf(exitcode.Config, "handoff: absolute reinstate home is required for argv fallback")
			}
			short := []byte("Reinstate structured handoff — not native resume. Read " + projectionPath + " and continue from that briefing only. Acknowledge before any mutation.")
			plan.Bootstrap = short
			plan = rewriteBootstrapArgs(plan, short)
			if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
				return DestinationPlan{}, fidelity, pipelineWrap(exitcode.Runtime, err)
			}
		}
	}
	return plan, fidelity, nil
}

func rewriteBootstrapArgs(plan DestinationPlan, bootstrap []byte) DestinationPlan {
	text := string(bootstrap)
	switch normalizeAgent(plan.Agent) {
	case "claude":
		sid := strings.TrimSpace(plan.SessionID)
		if sid == "" && len(plan.Args) >= 2 && plan.Args[0] == "--session-id" {
			sid = plan.Args[1]
		}
		plan.Args = []string{"--session-id", sid, text}
	default:
		plan.Args = []string{text}
	}
	return plan
}

func discoverInventory(ctx context.Context, agent string, opts capability.Options) capability.Inventory {
	switch normalizeAgent(agent) {
	case string(capability.AgentClaude):
		return capability.DiscoverAgentContext(ctx, capability.AgentClaude, opts)
	case string(capability.AgentCodex):
		return capability.DiscoverAgentContext(ctx, capability.AgentCodex, opts)
	default:
		return capability.Inventory{}
	}
}

func redactEvents(events []capsule.Event, noRedact bool) ([]capsule.Event, []capsule.Redaction, map[string]int, error) {
	out := make([]capsule.Event, len(events))
	var all []capsule.Redaction
	counts := map[string]int{}
	for i, e := range events {
		cp := e
		cp.Blocks = append([]capsule.Block(nil), e.Blocks...)
		cp.Redactions = append([]capsule.Redaction(nil), e.Redactions...)
		if noRedact {
			out[i] = cp
			continue
		}
		eventRedactions := append([]capsule.Redaction(nil), e.Redactions...)
		redact := func(value string) string {
			text, matches := secretscan.Redact(value)
			for _, m := range matches {
				cat := capsule.Category(m.Category)
				eventRedactions = append(eventRedactions, capsule.Redaction{Category: cat, Digest: m.Digest})
				counts[string(cat)]++
			}
			return text
		}
		cp.NativeType = redact(cp.NativeType)
		cp.NativeName = redact(cp.NativeName)
		cp.CallID = redact(cp.CallID)
		cp.LinkedCallID = redact(cp.LinkedCallID)
		cp.Reason = redact(cp.Reason)
		for j := range cp.Blocks {
			cp.Blocks[j].Text = redact(cp.Blocks[j].Text)
			cp.Blocks[j].MIME = redact(cp.Blocks[j].MIME)
			cp.Blocks[j].Ref = redact(cp.Blocks[j].Ref)
			cp.Blocks[j].Path = redact(cp.Blocks[j].Path)
			if cp.Blocks[j].Meta != nil {
				meta := make(map[string]string, len(cp.Blocks[j].Meta))
				for key, value := range cp.Blocks[j].Meta {
					meta[redact(key)] = redact(value)
				}
				cp.Blocks[j].Meta = meta
			}
		}
		cp.Redactions = eventRedactions
		all = append(all, eventRedactions...)
		out[i] = cp
	}
	return out, all, counts, nil
}

func encodeSidecarEvents(all []capsule.Event, refs []capsule.SidecarRef, includeBodies bool) ([]byte, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	var b strings.Builder
	if !includeBodies {
		for _, ref := range refs {
			line, err := json.Marshal(ref)
			if err != nil {
				return nil, err
			}
			b.Write(line)
			b.WriteByte('\n')
		}
		return []byte(b.String()), nil
	}
	byID := make(map[string]capsule.Event, len(all))
	for _, e := range all {
		byID[e.ID] = e
	}
	for _, ref := range refs {
		ev, ok := byID[ref.EventID]
		if !ok {
			continue
		}
		line, err := json.Marshal(ev)
		if err != nil {
			return nil, err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func writePreviewArtifacts(dir string, c capsule.Capsule, arts Artifacts) error {
	capsuleBytes, err := capsule.CanonicalBytes(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, capsuleFileName), capsuleBytes, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, projectionFile), arts.ProjectionMD, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, bootstrapFileName), arts.Bootstrap, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, fidelityFileName), arts.FidelityJSON, 0o600); err != nil {
		return err
	}
	if len(arts.SidecarEvents) > 0 {
		sidecar := filepath.Join(dir, sidecarDirName)
		if err := os.MkdirAll(sidecar, 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(sidecar, eventsFileName), arts.SidecarEvents, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func permanentHandoffDir(reinstateHome, id string) string {
	home := strings.TrimSpace(reinstateHome)
	if home == "" {
		return filepath.Join("handoffs", id)
	}
	return filepath.Join(home, handoffsDirName, id)
}

func permanentPlannedFiles(reinstateHome, id string, arts Artifacts) []string {
	root := permanentHandoffDir(reinstateHome, id)
	files := []string{
		filepath.Join(root, capsuleFileName),
		filepath.Join(root, projectionFile),
		filepath.Join(root, bootstrapFileName),
		filepath.Join(root, fidelityFileName),
	}
	if len(arts.SidecarEvents) > 0 {
		files = append(files, filepath.Join(root, sidecarDirName, eventsFileName))
	}
	sort.Strings(files)
	return files
}

func mergeWarningIDs(a, b []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(a)+len(b))
	for _, id := range append(append([]string{}, a...), b...) {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// validateWarningAcks enforces preflight.Authorize semantics on the union of
// warning IDs: exact IDs, no wildcards, no duplicates, unknown ID is usage.
// When requireAll is true, missing acknowledgements are a safety error.
func validateWarningAcks(current, allowed []string, requireAll bool) error {
	currentSet := make(map[string]struct{}, len(current))
	for _, id := range current {
		currentSet[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(allowed))
	for _, raw := range allowed {
		id := strings.TrimSpace(raw)
		if id == "" || strings.ContainsAny(id, "*?[]") {
			return pipelineErrorf(exitcode.Usage, "%w: invalid warning ID %q", ErrUsage, id)
		}
		if _, dup := seen[id]; dup {
			return pipelineErrorf(exitcode.Usage, "%w: duplicate warning ID %q", ErrUsage, id)
		}
		seen[id] = struct{}{}
		if _, ok := currentSet[id]; !ok {
			return pipelineErrorf(exitcode.Usage, "%w: warning ID %q is not a current warning", ErrUsage, id)
		}
	}
	if !requireAll {
		return nil
	}
	var missing []string
	for id := range currentSet {
		if _, ok := seen[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return pipelineErrorf(exitcode.Safety, "%w: %s", ErrWarningAck, strings.Join(missing, ", "))
	}
	return nil
}

func filterPreflightWarnings(allowed []string, report preflight.Report) []string {
	current := make(map[string]struct{})
	for _, id := range preflight.WarningIDs(report) {
		current[id] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for _, id := range allowed {
		id = strings.TrimSpace(id)
		if _, ok := current[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

const maxLineageLookupBytes int64 = 8 << 20

// sourceLineageRoot finds the newest completed handoff whose destination is
// the resolved source session. It performs one bounded, read-only tail read and
// never creates the handoff store. Malformed and partial lines are ignored.
func sourceLineageRoot(reinstateHome string, rec sessionindex.Record) (string, error) {
	home := strings.TrimSpace(reinstateHome)
	if home == "" || !filepath.IsAbs(home) {
		return "", nil
	}
	path := filepath.Join(home, handoffsDirName, lineageFileName)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	start := info.Size() - maxLineageLookupBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	body, err := io.ReadAll(io.LimitReader(f, maxLineageLookupBytes))
	if err != nil {
		return "", err
	}
	if start > 0 {
		newline := strings.IndexByte(string(body), '\n')
		if newline < 0 {
			return "", nil
		}
		body = body[newline+1:]
	}
	lastNewline := strings.LastIndexByte(string(body), '\n')
	if lastNewline < 0 {
		return "", nil
	}
	body = body[:lastNewline]

	root := ""
	for _, line := range strings.Split(string(body), "\n") {
		var entry LineageEntry
		if json.Unmarshal([]byte(strings.TrimSpace(line)), &entry) != nil ||
			!entry.Launched || entry.Destination.State != VerifyResolved ||
			entry.Destination.Agent != rec.Agent || entry.Destination.SessionID != rec.ID {
			continue
		}
		candidate := strings.TrimSpace(entry.LineageRoot)
		if candidate == "" {
			candidate = strings.TrimSpace(entry.HandoffID)
		}
		if validateHandoffID(candidate) == nil {
			root = candidate
		}
	}
	return root, nil
}

func claudeSessionLockName(sessionID string) string {
	digest := sha256Hex([]byte(sessionID))
	return ".claude-session-" + digest[:32] + ".lock"
}

func mustCanonical(c capsule.Capsule) []byte {
	b, err := capsule.CanonicalBytes(c)
	if err != nil {
		return nil
	}
	return b
}
