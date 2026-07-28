package processcheck

import (
	"context"
	"testing"
)

func TestMatchesAgentProcess(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		image       string
		commandLine string
		want        bool
	}{
		{name: "claude native", agent: "claude", image: "/usr/local/bin/claude", want: true},
		{name: "claude windows", agent: "claude", image: "claude.exe", want: true},
		{name: "claude node", agent: "claude", image: "node", commandLine: "node /opt/node_modules/@anthropic-ai/claude-code/cli.js", want: true},
		{name: "codex native variant", agent: "codex", image: "codex-aarch64-apple-darwin", want: true},
		{name: "codex node windows", agent: "codex", image: "node.exe", commandLine: `node C:\npm\node_modules\@openai\codex\bin\codex.js`, want: true},
		{name: "codex host is not cli", agent: "codex", image: "codex-code-mode-host", want: false},
		{name: "reinstate agent argument", agent: "claude", image: "reinstate", commandLine: "reinstate pull --agent claude", want: false},
		{name: "unrelated node", agent: "codex", image: "node", commandLine: "node server.js --label codex", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesAgentProcess(test.agent, test.image, test.commandLine); got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}

func TestNormalizeAgentRejectsUnknown(t *testing.T) {
	if _, err := normalizeAgent("cursor"); err == nil {
		t.Fatal("unsupported agent accepted")
	}
	for _, agent := range []string{"Claude", "  codex  "} {
		if _, err := normalizeAgent(agent); err != nil {
			t.Fatalf("supported agent %q rejected: %v", agent, err)
		}
	}
}

// TestSessionBusyScoping covers the behavior that matters to a developer with
// unrelated agents running: only a process holding the target session file may
// block a restore.
func TestSessionBusyScoping(t *testing.T) {
	// A machine with several agents alive, none of them holding the target.
	allProcesses := []Process{
		{PID: 100, Image: "/usr/local/bin/claude"},
		{PID: 200, Image: "codex"},
		{PID: 300, Image: "node", CommandLine: "node /opt/node_modules/@openai/codex/bin/codex.js"},
		{PID: 400, Image: "vim", CommandLine: "vim notes.txt"},
	}

	tests := []struct {
		name       string
		agent      string
		holders    []int
		supported  bool
		wantBusy   bool
		wantScoped bool
	}{
		{
			name:       "unrelated agents running but nothing holds the target",
			agent:      "codex",
			holders:    nil,
			supported:  true,
			wantBusy:   false,
			wantScoped: true,
		},
		{
			name:       "the agent itself holds the target",
			agent:      "codex",
			holders:    []int{200},
			supported:  true,
			wantBusy:   true,
			wantScoped: true,
		},
		{
			name:       "a node-hosted agent holds the target",
			agent:      "codex",
			holders:    []int{300},
			supported:  true,
			wantBusy:   true,
			wantScoped: true,
		},
		{
			name:       "a non-agent process holds the target",
			agent:      "claude",
			holders:    []int{400},
			supported:  true,
			wantBusy:   false,
			wantScoped: true,
		},
		{
			name:       "a different agent holds the target",
			agent:      "claude",
			holders:    []int{200},
			supported:  true,
			wantBusy:   false,
			wantScoped: true,
		},
		{
			name:       "handle enumeration unavailable falls back to host-wide",
			agent:      "claude",
			supported:  false,
			wantBusy:   true,
			wantScoped: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			busy, scoped := decideSessionBusy(tc.agent, allProcesses, tc.holders, tc.supported)
			if busy != tc.wantBusy || scoped != tc.wantScoped {
				t.Fatalf("busy=%v scoped=%v, want busy=%v scoped=%v",
					busy, scoped, tc.wantBusy, tc.wantScoped)
			}
		})
	}
}

func TestSessionBusyRejectsUnsupportedAgent(t *testing.T) {
	if _, _, err := SessionBusy(context.Background(), "emacs", "/tmp/x.jsonl"); err == nil {
		t.Fatal("unsupported agent accepted")
	}
}
