package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/codex"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	codexsrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/codex"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Codex()) }

var codexVersionPattern = regexp.MustCompile(`^codex-cli ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

// Codex is the shipped Codex CLI descriptor (T5).
func Codex() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentCodex,
		DisplayName: "Codex CLI",
		Vendor:      "OpenAI",
		DocsURL:     "https://github.com/openai/codex",
		Tier:        agents.TierSync,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "CODEX_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{
					{Path: home.Join(".codex")},
					{Path: home.Join(".config", "codex")},
				}
			},
			Marker:      "sessions",
			Layout:      "sessions-rollout-jsonl",
			SessionGlob: "sessions/**/*.jsonl",
			ProjectKey:  agents.ProjectKeyNone,
			Excluded:    codexsrc.Excluded,
		},
		Native: &agents.NativeSpec{
			Executable:    "codex",
			Resume:        []string{"resume", "{{.SessionID}}"},
			Fork:          []string{"fork", "{{.SessionID}}"},
			InitialPrompt: agents.PromptArgv,
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parseCodexVersion,
			Min:   "0.133.0",
			Max:   "0.149.0",
		},
		Process: agents.ProcessSpec{
			Images:      []string{"codex"},
			NodeMarkers: []string{"/@openai/codex/"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/codex.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-21-macos-codex.json",
				"docs/testing/results/agent-probes/2026-08-21-windows-codex.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/codex/forks",
				"testdata/adapters/codex/macos",
				"testdata/adapters/codex/windows",
				"testdata/adapters/codex/wsl",
				"testdata/handoff/codex",
			},
			DeviceReports: []string{
				"docs/testing/results/2026-08-11-macos-phase3-V030.md",
				"docs/testing/results/2026-08-11-windows-phase3-V030.md",
				"docs/testing/results/2026-08-15-macos-phase4-V040RC11.md",
				"docs/testing/results/2026-08-15-windows-phase4-V040RC11.md",
			},
		},
		NewIndexSource: codexsrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return &transcript.CodexReader{}, nil
		},
		NewTarget: func(env agents.Env) (handoff.HandoffTarget, error) {
			return handoff.NewCodexTarget(&handoff.CodexTarget{Root: env.FixtureRoot}), nil
		},
		NewSyncAdapter: func(env agents.Env) (adapter.Adapter, error) {
			return &codex.Adapter{Root: env.FixtureRoot, Home: env.Home}, nil
		},
	}
}

func parseCodexVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, codexVersionPattern)
}
