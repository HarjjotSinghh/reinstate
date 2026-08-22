// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package handoffui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
)

// The destinations used across both test files. They are catalog agent keys,
// but no test here reaches a vendor: every plan below is synthetic.
const (
	destCodex    = "codex"
	destGemini   = "gemini"
	destClaude   = "claude"
	destOpenCode = "opencode"
)

// fixtureReference is the source session every studio in these tests hands off.
const fixtureReference = "claude:5f1c0b2a3e7d"

// secretValue is planted in the synthetic capsule wherever a real transcript
// would carry a credential. No frame may ever contain it: the studio reports
// redaction counts and category names, never values.
const secretValue = "sk-live-51H9xQnotarealkey"

// planShape is the synthetic measurement for one projection policy.
//
// Every number differs between policies, and fixturePlan offsets them again per
// destination, so a cache that served the wrong entry shows up as a wrong
// number rather than as a plausible one.
type planShape struct {
	events     int
	bytes      int64
	tokens     int
	overall    capsule.Portability
	components []capsule.Component
	redactions map[string]int
	warnings   []string
}

// planShapes is deliberately written with components out of alphabetical order,
// so previewFromPlan sorting them is an assertion rather than a coincidence.
var planShapes = map[string]planShape{
	string(handoff.PolicyCheckpoint): {
		events:  0,
		bytes:   812,
		tokens:  204,
		overall: capsule.PortabilitySummarized,
		components: []capsule.Component{
			{Name: "user_messages", Portability: capsule.PortabilitySummarized, Count: 1, Bytes: 512},
			{Name: "tool_results", Portability: capsule.PortabilityOmitted, Reason: "checkpoint_policy"},
			{Name: "attachments", Portability: capsule.PortabilityOmitted, Reason: "checkpoint_policy"},
		},
	},
	string(handoff.PolicyBalanced): {
		events:  12,
		bytes:   11776,
		tokens:  2940,
		overall: capsule.PortabilityNormalized,
		components: []capsule.Component{
			{Name: "tool_results", Portability: capsule.PortabilityNormalized, Count: 9, Bytes: 4096, Reason: "arguments_normalized"},
			{Name: "attachments", Portability: capsule.PortabilityReferenced, Count: 2, Reason: "left_in_place"},
			{Name: "user_messages", Portability: capsule.PortabilityExact, Count: 12, Bytes: 6144},
			{Name: "subagent_transcripts", Portability: capsule.PortabilityOmitted, Reason: "not_representable"},
		},
		redactions: map[string]int{"api_key": 2, "aws_access_key_id": 1},
		warnings:   []string{"source_may_have_advanced"},
	},
	string(handoff.PolicyFull): {
		events:  37,
		bytes:   48128,
		tokens:  12400,
		overall: capsule.PortabilityNormalized,
		components: []capsule.Component{
			{Name: "tool_results", Portability: capsule.PortabilityNormalized, Count: 34, Bytes: 20480, Reason: "arguments_normalized"},
			{Name: "user_messages", Portability: capsule.PortabilityExact, Count: 37, Bytes: 26624},
			{Name: "attachments", Portability: capsule.PortabilityReferenced, Count: 5, Reason: "left_in_place"},
			{Name: "subagent_transcripts", Portability: capsule.PortabilityOmitted, Reason: "not_representable"},
		},
		redactions: map[string]int{"api_key": 4, "aws_access_key_id": 1, "bearer_token": 2},
		warnings:   []string{"source_may_have_advanced", "destination_lacks_subagents"},
	},
}

// destinationDeltas make the same policy measure differently per destination,
// which is what a real capability diff does to a projection.
var destinationDeltas = map[string]int{
	destCodex:    0,
	destGemini:   96,
	destClaude:   192,
	destOpenCode: 288,
}

// fixturePlan is the entire engine dependency of this package's tests: one
// synthetic PlanResult per destination and policy. Nothing here parses a
// transcript, probes a vendor, or touches the filesystem.
func fixturePlan(destination, policy string) handoff.PlanResult {
	shape, ok := planShapes[policy]
	if !ok {
		shape = planShapes[string(handoff.PolicyBalanced)]
	}
	delta := destinationDeltas[destination]

	result := handoff.PlanResult{
		EstimatedBytes:  shape.bytes + int64(delta),
		EstimatedTokens: shape.tokens + delta,
		WarningIDs:      append([]string(nil), shape.warnings...),
	}
	if len(shape.redactions) > 0 {
		result.RedactionCounts = make(map[string]int, len(shape.redactions))
		for name, count := range shape.redactions {
			result.RedactionCounts[name] = count
		}
	}
	result.Capsule.Fidelity.Overall = shape.overall
	result.Capsule.Fidelity.Components = append([]capsule.Component(nil), shape.components...)
	result.Capsule.Conversation.Events = fixtureEvents(shape.events)
	return result
}

// fixtureEvents builds n events whose text carries the planted secret. The
// studio counts events and never reads them, which is precisely what the
// redaction test proves about the rendered frame.
func fixtureEvents(count int) []capsule.Event {
	events := make([]capsule.Event, 0, count)
	for i := 0; i < count; i++ {
		events = append(events, capsule.Event{
			ID:          fmt.Sprintf("e%02d", i),
			Order:       i,
			Actor:       capsule.ActorUser,
			Kind:        capsule.KindMessage,
			Portability: capsule.PortabilityExact,
			Blocks: []capsule.Block{{
				Type: capsule.BlockTypeText,
				Text: "export DEPLOY_TOKEN=" + secretValue,
			}},
		})
	}
	return events
}

// fakePlanner answers plans from the fixture, counts every call, and can fail a
// chosen pair. Counting is how the tests prove a preview is computed once and
// then served from cache.
type fakePlanner struct {
	mu    sync.Mutex
	calls map[string]int
	// fail maps previewKey(destination, policy) to the error that pair returns.
	fail map[string]error
	// components overrides the fidelity components for every plan, for the
	// tests that need hostile or absent component names.
	components []capsule.Component
	// mutate rewrites a plan after the fixture built it.
	mutate func(destination, policy string, result *handoff.PlanResult)
}

func newFakePlanner() *fakePlanner {
	return &fakePlanner{calls: make(map[string]int)}
}

func (f *fakePlanner) plan(ctx context.Context, destination, policy string) (handoff.PlanResult, error) {
	f.mu.Lock()
	if f.calls == nil {
		f.calls = make(map[string]int)
	}
	f.calls[previewKey(destination, policy)]++
	failure := f.fail[previewKey(destination, policy)]
	components := f.components
	mutate := f.mutate
	f.mu.Unlock()

	if failure != nil {
		return handoff.PlanResult{}, failure
	}
	result := fixturePlan(destination, policy)
	if components != nil {
		result.Capsule.Fidelity.Components = append([]capsule.Component(nil), components...)
	}
	if mutate != nil {
		mutate(destination, policy, &result)
	}
	return result, nil
}

func (f *fakePlanner) callCount(destination, policy string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[previewKey(destination, policy)]
}

func (f *fakePlanner) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for _, count := range f.calls {
		total += count
	}
	return total
}

// run executes a command and returns the message it produced, failing the test
// when the command is missing. Every planner command is expected to settle.
func run(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	return cmd()
}

// TestComponentIncluded pins which portability classes actually travel. A
// referenced or omitted component is named in the capsule but its content stays
// behind, which is the distinction a reader picking a policy is looking at.
func TestComponentIncluded(t *testing.T) {
	tests := []struct {
		name        string
		portability string
		want        bool
	}{
		{"exact", string(capsule.PortabilityExact), true},
		{"normalized", string(capsule.PortabilityNormalized), true},
		{"summarized", string(capsule.PortabilitySummarized), true},
		{"referenced", string(capsule.PortabilityReferenced), false},
		{"omitted", string(capsule.PortabilityOmitted), false},
		{"empty", "", false},
		{"unknown", "teleported", false},
		{"wrong case", "EXACT", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			component := Component{Name: "user_messages", Portability: test.portability}
			if got := component.Included(); got != test.want {
				t.Fatalf("Included() = %v for portability %q, want %v", got, test.portability, test.want)
			}
		})
	}
}

// TestEveryPortabilityConstantIsClassified fails when a new portability class is
// added to capsule without a decision here about whether it travels.
func TestEveryPortabilityConstantIsClassified(t *testing.T) {
	all := []capsule.Portability{
		capsule.PortabilityExact,
		capsule.PortabilityNormalized,
		capsule.PortabilitySummarized,
		capsule.PortabilityReferenced,
		capsule.PortabilityOmitted,
	}
	included := 0
	for _, portability := range all {
		if (Component{Portability: string(portability)}).Included() {
			included++
		}
	}
	if included != 3 {
		t.Fatalf("%d of %d portability classes travel, want exactly exact, normalized and summarized",
			included, len(all))
	}
}

func TestPlannerComputesAndCachesAPreview(t *testing.T) {
	t.Run("maps every measured field", func(t *testing.T) {
		fake := newFakePlanner()
		planner := NewPlanner(fake.plan)

		msg := run(t, planner.Compute(context.Background(), destGemini, string(handoff.PolicyBalanced)))
		previewed, ok := msg.(PreviewedMsg)
		if !ok {
			t.Fatalf("command returned %T, want PreviewedMsg", msg)
		}
		if previewed.Destination != destGemini || previewed.Policy != string(handoff.PolicyBalanced) {
			t.Fatalf("message = %+v, want the pair that was computed", previewed)
		}

		preview, ready := planner.Lookup(destGemini, string(handoff.PolicyBalanced))
		if !ready {
			t.Fatal("the preview was not cached after the command ran")
		}
		want := fixturePlan(destGemini, string(handoff.PolicyBalanced))
		if preview.Err != nil {
			t.Fatalf("Err = %v, want none", preview.Err)
		}
		if preview.Destination != destGemini || preview.Policy != string(handoff.PolicyBalanced) {
			t.Fatalf("preview identifies %q/%q, want %q/%q",
				preview.Destination, preview.Policy, destGemini, handoff.PolicyBalanced)
		}
		if preview.Bytes != want.EstimatedBytes {
			t.Errorf("Bytes = %d, want %d", preview.Bytes, want.EstimatedBytes)
		}
		if preview.Tokens != want.EstimatedTokens {
			t.Errorf("Tokens = %d, want %d", preview.Tokens, want.EstimatedTokens)
		}
		if preview.Events != len(want.Capsule.Conversation.Events) {
			t.Errorf("Events = %d, want %d", preview.Events, len(want.Capsule.Conversation.Events))
		}
		if preview.Overall != string(want.Capsule.Fidelity.Overall) {
			t.Errorf("Overall = %q, want %q", preview.Overall, want.Capsule.Fidelity.Overall)
		}
		if strings.Join(preview.Warnings, ",") != strings.Join(want.WarningIDs, ",") {
			t.Errorf("Warnings = %v, want %v", preview.Warnings, want.WarningIDs)
		}
		if len(preview.Redactions) != len(want.RedactionCounts) {
			t.Fatalf("Redactions = %v, want %v", preview.Redactions, want.RedactionCounts)
		}
		for name, count := range want.RedactionCounts {
			if preview.Redactions[name] != count {
				t.Errorf("Redactions[%q] = %d, want %d", name, preview.Redactions[name], count)
			}
		}
		if len(preview.Components) != len(want.Capsule.Fidelity.Components) {
			t.Fatalf("Components = %d, want %d", len(preview.Components), len(want.Capsule.Fidelity.Components))
		}
	})

	t.Run("sorts components by name", func(t *testing.T) {
		planner := NewPlanner(newFakePlanner().plan)
		run(t, planner.Compute(context.Background(), destCodex, string(handoff.PolicyBalanced)))
		preview, _ := planner.Lookup(destCodex, string(handoff.PolicyBalanced))

		names := make([]string, 0, len(preview.Components))
		for _, component := range preview.Components {
			names = append(names, component.Name)
		}
		want := "attachments,subagent_transcripts,tool_results,user_messages"
		if got := strings.Join(names, ","); got != want {
			t.Fatalf("component order = %q, want %q", got, want)
		}
	})

	t.Run("carries each component field across", func(t *testing.T) {
		planner := NewPlanner(newFakePlanner().plan)
		run(t, planner.Compute(context.Background(), destCodex, string(handoff.PolicyBalanced)))
		preview, _ := planner.Lookup(destCodex, string(handoff.PolicyBalanced))

		var attachments Component
		for _, component := range preview.Components {
			if component.Name == "attachments" {
				attachments = component
			}
		}
		want := Component{
			Name:        "attachments",
			Portability: string(capsule.PortabilityReferenced),
			Count:       2,
			Reason:      "left_in_place",
		}
		if attachments != want {
			t.Fatalf("attachments = %+v, want %+v", attachments, want)
		}
		if attachments.Included() {
			t.Error("a referenced component must not read as carried across")
		}
	})

	t.Run("an error leaves nothing but the error", func(t *testing.T) {
		fake := newFakePlanner()
		failure := errors.New("read transcript: permission denied")
		fake.fail = map[string]error{previewKey(destCodex, string(handoff.PolicyFull)): failure}
		planner := NewPlanner(fake.plan)

		run(t, planner.Compute(context.Background(), destCodex, string(handoff.PolicyFull)))
		preview, ready := planner.Lookup(destCodex, string(handoff.PolicyFull))
		if !ready {
			t.Fatal("a failed plan must still be cached, or the studio recomputes it forever")
		}
		if !errors.Is(preview.Err, failure) {
			t.Fatalf("Err = %v, want %v", preview.Err, failure)
		}
		want := Preview{Destination: destCodex, Policy: string(handoff.PolicyFull), Err: failure}
		if preview.Bytes != want.Bytes || preview.Tokens != want.Tokens || preview.Events != want.Events ||
			preview.Overall != "" || preview.Components != nil || preview.Redactions != nil ||
			preview.Warnings != nil {
			t.Fatalf("failed preview = %+v, want only the destination, policy and error", preview)
		}
	})

	t.Run("the preview does not alias the plan", func(t *testing.T) {
		fake := newFakePlanner()
		var captured handoff.PlanResult
		fake.mutate = func(_, _ string, result *handoff.PlanResult) { captured = *result }
		planner := NewPlanner(fake.plan)

		run(t, planner.Compute(context.Background(), destCodex, string(handoff.PolicyBalanced)))
		preview, _ := planner.Lookup(destCodex, string(handoff.PolicyBalanced))

		captured.RedactionCounts["api_key"] = 999
		captured.WarningIDs[0] = "rewritten"
		if preview.Redactions["api_key"] == 999 {
			t.Error("the cached preview shares the plan's redaction map")
		}
		if preview.Warnings[0] == "rewritten" {
			t.Error("the cached preview shares the plan's warning slice")
		}
	})
}

func TestRedactionReporting(t *testing.T) {
	t.Run("total sums every category", func(t *testing.T) {
		preview := Preview{Redactions: map[string]int{"api_key": 4, "bearer_token": 2, "aws_access_key_id": 1}}
		if got := preview.RedactionTotal(); got != 7 {
			t.Fatalf("RedactionTotal() = %d, want 7", got)
		}
	})

	t.Run("categories are sorted", func(t *testing.T) {
		preview := Preview{Redactions: map[string]int{"ssh_private_key": 1, "api_key": 4, "bearer_token": 2}}
		want := "api_key,bearer_token,ssh_private_key"
		if got := strings.Join(preview.RedactionCategories(), ","); got != want {
			t.Fatalf("RedactionCategories() = %q, want %q", got, want)
		}
	})

	t.Run("a nil map is safe", func(t *testing.T) {
		var preview Preview
		if got := preview.RedactionTotal(); got != 0 {
			t.Fatalf("RedactionTotal() = %d on a nil map, want 0", got)
		}
		if got := preview.RedactionCategories(); len(got) != 0 {
			t.Fatalf("RedactionCategories() = %v on a nil map, want none", got)
		}
	})

	t.Run("an empty map reports nothing", func(t *testing.T) {
		preview := Preview{Redactions: map[string]int{}}
		if preview.RedactionTotal() != 0 || len(preview.RedactionCategories()) != 0 {
			t.Fatalf("an empty map reported %d in %v",
				preview.RedactionTotal(), preview.RedactionCategories())
		}
	})
}

// TestComputeReturnsNilWhenThereIsNothingToDo covers the four ways the studio
// asks for work that must not be done twice. Every one of them is a real parse
// of the source transcript when it escapes.
func TestComputeReturnsNilWhenThereIsNothingToDo(t *testing.T) {
	t.Run("a nil planner", func(t *testing.T) {
		var planner *Planner
		if cmd := planner.Compute(context.Background(), destCodex, string(handoff.PolicyFull)); cmd != nil {
			t.Fatal("a nil planner returned a command")
		}
		if preview, ready := planner.Lookup(destCodex, string(handoff.PolicyFull)); ready || preview.Err != nil {
			t.Fatalf("a nil planner looked up %+v, want nothing", preview)
		}
	})

	t.Run("a planner with no plan function", func(t *testing.T) {
		planner := NewPlanner(nil)
		if cmd := planner.Compute(context.Background(), destCodex, string(handoff.PolicyFull)); cmd != nil {
			t.Fatal("a planner without a plan function returned a command")
		}
	})

	t.Run("an answer that is already cached", func(t *testing.T) {
		fake := newFakePlanner()
		planner := NewPlanner(fake.plan)
		run(t, planner.Compute(context.Background(), destCodex, string(handoff.PolicyFull)))

		if cmd := planner.Compute(context.Background(), destCodex, string(handoff.PolicyFull)); cmd != nil {
			t.Fatal("a cached pair was scheduled again")
		}
		if got := fake.callCount(destCodex, string(handoff.PolicyFull)); got != 1 {
			t.Fatalf("plan ran %d times, want once", got)
		}
	})

	t.Run("an answer that is already in flight", func(t *testing.T) {
		fake := newFakePlanner()
		planner := NewPlanner(fake.plan)

		first := planner.Compute(context.Background(), destCodex, string(handoff.PolicyBalanced))
		if first == nil {
			t.Fatal("the first request produced no command")
		}
		if second := planner.Compute(context.Background(), destCodex, string(handoff.PolicyBalanced)); second != nil {
			t.Fatal("a request already in flight was scheduled a second time")
		}
		if got := fake.totalCalls(); got != 0 {
			t.Fatalf("plan ran %d times before the command was executed", got)
		}
		run(t, first)
		if got := fake.callCount(destCodex, string(handoff.PolicyBalanced)); got != 1 {
			t.Fatalf("plan ran %d times, want once", got)
		}
	})
}

// TestComputeWithoutAContextDoesNotPanic guards the nil-context substitution.
// The studio hands its own context through, and a caller that never set one is
// a mistake to survive rather than a crash to ship.
func TestComputeWithoutAContextDoesNotPanic(t *testing.T) {
	fake := newFakePlanner()
	var seen context.Context
	planner := NewPlanner(func(ctx context.Context, destination, policy string) (handoff.PlanResult, error) {
		seen = ctx
		return fake.plan(ctx, destination, policy)
	})

	// A caller that never set a context, spelled as a variable so the check is
	// about the planner's substitution rather than about a nil literal.
	var missing context.Context
	run(t, planner.Compute(missing, destCodex, string(handoff.PolicyBalanced)))

	if seen == nil {
		t.Fatal("the plan function was handed a nil context")
	}
	if _, ready := planner.Lookup(destCodex, string(handoff.PolicyBalanced)); !ready {
		t.Fatal("the preview was not cached")
	}
}

// TestEachPairIsCachedSeparately is the reason the studio can be cycled freely:
// a preview for one policy must never overwrite another's.
func TestEachPairIsCachedSeparately(t *testing.T) {
	fake := newFakePlanner()
	planner := NewPlanner(fake.plan)

	pairs := []struct{ destination, policy string }{
		{destCodex, string(handoff.PolicyFull)},
		{destCodex, string(handoff.PolicyBalanced)},
		{destCodex, string(handoff.PolicyCheckpoint)},
		{destGemini, string(handoff.PolicyBalanced)},
	}
	for _, pair := range pairs {
		run(t, planner.Compute(context.Background(), pair.destination, pair.policy))
	}

	for _, pair := range pairs {
		preview, ready := planner.Lookup(pair.destination, pair.policy)
		if !ready {
			t.Fatalf("%s/%s is missing from the cache", pair.destination, pair.policy)
		}
		want := fixturePlan(pair.destination, pair.policy)
		if preview.Bytes != want.EstimatedBytes || preview.Events != len(want.Capsule.Conversation.Events) {
			t.Fatalf("%s/%s cached %d bytes and %d events, want %d and %d",
				pair.destination, pair.policy, preview.Bytes, preview.Events,
				want.EstimatedBytes, len(want.Capsule.Conversation.Events))
		}
	}

	// The two entries the studio flips between with a single key press.
	full, _ := planner.Lookup(destCodex, string(handoff.PolicyFull))
	balanced, _ := planner.Lookup(destCodex, string(handoff.PolicyBalanced))
	if full.Bytes == balanced.Bytes || full.Events == balanced.Events {
		t.Fatalf("full (%d bytes, %d events) and balanced (%d bytes, %d events) share a cache entry",
			full.Bytes, full.Events, balanced.Bytes, balanced.Events)
	}
	if full.Policy != string(handoff.PolicyFull) || balanced.Policy != string(handoff.PolicyBalanced) {
		t.Fatalf("cached policies are %q and %q, want full and balanced", full.Policy, balanced.Policy)
	}

	// Same policy, different destination: also a distinct entry.
	geminiBalanced, _ := planner.Lookup(destGemini, string(handoff.PolicyBalanced))
	if geminiBalanced.Bytes == balanced.Bytes {
		t.Fatalf("codex and gemini both cached %d bytes for balanced", balanced.Bytes)
	}
}

// TestConcurrentComputesRunEachPairExactlyOnce is the -race test. Bubble Tea
// runs every command on its own goroutine while the view keeps reading the
// cache, so the planner is genuinely concurrent in production.
func TestConcurrentComputesRunEachPairExactlyOnce(t *testing.T) {
	destinations := []string{
		destCodex, destGemini, destClaude, destOpenCode,
		"cursor", "copilot", "aider", "amp", "kimi", "qwen",
	}
	type pair struct{ destination, policy string }

	fake := newFakePlanner()
	planner := NewPlanner(fake.plan)

	var (
		pairs []pair
		cmds  []tea.Cmd
	)
	for _, destination := range destinations {
		for _, policy := range Policies {
			cmd := planner.Compute(context.Background(), destination, policy)
			if cmd == nil {
				t.Fatalf("%s/%s produced no command", destination, policy)
			}
			// Asking again while the first is in flight must be free.
			if again := planner.Compute(context.Background(), destination, policy); again != nil {
				t.Fatalf("%s/%s was scheduled twice", destination, policy)
			}
			pairs = append(pairs, pair{destination, policy})
			cmds = append(cmds, cmd)
		}
	}
	if len(pairs) != 30 {
		t.Fatalf("built %d pairs, want 30", len(pairs))
	}

	msgs := make([]tea.Msg, len(cmds))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for index, cmd := range cmds {
		wg.Add(1)
		go func(index int, cmd tea.Cmd) {
			defer wg.Done()
			<-start
			msgs[index] = cmd()
		}(index, cmd)
	}
	// Readers race the writers, the way the view reads the cache while
	// commands are still landing.
	for reader := 0; reader < 8; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for _, current := range pairs {
				planner.Lookup(current.destination, current.policy)
			}
		}()
	}
	close(start)
	wg.Wait()

	for index, current := range pairs {
		if got := fake.callCount(current.destination, current.policy); got != 1 {
			t.Errorf("%s/%s planned %d times, want exactly once", current.destination, current.policy, got)
		}
		previewed, ok := msgs[index].(PreviewedMsg)
		if !ok {
			t.Fatalf("%s/%s returned %T, want PreviewedMsg", current.destination, current.policy, msgs[index])
		}
		if previewed.Destination != current.destination || previewed.Policy != current.policy {
			t.Errorf("message %+v does not name %s/%s", previewed, current.destination, current.policy)
		}
		preview, ready := planner.Lookup(current.destination, current.policy)
		if !ready {
			t.Fatalf("%s/%s never reached the cache", current.destination, current.policy)
		}
		want := fixturePlan(current.destination, current.policy)
		if preview.Bytes != want.EstimatedBytes {
			t.Errorf("%s/%s cached %d bytes, want %d",
				current.destination, current.policy, preview.Bytes, want.EstimatedBytes)
		}
	}
	if got := fake.totalCalls(); got != len(pairs) {
		t.Fatalf("the plan function ran %d times for %d pairs", got, len(pairs))
	}

	// Every pair is now cached, so a second pass schedules nothing at all.
	for _, current := range pairs {
		if cmd := planner.Compute(context.Background(), current.destination, current.policy); cmd != nil {
			t.Fatalf("%s/%s was recomputed after it was cached", current.destination, current.policy)
		}
	}
}
