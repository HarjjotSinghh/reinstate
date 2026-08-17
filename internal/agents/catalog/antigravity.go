package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Antigravity()) }

// Antigravity is the Antigravity CLI descriptor (T0, layout_unverified).
//
// Google retired the individual OAuth path for Gemini CLI on 2026-06-18 and
// named Antigravity CLI the destination for those users, so this is where the
// Gemini CLI consumer population went rather than a speculative addition.
//
// It nests inside Gemini CLI's root at ~/.gemini/antigravity-cli, which is why
// the Gemini descriptor excludes that subtree: two catalog agents sharing one
// home directory must not index each other's files.
//
// No probe has observed it. There is no index source, reader, target, or sync
// adapter.
func Antigravity() agents.Descriptor {
	return agents.Descriptor{
		Key:         "antigravity",
		DisplayName: "Antigravity CLI",
		Vendor:      "Google",
		DocsURL:     "https://www.antigravity.google/docs/cli/install/",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyHomeTree,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".gemini", "antigravity-cli")}}
			},
			// Unverified. Vendor docs place a workspace-keyed conversation
			// cache under cache/, and settings.json is written sparsely so it
			// cannot be relied on to exist.
			Marker: "cache",
			Excluded: []string{
				// Linux keeps the OAuth token as a plain file in this tree.
				// macOS uses the Keychain, but the descriptor is one contract
				// across platforms.
				"antigravity-oauth-token",
				"settings.json",
				"keybindings.json",
				"mcp_config.json",
				"**/antigravity-oauth-token",
			},
		},
		Process: agents.ProcessSpec{
			Images: []string{"agy"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/antigravity.md",
		},
	}
}
