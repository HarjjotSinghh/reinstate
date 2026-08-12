package capsule

import (
	"testing"
)

func TestAggregateFidelityWorstWinsAllCombinations(t *testing.T) {
	t.Parallel()

	all := []Portability{
		PortabilityExact,
		PortabilityNormalized,
		PortabilitySummarized,
		PortabilityReferenced,
		PortabilityOmitted,
	}

	for _, a := range all {
		for _, b := range all {
			a, b := a, b
			t.Run(string(a)+"+"+string(b), func(t *testing.T) {
				t.Parallel()
				events := []Event{
					{
						ID:          "u1",
						Order:       0,
						Actor:       ActorUser,
						Kind:        KindMessage,
						Portability: a,
						Reason:      reasonFor(a),
						Blocks:      []Block{{Type: BlockTypeText, Text: "hi"}},
					},
					{
						ID:          "t1",
						Order:       1,
						Actor:       ActorTool,
						Kind:        KindToolResult,
						Portability: b,
						Reason:      reasonFor(b),
						Blocks:      []Block{{Type: BlockTypeText, Text: "out"}},
					},
				}
				got := AggregateFidelity(events, nil)
				want := WorstPortability(a, b)
				if got.Overall != want {
					t.Fatalf("Overall = %q, want %q", got.Overall, want)
				}
				if got.Mode != FidelityModeStructuredHandoff {
					t.Fatalf("Mode = %q, want %q", got.Mode, FidelityModeStructuredHandoff)
				}
				if len(got.Components) != 2 {
					t.Fatalf("Components len = %d, want 2", len(got.Components))
				}
				byName := map[string]Component{}
				for _, c := range got.Components {
					byName[c.Name] = c
				}
				if byName["user_messages"].Portability != a {
					t.Fatalf("user_messages = %q, want %q", byName["user_messages"].Portability, a)
				}
				if byName["tool_results"].Portability != b {
					t.Fatalf("tool_results = %q, want %q", byName["tool_results"].Portability, b)
				}
			})
		}
	}
}

func TestAggregateFidelityMergesIncludedTaskComponents(t *testing.T) {
	t.Parallel()

	events := []Event{{
		ID:          "u1",
		Actor:       ActorUser,
		Kind:        KindMessage,
		Portability: PortabilityExact,
		Blocks:      []Block{{Type: BlockTypeText, Text: "go"}},
	}}
	included := Components{
		{Name: "constraints", Portability: PortabilityOmitted, Count: 1, Reason: "requires_optional_summarizer"},
		{Name: "goal", Portability: PortabilityNormalized, Count: 1, Bytes: 4},
	}
	got := AggregateFidelity(events, included)
	if got.Overall != PortabilityOmitted {
		t.Fatalf("Overall = %q, want omitted (task field wins)", got.Overall)
	}
	found := false
	for _, c := range got.Components {
		if c.Name == "constraints" && c.Portability == PortabilityOmitted {
			found = true
		}
	}
	if !found {
		t.Fatalf("constraints component missing: %+v", got.Components)
	}
}

func TestAggregateFidelityEmptyIsExact(t *testing.T) {
	t.Parallel()
	got := AggregateFidelity(nil, nil)
	if got.Overall != PortabilityExact {
		t.Fatalf("Overall = %q, want exact", got.Overall)
	}
	if got.Mode != FidelityModeStructuredHandoff {
		t.Fatalf("Mode = %q", got.Mode)
	}
	if len(got.Components) != 0 {
		t.Fatalf("Components = %+v, want empty", got.Components)
	}
}

func TestWorstPortabilityOrder(t *testing.T) {
	t.Parallel()
	order := []Portability{
		PortabilityExact,
		PortabilityNormalized,
		PortabilitySummarized,
		PortabilityReferenced,
		PortabilityOmitted,
	}
	for i := 0; i < len(order); i++ {
		for j := 0; j < len(order); j++ {
			got := WorstPortability(order[i], order[j])
			want := order[i]
			if j > i {
				want = order[j]
			}
			if got != want {
				t.Fatalf("WorstPortability(%q,%q)=%q want %q", order[i], order[j], got, want)
			}
		}
	}
}

func reasonFor(p Portability) string {
	if p == PortabilityExact {
		return ""
	}
	return "test_" + string(p)
}
