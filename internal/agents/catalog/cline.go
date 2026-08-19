package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Cline()) }

// Cline is the official Cline product descriptor (T0).
//
// A 2026-08-19 macOS probe named ~/.cline/data/sessions after cline 3.0.55.
// Native Windows is still missing, so the shipped tier stays T0. There is
// no index source and no reader.
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
			// 2026-08-19 macOS probe: live store is ~/.cline/data, override
			// CLINE_DATA_DIR / --data-dir. Native Windows is still missing,
			// so this stays T0.
			RootEnv: "CLINE_DATA_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".cline", "data")}}
			},
			Marker:      "sessions",
			SessionGlob: "sessions/*/*.json",
			Layout:      "sessions-id-json-plus-sqlite-index",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded: []string{
				"settings/providers.json",
				"**/providers.json",
				"locks",
				"logs",
				"cache",
			},
		},
		Process: agents.ProcessSpec{
			Images:      []string{"cline"},
			NodeMarkers: []string{"saoudrizwan.claude-dev"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/cline.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-19-macos-cline.json",
			},
		},
	}
}
