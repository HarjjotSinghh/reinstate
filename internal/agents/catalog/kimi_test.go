package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestKimiConformance(t *testing.T) {
	conformance.Run(t, Kimi(), conformance.Fixtures{})
}

func TestKimiStaysT0WithoutDualProbes(t *testing.T) {
	d := Kimi()
	if d.Tier != agents.TierKnown {
		t.Fatalf("tier = %s, want T0", d.Tier)
	}
	if d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("T0Reason = %q, want %q", d.T0Reason, agents.T0LayoutUnverified)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not ship index, reader, target, or sync constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 descriptor must not claim native resume or a version range")
	}
	if len(d.Evidence.ProbeReports) != 0 || len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 descriptor must not cite probes, fixtures, or device reports")
	}
	if !contains(d.Storage.Excluded, "credentials") || !contains(d.Storage.Excluded, "agents/agent-*") {
		t.Fatalf("excluded = %v, want credentials and agents/agent-*", d.Storage.Excluded)
	}
}
