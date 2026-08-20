package conformance

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
)

func TestShippedAgentsConformance(t *testing.T) {
	cases := []struct {
		name     string
		desc     agents.Descriptor
		fixtures Fixtures
	}{
		{"claude", catalog.Claude(), Fixtures{Root: "testdata/sessionindex/claude", OS: []string{"macos", "windows"}}},
		{"codex", catalog.Codex(), Fixtures{Root: "testdata/sessionindex/codex/forks"}},
		{"gemini", catalog.Gemini(), Fixtures{Root: "testdata/sessionindex/gemini", OS: []string{"macos", "windows"}}},
		{"opencode", catalog.OpenCode(), Fixtures{Root: "testdata/sessionindex/opencode", OS: []string{"macos", "windows"}}},
		{"grok", catalog.Grok(), Fixtures{Root: "testdata/sessionindex/grok", OS: []string{"macos", "windows"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Every check fails the suite, evidence included. It used to be
			// collected and logged, so a descriptor could name a storage page,
			// probe report, or fixture that did not exist and nothing caught it.
			for _, check := range Evaluate(tc.desc, tc.fixtures) {
				if check.Err != nil {
					t.Errorf("%s: %v", check.Name, check.Err)
				}
			}
		})
	}
}

// TestShippedEvidenceIsComplete requires every shipped descriptor to name
// evidence that actually exists. The previous form allowed any error whose
// text mentioned StoragePage or ProbeReports, which is what every descriptor
// reported, so a missing fixture path rode along undetected.
func TestShippedEvidenceIsComplete(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, desc := range []agents.Descriptor{
		catalog.Claude(), catalog.Codex(), catalog.Gemini(), catalog.OpenCode(), catalog.Grok(),
	} {
		if err := checkEvidence(desc, root); err != nil {
			t.Errorf("%s: %v", desc.Key, err)
		}
	}
}

// TestBrokenEvidencePathIsCaught is the negative control for Matrix A9.
func TestBrokenEvidencePathIsCaught(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	desc := catalog.Claude()
	desc.Evidence.Fixtures = append(
		append([]string(nil), desc.Evidence.Fixtures...),
		"testdata/sessionindex/claude/this-path-does-not-exist",
	)
	if err := checkEvidence(desc, root); err == nil {
		t.Fatal("a descriptor naming a nonexistent evidence path passed checkEvidence")
	}
}
