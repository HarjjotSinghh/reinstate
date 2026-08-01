package sessionindex

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
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
	wantUpdatedAt := time.Unix(1785402000, 0).UTC()
	if !record.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("updated_at = %s, want %s", record.UpdatedAt, wantUpdatedAt)
	}
	if !hasWarningCode(result.Warnings, "missing_session_id") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestOpenCodeRecordSupportsTopLevelTimestamps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload string
		want    time.Time
	}{
		{
			name:    "updated preferred over created",
			payload: `[{"id":"oc-updated","created":1785402000000,"updated":1785402060000}]`,
			want:    time.Unix(1785402060, 0).UTC(),
		},
		{
			name:    "created fallback",
			payload: `[{"id":"oc-created","created":1785402120000}]`,
			want:    time.Unix(1785402120, 0).UTC(),
		},
		{
			name:    "nested created compatibility",
			payload: `[{"id":"oc-nested-created","time":{"created":1785402180000}}]`,
			want:    time.Unix(1785402180, 0).UTC(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sessions, err := decodeOpenCodeSessions([]byte(test.payload))
			if err != nil {
				t.Fatalf("decodeOpenCodeSessions() error = %v", err)
			}
			if len(sessions) != 1 {
				t.Fatalf("sessions = %d, want 1", len(sessions))
			}
			record, ok := openCodeRecord(sessions[0])
			if !ok {
				t.Fatal("openCodeRecord() rejected session with an ID")
			}
			if !record.UpdatedAt.Equal(test.want) {
				t.Fatalf("updated_at = %s, want %s", record.UpdatedAt, test.want)
			}
			if record.SourceModTime != test.want.UnixNano() {
				t.Fatalf("source_mod_time = %d, want %d", record.SourceModTime, test.want.UnixNano())
			}
		})
	}
}

func TestOpenCodeTopLevelTimestampRemainsVisibleUnderDefaultLimit(t *testing.T) {
	t.Parallel()
	const payload = `[{"id":"oc-current","created":1785402000000,"updated":1785405600000}]`
	sessions, err := decodeOpenCodeSessions([]byte(payload))
	if err != nil {
		t.Fatalf("decodeOpenCodeSessions() error = %v", err)
	}
	openCode, ok := openCodeRecord(sessions[0])
	if !ok {
		t.Fatal("openCodeRecord() rejected session with an ID")
	}

	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.ReplaceSource(ctx, AgentOpenCode, []Record{openCode}); err != nil {
		t.Fatal(err)
	}

	older := make([]Record, DefaultLimit)
	for index := range older {
		older[index] = testRecord(
			AgentClaude,
			fmt.Sprintf("older-%03d", index),
			openCode.UpdatedAt.Add(-time.Duration(index+1)*time.Minute),
			fmt.Sprintf("/sessions/older-%03d.jsonl", index),
			int64(index+1),
		)
	}
	if _, err := store.ReplaceSource(ctx, AgentClaude, older); err != nil {
		t.Fatal(err)
	}

	records, err := store.Search(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != DefaultLimit {
		t.Fatalf("default results = %d, want %d", len(records), DefaultLimit)
	}
	if records[0].Reference() != "opencode:oc-current" {
		t.Fatalf("first default result = %q, want OpenCode session", records[0].Reference())
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
