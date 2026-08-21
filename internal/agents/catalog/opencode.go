package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	opencodesrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/opencode"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(OpenCode()) }

// OpenCode is the shipped OpenCode descriptor (T2, F3).
func OpenCode() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentOpenCode,
		DisplayName: "OpenCode",
		Vendor:      "anomalyco",
		DocsURL:     "https://opencode.ai",
		Tier:        agents.TierHandoffFrom,
		Family:      agents.FamilyEmbeddedDB,
		Storage: agents.StorageSpec{
			// OpenCode reads $XDG_DATA_HOME/opencode, so the variable names the
			// parent and the remaining segment is appended.
			RootEnv:       "XDG_DATA_HOME",
			RootEnvSuffix: "opencode",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".local", "share", "opencode")}}
			},
			Marker:     opencodesrc.DatabaseName,
			Layout:     "embedded-sqlite-session-store",
			ProjectKey: agents.ProjectKeyOpaqueID,
			Excluded:   opencodesrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images: []string{"opencode"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/opencode.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-21-macos-opencode.json",
				"docs/testing/results/agent-probes/2026-08-21-windows-opencode.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/opencode/macos",
				"testdata/sessionindex/opencode/windows",
				"testdata/handoff/opencode",
			},
		},
		NewIndexSource: opencodesrc.NewSQLite,
		NewReader: func(env agents.Env) (transcript.Reader, error) {
			reader := transcript.NewOpenCodeReader(nil)
			reader.DataRoot = env.FixtureRoot
			reader.Getenv = env.LookupEnv
			return reader, nil
		},
	}
}
