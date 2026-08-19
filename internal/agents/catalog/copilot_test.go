package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestCopilotConformance(t *testing.T) {
	conformance.Run(t, Copilot(), conformance.Fixtures{
		Root: "testdata/sessionindex/copilot",
		OS:   []string{"macos", "windows"},
	})
}

func TestCopilotIsDiscoverOnly(t *testing.T) {
	got := Copilot()
	if got.Key != "copilot" {
		t.Fatalf("key = %q", got.Key)
	}
	if got.Tier != agents.TierDiscover {
		t.Fatalf("tier = %s, want T1", got.Tier)
	}
	if got.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", got.T0Reason)
	}
	if got.NewIndexSource == nil {
		t.Fatal("T1 requires an index source")
	}
	if got.NewReader != nil || got.NewTarget != nil || got.NewSyncAdapter != nil {
		t.Fatal("T1 descriptor must not ship reader, target, or sync constructors")
	}
	if got.Native != nil || got.Version != nil {
		t.Fatal("native resume and a version range are T3 claims")
	}
}

func TestCopilotCitesBothPlatformProbes(t *testing.T) {
	d := Copilot()
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

func TestCopilotExcludesCredentials(t *testing.T) {
	d := Copilot()
	for _, want := range []string{"config.json", "mcp-oauth-config", "mcp-secrets"} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %s", d.Storage.Excluded, want)
		}
	}
}
