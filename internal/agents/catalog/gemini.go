package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	geminisrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/gemini"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Gemini()) }

// yargs .version(getVersion()) prints package.json version as one line.
// Nightly/preview suffixes and the vendor fallback "unknown" fail closed.
var geminiVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

// Gemini is the shipped Gemini CLI descriptor (T2).
// VersionSpec records the maintainer pin (latest stable @google/gemini-cli).
// NativeSpec stays unset: T3 still needs dual-platform physical resume.
func Gemini() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentGemini,
		DisplayName: "Gemini CLI",
		Vendor:      "Google",
		DocsURL:     "https://github.com/google-gemini/gemini-cli",
		Tier:        agents.TierHandoffFrom,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "GEMINI_CLI_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".gemini")}}
			},
			Marker:      "tmp",
			Layout:      "tmp-chats-session-json",
			SessionGlob: "tmp/*/chats/session-*.json*",
			ProjectKey:  agents.ProjectKeyPathHash,
			// Antigravity CLI installs into ~/.gemini/antigravity-cli and
			// keeps an OAuth token there on Linux. It is a separate catalog
			// agent, so this root must not be walked into it.
			Excluded: append([]string{
				"antigravity-cli",
				"oauth_creds.json",
				"google_accounts.json",
			}, geminisrc.Excluded...),
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parseGeminiVersion,
			Min:   "0.55.1",
			Max:   "0.55.1",
		},
		Process: agents.ProcessSpec{
			Images: []string{"gemini"},
		},
		Evidence: agents.Evidence{
			Fixtures: []string{
				"testdata/sessionindex/gemini/macos",
				"testdata/sessionindex/gemini/windows",
				"testdata/handoff/gemini",
			},
		},
		NewIndexSource: geminisrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return &transcript.GeminiReader{}, nil
		},
	}
}

func parseGeminiVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, geminiVersionPattern)
}
