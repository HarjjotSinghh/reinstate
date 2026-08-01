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

	adapters := read(t, "docs/adapters.md")
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\| Gemini CLI \|[^\n]*\| No \| No \|`),
		regexp.MustCompile(`(?m)^\| OpenCode \|[^\n]*\| No \| No \|`),
	} {
		if !re.MatchString(adapters) {
			t.Errorf("docs/adapters.md must keep read-only adapter capabilities explicit: %s", re.String())
		}
	}

	readme := read(t, "README.md")
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\| \[Gemini CLI\][^\n]*\| — \| — \|`),
		regexp.MustCompile(`(?m)^\| \[OpenCode\][^\n]*\| — \| — \|`),
	} {
		if !re.MatchString(readme) {
			t.Errorf("README.md must not claim Gemini/OpenCode native mutation or encrypted sync: %s", re.String())
		}
	}

	for _, path := range []string{"docs/compatibility.md", "docs/adapters.md", "README.md", "SUPPORT.md"} {
		if regexp.MustCompile(`(?i)cursor.*✅`).MatchString(read(t, path)) {
			t.Errorf("%s claims implemented Cursor support", path)
		}
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
