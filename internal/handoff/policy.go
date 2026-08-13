package handoff

import (
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

// Policy selects how much conversation history enters the destination projection.
type Policy string

const (
	// PolicyCheckpoint includes no verbatim conversation events.
	PolicyCheckpoint Policy = "checkpoint"
	// PolicyBalanced is the default: newest events within the projection budget.
	PolicyBalanced Policy = "balanced"
	// PolicyFull includes portable visible events up to the hard projection cap.
	PolicyFull Policy = "full"
)

const (
	// DefaultProjectionBudgetBytes is the prompt-bound budget for balanced.
	DefaultProjectionBudgetBytes = 64 << 10
	// HardProjectionCapBytes is the absolute ceiling for full projections.
	HardProjectionCapBytes = 2 << 20
	// BootstrapMaxBytes is the argv bootstrap prompt ceiling.
	BootstrapMaxBytes = 8 << 10
)

const (
	reasonProjectionBudget    = "projection_budget"
	reasonProjectionTruncated = "projection_truncated"
	reasonAlreadyReferenced   = "already_referenced"
	reasonOmittedNotProjected = "omitted_not_projected"
)

// Apply selects included events under p and produces sidecar references for the
// rest. Selection is newest-first within the policy budget, then re-sorted into
// source order. Excluded events are always referenced — never silently dropped.
//
// Empty or unknown policies default to PolicyBalanced.
func Apply(p Policy, events []capsule.Event) (included []capsule.Event, sidecar []capsule.SidecarRef, report capsule.Fidelity) {
	p = normalizePolicy(p)
	budget := budgetBytes(p)

	chosen := make(map[int]capsule.Event, len(events))
	used := 0

	if budget > 0 {
		for i := len(events) - 1; i >= 0; i-- {
			e := events[i]
			if !projectionEligible(e) {
				continue
			}
			n := eventTextBytes(e)
			if used+n <= budget {
				chosen[i] = cloneEvent(e)
				used += n
				continue
			}
			remaining := budget - used
			if remaining <= len(transcript.TruncationMarker) {
				continue
			}
			truncated := truncateEventTo(e, remaining)
			chosen[i] = truncated
			// Remaining budget is filled exactly by the truncated marginal event.
			break
		}
	}

	included = make([]capsule.Event, 0, len(chosen))
	classified := make([]capsule.Event, 0, len(events))
	sidecar = make([]capsule.SidecarRef, 0, len(events)-len(chosen))

	for i, e := range events {
		if ev, ok := chosen[i]; ok {
			included = append(included, ev)
			classified = append(classified, ev)
			continue
		}
		ref := sidecarRefFor(e)
		sidecar = append(sidecar, ref)
		if e.Portability == capsule.PortabilityOmitted {
			classified = append(classified, cloneEvent(e))
			continue
		}
		if e.Portability == capsule.PortabilitySummarized || e.Kind == capsule.KindSummary {
			classified = append(classified, cloneEvent(e))
			continue
		}
		classified = append(classified, referencedCopy(e, ref.Reason))
	}

	report = capsule.AggregateFidelity(classified, nil)
	return included, sidecar, report
}

func normalizePolicy(p Policy) Policy {
	switch p {
	case PolicyCheckpoint, PolicyBalanced, PolicyFull:
		return p
	default:
		return PolicyBalanced
	}
}

func budgetBytes(p Policy) int {
	switch p {
	case PolicyCheckpoint:
		return 0
	case PolicyFull:
		return HardProjectionCapBytes
	default:
		return DefaultProjectionBudgetBytes
	}
}

func projectionEligible(e capsule.Event) bool {
	switch e.Portability {
	case capsule.PortabilityOmitted, capsule.PortabilityReferenced:
		return false
	default:
		return true
	}
}

func sidecarRefFor(e capsule.Event) capsule.SidecarRef {
	reason := reasonProjectionBudget
	switch e.Portability {
	case capsule.PortabilityReferenced:
		reason = reasonAlreadyReferenced
		if e.Reason != "" {
			reason = e.Reason
		}
	case capsule.PortabilityOmitted:
		reason = reasonOmittedNotProjected
		if e.Reason != "" {
			reason = e.Reason
		}
	}
	return capsule.SidecarRef{
		EventID:     e.ID,
		ContentHash: e.ContentHash,
		Bytes:       int64(eventTextBytes(e)),
		Portability: capsule.PortabilityReferenced,
		Reason:      reason,
	}
}

func referencedCopy(e capsule.Event, reason string) capsule.Event {
	out := cloneEvent(e)
	out.Portability = capsule.PortabilityReferenced
	if reason != "" {
		out.Reason = reason
	}
	return out
}

func eventTextBytes(e capsule.Event) int {
	n := 0
	for _, b := range e.Blocks {
		n += len(b.Text)
	}
	return n
}

func cloneEvent(e capsule.Event) capsule.Event {
	out := e
	if e.Blocks != nil {
		out.Blocks = make([]capsule.Block, len(e.Blocks))
		copy(out.Blocks, e.Blocks)
	}
	if e.Redactions != nil {
		out.Redactions = make([]capsule.Redaction, len(e.Redactions))
		copy(out.Redactions, e.Redactions)
	}
	return out
}

func truncateEventTo(e capsule.Event, maxBytes int) capsule.Event {
	out := cloneEvent(e)
	if maxBytes < 0 {
		maxBytes = 0
	}
	if eventTextBytes(out) <= maxBytes {
		return out
	}

	cur := eventTextBytes(out)
	for i := len(out.Blocks) - 1; i >= 0 && cur > maxBytes; i-- {
		blockLen := len(out.Blocks[i].Text)
		if blockLen == 0 {
			continue
		}
		other := cur - blockLen
		thisMax := maxBytes - other
		if thisMax < 0 {
			thisMax = 0
		}
		out.Blocks[i] = transcript.TruncateBlock(out.Blocks[i], thisMax)
		cur = other + len(out.Blocks[i].Text)
	}

	out.Truncated = true
	out.Portability = capsule.WorstPortability(out.Portability, capsule.PortabilitySummarized)
	if out.Reason == "" {
		out.Reason = reasonProjectionTruncated
	}
	return out
}

func taskFidelityComponents(task capsule.Task) capsule.Components {
	var out capsule.Components
	addText := func(name string, field capsule.TextField) {
		if field.Portability == "" {
			return
		}
		out = append(out, capsule.Component{
			Name: name, Portability: field.Portability, Reason: field.Reason,
			Count: 1, Bytes: int64(len(field.Text)),
		})
	}
	addList := func(name string, field capsule.ListField) {
		if field.Portability == "" {
			return
		}
		count := len(field.Items)
		if count == 0 {
			count = 1
		}
		bytes := 0
		for _, item := range field.Items {
			bytes += len(item)
		}
		out = append(out, capsule.Component{
			Name: name, Portability: field.Portability, Reason: field.Reason,
			Count: count, Bytes: int64(bytes),
		})
	}
	addText("goal", task.Goal)
	addText("latest_user_intent", task.LatestUserIntent)
	addList("recent_user_messages", task.RecentUserMessages)
	addList("constraints", task.Constraints)
	addList("decisions", task.Decisions)
	addList("rejected_approaches", task.RejectedApproaches)
	addList("completed", task.Completed)
	addList("pending", task.Pending)
	addList("changed_files", task.ChangedFiles)
	addList("files_touched_per_transcript", task.FilesTouchedPerTranscript)
	addList("tests", task.Tests)
	addText("next_action", task.NextAction)
	addList("open_questions", task.OpenQuestions)
	return out
}
