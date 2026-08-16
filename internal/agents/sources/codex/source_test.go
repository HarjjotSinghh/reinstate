package codex

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestScanPreservesFilenameWinsIdentity(t *testing.T) {
	t.Parallel()
	root := abs(t, testdata(t, "sessionindex", "codex", "forks"))
	want, err := sessionindex.NewCodexSource(root).Scan(context.Background())
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
	if len(got.Records) < 2 {
		t.Fatalf("records = %d, want forks", len(got.Records))
	}
	seen := map[string]struct{}{}
	for _, record := range got.Records {
		if _, exists := seen[record.ID]; exists {
			t.Fatalf("fork coalesced into %q", record.ID)
		}
		seen[record.ID] = struct{}{}
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
