package doctest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
)

// t012MapBacked are the shipped catalog keys whose storage pages still live in
// docs/session-storage-map.md. T-012 moves them; do not invent pages here.
var t012MapBacked = map[string]bool{
	"claude":   true,
	"codex":    true,
	"gemini":   true,
	"opencode": true,
	"grok":     true,
}

var shippedAgentDocs = []string{
	"README.md",
	"docs/README.md",
	"docs/adapters.md",
	"docs/agent-support-tiers.md",
	"docs/cli-reference.md",
	"docs/compatibility.md",
	"docs/faq.md",
	"docs/features.md",
	"docs/getting-started.md",
	"docs/handoff.md",
	"docs/troubleshooting.md",
}

var (
	tierToken       = regexp.MustCompile(`(?i)^T[0-5]$`)
	pipeKeyList     = regexp.MustCompile("`--(?:from|to|with|agent) ([^`]+)`")
	synopsisKeys    = regexp.MustCompile(`--(?:from|to|with|agent) ([a-z0-9_|]+)`)
	t0ReasonToken   = regexp.MustCompile("`([a-z0-9_]+)`")
	sentenceSplit   = regexp.MustCompile(`[.!?\n]+`)
	overClaimWord   = regexp.MustCompile(`(?i)\b(resume|sync|push|pull)\b|handoff to`)
	overClaimDeny   = regexp.MustCompile(`(?i)\b(no|not|never|without|refuse|refuses|refused|fail|fails|failed|read-only|source-only|unsupported|unverified|unavailable|later|planned|exploring)\b|mutation/sync|—`)
	overClaimAffirm = regexp.MustCompile(`(?i)\b(support|supports|supported|include|includes|included|can|allow|allows|provide|provides|enable|enables)\b`)
	affirmativeCell = regexp.MustCompile(`(?i)^(yes|included|supported|full|same-vendor|✅)`)
	negativeCell    = regexp.MustCompile(`(?i)^(no|—|-|n/?a|not\b|read-only|source-only|later|planned|exploring)`)
)

func TestCatalogAgentsHaveCompatibilityTiers(t *testing.T) {
	doc := read(t, "docs/compatibility.md")
	table, ok := tableWithHeader(doc, "Tier")
	if !ok {
		t.Fatal("docs/compatibility.md is missing a matrix with a Tier column")
	}
	claimed := claimedTiers(table)
	for _, desc := range agents.All() {
		got, ok := claimed[matchAgentRow(table, desc)]
		if !ok {
			t.Errorf("docs/compatibility.md has no row for catalog agent %s (%q)", desc.Key, desc.DisplayName)
			continue
		}
		if got != desc.Tier.String() {
			t.Errorf("docs/compatibility.md %s tier %s, catalog %s", desc.Key, got, desc.Tier)
		}
	}
	for name, tier := range claimed {
		if _, ok := catalogByRowName(name); !ok {
			t.Errorf("docs/compatibility.md claims %s at %s but that agent is not in the catalog", name, tier)
		}
	}
}

func TestCatalogEvidencePathsExist(t *testing.T) {
	root := repoRoot(t)
	storageMap := read(t, "docs/session-storage-map.md")
	for _, desc := range agents.All() {
		paths := append(append([]string{}, desc.Evidence.ProbeReports...), desc.Evidence.Fixtures...)
		paths = append(paths, desc.Evidence.DeviceReports...)
		if desc.Evidence.StoragePage != "" {
			paths = append(paths, desc.Evidence.StoragePage)
		}
		for _, rel := range paths {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				t.Errorf("%s evidence %s: %v", desc.Key, rel, err)
			}
		}
		if desc.Evidence.StoragePage != "" {
			continue
		}
		// T-012 skip: shipped five still live in the shared map. New catalog
		// agents must set Evidence.StoragePage to an existing file.
		if t012MapBacked[desc.Key] {
			if !strings.Contains(storageMap, desc.DisplayName) {
				t.Errorf("%s has no StoragePage and is missing from docs/session-storage-map.md", desc.Key)
			}
			continue
		}
		t.Errorf("%s has empty Evidence.StoragePage; add docs/session-storage/%s.md", desc.Key, desc.Key)
	}
}

func TestTierTableMatchesCatalog(t *testing.T) {
	doc := read(t, "docs/agent-support-tiers.md")
	table, ok := tableWithHeader(doc, "Current")
	if !ok {
		t.Fatal("docs/agent-support-tiers.md is missing the Current/target tier table")
	}
	// name -> Current token. Empty string means the row exists with "—".
	current := map[string]string{}
	for _, row := range table.rows {
		name := strings.TrimSpace(row[table.col["Agent"]])
		token := strings.Trim(strings.TrimSpace(row[table.col["Current"]]), "*")
		if name == "" {
			continue
		}
		if tierToken.MatchString(token) {
			current[name] = strings.ToUpper(token)
			continue
		}
		current[name] = ""
	}
	for _, desc := range agents.All() {
		got, ok := current[desc.DisplayName]
		if !ok {
			t.Errorf("docs/agent-support-tiers.md has no row for catalog agent %s (%q)", desc.Key, desc.DisplayName)
			continue
		}
		// Coordinator-owned Current "—" is not a claimed tier. Compare only T0–T5.
		if got == "" {
			continue
		}
		if got != desc.Tier.String() {
			t.Errorf("docs/agent-support-tiers.md %s current %s, catalog %s", desc.Key, got, desc.Tier)
		}
	}
	// A Current T0–T5 claim without a catalog descriptor is documentation drift.
	for name, tier := range current {
		if tier == "" {
			continue
		}
		if _, ok := catalogByDisplayName(name); !ok {
			t.Errorf("docs/agent-support-tiers.md claims %s at %s but that agent is not in the catalog", name, tier)
		}
	}
}

func TestT0ReasonsMatchDocs(t *testing.T) {
	for _, desc := range agents.All() {
		if desc.Tier != agents.TierKnown {
			continue
		}
		if desc.T0Reason == "" {
			t.Errorf("%s is T0 without T0Reason", desc.Key)
			continue
		}
		bodies := []string{read(t, "docs/agent-support-tiers.md")}
		if desc.Evidence.StoragePage != "" {
			bodies = append(bodies, read(t, desc.Evidence.StoragePage))
		} else {
			page := "docs/session-storage/" + desc.Key + ".md"
			if _, err := os.Stat(filepath.Join(repoRoot(t), page)); err == nil {
				bodies = append(bodies, read(t, page))
			}
		}
		if !t0ReasonDocumented(strings.Join(bodies, "\n"), desc) {
			t.Errorf("%s T0Reason %q is not documented next to %q", desc.Key, desc.T0Reason, desc.DisplayName)
		}
		_ = t0ReasonToken
	}
}

func TestCLIReferenceListsCatalogKeys(t *testing.T) {
	doc := read(t, "docs/cli-reference.md")
	// Each command lists the keys it accepts. T0 keys are not session filters
	// and are not required in this file.
	agent := commandKeySet(doc, "--agent")
	from := commandKeySet(doc, "--from")
	to := commandKeySet(doc, "--to")
	with := commandKeySet(doc, "--with")

	for _, desc := range agents.AtLeast(agents.TierDiscover) {
		if !agent[desc.Key] && !from[desc.Key] {
			t.Errorf("docs/cli-reference.md --agent/sessions filters omit T1+ key %q", desc.Key)
		}
	}
	for key := range agent {
		desc, ok := agents.Get(key)
		if !ok || desc.Tier < agents.TierDiscover {
			t.Errorf("docs/cli-reference.md --agent lists %q below T1", key)
		}
	}
	for _, desc := range agents.AtLeast(agents.TierHandoffFrom) {
		if !from[desc.Key] {
			t.Errorf("docs/cli-reference.md --from omits T2+ key %q", desc.Key)
		}
	}
	for _, desc := range agents.AtLeast(agents.TierHandoffTo) {
		if !to[desc.Key] {
			t.Errorf("docs/cli-reference.md --to omits T4+ key %q", desc.Key)
		}
	}
	for key := range to {
		desc, ok := agents.Get(key)
		if !ok || desc.Tier < agents.TierHandoffTo {
			t.Errorf("docs/cli-reference.md --to lists %q above that agent's tier", key)
		}
	}
	for key := range with {
		desc, ok := agents.Get(key)
		if !ok || desc.Tier < agents.TierResume {
			t.Errorf("docs/cli-reference.md --with lists %q above that agent's tier", key)
		}
	}
}

func TestDocsDoNotClaimAboveDeclaredTier(t *testing.T) {
	for _, path := range shippedAgentDocs {
		body := read(t, path)
		for _, desc := range agents.All() {
			if hits := overClaims(body, desc); len(hits) > 0 {
				t.Errorf("%s claims above %s %s: %s", path, desc.Key, desc.Tier, strings.Join(hits, "; "))
			}
		}
	}
}

func TestOverClaimScannerExamples(t *testing.T) {
	gemini, ok := agents.Get("gemini")
	if !ok {
		t.Fatal("gemini missing from catalog")
	}
	if hits := overClaims("Gemini CLI native resume is included.", gemini); len(hits) == 0 {
		t.Fatal("expected resume overclaim for Gemini")
	}
	if hits := overClaims("Gemini CLI resume fails with exit 5.", gemini); len(hits) != 0 {
		t.Fatalf("negated resume should pass, got %v", hits)
	}
	if hits := overClaims("| Gemini CLI | read-only | — |", gemini); len(hits) != 0 {
		t.Fatalf("table denial should pass, got %v", hits)
	}
	if hits := overClaims("- Mutation/sync for Gemini CLI and OpenCode in Phase 2", gemini); len(hits) != 0 {
		t.Fatalf("unsupported-list bullet should pass, got %v", hits)
	}
	pi, ok := agents.Get("pi")
	if !ok {
		t.Fatal("pi missing from catalog")
	}
	if hits := overClaims("Native vendor sync typically serves its own ecosystem", pi); len(hits) != 0 {
		t.Fatalf("Pi must not match typically, got %v", hits)
	}
}

type mdTable struct {
	headers []string
	col     map[string]int
	rows    [][]string
}

func tableWithHeader(doc, header string) (mdTable, bool) {
	for _, table := range parseMarkdownTables(doc) {
		if _, ok := table.col[header]; ok {
			if _, hasAgent := table.col["Agent"]; hasAgent {
				return table, true
			}
		}
	}
	return mdTable{}, false
}

func parseMarkdownTables(doc string) []mdTable {
	var tables []mdTable
	var lines []string
	flush := func() {
		if len(lines) < 2 {
			lines = nil
			return
		}
		headers := splitRow(lines[0])
		start := 1
		if isSeparator(lines[1]) {
			start = 2
		}
		col := map[string]int{}
		for i, header := range headers {
			col[header] = i
		}
		var rows [][]string
		for _, line := range lines[start:] {
			cells := splitRow(line)
			if len(cells) == 0 {
				continue
			}
			rows = append(rows, cells)
		}
		tables = append(tables, mdTable{headers: headers, col: col, rows: rows})
		lines = nil
	}
	for _, line := range strings.Split(doc, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "|") {
			lines = append(lines, trim)
			continue
		}
		flush()
	}
	flush()
	return tables
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, part := range parts {
		out[i] = strings.TrimSpace(part)
	}
	return out
}

func isSeparator(line string) bool {
	for _, cell := range splitRow(line) {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		for _, r := range cell {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
	}
	return true
}

func claimedTiers(table mdTable) map[string]string {
	out := map[string]string{}
	for _, row := range table.rows {
		if table.col["Agent"] >= len(row) || table.col["Tier"] >= len(row) {
			continue
		}
		name := strings.TrimSpace(row[table.col["Agent"]])
		token := strings.Trim(strings.TrimSpace(row[table.col["Tier"]]), "*")
		if name == "" || !tierToken.MatchString(token) {
			continue
		}
		out[name] = strings.ToUpper(token)
	}
	return out
}

func matchAgentRow(table mdTable, desc agents.Descriptor) string {
	for _, row := range table.rows {
		name := strings.TrimSpace(row[table.col["Agent"]])
		if agentNamesMatch(name, desc) {
			return name
		}
	}
	return ""
}

func catalogByRowName(name string) (agents.Descriptor, bool) {
	for _, desc := range agents.All() {
		if agentNamesMatch(name, desc) {
			return desc, true
		}
	}
	return agents.Descriptor{}, false
}

func catalogByDisplayName(name string) (agents.Descriptor, bool) {
	for _, desc := range agents.All() {
		if strings.EqualFold(name, desc.DisplayName) {
			return desc, true
		}
	}
	return agents.Descriptor{}, false
}

func agentNamesMatch(cell string, desc agents.Descriptor) bool {
	cell = stripMarkdown(cell)
	switch {
	case strings.EqualFold(cell, desc.DisplayName):
		return true
	case strings.EqualFold(cell, "OpenAI "+desc.DisplayName):
		return true
	case strings.EqualFold(cell, desc.Key):
		return true
	default:
		return false
	}
}

func stripMarkdown(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") {
		if end := strings.Index(value, "]"); end > 1 {
			return value[1:end]
		}
	}
	value = strings.Trim(value, "*")
	return strings.TrimSpace(value)
}

func commandKeySet(doc, flag string) map[string]bool {
	out := map[string]bool{}
	collect := func(raw string) {
		for _, part := range strings.Split(raw, "|") {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "`")
			if part == "" || part == "all" || part == "..." || strings.Contains(part, " ") {
				continue
			}
			if regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(part) {
				out[part] = true
			}
		}
	}
	for _, match := range pipeKeyList.FindAllStringSubmatch(doc, -1) {
		if strings.Contains(match[0], flag+" ") {
			collect(match[1])
		}
	}
	for _, match := range synopsisKeys.FindAllStringSubmatch(doc, -1) {
		if strings.Contains(match[0], flag+" ") {
			collect(match[1])
		}
	}
	return out
}

func t0ReasonDocumented(body string, desc agents.Descriptor) bool {
	token := string(desc.T0Reason)
	for _, sentence := range sentenceSplit.Split(body, -1) {
		if !strings.Contains(sentence, token) {
			continue
		}
		if agentMentioned(sentence, desc) {
			return true
		}
	}
	return strings.Contains(body, token)
}

func overClaims(body string, desc agents.Descriptor) []string {
	var hits []string
	if desc.Tier >= agents.TierSync {
		return nil
	}
	for _, table := range parseMarkdownTables(body) {
		hits = append(hits, tableOverClaims(table, desc)...)
	}
	for _, sentence := range sentenceSplit.Split(body, -1) {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" || strings.HasPrefix(sentence, "|") {
			continue
		}
		if !agentMentioned(sentence, desc) || !overClaimWord.MatchString(sentence) {
			continue
		}
		if overClaimDeny.MatchString(sentence) {
			continue
		}
		// Unsupported-list bullets name a withheld capability; they are not claims.
		if strings.HasPrefix(sentence, "-") && !overClaimAffirm.MatchString(sentence) {
			continue
		}
		if desc.Tier >= agents.TierResume && !strings.Contains(strings.ToLower(sentence), "handoff to") &&
			!regexp.MustCompile(`(?i)\b(sync|push|pull)\b`).MatchString(sentence) {
			continue
		}
		if desc.Tier >= agents.TierHandoffTo && !regexp.MustCompile(`(?i)\b(sync|push|pull)\b`).MatchString(sentence) {
			continue
		}
		hits = append(hits, compactSpace(sentence))
	}
	return hits
}

func tableOverClaims(table mdTable, desc agents.Descriptor) []string {
	var hits []string
	agentCol, hasAgent := table.col["Agent"]
	if !hasAgent {
		if idx, ok := table.col["Adapter"]; ok {
			agentCol = idx
			hasAgent = true
		}
	}
	if !hasAgent {
		return nil
	}
	for _, row := range table.rows {
		if agentCol >= len(row) || !agentNamesMatch(row[agentCol], desc) {
			continue
		}
		for i, header := range table.headers {
			if i == agentCol || i >= len(row) {
				continue
			}
			need := headerMinTier(header)
			if need == 0 || desc.Tier >= need {
				continue
			}
			cell := strings.TrimSpace(row[i])
			if cell == "" || negativeCell.MatchString(cell) {
				continue
			}
			if affirmativeCell.MatchString(cell) || overClaimWord.MatchString(cell) {
				hits = append(hits, header+": "+cell)
			}
		}
	}
	return hits
}

func headerMinTier(header string) agents.Tier {
	lower := strings.ToLower(header)
	switch {
	case strings.Contains(lower, "sync") || strings.Contains(lower, "push") || strings.Contains(lower, "pull"):
		return agents.TierSync
	case strings.Contains(lower, "handoff target") || strings.Contains(lower, "handoff to"):
		return agents.TierHandoffTo
	case strings.Contains(lower, "resume") || strings.Contains(lower, "fork"):
		return agents.TierResume
	default:
		return 0
	}
}

func agentMentioned(text string, desc agents.Descriptor) bool {
	name := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(desc.DisplayName) + `\b`)
	key := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(desc.Key) + `\b`)
	return name.MatchString(text) || key.MatchString(text)
}

func compactSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// unevidencedPhrases are the ways the tier doc says an agent's physical
// journeys do not exist yet. They are correct prose for an unevidenced claim
// and false prose for an evidenced one.
//
// The list is deliberately narrow. It must not match a legitimate caveat about
// a single outstanding row, only a claim that the journeys as a whole are
// missing.
var unevidencedPhrases = []string{
	"have not been recorded",
	"has not been recorded",
	"pending confirmation",
	"awaiting confirmation",
	"journeys are still outstanding",
	"code-complete claim",
}

// TestTierDocDoesNotCallAnEvidencedAgentPending pins the direction of drift
// that nothing else catches.
//
// Conformance already fails when a descriptor claims a tier it has no evidence
// for. The reverse — evidence lands, the tier moves, and a prose paragraph
// still tells the reader to treat it as unproven — is invisible to every gate
// we have, because the tier table and the descriptor both read correctly. It
// happened: Grok Build's paragraph called T4 "a code-complete claim whose
// physical journeys are still outstanding" for three commits after all four of
// its journey reports were merged and cited.
func TestTierDocDoesNotCallAnEvidencedAgentPending(t *testing.T) {
	t.Parallel()

	const rel = "docs/agent-support-tiers.md"
	evidenced := map[string]int{}
	for _, desc := range agents.All() {
		if name := strings.TrimSpace(desc.DisplayName); name != "" && len(desc.Evidence.DeviceReports) > 0 {
			evidenced[name] = len(desc.Evidence.DeviceReports)
		}
	}

	// Paragraph scope, and within a paragraph the subject is the agent named
	// first. Such a paragraph routinely cites other agents as the standard
	// being fallen short of ("not as evidenced the way Claude Code is"), and
	// blaming those turns a true failure into a misdirected one.
	for _, para := range strings.Split(read(t, rel), "\n\n") {
		if strings.HasPrefix(strings.TrimSpace(para), "|") {
			continue // a table row, not a claim about evidence
		}
		lower := strings.ToLower(para)
		hit := ""
		for _, phrase := range unevidencedPhrases {
			if strings.Contains(lower, phrase) {
				hit = phrase
				break
			}
		}
		if hit == "" {
			continue
		}
		subject, at := "", -1
		for name := range evidenced {
			if i := strings.Index(para, name); i >= 0 && (at < 0 || i < at) {
				subject, at = name, i
			}
		}
		if subject == "" {
			continue // the paragraph is about an agent with no evidence to contradict
		}
		t.Errorf(
			"%s calls %s unevidenced (%q) but its descriptor cites %d device report(s)",
			rel, subject, hit, evidenced[subject],
		)
	}
}
