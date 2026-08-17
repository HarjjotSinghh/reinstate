package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestGrokExcludesInstallNoise(t *testing.T) {
	d := Grok()
	if d.Key != sessionindex.AgentGrok {
		t.Fatalf("Key = %q", d.Key)
	}
	if d.Tier != agents.TierHandoffFrom {
		t.Fatalf("Tier = %s, want T2", d.Tier)
	}
	for _, want := range []string{
		"bundled",
		"marketplace-cache",
		"bin",
		"downloads",
		"docs",
		"auth.json",
		"auth.json.lock",
		"subagents",
	} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %q", d.Storage.Excluded, want)
		}
	}
}
