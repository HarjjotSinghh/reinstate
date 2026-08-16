package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	groksrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/grok"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Grok()) }

// Grok is the shipped Grok Build descriptor (T2).
func Grok() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentGrok,
		DisplayName: "Grok Build",
		Vendor:      "xAI",
		DocsURL:     "https://docs.x.ai",
		Tier:        agents.TierHandoffFrom,
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
			Excluded:    groksrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images: []string{"grok"},
		},
		Evidence: agents.Evidence{
			Fixtures: []string{
				"testdata/sessionindex/grok/macos",
				"testdata/sessionindex/grok/windows",
				"testdata/handoff/grok",
			},
		},
		NewIndexSource: groksrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return transcript.NewGrokReader(), nil
		},
	}
}
