package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(Cursor()) }

// Cursor is the Cursor CLI descriptor (T0, layout_unverified).
//
// Catalog key `cursor` is the terminal agent only. The in-editor agent is a
// different product and is not this key.
//
// A 2026-08-17 macOS CLI session created ~/.cursor/chats. That directory is
// the marker; ~/.cursor is the candidate root; projects/ stays excluded so
// the editor agent's agent-transcripts are never filed under this key.
// Native Windows is still unprobed, so there is no index source and no reader.
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
			Marker:     "chats",
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"projects",
				"extensions",
				"plugins",
				"agents",
				"ai-tracking",
				"plans",
				"skills",
				"skills-cursor",
				"rules",
				"sandbox-policies",
				"ide_state.json",
				"argv.json",
				"hooks.json",
				"hooks.json.bak",
				"statsig-cache.json",
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
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-cursor.json",
			},
		},
	}
}
