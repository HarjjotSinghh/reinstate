package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// TestCatalogNativeLaunchArgv pins the argv the catalog produces for each
// resumable agent, and — for Grok Build — pins the refusal that keeps a session
// *title* off a `grok` command line.
//
// `grok --resume [<SESSION_ID_OR_TITLE>]` resolves any value that is not
// UUID-shaped as a title, and two sessions in one directory can share a title.
// A title in that position would address a session Reinstate never resolved,
// so the substitution is refused rather than attempted.
func TestCatalogNativeLaunchArgv(t *testing.T) {
	const uuid = "01987654-3210-7890-abcd-ef0123456789"
	tests := []struct {
		name           string
		agent          string
		operation      string
		sessionID      string
		wantExecutable string
		wantArgs       []string
		wantOK         bool
	}{
		{
			name: "grok resume by uuid", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationResume, sessionID: uuid,
			wantExecutable: "grok", wantArgs: []string{"--resume", uuid}, wantOK: true,
		},
		{
			name: "grok fork by uuid", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationFork, sessionID: uuid,
			wantExecutable: "grok", wantArgs: []string{"--resume", uuid, "--fork-session"}, wantOK: true,
		},
		{
			name: "grok refuses a title", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationResume, sessionID: "fix the parser",
		},
		{
			name: "grok refuses a single-word title", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationResume, sessionID: "refactor",
		},
		{
			name: "grok refuses a near-uuid", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationFork, sessionID: "01987654-basic-0000-0000-000000000001",
		},
		{
			name: "grok refuses an empty id", agent: sessionindex.AgentGrok,
			operation: sessionindex.OperationResume, sessionID: "",
		},
		{
			// Claude declares no pattern, so its behaviour is unchanged by the
			// gate Grok introduced.
			name: "claude resume is unaffected", agent: sessionindex.AgentClaude,
			operation: sessionindex.OperationResume, sessionID: uuid,
			wantExecutable: "claude", wantArgs: []string{"--resume", uuid}, wantOK: true,
		},
		{
			name: "unknown agent", agent: "nope",
			operation: sessionindex.OperationResume, sessionID: uuid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executable, args, ok := catalogNativeLaunch(test.agent, test.operation, test.sessionID)
			if ok != test.wantOK {
				t.Fatalf("ok = %t, want %t (argv %s %v)", ok, test.wantOK, executable, args)
			}
			if !ok {
				return
			}
			if executable != test.wantExecutable || !reflect.DeepEqual(args, test.wantArgs) {
				t.Fatalf("argv = %s %v, want %s %v", executable, args, test.wantExecutable, test.wantArgs)
			}
		})
	}
}

// TestGrokTitleAddressableSessionIsNotLaunchable closes the loop through
// PlanLaunch: an indexed record carrying a non-UUID id is refused with the
// compatibility contract, not launched by name.
func TestGrokTitleAddressableSessionIsNotLaunchable(t *testing.T) {
	record := sessionindex.Record{
		ID:             "morning debugging",
		Agent:          sessionindex.AgentGrok,
		Workspace:      t.TempDir(),
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.GrokTitleAddressableReason,
	}
	for _, operation := range []string{sessionindex.OperationResume, sessionindex.OperationFork} {
		if _, err := sessionindex.PlanLaunch(record, operation); err == nil {
			t.Fatalf("PlanLaunch(%s) succeeded for a title-addressable Grok session", operation)
		}
	}
}

// TestDestinationAgentGateIsT4 pins the tier the `--to` / `--with` flags
// validate against. A T3 agent that is not a destination must be refused here,
// with a message naming the agents that can receive a handoff, rather than
// reaching the pipeline and failing as an unknown destination.
func TestDestinationAgentGateIsT4(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		valid bool
	}{
		{"claude", "claude", true},
		{"codex", "codex", true},
		{"grok is T3, not a destination", "grok", false},
		{"gemini is T2", "gemini", false},
		{"all is never a destination", "all", false},
		{"unknown", "nope", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDestinationAgent(test.agent)
			if test.valid {
				if err != nil {
					t.Fatalf("validateDestinationAgent(%q) = %v, want nil", test.agent, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateDestinationAgent(%q) = nil, want a usage refusal", test.agent)
			}
			if !strings.Contains(err.Error(), "destination agent") {
				t.Fatalf("refusal does not name the destination flag: %v", err)
			}
		})
	}
}
