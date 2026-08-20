package copilot

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func scan(t *testing.T, root string) sessionindex.ScanResult {
	t.Helper()
	source, err := New(agents.Env{FixtureRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	result, err := source.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func fixture(t *testing.T, osName string) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "testdata", "sessionindex", "copilot", osName)
}

func TestScanFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		osName, wantID, wantWorkspace string
	}{
		{"macos", "01987654-3210-7890-abcd-ef0123456789", "/Users/fixture-user/code/demo"},
		{"windows", "01912345-6789-7abc-def0-123456789abc", `C:\Users\fixture-user\code\demo`},
	}
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			result := scan(t, fixture(t, tt.osName))
			if len(result.Records) != 1 {
				t.Fatalf("records = %d warnings=%v", len(result.Records), result.Warnings)
			}
			record := result.Records[0]
			if record.Agent != sessionindex.AgentCopilot || record.ID != tt.wantID {
				t.Fatalf("identity = %q / %q", record.Agent, record.ID)
			}
			if record.Workspace != tt.wantWorkspace {
				t.Fatalf("workspace = %q", record.Workspace)
			}
			if record.CanResume || record.ReadOnlyReason != sessionindex.CopilotReadOnlyReason {
				t.Fatalf("resume=%t reason=%q", record.CanResume, record.ReadOnlyReason)
			}
			if record.PromptPreview == "" || record.MessageCount < 1 {
				t.Fatalf("preview=%q messages=%d", record.PromptPreview, record.MessageCount)
			}
		})
	}
}

func TestScanIsDeterministic(t *testing.T) {
	t.Parallel()
	root := fixture(t, "macos")
	if !reflect.DeepEqual(scan(t, root).Records, scan(t, root).Records) {
		t.Fatal("two scans of one fixture disagreed")
	}
}

func TestUnknownLayoutIsSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "session-state", "bad-session")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{\"nope\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(result.Records))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("want a session_read_failed warning")
	}
}

func TestSQLiteSidecarIsNotIndexed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "session-state", "only-db")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.db"), []byte("not-a-session"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "session-store.db"), []byte("not-a-session"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := scan(t, root)
	if len(result.Records) != 0 {
		t.Fatalf("records = %+v", result.Records)
	}
}

// writeCopilotSession lays out one session-state directory. events carries the
// raw JSONL lines so a test can withhold the cwd that older builds emitted.
func writeCopilotSession(t *testing.T, root, id, events, manifest string) {
	t.Helper()
	dir := filepath.Join(root, "session-state", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestWorkspaceManifestFallback covers Copilot CLI 1.0.80, which stopped
// emitting cwd inside events.jsonl. Without the sibling workspace.yaml the
// session indexes with project "unknown" and no workspace, so Matrix C1 can
// never see two distinct projects.
func TestWorkspaceManifestFallback(t *testing.T) {
	t.Parallel()
	const prompt = `{"data":{"content":"hello"},"id":"evt-1","parentId":null,` +
		`"timestamp":"2026-08-17T10:00:00.000Z","type":"user"}` + "\n"
	tests := []struct {
		name          string
		events        string
		manifest      string
		wantProject   string
		wantWorkspace string
		wantBranch    string
	}{
		{
			name:          "manifest supplies cwd and branch",
			events:        prompt,
			manifest:      "id: s1\ncwd: /Users/fixture-user/code/demo\ngit_root: /Users/fixture-user/code/demo\nbranch: main\n",
			wantProject:   "demo",
			wantWorkspace: "/Users/fixture-user/code/demo",
			wantBranch:    "main",
		},
		{
			name:          "git_root used when cwd absent",
			events:        prompt,
			manifest:      "id: s1\ngit_root: /Users/fixture-user/code/other\nbranch: dev\n",
			wantProject:   "other",
			wantWorkspace: "/Users/fixture-user/code/other",
			wantBranch:    "dev",
		},
		{
			name: "event cwd still wins",
			events: `{"data":{"content":"hello","cwd":"/Users/fixture-user/code/fromevent"},` +
				`"id":"evt-1","parentId":null,"timestamp":"2026-08-17T10:00:00.000Z","type":"user"}` + "\n",
			manifest:      "id: s1\ncwd: /Users/fixture-user/code/demo\nbranch: main\n",
			wantProject:   "fromevent",
			wantWorkspace: "/Users/fixture-user/code/fromevent",
			wantBranch:    "main",
		},
		{
			name:          "no manifest degrades to unknown",
			events:        prompt,
			manifest:      "",
			wantProject:   "unknown",
			wantWorkspace: "",
			wantBranch:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeCopilotSession(t, root, "4374b22c-165a-43a0-983b-344dbd503b03", tt.events, tt.manifest)
			result := scan(t, root)
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			got := result.Records[0]
			if got.Project != tt.wantProject {
				t.Fatalf("Project = %q, want %q", got.Project, tt.wantProject)
			}
			if got.Workspace != tt.wantWorkspace {
				t.Fatalf("Workspace = %q, want %q", got.Workspace, tt.wantWorkspace)
			}
			if got.Branch != tt.wantBranch {
				t.Fatalf("Branch = %q, want %q", got.Branch, tt.wantBranch)
			}
		})
	}
}
