package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	clinesrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/cline"
)

func init() { agents.MustRegister(Cline()) }

// Cline is the official Cline product descriptor (T1, discover).
//
// Promoted on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both platforms
// write ~/.cline/data/sessions/<slug>/<slug>.json after cline 3.0.55.
// db/sessions.db and *.messages.json are not parsed. cline history --json
// listed the session on both platforms; that is an F2 candidate, not a
// shipped read API.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused.
func Cline() agents.Descriptor {
	return agents.Descriptor{
		Key:         "cline",
		DisplayName: "Cline",
		Vendor:      "Cline",
		DocsURL:     "https://docs.cline.bot/",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "CLINE_DATA_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".cline", "data")}}
			},
			Marker:      "sessions",
			SessionGlob: clinesrc.SessionGlob,
			Layout:      "sessions-id-json-plus-sqlite-index",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded:    clinesrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images:      []string{"cline"},
			NodeMarkers: []string{"saoudrizwan.claude-dev"},
		},
		NewIndexSource: clinesrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/cline.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-19-macos-cline.json",
				"docs/testing/results/agent-probes/2026-08-19-windows-cline.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/cline/macos",
				"testdata/sessionindex/cline/windows",
			},
		},
	}
}
