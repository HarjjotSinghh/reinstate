package sessionindex

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func TestOpenCodeSourceUsesDocumentedJSONCommand(t *testing.T) {
	t.Parallel()
	var (
		executable string
		args       []string
		hasTimeout bool
	)
	runner := CommandRunnerFunc(func(ctx context.Context, name string, commandArgs ...string) ([]byte, error) {
		executable = name
		args = append([]string(nil), commandArgs...)
		deadline, ok := ctx.Deadline()
		hasTimeout = ok && time.Until(deadline) <= openCodeListTimeout
		return []byte(`[
			{
				"id":"oc-1",
				"title":"Local session list",
				"projectID":"reinstate",
				"directory":"C:\\work\\reinstate",
				"branch":"phase-two",
				"time":{"updated":1785402000000},
				"messageCount":7
			},
			{"title":"missing identity"}
		]`), nil
	})

	result, err := NewOpenCodeSource(runner).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if executable != "opencode" ||
		!reflect.DeepEqual(args, []string{"session", "list", "--format", "json"}) {
		t.Fatalf("command = %q %#v", executable, args)
	}
	if !hasTimeout {
		t.Fatal("runner context does not have a bounded deadline")
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.Key != "opencode:oc-1" || record.Workspace != `C:\work\reinstate` {
		t.Fatalf("identity/workspace = %q / %q", record.Key, record.Workspace)
	}
	if record.CanResume || record.CanFork || record.ReadOnlyReason == "" {
		t.Fatalf("capabilities = resume:%t fork:%t reason:%q", record.CanResume, record.CanFork, record.ReadOnlyReason)
	}
	if record.MessageCount != 7 {
		t.Fatalf("message_count = %d", record.MessageCount)
	}
	if !hasWarningCode(result.Warnings, "missing_session_id") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestOpenCodeSourceCommandFailuresAreNonDestructive(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("command failed")
	source := NewOpenCodeSource(CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return nil, sentinel
	}))
	_, err := source.Scan(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("Scan() error = %v, want sentinel", err)
	}
}

func TestOpenCodeSourceMissingExecutableIsOptional(t *testing.T) {
	t.Parallel()
	for _, commandErr := range []error{
		&exec.Error{Name: "opencode", Err: exec.ErrNotFound},
		fakeExitCodeError(127),
	} {
		commandErr := commandErr
		source := NewOpenCodeSource(CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
			return nil, commandErr
		}))
		result, err := source.Scan(context.Background())
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		if !hasWarningCode(result.Warnings, "agent_not_installed") {
			t.Fatalf("warnings = %#v", result.Warnings)
		}
	}
}

type fakeExitCodeError int

func (err fakeExitCodeError) Error() string { return "controlled command exit" }
func (err fakeExitCodeError) ExitCode() int { return int(err) }

func TestOpenCodeSourceRejectsOversizedCommandOutput(t *testing.T) {
	t.Parallel()
	source := NewOpenCodeSource(CommandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return make([]byte, maxOpenCodeListOutput+1), nil
	}))
	_, err := source.Scan(context.Background())
	if !errors.Is(err, ErrCommandOutputTooLarge) {
		t.Fatalf("Scan() error = %v", err)
	}
}

func TestBoundedCommandOutputDrainsWithoutRetainingOverflow(t *testing.T) {
	t.Parallel()
	output := boundedCommandOutput{remaining: 4}
	written, err := output.Write([]byte("123456"))
	if err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if !output.exceeded || output.String() != "1234" {
		t.Fatalf("bounded output = %q exceeded=%t", output.String(), output.exceeded)
	}
}
