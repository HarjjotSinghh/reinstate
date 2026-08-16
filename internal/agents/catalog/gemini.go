package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	geminisrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/gemini"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Gemini()) }

// Gemini is the shipped Gemini CLI descriptor (T2).
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
			Excluded:    geminisrc.Excluded,
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
