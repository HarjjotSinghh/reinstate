package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Pi()) }

// Pi is the Pi coding agent descriptor. It stays at T0 until macOS and native
// Windows AGENT-PROBE-V1 artifacts exist. Vendor docs describe an F1 JSONL
// tree; they are not a tier promotion.
func Pi() agents.Descriptor {
	return agents.Descriptor{
		Key:         "pi",
		DisplayName: "Pi",
		Vendor:      "earendil-works",
		DocsURL:     "https://pi.dev/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyHomeTree,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			RootEnv: "PI_CODING_AGENT_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".pi", "agent")}}
			},
			Marker:      "sessions",
			Layout:      "sessions-cwd-slug-jsonl",
			SessionGlob: "sessions/**/*.jsonl",
			ProjectKey:  agents.ProjectKeyPathSlug,
			Excluded:    piExcluded,
		},
		Process: agents.ProcessSpec{
			Images: []string{"pi"},
			Identify: []agents.EnvIdentity{
				{Name: "PI_CODING_AGENT", Value: "true"},
				{Name: "AI_AGENT", Value: "pi"},
			},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/pi.md",
		},
	}
}

// piExcluded keeps credentials, caches, packages, and HTML exports out of
// any future walk. PI_CODING_AGENT_SESSION_DIR is a separate override; it is
// not RootEnv because the default session tree lives under the config root.
var piExcluded = []string{
	"auth.json",
	"**/auth.json",
	"npm",
	"git",
	"extensions",
	"skills",
	"prompts",
	"themes",
	"models-store.json",
	"**/*.html",
}
