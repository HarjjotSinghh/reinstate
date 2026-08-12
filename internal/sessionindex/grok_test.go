package sessionindex

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGrokSourceIndexesMacOSAndWindowsFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		root          string
		wantID        string
		wantWorkspace string
		wantTitle     string
		wantPrompt    string
		wantCompacted bool
	}{
		{
			name:          "macOS compacted",
			root:          filepath.Join("..", "..", "testdata", "sessionindex", "grok", "macos"),
			wantID:        "01987654-3210-7890-abcd-ef0123456789",
			wantWorkspace: "/Users/fixture-user/code/demo",
			wantTitle:     "Synthetic Grok macOS fixture session",
			wantPrompt:    "Post-compaction continuation prompt",
			wantCompacted: true,
		},
		{
			name:          "Windows basic",
			root:          filepath.Join("..", "..", "testdata", "sessionindex", "grok", "windows"),
			wantID:        "01987654-3210-7890-abcd-ef0123456790",
			wantWorkspace: `C:\Users\fixture-user\code\demo`,
			wantTitle:     "Synthetic Grok Windows fixture session",
			wantPrompt:    "Synthetic Grok Windows user prompt",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewGrokSource(test.root).Scan(context.Background())
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.Agent != AgentGrok || record.ID != test.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Key != CompositeReference(AgentGrok, test.wantID) {
				t.Fatalf("key = %q", record.Key)
			}
			if record.Workspace != test.wantWorkspace {
				t.Fatalf("workspace = %q, want %q", record.Workspace, test.wantWorkspace)
			}
			if record.Title != test.wantTitle {
				t.Fatalf("title = %q, want %q", record.Title, test.wantTitle)
			}
			if record.PromptPreview != test.wantPrompt {
				t.Fatalf("prompt_preview = %q, want %q", record.PromptPreview, test.wantPrompt)
			}
			if record.CanResume || record.CanFork {
				t.Fatalf("capabilities resume=%t fork=%t, want false", record.CanResume, record.CanFork)
			}
			if record.ReadOnlyReason != GrokReadOnlyReason {
				t.Fatalf("read_only_reason = %q", record.ReadOnlyReason)
			}
			if record.Project != "demo" {
				t.Fatalf("project = %q", record.Project)
			}
			if !record.UpdatedAt.Equal(time.Date(2026, 8, 12, 2, 10, 0, 0, time.UTC)) {
				t.Fatalf("updated_at = %s", record.UpdatedAt)
			}
			if got := AgentRoot(record); got != "" {
				wantRoot, absErr := filepath.Abs(test.root)
				if absErr != nil {
					t.Fatalf("Abs(%q): %v", test.root, absErr)
				}
				if got != wantRoot {
					t.Fatalf("AgentRoot() = %q, want %q", got, wantRoot)
				}
			} else {
				t.Fatal("AgentRoot() empty")
			}
			if test.wantCompacted && !strings.Contains(record.SearchText, "PRE_COMPACT_USER_TURN") {
				t.Fatalf("compacted search text missing pre-compact prompt: %q", record.SearchText)
			}
			for _, forbidden := range []string{
				"Synthetic system preamble",
				"Post-compaction assistant reply",
				"PRE_COMPACT_ASSISTANT_TURN",
			} {
				if strings.Contains(record.SearchText, forbidden) {
					t.Fatalf("search text contains assistant/system text %q", forbidden)
				}
			}
		})
	}
}

func TestGrokPlanLaunchRefusesResumeAndFork(t *testing.T) {
	t.Parallel()
	record := Record{
		ID:             "grok-session",
		Agent:          AgentGrok,
		Workspace:      "/Users/fixture-user/code/demo",
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: GrokReadOnlyReason,
	}
	for _, operation := range []string{OperationResume, OperationFork} {
		_, err := PlanLaunch(record, operation)
		if !errors.Is(err, ErrNativeActionUnsupported) {
			t.Fatalf("PlanLaunch(%s) error = %v, want ErrNativeActionUnsupported", operation, err)
		}
		if !strings.Contains(err.Error(), GrokReadOnlyReason) {
			t.Fatalf("PlanLaunch(%s) error = %v, want read-only reason", operation, err)
		}
	}
}

func TestResolveGrokRootUsesGROKHOME(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GROK_HOME", root)
	got, err := resolveGrokRoot("")
	if err != nil {
		t.Fatalf("resolveGrokRoot() error = %v", err)
	}
	if got != filepath.Clean(root) {
		t.Fatalf("resolveGrokRoot() = %q, want %q", got, root)
	}
}

func TestDecodeGrokWorkspaceURLEncoded(t *testing.T) {
	t.Parallel()
	got := decodeGrokWorkspace("%2FUsers%2Ffixture-user%2Fcode%2Fdemo", t.TempDir())
	if got != "/Users/fixture-user/code/demo" {
		t.Fatalf("decode = %q", got)
	}
	got = decodeGrokWorkspace(`C%3A%5CUsers%5Cfixture-user%5Ccode%5Cdemo`, t.TempDir())
	if got != `C:\Users\fixture-user\code\demo` {
		t.Fatalf("decode windows = %q", got)
	}
}
