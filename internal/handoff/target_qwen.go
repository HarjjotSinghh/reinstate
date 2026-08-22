package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	qwenTargetName     = sessionindex.AgentQwen
	qwenExecutable     = "qwen"
	qwenSessionIDSalt  = "reinstate:qwen-session:"
	qwenProjectsDir    = "projects"
	qwenChatsDir       = "chats"
	qwenSessionIDFlag  = "--session-id"
	qwenInitialPrompt  = "--prompt-interactive"
	qwenSessionFileExt = ".jsonl"
)

// qwenSessionIDPattern is the vendor's own accepted shape. `--session-id` with
// anything else is rejected as a usage error, so a plan that cannot produce a
// lowercase UUID must fail before launch rather than after.
var qwenSessionIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ErrQwenSessionIDCollision means the capsule-derived destination UUID is
// already present in the indexed Qwen sessions.
//
// Unlike Claude, this is belt and braces rather than the only guard: Qwen
// refuses a duplicate itself — "Session Id … already exists (active or
// archived). Delete or unarchive it first." — measured on macOS 2026-08-22. The
// index check exists so the refusal happens before a process is spawned.
var ErrQwenSessionIDCollision = errors.New("handoff: Qwen --session-id collided with indexed sessions; refuse rather than reuse")

// QwenSessionExists reports whether a Qwen session ID is already indexed.
// Nil means no pre-launch collision check is performed.
type QwenSessionExists func(ctx context.Context, sessionID string) (bool, error)

// QwenTarget is the Qwen Code handoff destination.
//
// It launches `qwen --session-id <uuid> --prompt-interactive "<bootstrap>"` in
// the verified workspace: a **new** Qwen session seeded with a briefing, never
// a cross-agent resume and never a reconstruction of the source thread. It
// writes no file under the Qwen home.
type QwenTarget struct {
	// Root overrides the Qwen home ($QWEN_HOME). Tests must set this to a
	// synthetic fixture root — never a real ~/.qwen.
	Root string
	// SessionExists checks the local index for an existing Qwen session ID.
	// Nil skips the pre-launch check; the vendor still refuses a duplicate.
	SessionExists QwenSessionExists
	// NewSessionID overrides deterministic capsule-derived UUID generation in tests.
	NewSessionID func() (string, error)
	// Bootstrap builds the argv prompt. Nil uses a bounded stub; the pipeline
	// replaces it with RenderBootstrap.
	Bootstrap func(capsule.Capsule, Policy) ([]byte, error)
	// MaxArgvBytes overrides TargetCapabilities.MaxArgvBytes.
	MaxArgvBytes int
	// ForceCompat overrides install detection (tests only), so no unit test
	// depends on whether a contributor happens to have Qwen installed.
	ForceCompat adapter.Compatibility
}

func init() {
	if err := RegisterTarget(&QwenTarget{}); err != nil {
		panic(err)
	}
}

// Name returns the destination agent name.
func (t *QwenTarget) Name() string { return qwenTargetName }

// Capabilities reports Qwen Code destination support.
//
// SupportsPinnedID is true: `--session-id <uuid>` was observed creating the
// session at exactly that id, so Verify never has to guess which session the
// launch produced. AttachmentSupport is false — the capsule never re-embeds
// vendor attachments, so there is nothing for a destination to accept.
func (t *QwenTarget) Capabilities() TargetCapabilities {
	max := DefaultMaxArgvBytes
	if t != nil && t.MaxArgvBytes > 0 {
		max = t.MaxArgvBytes
	}
	return TargetCapabilities{
		Agent:                 qwenTargetName,
		SupportsPinnedID:      true,
		SupportsInitialPrompt: true,
		MaxArgvBytes:          max,
		ContextCeiling:        0, // no vendor-published harness token ceiling
		AttachmentSupport:     false,
	}
}

// Compatible reports Qwen install compatibility.
//
// There is no Qwen sync adapter to Detect through — Qwen is not a T5 agent — so
// this reuses the same bounded agentcheck probe that backs `rein inspect`:
// executable presence, layout recognition, and the descriptor's version range.
// It reads no session body.
func (t *QwenTarget) Compatible(ctx context.Context) (adapter.Compatibility, error) {
	if t != nil && t.ForceCompat != "" {
		return t.ForceCompat, nil
	}
	root := ""
	if t != nil {
		root = strings.TrimSpace(t.Root)
	}
	result := agentcheck.Inspect(ctx, qwenTargetName, agentcheck.Options{Root: root})
	switch result.Status {
	case agentcheck.StatusSupported:
		return adapter.CompatibilitySupported, nil
	case agentcheck.StatusNotInstalled:
		return adapter.CompatibilityNotInstalled, nil
	default:
		// A recognizable but unverified install, or a probe that failed, is
		// UNTESTED. A destination launch creates state in another vendor's
		// product, so uncertainty never passes as supported.
		return adapter.CompatibilityUntested, nil
	}
}

// Plan derives a non-colliding UUID v4 and builds the launch argv.
func (t *QwenTarget) Plan(c capsule.Capsule, policy Policy) (DestinationPlan, capsule.Fidelity, error) {
	workspace := strings.TrimSpace(c.Workspace.Path)
	if workspace == "" {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Qwen destination requires a verified workspace path")
	}

	bootstrap, err := t.renderBootstrap(c, policy)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}
	if len(bootstrap) == 0 {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Qwen bootstrap prompt is empty")
	}
	if len(bootstrap) > BootstrapMaxBytes {
		bootstrap = append([]byte(nil), bootstrap[:BootstrapMaxBytes]...)
	}

	sessionID, err := t.allocateSessionID(context.Background(), c)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}

	_, _, fidelity := Apply(policy, c.Conversation.Events)
	if fidelity.Mode == "" {
		fidelity.Mode = capsule.FidelityModeStructuredHandoff
	}
	plan := DestinationPlan{
		Agent:      qwenTargetName,
		Executable: qwenExecutable,
		Args:       []string{qwenSessionIDFlag, sessionID, qwenInitialPrompt, string(bootstrap)},
		Dir:        workspace,
		Files:      nil, // never write vendor-internal Qwen files
		SessionID:  sessionID,
		Bootstrap:  bootstrap,
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return DestinationPlan{}, fidelity, err
	}
	if argvUnsafeForLaunch(runtime.GOOS, bootstrap) {
		return DestinationPlan{}, fidelity, errors.New("handoff: Qwen bootstrap is not safe to pass as argv on this platform")
	}
	return plan, fidelity, nil
}

// Materialize validates argv and writes the planned handoff files. It writes
// nothing under the Qwen home: Qwen did not prompt for workspace trust on a
// first launch in a fresh root, so there is no trust record to pre-accept, and
// inventing one would be a vendor-internal write for no observed reason.
func (t *QwenTarget) Materialize(_ context.Context, plan DestinationPlan) error {
	return WritePlannedFiles(plan, t.Capabilities().MaxArgvBytes, nil)
}

// Launch runs the destination through the injected LaunchRunner, so TTY,
// executable identity, and workspace identity guards apply unchanged.
func (t *QwenTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if strings.TrimSpace(plan.Executable) == "" || len(plan.Args) == 0 {
		return errors.New("handoff: Qwen launch plan is incomplete")
	}
	if strings.TrimSpace(plan.Dir) == "" {
		return fmt.Errorf("%w: Qwen launch workspace is missing", sessionindex.ErrWorkspaceUnavailable)
	}
	launch := sessionindex.LaunchPlan{
		Agent:      qwenTargetName,
		SessionRef: sessionindex.CompositeReference(qwenTargetName, plan.SessionID),
		Operation:  sessionindex.OperationHandoff,
		Executable: plan.Executable,
		Args:       append([]string(nil), plan.Args...),
		Dir:        plan.Dir,
	}
	return sessionindex.RunLaunch(ctx, launch, runner)
}

// Verify returns the pinned session ID when the conversation file exists under
// this device's Qwen project bucket for plan.Dir.
//
// The bucket is always recomputed from the destination workspace. A source
// device's directory name is never reused: the vendor lower-cases the path
// before sanitising it on Windows and not elsewhere, so the same project has
// two different bucket names across a Windows/macOS pair.
func (t *QwenTarget) Verify(_ context.Context, plan DestinationPlan, _ time.Time) (string, string, error) {
	id := strings.TrimSpace(plan.SessionID)
	if id == "" {
		return "", VerifyUnresolved, nil
	}
	workspace := strings.TrimSpace(plan.Dir)
	if workspace == "" {
		return "", VerifyUnresolved, errors.New("handoff: Qwen verify requires destination workspace")
	}
	root := t.qwenRoot()
	if root == "" {
		return "", VerifyUnresolved, nil
	}
	path := filepath.Join(root, qwenProjectsDir, QwenProjectKey(workspace), qwenChatsDir, id+qwenSessionFileExt)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", VerifyUnresolved, nil
		}
		return "", VerifyUnresolved, fmt.Errorf("handoff: inspect Qwen session file: %w", err)
	}
	if info.IsDir() {
		return "", VerifyUnresolved, nil
	}
	return id, VerifyResolved, nil
}

func (t *QwenTarget) renderBootstrap(c capsule.Capsule, policy Policy) ([]byte, error) {
	if t != nil && t.Bootstrap != nil {
		return t.Bootstrap(c, policy)
	}
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		id = "unknown"
	}
	msg := "structured handoff — a new Qwen Code session, not native resume. " +
		"Continue capsule " + id + ". Read projection.md in the handoff directory for the full briefing."
	return []byte(msg), nil
}

func (t *QwenTarget) allocateSessionID(ctx context.Context, c capsule.Capsule) (string, error) {
	id, err := qwenSessionID(c)
	if t != nil && t.NewSessionID != nil {
		id, err = t.NewSessionID()
	}
	if err != nil {
		return "", fmt.Errorf("handoff: generate Qwen session id: %w", err)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("handoff: generated Qwen session id is empty")
	}
	if !qwenSessionIDPattern.MatchString(id) {
		// `qwen --session-id not-a-uuid` is a usage error, so refusing here
		// turns a launch-time failure into a plan-time one.
		return "", fmt.Errorf("handoff: Qwen session id %q is not a lowercase UUID", id)
	}
	exists, err := t.sessionExists(ctx, id)
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrQwenSessionIDCollision
	}
	return id, nil
}

func (t *QwenTarget) sessionExists(ctx context.Context, sessionID string) (bool, error) {
	if t == nil || t.SessionExists == nil {
		return false, nil
	}
	return t.SessionExists(ctx, sessionID)
}

func (t *QwenTarget) qwenRoot() string {
	if t != nil {
		if root := strings.TrimSpace(t.Root); root != "" {
			return filepath.Clean(root)
		}
	}
	if configured := strings.TrimSpace(os.Getenv("QWEN_HOME")); configured != "" {
		return filepath.Clean(configured)
	}
	// Fail closed without an explicit root or env override so no unit test
	// reaches a contributor's real ~/.qwen.
	return ""
}

// qwenSessionID returns a deterministic UUIDv4 derived from the capsule content
// ID, so dry-run and execute plan identical argv. The salt is Qwen-specific:
// handing one capsule to two destinations must not derive one shared id.
func qwenSessionID(c capsule.Capsule) (string, error) {
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		var err error
		id, err = capsule.ComputeID(c)
		if err != nil {
			return "", err
		}
	}
	b := sha256.Sum256([]byte(qwenSessionIDSalt + id))
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

// QwenProjectKey reproduces the vendor's sanitizeCwd: every byte that is not
// ASCII alphanumeric becomes '-', after lower-casing the whole path on Windows
// and only on Windows. There is no length cap and no hash suffix.
func QwenProjectKey(workspace string) string {
	path := filepath.Clean(workspace)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	var key strings.Builder
	key.Grow(len(path))
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			key.WriteByte(c)
		default:
			key.WriteByte('-')
		}
	}
	return key.String()
}
