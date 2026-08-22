package catalog

import (
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	qwensrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/qwen"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Qwen()) }

// Qwen is the Qwen Code descriptor (T2, handoff source).
//
// Promoted to T1 on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both
// platforms write JSONL conversations under
// ~/.qwen/projects/<sanitized-cwd>/chats/<uuid-v4>.jsonl with matching
// first-line keys. Runtime status sidecars are JSON, not JSONL, and never
// match the session glob.
//
// T2 since 2026-08-22: transcript.QwenReader turns those conversations into
// capsule events, so `rein handoff --from qwen` works. Native resume and fork
// stay refused — that is T3 and needs a dual-platform device journey.
//
// The Claude reader is still not reusable here. Qwen's top-level record keys
// match Claude Code's, but the body is a Gemini Content value
// ({"role":…,"parts":[…]}), and Qwen encodes /rewind by re-rooting the
// parentUuid chain rather than by writing a marker.
func Qwen() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentQwen,
		DisplayName: "Qwen Code",
		Vendor:      "Alibaba",
		DocsURL:     "https://qwenlm.github.io/qwen-code-docs/",
		Tier:        agents.TierHandoffFrom,
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
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/qwen.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-qwen.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-qwen.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/qwen/macos",
				"testdata/sessionindex/qwen/windows",
				"testdata/handoff/qwen",
			},
		},
		NewIndexSource: qwensrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return transcript.NewQwenReader(), nil
		},
	}
}
