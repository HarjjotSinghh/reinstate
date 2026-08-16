package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Copilot()) }

// Copilot is the GitHub Copilot CLI descriptor (T0).
// Vendor docs describe ~/.copilot/session-state/, but no cache-clear or
// re-login probe has classified that tree as local-authoritative vs a
// rebuildable account cache, so the layout stays unverified.
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
