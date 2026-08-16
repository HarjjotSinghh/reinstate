package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Cline()) }

// Cline is the official Cline product descriptor (T0).
//
// The product is identified (editor extension + CLI). Dual-platform probes
// are absent, so the shipped tier is T0 (layout_unverified). There is no
// index source, no F3 scanner, and no transcript reader.
func Cline() agents.Descriptor {
	return agents.Descriptor{
		Key:         "cline",
		DisplayName: "Cline",
		Vendor:      "Cline",
		DocsURL:     "https://docs.cline.bot/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyEmbeddedDB,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			// Official docs name CLINE_DATA_DIR as the ~/.cline/data/ override.
			// Roots stay empty until a dual-platform probe names the live store.
			RootEnv:    "CLINE_DATA_DIR",
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"data/settings/providers.json",
				"**/providers.json",
			},
		},
		Process: agents.ProcessSpec{
			Images:      []string{"cline"},
			NodeMarkers: []string{"saoudrizwan.claude-dev"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/cline.md",
		},
	}
}
