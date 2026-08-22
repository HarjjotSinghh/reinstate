package handoff

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"

	_ "modernc.org/sqlite"
)

const (
	openCodeExecutable      = "opencode"
	openCodeLaunchOperation = "handoff"
	// OpenCodeNewSessionFlag is the vendor option that starts a new session
	// seeded with an initial prompt. It is exported so the agent catalog can
	// declare exactly the flag this target launches, instead of the two drifting
	// apart.
	OpenCodeNewSessionFlag       = "--prompt"
	openCodeProjectionFileName   = "projection.md"
	openCodeShortBootstrapPrefix = "Reinstate structured handoff — not native resume. Read "
	openCodeShortBootstrapSuffix = " and continue from that briefing only. "
	// maxOpenCodeVerifyMessages bounds how far into one candidate session the
	// reconciliation reads looking for the first human turn.
	maxOpenCodeVerifyMessages = 32
	// maxOpenCodeVerifyCandidates bounds the reconciliation scan itself.
	maxOpenCodeVerifyCandidates = 10000
)

func init() {
	if err := RegisterTarget(NewOpenCodeTarget(nil)); err != nil {
		panic(err)
	}
}

// OpenCodeTarget starts a new OpenCode session for a structured handoff and
// reconciles the vendor-assigned session ID afterwards.
//
// It writes nothing into the OpenCode store, and that is a design conclusion
// rather than a convenience. OpenCode keeps its sessions in one embedded SQLite
// database whose schema spans twenty tables, including an event/event_sequence
// pair with per-aggregate sequence counters, a migration table, and foreign keys
// from message and part back to session. Materializing a destination session by
// inserting rows would mean reproducing invariants the vendor has not published,
// inside a file the vendor owns and is very likely holding open. It is also
// unnecessary: `opencode --prompt "<bootstrap>"` makes the vendor create the
// session, the first message and its parts itself — measured on macOS against
// OpenCode 1.18.21 — which is the same launch route Codex uses and what ADR 0003
// requires.
//
// The store is therefore only ever read, through the single read-only immutable
// DSN in internal/transcript, which is what stops SQLite creating `-wal` and
// `-shm` files beside the vendor's database.
type OpenCodeTarget struct {
	// Root overrides the resolved OpenCode data root (the directory holding
	// opencode.db). Empty resolves from XDG_DATA_HOME or the user home. Tests
	// must set this to a synthetic fixture root — never a real store.
	Root string
	// MaxArgvBytes overrides TargetCapabilities.MaxArgvBytes. Non-positive uses
	// DefaultMaxArgvBytes.
	MaxArgvBytes int
	// ForceCompat overrides installation detection (tests only).
	ForceCompat adapter.Compatibility
	// Inspect overrides the vendor compatibility probe (tests only).
	Inspect func(context.Context) adapter.Compatibility
}

// NewOpenCodeTarget returns an OpenCode destination target. opts may be nil.
func NewOpenCodeTarget(opts *OpenCodeTarget) *OpenCodeTarget {
	if opts == nil {
		return &OpenCodeTarget{}
	}
	cp := *opts
	return &cp
}

// Name returns the stable agent key "opencode".
func (t *OpenCodeTarget) Name() string { return sessionindex.AgentOpenCode }

// Capabilities reports OpenCode destination support.
//
// SupportsPinnedID is false, and that is measured rather than assumed:
// `opencode --session <unknown-id>` refuses with "Session not found" and creates
// nothing, so the destination ID cannot be chosen in advance and Verify
// reconciles it after launch.
func (t *OpenCodeTarget) Capabilities() TargetCapabilities {
	max := DefaultMaxArgvBytes
	if t != nil && t.MaxArgvBytes > 0 {
		max = t.MaxArgvBytes
	}
	return TargetCapabilities{
		Agent:                 sessionindex.AgentOpenCode,
		SupportsPinnedID:      false,
		SupportsInitialPrompt: true,
		MaxArgvBytes:          max,
		ContextCeiling:        0,
		AttachmentSupport:     false,
	}
}

// Compatible reports OpenCode install compatibility without reading any session
// body. It reuses the same bounded executable, version and layout probe that
// backs verified resume, so a destination is judged by exactly the rule a
// resume is.
func (t *OpenCodeTarget) Compatible(ctx context.Context) (adapter.Compatibility, error) {
	if t != nil {
		if forced := strings.TrimSpace(string(t.ForceCompat)); forced != "" {
			return t.ForceCompat, nil
		}
		if t.Inspect != nil {
			return t.Inspect(ctx), nil
		}
	}
	root := ""
	if t != nil {
		root = strings.TrimSpace(t.Root)
	}
	result := agentcheck.Inspect(ctx, sessionindex.AgentOpenCode, agentcheck.Options{Root: root})
	switch result.Status {
	case agentcheck.StatusSupported:
		return adapter.CompatibilitySupported, nil
	case agentcheck.StatusNotInstalled:
		return adapter.CompatibilityNotInstalled, nil
	default:
		// A recognizable install whose version is unknown, out of range, or
		// unreadable is uncertainty, not a clean bill of health.
		return adapter.CompatibilityUntested, nil
	}
}

// Plan builds argv `opencode --prompt "<bootstrap>"` with Dir set to the
// verified workspace. SessionID stays empty because the vendor assigns it.
// When the full bootstrap exceeds MaxArgvBytes, Plan falls back to a short
// bootstrap that references projection.md only.
func (t *OpenCodeTarget) Plan(c capsule.Capsule, p Policy) (DestinationPlan, capsule.Fidelity, error) {
	workspace := strings.TrimSpace(c.Workspace.Path)
	if workspace == "" {
		return DestinationPlan{}, capsule.Fidelity{}, errors.New("handoff: opencode plan requires verified workspace path")
	}

	_, _, fidelity := Apply(p, c.Conversation.Events)
	if fidelity.Mode == "" {
		fidelity.Mode = capsule.FidelityModeStructuredHandoff
	}

	caps := t.Capabilities()
	full := buildOpenCodeBootstrap(c, false)
	plan := DestinationPlan{
		Agent:      sessionindex.AgentOpenCode,
		Executable: openCodeExecutable,
		Args:       []string{OpenCodeNewSessionFlag, string(full)},
		Dir:        workspace,
		SessionID:  "",
		Bootstrap:  full,
	}
	if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil || argvUnsafeForLaunch(runtime.GOOS, full) {
		short := buildOpenCodeBootstrap(c, true)
		plan.Args = []string{OpenCodeNewSessionFlag, string(short)}
		plan.Bootstrap = short
		if err := ValidateDestinationArgv(plan, caps.MaxArgvBytes); err != nil {
			return DestinationPlan{}, fidelity, err
		}
	}
	return plan, fidelity, nil
}

// Materialize validates argv and writes nothing.
//
// Every other destination either writes no vendor file at all or records a
// workspace-trust entry so the vendor's directory-trust prompt does not block
// acknowledgement. OpenCode has no such prompt — measured: its TUI starts
// straight into a brand-new directory — so there is nothing to pre-accept, and
// inventing a write into a vendor-owned database to have something to
// materialize would be the opposite of the contract.
func (t *OpenCodeTarget) Materialize(_ context.Context, plan DestinationPlan) error {
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if len(plan.Files) > 0 {
		return errors.New("handoff: opencode destination never writes vendor files")
	}
	return nil
}

// Launch runs the planned OpenCode argv through an injectable LaunchRunner.
// Tests must supply a fake runner — never a real opencode process.
func (t *OpenCodeTarget) Launch(ctx context.Context, plan DestinationPlan, runner sessionindex.LaunchRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runner == nil {
		return errors.New("handoff: opencode launch requires a LaunchRunner")
	}
	if err := ValidateDestinationArgv(plan, t.Capabilities().MaxArgvBytes); err != nil {
		return err
	}
	if plan.Executable == "" || len(plan.Args) == 0 || strings.TrimSpace(plan.Dir) == "" {
		return errors.New("handoff: opencode launch plan is incomplete")
	}
	return runner.Run(ctx, sessionindex.LaunchPlan{
		Agent:      sessionindex.AgentOpenCode,
		SessionRef: "",
		Operation:  openCodeLaunchOperation,
		Executable: plan.Executable,
		Args:       append([]string(nil), plan.Args...),
		Dir:        plan.Dir,
	})
}

// Verify reads the OpenCode store after launch and reconciles the session ID.
//
// A candidate must have been created in the verified workspace at or after
// launchStart, and its first human turn must hash to the planned bootstrap.
// Exactly one match is resolved; zero is unresolved; more than one is
// ambiguous. It never picks arbitrarily.
//
// Freshness comes from the session row's own `time_created`, not from the
// database file's modification time. Every session in an embedded store shares
// one file mtime, so an mtime filter of the kind a per-session-file vendor can
// use would admit every session in the store here.
//
// KNOWN LIMITATION — reconciliation is expected to report `unresolved` for a
// session the vendor has only just created.
//
// OpenCode keeps its writes in a SQLite write-ahead log and does not checkpoint
// them into the database file when it exits. Measured on macOS against OpenCode
// 1.18.21: after a session was created and the vendor quit through its own UI,
// the database file was 4096 bytes with no `session` table at all and 543 KB
// sat in `opencode.db-wal`. `immutable=1` tells SQLite to ignore that log — the
// same property that stops Reinstate creating `-wal` and `-shm` beside a store
// it does not own — so the newest rows are simply not visible to any reader
// Reinstate is allowed to use.
//
// Nothing here works around that, because every workaround trades away the
// no-write invariant or copies a live log. `unresolved` is the honest answer
// and the contract allows it, but a maintainer should know that for this vendor
// it is the usual answer rather than the rare one, and that the same
// limitation makes recently written OpenCode sessions invisible to `rein
// sessions` long before any handoff is involved.
func (t *OpenCodeTarget) Verify(ctx context.Context, plan DestinationPlan, launchStart time.Time) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", VerifyUnresolved, err
	}
	workspace := strings.TrimSpace(plan.Dir)
	if workspace == "" {
		return "", VerifyUnresolved, errors.New("handoff: opencode verify requires plan.Dir")
	}
	if len(plan.Bootstrap) == 0 {
		return "", VerifyUnresolved, errors.New("handoff: opencode verify requires plan.Bootstrap")
	}

	path, err := t.databasePath()
	if err != nil {
		return "", VerifyUnresolved, err
	}
	if info, statErr := os.Stat(path); statErr != nil || !info.Mode().IsRegular() {
		// No store means the vendor created nothing this run. That is an
		// unresolved handoff, not a failure to report.
		return "", VerifyUnresolved, nil
	}

	db, err := sql.Open("sqlite", transcript.OpenCodeReadOnlyDSN(path))
	if err != nil {
		return "", VerifyUnresolved, nil
	}
	defer func() { _ = db.Close() }()

	candidates, err := openCodeCandidateSessions(ctx, db, workspace, launchStart)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", VerifyUnresolved, ctxErr
		}
		// A store this reader cannot query is an unresolved reconciliation, not
		// a failed handoff. The destination session may well have been created
		// — a store whose schema still lives entirely in an un-checkpointed
		// write-ahead log has no tables at all through an immutable handle —
		// and reporting that as a command failure would tell the operator the
		// handoff did not happen when it did.
		return "", VerifyUnresolved, nil
	}

	wantHash := sha256Hex(plan.Bootstrap)
	var matches []string
	for _, id := range candidates {
		if err := ctx.Err(); err != nil {
			return "", VerifyUnresolved, err
		}
		first, err := openCodeFirstUserMessage(ctx, db, id)
		if err != nil || first == "" {
			continue
		}
		if sha256Hex([]byte(first)) != wantHash {
			continue
		}
		matches = append(matches, id)
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

func (t *OpenCodeTarget) databasePath() (string, error) {
	root := ""
	if t != nil {
		root = strings.TrimSpace(t.Root)
	}
	if root == "" {
		resolved, err := transcript.ResolveOpenCodeDataRoot(nil, nil)
		if err != nil {
			return "", err
		}
		root = resolved
	}
	if root == "" {
		return "", errors.New("handoff: opencode data root is unavailable")
	}
	return filepath.Join(root, transcript.OpenCodeDatabaseName), nil
}

// openCodeCandidateSessions returns sessions in workspace created at or after
// launchStart, newest first.
func openCodeCandidateSessions(
	ctx context.Context, db *sql.DB, workspace string, launchStart time.Time,
) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, COALESCE(directory, ''), COALESCE(time_created, 0)
  FROM session
 ORDER BY time_created DESC, id DESC
 LIMIT ?`, maxOpenCodeVerifyCandidates)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var (
			id, directory string
			created       int64
		)
		if err := rows.Scan(&id, &directory, &created); err != nil {
			return nil, err
		}
		if strings.TrimSpace(id) == "" {
			continue
		}
		if !openCodeWorkspaceEqual(directory, workspace) {
			continue
		}
		if !openCodeCreatedAtOrAfter(created, launchStart) {
			continue
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// openCodeFirstUserMessage returns the text of the session's first human turn.
//
// The message row carries role and timing; the text lives in the part rows, so
// both are read. A row whose shape the reader does not recognize is skipped,
// never guessed at.
func openCodeFirstUserMessage(ctx context.Context, db *sql.DB, sessionID string) (string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, data
  FROM message
 WHERE session_id = ?
 ORDER BY time_created, id
 LIMIT ?`, sessionID, maxOpenCodeVerifyMessages)
	if err != nil {
		return "", err
	}
	messageID := ""
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			_ = rows.Close()
			return "", err
		}
		var envelope struct {
			Role string `json:"role"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(envelope.Role), "user") {
			messageID = id
			break
		}
	}
	closeErr := rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if messageID == "" {
		return "", nil
	}
	return openCodeMessageText(ctx, db, messageID)
}

func openCodeMessageText(ctx context.Context, db *sql.DB, messageID string) (string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT data FROM part WHERE message_id = ? ORDER BY id`, messageID)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()

	var text strings.Builder
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return "", err
		}
		var part struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(data, &part) != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(part.Type), "text") {
			text.WriteString(part.Text)
		}
	}
	return text.String(), rows.Err()
}

func openCodeWorkspaceEqual(recorded, want string) bool {
	recorded = strings.TrimSpace(recorded)
	want = strings.TrimSpace(want)
	if recorded == "" || want == "" {
		return false
	}
	return recorded == want || filepath.Clean(recorded) == filepath.Clean(want)
}

// openCodeCreatedAtOrAfter accepts OpenCode's millisecond timestamps, and the
// second-resolution encoding an older store may carry, without treating a
// second-resolution value as an instant in 1970.
func openCodeCreatedAtOrAfter(created int64, launchStart time.Time) bool {
	if created <= 0 {
		return false
	}
	stamp := time.UnixMilli(created)
	if created <= 1e12 {
		stamp = time.Unix(created, 0)
	}
	// Launch and the vendor's own clock are compared at whole-second
	// resolution, because a store that records seconds cannot express anything
	// finer and would otherwise lose a session created in the same second the
	// launch began.
	return !stamp.Before(launchStart.Truncate(time.Second))
}

func buildOpenCodeBootstrap(c capsule.Capsule, shortOnly bool) []byte {
	projection := openCodeProjectionFileName
	short := []byte(openCodeShortBootstrapPrefix + projection + openCodeShortBootstrapSuffix + firstReplyAckOneLine())
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
	b.WriteString(" for the full destination briefing.\n")
	b.WriteString(acknowledgementBlock())
	full := []byte(b.String())
	if len(full) > BootstrapMaxBytes {
		full = append([]byte(nil), full[:BootstrapMaxBytes]...)
	}
	if len(full) == 0 {
		return short
	}
	return full
}

var _ HandoffTarget = (*OpenCodeTarget)(nil)
