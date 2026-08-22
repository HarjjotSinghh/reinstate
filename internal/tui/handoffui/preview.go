// Package handoffui is the interactive handoff studio.
//
// A structured handoff is governed by a projection policy — checkpoint,
// balanced, or full — and by the choice of destination agent. Both are
// consequential and, on the flag path, both are invisible: nothing tells you
// what `--policy full` will actually carry across until after you have run it.
//
// The studio makes that trade-off visible. Changing the policy or the
// destination recomputes a real dry-run plan and shows what the capsule would
// contain, how large it is, and what was redacted, before anything is written.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package handoffui

import (
	"context"
	"sort"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
)

// Component is one part of the capsule, flattened for display.
type Component struct {
	Name        string
	Portability string
	Count       int
	Bytes       int64
	Reason      string
}

// Included reports whether the component survives into the destination
// briefing. Referenced and omitted components are named in the capsule but
// their content does not travel, which is precisely what a reader choosing a
// policy needs to see.
func (c Component) Included() bool {
	switch capsule.Portability(c.Portability) {
	case capsule.PortabilityExact, capsule.PortabilityNormalized, capsule.PortabilitySummarized:
		return true
	default:
		return false
	}
}

// Preview is a policy-and-destination combination, flattened so the view never
// touches capsule or handoff types.
type Preview struct {
	Destination string
	Policy      string
	Bytes       int64
	Tokens      int
	Events      int
	Overall     string
	Components  []Component
	Redactions  map[string]int
	Warnings    []string
	Err         error
}

// RedactionTotal is the number of values hidden, across every category. The
// studio reports counts and category names only; it never shows a redacted
// value, which is the whole point of redacting it.
func (p Preview) RedactionTotal() int {
	total := 0
	for _, count := range p.Redactions {
		total += count
	}
	return total
}

// RedactionCategories returns the category names in a stable order.
func (p Preview) RedactionCategories() []string {
	names := make([]string, 0, len(p.Redactions))
	for name := range p.Redactions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PlanFunc produces a dry-run plan for one destination and policy. It is the
// only engine dependency, injected so the studio is testable without a
// transcript, a vendor, or a filesystem.
type PlanFunc func(ctx context.Context, destination, policy string) (handoff.PlanResult, error)

// PreviewedMsg reports that one preview finished and the cache is worth reading
// again.
type PreviewedMsg struct {
	Destination string
	Policy      string
}

// Planner computes and caches previews.
//
// Every distinct destination-and-policy pair is a real parse of the source
// transcript, so answers are cached and never recomputed. Cycling back and
// forth through the policies costs the user nothing after the first pass.
type Planner struct {
	plan PlanFunc

	mu      sync.Mutex
	cache   map[string]Preview
	pending map[string]struct{}
}

// NewPlanner builds a planner.
func NewPlanner(plan PlanFunc) *Planner {
	return &Planner{
		plan:    plan,
		cache:   make(map[string]Preview),
		pending: make(map[string]struct{}),
	}
}

func previewKey(destination, policy string) string { return destination + "\x00" + policy }

// Lookup returns a cached preview without blocking.
func (p *Planner) Lookup(destination, policy string) (Preview, bool) {
	if p == nil {
		return Preview{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	preview, ok := p.cache[previewKey(destination, policy)]
	return preview, ok
}

// Compute returns a command that builds one preview, or nil when the answer is
// already known or already in flight.
func (p *Planner) Compute(ctx context.Context, destination, policy string) tea.Cmd {
	if p == nil || p.plan == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	key := previewKey(destination, policy)
	p.mu.Lock()
	_, cached := p.cache[key]
	_, inFlight := p.pending[key]
	if cached || inFlight {
		p.mu.Unlock()
		return nil
	}
	p.pending[key] = struct{}{}
	p.mu.Unlock()

	return func() tea.Msg {
		result, err := p.plan(ctx, destination, policy)
		preview := previewFromPlan(destination, policy, result, err)

		p.mu.Lock()
		p.cache[key] = preview
		delete(p.pending, key)
		p.mu.Unlock()

		return PreviewedMsg{Destination: destination, Policy: policy}
	}
}

func previewFromPlan(destination, policy string, result handoff.PlanResult, err error) Preview {
	preview := Preview{Destination: destination, Policy: policy, Err: err}
	if err != nil {
		return preview
	}
	preview.Bytes = result.EstimatedBytes
	preview.Tokens = result.EstimatedTokens
	preview.Events = len(result.Capsule.Conversation.Events)
	preview.Overall = string(result.Capsule.Fidelity.Overall)
	preview.Warnings = append([]string(nil), result.WarningIDs...)
	if len(result.RedactionCounts) > 0 {
		preview.Redactions = make(map[string]int, len(result.RedactionCounts))
		for name, count := range result.RedactionCounts {
			preview.Redactions[name] = count
		}
	}
	for _, component := range result.Capsule.Fidelity.Components {
		preview.Components = append(preview.Components, Component{
			Name:        component.Name,
			Portability: string(component.Portability),
			Count:       component.Count,
			Bytes:       component.Bytes,
			Reason:      component.Reason,
		})
	}
	sort.SliceStable(preview.Components, func(i, j int) bool {
		return preview.Components[i].Name < preview.Components[j].Name
	})
	return preview
}
