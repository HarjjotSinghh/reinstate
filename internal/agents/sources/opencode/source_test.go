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
