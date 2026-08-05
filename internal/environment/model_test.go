package environment

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRepositoryIDRemovesCredentialsAndTransportDetails(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"https://user:secret@GitHub.com/HarjjotSinghh/reinstate.git?token=secret#fragment",
		"ssh://git@github.com/HarjjotSinghh/reinstate.git",
		"git@github.com:HarjjotSinghh/reinstate.git",
	}
	want := NormalizeRepositoryID(inputs[0])
	if !strings.HasPrefix(want, "remote-sha256:") || len(want) != len("remote-sha256:")+64 {
		t.Fatalf("normalized identity = %q", want)
	}
	for _, input := range inputs {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeRepositoryID(input); got != want {
				t.Fatalf("NormalizeRepositoryID(%q) = %q, want %q", input, got, want)
			}
		})
	}
	for _, input := range []string{
		"user:secret@github.com/private/reinstate",
		"file:///Users/private/reinstate",
		"/Users/private/reinstate",
		`C:\Users\private\reinstate`,
	} {
		if got := NormalizeRepositoryID(input); got != "" {
			t.Fatalf("NormalizeRepositoryID(%q) = %q, want empty", input, got)
		}
	}
	if got := NormalizeRepositoryID(want); got != want {
		t.Fatalf("stored identity changed: %q", got)
	}
	roots := "roots-sha256:" + strings.Repeat("a", 64)
	if got := NormalizeRepositoryID(roots); got != roots {
		t.Fatalf("roots identity changed: %q", got)
	}
}

func TestPrelaunchInventoryBounds(t *testing.T) {
	t.Parallel()
	base := PrelaunchBaseline{
		SessionRef: "codex:bounded", WorkingTreeState: WorkingTreeUnavailable,
		ObservedAt: time.Now(), Provenance: PrelaunchObservedProvenance,
	}
	for index := 0; index <= MaxCapabilitiesPerKind; index++ {
		base.Capabilities = append(base.Capabilities, Capability{
			Agent: "codex", Kind: "mcp", Name: fmt.Sprintf("capability-%03d", index),
			Scope: "project", State: "enabled", Provenance: PrelaunchObservedProvenance,
		})
	}
	if _, err := NormalizePrelaunchBaseline(base); err == nil {
		t.Fatal("per-kind capability overflow unexpectedly accepted")
	}

	base.Capabilities = make([]Capability, MaxCapabilities+1)
	if _, err := NormalizePrelaunchBaseline(base); err == nil {
		t.Fatal("total capability overflow unexpectedly accepted")
	}
	base.Capabilities = nil
	base.Runtimes = make([]Runtime, MaxRuntimes+1)
	if _, err := NormalizePrelaunchBaseline(base); err == nil {
		t.Fatal("runtime overflow unexpectedly accepted")
	}
}

func TestNormalizeRecordedEnvironmentRequiresPerFieldProvenance(t *testing.T) {
	t.Parallel()
	_, err := NormalizeRecordedEnvironment(RecordedEnvironment{
		GitHead: RecordedField{Value: strings.Repeat("a", 40)},
	})
	if err == nil {
		t.Fatal("unproven git head unexpectedly accepted")
	}

	value, err := NormalizeRecordedEnvironment(RecordedEnvironment{
		RepositoryID: RecordedField{
			Value:      "https://user:secret@github.com/example/demo.git?token=secret",
			Provenance: "codex.session_meta.git.repository_url",
		},
		Branch: RecordedField{
			Value:      " feature/verified\nresume ",
			Provenance: "codex.session_meta.git.branch",
		},
		GitHead: RecordedField{
			Value:      strings.Repeat("A", 40),
			Provenance: "codex.session_meta.git.commit_hash",
		},
		Requirements: []Requirement{
			{Kind: "MCP", Name: "github", Provenance: "codex.session_meta.mcp"},
			{Kind: "mcp", Name: "github", Provenance: "codex.session_meta.mcp"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.RepositoryID.Value != NormalizeRepositoryID("git@github.com:example/demo.git") ||
		value.Branch.Value != "feature/verified resume" ||
		value.GitHead.Value != strings.Repeat("a", 40) {
		t.Fatalf("normalized environment = %+v", value)
	}
	if len(value.Requirements) != 1 || value.Requirements[0].Kind != "mcp" {
		t.Fatalf("normalized requirements = %+v", value.Requirements)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"user:secret", "token=secret", "github.com/example/demo"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("recorded environment leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestNormalizeEnvironmentStripsTerminalSequencesAndKeepsDistinctProvenance(t *testing.T) {
	t.Parallel()
	value, err := NormalizeRecordedEnvironment(RecordedEnvironment{
		Branch: RecordedField{Value: "\x1b[31mfeature/safe\x1b[0m", Provenance: "vendor.branch"},
		Requirements: []Requirement{
			{Kind: "mcp", Name: "github", Provenance: "vendor.requirement.one"},
			{Kind: "mcp", Name: "github", Provenance: "vendor.requirement.two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Branch.Value != "feature/safe" || strings.Contains(value.Branch.Value, "[31m") {
		t.Fatalf("sanitized branch = %q", value.Branch.Value)
	}
	if len(value.Requirements) != 2 || value.Requirements[0].Provenance == value.Requirements[1].Provenance {
		t.Fatalf("requirements = %+v", value.Requirements)
	}
}

func TestEnvironmentValidationErrorsDoNotEchoHostileValues(t *testing.T) {
	t.Parallel()
	sentinel := "PRIVATE-CONTROLLED-SENTINEL"
	_, err := NormalizePrelaunchBaseline(PrelaunchBaseline{
		SessionRef: "codex:one", WorkingTreeState: WorkingTreeState(sentinel),
		ObservedAt: time.Now(), Provenance: sentinel,
	})
	if err == nil || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("validation error = %v", err)
	}
	_, err = NormalizePrelaunchBaseline(PrelaunchBaseline{
		SessionRef: "codex:one", WorkingTreeState: WorkingTreeUnavailable,
		ObservedAt: time.Now(), Provenance: PrelaunchObservedProvenance,
		Capabilities: []Capability{{Agent: sentinel + "/escape", Kind: "mcp", Name: "one", Scope: "user", State: "present", Provenance: PrelaunchObservedProvenance}},
	})
	if err == nil || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("capability validation error = %v", err)
	}
}

func TestNormalizePrelaunchBaseline(t *testing.T) {
	t.Parallel()
	observed := time.Date(2026, 8, 5, 1, 2, 3, 4, time.FixedZone("test", 3600))
	baseline, err := NormalizePrelaunchBaseline(PrelaunchBaseline{
		SessionRef:        " codex:session-one ",
		RepositoryID:      "git@github.com:example/demo.git",
		Branch:            "main",
		GitHead:           strings.Repeat("B", 40),
		WorkingTreeDigest: strings.Repeat("C", 64),
		WorkingTreeState:  WorkingTreeModified,
		ObservedAt:        observed,
		Provenance:        PrelaunchObservedProvenance,
		SourceSessionRef:  "codex:source",
		Capabilities: []Capability{
			{Agent: "codex", Kind: "mcp", Name: "github", Scope: "project", State: "present", Provenance: PrelaunchObservedProvenance},
			{Agent: "claude", Kind: "skill", Name: "review", Scope: "user", State: "present", Provenance: PrelaunchObservedProvenance},
			{Agent: "codex", Kind: "mcp", Name: "github", Scope: "project", State: "present", Provenance: PrelaunchObservedProvenance},
		},
		Runtimes: []Runtime{
			{Name: "node", Version: "22.12.0", SourceKind: "executable", Provenance: PrelaunchObservedProvenance},
			{Name: "go", Version: "1.25.12", SourceKind: "go_mod", Provenance: PrelaunchObservedProvenance},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if baseline.SessionRef != "codex:session-one" ||
		baseline.RepositoryID != NormalizeRepositoryID("git@github.com:example/demo.git") ||
		baseline.GitHead != strings.Repeat("b", 40) ||
		baseline.WorkingTreeDigest != "sha256:"+strings.Repeat("c", 64) ||
		baseline.ObservedAt.Location() != time.UTC {
		t.Fatalf("normalized baseline = %+v", baseline)
	}
	if len(baseline.Capabilities) != 2 || baseline.Capabilities[0].Agent != "claude" ||
		len(baseline.Runtimes) != 2 || baseline.Runtimes[0].Name != "go" {
		t.Fatalf("normalized inventories = capabilities:%+v runtimes:%+v", baseline.Capabilities, baseline.Runtimes)
	}

	for name, invalid := range map[string]PrelaunchBaseline{
		"wrong provenance": {
			SessionRef: "codex:one", WorkingTreeState: WorkingTreeUnavailable,
			ObservedAt: observed, Provenance: "vendor_claim",
		},
		"missing digest": {
			SessionRef: "codex:one", WorkingTreeState: WorkingTreeClean,
			ObservedAt: observed, Provenance: PrelaunchObservedProvenance,
		},
		"zero observation": {
			SessionRef: "codex:one", WorkingTreeState: WorkingTreeUnavailable,
			Provenance: PrelaunchObservedProvenance,
		},
		"invalid capability": {
			SessionRef: "codex:one", WorkingTreeState: WorkingTreeUnavailable,
			ObservedAt: observed, Provenance: PrelaunchObservedProvenance,
			Capabilities: []Capability{{Agent: "codex", Kind: "mcp", Name: "github"}},
		},
	} {
		invalid := invalid
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NormalizePrelaunchBaseline(invalid); err == nil {
				t.Fatalf("invalid baseline unexpectedly accepted: %+v", invalid)
			}
		})
	}
}
