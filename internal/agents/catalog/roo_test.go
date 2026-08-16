package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestRooConformance(t *testing.T) {
	conformance.Run(t, Roo(), conformance.Fixtures{})
}

func TestRooStaysT0WithoutDualProbes(t *testing.T) {
	d := Roo()
	if d.Key != "roo" || d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("roo identity = %s %s %s", d.Key, d.Tier, d.T0Reason)
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
		t.Fatal("T0 roo must not declare unverified candidate roots")
	}
	if len(d.Evidence.ProbeReports) != 0 || len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 descriptor must not cite probes, fixtures, or device reports")
	}
}
