package transcript

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	fixtureProjectID = "github.com/example/demo"
	fixtureWorkspace = "/Users/fixture-user/code/demo"
	fixtureHome      = "/Users/fixture-user"
	fixtureRepoToken = "${REPO:github.com/example/demo}"
)

// fixtureHomeEnvironment pins the home used for ${HOME} tokenization. Only the
// string is used; readers never read the home tree.
func fixtureHomeEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", fixtureHome)
	t.Setenv("USERPROFILE", fixtureHome)
}

func absolutePathRecord(t *testing.T, agent, name, file string) sessionindex.Record {
	t.Helper()
	return sessionindex.Record{
		ID:         "absolute-paths",
		Agent:      agent,
		Project:    fixtureProjectID,
		Workspace:  fixtureWorkspace,
		SourcePath: filepath.Join(repoRoot(t), "testdata", "handoff", agent, name, file),
	}
}

func parseRecord(t *testing.T, reader Reader, rec sessionindex.Record) []capsule.Event {
	t.Helper()
	boundary, err := reader.Snapshot(context.Background(), rec)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	events, _, err := reader.Parse(context.Background(), boundary)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return events
}

func toolInputByCallID(t *testing.T, events []capsule.Event, callID string) string {
	t.Helper()
	for _, ev := range events {
		if ev.Kind != capsule.KindToolCall || ev.CallID != callID {
			continue
		}
		for _, block := range ev.Blocks {
			if block.Type == capsule.BlockTypeToolInput {
				return block.Text
			}
		}
	}
	t.Fatalf("no tool_input block for call %q in %d events", callID, len(events))
	return ""
}

// TestClaudeToolInputPathsAreTokenized reproduces the rc.1 blocker where the
// Claude reader lifted a Read tool's absolute file_path into the capsule
// verbatim, which capsule validation correctly rejects.
func TestClaudeToolInputPathsAreTokenized(t *testing.T) {
	fixtureHomeEnvironment(t)
	rec := absolutePathRecord(t, "claude", "absolute-paths",
		filepath.Join("projects", "-Users-fixture-user-code-demo", "session-syn-001.jsonl"))
	events := parseRecord(t, &ClaudeReader{}, rec)

	repoInput := toolInputByCallID(t, events, "call_read_repo")
	if !strings.Contains(repoInput, fixtureRepoToken+"/calc.go") {
		t.Fatalf("repo tool input = %s, want %s/calc.go", repoInput, fixtureRepoToken)
	}
	homeInput := toolInputByCallID(t, events, "call_read_home")
	if !strings.Contains(homeInput, "${HOME}/.config/demo/settings.json") {
		t.Fatalf("home tool input = %s, want ${HOME}/.config/demo/settings.json", homeInput)
	}
	outsideInput := toolInputByCallID(t, events, "call_read_outside")
	if !strings.Contains(outsideInput, "${EXTERNAL:") {
		t.Fatalf("outside-root tool input = %s, want an ${EXTERNAL:...} token", outsideInput)
	}
	if strings.Contains(outsideInput, "/etc/fixture-hosts") {
		t.Fatalf("outside-root tool input leaked its absolute path: %s", outsideInput)
	}
	bashInput := toolInputByCallID(t, events, "call_bash")
	if !strings.Contains(bashInput, fixtureRepoToken+"/calc_test.go") {
		t.Fatalf("bash paths list = %s, want %s/calc_test.go", bashInput, fixtureRepoToken)
	}
	if !strings.Contains(bashInput, "go test ./...") {
		t.Fatalf("bash command body was mangled: %s", bashInput)
	}

	assertNoAbsolutePaths(t, events)
}

// TestCodexToolInputPathsAreTokenized covers the same latent defect in the
// Codex reader, whose rc.1 fixtures happened not to carry an absolute path.
func TestCodexToolInputPathsAreTokenized(t *testing.T) {
	fixtureHomeEnvironment(t)
	rec := absolutePathRecord(t, "codex", "absolute-paths",
		"rollout-2026-08-01T16-00-00-00000000-0000-4000-8000-00000000ab01.jsonl")
	events := parseRecord(t, &CodexReader{}, rec)

	repoInput := toolInputByCallID(t, events, "call_read_repo")
	if !strings.Contains(repoInput, fixtureRepoToken+"/calc.go") {
		t.Fatalf("repo tool input = %s, want %s/calc.go", repoInput, fixtureRepoToken)
	}
	homeInput := toolInputByCallID(t, events, "call_read_home")
	if !strings.Contains(homeInput, "${HOME}/.config/demo/settings.json") {
		t.Fatalf("home tool input = %s, want ${HOME}/.config/demo/settings.json", homeInput)
	}
	shellInput := toolInputByCallID(t, events, "call_shell")
	if !strings.Contains(shellInput, fixtureRepoToken) {
		t.Fatalf("shell workdir = %s, want %s", shellInput, fixtureRepoToken)
	}
	if !strings.Contains(shellInput, "${EXTERNAL:") {
		t.Fatalf("shell argv = %s, want an ${EXTERNAL:...} token for /bin/fixture-sh", shellInput)
	}
	if !strings.Contains(shellInput, "go test ./...") {
		t.Fatalf("shell command body was mangled: %s", shellInput)
	}

	assertNoAbsolutePaths(t, events)
}

// TestExternalTokensAreStableAndOpaque proves outside-root paths collapse to a
// stable, non-reversible token so repeated references stay comparable.
func TestExternalTokensAreStableAndOpaque(t *testing.T) {
	fixtureHomeEnvironment(t)
	rec := absolutePathRecord(t, "claude", "absolute-paths",
		filepath.Join("projects", "-Users-fixture-user-code-demo", "session-syn-001.jsonl"))
	first := toolInputByCallID(t, parseRecord(t, &ClaudeReader{}, rec), "call_read_outside")
	second := toolInputByCallID(t, parseRecord(t, &ClaudeReader{}, rec), "call_read_outside")
	if first != second {
		t.Fatalf("external token is not stable: %s vs %s", first, second)
	}
	if !strings.Contains(first, pathmap.ExternalPrefix) || strings.Contains(first, "/etc/") {
		t.Fatalf("external path was not replaced by a token: %s", first)
	}
	if !strings.Contains(first, "fixture-hosts") {
		t.Fatalf("external token dropped the portable basename: %s", first)
	}
}

// assertNoAbsolutePaths enforces the reader-boundary invariant: no emitted
// block value may be a string capsule canonicalization rejects.
func assertNoAbsolutePaths(t *testing.T, events []capsule.Event) {
	t.Helper()
	for _, ev := range events {
		for _, block := range ev.Blocks {
			for label, value := range map[string]string{
				"text": block.Text, "ref": block.Ref, "path": block.Path,
			} {
				if capsule.AbsolutePathForbidden(value) {
					t.Fatalf("event %s block %s emitted an absolute path: %q", ev.ID, label, value)
				}
			}
			for key, value := range block.Meta {
				if capsule.AbsolutePathForbidden(value) {
					t.Fatalf("event %s meta %q emitted an absolute path: %q", ev.ID, key, value)
				}
			}
		}
	}
}
