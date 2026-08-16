package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Aider()) }

// Aider is the Aider descriptor.
//
// Official product is identified. Vendor docs describe F4 per-repository
// Markdown history, but dual-platform probes are absent, so the shipped tier
// is T0 (layout_unverified). There is no index source and no transcript
// reader. Roots stays nil so doctor --agents does not walk $HOME.
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
