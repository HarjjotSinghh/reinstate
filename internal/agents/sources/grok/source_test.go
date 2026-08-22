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

// TestScanMarksResumeCapabilityFromSessionIDShape pins the one thing that can
// still make a Grok session read-only at T3.
//
// `grok --resume [<SESSION_ID_OR_TITLE>]` resolves any value that is not
// UUID-shaped as a session *title*, and titles are neither unique nor stable.
// A session whose recorded id is not a UUID therefore stays read-only rather
// than being resumed by name.
func TestScanMarksResumeCapabilityFromSessionIDShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		root           []string
		wantResumable  bool
		wantReasonPart string
	}{
		{
			name:          "uuid_session_is_resumable",
			root:          []string{"sessionindex", "grok", "macos"},
			wantResumable: true,
		},
		{
			name:           "non_uuid_session_stays_read_only",
			root:           []string{"handoff", "grok", "basic"},
			wantResumable:  false,
			wantReasonPart: "not a UUID",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			src, err := New(agents.Env{FixtureRoot: abs(t, testdata(t, test.root...))})
			if err != nil {
				t.Fatal(err)
			}
			result, err := src.Scan(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Records) == 0 {
				t.Fatal("fixture produced no records")
			}
			for _, record := range result.Records {
				if record.CanResume != test.wantResumable || record.CanFork != test.wantResumable {
					t.Fatalf("%s: resume=%t fork=%t, want %t",
						record.ID, record.CanResume, record.CanFork, test.wantResumable)
				}
				if test.wantResumable {
					if record.ReadOnlyReason != "" {
						t.Fatalf("%s: read_only_reason = %q, want empty", record.ID, record.ReadOnlyReason)
					}
					continue
				}
				if !strings.Contains(record.ReadOnlyReason, test.wantReasonPart) {
					t.Fatalf("%s: read_only_reason = %q, want it to mention %q",
						record.ID, record.ReadOnlyReason, test.wantReasonPart)
				}
			}
		})
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
