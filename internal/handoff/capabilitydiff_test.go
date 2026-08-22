package handoff

import (
	"reflect"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
)

func TestDiffCapabilitiesMissingSourceMCP(t *testing.T) {
	t.Parallel()

	source := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "browser", Scope: capability.ScopeUser, State: capability.StateDeclared},
		{Agent: capability.AgentClaude, Kind: capability.KindSkill, Name: "review", Scope: capability.ScopeProject, State: capability.StateCandidate},
	}}
	destination := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentCodex, Kind: capability.KindSkill, Name: "review", Scope: capability.ScopeUser, State: capability.StateDeclared},
	}}

	diff := DiffCapabilities(source, destination, "claude", "codex")
	if got := diff.Source["mcp_count"]; got != 1 {
		t.Fatalf("source mcp_count = %v, want 1", got)
	}
	if got := diff.Destination["mcp_count"]; got != 0 {
		t.Fatalf("destination mcp_count = %v, want 0", got)
	}

	want := []capsule.MissingCapability{
		{Kind: KindAttachment, Name: "support", Impact: ImpactInformational},
		{Kind: KindMCP, Name: "browser", Impact: ImpactDegraded},
	}
	if !reflect.DeepEqual(diff.Missing, want) {
		t.Fatalf("Missing = %+v, want %+v", diff.Missing, want)
	}
}

func TestDiffCapabilitiesUnverifiedDoesNotSatisfyPresence(t *testing.T) {
	t.Parallel()

	source := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "docs", Scope: capability.ScopeUser, State: capability.StateDeclared},
	}}
	destination := capability.Inventory{Items: []capability.Item{
		// Symlink / unverified names are diagnostics only — not presence.
		{Agent: capability.AgentCodex, Kind: capability.KindMCP, Name: "docs", Scope: capability.ScopeUser, State: capability.StateUnverified},
	}}

	diff := DiffCapabilities(source, destination, "claude", "codex")
	found := false
	for _, m := range diff.Missing {
		if m.Kind == KindMCP && m.Name == "docs" {
			found = true
			if m.Impact != ImpactDegraded {
				t.Fatalf("docs impact = %q, want %q", m.Impact, ImpactDegraded)
			}
		}
	}
	if !found {
		t.Fatalf("Missing = %+v, want mcp docs degraded", diff.Missing)
	}
}

func TestDiffCapabilitiesUnverifiedSourceDoesNotCreateGap(t *testing.T) {
	t.Parallel()

	source := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "ghost", Scope: capability.ScopeUser, State: capability.StateUnverified},
	}}
	destination := capability.Inventory{}

	diff := DiffCapabilities(source, destination, "claude", "codex")
	for _, m := range diff.Missing {
		if m.Kind == KindMCP && m.Name == "ghost" {
			t.Fatalf("unverified source MCP must not create Missing: %+v", diff.Missing)
		}
	}
}

func TestDiffCapabilitiesWarningIDsStableAndSorted(t *testing.T) {
	t.Parallel()

	mk := func(order []capability.Item) capability.Inventory {
		return capability.Inventory{Items: order}
	}
	a := capability.Item{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "zeta", State: capability.StateDeclared}
	b := capability.Item{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "alpha", State: capability.StateDeclared}
	c := capability.Item{Agent: capability.AgentClaude, Kind: capability.KindSkill, Name: "lint", State: capability.StateCandidate}
	d := capability.Item{Agent: capability.AgentClaude, Kind: capability.KindInstruction, Name: "CLAUDE.md", State: capability.StateDeclared}

	diff1 := DiffCapabilities(mk([]capability.Item{a, b, c, d}), capability.Inventory{}, "claude", "codex")
	diff2 := DiffCapabilities(mk([]capability.Item{d, c, b, a}), capability.Inventory{}, "claude", "codex")

	if !reflect.DeepEqual(diff1.Missing, diff2.Missing) {
		t.Fatalf("Missing order drifted:\n1=%+v\n2=%+v", diff1.Missing, diff2.Missing)
	}

	ids1 := CapabilityWarningIDs(diff1)
	ids2 := CapabilityWarningIDs(diff2)
	if !reflect.DeepEqual(ids1, ids2) {
		t.Fatalf("warning IDs drifted:\n1=%v\n2=%v", ids1, ids2)
	}
	wantPrefix := []string{
		"handoff.capability.attachment.support",
		"handoff.capability.instruction.claude-md",
		"handoff.capability.mcp.alpha",
		"handoff.capability.mcp.zeta",
		"handoff.capability.skill.lint",
	}
	if !reflect.DeepEqual(ids1, wantPrefix) {
		t.Fatalf("warning IDs = %v, want %v", ids1, wantPrefix)
	}
	for i := 1; i < len(ids1); i++ {
		if ids1[i-1] >= ids1[i] {
			t.Fatalf("warning IDs not strictly sorted: %v", ids1)
		}
	}
}

func TestDiffCapabilitiesContextCeilingOmittedR7(t *testing.T) {
	t.Parallel()

	diff := DiffCapabilities(capability.Inventory{}, capability.Inventory{}, "claude", "codex")
	for _, side := range []map[string]any{diff.Source, diff.Destination} {
		if side["context_ceiling"] != summaryOmitted {
			t.Fatalf("context_ceiling = %v, want %q", side["context_ceiling"], summaryOmitted)
		}
		if side["context_ceiling_reason"] != contextCeilingReason {
			t.Fatalf("context_ceiling_reason = %v, want %q", side["context_ceiling_reason"], contextCeilingReason)
		}
	}
	for _, m := range diff.Missing {
		if m.Kind == KindContext {
			t.Fatalf("both-omitted ceilings must not invent a context Missing: %+v", diff.Missing)
		}
	}
}

func TestDiffCapabilitiesSameAgentNoAttachmentGap(t *testing.T) {
	t.Parallel()

	diff := DiffCapabilities(capability.Inventory{}, capability.Inventory{}, "claude", "claude")
	for _, m := range diff.Missing {
		if m.Kind == KindAttachment {
			t.Fatalf("same-agent Claude handoff must not flag attachment: %+v", diff.Missing)
		}
	}
	if diff.Source["attachments"] != true || diff.Destination["attachments"] != true {
		t.Fatalf("claude attachments = source=%v dest=%v, want true/true", diff.Source["attachments"], diff.Destination["attachments"])
	}
}

func TestWarningID(t *testing.T) {
	t.Parallel()

	got := WarningID(Missing{Kind: KindMCP, Name: "Browser Server", Impact: ImpactDegraded})
	if got != "handoff.capability.mcp.browserserver" {
		t.Fatalf("WarningID = %q", got)
	}
}

// TestDiffCapabilitiesDoesNotAssertGapsAtAnUnenumeratedDestination is the rule
// that "absent from the destination inventory" only means the destination lacks
// it when the destination was actually enumerated. Capability discovery covers
// Claude Code and Codex; anything else arrives with an empty inventory, and
// calling every source capability degraded there asserts a gap nobody looked
// for.
func TestDiffCapabilitiesDoesNotAssertGapsAtAnUnenumeratedDestination(t *testing.T) {
	t.Parallel()

	source := capability.Inventory{Items: []capability.Item{
		{Agent: capability.AgentClaude, Kind: capability.KindMCP, Name: "browser", Scope: capability.ScopeUser, State: capability.StateDeclared},
		{Agent: capability.AgentClaude, Kind: capability.KindSkill, Name: "review", Scope: capability.ScopeUser, State: capability.StateDeclared},
	}}

	enumerated := DiffCapabilities(source, capability.Inventory{}, "claude", "codex")
	for _, m := range enumerated.Missing {
		if m.Kind != KindMCP && m.Kind != KindSkill {
			continue
		}
		if m.Impact != ImpactDegraded {
			t.Fatalf("codex is enumerated, so %s/%s should be degraded, got %s", m.Kind, m.Name, m.Impact)
		}
	}

	unenumerated := DiffCapabilities(source, capability.Inventory{}, "claude", "qwen")
	seen := 0
	for _, m := range unenumerated.Missing {
		if m.Kind != KindMCP && m.Kind != KindSkill {
			continue
		}
		seen++
		if m.Impact != ImpactInformational {
			t.Fatalf("qwen is never enumerated, so %s/%s must be informational, got %s", m.Kind, m.Name, m.Impact)
		}
	}
	if seen != 2 {
		t.Fatalf("expected both source capabilities to be reported, saw %d", seen)
	}
}

func TestCapabilityDiscoverySupportedMatchesDiscoverInventory(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{"claude": true, "codex": true, "qwen": false, "gemini": false, "": false}
	for agent, want := range tests {
		if got := capabilityDiscoverySupported(agent); got != want {
			t.Fatalf("capabilityDiscoverySupported(%q) = %t, want %t", agent, got, want)
		}
	}
}
