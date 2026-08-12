package handoff

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	codexadapter "github.com/HarjjotSinghh/reinstate/internal/adapter/codex"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	codexExecutable           = "codex"
	codexLaunchOperation      = "handoff"
	codexProjectionFileName   = "projection.md"
	codexShortBootstrapPrefix = "Reinstate structured handoff — not native resume. Read "
	codexShortBootstrapSuffix = " and continue from that briefing only. Acknowledge before any mutation."
)

func init() {
	if err := RegisterTarget(NewCodexTarget(nil)); err != nil {
		panic(err)
	}
}

// CodexTarget launches a new Codex CLI session via argv bootstrap and
// reconciles the vendor-assigned session ID after launch (ADR 0003).
//
// It never writes into ~/.codex or any other vendor-internal tree.
type CodexTarget struct {
	// Root overrides Codex home ($CODEX_HOME). Empty uses CodexSource resolution.
	// Tests must set this to a synthetic fixture root — never a real ~/.codex.
	Root string
	// MaxArgvBytes overrides TargetCapabilities.MaxArgvBytes. Non-positive uses
	// DefaultMaxArgvBytes (R6 practical Windows-safe ceiling).
	MaxArgvBytes int
	// ForceCompat overrides adapter detection (tests only).
	ForceCompat adapter.Compatibility
}

// NewCodexTarget returns a Codex destination target. opts may be nil.
func NewCodexTarget(opts *CodexTarget) *CodexTarget {
	if opts == nil {
		return &CodexTarget{}
	}
	cp := *opts
	return &cp
}

// Name returns the stable agent key "codex".
func (t *CodexTarget) Name() string { return sessionindex.AgentCodex }

// Capabilities reports Codex destination support. SupportsPinnedID is false:
// Codex assigns the session ID; Verify reconciles it after launch.
func (t *CodexTarget) Capabilities() TargetCapabilities {
	max := DefaultMaxArgvBytes
	if t != nil && t.MaxArgvBytes > 0 {
		max = t.MaxArgvBytes
	}
	return TargetCapabilities{
		Agent:                 sessionindex.AgentCodex,
		SupportsPinnedID:      false,
		SupportsInitialPrompt: true,
		MaxArgvBytes:          max,
		ContextCeiling:        0,
		AttachmentSupport:     false,
	}
}

// Compatible reports Codex install compatibility without reading session bodies.
func (t *CodexTarget) Compatible(ctx context.Context) (adapter.Compatibility, error) {
	root := ""
	force := adapter.Compatibility("")
	if t != nil {
		root = t.Root
		force = t.ForceCompat
	}
	a := &codexadapter.Adapter{Root: root, ForceCompat: force}
	_, compat, err := a.Detect(ctx)
	return compat, err
}

// Plan builds argv `codex "<bootstrap>"` with Dir set to the verified workspace.
// SessionID stays empty (vendor-assigned). When the full bootstrap exceeds
// MaxArgvBytes, Plan falls back to a short bootstrap that references
// projection.md only (R6).
func (t *CodexTarget) Plan(c capsule.Capsule, p Policy) (DestinationPlan, capsule.Fidelity, error) {
	workspace := strings.TrimSpace(c.Workspace.Path)
	if workspace == "" {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: codex plan requires verified workspace path")
	}

	_, _, fidelity := Apply(p, c.Conversation.Events)
	if fidelity.Mode == "" {
		fidelity.Mode = capsule.FidelityModeStructuredHandoff
	}

	caps := t.Capabilities()
	full := buildCodexBootstrap(c, false)
	plan := DestinationPlan{
		Agent:      sessionindex.AgentCodex,
		Executable: codexExecutable,
		Args:       []string{string(full)},
		Dir:        workspace,
		SessionID:  "",
		Bootstrap:  full,
	}
	if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
		short := buildCodexBootstrap(c, true)
		plan.Args = []string{string(short)}
		plan.Bootstrap = short
		if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
			return DestinationPlan{}, fidelity, err
		}
	}
	return plan, fidelity, nil
}

// Materialize validates argv budget. Codex destination materialization never
// writes vendor-internal files; planned Files (if any) are written owner-only
// outside the Codex tree via WritePlannedFiles.
func (t *CodexTarget) Materialize(_ context.Context, plan DestinationPlan) error {
	return ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes)
}

// Launch runs the planned Codex argv through an injectable LaunchRunner.
// Tests must supply a fake runner — never a real codex process.
func (t *CodexTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runner == nil {
		return errors.New("handoff: codex launch requires a LaunchRunner")
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if plan.Executable == "" || len(plan.Args) == 0 || strings.TrimSpace(plan.Dir) == "" {
		return errors.New("handoff: codex launch plan is incomplete")
	}
	return runner.Run(ctx, sessionindex.LaunchPlan{
		Agent:      sessionindex.AgentCodex,
		SessionRef: "",
		Operation:  codexLaunchOperation,
		Executable: plan.Executable,
		Args:       append([]string(nil), plan.Args...),
		Dir:        plan.Dir,
	})
}

// Verify rescans the Codex source after launch and reconciles the session ID.
// Candidates must match verified workspace cwd, mtime >= launchStart, and a
// first-user-message SHA-256 equal to the planned bootstrap. Exactly one match
// → resolved; zero → unresolved; more than one → ambiguous. Never picks
// arbitrarily.
func (t *CodexTarget) Verify(ctx context.Context, plan DestinationPlan, launchStart time.Time) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", VerifyUnresolved, err
	}
	workspace := strings.TrimSpace(plan.Dir)
	if workspace == "" {
		return "", VerifyUnresolved, errors.New("handoff: codex verify requires plan.Dir")
	}
	if len(plan.Bootstrap) == 0 {
		return "", VerifyUnresolved, errors.New("handoff: codex verify requires plan.Bootstrap")
	}

	root := ""
	if t != nil {
		root = t.Root
	}
	result, err := sessionindex.NewCodexSource(root).Scan(ctx)
	if err != nil {
		return "", VerifyUnresolved, err
	}

	wantHash := sha256Hex(plan.Bootstrap)
	var matches []string
	for _, rec := range result.Records {
		if err := ctx.Err(); err != nil {
			return "", VerifyUnresolved, err
		}
		if !codexWorkspaceEqual(rec.Workspace, workspace) {
			continue
		}
		if !codexMtimeAtOrAfter(rec.SourceModTime, launchStart) {
			continue
		}
		first, err := codexFirstUserMessage(rec.SourcePath)
		if err != nil || first == "" {
			continue
		}
		if sha256Hex([]byte(first)) != wantHash {
			continue
		}
		matches = append(matches, rec.ID)
	}

	switch len(matches) {
	case 0:
		return "", VerifyUnresolved, nil
	case 1:
		return matches[0], VerifyResolved, nil
	default:
		return "", VerifyAmbiguous, nil
	}
}

func buildCodexBootstrap(c capsule.Capsule, shortOnly bool) []byte {
	projection := codexProjectionFileName
	short := []byte(codexShortBootstrapPrefix + projection + codexShortBootstrapSuffix)
	if shortOnly {
		return short
	}

	var b strings.Builder
	b.WriteString("Reinstate structured handoff — not native resume.\n")
	if goal := strings.TrimSpace(c.Task.Goal.Text); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteByte('\n')
	}
	if intent := strings.TrimSpace(c.Task.LatestUserIntent.Text); intent != "" {
		b.WriteString("Latest request: ")
		b.WriteString(intent)
		b.WriteByte('\n')
	}
	b.WriteString("Read ")
	b.WriteString(projection)
	b.WriteString(" for the full destination briefing. Acknowledge before any mutation.")
	full := []byte(b.String())
	if len(full) > BootstrapMaxBytes {
		full = append([]byte(nil), full[:BootstrapMaxBytes]...)
	}
	if len(full) == 0 {
		return short
	}
	return full
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func codexWorkspaceEqual(recorded, want string) bool {
	recorded = strings.TrimSpace(recorded)
	want = strings.TrimSpace(want)
	if recorded == "" || want == "" {
		return false
	}
	return recorded == want || filepath.Clean(recorded) == filepath.Clean(want)
}

func codexMtimeAtOrAfter(sourceModTimeNano int64, launchStart time.Time) bool {
	if sourceModTimeNano <= 0 {
		return false
	}
	mtime := time.Unix(0, sourceModTimeNano)
	return !mtime.Before(launchStart)
}

// codexFirstUserMessage returns the first human user prompt in a Codex rollout.
// Prefer event_msg/user_message over response_item duplicates (same as the index).
func codexFirstUserMessage(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	var (
		direct   string
		fallback string
	)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), sessionindex.MaxJSONLineBytes)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		payload, _ := event["payload"].(map[string]any)
		msg, ok := codexUserMessageText(event, payload)
		if !ok || msg == "" {
			continue
		}
		eventType := strings.ToLower(stringField(event, "type"))
		payloadType := strings.ToLower(stringField(payload, "type"))
		if eventType == "event_msg" && payloadType == "user_message" {
			if direct == "" {
				direct = msg
			}
			continue
		}
		if fallback == "" {
			fallback = msg
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if direct != "" {
		return direct, nil
	}
	return fallback, nil
}

func codexUserMessageText(event, payload map[string]any) (string, bool) {
	eventType := strings.ToLower(stringField(event, "type"))
	payloadType := strings.ToLower(stringField(payload, "type"))
	switch eventType {
	case "event_msg":
		if payloadType == "user_message" {
			if msg := stringField(payload, "message", "text"); msg != "" {
				return msg, true
			}
			return extractCodexTextContent(payload["content"]), true
		}
	case "response_item":
		if payloadType == "message" && strings.EqualFold(stringField(payload, "role"), "user") {
			return extractCodexTextContent(payload["content"]), true
		}
	case "message":
		if strings.EqualFold(stringField(event, "role"), "user") {
			return extractCodexTextContent(event["content"]), true
		}
	}
	if strings.EqualFold(stringField(event, "role"), "user") {
		return extractCodexTextContent(event["content"]), true
	}
	return "", false
}

func extractCodexTextContent(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t := stringField(m, "text"); t != "" {
				b.WriteString(t)
			}
		}
		return b.String()
	default:
		return ""
	}
}

func stringField(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

var _ HandoffTarget = (*CodexTarget)(nil)
