package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Cursor()) }

// Cursor is the Cursor CLI descriptor (T0, layout_unverified).
//
// Catalog key `cursor` is the terminal agent only. The in-editor agent is a
// different product and is not this key. Dual-platform probes are absent, so
// there is no index source and no reader.
func Cursor() agents.Descriptor {
	return agents.Descriptor{
		Key:         "cursor",
		DisplayName: "Cursor CLI",
		Vendor:      "Anysphere",
		DocsURL:     "https://cursor.com/docs/cli/overview",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyEmbeddedDB,
		T0Reason:    agents.T0LayoutUnverified,
		Storage: agents.StorageSpec{
			// Roots stay unset deliberately. A 2026-08-16 macOS observation
			// found ~/.cursor/projects/<path-slug>/agent-transcripts holding
			// the *editor* agent's conversations while the CLI's documented
			// ~/.cursor/chats did not exist. Declaring ~/.cursor here would
			// resolve on the editor's tree and file another product's
			// sessions under this key. CURSOR_CONFIG_DIR relocates
			// cli-config.json, not a confirmed session store.
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"cli-config.json",
				"mcp.json",
				"**/mcp.json",
				"worktrees",
			},
		},
		Process: agents.ProcessSpec{
			// Official docs name the binary `agent`. That basename collides
			// with other tools, so the process image is the specific
			// `cursor-agent` name until a probe can tell them apart.
			Images: []string{"cursor-agent"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/cursor.md",
		},
	}
}
