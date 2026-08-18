package conformance

import (
	"strings"
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
			var evidence error
			for _, check := range Evaluate(tc.desc, tc.fixtures) {
				if check.Err == nil {
					continue
				}
				if check.Name == "evidence" {
					evidence = check.Err
					continue
				}
				t.Errorf("%s: %v", check.Name, check.Err)
			}
			if evidence != nil {
				t.Logf("escalation %s evidence: %v", tc.name, evidence)
			}
		})
	}
}

func TestShippedEvidenceGapsAreExact(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, desc := range []agents.Descriptor{catalog.Claude(), catalog.Codex(), catalog.Gemini(), catalog.OpenCode(), catalog.Grok()} {
		err := checkEvidence(desc, root)
		if err == nil {
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, "StoragePage") && !strings.Contains(msg, "ProbeReports") {
			t.Fatalf("%s unexpected evidence error: %v", desc.Key, err)
		}
	}
}
