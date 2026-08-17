package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Qwen()) }

// Qwen is the Qwen Code descriptor.
//
// Official product is identified. Dual-platform probes exist, but they
// disagree on conversation file shape (macOS stub JSON vs Windows JSONL),
// so the shipped tier stays T0 (layout_unverified). There is no index
// source and no transcript reader.
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
			// macOS probe 2026-08-16 (qwen 0.21.12): conversations are at
			// projects/<slug>/chats/<slug>.json. Native Windows 2026-08-17
			// (qwen 0.21.13): projects/<slug>/chats/<uuid-v4>.jsonl. The
			// Gemini-fork hypothesis predicted tmp/, and tmp/<64-hex> does
			// exist, but it is not the conversation store.
			Marker: "projects",
			Excluded: []string{
				"settings.json",
				".env",
				"**/.env",
				// Configuration, not sessions. Left in, a populated skills
				// library is 176 directories of noise that crowds the actual
				// evidence out of a probe artifact.
				"skills",
				"extension-store",
				// The self-updater unpacks a full npm tree here. The
				// 2026-08-17 Windows probe spent its entire file budget on
				// node_modules — 289 chunk files and 61 font files — and the
				// two real conversations barely made the artifact.
				"updates",
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
