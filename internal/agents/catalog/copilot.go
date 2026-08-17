package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Copilot()) }

// Copilot is the GitHub Copilot CLI descriptor (T0).
// Vendor docs describe ~/.copilot/session-state/. A 2026-08-17 rename-aside
// probe showed an old session ID did not reappear in a fresh tree. The layout
// is local files; T0 stays until a reader exists.
func Copilot() agents.Descriptor {
	return agents.Descriptor{
		Key:         "copilot",
		DisplayName: "GitHub Copilot CLI",
		Vendor:      "GitHub",
		DocsURL:     "https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-copilot-cli",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyHomeTree,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			RootEnv: "COPILOT_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".copilot")}}
			},
			Marker:     "session-state",
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"config.json",
				"mcp-oauth-config",
				"mcp-secrets",
			},
		},
		Process: agents.ProcessSpec{
			Images:      []string{"copilot"},
			NodeMarkers: []string{"/@github/copilot/"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/copilot.md",
		},
	}
}
