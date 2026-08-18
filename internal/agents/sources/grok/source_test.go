package grok

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestScanPreservesWorkspaceKeyEncoding(t *testing.T) {
	t.Parallel()
	root := abs(t, testdata(t, "sessionindex", "grok", "macos"))
	want, err := sessionindex.NewGrokSource(root).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	src, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	got, err := src.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog source drifted from sessionindex\n got %#v\nwant %#v", got, want)
	}
	if len(got.Records) == 0 || !strings.Contains(got.Records[0].Workspace, "/Users/fixture-user/code/demo") {
		t.Fatalf("workspace key was not decoded: %#v", got.Records)
	}
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
