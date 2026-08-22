package processcheck

import "testing"

// TestMatchesAgentProcessSurvivesTruncatedImage covers the shape macOS `ps`
// actually returns for an agent Reinstate launched itself.
//
// Reinstate resumes a vendor by its verified absolute path, and
// `ps -axo pid=,comm=,args=` fixes the `comm` column at sixteen characters, so
// the image arrives as the first sixteen characters of that path. Measured on
// macOS 25.5.0: a process started as
// `/Users/<user>/.opencode/bin/opencode --session <id>` is reported with the
// image `/Users/harjjotsi`, whose basename matches no agent. Matching on the
// image alone therefore makes every self-launched agent invisible to the
// liveness check, which then reports "nothing is running" for a session that is
// open right now — a safety check that always passes.
func TestMatchesAgentProcessSurvivesTruncatedImage(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		image       string
		commandLine string
		want        bool
	}{
		{
			name:        "opencode truncated absolute path",
			agent:       "opencode",
			image:       "/Users/exampleus",
			commandLine: "/Users/exampleuser/.opencode/bin/opencode --session ses_abc123",
			want:        true,
		},
		{
			name:        "claude truncated absolute path",
			agent:       "claude",
			image:       "/usr/local/bin/c",
			commandLine: "/usr/local/bin/claude --resume 1cf4ab6d-3e36-424d-8f30-4f41858b7f20",
			want:        true,
		},
		{
			name:        "codex truncated absolute path",
			agent:       "codex",
			image:       "/opt/homebrew/bi",
			commandLine: "/opt/homebrew/bin/codex resume 0199f1a2-1111-7000-8000-aaaaaaaaaaaa",
			want:        true,
		},
		{
			name:        "truncated node host still needs its marker",
			agent:       "claude",
			image:       "/usr/local/bin/n",
			commandLine: "/usr/local/bin/node /opt/node_modules/@anthropic-ai/claude-code/cli.js",
			want:        true,
		},
		{
			name:        "truncated node host without its marker stays unmatched",
			agent:       "claude",
			image:       "/usr/local/bin/n",
			commandLine: "/usr/local/bin/node /opt/node_modules/some-other-tool/cli.js",
			want:        false,
		},
		{
			name:        "truncated path to an unrelated binary stays unmatched",
			agent:       "opencode",
			image:       "/Users/exampleus",
			commandLine: "/Users/exampleuser/.local/bin/ripgrep --files",
			want:        false,
		},
		{
			name:        "an agent named only in an argument is not a match",
			agent:       "opencode",
			image:       "reinstate",
			commandLine: "reinstate resume opencode:ses_abc123",
			want:        false,
		},
		{
			name:        "windows argv0 with extension",
			agent:       "opencode",
			image:       "",
			commandLine: `C:\Users\example\AppData\Local\opencode\opencode.exe --session ses_abc123`,
			want:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesAgentProcess(test.agent, test.image, test.commandLine); got != test.want {
				t.Fatalf("matchesAgentProcess(%q, %q, %q) = %v, want %v",
					test.agent, test.image, test.commandLine, got, test.want)
			}
		})
	}
}

// TestSessionBusyScopesTruncatedImageToTheSession pins the end-to-end effect:
// a truncated image must still let the session-scoped signals decide, and must
// not turn a truncated row into a match for every session.
func TestSessionBusyScopesTruncatedImageToTheSession(t *testing.T) {
	const sessionID = "ses_abc123"
	procs := []Process{{
		PID:         4242,
		Image:       "/Users/exampleus",
		CommandLine: "/Users/exampleuser/.opencode/bin/opencode --session " + sessionID,
	}}
	if !decideSessionBusy("opencode", Target{SessionID: sessionID}, procs, nil, nil) {
		t.Fatal("a running opencode on this exact session was reported as free")
	}
	if decideSessionBusy("opencode", Target{SessionID: "ses_other"}, procs, nil, nil) {
		t.Fatal("a running opencode on a different session was reported as busy")
	}
}
