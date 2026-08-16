package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Qwen()) }

// Qwen is the Qwen Code descriptor.
//
// Official product is identified. Session-file layout is not verified on
// macOS and native Windows, so the shipped tier is T0 (layout_unverified).
// There is no index source and no transcript reader.
func Qwen() agents.Descriptor {
	return agents.Descriptor{
		Key:         "qwen",
		DisplayName: "Qwen Code",
		Vendor:      "Alibaba",
		DocsURL:     "https://qwenlm.github.io/qwen-code-docs/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyHomeTree,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			RootEnv: "QWEN_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".qwen")}}
			},
			Excluded: []string{
				"settings.json",
				".env",
				"**/.env",
			},
		},
		Process: agents.ProcessSpec{
			Images:      []string{"qwen"},
			NodeMarkers: []string{"@qwen-code/qwen-code"},
			Identify: []agents.EnvIdentity{
				{Name: "QWEN_CODE", Value: "1"},
			},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/qwen.md",
		},
	}
}
