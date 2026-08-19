package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestCursorConformance(t *testing.T) {
	conformance.Run(t, Cursor(), conformance.Fixtures{
		Root: "testdata/sessionindex/cursor",
		OS:   []string{"macos", "windows"},
	})
}

func TestCursorIsDiscoverOnly(t *testing.T) {
	d := Cursor()
	if d.Key != "cursor" {
		t.Fatalf("key = %q", d.Key)
	}
	if d.DisplayName != "Cursor CLI" {
		t.Fatalf("DisplayName = %q", d.DisplayName)
	}
	if d.Tier != agents.TierDiscover {
		t.Fatalf("tier = %s, want T1", d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.Family != agents.FamilyHomeTree {
		t.Fatalf("family = %s, want F1 (meta.json via hometree)", d.Family)
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
	if d.Storage.Roots == nil || d.Storage.Marker != "chats" {
		t.Fatalf("CLI root must be marker-gated on chats: %+v", d.Storage)
	}
	if d.Storage.SessionGlob == "" {
		t.Fatal("T1 must declare the meta.json glob")
	}
	for _, want := range []string{"projects", "extensions", "plugins", "skills", "skills-cursor", "plans"} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %q", d.Storage.Excluded, want)
		}
	}
	if len(d.Process.Images) != 1 || d.Process.Images[0] != "cursor-agent" {
		t.Fatalf("Images = %v, want only cursor-agent", d.Process.Images)
	}
}

func TestCursorCitesBothPlatformProbes(t *testing.T) {
	d := Cursor()
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
