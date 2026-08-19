package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	copilotsrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/copilot"
)

func init() { agents.MustRegister(Copilot()) }

// Copilot is the GitHub Copilot CLI descriptor (T1, discover).
//
// Promoted on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both platforms
// write ~/.copilot/session-state/<uuid-v4>/events.jsonl with matching
// first-line keys. Windows also writes session-store.db and per-session
// session.db; those are not parsed. A rename-aside probe showed an old
// session ID did not return, so this is a local file tree.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused.
func Copilot() agents.Descriptor {
	return agents.Descriptor{
		Key:         "copilot",
		DisplayName: "GitHub Copilot CLI",
		Vendor:      "GitHub",
		DocsURL:     "https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-copilot-cli",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "COPILOT_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".copilot")}}
			},
			Marker:      "session-state",
			SessionGlob: copilotsrc.SessionGlob,
			Layout:      "session-state-uuid-events-jsonl",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded:    copilotsrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images:      []string{"copilot"},
			NodeMarkers: []string{"/@github/copilot/"},
		},
		NewIndexSource: copilotsrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/copilot.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-16-macos-copilot.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-copilot.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/copilot/macos",
				"testdata/sessionindex/copilot/windows",
			},
		},
	}
}
