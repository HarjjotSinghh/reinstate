package opencode

import (
	"context"
	"reflect"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/cliquery"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestScanMatchesSessionindexJSONList(t *testing.T) {
	t.Parallel()
	payload := []byte(`[
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
	]`)
	legacy := sessionindex.CommandRunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return payload, nil
	})
	want, err := sessionindex.NewOpenCodeSource(legacy).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewWithRunner(agents.Env{}, cliquery.RunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return payload, nil
	})).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog source drifted from sessionindex\n got %#v\nwant %#v", got, want)
	}
}

// TestProjectPrefersDirectoryOverOpaqueID covers OpenCode builds that report
// projectId as a 40-hex digest. Naming a session after the digest hides which
// project it belongs to and breaks Matrix C2, which compares the project
// against what the agent shows.
func TestProjectPrefersDirectoryOverOpaqueID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		entry       string
		wantProject string
	}{
		{
			name:        "digest id yields to directory",
			entry:       `{"id":"oc-1","projectId":"1b7374f1679155256b7b93dc043bf4a6ec4bb892","directory":"/Users/fixture-user/code/demo"}`,
			wantProject: "demo",
		},
		{
			name:        "named project object still wins",
			entry:       `{"id":"oc-1","project":{"name":"named"},"directory":"/Users/fixture-user/code/demo"}`,
			wantProject: "named",
		},
		{
			name:        "digest survives when no directory is reported",
			entry:       `{"id":"oc-1","projectId":"1b7374f1679155256b7b93dc043bf4a6ec4bb892"}`,
			wantProject: "1b7374f1679155256b7b93dc043bf4a6ec4bb892",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload := []byte("[" + tt.entry + "]")
			runner := cliquery.RunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return payload, nil
			})
			result, err := NewWithRunner(agents.Env{}, runner).Scan(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			if got := result.Records[0].Project; got != tt.wantProject {
				t.Fatalf("Project = %q, want %q", got, tt.wantProject)
			}
		})
	}
}
