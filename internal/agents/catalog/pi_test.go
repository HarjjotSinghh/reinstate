package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestPiConformance(t *testing.T) {
	conformance.Run(t, Pi(), conformance.Fixtures{})
}

func TestPiStaysT0WithoutCapabilities(t *testing.T) {
	d := Pi()
	if d.Key != "pi" || d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("pi identity = %s %s %s", d.Key, d.Tier, d.T0Reason)
	}
	if d.Family != agents.FamilyHomeTree {
		t.Fatalf("family = %s, want F1 (no CLI session list)", d.Family)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not expose constructors")
	}
	if d.Native != nil {
		t.Fatal("T0 descriptor must not claim native resume")
	}
	if d.Version == nil || d.Version.Min != "0.73.1" || d.Version.Max != "0.73.1" {
		t.Fatalf("Version = %+v, want 0.73.1–0.73.1", d.Version)
	}
}

func TestPiProcessPrefersVendorIdentity(t *testing.T) {
	d := Pi()
	if len(d.Process.Identify) != 2 {
		t.Fatalf("Identify = %#v", d.Process.Identify)
	}
	got := map[string]string{}
	for _, identity := range d.Process.Identify {
		got[identity.Name] = identity.Value
	}
	if got["PI_CODING_AGENT"] != "true" || got["AI_AGENT"] != "pi" {
		t.Fatalf("Identify = %#v", d.Process.Identify)
	}
}

func TestPiExcludesCredentialsAndExports(t *testing.T) {
	d := Pi()
	want := []string{"auth.json", "**/auth.json", "npm", "git", "**/*.html"}
	for _, name := range want {
		if !contains(d.Storage.Excluded, name) {
			t.Fatalf("excluded = %v, missing %s", d.Storage.Excluded, name)
		}
	}
}
