package capsule

import (
	"fmt"
	"unicode/utf8"
)

// Hard maxima for continuity-capsule v1. Validation rejects rather than truncating.
const (
	MaxEvents             = 20000
	MaxBlocksPerEvent     = 256
	MaxTextBlockBytes     = 256 << 10
	MaxCapsuleBytes       = 8 << 20
	MaxTaskFieldRunes     = 8192
	MaxFileReferences     = 512
	MaxRedactionsPerEvent = 256
)

// Validate enforces the complete v1 contract and returns the first violation.
//
// It rejects a missing or wrong Schema, Mode other than structured_handoff, a
// Fidelity.Overall inconsistent with the component set, events missing
// Portability, non-exact events without Reason, tool_result events whose
// LinkedCallID has no matching tool_call, duplicate event IDs, out-of-order
// Order values, and any exceeded bound.
func Validate(c Capsule) error {
	if c.Schema != Schema {
		return fmt.Errorf("capsule: schema %q, want %q", c.Schema, Schema)
	}
	if c.Fidelity.Mode != FidelityModeStructuredHandoff {
		return fmt.Errorf("capsule: fidelity mode %q, want %q", c.Fidelity.Mode, FidelityModeStructuredHandoff)
	}
	for _, comp := range c.Fidelity.Components {
		if !ValidPortability(comp.Portability) {
			return fmt.Errorf("capsule: fidelity component %q has invalid portability %q", comp.Name, comp.Portability)
		}
	}
	wantOverall := worstOfComponents(c.Fidelity.Components)
	if c.Fidelity.Overall != wantOverall {
		return fmt.Errorf("capsule: fidelity overall %q inconsistent with components (want %q)", c.Fidelity.Overall, wantOverall)
	}

	events := c.Conversation.Events
	if len(events) > MaxEvents {
		return fmt.Errorf("capsule: %d events exceeds MaxEvents (%d)", len(events), MaxEvents)
	}

	if err := validateTaskBounds(c.Task); err != nil {
		return err
	}
	if n := countFileReferences(c); n > MaxFileReferences {
		return fmt.Errorf("capsule: %d file references exceeds MaxFileReferences (%d)", n, MaxFileReferences)
	}

	callIDs := make(map[string]struct{}, len(events))
	for _, e := range events {
		if e.Kind == KindToolCall && e.CallID != "" {
			callIDs[e.CallID] = struct{}{}
		}
	}

	seen := make(map[string]struct{}, len(events))
	for i, e := range events {
		if e.Portability == "" || !ValidPortability(e.Portability) {
			return fmt.Errorf("capsule: event %q missing portability", e.ID)
		}
		if e.Portability != PortabilityExact && e.Reason == "" {
			return fmt.Errorf("capsule: event %q portability %q requires reason", e.ID, e.Portability)
		}
		if e.ID == "" {
			return fmt.Errorf("capsule: event at order %d has empty id", e.Order)
		}
		if _, dup := seen[e.ID]; dup {
			return fmt.Errorf("capsule: duplicate event id %q", e.ID)
		}
		seen[e.ID] = struct{}{}

		if i > 0 && e.Order < events[i-1].Order {
			return fmt.Errorf("capsule: event order out of order at index %d (%d < %d)", i, e.Order, events[i-1].Order)
		}

		if len(e.Blocks) > MaxBlocksPerEvent {
			return fmt.Errorf("capsule: event %q has %d blocks, MaxBlocksPerEvent is %d", e.ID, len(e.Blocks), MaxBlocksPerEvent)
		}
		if len(e.Redactions) > MaxRedactionsPerEvent {
			return fmt.Errorf("capsule: event %q has %d redactions, MaxRedactionsPerEvent is %d", e.ID, len(e.Redactions), MaxRedactionsPerEvent)
		}
		for j, b := range e.Blocks {
			if len(b.Text) > MaxTextBlockBytes {
				return fmt.Errorf("capsule: event %q block %d text is %d bytes, MaxTextBlockBytes is %d", e.ID, j, len(b.Text), MaxTextBlockBytes)
			}
		}

		if e.Kind == KindToolResult {
			if e.LinkedCallID == "" {
				return fmt.Errorf("capsule: tool_result event %q has empty linked_call_id", e.ID)
			}
			if _, ok := callIDs[e.LinkedCallID]; !ok {
				return fmt.Errorf("capsule: tool_result event %q linked_call_id %q has no matching tool_call", e.ID, e.LinkedCallID)
			}
		}
	}

	raw, err := CanonicalBytes(c)
	if err != nil {
		return fmt.Errorf("capsule: canonical encode: %w", err)
	}
	if len(raw) > MaxCapsuleBytes {
		return fmt.Errorf("capsule: canonical size %d exceeds MaxCapsuleBytes (%d)", len(raw), MaxCapsuleBytes)
	}
	return nil
}

func validateTaskBounds(t Task) error {
	checkText := func(name string, f TextField) error {
		if utf8.RuneCountInString(f.Text) > MaxTaskFieldRunes {
			return fmt.Errorf("capsule: task.%s exceeds MaxTaskFieldRunes (%d)", name, MaxTaskFieldRunes)
		}
		return nil
	}
	checkList := func(name string, f ListField) error {
		for i, item := range f.Items {
			if utf8.RuneCountInString(item) > MaxTaskFieldRunes {
				return fmt.Errorf("capsule: task.%s[%d] exceeds MaxTaskFieldRunes (%d)", name, i, MaxTaskFieldRunes)
			}
		}
		return nil
	}

	if err := checkText("goal", t.Goal); err != nil {
		return err
	}
	if err := checkText("latest_user_intent", t.LatestUserIntent); err != nil {
		return err
	}
	if err := checkText("next_action", t.NextAction); err != nil {
		return err
	}
	lists := []struct {
		name string
		f    ListField
	}{
		{"recent_user_messages", t.RecentUserMessages},
		{"constraints", t.Constraints},
		{"decisions", t.Decisions},
		{"rejected_approaches", t.RejectedApproaches},
		{"completed", t.Completed},
		{"pending", t.Pending},
		{"changed_files", t.ChangedFiles},
		{"files_touched_per_transcript", t.FilesTouchedPerTranscript},
		{"tests", t.Tests},
		{"open_questions", t.OpenQuestions},
	}
	for _, l := range lists {
		if err := checkList(l.name, l.f); err != nil {
			return err
		}
	}
	return nil
}

func countFileReferences(c Capsule) int {
	n := len(c.Workspace.ChangedFiles) + len(c.Workspace.Tests)
	n += len(c.Task.ChangedFiles.Items)
	n += len(c.Task.FilesTouchedPerTranscript.Items)
	for _, e := range c.Conversation.Events {
		for _, b := range e.Blocks {
			if b.Path != "" || b.Type == BlockTypeAttachment || b.Type == BlockTypeRef {
				n++
			}
		}
	}
	return n
}
