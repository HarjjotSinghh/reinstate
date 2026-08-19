package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	cursorsrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/cursor"
)

func init() { agents.MustRegister(Cursor()) }

// Cursor is the Cursor CLI descriptor (T1, discover).
//
// Catalog key `cursor` is the terminal agent only. The in-editor agent is a
// different product and is not this key. Dual-platform probes on 2026-08-17
// show CLI sessions as ~/.cursor/chats/<32-hex>/<uuid-v4>/meta.json. The
// editor tree under projects/ stays excluded. store.db exists beside
// meta.json and is not parsed.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused.
func Cursor() agents.Descriptor {
	return agents.Descriptor{
		Key:         "cursor",
		DisplayName: "Cursor CLI",
		Vendor:      "Anysphere",
		DocsURL:     "https://cursor.com/docs/cli/overview",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".cursor")}}
			},
			Marker:      "chats",
			SessionGlob: cursorsrc.SessionGlob,
			Layout:      "chats-hex-uuid-meta-json",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded:    cursorsrc.Excluded,
		},
		Process: agents.ProcessSpec{
			// Official docs name the binary `agent`. That basename collides
			// with other tools, so the process image is the specific
			// `cursor-agent` name until a probe can tell them apart.
			Images: []string{"cursor-agent"},
		},
		NewIndexSource: cursorsrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/cursor.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-cursor.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-cursor.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/cursor/macos",
				"testdata/sessionindex/cursor/windows",
			},
		},
	}
}
