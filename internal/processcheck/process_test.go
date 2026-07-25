package processcheck

import "testing"

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
