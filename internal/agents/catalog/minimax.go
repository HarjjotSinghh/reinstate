package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(MiniMaxCode()) }

// MiniMaxCode is MiniMax's official desktop coding harness (T0).
//
// The public catalog key is minimax-code, not minimax: Token Plan API keys
// (sk-cp-*) drive MiniMax models inside other harnesses, and mmx is a
// different platform CLI. Reinstate indexes harnesses, not models.
func MiniMaxCode() agents.Descriptor {
	return agents.Descriptor{
		Key:         "minimax-code",
		DisplayName: "MiniMax",
		Vendor:      "MiniMax",
		DocsURL:     "https://agent.minimax.io/docs/code/welcome",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyRemote,
		T0Reason:    agents.T0LayoutUnverified,
		Process: agents.ProcessSpec{
			Images: []string{"minimax"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/minimax.md",
		},
	}
}
