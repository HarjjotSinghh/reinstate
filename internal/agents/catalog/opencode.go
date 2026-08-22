package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	opencodesrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/opencode"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(OpenCode()) }

// opencodeVersionPattern matches OpenCode's `--version` output, which is a bare
// stable version and nothing else. Anchoring both ends is what keeps a bare
// number from being read out of some other vendor's sentence.
var opencodeVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

// OpenCode is the shipped OpenCode descriptor (T4, F3).
func OpenCode() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentOpenCode,
		DisplayName: "OpenCode",
		Vendor:      "anomalyco",
		DocsURL:     "https://opencode.ai",
		Tier:        agents.TierHandoffTo,
		Family:      agents.FamilyEmbeddedDB,
		Storage: agents.StorageSpec{
			// OpenCode reads $XDG_DATA_HOME/opencode, so the variable names the
			// parent and the remaining segment is appended.
			RootEnv:       "XDG_DATA_HOME",
			RootEnvSuffix: "opencode",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".local", "share", "opencode")}}
			},
			Marker:     opencodesrc.DatabaseName,
			Layout:     "embedded-sqlite-session-store",
			ProjectKey: agents.ProjectKeyOpaqueID,
			Excluded:   opencodesrc.Excluded,
		},
		Native: &agents.NativeSpec{
			// OpenCode spells continuation as options on its default command
			// rather than as verbs: `--session <id>` continues one session,
			// `--continue` continues the newest one, and `--fork` is a modifier
			// on either. Fork is therefore resume plus a flag, not its own
			// subcommand, and must not be modelled on Codex's `codex fork <id>`.
			Executable: "opencode",
			Resume:     []string{"--session", "{{.SessionID}}"},
			Fork:       []string{"--session", "{{.SessionID}}", "--fork"},
			Continue:   []string{"--continue"},
			// A new session carries no {{.SessionID}} because OpenCode assigns
			// it: `--session <unknown-id>` refuses with "Session not found" and
			// creates nothing, so the destination id is only knowable after
			// launch. The flag is the destination target's own, so the two
			// cannot drift.
			NewSession:    []string{handoff.OpenCodeNewSessionFlag},
			InitialPrompt: agents.PromptArgv,
		},
		Version: &agents.VersionSpec{
			// `opencode --version` prints a bare stable version on stdout and
			// nothing on stderr. Min and Max are the single build physically
			// measured on macOS for this promotion; the range widens only as
			// further builds are measured, never by assumption.
			Args:  []string{"--version"},
			Parse: parseOpenCodeVersion,
			Min:   "1.18.21",
			Max:   "1.18.21",
		},
		Process: agents.ProcessSpec{
			// OpenCode ships as a single native executable, so the image name
			// is the whole signal; there is no node-hosted entry point to match.
			Images: []string{"opencode"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/opencode.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-21-macos-opencode.json",
				"docs/testing/results/agent-probes/2026-08-21-windows-opencode.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/opencode/macos",
				"testdata/sessionindex/opencode/windows",
				"testdata/handoff/opencode",
			},
			DeviceReports: []string{
				"docs/testing/results/2026-08-22-macos-opencode-t3-journey.md",
				"docs/testing/results/2026-08-22-windows-opencode-t3.md",
				"docs/testing/results/2026-08-22-macos-opencode-t4-journey.md",
				"docs/testing/results/2026-08-22-windows-opencode-t4.md",
			},
		},
		NewIndexSource: opencodesrc.NewSQLite,
		NewTarget: func(env agents.Env) (handoff.HandoffTarget, error) {
			return handoff.NewOpenCodeTarget(&handoff.OpenCodeTarget{Root: env.FixtureRoot}), nil
		},
		NewReader: func(env agents.Env) (transcript.Reader, error) {
			reader := transcript.NewOpenCodeReader(nil)
			reader.DataRoot = env.FixtureRoot
			reader.Getenv = env.LookupEnv
			return reader, nil
		},
	}
}

func parseOpenCodeVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, opencodeVersionPattern)
}
