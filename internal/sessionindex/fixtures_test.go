package sessionindex

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase2MacOSWindowsAndWSLFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		source          Source
		wantWorkspace   string
		wantPromptToken string
	}{
		{
			name: "Claude macOS",
			source: NewClaudeSource(filepath.Join(
				"..", "..", "testdata", "sessionindex", "claude", "macos",
			)),
			wantWorkspace: "/Users/fixture-user/code/demo", wantPromptToken: "macOS",
		},
		{
			name: "Claude Windows",
			source: NewClaudeSource(filepath.Join(
				"..", "..", "testdata", "sessionindex", "claude", "windows",
			)),
			wantWorkspace: `C:\Users\fixture-user\code\demo`, wantPromptToken: "Windows",
		},
		{
			name: "Claude WSL",
			source: NewClaudeSource(filepath.Join(
				"..", "..", "testdata", "sessionindex", "claude", "wsl",
			)),
			wantWorkspace: "/home/fixture-user/code/demo", wantPromptToken: "WSL2",
		},
		{
			name: "Codex macOS",
			source: NewCodexSource(filepath.Join(
				"..", "..", "testdata", "adapters", "codex", "macos",
			)),
			wantWorkspace: "/Users/fixture-user/code/demo", wantPromptToken: "fixture request",
		},
		{
			name: "Codex Windows",
			source: NewCodexSource(filepath.Join(
				"..", "..", "testdata", "adapters", "codex", "windows",
			)),
			wantWorkspace: `C:\Users\fixture-user\code\demo`, wantPromptToken: "Windows",
		},
		{
			name: "Codex WSL",
			source: NewCodexSource(filepath.Join(
				"..", "..", "testdata", "adapters", "codex", "wsl",
			)),
			wantWorkspace: "/home/fixture-user/code/demo", wantPromptToken: "WSL2",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := test.source.Scan(context.Background())
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.Workspace != test.wantWorkspace || record.Project != "demo" {
				t.Fatalf(
					"workspace/project = %q / %q, want %q / demo",
					record.Workspace,
					record.Project,
					test.wantWorkspace,
				)
			}
			if !strings.Contains(record.PromptPreview, test.wantPromptToken) {
				t.Fatalf("preview %q does not contain %q", record.PromptPreview, test.wantPromptToken)
			}
		})
	}
}
