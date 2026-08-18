package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestKimiConformance(t *testing.T) {
	conformance.Run(t, Kimi(), conformance.Fixtures{
		Root: "testdata/sessionindex/kimi",
		OS:   []string{"macos", "windows"},
	})
}

// Kimi is T1 and no further. It reached T1 on 2026-08-17 when a native Windows
// probe joined the macOS one; everything above T1 needs evidence that does not
// exist, and a device journey running `kimi -r` most of all.
func TestKimiIsDiscoverOnly(t *testing.T) {
	d := Kimi()
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
		t.Fatal("native resume and a version range are T3 claims; no device journey has run kimi -r")
	}
}

// The tier rests on two artifacts. Losing either one silently would leave a
// promoted agent standing on single-platform evidence, which is the exact
// failure probePlatformGap exists to prevent.
func TestKimiCitesBothPlatformProbes(t *testing.T) {
	d := Kimi()
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

// Subagent trees are excluded because a Kimi session nests agents/agent-0
// through agent-7, each with its own wire log, and one Windows session carried
// a tasks/bash-<id>/output.log below that. Credentials are excluded because a
// descriptor is one contract across platforms.
func TestKimiExcludesSubagentsAndCredentials(t *testing.T) {
	d := Kimi()
	for _, want := range []string{"agents", "subagents", "credentials", "mcp-oauth"} {
		if !contains(d.Storage.Excluded, want) {
			t.Errorf("excluded = %v, missing %q", d.Storage.Excluded, want)
		}
	}
}
