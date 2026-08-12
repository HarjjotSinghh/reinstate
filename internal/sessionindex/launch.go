package sessionindex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

const (
	OperationResume = "resume"
	OperationFork   = "fork"
	// OperationHandoff launches a destination agent for a structured handoff
	// (Phase 4). Reuses the same ExecLaunchRunner TTY and identity guards.
	OperationHandoff = "handoff"
)

var (
	ErrNativeActionUnsupported = errors.New("native session action is unsupported")
	ErrExecutableNotFound      = errors.New("native agent executable is unavailable")
	ErrWorkspaceUnavailable    = errors.New("recorded session workspace is unavailable")
	ErrLaunchBoundaryChanged   = errors.New("native launch target changed at the execution boundary")
	// ErrNonInteractiveLaunch is returned when a native vendor resume/fork is
	// attempted without a TTY. Codex and Claude Code refuse non-interactive
	// stdio; fail closed with a clear contract before spawning the child.
	ErrNonInteractiveLaunch = errors.New("native agent resume/fork requires an interactive terminal")
)

// LaunchPlan is a shell-free native child process description.
type LaunchPlan struct {
	Agent      string   `json:"agent"`
	SessionRef string   `json:"session_ref"`
	Operation  string   `json:"operation"`
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	Dir        string   `json:"cwd"`
}

// PlanLaunch maps an indexed session to the vendor's exact native argv.
func PlanLaunch(record Record, operation string) (LaunchPlan, error) {
	operation = strings.ToLower(strings.TrimSpace(operation))
	if operation != OperationResume && operation != OperationFork {
		return LaunchPlan{}, fmt.Errorf("unknown native session operation %q", operation)
	}
	if strings.TrimSpace(record.ID) == "" {
		return LaunchPlan{}, errors.New("native session ID is missing")
	}
	switch operation {
	case OperationResume:
		if !record.CanResume {
			return LaunchPlan{}, unsupportedNativeAction(record, operation)
		}
	case OperationFork:
		if !record.CanFork {
			return LaunchPlan{}, unsupportedNativeAction(record, operation)
		}
	}
	// Capability is checked before workspace so a read-only record always
	// reports its real compatibility contract and never reaches environment or
	// launch probing merely because its vendor omitted a workspace.
	if strings.TrimSpace(record.Workspace) == "" {
		return LaunchPlan{}, fmt.Errorf("%w: recorded session workspace is missing", ErrWorkspaceUnavailable)
	}

	plan := LaunchPlan{
		Agent:      record.Agent,
		SessionRef: record.Reference(),
		Operation:  operation,
		Dir:        record.Workspace,
	}
	switch strings.ToLower(record.Agent) {
	case AgentClaude:
		plan.Executable = "claude"
		plan.Args = []string{"--resume", record.ID}
		if operation == OperationFork {
			plan.Args = append(plan.Args, "--fork-session")
		}
	case AgentCodex:
		plan.Executable = "codex"
		plan.Args = []string{operation, record.ID}
	default:
		return LaunchPlan{}, unsupportedNativeAction(record, operation)
	}
	return plan, nil
}

func unsupportedNativeAction(record Record, operation string) error {
	reason := strings.TrimSpace(record.ReadOnlyReason)
	if reason == "" {
		reason = fmt.Sprintf("%s sessions do not support native %s", record.Agent, operation)
	}
	return fmt.Errorf("%w: %s", ErrNativeActionUnsupported, reason)
}

// LaunchRunner makes native launch execution injectable.
type LaunchRunner interface {
	Run(context.Context, LaunchPlan) error
}

// ExecLaunchRunner executes the plan with the user's attached terminal streams.
// Its zero value uses os.Stdin, os.Stdout, and os.Stderr.
type ExecLaunchRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Executable is the private absolute path verified immediately before
	// launch. It is never added to LaunchPlan or public output.
	Executable string
	// ExecutableIdentity is the private identity captured by the successful
	// compatibility probe. The production runner requires it to remain stable.
	ExecutableIdentity fileidentity.Identity
	// WorkspaceIdentity is the private directory identity captured by the
	// authorized environment report.
	WorkspaceIdentity fileidentity.Identity
	// BeforeExec is the final launch-bound safety guard. Production uses it to
	// revalidate the selected source, plan, and environment after executable
	// and directory checks and immediately before creating the native child.
	BeforeExec func(context.Context, LaunchPlan) error
}

func (runner ExecLaunchRunner) Run(ctx context.Context, plan LaunchPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateLaunchPlan(plan); err != nil {
		return err
	}
	executable := strings.TrimSpace(runner.Executable)
	if executable == "" {
		var err error
		executable, err = exec.LookPath(plan.Executable)
		if err != nil {
			return fmt.Errorf("%w: find %s executable: %v", ErrExecutableNotFound, plan.Executable, err)
		}
	} else if !filepath.IsAbs(executable) {
		return fmt.Errorf("%w: verified executable path is not absolute", ErrExecutableNotFound)
	}
	executableIdentity, err := captureExecutableAtBoundary(ctx, executable)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil || !executableIdentity.IsRegular() {
		return fmt.Errorf("%w: inspect verified executable", ErrExecutableNotFound)
	}
	if !runner.ExecutableIdentity.IsZero() &&
		!fileidentity.SameExecutable(runner.ExecutableIdentity, executableIdentity) {
		return fmt.Errorf("%w: verified executable identity changed", ErrLaunchBoundaryChanged)
	}
	workspaceIdentity, err := fileidentity.Capture(plan.Dir)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		return fmt.Errorf("%w: recorded workspace cannot be inspected", ErrWorkspaceUnavailable)
	}
	if !workspaceIdentity.IsDir() {
		return fmt.Errorf("%w: recorded workspace is not a directory", ErrWorkspaceUnavailable)
	}
	if !runner.WorkspaceIdentity.IsZero() &&
		!fileidentity.SameObject(runner.WorkspaceIdentity, workspaceIdentity) {
		return fmt.Errorf("%w: authorized workspace identity changed", ErrLaunchBoundaryChanged)
	}
	if runner.BeforeExec != nil {
		if err := runner.BeforeExec(ctx, plan); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	finalExecutableIdentity, err := captureExecutableAtBoundary(ctx, executable)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil || !fileidentity.SameExecutable(executableIdentity, finalExecutableIdentity) {
		return fmt.Errorf("%w: verified executable changed after final guard", ErrLaunchBoundaryChanged)
	}
	finalWorkspaceIdentity, err := fileidentity.Capture(plan.Dir)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil || !fileidentity.SameObject(workspaceIdentity, finalWorkspaceIdentity) ||
		!finalWorkspaceIdentity.IsDir() {
		return fmt.Errorf("%w: recorded workspace changed after final guard", ErrLaunchBoundaryChanged)
	}

	stdin := runner.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := runner.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := runner.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	// Production vendors require a real interactive terminal. Allow tests and
	// deterministic local smoke stubs to opt out with REINSTATE_ALLOW_NON_TTY_LAUNCH=1.
	if !allowNonInteractiveNativeLaunch() {
		if file, ok := stdin.(*os.File); ok && !term.IsTerminal(int(file.Fd())) {
			return fmt.Errorf("%w: re-run from a real TTY or use --dry-run for non-interactive inspection", ErrNonInteractiveLaunch)
		}
	}

	command := exec.CommandContext(ctx, executable, plan.Args...)
	command.Dir = plan.Dir
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%s native %s failed: %w", plan.Agent, plan.Operation, err)
	}
	return ctx.Err()
}

func allowNonInteractiveNativeLaunch() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("REINSTATE_ALLOW_NON_TTY_LAUNCH"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func captureExecutableAtBoundary(ctx context.Context, path string) (fileidentity.Identity, error) {
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return fileidentity.CaptureExecutable(bounded, path)
}

// RunLaunch validates a structured plan through the selected runner and waits
// for the native child to finish.
func RunLaunch(ctx context.Context, plan LaunchPlan, runner LaunchRunner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateLaunchPlan(plan); err != nil {
		return err
	}
	if runner == nil {
		runner = ExecLaunchRunner{}
	}
	if err := runner.Run(ctx, plan); err != nil {
		return err
	}
	return ctx.Err()
}

func validateLaunchPlan(plan LaunchPlan) error {
	if plan.Executable == "" {
		return fmt.Errorf("%w: launch executable is missing", ErrExecutableNotFound)
	}
	if len(plan.Args) == 0 {
		return errors.New("launch argv is missing")
	}
	if plan.Dir == "" {
		return fmt.Errorf("%w: launch working directory is missing", ErrWorkspaceUnavailable)
	}
	if plan.Operation != OperationResume && plan.Operation != OperationFork && plan.Operation != OperationHandoff {
		return fmt.Errorf("unknown launch operation %q", plan.Operation)
	}
	return nil
}
