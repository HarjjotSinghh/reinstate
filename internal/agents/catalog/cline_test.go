package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestClineConformance(t *testing.T) {
	conformance.Run(t, Cline(), conformance.Fixtures{
		Root: "testdata/sessionindex/cline",
		OS:   []string{"macos", "windows"},
	})
}

func TestClineIsDiscoverOnly(t *testing.T) {
	d := Cline()
	if d.Key != "cline" || d.Tier != agents.TierDiscover {
		t.Fatalf("cline identity = %s %s", d.Key, d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.Family != agents.FamilyHomeTree {
		t.Fatalf("family = %s, want F1 (session JSON via hometree)", d.Family)
	}
	if d.NewIndexSource == nil {
		t.Fatal("T1 requires an index source")
	}
	if d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T1 descriptor must not ship reader, target, or sync constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("native resume and a version range are T3 claims")
	}
}

func TestClineCitesBothPlatformProbes(t *testing.T) {
	d := Cline()
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

func TestClineExcludesProviders(t *testing.T) {
	d := Cline()
	for _, want := range []string{"settings/providers.json", "**/providers.json"} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %s", d.Storage.Excluded, want)
		}
	}
}
