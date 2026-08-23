package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	groksrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/grok"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Grok()) }

// `grok --version` prints one line on stdout and nothing on stderr:
//
//	grok 1.0.5 (5115b46bc909)
//
// The parenthesised build id is metadata, not part of the version, so it is
// optional here — a build that omits it still yields a version. Anything that
// is not `grok <semver>` fails closed, which agentcheck reports as UNTESTED.
//
// The trailing release channel is also metadata and also optional. It was
// missed because it was measured from a transcription rather than from the
// bytes: the shipped CLI prints `grok 1.0.5 (5115b46bc909) [stable]` on macOS
// and `grok 1.0.5 (5115b46bc9) [stable]` on native Windows, and without this
// the pattern matched neither. Every Grok resume on every platform failed
// closed as UNTESTED and exited 5, so the T3 promotion did not work at all.
// Physical dual-platform acceptance is what found it.
var grokVersionPattern = regexp.MustCompile(
	`^grok ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))` +
		`(?: \([0-9A-Za-z][0-9A-Za-z._-]{0,63}\))?` +
		`(?: \[[0-9A-Za-z][0-9A-Za-z._-]{0,31}\])?$`,
)

// Grok is the shipped Grok Build descriptor (T4).
func Grok() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentGrok,
		DisplayName: "Grok Build",
		Vendor:      "xAI",
		DocsURL:     "https://docs.x.ai",
		Tier:        agents.TierHandoffTo,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "GROK_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".grok")}}
			},
			Marker:      "sessions",
			Layout:      "sessions-summary-json",
			SessionGlob: "sessions/**/summary.json",
			ProjectKey:  agents.ProjectKeyURLEncoding,
			// A 2026-08-17 Windows probe never reached sessions/: bundled/
			// (137 MB binaries, skills, personas) and marketplace-cache/
			// (cloned plugin git trees) consumed the walk. Those are the
			// install, not the session store. auth.json is a credential.
			Excluded: append([]string{
				"bundled",
				"marketplace-cache",
				"bin",
				"downloads",
				"docs",
				"skills",
				"auth.json",
				"auth.json.lock",
				"mcp_credentials.json",
			}, groksrc.Excluded...),
		},
		// Measured from `grok --help` on Grok Build 1.0.5:
		//
		//	-r, --resume [<SESSION_ID_OR_TITLE>]
		//	    --fork-session
		//	-c, --continue
		//	-s, --session-id <SESSION_ID>   new conversation with this UUID;
		//	                                must not already exist under the
		//	                                target session directory
		//
		// --resume falls back to title matching for any value that is not
		// UUID-shaped, and titles are neither unique nor stable, so
		// SessionIDPattern makes a title unrepresentable in this position.
		// NewSession is the opposite direction: it starts a *new* session with
		// a caller-chosen id and never resumes.
		Native: &agents.NativeSpec{
			Executable:       "grok",
			Resume:           []string{"--resume", "{{.SessionID}}"},
			Fork:             []string{"--resume", "{{.SessionID}}", "--fork-session"},
			Continue:         []string{"--continue"},
			NewSession:       []string{"--session-id", "{{.SessionID}}"},
			InitialPrompt:    agents.PromptArgv,
			SessionIDPattern: sessionindex.GrokSessionIDPattern,
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parseGrokVersion,
			// Pinned to the single build measured on the macOS acceptance host
			// (2026-08-22, `grok --version` = "grok 1.0.5 (5115b46bc909)").
			// The range widens only when another build is physically measured.
			Min: sessionindex.GrokMinVerifiedVersion,
			Max: sessionindex.GrokMaxVerifiedVersion,
		},
		// Grok Build ships a native binary (Mach-O on macOS), not a node
		// launcher, so an image match is the whole recognition rule and
		// NodeMarkers stay empty. `nativeVariant` already covers the
		// per-target executable names the vendor publishes. The resume argv
		// carries the session UUID, so preflight's agent.active probe can
		// scope "is this session already open" rather than only "is grok
		// running".
		Process: agents.ProcessSpec{
			Images: []string{"grok"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/grok.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-21-macos-grok.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-grok.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/grok/macos",
				"testdata/sessionindex/grok/windows",
				"testdata/handoff/grok",
			},
			// The committed Grok device rows to date (C1-C6, D1-D5, G3) are
			// index, inspect and handoff-source rows. The physical resume
			// journey this tier requires is specified in
			// docs/testing/grok-native-resume-acceptance.md and is not yet
			// recorded on either platform.
			DeviceReports: []string{
				"docs/testing/results/2026-08-22-macos-grok-t3.md",
				"docs/testing/results/2026-08-22-windows-grok-t3.md",
				"docs/testing/results/2026-08-23-windows-grok-t4.md",
			},
		},
		NewIndexSource: groksrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return transcript.NewGrokReader(), nil
		},
		NewTarget: func(env agents.Env) (handoff.HandoffTarget, error) {
			return &handoff.GrokTarget{Root: env.FixtureRoot}, nil
		},
	}
}

func parseGrokVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, grokVersionPattern)
}
