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
)

const (
	OperationResume = "resume"
	OperationFork   = "fork"
)

var (
	ErrNativeActionUnsupported = errors.New("native session action is unsupported")
	ErrExecutableNotFound      = errors.New("native agent executable is unavailable")
	ErrWorkspaceUnavailable    = errors.New("recorded session workspace is unavailable")
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
	// BeforeExec is the final launch-bound safety guard. Production uses it to
	// revalidate the selected source, plan, and environment after executable
	// and directory checks and immediately before creating the native child.
	BeforeExec func(context.Context, LaunchPlan) error
}

func (runner ExecLaunchRunner) Run(ctx context.Context, plan LaunchPlan) error {
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
	info, err := os.Stat(plan.Dir)
	if err != nil {
		return fmt.Errorf("%w: inspect recorded workspace %q: %v", ErrWorkspaceUnavailable, plan.Dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: recorded workspace %q is not a directory", ErrWorkspaceUnavailable, plan.Dir)
	}
	if runner.BeforeExec != nil {
		if err := runner.BeforeExec(ctx, plan); err != nil {
			return err
		}
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

	command := exec.CommandContext(ctx, executable, plan.Args...)
	command.Dir = plan.Dir
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s native %s failed: %w", plan.Agent, plan.Operation, err)
	}
	return nil
}

// RunLaunch validates a structured plan through the selected runner and waits
// for the native child to finish.
func RunLaunch(ctx context.Context, plan LaunchPlan, runner LaunchRunner) error {
	if err := validateLaunchPlan(plan); err != nil {
		return err
	}
	if runner == nil {
		runner = ExecLaunchRunner{}
	}
	return runner.Run(ctx, plan)
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
	if plan.Operation != OperationResume && plan.Operation != OperationFork {
		return fmt.Errorf("unknown launch operation %q", plan.Operation)
	}
	return nil
}
