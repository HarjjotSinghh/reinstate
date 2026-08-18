package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestZCodeConformance(t *testing.T) {
	conformance.Run(t, ZCode(), conformance.Fixtures{})
}

func TestZCodeIsDesktopOnlyT0(t *testing.T) {
	d := ZCode()
	if d.Key != "zcode" {
		t.Fatalf("key = %q", d.Key)
	}
	if d.Tier != agents.TierKnown || d.T0Reason != agents.T0DesktopOnly {
		t.Fatalf("tier/reason = %s/%s", d.Tier, d.T0Reason)
	}
	if d.Family != agents.FamilyRemote {
		t.Fatalf("family = %s", d.Family)
	}
	if d.Storage.Layout != "" || d.Storage.SessionGlob != "" || d.Storage.Roots != nil {
		t.Fatalf("T0 claimed an unofficial layout: %+v", d.Storage)
	}
	if d.Native != nil || d.Version != nil || d.NewIndexSource != nil || d.NewReader != nil {
		t.Fatalf("T0 claimed a capability above T0")
	}
}
