package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

func init() { agents.MustRegister(Amp()) }

// Amp is the shipped Amp descriptor (T0, server_backed, F5).
// Threads are authoritative on Amp Server. Do not add a network reader.
func Amp() agents.Descriptor {
	return agents.Descriptor{
		Key:         "amp",
		DisplayName: "Amp",
		Vendor:      "Amp",
		DocsURL:     "https://ampcode.com/manual",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyRemote,
		T0Reason:    agents.T0ServerBacked,
		Process: agents.ProcessSpec{
			Images: []string{"amp"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/amp.md",
		},
	}
}
