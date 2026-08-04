package sessionindex

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
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

func TestExecLaunchRunnerClassifiesPreflightFailures(t *testing.T) {
	t.Parallel()
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

type launchRunnerFunc func(context.Context, LaunchPlan) error

func (runner launchRunnerFunc) Run(ctx context.Context, plan LaunchPlan) error {
	return runner(ctx, plan)
}
