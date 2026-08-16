package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Kimi()) }

// Kimi is the Kimi Code CLI descriptor. Dual-platform probes are absent, so
// the descriptor stays at T0 (layout_unverified).
func Kimi() agents.Descriptor {
	return agents.Descriptor{
		Key:         "kimi",
		DisplayName: "Kimi Code CLI",
		Vendor:      "Moonshot AI",
		DocsURL:     "https://www.kimi.com/code/docs/en/kimi-code-cli/guides/sessions.html",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyHomeTree,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			RootEnv: "KIMI_CODE_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				// Official moonshot/kimi.com root first; kimi-cli.com mirror second.
				// Neither root is verified by a committed probe.
				return []agents.Root{
					{Path: home.Join(".kimi-code")},
					{Path: home.Join(".kimi")},
				}
			},
			Marker:      "sessions",
			SessionGlob: "sessions/*/*/state.json",
			Excluded: []string{
				"credentials",
				"mcp-oauth",
				"agents/agent-*",
				"subagents",
			},
		},
		Process: agents.ProcessSpec{
			Images: []string{"kimi"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/kimi.md",
		},
	}
}
