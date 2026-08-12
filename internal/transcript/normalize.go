package transcript

import (
	"strings"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
)

// TruncationMarker is appended whenever TruncateBlock shortens block text.
// The marker is intentionally distinctive so it cannot be mistaken for vendor
// transcript content.
const TruncationMarker = "\n[truncated]"

// NormalizeActor maps a vendor role/speaker string onto a capsule.Actor.
// Unrecognized values become ActorUnknown.
func NormalizeActor(raw string) capsule.Actor {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "user", "human":
		return capsule.ActorUser
	case "assistant", "model", "ai", "gemini":
		return capsule.ActorAssistant
	case "tool", "function":
		return capsule.ActorTool
	case "system", "developer", "harness", "meta":
		return capsule.ActorHarness
	default:
		return capsule.ActorUnknown
	}
}

// NormalizeKind maps a vendor record/event type onto a capsule.Kind.
// Unrecognized values become KindUnknown.
func NormalizeKind(raw string) capsule.Kind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "message", "user_message", "agent_message", "assistant_message":
		return capsule.KindMessage
	case "tool_call", "tool_use", "function_call":
		return capsule.KindToolCall
	case "tool_result", "tool_output", "function_output", "function_call_output":
		return capsule.KindToolResult
	case "attachment", "image", "file":
		return capsule.KindAttachment
	case "summary", "compaction":
		return capsule.KindSummary
	case "checkpoint":
		return capsule.KindCheckpoint
	case "metadata", "session_meta", "system", "developer":
		return capsule.KindMetadata
	default:
		return capsule.KindUnknown
	}
}

// TextBlock builds a text content block.
func TextBlock(text string) capsule.Block {
	return capsule.Block{
		Type: capsule.BlockTypeText,
		Text: text,
		Size: int64(len(text)),
	}
}

// RefBlock builds a reference content block (sidecar or external pointer).
func RefBlock(ref string) capsule.Block {
	return capsule.Block{
		Type: capsule.BlockTypeRef,
		Ref:  ref,
	}
}

// LinkToolResults fills empty LinkedCallID on tool_result events from unmatched
// preceding tool_call CallIDs in FIFO order. Events that already set
// LinkedCallID are left unchanged. Returns a shallow-copied slice.
func LinkToolResults(events []capsule.Event) []capsule.Event {
	if len(events) == 0 {
		return events
	}
	out := make([]capsule.Event, len(events))
	copy(out, events)

	pending := make([]string, 0)
	for i := range out {
		switch out[i].Kind {
		case capsule.KindToolCall:
			if out[i].CallID != "" {
				pending = append(pending, out[i].CallID)
			}
		case capsule.KindToolResult:
			if out[i].LinkedCallID != "" {
				// Consume a matching pending id when present; otherwise keep as-is.
				for j, id := range pending {
					if id == out[i].LinkedCallID {
						pending = append(pending[:j], pending[j+1:]...)
						break
					}
				}
				continue
			}
			if len(pending) == 0 {
				continue
			}
			out[i].LinkedCallID = pending[0]
			pending = pending[1:]
		}
	}
	return out
}

// ClassifyUnknown assigns the canonical unknown classification for an
// unrecognized native record type.
func ClassifyUnknown(nativeType string) (capsule.Actor, capsule.Kind, capsule.Portability, string) {
	_ = nativeType
	return capsule.ActorUnknown, capsule.KindUnknown, capsule.PortabilityReferenced, "unrecognized_record_type"
}

// TruncateBlock shortens block text to at most maxBytes, always appending
// TruncationMarker when truncation occurs. maxBytes must be large enough to
// hold the marker; otherwise the result is only the marker (still visible).
func TruncateBlock(b capsule.Block, maxBytes int) capsule.Block {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if len(b.Text) <= maxBytes {
		return b
	}

	marker := TruncationMarker
	if maxBytes < len(marker) {
		b.Text = marker[:maxBytes]
		b.Size = int64(len(b.Text))
		return b
	}

	keep := maxBytes - len(marker)
	text := b.Text[:keep]
	for !utf8.ValidString(text) && len(text) > 0 {
		text = text[:len(text)-1]
	}
	b.Text = text + marker
	b.Size = int64(len(b.Text))
	return b
}
