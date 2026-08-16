package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

func TestMiniMaxCodeDescriptorIsT0WithoutReader(t *testing.T) {
	got := MiniMaxCode()
	if got.Key != "minimax-code" {
		t.Fatalf("Key = %q, want minimax-code", got.Key)
	}
	if got.DisplayName != "MiniMax" {
		t.Fatalf("DisplayName = %q", got.DisplayName)
	}
	if got.Tier != agents.TierKnown || got.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("tier/reason = %s %q", got.Tier, got.T0Reason)
	}
	if got.NewIndexSource != nil || got.NewReader != nil || got.NewTarget != nil || got.NewSyncAdapter != nil {
		t.Fatal("T0 must not expose constructors")
	}
	if got.Evidence.StoragePage != "docs/session-storage/minimax.md" {
		t.Fatalf("StoragePage = %q", got.Evidence.StoragePage)
	}
}
