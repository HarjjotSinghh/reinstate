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

// Kimi is T2: a handoff source. Resume stays refused until a device journey
// runs `kimi -r` and a fail-closed version range exists.
func TestKimiIsHandoffFrom(t *testing.T) {
	d := Kimi()
	if d.Tier != agents.TierHandoffFrom {
		t.Fatalf("tier = %s, want T2", d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.NewIndexSource == nil || d.NewReader == nil {
		t.Fatal("T2 requires an index source and a transcript reader")
	}
	if d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T2 descriptor must not ship target or sync constructors")
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
	var fixtureMacOS, fixtureWindows, handoff bool
	for _, fixture := range d.Evidence.Fixtures {
		fixtureMacOS = fixtureMacOS || strings.Contains(fixture, "/macos")
		fixtureWindows = fixtureWindows || strings.Contains(fixture, "/windows")
		handoff = handoff || strings.Contains(fixture, "testdata/handoff/kimi")
	}
	if !fixtureMacOS || !fixtureWindows || !handoff {
		t.Fatalf("fixtures = %v, want macos, windows, and testdata/handoff/kimi", d.Evidence.Fixtures)
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
