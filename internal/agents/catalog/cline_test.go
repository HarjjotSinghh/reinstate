package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestClineConformance(t *testing.T) {
	conformance.Run(t, Cline(), conformance.Fixtures{})
}

func TestClineStaysT0WithoutDualProbes(t *testing.T) {
	d := Cline()
	if d.Key != "cline" || d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("cline identity = %s %s %s", d.Key, d.Tier, d.T0Reason)
	}
	if d.Family != agents.FamilyEmbeddedDB {
		t.Fatalf("family = %s, want F3", d.Family)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not expose constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 descriptor must not claim native resume or a version range")
	}
	if d.Storage.Roots != nil {
		t.Fatal("T0 cline must not declare unverified candidate roots")
	}
	if len(d.Evidence.ProbeReports) != 0 || len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 descriptor must not cite probes, fixtures, or device reports")
	}
}
