package catalog

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
)

func TestAiderConformance(t *testing.T) {
	conformance.Run(t, Aider(), conformance.Fixtures{})
}

func TestAiderStaysT0LayoutUnverified(t *testing.T) {
	d := Aider()
	if d.Key != "aider" || d.Tier != agents.TierKnown || d.T0Reason != agents.T0LayoutUnverified {
		t.Fatalf("aider identity = %s %s %s", d.Key, d.Tier, d.T0Reason)
	}
	if d.Family != agents.FamilyProjectFile {
		t.Fatalf("family = %s, want F4", d.Family)
	}
	if d.NewIndexSource != nil || d.NewReader != nil || d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T0 aider must not expose constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("T0 aider must not claim native resume or a version range")
	}
	if d.Storage.Roots != nil || d.Storage.RootEnv != "" {
		t.Fatal("T0 F4 aider must not declare a home root")
	}
	if d.Storage.SessionGlob != ".aider.chat.history.md" {
		t.Fatalf("SessionGlob = %q", d.Storage.SessionGlob)
	}
	if d.Storage.ProjectKey != agents.ProjectKeyNone {
		t.Fatalf("ProjectKey = %q, want none (one file per repo)", d.Storage.ProjectKey)
	}
	if len(d.Evidence.ProbeReports) != 1 {
		t.Fatalf("ProbeReports = %v, want the macOS artifact only", d.Evidence.ProbeReports)
	}
	if len(d.Evidence.Fixtures) != 0 || len(d.Evidence.DeviceReports) != 0 {
		t.Fatal("T0 F4 descriptor must not cite fixtures or device reports")
	}
}

func TestAiderExcludesSecretsAndNonSessions(t *testing.T) {
	d := Aider()
	want := []string{".aider.conf.yml", ".env", ".aider.input.history", ".aider.tags.cache*"}
	for _, name := range want {
		if !contains(d.Storage.Excluded, name) {
			t.Fatalf("excluded = %v, missing %s", d.Storage.Excluded, name)
		}
	}
}
