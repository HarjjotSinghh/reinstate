package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestCopilotConformance(t *testing.T) {
	conformance.Run(t, Copilot(), conformance.Fixtures{})
}

func TestCopilotStaysT0LayoutUnverified(t *testing.T) {
	got := Copilot()
	if got.Key != "copilot" {
		t.Fatalf("key = %q", got.Key)
	}
	if got.Tier != agents.TierKnown || got.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("tier/reason = %s/%s", got.Tier, got.T0Reason)
	}
	if got.Evidence.StoragePage != "docs/session-storage/copilot.md" {
		t.Fatalf("StoragePage = %q", got.Evidence.StoragePage)
	}
	if got.NewIndexSource != nil || got.NewReader != nil || got.NewTarget != nil || got.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not grow a capability constructor")
	}
	if got.Native != nil || got.Version != nil {
		t.Fatal("T0 descriptor must not claim native resume or a version range")
	}
}
