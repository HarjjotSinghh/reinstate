package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	kimisrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/kimi"
)

func init() { agents.MustRegister(Kimi()) }

// Kimi is the Kimi Code CLI descriptor (T1, discover).
//
// Promoted on 2026-08-17 by a native Windows probe that joined the macOS one
// from the day before. The Windows device carried five sessions across three
// projects, which is what settled the two questions a single-session macOS run
// could not: state.json carries an identical thirteen-key shape on both
// platforms, and session_index.jsonl enumerated exactly the five sessions
// present on disk.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused,
// because no device journey has run kimi -r against a real session.
func Kimi() agents.Descriptor {
	return agents.Descriptor{
		Key:         "kimi",
		DisplayName: "Kimi Code CLI",
		Vendor:      "Moonshot AI",
		DocsURL:     "https://www.kimi.com/code/docs/en/kimi-code-cli/guides/sessions.html",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "KIMI_CODE_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				// ~/.kimi-code is the live root on both probed platforms.
				// ~/.kimi is the conflicting mirror's claim; the Windows
				// device has it, without the marker, alongside a
				// migration-report.json in the live root. It is a legacy
				// location, kept as a candidate so a machine that never
				// migrated still resolves.
				return []agents.Root{
					{Path: home.Join(".kimi-code")},
					{Path: home.Join(".kimi")},
				}
			},
			Marker:      "sessions",
			SessionGlob: kimisrc.SessionGlob,
			Layout:      "sessions-workspace-session-state-json",
			ProjectKey:  agents.ProjectKeyPathHash,
			Excluded:    kimisrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images: []string{"kimi"},
		},
		NewIndexSource: kimisrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/kimi.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-16-macos-kimi.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-kimi.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/kimi/macos",
				"testdata/sessionindex/kimi/windows",
			},
		},
	}
}
