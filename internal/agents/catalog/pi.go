package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	pisrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/pi"
)

func init() { agents.MustRegister(Pi()) }

// Latest stable @mariozechner/pi-coding-agent as of 2026-08-16.
var piVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

// Pi is the Pi coding agent descriptor (T1, discover).
//
// Promoted on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both platforms
// write JSONL under ~/.pi/agent/sessions/<slug>/<slug>-<uuid-v4>.jsonl with
// matching first-line keys. The fail-closed version pin stays 0.73.1; that
// is identity, not a T3 resume claim.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused.
func Pi() agents.Descriptor {
	return agents.Descriptor{
		Key:         "pi",
		DisplayName: "Pi",
		Vendor:      "earendil-works",
		DocsURL:     "https://pi.dev/",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "PI_CODING_AGENT_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".pi", "agent")}}
			},
			Marker:      "sessions",
			Layout:      "sessions-cwd-slug-jsonl",
			SessionGlob: pisrc.SessionGlob,
			ProjectKey:  agents.ProjectKeyPathSlug,
			Excluded:    pisrc.Excluded,
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parsePiVersion,
			Min:   "0.73.1",
			Max:   "0.73.1",
		},
		Process: agents.ProcessSpec{
			Images: []string{"pi"},
			Identify: []agents.EnvIdentity{
				{Name: "PI_CODING_AGENT", Value: "true"},
				{Name: "AI_AGENT", Value: "pi"},
			},
		},
		NewIndexSource: pisrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/pi.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-pi.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-pi.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/pi/macos",
				"testdata/sessionindex/pi/windows",
			},
		},
	}
}

func parsePiVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, piVersionPattern)
}
