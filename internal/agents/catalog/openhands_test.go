package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestOpenHandsConformance(t *testing.T) {
	conformance.Run(t, OpenHands(), conformance.Fixtures{})
}

func TestOpenHandsIsT0ServerBacked(t *testing.T) {
	got, ok := agents.Get("openhands")
	if !ok {
		t.Fatal("openhands missing from catalog")
	}
	if got.Tier != agents.TierKnown || got.T0Reason != agents.T0ServerBacked || got.Family != agents.FamilyRemote {
		t.Fatalf("openhands = %s/%s/%s, want T0/server_backed/F5", got.Tier, got.T0Reason, got.Family)
	}
	if got.NewIndexSource != nil || got.NewReader != nil || got.NewTarget != nil || got.NewSyncAdapter != nil {
		t.Fatal("openhands must not expose constructors above T0")
	}
	if got.Native != nil || got.Version != nil {
		t.Fatal("openhands must not claim native resume or a version range")
	}
	if got.Storage.RootEnv != "OH_PERSISTENCE_DIR" {
		t.Fatalf("root env = %q", got.Storage.RootEnv)
	}
}
