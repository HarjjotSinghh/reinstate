package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestPiConformance(t *testing.T) {
	conformance.Run(t, Pi(), conformance.Fixtures{
		Root: "testdata/sessionindex/pi",
		OS:   []string{"macos", "windows"},
	})
}

func TestPiIsDiscoverOnly(t *testing.T) {
	d := Pi()
	if d.Key != "pi" || d.Tier != agents.TierDiscover {
		t.Fatalf("pi identity = %s %s", d.Key, d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.Family != agents.FamilyHomeTree {
		t.Fatalf("family = %s, want F1", d.Family)
	}
	if d.NewIndexSource == nil {
		t.Fatal("T1 requires an index source")
	}
	if d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T1 descriptor must not ship reader, target, or sync constructors")
	}
	if d.Native != nil {
		t.Fatal("T1 descriptor must not claim native resume")
	}
	if d.Version == nil || d.Version.Min != "0.73.1" || d.Version.Max != "0.73.1" {
		t.Fatalf("Version = %+v, want 0.73.1–0.73.1", d.Version)
	}
}

func TestPiCitesBothPlatformProbes(t *testing.T) {
	d := Pi()
	var macOS, windows bool
	for _, report := range d.Evidence.ProbeReports {
		macOS = macOS || strings.Contains(report, "-macos-")
		windows = windows || strings.Contains(report, "-windows-")
	}
	if !macOS || !windows {
		t.Fatalf("probe reports = %v, want one macOS and one native Windows", d.Evidence.ProbeReports)
	}
	if len(d.Evidence.Fixtures) != 2 {
		t.Fatalf("fixtures = %v, want one per platform", d.Evidence.Fixtures)
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
