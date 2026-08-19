package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestQwenConformance(t *testing.T) {
	conformance.Run(t, Qwen(), conformance.Fixtures{
		Root: "testdata/sessionindex/qwen",
		OS:   []string{"macos", "windows"},
	})
}

func TestQwenIsDiscoverOnly(t *testing.T) {
	d := Qwen()
	if d.Tier != agents.TierDiscover {
		t.Fatalf("tier = %s, want T1", d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
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

func TestQwenCitesBothPlatformProbes(t *testing.T) {
	d := Qwen()
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

func TestQwenExcludesUpdaterTree(t *testing.T) {
	d := Qwen()
	if !contains(d.Storage.Excluded, "updates") {
		t.Fatalf("excluded = %v, missing updates (npm self-updater drowned the Windows probe)", d.Storage.Excluded)
	}
}
