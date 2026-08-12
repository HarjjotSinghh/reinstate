package handoff

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	claudeTargetName = "claude"
	// claudeSessionIDAttempts is how many crypto/rand UUIDs Plan may try before
	// refusing (R5). Vendor collision-on-reuse behavior is unpublished; never
	// guess silent overwrite.
	claudeSessionIDAttempts = 8
)

// ErrClaudeSessionIDCollision means Plan could not allocate a UUID that is
// absent from the indexed Claude sessions after claudeSessionIDAttempts tries.
var ErrClaudeSessionIDCollision = errors.New("handoff: Claude --session-id collided with indexed sessions; refuse rather than overwrite")

// ClaudeSessionExists reports whether a Claude session ID is already present in
// the local session index. Callers (CLI) should wire this to the index; nil
// means no collision check is performed.
type ClaudeSessionExists func(ctx context.Context, sessionID string) (bool, error)

// ClaudeTarget is the Claude Code handoff destination (ADR 0003 launch route).
//
// It launches `claude --session-id <uuid-v4> "<bootstrap>"` in the verified
// workspace and never writes vendor-internal files under ~/.claude.
type ClaudeTarget struct {
	// ConfigDir overrides Claude's config root (tests / CLAUDE_CONFIG_DIR).
	// Empty uses Detect's normal resolution. Never point this at a real
	// contributor ~/.claude from unit tests.
	ConfigDir string
	// SessionExists checks the local index for an existing Claude session ID.
	// When nil, Plan does not treat any UUID as colliding (callers must wire
	// the index before production Plan).
	SessionExists ClaudeSessionExists
	// NewSessionID generates a UUID v4. Nil uses crypto/rand.
	NewSessionID func() (string, error)
	// Bootstrap builds the argv prompt. Nil uses a bounded stub; WP-18 replaces
	// this with RenderBootstrap when wired by the CLI.
	Bootstrap func(capsule.Capsule, Policy) ([]byte, error)
}

func init() {
	if err := RegisterTarget(&ClaudeTarget{}); err != nil {
		panic(err)
	}
}

// Name returns the destination agent name.
func (t *ClaudeTarget) Name() string { return claudeTargetName }

// Capabilities reports Claude Code destination support for pinned session IDs.
func (t *ClaudeTarget) Capabilities() TargetCapabilities {
	return TargetCapabilities{
		Agent:                 claudeTargetName,
		SupportsPinnedID:      true,
		SupportsInitialPrompt: true,
		MaxArgvBytes:          DefaultMaxArgvBytes,
		ContextCeiling:        0, // R7 omitted
		AttachmentSupport:     true,
	}
}

// Compatible probes Claude Code install state without reading a real home
// directory when ConfigDir is set.
func (t *ClaudeTarget) Compatible(ctx context.Context) (adapter.Compatibility, error) {
	a := &claude.Adapter{Root: strings.TrimSpace(t.ConfigDir)}
	_, compat, err := a.Detect(ctx)
	return compat, err
}

// Plan allocates a non-colliding UUID v4 and builds the ADR 0003 argv.
func (t *ClaudeTarget) Plan(c capsule.Capsule, policy Policy) (DestinationPlan, capsule.Fidelity, error) {
	workspace := strings.TrimSpace(c.Workspace.Path)
	if workspace == "" {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Claude destination requires a verified workspace path")
	}

	bootstrap, err := t.renderBootstrap(c, policy)
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}
	if len(bootstrap) == 0 {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: Claude bootstrap prompt is empty")
	}
	if len(bootstrap) > BootstrapMaxBytes {
		bootstrap = append([]byte(nil), bootstrap[:BootstrapMaxBytes]...)
	}

	sessionID, err := t.allocateSessionID(context.Background())
	if err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}

	_, _, fidelity := Apply(policy, c.Conversation.Events)
	plan := DestinationPlan{
		Agent:      claudeTargetName,
		Executable: "claude",
		Args:       []string{"--session-id", sessionID, string(bootstrap)},
		Dir:        workspace,
		Files:      nil, // never write vendor-internal Claude files
		SessionID:  sessionID,
		Bootstrap:  bootstrap,
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return DestinationPlan{}, capsule.Fidelity{}, err
	}
	return plan, fidelity, nil
}

// Materialize validates argv and writes only PlannedFile entries (none for the
// Claude launch route — capsule/projection live under $REINSTATE_HOME/handoffs).
func (t *ClaudeTarget) Materialize(_ context.Context, plan DestinationPlan) error {
	return WritePlannedFiles(plan, t.Capabilities().MaxArgvBytes, nil)
}

// Launch runs the destination via the injected LaunchRunner. Production passes
// sessionindex.ExecLaunchRunner so TTY, executable identity, and workspace
// identity guards apply unchanged.
func (t *ClaudeTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if strings.TrimSpace(plan.Executable) == "" || len(plan.Args) == 0 {
		return errors.New("handoff: Claude launch plan is incomplete")
	}
	if strings.TrimSpace(plan.Dir) == "" {
		return fmt.Errorf("%w: Claude launch workspace is missing", sessionindex.ErrWorkspaceUnavailable)
	}
	launch := sessionindex.LaunchPlan{
		Agent:      claudeTargetName,
		SessionRef: sessionindex.CompositeReference(claudeTargetName, plan.SessionID),
		Operation:  sessionindex.OperationHandoff,
		Executable: plan.Executable,
		Args:       append([]string(nil), plan.Args...),
		Dir:        plan.Dir,
	}
	return sessionindex.RunLaunch(ctx, launch, runner)
}

// Verify returns the pinned session ID when the session JSONL exists under this
// device's Claude project key for plan.Dir. It never reuses a source-device key.
func (t *ClaudeTarget) Verify(_ context.Context, plan DestinationPlan, _ time.Time) (string, string, error) {
	id := strings.TrimSpace(plan.SessionID)
	if id == "" {
		return "", VerifyUnresolved, nil
	}
	workspace := strings.TrimSpace(plan.Dir)
	if workspace == "" {
		return "", VerifyUnresolved, errors.New("handoff: Claude verify requires destination workspace")
	}
	root, err := t.claudeRoot()
	if err != nil {
		return "", VerifyUnresolved, err
	}
	if root == "" {
		return "", VerifyUnresolved, nil
	}
	key := claudeProjectKey(workspace)
	path := filepath.Join(root, "projects", key, id+".jsonl")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", VerifyUnresolved, nil
		}
		return "", VerifyUnresolved, fmt.Errorf("handoff: inspect Claude session file: %w", err)
	}
	if info.IsDir() {
		return "", VerifyUnresolved, nil
	}
	return id, VerifyResolved, nil
}

func (t *ClaudeTarget) renderBootstrap(c capsule.Capsule, policy Policy) ([]byte, error) {
	if t != nil && t.Bootstrap != nil {
		return t.Bootstrap(c, policy)
	}
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		id = "unknown"
	}
	msg := "structured handoff — a new Claude Code session, not native resume. " +
		"Continue capsule " + id + ". Read projection.md in the handoff directory for the full briefing."
	return []byte(msg), nil
}

func (t *ClaudeTarget) allocateSessionID(ctx context.Context) (string, error) {
	newID := newClaudeSessionID
	if t != nil && t.NewSessionID != nil {
		newID = t.NewSessionID
	}
	for attempt := 0; attempt < claudeSessionIDAttempts; attempt++ {
		id, err := newID()
		if err != nil {
			return "", fmt.Errorf("handoff: generate Claude session id: %w", err)
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return "", errors.New("handoff: generated Claude session id is empty")
		}
		exists, err := t.sessionExists(ctx, id)
		if err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w after %d attempts (R5)", ErrClaudeSessionIDCollision, claudeSessionIDAttempts)
}

func (t *ClaudeTarget) sessionExists(ctx context.Context, sessionID string) (bool, error) {
	if t == nil || t.SessionExists == nil {
		return false, nil
	}
	return t.SessionExists(ctx, sessionID)
}

func (t *ClaudeTarget) claudeRoot() (string, error) {
	if t != nil {
		if root := strings.TrimSpace(t.ConfigDir); root != "" {
			return filepath.Clean(root), nil
		}
	}
	if configured := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); configured != "" {
		return filepath.Clean(configured), nil
	}
	// Compatible/Detect may still resolve a home root; Verify stays fail-closed
	// without an explicit test root or env override so unit tests never touch
	// a real ~/.claude.
	return "", nil
}

// newClaudeSessionID returns a UUID v4 from crypto/rand.
func newClaudeSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	var node uint64
	for _, v := range b[10:] {
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

// claudeProjectKey reproduces Phase 1 Claude Code project-directory encoding
// from this device's absolute workspace path (see adapter/claude).
func claudeProjectKey(projectPath string) string {
	canonicalPath := filepath.Clean(projectPath)
	if resolved, err := filepath.EvalSymlinks(canonicalPath); err == nil {
		canonicalPath = resolved
	}
	units := utf16.Encode([]rune(canonicalPath))
	var encoded strings.Builder
	encoded.Grow(len(units))
	for _, unit := range units {
		if unit >= 'a' && unit <= 'z' ||
			unit >= 'A' && unit <= 'Z' ||
			unit >= '0' && unit <= '9' {
			encoded.WriteByte(byte(unit))
		} else {
			encoded.WriteByte('-')
		}
	}
	key := encoded.String()
	if len(key) <= 200 {
		return key
	}
	return key[:200] + "-" + strconv.FormatInt(absJSStringHash(units), 36)
}

func absJSStringHash(units []uint16) int64 {
	var hash int32
	for _, unit := range units {
		hash = int32(int64(hash)*31 + int64(unit))
	}
	value := int64(hash)
	if value < 0 {
		return -value
	}
	return value
}
