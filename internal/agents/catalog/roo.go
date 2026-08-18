package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Roo()) }

// Roo is the official Roo Code product descriptor (T0).
//
// The product is identified (editor extension RooVeterinaryInc.roo-cline).
// Dual-platform probes are absent, so the shipped tier is T0
// (layout_unverified). There is no index source, no F3 scanner, and no
// transcript reader.
func Roo() agents.Descriptor {
	return agents.Descriptor{
		Key:         "roo",
		DisplayName: "Roo Code",
		Vendor:      "Roo",
		DocsURL:     "https://docs.roocode.com/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyEmbeddedDB,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			// Official docs name roo-cline.customStoragePath, a VS Code
			// setting, not an environment variable. Roots stay empty until
			// a dual-platform probe names the live store.
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"settings",
				"**/settings",
			},
		},
		Process: agents.ProcessSpec{
			NodeMarkers: []string{"RooVeterinaryInc.roo-cline"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/roo.md",
		},
	}
}
