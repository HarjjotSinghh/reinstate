package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

const (
	grokTargetName     = "grok"
	grokExecutable     = "grok"
	grokSessionsDir    = "sessions"
	grokSummaryFile    = "summary.json"
	grokMaxProjectDirs = 4096
)

// DestinationWarningGrokDestination is surfaced before any handoff whose
// destination is Grok Build.
//
// docs/session-storage/grok.md records that this CLI has a documented history
// of transmitting repository contents — Git history and unredacted .env
// material — to xAI cloud storage, and requires an explicit warning plus
// unconditional redaction on the Grok path. That requirement was written for
// Grok as a source; it applies at least as strongly when Reinstate is the one
// putting a briefing about the operator's repository into Grok.
const DestinationWarningGrokDestination = "grok_destination_upload_history"

// ErrGrokSessionIDCollision means the capsule-derived destination UUID already
// exists under the destination Grok session store.
//
// This is a vendor precondition, not a Reinstate preference: `grok
// --session-id <uuid>` requires that the UUID "must not already exist under
// the target session directory". Refusing is the only correct response —
// Reinstate never writes into a vendor's session store, so it cannot make room.
var ErrGrokSessionIDCollision = errors.New(
	"handoff: Grok --session-id already exists under the destination session store; refuse rather than collide",
)

// refuseNoRedactDestination returns transcript.ErrNoRedactRefused when the
// destination is Grok Build. Callers map it to exit 2.
func refuseNoRedactDestination(agent string) error {
	if !strings.EqualFold(strings.TrimSpace(agent), grokTargetName) {
		return nil
	}
	return fmt.Errorf(
		"%w: redaction is forced for Grok Build destinations (%s)",
		transcript.ErrNoRedactRefused,
		DestinationWarningGrokDestination,
	)
}

// GrokTarget is the Grok Build handoff destination.
//
// It launches `grok --session-id <uuid> "<bootstrap>"` in the verified
// workspace, which starts a **new** Grok session — never a cross-agent resume
// and never a native resume of the source. It writes nothing under the Grok
// root (ADR 0003), including no directory-trust record: unlike Claude Code and
// Codex CLI, no Grok workspace-trust file shape has been measured, and
// inventing one would be a vendor-internal write on a guess.
type GrokTarget struct {
	// Root overrides the Grok home ($GROK_HOME). Tests must set this to a
	// synthetic fixture root — never a real ~/.grok. A non-empty Root also
	// suppresses the executable version probe, so unit tests never spawn grok.
	Root string
	// MaxArgvBytes overrides TargetCapabilities.MaxArgvBytes. Non-positive
	// uses DefaultMaxArgvBytes.
	MaxArgvBytes int
	// ForceCompat overrides compatibility detection (tests only).
	ForceCompat adapter.Compatibility
	// NewSessionID overrides the deterministic capsule-derived UUID (tests).
	NewSessionID func() (string, error)
	// Bootstrap builds the argv prompt. Nil uses a bounded stub; the pipeline
	// replaces it with RenderBootstrap.
	Bootstrap func(capsule.Capsule, Policy) ([]byte, error)
}

func init() {
	if err := RegisterTarget(&GrokTarget{}); err != nil {
		panic(err)
	}
}

// Name returns the stable agent key "grok".
func (t *GrokTarget) Name() string { return grokTargetName }

// Capabilities reports Grok destination support. SupportsPinnedID is true:
// `--session-id` names the new conversation's UUID, so the destination session
// id is known before launch rather than reconciled from a scan afterwards.
//
// AttachmentSupport is false because no vendor-published attachment contract
// is recorded for Grok Build; an unpublished trait is omitted, never guessed.
func (t *GrokTarget) Capabilities() TargetCapabilities {
	max := DefaultMaxArgvBytes
	if t != nil && t.MaxArgvBytes > 0 {
		max = t.MaxArgvBytes
	}
	return TargetCapabilities{
		Agent:                 grokTargetName,
		SupportsPinnedID:      true,
		SupportsInitialPrompt: true,
		MaxArgvBytes:          max,
		ContextCeiling:        0,
		AttachmentSupport:     false,
	}
}

// Compatible reports Grok install compatibility without reading session bodies.
//
// An explicit Root (or $GROK_HOME) is treated as a prepared destination and is
// judged on layout alone, matching the Codex adapter: that is the shape a test
// or a sanitized acceptance root has, and it must not spawn the vendor binary.
func (t *GrokTarget) Compatible(ctx context.Context) (adapter.Compatibility, error) {
	if t != nil && t.ForceCompat != "" {
		return t.ForceCompat, nil
	}
	root, explicit, err := t.resolveRoot()
	if err != nil {
		return adapter.CompatibilityNotInstalled, err
	}
	if root == "" {
		return adapter.CompatibilityNotInstalled, nil
	}
	sessions := filepath.Join(root, grokSessionsDir)
	if info, statErr := os.Stat(sessions); statErr != nil || !info.IsDir() {
		if explicit {
			// A prepared root that has not been used yet is still a usable
			// destination: grok creates sessions/ on its first session.
			return adapter.CompatibilitySupported, nil
		}
		return adapter.CompatibilityUntested, nil
	}
	if explicit {
		return adapter.CompatibilitySupported, nil
	}
	output, versionErr := adapter.RunVersionCommand(ctx, grokExecutable)
	if versionErr != nil {
		return adapter.CompatibilityUntested, nil
	}
	reported := adapter.StableVersionFromOutput(string(output))
	if reported == "" {
		return adapter.CompatibilityUntested, nil
	}
	if !adapter.StableVersionInRange(
		reported, sessionindex.GrokMinVerifiedVersion, sessionindex.GrokMaxVerifiedVersion,
	) {
		return adapter.CompatibilityUntested, nil
	}
	return adapter.CompatibilitySupported, nil
}

// Plan derives a non-colliding UUID and builds `grok --session-id <uuid>
// "<bootstrap>"` with Dir set to the verified workspace.
func (t *GrokTarget) Plan(c capsule.Capsule, policy Policy) (DestinationPlan, capsule.Fidelity, error) {
	workspace := strings.TrimSpace(c.Workspace.Path)
	if workspace == "" {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Grok destination requires a verified workspace path")
	}

	bootstrap, err := t.renderBootstrap(c, policy)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}
	if len(bootstrap) == 0 {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Grok bootstrap prompt is empty")
	}
	if len(bootstrap) > BootstrapMaxBytes {
		bootstrap = append([]byte(nil), bootstrap[:BootstrapMaxBytes]...)
	}

	sessionID, err := t.allocateSessionID(c)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}

	_, _, fidelity := Apply(policy, c.Conversation.Events)
	if fidelity.Mode == "" {
		fidelity.Mode = capsule.FidelityModeStructuredHandoff
	}
	plan := DestinationPlan{
		Agent:      grokTargetName,
		Executable: grokExecutable,
		Args:       []string{"--session-id", sessionID, string(bootstrap)},
		Dir:        workspace,
		Files:      nil, // never write vendor-internal Grok files
		SessionID:  sessionID,
		Bootstrap:  bootstrap,
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}
	return plan, fidelity, nil
}

// Materialize validates argv and re-checks the vendor precondition immediately
// before launch. It writes nothing: no planned files, and no directory-trust
// record under the Grok root.
//
// The re-check matters because Plan and Launch are separated by capsule
// storage and by an operator confirmation, and `--session-id` is documented to
// require that the UUID not already exist. A collision that appeared in that
// window must refuse, not race the vendor.
func (t *GrokTarget) Materialize(_ context.Context, plan DestinationPlan) error {
	if err := WritePlannedFiles(plan, t.Capabilities().MaxArgvBytes, nil); err != nil {
		return err
	}
	return t.refuseExistingSessionID(plan.SessionID)
}

// Launch runs the planned Grok argv through the injected LaunchRunner.
// Production passes sessionindex.ExecLaunchRunner, so TTY, executable identity
// and workspace identity guards apply unchanged.
func (t *GrokTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runner == nil {
		return errors.New("handoff: Grok launch requires a LaunchRunner")
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if strings.TrimSpace(plan.Executable) == "" || len(plan.Args) == 0 {
		return errors.New("handoff: Grok launch plan is incomplete")
	}
	if strings.TrimSpace(plan.Dir) == "" {
		return fmt.Errorf("%w: Grok launch workspace is missing", sessionindex.ErrWorkspaceUnavailable)
	}
	return sessionindex.RunLaunch(ctx, sessionindex.LaunchPlan{
		Agent:      grokTargetName,
		SessionRef: sessionindex.CompositeReference(grokTargetName, plan.SessionID),
		Operation:  sessionindex.OperationHandoff,
		Executable: plan.Executable,
		Args:       append([]string(nil), plan.Args...),
		Dir:        plan.Dir,
	}, runner)
}

// Verify resolves the pinned session ID by locating <root>/sessions/<project>/
// <uuid>/ on this device.
//
// It does not compare mtimes. Plan and Materialize both proved the id absent,
// so its presence after launch is the evidence — and a clock comparison
// against a directory the vendor owns would add a way to be wrong without
// adding a way to be right. When the session's own summary.json records a
// working directory, it must equal the launch workspace; a mismatch is
// reported as unresolved rather than claimed.
func (t *GrokTarget) Verify(ctx context.Context, plan DestinationPlan, _ time.Time) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", VerifyUnresolved, err
	}
	id := strings.TrimSpace(plan.SessionID)
	if id == "" {
		return "", VerifyUnresolved, nil
	}
	workspace := strings.TrimSpace(plan.Dir)
	if workspace == "" {
		return "", VerifyUnresolved, errors.New("handoff: Grok verify requires destination workspace")
	}
	root, _, err := t.resolveRoot()
	if err != nil || root == "" {
		return "", VerifyUnresolved, err
	}
	dirs, err := grokSessionDirsFor(root, id)
	if err != nil {
		return "", VerifyUnresolved, err
	}
	var matched []string
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return "", VerifyUnresolved, err
		}
		recorded, ok := grokRecordedWorkspace(dir)
		if ok && !grokWorkspaceEqual(recorded, workspace) {
			continue
		}
		matched = append(matched, dir)
	}
	switch len(matched) {
	case 0:
		return "", VerifyUnresolved, nil
	case 1:
		return id, VerifyResolved, nil
	default:
		// One UUID under two project directories is not something the vendor
		// should produce. Report it rather than picking.
		return "", VerifyAmbiguous, nil
	}
}

func (t *GrokTarget) renderBootstrap(c capsule.Capsule, policy Policy) ([]byte, error) {
	if t != nil && t.Bootstrap != nil {
		return t.Bootstrap(c, policy)
	}
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		id = "unknown"
	}
	return []byte("structured handoff — a new Grok Build session, not native resume. " +
		"Continue capsule " + id + ". Read projection.md in the handoff directory for the full briefing."), nil
}

func (t *GrokTarget) allocateSessionID(c capsule.Capsule) (string, error) {
	id, err := grokSessionIDFor(c)
	if t != nil && t.NewSessionID != nil {
		id, err = t.NewSessionID()
	}
	if err != nil {
		return "", fmt.Errorf("handoff: generate Grok session id: %w", err)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("handoff: generated Grok session id is empty")
	}
	// `grok --session-id` requires a valid UUID. A generated value of any other
	// shape is a bug in the generator, not something to hand to the vendor.
	if !sessionindex.IsGrokSessionID(id) {
		return "", fmt.Errorf("handoff: Grok session id %q is not a UUID", id)
	}
	if err := t.refuseExistingSessionID(id); err != nil {
		return "", err
	}
	return id, nil
}

// refuseExistingSessionID enforces the documented `--session-id` precondition
// against the destination store. When no root is resolvable there is nothing to
// check and nothing is claimed.
func (t *GrokTarget) refuseExistingSessionID(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	root, _, err := t.resolveRoot()
	if err != nil || root == "" {
		return err
	}
	dirs, err := grokSessionDirsFor(root, sessionID)
	if err != nil {
		return err
	}
	if len(dirs) > 0 {
		// Compatibility, not Runtime: the plan is well-formed and Reinstate is
		// healthy — the destination's state cannot accept this id. Reinstate
		// never writes into a vendor store, so this is not recoverable by
		// making room, and the acceptance contract pins exit 5 for it.
		return pipelineWrap(
			exitcode.Compatibility,
			fmt.Errorf("%w: %s", ErrGrokSessionIDCollision, sessionID),
		)
	}
	return nil
}

// resolveRoot returns the destination Grok home and whether it was named
// explicitly. An explicit root is a prepared destination; an implicit one is
// the operator's own install.
func (t *GrokTarget) resolveRoot() (string, bool, error) {
	if t != nil {
		if root := strings.TrimSpace(t.Root); root != "" {
			return filepath.Clean(root), true, nil
		}
	}
	if configured := strings.TrimSpace(os.Getenv("GROK_HOME")); configured != "" {
		return filepath.Clean(configured), true, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, nil
	}
	candidate := filepath.Join(home, ".grok")
	if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
		return candidate, false, nil
	}
	return "", false, nil
}

// grokSessionDirsFor lists every <root>/sessions/<project>/<sessionID>
// directory. Grok buckets sessions by a URL-encoded working directory, and
// Reinstate never recomputes that encoding: it enumerates the buckets the
// vendor actually created instead of guessing a directory name.
func grokSessionDirsFor(root, sessionID string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, grokSessionsDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("handoff: list Grok sessions: %w", err)
	}
	var found []string
	for i, entry := range entries {
		if i >= grokMaxProjectDirs {
			break
		}
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(root, grokSessionsDir, entry.Name(), sessionID)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			found = append(found, candidate)
		}
	}
	return found, nil
}

// grokRecordedWorkspace reads info.cwd from a session's summary.json. The bool
// is false when the file is absent or unreadable, which is "not stated" rather
// than "does not match".
func grokRecordedWorkspace(sessionDir string) (string, bool) {
	file, err := os.Open(filepath.Join(sessionDir, grokSummaryFile))
	if err != nil {
		return "", false
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)))
	if err != nil {
		return "", false
	}
	var summary struct {
		Info struct {
			CWD string `json:"cwd"`
		} `json:"info"`
	}
	if json.Unmarshal(data, &summary) != nil {
		return "", false
	}
	cwd := strings.TrimSpace(summary.Info.CWD)
	return cwd, cwd != ""
}

func grokWorkspaceEqual(recorded, want string) bool {
	recorded = strings.TrimSpace(recorded)
	want = strings.TrimSpace(want)
	if recorded == "" || want == "" {
		return false
	}
	return recorded == want || filepath.Clean(recorded) == filepath.Clean(want)
}

// grokSessionIDFor returns a deterministic UUIDv4 derived from the capsule
// content ID, so dry-run and execute plan the exact same argv.
//
// The salt differs from the Claude derivation on purpose: one capsule handed
// off to both agents must not produce the same UUID in two vendors' stores.
func grokSessionIDFor(c capsule.Capsule) (string, error) {
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		var err error
		id, err = capsule.ComputeID(c)
		if err != nil {
			return "", err
		}
	}
	b := sha256.Sum256([]byte("reinstate:grok-session:" + id))
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	var node uint64
	for _, v := range b[10:16] {
		node = node<<8 | uint64(v)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		node,
	), nil
}

var _ HandoffTarget = (*GrokTarget)(nil)
