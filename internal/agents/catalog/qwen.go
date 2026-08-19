package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	qwensrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/qwen"
)

func init() { agents.MustRegister(Qwen()) }

// Qwen is the Qwen Code descriptor (T1, discover).
//
// Promoted on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both platforms
// write JSONL conversations under ~/.qwen/projects/<slug>/chats/<uuid-v4>.jsonl
// with matching first-line keys. macOS also writes <uuid-v4>-runtime.json
// sidecars; those are not conversations and are not indexed.
//
// T1 only. Sessions are indexed and searchable; resume and fork stay refused.
// Do not reuse the Claude reader: matching keys are not the same format.
func Qwen() agents.Descriptor {
	return agents.Descriptor{
		Key:         "qwen",
		DisplayName: "Qwen Code",
		Vendor:      "Alibaba",
		DocsURL:     "https://qwenlm.github.io/qwen-code-docs/",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "QWEN_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".qwen")}}
			},
			Marker:      "projects",
			SessionGlob: qwensrc.SessionGlob,
			Layout:      "projects-slug-chats-jsonl",
			ProjectKey:  agents.ProjectKeyPathSlug,
			Excluded:    qwensrc.Excluded,
		},
		Process: agents.ProcessSpec{
			Images:      []string{"qwen"},
			NodeMarkers: []string{"@qwen-code/qwen-code"},
			Identify: []agents.EnvIdentity{
				{Name: "QWEN_CODE", Value: "1"},
			},
		},
		NewIndexSource: qwensrc.New,
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/qwen.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-qwen.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-qwen.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/qwen/macos",
				"testdata/sessionindex/qwen/windows",
			},
		},
	}
}
