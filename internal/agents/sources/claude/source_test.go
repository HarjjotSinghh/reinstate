package claude

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestScanMatchesSessionindexAndSkipsSubagents(t *testing.T) {
	t.Parallel()
	root := abs(t, testdata(t, "sessionindex", "claude", "macos"))
	want, err := sessionindex.NewClaudeSource(root).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := mustSource(t, root).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog source drifted from sessionindex\n got %#v\nwant %#v", got, want)
	}

	subagents := abs(t, testdata(t, "handoff", "claude", "subagents"))
	result, err := mustSource(t, subagents).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range result.Records {
		if filepath.Base(filepath.Dir(record.SourcePath)) == "subagents" {
			t.Fatalf("indexed subagent %q", record.SourcePath)
		}
	}
	if len(result.Records) == 0 {
		t.Fatal("expected parent session, got none")
	}
}

func mustSource(t *testing.T, root string) sessionindex.Source {
	t.Helper()
	src, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func testdata(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	elems := append([]string{filepath.Dir(file), "..", "..", "..", "..", "testdata"}, parts...)
	return filepath.Join(elems...)
}

func abs(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}
