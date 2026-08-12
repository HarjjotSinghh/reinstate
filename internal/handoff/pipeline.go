package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// Options configures Plan and Execute. Callers inject fakes in unit tests;
// production wires preflight, processcheck, and sessionindex.LaunchRunner.
type Options struct {
	// ToAgent is the destination agent key (required): claude or codex in rc.1.
	ToAgent string
	// Policy defaults to PolicyBalanced when empty.
	Policy Policy
	// Verifier is required. Production uses preflight.DefaultService().
	Verifier preflight.Verifier
	// SessionBusy overrides processcheck.SessionBusy. Nil skips the busy check.
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
	// ChangedFiles are live Git porcelain paths for DeriveCheckpoint. The
	// workspace package does not expose path lists; callers must supply them
	// (empty is honest when unavailable — do not invent paths).
	ChangedFiles []string
	// Capability configures DiscoverAgentContext. When UserHome is empty,
	// discovery skips real home roots (tests must not point at ~/.claude).
	Capability capability.Options
	// Target overrides the registered HandoffTarget (tests).
	Target HandoffTarget
	// SessionExists wires Claude UUID collision checks (production index).
	SessionExists ClaudeSessionExists
	// NewSessionID overrides Claude UUID generation (tests / determinism).
	NewSessionID func() (string, error)
	// Now overrides time.Now for lineage timestamps (tests).
	Now func() time.Time
	// MkdirTemp overrides os.MkdirTemp (tests).
	MkdirTemp func(dir, pattern string) (string, error)
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
	to := normalizeAgent(opts.ToAgent)
	if to == "" {
		return PlanResult{}, pipelineErrorf(exitcode.Usage, "%w: --to AGENT is required", ErrUsage)
	}
	if opts.Verifier == nil {
		return PlanResult{}, pipelineErrorf(exitcode.Runtime, "handoff: verifier is required")
	}
	policy := normalizePolicy(opts.Policy)

	if opts.NoRedact {
		if err := transcript.RefuseNoRedact(rec.Agent); err != nil {
			return PlanResult{}, pipelineWrap(exitcode.Usage, err)
		}
	}

	reader, ok := transcript.Get(normalizeAgent(rec.Agent))
	if !ok {
		return PlanResult{}, pipelineErrorf(exitcode.Compatibility, "%w: no transcript reader for source agent %q", ErrCompatibility, rec.Agent)
	}

	sourceMayAdvance := false
	if opts.SessionBusy != nil && strings.TrimSpace(rec.SourcePath) != "" {
		busy, _, err := opts.SessionBusy(ctx, rec.Agent, processcheck.Target{
			SessionID:   rec.ID,
			Path:        rec.SourcePath,
			ProjectRoot: rec.Workspace,
		})
		if err != nil {
			return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: session busy check: %w", err))
		}
		if busy && !opts.AllowActive {
			return PlanResult{}, pipelineErrorf(exitcode.Safety, "%w: source session is active; close it or pass --allow-active", ErrSafety)
		}
		if busy && opts.AllowActive {
			sourceMayAdvance = true
		}
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

	ws, report, err := BindWorkspace(ctx, opts.Verifier, rec)
	if err != nil {
		var blocked *BlockedError
		if errors.As(err, &blocked) {
			return PlanResult{Preflight: report}, pipelineWrap(blocked.ExitCode, blocked)
		}
		return PlanResult{Preflight: report}, pipelineWrap(exitcode.Runtime, err)
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

	task := DeriveCheckpoint(CheckpointInput{
		Events:    events,
		Workspace: report.Workspace,
		Changed:   append([]string(nil), opts.ChangedFiles...),
	})
	ws.ChangedFiles = append([]string(nil), task.ChangedFiles.Items...)
	ws.Tests = append([]string(nil), task.Tests.Items...)

	redactedEvents, redactions, redactionCounts, err := redactEvents(events, opts.NoRedact)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	included, sidecar, fidelity := Apply(policy, redactedEvents)
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
			Redactions:                             redactions,
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
	c.Identity.LineageRoot = ""
	c.Projection.BootstrapSHA256 = ""
	c.Projection.MarkdownSHA256 = ""
	id, err := capsule.ComputeID(c)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}
	c.Identity.ID = id
	c.Identity.LineageRoot = id

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
	sidecarEvents, err := encodeSidecarEvents(redactedEvents, sidecar)
	if err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, err)
	}

	c.Projection.EstimatedBytes = int64(len(projectionMD))
	c.Projection.EstimatedTokens = int64(EstimateTokens(projectionMD))
	c.Projection.BootstrapSHA256 = sha256Hex(bootstrap)
	c.Projection.MarkdownSHA256 = sha256Hex(projectionMD)

	if err := capsule.Validate(c); err != nil {
		return PlanResult{}, pipelineWrap(exitcode.Runtime, fmt.Errorf("handoff: capsule validate: %w", err))
	}

	destPlan, _, err := planDestination(ctx, target, c, policy, opts, bootstrap)
	if err != nil {
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

	warningIDs := mergeWarningIDs(preflight.WarningIDs(report), CapabilityWarningIDs(capDiff))
	if err := validateWarningAcks(warningIDs, opts.AllowWarnings, false); err != nil {
		return PlanResult{}, err
	}

	planned := permanentPlannedFiles(opts.ReinstateHome, id, arts)
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

	home := strings.TrimSpace(opts.ReinstateHome)
	if home == "" {
		return ExecuteResult{Plan: plan}, pipelineErrorf(exitcode.Config, "handoff: reinstate home is required for execute")
	}
	store, err := OpenStore(home)
	if err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Config, err)
	}

	id, err := store.Put(plan.Capsule, plan.Artifacts)
	if err != nil {
		return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
	}

	target, err := resolveTarget(opts, normalizeAgent(opts.ToAgent))
	if err != nil {
		return ExecuteResult{Plan: plan}, err
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
	if launch {
		if opts.LaunchRunner == nil {
			return ExecuteResult{Plan: plan}, pipelineErrorf(exitcode.Runtime, "handoff: LaunchRunner is required")
		}
		launchStart := now
		if err := target.Launch(ctx, plan.Destination, opts.LaunchRunner); err != nil {
			return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, err)
		}
		launched = true
		verifiedID, state, verifyErr := target.Verify(ctx, plan.Destination, launchStart)
		if verifyErr != nil {
			return ExecuteResult{Plan: plan}, pipelineWrap(exitcode.Runtime, verifyErr)
		}
		destID = verifiedID
		destState = state
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

	return ExecuteResult{
		Plan:                 plan,
		HandoffID:            id,
		Launched:             launched,
		DestinationSessionID: destID,
		DestinationState:     destState,
		Lineage:              entry,
	}, nil
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
			short := []byte("Reinstate structured handoff — not native resume. Read projection.md and continue from that briefing only. Acknowledge before any mutation.")
			plan.Bootstrap = short
			plan = rewriteBootstrapArgs(plan, short)
			if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
				return DestinationPlan{}, fidelity, pipelineWrap(exitcode.Runtime, err)
			}
		}
	}
	_ = opts
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
		if noRedact {
			out[i] = cp
			continue
		}
		var eventRedactions []capsule.Redaction
		for j, b := range cp.Blocks {
			if b.Text == "" {
				continue
			}
			text, matches := secretscan.Redact(b.Text)
			cp.Blocks[j].Text = text
			for _, m := range matches {
				cat := capsule.Category(m.Category)
				eventRedactions = append(eventRedactions, capsule.Redaction{Category: cat, Digest: m.Digest})
				counts[string(cat)]++
			}
		}
		cp.Redactions = append(append([]capsule.Redaction(nil), e.Redactions...), eventRedactions...)
		all = append(all, eventRedactions...)
		out[i] = cp
	}
	return out, all, counts, nil
}

func encodeSidecarEvents(all []capsule.Event, refs []capsule.SidecarRef) ([]byte, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	byID := make(map[string]capsule.Event, len(all))
	for _, e := range all {
		byID[e.ID] = e
	}
	var b strings.Builder
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

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func mustCanonical(c capsule.Capsule) []byte {
	b, err := capsule.CanonicalBytes(c)
	if err != nil {
		return nil
	}
	return b
}
