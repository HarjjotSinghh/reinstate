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
		"handoff":         {"### `rein handoff`", []string{"allow-active", "allow-untested", "allow-warning", "dry-run", "export", "from", "json", "last", "no-launch", "no-redact", "policy", "show-redactions", "to"}},
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
		"| **OpenCode** | structured handoff | structured handoff | not in v0.4.0 | not a target (source-only) |",
		"| **Grok Build** | structured handoff | structured handoff | not in v0.4.0 | not in v0.4.0 | not a target (source-only) |",
		"| **Kimi Code CLI** | structured handoff | structured handoff | not in v0.4.0 | not in v0.4.0 | not planned |",
	} {
		if !strings.Contains(doc, row) {
			t.Errorf("docs/compatibility.md is missing directional row %q", row)
		}
	}
}

func TestProductDocsDoNotClaimCrossAgentNativeIdentity(t *testing.T) {
	paths := []string{
		"README.md", "ROADMAP.md", "CHANGELOG.md", "docs/README.md",
		"docs/adapters.md", "docs/compatibility.md", "docs/cli-reference.md",
		"docs/comparison.md", "docs/cross-agent-continuation.md", "docs/faq.md",
		"docs/getting-started.md", "docs/handoff.md", "docs/security-model.md",
		"docs/troubleshooting.md", "docs/seo/product-truth-register.md",
	}
	for _, path := range paths {
		for number, paragraph := range markdownParagraphs(read(t, path)) {
			if claimsCrossAgentNativeIdentity(paragraph) {
				t.Errorf("%s paragraph %d makes an affirmative cross-agent identity claim: %s", path, number+1, paragraph)
			}
		}
	}
}

func TestCrossAgentIdentityClaimClassifier(t *testing.T) {
	tests := []struct {
		name      string
		paragraph string
		want      bool
	}{
		{"positive same session mutation", "A structured handoff continues the\nsame session in Codex.", true},
		{"positive native mutation", "Cross-agent handoff provides native resume.", true},
		{"positive reverse mutation", "Native resume is the cross-agent handoff mode.", true},
		{"positive mixed mutation", "Native resume stays same-vendor. A structured handoff preserves the same session.", true},
		{"positive repeated native mutation", "Native resume stays same-vendor. A structured handoff provides native resume.", true},
		{"positive after negation mutation", "One handoff is not native resume. Another cross-agent handoff provides native resume.", true},
		{"positive same-clause native mutation", "A handoff is not native resume, but it provides native resume.", true},
		{"positive same-clause session mutation", "A structured handoff does not preserve the same session, but it opens the same session.", true},
		{"negative direct", "A structured handoff is not native resume.", false},
		{"negative ordinary contraction", "A cross-agent handoff isn't native resume.", false},
		{"negative does not imply", "A handoff doesn't imply native resume.", false},
		{"negative same session", "A handoff creates a new session, not the same session.", false},
		{"negative cannot preserve", "A handoff cannot preserve the same session.", false},
		{"negative same vendor", "Native resume stays same-vendor; cross-agent work uses a structured handoff.", false},
		{"negative invalid claim", "Same exact session and lossless native resume are not valid cross-agent claims.", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paragraphs := markdownParagraphs(test.paragraph)
			if len(paragraphs) != 1 {
				t.Fatalf("markdownParagraphs() returned %d paragraphs, want 1", len(paragraphs))
			}
			if got := claimsCrossAgentNativeIdentity(paragraphs[0]); got != test.want {
				t.Fatalf("claimsCrossAgentNativeIdentity()=%t want %t for %q", got, test.want, test.paragraph)
			}
		})
	}
}

var markdownParagraphBreak = regexp.MustCompile(`\n[\t ]*\n+`)

func markdownParagraphs(doc string) []string {
	doc = strings.ReplaceAll(doc, "\r\n", "\n")
	blocks := markdownParagraphBreak.Split(doc, -1)
	paragraphs := make([]string, 0, len(blocks))
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		table := len(lines) > 0
		for _, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				table = false
				break
			}
		}
		paragraph := strings.Join(strings.Fields(block), " ")
		if table {
			for index, line := range lines {
				lines[index] = strings.Join(strings.Fields(line), " ")
			}
			paragraph = strings.Join(lines, "\n")
		}
		if paragraph != "" {
			paragraphs = append(paragraphs, paragraph)
		}
	}
	return paragraphs
}

var (
	crossAgentTerm = regexp.MustCompile(`(?i)(cross-agent|structured handoff|handoff)`)
	identityTerm   = regexp.MustCompile(`(?i)(native resume|same(?: exact)? session)`)
	nativeTruth    = regexp.MustCompile(`(?i)(IDENTITY_TERM/fork\s*\||same[- ]vendor\s+IDENTITY_TERM|not (?:a )?IDENTITY_TERM|isn't IDENTITY_TERM|does(?: not|n't)[^|]{0,80}IDENTITY_TERM|(?:never|cannot|can't|without|rather than)[^|]{0,80}IDENTITY_TERM|IDENTITY_TERM[^|]{0,80}(?:is|are) not(?: valid)?|IDENTITY_TERM[^|]{0,80}(?:same (?:agent|harness/vendor)|claude\s*→\s*claude(?: code)?|codex\s*→\s*codex)|IDENTITY_TERM[^|]{0,40}(?:stays|remains|is) same[- ]vendor)`)
	sessionTruth   = regexp.MustCompile(`(?i)(not (?:the )?IDENTITY_TERM|isn't the IDENTITY_TERM|does(?: not|n't)[^|]{0,80}IDENTITY_TERM|(?:never|cannot|can't|without|rather than)[^|]{0,80}IDENTITY_TERM|IDENTITY_TERM[^|]{0,80}(?:is|are) not valid|IDENTITY_TERM[^|]{0,40}(?:overclaim|claim))`)
	claimBreak     = regexp.MustCompile(`[.!?;]+(?:\s+|$)`)
)

func claimsCrossAgentNativeIdentity(paragraph string) bool {
	for _, row := range strings.Split(paragraph, "\n") {
		for _, clause := range claimBreak.Split(row, -1) {
			if !crossAgentTerm.MatchString(clause) {
				continue
			}
			matches := identityTerm.FindAllStringIndex(clause, -1)
			for index, match := range matches {
				identity := strings.ToLower(clause[match[0]:match[1]])
				context := identityOccurrenceContext(clause, matches, index, identity)
				if strings.Contains(identity, "native resume") {
					if !nativeTruth.MatchString(context) {
						return true
					}
					continue
				}
				if !sessionTruth.MatchString(context) {
					return true
				}
			}
		}
	}
	return false
}

func identityOccurrenceContext(clause string, matches [][]int, current int, identity string) string {
	start, end := 0, len(clause)
	for index := current - 1; index >= 0; index-- {
		if sameIdentityKind(clause[matches[index][0]:matches[index][1]], identity) {
			start = matches[index][1]
			break
		}
	}
	for index := current + 1; index < len(matches); index++ {
		if sameIdentityKind(clause[matches[index][0]:matches[index][1]], identity) {
			end = matches[index][0]
			break
		}
	}
	match := matches[current]
	return clause[start:match[0]] + "IDENTITY_TERM" + clause[match[1]:end]
}

func sameIdentityKind(left, right string) bool {
	return strings.Contains(strings.ToLower(left), "native resume") == strings.Contains(right, "native resume")
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
