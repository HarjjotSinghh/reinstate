package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestCursorConformance(t *testing.T) {
	conformance.Run(t, Cursor(), conformance.Fixtures{})
}

func TestCursorStaysT0UntilWindows(t *testing.T) {
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
		t.Fatalf("family = %s, want F3; conversation bodies live in store.db", d.Family)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 descriptor must not grow a capability constructor")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 descriptor must not claim native resume or a version range")
	}
	if d.Storage.Roots == nil || d.Storage.Marker != "chats" {
		t.Fatalf("want ~/.cursor with chats marker, got %+v", d.Storage)
	}
	if d.Storage.Layout != "" || d.Storage.SessionGlob != "" {
		t.Fatalf("T0 must not claim a reader layout: %+v", d.Storage)
	}
	for _, want := range []string{"projects", "plugins", "skills", "skills-cursor"} {
		if !contains(d.Storage.Excluded, want) {
			t.Fatalf("excluded = %v, missing %q (editor tree)", d.Storage.Excluded, want)
		}
	}
	if len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 descriptor must not cite fixtures or device reports")
	}
	if d.Evidence.StoragePage != "docs/session-storage/cursor.md" {
		t.Fatalf("StoragePage = %q", d.Evidence.StoragePage)
	}
	var macOS bool
	for _, report := range d.Evidence.ProbeReports {
		macOS = macOS || strings.Contains(report, "-macos-")
	}
	if !macOS {
		t.Fatalf("probe reports = %v, want a macOS chats/ artifact", d.Evidence.ProbeReports)
	}
	if len(d.Process.Images) != 1 || d.Process.Images[0] != "cursor-agent" {
		t.Fatalf("Images = %v, want only cursor-agent", d.Process.Images)
	}
}
