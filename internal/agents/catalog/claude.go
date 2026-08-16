package catalog

import (
	"regexp"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	claudesrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/claude"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Claude()) }

var claudeVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)) \(Claude Code\)$`)

// Claude is the shipped Claude Code descriptor (T5).
func Claude() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentClaude,
		DisplayName: "Claude Code",
		Vendor:      "Anthropic",
		DocsURL:     "https://docs.anthropic.com/en/docs/claude-code",
		Tier:        agents.TierSync,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "CLAUDE_CONFIG_DIR",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{
					{Path: home.Join(".claude")},
					{Path: home.Join(".config", "claude")},
				}
			},
			Marker:      "projects",
			Layout:      "projects-jsonl",
			SessionGlob: "projects/**/*.jsonl",
			ProjectKey:  agents.ProjectKeyPathSlug,
			Excluded:    claudesrc.Excluded,
		},
		Native: &agents.NativeSpec{
			Executable:    "claude",
			Resume:        []string{"--resume", "{{.SessionID}}"},
			Fork:          []string{"--resume", "{{.SessionID}}", "--fork-session"},
			NewSession:    []string{"--session-id", "{{.SessionID}}"},
			InitialPrompt: agents.PromptArgv,
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parseClaudeVersion,
			Min:   "2.1.219",
			Max:   "2.1.229",
		},
		Process: agents.ProcessSpec{
			Images:      []string{"claude"},
			NodeMarkers: []string{"/@anthropic-ai/claude-code/", "/claude-code/cli.js"},
		},
		Evidence: agents.Evidence{
			Fixtures: []string{
				"testdata/sessionindex/claude/macos",
				"testdata/sessionindex/claude/windows",
				"testdata/sessionindex/claude/wsl",
				"testdata/adapters/claude/macos",
				"testdata/adapters/claude/windows",
				"testdata/adapters/claude/wsl",
				"testdata/handoff/claude",
			},
			DeviceReports: []string{
				"docs/testing/results/2026-08-11-macos-phase3-V030.md",
				"docs/testing/results/2026-08-11-windows-phase3-V030.md",
				"docs/testing/results/2026-08-15-macos-phase4-V040RC11.md",
				"docs/testing/results/2026-08-15-windows-phase4-V040RC11.md",
			},
		},
		NewIndexSource: claudesrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return &transcript.ClaudeReader{}, nil
		},
		NewTarget: func(env agents.Env) (handoff.HandoffTarget, error) {
			return &handoff.ClaudeTarget{ConfigDir: env.FixtureRoot}, nil
		},
		NewSyncAdapter: func(env agents.Env) (adapter.Adapter, error) {
			return &claude.Adapter{Root: env.FixtureRoot, Home: env.Home}, nil
		},
	}
}

func parseClaudeVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, claudeVersionPattern)
}

func parseVersionLine(output agents.VersionOutput, pattern *regexp.Regexp) (string, bool) {
	if output.Stderr != "" {
		return "", false
	}
	line, ok := oneVersionLine(output.Stdout)
	if !ok {
		return "", false
	}
	matches := pattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

func oneVersionLine(output string) (string, bool) {
	if strings.HasSuffix(output, "\r\n") {
		output = strings.TrimSuffix(output, "\r\n")
	} else if strings.HasSuffix(output, "\n") {
		output = strings.TrimSuffix(output, "\n")
	}
	if output == "" || strings.ContainsAny(output, "\r\n") {
		return "", false
	}
	for _, character := range output {
		if character < 0x20 || character == 0x7f {
			return "", false
		}
	}
	return output, true
}
