package handoff

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestApplyCheckpointIncludesZeroVerbatim(t *testing.T) {
	t.Parallel()

	events := sampleEvents(5, 100)
	included, sidecar, report := Apply(PolicyCheckpoint, events)

	if len(included) != 0 {
		t.Fatalf("checkpoint included = %d, want 0", len(included))
	}
	if len(sidecar) != len(events) {
		t.Fatalf("sidecar = %d, want %d", len(sidecar), len(events))
	}
	assertPartition(t, events, included, sidecar)
	if report.Overall != capsule.PortabilityReferenced {
		t.Fatalf("Overall = %q, want referenced", report.Overall)
	}
	for _, ref := range sidecar {
		if ref.Portability != capsule.PortabilityReferenced {
			t.Fatalf("sidecar %q portability = %q", ref.EventID, ref.Portability)
		}
	}
}

func TestApplyBalancedRespectsByteBudgetExactly(t *testing.T) {
	t.Parallel()

	// Newest fits; next overflows remaining and is truncated to fill the budget exactly.
	budget := DefaultProjectionBudgetBytes
	half := budget / 2
	events := []capsule.Event{
		msgEvent("e0", 0, strings.Repeat("a", half)),
		msgEvent("e1", 1, strings.Repeat("b", half+100)),
		msgEvent("e2", 2, strings.Repeat("c", half)),
	}

	included, sidecar, _ := Apply(PolicyBalanced, events)
	assertPartition(t, events, included, sidecar)

	got := 0
	for _, e := range included {
		got += eventTextBytes(e)
	}
	if got > budget {
		t.Fatalf("included bytes = %d, over budget %d", got, budget)
	}
	if got != budget {
		t.Fatalf("included bytes = %d, want exact budget fill %d", got, budget)
	}

	// Newest-first: e2 and e1 selected (e1 truncated), e0 sidecar; source order preserved.
	if len(included) != 2 {
		t.Fatalf("included len = %d, want 2", len(included))
	}
	if included[0].ID != "e1" || included[1].ID != "e2" {
		t.Fatalf("included ids = %q,%q want e1,e2 (source order)", included[0].ID, included[1].ID)
	}
	if !included[0].Truncated {
		t.Fatal("marginal included event must be truncated")
	}
	if !strings.HasSuffix(included[0].Blocks[0].Text, transcript.TruncationMarker) {
		t.Fatalf("truncated marker missing: %q", included[0].Blocks[0].Text)
	}
	if len(sidecar) != 1 || sidecar[0].EventID != "e0" {
		t.Fatalf("sidecar = %+v, want e0 only", sidecar)
	}
}

func TestApplyFullCapsAtHardLimitAndReferencesOverflow(t *testing.T) {
	t.Parallel()

	capBytes := HardProjectionCapBytes
	chunk := capBytes / 2
	events := []capsule.Event{
		msgEvent("old", 0, strings.Repeat("x", chunk+1024)),
		msgEvent("mid", 1, strings.Repeat("y", chunk)),
		msgEvent("new", 2, strings.Repeat("z", chunk)),
	}

	included, sidecar, report := Apply(PolicyFull, events)
	assertPartition(t, events, included, sidecar)

	got := 0
	for _, e := range included {
		got += eventTextBytes(e)
	}
	if got > capBytes {
		t.Fatalf("included bytes = %d, over hard cap %d", got, capBytes)
	}
	if got != capBytes {
		t.Fatalf("included bytes = %d, want exact hard-cap fill %d", got, capBytes)
	}
	if len(sidecar) == 0 {
		t.Fatal("expected overflow sidecar refs")
	}
	for _, ref := range sidecar {
		if ref.Portability != capsule.PortabilityReferenced {
			t.Fatalf("overflow ref %q portability = %q", ref.EventID, ref.Portability)
		}
	}
	if report.Overall != capsule.PortabilityReferenced && report.Overall != capsule.PortabilitySummarized {
		t.Fatalf("Overall = %q, want referenced or summarized", report.Overall)
	}
}

func TestApplyPartitionCoversAllInputs(t *testing.T) {
	t.Parallel()

	events := sampleEvents(12, 2048)
	for _, p := range []Policy{PolicyCheckpoint, PolicyBalanced, PolicyFull, ""} {
		included, sidecar, _ := Apply(p, events)
		assertPartition(t, events, included, sidecar)
	}
}

func TestApplyDefaultPolicyIsBalanced(t *testing.T) {
	t.Parallel()

	events := sampleEvents(3, 100)
	aInc, aSide, _ := Apply("", events)
	bInc, bSide, _ := Apply(PolicyBalanced, events)
	if len(aInc) != len(bInc) || len(aSide) != len(bSide) {
		t.Fatalf("empty policy != balanced: included %d/%d sidecar %d/%d",
			len(aInc), len(bInc), len(aSide), len(bSide))
	}
}

func TestApplyReferencedAndOmittedStaySidecar(t *testing.T) {
	t.Parallel()

	events := []capsule.Event{
		msgEvent("keep", 0, "hello"),
		{
			ID:          "sys",
			Order:       1,
			Actor:       capsule.ActorHarness,
			Kind:        capsule.KindMetadata,
			Portability: capsule.PortabilityReferenced,
			Reason:      "source_system_instruction",
			Blocks:      []capsule.Block{{Type: capsule.BlockTypeText, Text: "system"}},
			ContentHash: "sys",
		},
		{
			ID:          "omit",
			Order:       2,
			Actor:       capsule.ActorAssistant,
			Kind:        capsule.KindMessage,
			Portability: capsule.PortabilityOmitted,
			Reason:      "vendor_opaque_state",
			Blocks:      []capsule.Block{{Type: capsule.BlockTypeText, Text: "hidden"}},
			ContentHash: "omit",
		},
	}

	included, sidecar, _ := Apply(PolicyFull, events)
	assertPartition(t, events, included, sidecar)
	if len(included) != 1 || included[0].ID != "keep" {
		t.Fatalf("included = %+v, want keep only", idsOf(included))
	}
	if len(sidecar) != 2 {
		t.Fatalf("sidecar len = %d, want 2", len(sidecar))
	}
}

func TestEstimateTokensCeilBytesOver4(t *testing.T) {
	t.Parallel()

	cases := []struct {
		n    int
		want int
	}{
		{0, 0},
		{1, 1},
		{4, 1},
		{5, 2},
		{8, 2},
		{9, 3},
	}
	for _, tc := range cases {
		got := EstimateTokens(make([]byte, tc.n))
		if got != tc.want {
			t.Fatalf("EstimateTokens(%d bytes) = %d, want %d", tc.n, got, tc.want)
		}
	}
}

func sampleEvents(n, bytesEach int) []capsule.Event {
	out := make([]capsule.Event, n)
	for i := 0; i < n; i++ {
		id := "e" + strings.Repeat("x", i+1)
		out[i] = msgEvent(id, i, strings.Repeat("x", bytesEach))
	}
	return out
}

func msgEvent(id string, order int, text string) capsule.Event {
	return capsule.Event{
		ID:          id,
		Order:       order,
		Actor:       capsule.ActorUser,
		Kind:        capsule.KindMessage,
		Portability: capsule.PortabilityExact,
		Blocks:      []capsule.Block{{Type: capsule.BlockTypeText, Text: text, Size: int64(len(text))}},
		ContentHash: id,
		Source:      capsule.SourcePointer{Agent: "test", SessionID: "s", Index: order},
	}
}

func assertPartition(t *testing.T, input []capsule.Event, included []capsule.Event, sidecar []capsule.SidecarRef) {
	t.Helper()
	if len(included)+len(sidecar) != len(input) {
		t.Fatalf("included(%d)+sidecar(%d) != input(%d)", len(included), len(sidecar), len(input))
	}
	seen := make(map[string]bool, len(input))
	for _, e := range included {
		if seen[e.ID] {
			t.Fatalf("duplicate included id %q", e.ID)
		}
		seen[e.ID] = true
	}
	for _, ref := range sidecar {
		if seen[ref.EventID] {
			t.Fatalf("id %q in both included and sidecar", ref.EventID)
		}
		seen[ref.EventID] = true
		if ref.Portability != capsule.PortabilityReferenced {
			t.Fatalf("sidecar %q portability = %q", ref.EventID, ref.Portability)
		}
	}
	for _, e := range input {
		if !seen[e.ID] {
			t.Fatalf("input event %q missing from partition", e.ID)
		}
	}
}

func idsOf(events []capsule.Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.ID
	}
	return out
}

func TestApplyPullsMatchingToolCallWhenResultFitsBudget(t *testing.T) {
	t.Parallel()

	// Newest-first: a result that nearly fills the budget excludes its older
	// call. Pairing must pull the call back in so Validate accepts the capsule.
	call := toolCallEvent("call-ev", 0, "call-abc", strings.Repeat("c", 200))
	result := toolResultEvent("result-ev", 1, "call-abc", strings.Repeat("r", DefaultProjectionBudgetBytes-5))
	events := []capsule.Event{call, result}

	included, sidecar, _ := Apply(PolicyBalanced, events)
	assertPartition(t, events, included, sidecar)
	assertToolResultsPaired(t, included)
	if len(included) != 2 {
		t.Fatalf("included = %d, want call+result", len(included))
	}
	if included[0].ID != "call-ev" || included[1].ID != "result-ev" {
		t.Fatalf("included ids = %v, want call-ev,result-ev", idsOf(included))
	}
}

func TestApplyOmitsToolResultWhenCallIsAbsentFromSource(t *testing.T) {
	t.Parallel()

	orphan := toolResultEvent("orphan-ev", 0, "call-missing", "output")
	msg := msgEvent("msg-ev", 1, "hello")
	events := []capsule.Event{orphan, msg}

	included, sidecar, _ := Apply(PolicyBalanced, events)
	assertPartition(t, events, included, sidecar)
	assertToolResultsPaired(t, included)
	if len(included) != 1 || included[0].ID != "msg-ev" {
		t.Fatalf("included = %v, want msg-ev only", idsOf(included))
	}
	if len(sidecar) != 1 || sidecar[0].EventID != "orphan-ev" {
		t.Fatalf("sidecar = %+v, want orphan-ev", sidecar)
	}
	if sidecar[0].Reason != reasonUnmatchedToolResult {
		t.Fatalf("sidecar reason = %q, want %q", sidecar[0].Reason, reasonUnmatchedToolResult)
	}
}

func toolCallEvent(id string, order int, callID, args string) capsule.Event {
	return capsule.Event{
		ID:          id,
		Order:       order,
		Actor:       capsule.ActorAssistant,
		Kind:        capsule.KindToolCall,
		CallID:      callID,
		Portability: capsule.PortabilityNormalized,
		Reason:      "normalized_tool_call",
		Blocks:      []capsule.Block{{Type: capsule.BlockTypeToolInput, Text: args, Size: int64(len(args))}},
		ContentHash: id,
		Source:      capsule.SourcePointer{Agent: "codex", SessionID: "s", Index: order},
	}
}

func toolResultEvent(id string, order int, linked, output string) capsule.Event {
	return capsule.Event{
		ID:           id,
		Order:        order,
		Actor:        capsule.ActorTool,
		Kind:         capsule.KindToolResult,
		LinkedCallID: linked,
		Portability:  capsule.PortabilityNormalized,
		Reason:       "normalized_tool_result",
		Blocks:       []capsule.Block{{Type: capsule.BlockTypeToolOutput, Text: output, Size: int64(len(output))}},
		ContentHash:  id,
		Source:       capsule.SourcePointer{Agent: "codex", SessionID: "s", Index: order},
	}
}

func assertToolResultsPaired(t *testing.T, events []capsule.Event) {
	t.Helper()
	calls := make(map[string]struct{})
	for _, e := range events {
		if e.Kind == capsule.KindToolCall && e.CallID != "" {
			calls[e.CallID] = struct{}{}
		}
	}
	for _, e := range events {
		if e.Kind != capsule.KindToolResult {
			continue
		}
		if e.LinkedCallID == "" {
			t.Fatalf("included tool_result %q has empty linked_call_id", e.ID)
		}
		if _, ok := calls[e.LinkedCallID]; !ok {
			t.Fatalf("included tool_result %q linked_call_id %q has no matching tool_call", e.ID, e.LinkedCallID)
		}
	}
}

func TestApplyFidelityReportContainsAllPortabilityClasses(t *testing.T) {
	t.Parallel()
	events := []capsule.Event{
		{
			ID: "already-ref", Kind: capsule.KindAttachment, Actor: capsule.ActorAssistant,
			Portability: capsule.PortabilityReferenced, Reason: "on_disk",
			Blocks: []capsule.Block{{Text: "blob"}},
		},
		{
			ID: "omit", Kind: capsule.KindMessage, Actor: capsule.ActorHarness,
			Portability: capsule.PortabilityOmitted, Reason: "hidden_reasoning",
			Blocks: []capsule.Block{{Text: "secret"}},
		},
		{
			ID: "exact", Kind: capsule.KindMessage, Actor: capsule.ActorUser,
			Portability: capsule.PortabilityExact, Blocks: []capsule.Block{{Text: "hello"}},
		},
		{
			ID: "normalized", Kind: capsule.KindMessage, Actor: capsule.ActorAssistant,
			Portability: capsule.PortabilityNormalized, Blocks: []capsule.Block{{Text: "${REPO:x}/file.go"}},
		},
		{
			ID: "summarized", Kind: capsule.KindSummary, Actor: capsule.ActorAssistant,
			Portability: capsule.PortabilitySummarized, Reason: "vendor_summary",
			Blocks: []capsule.Block{{Text: "earlier work"}},
		},
	}
	_, sidecar, report := Apply(PolicyBalanced, events)
	if len(sidecar) == 0 {
		t.Fatal("expected overflow events to be referenced")
	}
	seen := map[capsule.Portability]bool{}
	for _, component := range report.Components {
		seen[component.Portability] = true
	}
	for _, want := range []capsule.Portability{
		capsule.PortabilityExact, capsule.PortabilityNormalized, capsule.PortabilitySummarized,
		capsule.PortabilityReferenced, capsule.PortabilityOmitted,
	} {
		if !seen[want] {
			t.Errorf("fidelity missing %s: %+v", want, report.Components)
		}
	}
}

func TestApplyCheckpointKeepsSummarizedFidelity(t *testing.T) {
	t.Parallel()
	events := []capsule.Event{
		{
			ID: "summarized", Kind: capsule.KindSummary, Actor: capsule.ActorAssistant,
			Portability: capsule.PortabilitySummarized, Reason: "vendor_summary",
			Blocks: []capsule.Block{{Text: "earlier work"}},
		},
		msgEvent("keep", 1, "hello"),
	}
	included, sidecar, report := Apply(PolicyCheckpoint, events)
	if len(included) != 0 {
		t.Fatalf("checkpoint included = %d, want 0", len(included))
	}
	if len(sidecar) != 2 {
		t.Fatalf("sidecar = %d, want 2", len(sidecar))
	}
	seen := map[capsule.Portability]bool{}
	for _, component := range report.Components {
		seen[component.Portability] = true
	}
	if !seen[capsule.PortabilitySummarized] {
		t.Fatalf("checkpoint fidelity dropped summarized: %+v", report.Components)
	}
}
