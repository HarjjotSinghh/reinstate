// Package doctest validates documentation claims against product truth.
package doctest

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/doctest -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// TestReleaseAndSupportClaims fails when docs invent released versions or
// blur local read capabilities into unsupported mutation/sync claims.
func TestReleaseAndSupportClaims(t *testing.T) {
	changelog := read(t, "CHANGELOG.md")
	if strings.Contains(changelog, "## [0.0.0]") {
		t.Error("CHANGELOG must not invent a published v0.0.0 release section")
	}
	if strings.Contains(changelog, "releases/tag/v0.0.0") {
		t.Error("CHANGELOG must not link to a nonexistent v0.0.0 tag")
	}

	citation := read(t, "CITATION.cff")
	// Citation metadata must name the version the public bootstrap actually
	// pins, so it can never advertise a release that has not shipped. This
	// replaces a hard-coded refusal of 0.1.0, which could only ever be correct
	// until 0.1.0 itself shipped.
	shipped := strings.TrimPrefix(publicBootstrapVersion, "v")
	citationVersion := regexp.MustCompile(`(?m)^version:\s*(\S+)\s*$`).FindStringSubmatch(citation)
	if citationVersion == nil {
		t.Error("CITATION.cff must declare a version")
	} else if citationVersion[1] != shipped {
		t.Errorf("CITATION.cff claims version %q but the public bootstrap pins %q",
			citationVersion[1], shipped)
	}

	roadmap := read(t, "ROADMAP.md")
	// Phase 1 in authority is Claude+Codex sessions; MCP/skills is post-Phase-1.
	if strings.Contains(roadmap, "Phase 0 — MVP (v0.1)") && strings.Contains(roadmap, "Claude Code adapter") {
		// Old confusing phase numbering should be gone.
		if !strings.Contains(roadmap, "Phase 0") || !strings.Contains(roadmap, "Phase 1") {
			t.Error("ROADMAP must define Phase 0 foundation and Phase 1 Claude/Codex sessions")
		}
	}

	// Gemini/OpenCode may advertise Phase 2 local read evidence, but neither may
	// be presented as a Phase 1 sync adapter or a mutation-capable native
	// executor. Keep the capability columns explicit instead of rejecting every
	// implementation checkmark next to the agent name.
	compatibility := read(t, "docs/compatibility.md")
	for _, expected := range []string{
		"| Gemini CLI | Not in Phase 1 |",
		"| OpenCode | Not in Phase 1 |",
	} {
		if !strings.Contains(compatibility, expected) {
			t.Errorf("docs/compatibility.md must retain stable Phase 1 exclusion %q", expected)
		}
	}

	// OpenCode moved up the ladder again — to T5, encrypted sync — so the guard
	// on its sync column inverts: it must now assert the sync claim rather than
	// refuse it. Gemini is still a source-only agent and keeps both trailing
	// columns pinned to "No".
	adapters := read(t, "docs/adapters.md")
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\| Gemini CLI \|[^\n]*\| No \| No \|`),
		regexp.MustCompile(`(?m)^\| OpenCode \|[^\n]*\| Supported[^|\n]*\| [^|\n]*\|$`),
	} {
		if !re.MatchString(adapters) {
			t.Errorf("docs/adapters.md must keep adapter capabilities explicit: %s", re.String())
		}
	}

	readme := read(t, "README.md")
	// Gemini must not claim native mutation; OpenCode's sync column must state
	// its evidenced scope rather than a bare em-dash.
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\| \[Gemini CLI\][^\n]*\| — \| — \|`),
		regexp.MustCompile(`(?m)^\| \[OpenCode\][^\n]*\| ✅ macOS \(Windows pending\) \|$`),
	} {
		if !re.MatchString(readme) {
			t.Errorf("README.md agent rows must state evidenced capabilities: %s", re.String())
		}
	}

	// Cursor ships at T1: its sessions are indexed and searchable while resume
	// and fork stay refused. A "read-only" index claim is therefore truthful,
	// and what must stay guarded is any capability above that tier. Pin the
	// rows rather than banning every checkmark next to the name.
	cursorClaims := []struct {
		doc  string
		body string
		re   *regexp.Regexp
	}{
		{"README.md", readme, regexp.MustCompile(
			`(?m)^\| \[Cursor CLI\][^\n]*\| T1 \| ✅ read-only \| — \| — \| — \| — \|`)},
		{"docs/adapters.md", adapters, regexp.MustCompile(
			`(?m)^\| Cursor CLI \| Read-only \(T1\) \| No \| No \| No \| No \|`)},
	}
	for _, claim := range cursorClaims {
		if !claim.re.MatchString(claim.body) {
			t.Errorf("%s must keep Cursor read-only at T1 with no resume, handoff or sync claim: %s",
				claim.doc, claim.re.String())
		}
	}
	if !strings.Contains(compatibility, "| Cursor | Not in Phase 1 |") {
		t.Error("docs/compatibility.md must retain the Cursor Phase 1 exclusion")
	}
	if regexp.MustCompile(`(?i)cursor.*✅`).MatchString(read(t, "SUPPORT.md")) {
		t.Error("SUPPORT.md claims implemented Cursor support")
	}

	// Required authority docs exist.
	for _, path := range []string{
		"docs/adr/0001-phase-0-phase-1-scope.md",
		"docs/compatibility.md",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot(t), path)); err != nil {
			t.Errorf("missing required doc %s: %v", path, err)
		}
	}
}

func TestNoFakeReleaseLinks(t *testing.T) {
	root := repoRoot(t)
	entries := []string{"CHANGELOG.md", "README.md", "RELEASING.md", "CITATION.cff"}
	fake := regexp.MustCompile(`v0\.0\.0`)
	for _, rel := range entries {
		body := read(t, rel)
		if fake.MatchString(body) {
			// Allow mention of "no v0.0.0" style prose, but not tag links/sections.
			if strings.Contains(body, "tag/v0.0.0") || strings.Contains(body, "## [0.0.0]") || strings.Contains(body, "[0.0.0]:") {
				t.Errorf("%s references nonexistent v0.0.0 release", rel)
			}
		}
		_ = root
	}
}
