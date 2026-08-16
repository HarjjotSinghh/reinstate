package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(OpenHands()) }

// OpenHands is the OpenHands descriptor (T0, F5, server_backed).
//
// Conversations are owned by an Agent Server, Cloud, or Enterprise backend.
// A documented ~/.openhands host bind-mount is that server's persistence
// directory, not a first-class local session store. No index source, reader,
// target, or sync adapter.
func OpenHands() agents.Descriptor {
	return agents.Descriptor{
		Key:         "openhands",
		DisplayName: "OpenHands",
		Vendor:      "All Hands AI",
		DocsURL:     "https://docs.openhands.dev/overview/introduction",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyRemote,
		T0Reason:    agents.T0ServerBacked,
		Storage: agents.StorageSpec{
			// Documented server persistence dir. Not an indexable session store.
			RootEnv: "OH_PERSISTENCE_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".openhands")}}
			},
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"settings.json",
				"agent_settings.json",
				"mcp.json",
				"**/settings.json",
				"**/agent_settings.json",
				"**/mcp.json",
			},
		},
		Process: agents.ProcessSpec{
			Images: []string{"openhands", "agent-canvas"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/openhands.md",
		},
	}
}
