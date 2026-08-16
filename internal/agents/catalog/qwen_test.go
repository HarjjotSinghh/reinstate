package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestQwenConformance(t *testing.T) {
	conformance.Run(t, Qwen(), conformance.Fixtures{})
}

func TestQwenIsIdentifiedT0(t *testing.T) {
	d := Qwen()
	if d.Key != "qwen" || d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("qwen identity = %s %s %s", d.Key, d.Tier, d.T0Reason)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 qwen must not expose constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 qwen must not claim native resume or a version range")
	}
}
