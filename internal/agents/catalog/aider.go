package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Aider()) }

// Aider is the Aider descriptor.
//
// Official product is identified. A 2026-08-19 macOS probe saw aider 0.86.2
// on PATH. Vendor docs describe F4 per-repository Markdown history. Native
// Windows is still missing, so the shipped tier stays T0. Roots stays nil
// so doctor --agents does not walk $HOME.
func Aider() agents.Descriptor {
	return agents.Descriptor{
		Key:         "aider",
		DisplayName: "Aider",
		Vendor:      "Aider AI LLC",
		DocsURL:     "https://aider.chat/docs/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyProjectFile,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			SessionGlob: ".aider.chat.history.md",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded:    aiderExcluded,
		},
		Process: agents.ProcessSpec{
			Images: []string{"aider"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/aider.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-19-macos-aider.json",
			},
		},
	}
}

// aiderExcluded keeps credentials, input history, caches, and debug logs out
// of any future known-project walk. These are not session records.
var aiderExcluded = []string{
	".aider.conf.yml",
	"**/.aider.conf.yml",
	".env",
	"**/.env",
	".aider.input.history",
	".aider.model.settings.yml",
	".aider.model.metadata.json",
	".aider.tags.cache*",
	".aider.llm.history",
}
