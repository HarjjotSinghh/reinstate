package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestCursorConformance(t *testing.T) {
	conformance.Run(t, Cursor(), conformance.Fixtures{})
}

func TestCursorStaysT0LayoutUnverified(t *testing.T) {
	d := Cursor()
	if d.Key != "cursor" {
		t.Fatalf("key = %q", d.Key)
	}
	if d.DisplayName != "Cursor CLI" {
		t.Fatalf("DisplayName = %q", d.DisplayName)
	}
	if d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("tier/reason = %s/%s", d.Tier, d.T0Reason)
	}
	if d.Family != agents.FamilyEmbeddedDB {
		t.Fatalf("family = %s, want F3 until ls is proven machine-readable", d.Family)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not grow a capability constructor")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 descriptor must not claim native resume or a version range")
	}
	if d.Storage.Roots != nil || d.Storage.Layout != "" || d.Storage.SessionGlob != "" {
		t.Fatalf("T0 must not claim an unverified layout: %+v", d.Storage)
	}
	if len(d.Evidence.ProbeReports) != 0 || len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 descriptor must not cite probes, fixtures, or device reports")
	}
	if d.Evidence.StoragePage != "docs/session-storage/cursor.md" {
		t.Fatalf("StoragePage = %q", d.Evidence.StoragePage)
	}
	if len(d.Process.Images) != 1 || d.Process.Images[0] != "cursor-agent" {
		t.Fatalf("Images = %v, want only cursor-agent", d.Process.Images)
	}
}
