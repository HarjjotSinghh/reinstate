package capsule

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestValidateMinimalValid(t *testing.T) {
	t.Parallel()
	c := sampleCapsule()
	if err := Validate(c); err != nil {
		t.Fatalf("Validate(sampleCapsule) = %v", err)
	}
}

func TestValidateRejections(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		edit    func(*Capsule)
		wantSub string
	}{
		{
			name:    "missing_schema",
			edit:    func(c *Capsule) { c.Schema = "" },
			wantSub: "schema",
		},
		{
			name:    "wrong_schema",
			edit:    func(c *Capsule) { c.Schema = "other/v1" },
			wantSub: "schema",
		},
		{
			name: "event_missing_portability",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Portability = ""
			},
			wantSub: "missing portability",
		},
		{
			name: "non_exact_without_reason",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Portability = PortabilityNormalized
				c.Conversation.Events[0].Reason = ""
			},
			wantSub: "requires reason",
		},
		{
			name: "tool_result_unmatched_linked_call",
			edit: func(c *Capsule) {
				c.Conversation.Events = append(c.Conversation.Events, Event{
					ID:           "result-1",
					Order:        1,
					Actor:        ActorTool,
					Kind:         KindToolResult,
					LinkedCallID: "missing-call",
					Portability:  PortabilityNormalized,
					Reason:       "normalized_tool_result",
					ContentHash:  "r1",
					Source:       SourcePointer{Agent: "claude", SessionID: "sess-1", Index: 1},
				})
			},
			wantSub: "no matching tool_call",
		},
		{
			name: "duplicate_event_ids",
			edit: func(c *Capsule) {
				dup := c.Conversation.Events[0]
				dup.Order = 1
				c.Conversation.Events = append(c.Conversation.Events, dup)
			},
			wantSub: "duplicate event id",
		},
		{
			name: "out_of_order",
			edit: func(c *Capsule) {
				c.Conversation.Events = []Event{
					eventWith(c.Conversation.Events[0], "a", 5),
					eventWith(c.Conversation.Events[0], "b", 1),
				}
			},
			wantSub: "out of order",
		},
		{
			name: "max_events",
			edit: func(c *Capsule) {
				base := c.Conversation.Events[0]
				events := make([]Event, MaxEvents+1)
				for i := range events {
					events[i] = eventWith(base, fmt.Sprintf("e-%d", i), i)
				}
				c.Conversation.Events = events
			},
			wantSub: "MaxEvents",
		},
		{
			name: "max_blocks_per_event",
			edit: func(c *Capsule) {
				blocks := make([]Block, MaxBlocksPerEvent+1)
				for i := range blocks {
					blocks[i] = Block{Type: BlockTypeText, Text: "x"}
				}
				c.Conversation.Events[0].Blocks = blocks
			},
			wantSub: "MaxBlocksPerEvent",
		},
		{
			name: "max_text_block_bytes",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks = []Block{{
					Type: BlockTypeText,
					Text: strings.Repeat("a", MaxTextBlockBytes+1),
				}}
			},
			wantSub: "MaxTextBlockBytes",
		},
		{
			name: "max_task_field_runes",
			edit: func(c *Capsule) {
				c.Task.Goal.Text = strings.Repeat("字", MaxTaskFieldRunes+1)
			},
			wantSub: "MaxTaskFieldRunes",
		},
		{
			name: "max_file_references",
			edit: func(c *Capsule) {
				files := make([]string, MaxFileReferences+1)
				for i := range files {
					files[i] = fmt.Sprintf("${REPO:github.com/example/demo}/f%d.go", i)
				}
				c.Workspace.ChangedFiles = files
				c.Workspace.Tests = nil
				c.Task.ChangedFiles.Items = nil
				c.Task.FilesTouchedPerTranscript.Items = nil
			},
			wantSub: "MaxFileReferences",
		},
		{
			name: "max_redactions_per_event",
			edit: func(c *Capsule) {
				reds := make([]Redaction, MaxRedactionsPerEvent+1)
				for i := range reds {
					reds[i] = Redaction{Category: CategoryHighEntropy, Digest: fmt.Sprintf("%012x", i)}
				}
				c.Conversation.Events[0].Redactions = reds
			},
			wantSub: "MaxRedactionsPerEvent",
		},
		{
			name: "max_capsule_bytes",
			edit: func(c *Capsule) {
				// Stay under MaxEvents / MaxTextBlockBytes while exceeding MaxCapsuleBytes.
				const chunk = 200 << 10 // 200 KiB
				n := (MaxCapsuleBytes / chunk) + 2
				base := c.Conversation.Events[0]
				events := make([]Event, n)
				text := strings.Repeat("b", chunk)
				for i := range events {
					ev := eventWith(base, fmt.Sprintf("big-%d", i), i)
					ev.Blocks = []Block{{Type: BlockTypeText, Text: text}}
					events[i] = ev
				}
				c.Conversation.Events = events
			},
			wantSub: "MaxCapsuleBytes",
		},
		{
			name: "fidelity_overall_inconsistent",
			edit: func(c *Capsule) {
				c.Fidelity.Overall = PortabilityExact // components include normalized
			},
			wantSub: "inconsistent with components",
		},
		{
			name: "mode_not_structured_handoff",
			edit: func(c *Capsule) {
				c.Fidelity.Mode = FidelityModeReconstructedConversation
			},
			wantSub: "fidelity mode",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := sampleCapsule()
			tc.edit(&c)
			err := Validate(c)
			if err == nil {
				t.Fatal("Validate succeeded; want error")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Validate error %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidateToolResultWithMatchingCall(t *testing.T) {
	t.Parallel()
	c := sampleCapsule()
	call := Event{
		ID:          "call-1",
		Order:       1,
		Actor:       ActorAssistant,
		Kind:        KindToolCall,
		CallID:      "call-abc",
		Portability: PortabilityNormalized,
		Reason:      "normalized_tool_call",
		ContentHash: "c1",
		Source:      SourcePointer{Agent: "claude", SessionID: "sess-1", Index: 1},
	}
	result := Event{
		ID:           "result-1",
		Order:        2,
		Actor:        ActorTool,
		Kind:         KindToolResult,
		LinkedCallID: "call-abc",
		Portability:  PortabilityNormalized,
		Reason:       "normalized_tool_result",
		ContentHash:  "r1",
		Source:       SourcePointer{Agent: "claude", SessionID: "sess-1", Index: 2},
	}
	c.Conversation.Events = append(c.Conversation.Events, call, result)
	// Overall stays normalized (already).
	if err := Validate(c); err != nil {
		t.Fatalf("Validate with matched tool_result: %v", err)
	}
}

func eventWith(base Event, id string, order int) Event {
	e := base
	e.ID = id
	e.Order = order
	e.Timestamp = time.Time{}
	return e
}
