package doctest

import (
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/HarjjotSinghh/reinstate/internal/cli"
)

func TestPhase4CLIFlagsAreDocumented(t *testing.T) {
	root := cli.NewRoot(cli.Options{
		Name:            "rein",
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	doc := read(t, "docs/cli-reference.md")
	want := map[string]struct {
		heading string
		flags   []string
	}{
		"handoff":         {"### `rein handoff`", []string{"allow-active", "allow-untested", "allow-warning", "dry-run", "export", "from", "json", "last", "no-launch", "policy", "show-redactions", "to"}},
		"handoff list":    {"### `rein handoff list`", []string{"json", "limit"}},
		"handoff inspect": {"### `rein handoff inspect`", []string{"acknowledged", "json", "not-acknowledged"}},
		"handoff export":  {"### `rein handoff export`", []string{"format", "out"}},
		"resume":          {"### `rein resume --with` and `--fork`", []string{"allow-environment-warning", "dry-run", "fork", "json", "with"}},
	}

	for path, contract := range want {
		cmd := findCommand(t, root, path)
		section := markdownSection(t, doc, contract.heading)
		expected := make(map[string]bool, len(contract.flags))
		for _, name := range contract.flags {
			expected[name] = true
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("rein %s is missing shipped flag --%s", path, name)
			}
			flagPattern := regexp.MustCompile("`--" + regexp.QuoteMeta(name) + "(?:`|[ =])")
			if !flagPattern.MatchString(section) {
				t.Errorf("docs/cli-reference.md does not document rein %s --%s", path, name)
			}
		}
		cmd.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) {
			if flag.Name == "help" {
				return
			}
			if !expected[flag.Name] {
				t.Errorf("rein %s ships undocumented flag --%s", path, flag.Name)
			}
		})
	}
}

func TestPhase4DirectionalCompatibilityIsDocumented(t *testing.T) {
	doc := read(t, "docs/compatibility.md")
	for _, row := range []string{
		"| **Claude Code** | same-vendor native resume | structured handoff |",
		"| **Codex CLI** | structured handoff | same-vendor native resume |",
		"| **Gemini CLI** | structured handoff | structured handoff | not a target (source-only) |",
		"| **OpenCode** | structured handoff | structured handoff | not in rc.1 | not a target (source-only) |",
		"| **Grok Build** | structured handoff | structured handoff | not in rc.1 | not in rc.1 | not a target (source-only) |",
	} {
		if !strings.Contains(doc, row) {
			t.Errorf("docs/compatibility.md is missing directional row %q", row)
		}
	}
}

func TestProductDocsDoNotClaimCrossAgentNativeIdentity(t *testing.T) {
	paths := []string{
		"README.md", "docs/README.md", "docs/adapters.md", "docs/compatibility.md",
		"docs/cli-reference.md", "docs/faq.md", "docs/comparison.md",
		"docs/getting-started.md", "docs/security-model.md", "docs/handoff.md",
		"docs/troubleshooting.md", "docs/seo/product-truth-register.md",
	}
	crossAgent := regexp.MustCompile(`(?i)(cross-agent|structured handoff|handoff)`)
	identityClaim := regexp.MustCompile(`(?i)(native resume|same session)`)
	truthful := regexp.MustCompile(`(?i)(not native resume|never[^.]*native resume|does not[^.]*same session|not[^.]*same session|same-vendor native resume)`)
	for _, path := range paths {
		for number, line := range strings.Split(read(t, path), "\n") {
			if strings.HasPrefix(line, "| Agent |") || strings.HasPrefix(line, "| Adapter |") {
				continue
			}
			if crossAgent.MatchString(line) && identityClaim.MatchString(line) && !truthful.MatchString(line) {
				t.Errorf("%s:%d makes an affirmative cross-agent identity claim: %s", path, number+1, line)
			}
		}
	}
}

func findCommand(t *testing.T, root *cobra.Command, path string) *cobra.Command {
	t.Helper()
	cmd, _, err := root.Find(strings.Fields(path))
	if err != nil || cmd == nil || cmd.CommandPath() != "rein "+path {
		t.Fatalf("find rein %s: command=%v err=%v", path, cmd, err)
	}
	return cmd
}

func markdownSection(t *testing.T, doc, heading string) string {
	t.Helper()
	start := strings.Index(doc, heading)
	if start < 0 {
		t.Fatalf("docs/cli-reference.md is missing section %q", heading)
	}
	section := doc[start+len(heading):]
	if end := strings.Index(section, "\n### "); end >= 0 {
		section = section[:end]
	}
	return section
}
