package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Cursor()) }

// Cursor is the Cursor CLI descriptor (T0, layout_unverified).
//
// Catalog key `cursor` is the terminal agent only. The in-editor agent is a
// different product and is not this key. A 2026-08-17 native Windows session
// created ~/.cursor/chats; that directory is the marker. ~/.cursor/projects
// is the editor agent's tree and stays excluded. macOS still has no CLI
// session, so there is no dual-platform probe and no reader.
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
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".cursor")}}
			},
			// chats/ appears only after a CLI session. Without this marker
			// the root would resolve on a machine that only uses the editor.
			Marker:     "chats",
			ProjectKey: agents.ProjectKeyNone,
			// chats/ is the CLI session store. Every other sibling under
			// ~/.cursor belongs to the editor, extensions, or user skills.
			// A 2026-08-17 macOS walk without these exclusions emitted a
			// plan filename and drowned in skills/.
			Excluded: []string{
				"projects",
				"extensions",
				"plugins",
				"skills",
				"skills-cursor",
				"plans",
				"agents",
				"rules",
				"ai-tracking",
				"sandbox-policies",
				"worktrees",
				"cli-config.json",
				"mcp.json",
				"**/mcp.json",
				"ide_state.json",
				"argv.json",
				"hooks.json",
				"hooks.json.bak",
				"agent-cli-state.json",
				"statsig-cache.json",
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
