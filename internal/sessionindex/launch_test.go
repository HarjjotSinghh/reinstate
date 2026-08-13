package sessionindex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

func TestPlanLaunchExactNativeArgv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		record    Record
		operation string
		want      LaunchPlan
	}{
		{
			name: "Claude resume",
			record: Record{
				Key: "claude:id-one", ID: "id-one", Agent: AgentClaude,
				Workspace: "/work/demo", CanResume: true, CanFork: true,
			},
			operation: OperationResume,
			want: LaunchPlan{
				Agent: AgentClaude, SessionRef: "claude:id-one",
				Operation: OperationResume, Executable: "claude",
				Args: []string{"--resume", "id-one"}, Dir: "/work/demo",
			},
		},
		{
			name: "Claude fork",
			record: Record{
				Key: "claude:id-one", ID: "id-one", Agent: AgentClaude,
				Workspace: "/work/demo", CanResume: true, CanFork: true,
			},
			operation: OperationFork,
			want: LaunchPlan{
				Agent: AgentClaude, SessionRef: "claude:id-one",
				Operation: OperationFork, Executable: "claude",
				Args: []string{"--resume", "id-one", "--fork-session"}, Dir: "/work/demo",
			},
		},
		{
			name: "Codex resume",
			record: Record{
				Key: "codex:id-two", ID: "id-two", Agent: AgentCodex,
				Workspace: `C:\work\demo`, CanResume: true, CanFork: true,
			},
			operation: OperationResume,
			want: LaunchPlan{
				Agent: AgentCodex, SessionRef: "codex:id-two",
				Operation: OperationResume, Executable: "codex",
				Args: []string{"resume", "id-two"}, Dir: `C:\work\demo`,
			},
		},
		{
			name: "Codex fork",
			record: Record{
				Key: "codex:id-two", ID: "id-two", Agent: AgentCodex,
				Workspace: `C:\work\demo`, CanResume: true, CanFork: true,
			},
			operation: OperationFork,
			want: LaunchPlan{
				Agent: AgentCodex, SessionRef: "codex:id-two",
				Operation: OperationFork, Executable: "codex",
				Args: []string{"fork", "id-two"}, Dir: `C:\work\demo`,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := PlanLaunch(test.record, test.operation)
			if err != nil {
				t.Fatalf("PlanLaunch() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("PlanLaunch() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPlanLaunchRejectsReadOnlyAndMissingWorkspace(t *testing.T) {
	t.Parallel()
	_, err := PlanLaunch(Record{
		ID: "gemini-id", Agent: AgentGemini, Workspace: "/work",
		ReadOnlyReason: geminiReadOnlyReason,
	}, OperationResume)
	if !errors.Is(err, ErrNativeActionUnsupported) {
		t.Fatalf("read-only error = %v", err)
	}

	_, err = PlanLaunch(Record{
		ID: "codex-id", Agent: AgentCodex, CanResume: true,
	}, OperationResume)
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("workspace error = %v", err)
	}
}

func TestPlanLaunchReportsReadOnlyBeforeMissingWorkspace(t *testing.T) {
	record := Record{
		ID: "gemini-id", Agent: AgentGemini,
		ReadOnlyReason: "Gemini CLI sessions are read-only in Phase 3",
	}
	_, err := PlanLaunch(record, OperationResume)
	if !errors.Is(err, ErrNativeActionUnsupported) {
		t.Fatalf("error = %v, want native-action unsupported", err)
	}
	if errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("read-only record reported workspace error: %v", err)
	}
}

func TestRunLaunchUsesInjectedStructuredPlanAndPropagatesFailure(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("child exit")
	var captured LaunchPlan
	runner := launchRunnerFunc(func(_ context.Context, plan LaunchPlan) error {
		captured = plan
		return sentinel
	})
	plan := LaunchPlan{
		Agent: AgentCodex, SessionRef: "codex:id", Operation: OperationFork,
		Executable: "codex", Args: []string{"fork", "id"}, Dir: "/work",
	}
	err := RunLaunch(context.Background(), plan, runner)
	if !errors.Is(err, sentinel) {
		t.Fatalf("RunLaunch() error = %v", err)
	}
	if !reflect.DeepEqual(captured, plan) {
		t.Fatalf("captured plan = %#v", captured)
	}
}

func TestRunLaunchDoesNotInvokeRunnerAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	plan := LaunchPlan{
		Agent: AgentCodex, SessionRef: "codex:id", Operation: OperationResume,
		Executable: "codex", Args: []string{"resume", "id"}, Dir: "/work",
	}
	err := RunLaunch(ctx, plan, launchRunnerFunc(func(context.Context, LaunchPlan) error {
		called = true
		return nil
	}))
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("RunLaunch() error/called = %v / %t", err, called)
	}
}

func TestRunLaunchPropagatesCancellationRaisedByRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	plan := LaunchPlan{
		Agent: AgentCodex, SessionRef: "codex:id", Operation: OperationResume,
		Executable: "codex", Args: []string{"resume", "id"}, Dir: "/work",
	}
	err := RunLaunch(ctx, plan, launchRunnerFunc(func(context.Context, LaunchPlan) error {
		cancel()
		return nil
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunLaunch() error = %v, want runner cancellation", err)
	}
}

func TestExecLaunchRunnerClassifiesPreflightFailures(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	err := (ExecLaunchRunner{}).Run(context.Background(), LaunchPlan{
		Agent: AgentCodex, Operation: OperationResume,
		Executable: "reinstate-definitely-not-an-executable", Args: []string{"resume", "id"}, Dir: t.TempDir(),
	})
	if !errors.Is(err, ErrExecutableNotFound) {
		t.Fatalf("executable error = %v", err)
	}

	err = (ExecLaunchRunner{}).Run(context.Background(), LaunchPlan{
		Agent: AgentCodex, Operation: OperationResume,
		Executable: os.Args[0], Args: []string{"-test.run=^$"}, Dir: "/reinstate-definitely-missing-workspace",
	})
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("workspace error = %v", err)
	}
}

func TestExecLaunchRunnerDistinguishesPostSpawnExitFromStartFailure(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	t.Setenv("REINSTATE_LAUNCH_EXIT_HELPER", "1")
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	plan := LaunchPlan{
		Agent: AgentClaude, SessionRef: "claude:controlled", Operation: OperationHandoff,
		Executable: "claude", Args: []string{"-test.run=^TestExecLaunchRunnerExitHelper$"}, Dir: t.TempDir(),
	}
	err = (ExecLaunchRunner{Executable: executable}).Run(context.Background(), plan)
	if !errors.Is(err, ErrChildStarted) {
		t.Fatalf("post-spawn error = %v, want ErrChildStarted", err)
	}

	invalid := filepath.Join(t.TempDir(), "invalid-executable")
	if err := os.WriteFile(invalid, []byte("not an executable format"), 0o700); err != nil {
		t.Fatal(err)
	}
	err = (ExecLaunchRunner{Executable: invalid}).Run(context.Background(), plan)
	if err == nil || errors.Is(err, ErrChildStarted) {
		t.Fatalf("pre-spawn error = %v, must not claim child started", err)
	}
}

func TestExecLaunchRunnerExitHelper(t *testing.T) {
	if os.Getenv("REINSTATE_LAUNCH_EXIT_HELPER") == "" {
		return
	}
	os.Exit(17)
}

func TestExecLaunchRunnerGuardRejectionPreventsChildCreation(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	marker := filepath.Join(t.TempDir(), "child-created")
	t.Setenv("REINSTATE_LAUNCH_GUARD_HELPER_MARKER", marker)
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	guardErr := errors.New("controlled final guard rejection")
	plan := LaunchPlan{
		Agent: AgentClaude, SessionRef: "claude:controlled", Operation: OperationResume,
		Executable: "claude", Args: []string{"-test.run=^TestExecLaunchRunnerGuardHelper$"}, Dir: t.TempDir(),
	}
	err = RunLaunch(context.Background(), plan, ExecLaunchRunner{
		Executable: executable,
		BeforeExec: func(context.Context, LaunchPlan) error { return guardErr },
	})
	if !errors.Is(err, guardErr) {
		t.Fatalf("guard rejection error = %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("child marker exists after guard rejection: %v", err)
	}
}

func TestExecLaunchRunnerPropagatesCancellationAtFinalGuard(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	marker := filepath.Join(t.TempDir(), "child-created")
	t.Setenv("REINSTATE_LAUNCH_GUARD_HELPER_MARKER", marker)
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	plan := LaunchPlan{
		Agent: AgentClaude, SessionRef: "claude:controlled", Operation: OperationResume,
		Executable: "claude", Args: []string{"-test.run=^TestExecLaunchRunnerGuardHelper$"}, Dir: t.TempDir(),
	}
	err = RunLaunch(ctx, plan, ExecLaunchRunner{
		Executable: executable,
		BeforeExec: func(context.Context, LaunchPlan) error {
			cancel()
			return nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("guard cancellation error = %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("child marker exists after guard cancellation: %v", err)
	}
}

func TestExecLaunchRunnerRejectsLaunchTargetReplacementAfterGuard(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	for _, target := range []string{"executable", "workspace"} {
		target := target
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			executable := filepath.Join(root, "agent")
			if err := os.WriteFile(executable, []byte("original executable"), 0o700); err != nil {
				t.Fatal(err)
			}
			executableIdentity, err := fileidentity.CaptureExecutable(context.Background(), executable)
			if err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(root, "workspace")
			if err := os.Mkdir(workspace, 0o700); err != nil {
				t.Fatal(err)
			}
			plan := LaunchPlan{
				Agent: AgentClaude, SessionRef: "claude:controlled", Operation: OperationResume,
				Executable: "claude", Args: []string{"--resume", "controlled"}, Dir: workspace,
			}
			runner := ExecLaunchRunner{
				Executable: executable, ExecutableIdentity: executableIdentity,
				BeforeExec: func(context.Context, LaunchPlan) error {
					switch target {
					case "executable":
						replacement := filepath.Join(root, "replacement-agent")
						if err := os.WriteFile(replacement, []byte("replacement executable contents"), 0o700); err != nil {
							return err
						}
						return os.Rename(replacement, executable)
					case "workspace":
						old := filepath.Join(root, "old-workspace")
						if err := os.Rename(workspace, old); err != nil {
							return err
						}
						return os.Mkdir(workspace, 0o700)
					default:
						return errors.New("unexpected target")
					}
				},
			}
			err = runner.Run(context.Background(), plan)
			if !errors.Is(err, ErrLaunchBoundaryChanged) {
				t.Fatalf("Run() error = %v, want launch-boundary change", err)
			}
		})
	}
}

func TestExecLaunchRunnerRejectsWorkspaceReplacedAfterAuthorization(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	authorizedIdentity, err := fileidentity.Capture(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(workspace, filepath.Join(root, "old-workspace")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "agent")
	if err := os.WriteFile(executable, []byte("controlled executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	executableIdentity, err := fileidentity.CaptureExecutable(context.Background(), executable)
	if err != nil {
		t.Fatal(err)
	}
	err = (ExecLaunchRunner{
		Executable: executable, ExecutableIdentity: executableIdentity,
		WorkspaceIdentity: authorizedIdentity,
	}).Run(context.Background(), LaunchPlan{
		Agent: AgentClaude, SessionRef: "claude:controlled", Operation: OperationResume,
		Executable: "claude", Args: []string{"--resume", "controlled"}, Dir: workspace,
	})
	if !errors.Is(err, ErrLaunchBoundaryChanged) {
		t.Fatalf("Run() error = %v, want launch-boundary change", err)
	}
}

func TestExecLaunchRunnerGuardHelper(t *testing.T) {
	marker := os.Getenv("REINSTATE_LAUNCH_GUARD_HELPER_MARKER")
	if marker == "" {
		return
	}
	if err := os.WriteFile(marker, []byte("child ran"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExecLaunchRunnerNonTTYRefusesBeforeLookPath(t *testing.T) {
	t.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "")
	t.Setenv("PATH", t.TempDir())
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	defer func() { _ = writer.Close() }()

	err = (ExecLaunchRunner{Stdin: stdin}).Run(context.Background(), LaunchPlan{
		Agent: AgentClaude, Operation: OperationHandoff,
		Executable: "claude", Args: []string{"--session-id", "controlled"}, Dir: t.TempDir(),
	})
	if !errors.Is(err, ErrNonInteractiveLaunch) {
		t.Fatalf("non-TTY error = %v, want %v before LookPath", err, ErrNonInteractiveLaunch)
	}
}

type launchRunnerFunc func(context.Context, LaunchPlan) error

func (runner launchRunnerFunc) Run(ctx context.Context, plan LaunchPlan) error {
	return runner(ctx, plan)
}
