package processcheck

import (
	"context"
	"path/filepath"
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
		{name: "grok native", agent: "grok", image: "/usr/local/bin/grok", want: true},
		{name: "gemini native", agent: "gemini", image: "gemini.exe", want: true},
		{name: "opencode native", agent: "opencode", image: "opencode", want: true},
		{name: "grok is not claude", agent: "claude", image: "grok", want: false},
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
	for _, agent := range []string{"Claude", "  codex  ", "grok", "gemini", "opencode"} {
		if _, err := normalizeAgent(agent); err != nil {
			t.Fatalf("supported agent %q rejected: %v", agent, err)
		}
	}
}

// TestSessionBusyDetectsLiveClaudeWithoutFileHandle is the regression a
// handle-only implementation cannot pass.
//
// Claude Code appends to its session file and closes it again, so a live
// Claude Code session holds no handle at all. Measured on macOS and reproduced
// on Windows, where Restart Manager correctly reported no holder. Treating that
// as "not in use" let a restore target a session someone was working in.
func TestSessionBusyDetectsLiveClaudeWithoutFileHandle(t *testing.T) {
	const sessionID = "1cf4ab6d-3e36-424d-8f30-4f41858b7f20"
	projectRoot := filepath.Join("/home", "dev", "projects", "acceptance")
	target := Target{
		SessionID:   sessionID,
		Path:        filepath.Join("/home", "dev", ".claude", "projects", "slug", sessionID+".jsonl"),
		ProjectRoot: projectRoot,
	}

	// A live `claude --resume <id>` holding no file handle whatsoever.
	resumed := []Process{
		{PID: 100, Image: "/usr/local/bin/claude", CommandLine: "claude --resume " + sessionID},
	}
	if !decideSessionBusy("claude", target, resumed, nil, nil) {
		t.Fatal("live claude --resume on the exact session was reported as free")
	}

	// The same session chosen interactively, so the id never reaches argv and
	// project affinity is the only remaining signal.
	interactive := []Process{
		{PID: 101, Image: "/usr/local/bin/claude", CommandLine: "claude"},
	}
	cwds := map[int]string{101: filepath.Join(projectRoot, "src")}
	if !decideSessionBusy("claude", target, interactive, nil, cwds) {
		t.Fatal("live claude working inside the session's project was reported as free")
	}
}

// TestSessionBusyScoping covers the behavior that motivated scoping in the
// first place: a developer with unrelated agents running must not be blocked.
func TestSessionBusyScoping(t *testing.T) {
	const sessionID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	projectRoot := filepath.Join("/home", "dev", "projects", "acceptance")
	target := Target{
		SessionID:   sessionID,
		Path:        filepath.Join("/home", "dev", ".claude", "projects", "slug", sessionID+".jsonl"),
		ProjectRoot: projectRoot,
	}

	tests := []struct {
		name    string
		agent   string
		procs   []Process
		holders []int
		cwds    map[int]string
		want    bool
	}{
		{
			name:  "unrelated agents in other projects are ignored",
			agent: "codex",
			procs: []Process{
				{PID: 200, Image: "codex", CommandLine: "codex"},
				{PID: 300, Image: "node", CommandLine: "node /opt/node_modules/@openai/codex/bin/codex.js"},
			},
			cwds: map[int]string{200: "/home/dev/projects/other", 300: "/tmp"},
			want: false,
		},
		{
			name:    "an agent holding the exact file is busy",
			agent:   "codex",
			procs:   []Process{{PID: 200, Image: "codex"}},
			holders: []int{200},
			want:    true,
		},
		{
			name:    "a non-agent process holding the file does not count",
			agent:   "claude",
			procs:   []Process{{PID: 400, Image: "vim", CommandLine: "vim notes.txt"}},
			holders: []int{400},
			want:    false,
		},
		{
			name:  "a different agent naming the session does not count",
			agent: "claude",
			procs: []Process{{PID: 200, Image: "codex", CommandLine: "codex resume " + sessionID}},
			want:  false,
		},
		{
			name:  "codex resume names the session on its command line",
			agent: "codex",
			procs: []Process{{PID: 200, Image: "codex", CommandLine: "codex resume " + sessionID}},
			want:  true,
		},
		{
			name:  "a sibling directory is not inside the project root",
			agent: "claude",
			procs: []Process{{PID: 100, Image: "claude", CommandLine: "claude"}},
			cwds:  map[int]string{100: projectRoot + "-other"},
			want:  false,
		},
		{
			name:  "no agent running at all",
			agent: "claude",
			procs: []Process{{PID: 400, Image: "vim"}},
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideSessionBusy(tc.agent, target, tc.procs, tc.holders, tc.cwds); got != tc.want {
				t.Fatalf("busy=%v want %v", got, tc.want)
			}
		})
	}
}

// TestSessionBusyWithoutProjectRoot guards the case where no project mapping is
// configured, so affinity cannot contribute.
func TestSessionBusyWithoutProjectRoot(t *testing.T) {
	const sessionID = "11111111-2222-3333-4444-555555555555"
	target := Target{SessionID: sessionID, Path: "/home/dev/.claude/projects/slug/x.jsonl"}

	named := []Process{{PID: 1, Image: "claude", CommandLine: "claude --resume " + sessionID}}
	if !decideSessionBusy("claude", target, named, nil, map[int]string{1: "/anywhere"}) {
		t.Fatal("command-line signal ignored when no project root is configured")
	}
	idle := []Process{{PID: 1, Image: "claude", CommandLine: "claude"}}
	if decideSessionBusy("claude", target, idle, nil, map[int]string{1: "/anywhere"}) {
		t.Fatal("an unrelated claude became busy with no project root to compare")
	}
}

func TestSessionBusyAcceptsSourceOnlyAgents(t *testing.T) {
	for _, agent := range []string{"grok", "gemini", "opencode"} {
		if _, _, err := SessionBusy(context.Background(), agent, Target{Path: filepath.Join(t.TempDir(), "session.jsonl")}); err != nil {
			t.Fatalf("source-only agent %q busy check: %v", agent, err)
		}
	}
}

func TestSessionBusyRejectsUnsupportedAgent(t *testing.T) {
	if _, _, err := SessionBusy(context.Background(), "emacs", Target{Path: "/tmp/x.jsonl"}); err == nil {
		t.Fatal("unsupported agent accepted")
	}
}
