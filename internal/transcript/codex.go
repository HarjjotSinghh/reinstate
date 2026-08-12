package transcript

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func init() {
	_ = Register(&CodexReader{})
}

// CodexReader converts Codex CLI rollout JSONL into canonical capsule events.
type CodexReader struct{}

// Name returns the stable agent key "codex".
func (r *CodexReader) Name() string { return sessionindex.AgentCodex }

// Probe reports whether the record looks like a Codex rollout JSONL session.
func (r *CodexReader) Probe(_ context.Context, rec sessionindex.Record) (Compatibility, error) {
	if rec.Agent != "" && !strings.EqualFold(rec.Agent, sessionindex.AgentCodex) {
		return CompatibilityUnsupported, nil
	}
	path := strings.TrimSpace(rec.SourcePath)
	if path == "" {
		return CompatibilityUnsupported, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".jsonl") {
		return CompatibilityUnsupported, nil
	}
	return CompatibilitySupported, nil
}

// Snapshot freezes the last complete JSONL record boundary. Session identity
// comes from the filename UUID when present (codexSessionIDFromFilename
// semantics), never from in-file session_meta IDs, so forks stay addressable.
func (r *CodexReader) Snapshot(_ context.Context, rec sessionindex.Record) (Boundary, error) {
	path := strings.TrimSpace(rec.SourcePath)
	if path == "" {
		return Boundary{}, fmt.Errorf("transcript: codex snapshot requires SourcePath")
	}
	sessionID := codexSessionIDFromFilename(path)
	if sessionID == "" {
		sessionID = strings.TrimSpace(rec.ID)
	}
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return SnapshotJSONL(path, sessionindex.AgentCodex, sessionID, MaxJSONLineBytes)
}

// Parse converts a frozen Codex rollout boundary into canonical events.
//
// When both event_msg and response_item representations of the same turn class
// exist, event_msg wins and the duplicate response_item message records are
// dropped — mirroring internal/sessionindex/codex.go prompt preference.
// Reasoning and encrypted reasoning items are emitted as omitted events with
// reason vendor_opaque_state and never carry payload bodies.
func (r *CodexReader) Parse(_ context.Context, b Boundary) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	if b.Agent != "" && !strings.EqualFold(b.Agent, sessionindex.AgentCodex) {
		return nil, report, fmt.Errorf("transcript: codex parse got agent %q", b.Agent)
	}

	type rawLine struct {
		lineNumber int
		byteOffset int64
		raw        []byte
		event      map[string]any
	}

	var (
		lines              []rawLine
		bytePos            int64
		hasDirectUser      bool
		hasDirectAssistant bool
	)

	warnings, err := VisitCompleteJSONL(b, MaxJSONLineBytes, func(lineNumber int, line []byte) error {
		start := bytePos
		bytePos += int64(len(line)) + 1 // newline stripped by scanner

		trimmed := trimJSONLSpace(line)
		if len(trimmed) == 0 {
			return nil
		}
		var event map[string]any
		if json.Unmarshal(trimmed, &event) != nil {
			report.MalformedLines++
			report.Warnings = append(report.Warnings, Warning{
				Agent:     sessionindex.AgentCodex,
				SessionID: b.SessionID,
				Code:      "malformed_jsonl_record",
				Message:   "Codex rollout line was not valid JSON; skipped",
			})
			return nil
		}
		payload, _ := event["payload"].(map[string]any)
		eventType := strings.ToLower(mapString(event, "type"))
		payloadType := strings.ToLower(mapString(payload, "type"))
		if eventType == "event_msg" && payloadType == "user_message" {
			hasDirectUser = true
		}
		if eventType == "event_msg" && payloadType == "agent_message" {
			hasDirectAssistant = true
		}
		lines = append(lines, rawLine{
			lineNumber: lineNumber,
			byteOffset: start,
			raw:        append([]byte(nil), trimmed...),
			event:      event,
		})
		return nil
	})
	if err != nil {
		return nil, report, err
	}
	report.Warnings = append(report.Warnings, warnings...)

	events := make([]capsule.Event, 0, len(lines))
	order := 0
	for _, line := range lines {
		payload, _ := line.event["payload"].(map[string]any)
		eventType := strings.ToLower(mapString(line.event, "type"))
		payloadType := strings.ToLower(mapString(payload, "type"))

		// Prefer event_msg over duplicate response_item message records when
		// both representations exist in this rollout (sessionindex dedup).
		if eventType == "response_item" && payloadType == "message" {
			role := strings.ToLower(mapString(payload, "role"))
			if role == "user" && hasDirectUser {
				continue
			}
			if role == "assistant" && hasDirectAssistant {
				continue
			}
		}

		ev, ok, skippedUnknown := codexMapRecord(b.SessionID, line.lineNumber, line.byteOffset, line.event, payload, line.raw)
		if skippedUnknown {
			report.UnknownRecords++
		}
		if !ok {
			continue
		}
		if ev.Truncated {
			report.TruncatedBlocks++
		}
		ev.Order = order
		order++
		events = append(events, ev)
		report.ByKind[ev.Kind]++
	}

	events = LinkToolResults(events)
	report.Events = len(events)
	return events, report, nil
}

func codexMapRecord(
	sessionID string,
	lineNumber int,
	byteOffset int64,
	event, payload map[string]any,
	raw []byte,
) (capsule.Event, bool, bool) {
	eventType := strings.ToLower(mapString(event, "type"))
	src := capsule.SourcePointer{
		Agent:      sessionindex.AgentCodex,
		SessionID:  sessionID,
		ByteOffset: byteOffset,
		Index:      lineNumber,
	}

	switch eventType {
	case "event_msg":
		return codexMapEventMsg(src, payload, raw)
	case "response_item":
		return codexMapResponseItem(src, payload, raw)
	case "session_meta":
		ev := baseEvent(src, raw)
		ev.Actor = capsule.ActorHarness
		ev.Kind = capsule.KindMetadata
		ev.NativeType = "session_meta"
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = "session_meta_referenced"
		ev.Blocks = nil
		return ev, true, false
	case "message":
		// Older bare message records (no response_item wrapper).
		return codexMapBareMessage(src, event, raw)
	default:
		if eventType == "" {
			actor, kind, port, reason := ClassifyUnknown("")
			ev := baseEvent(src, raw)
			ev.Actor = actor
			ev.Kind = kind
			ev.NativeType = ""
			ev.Portability = port
			ev.Reason = reason
			return ev, true, true
		}
		actor, kind, port, reason := ClassifyUnknown(eventType)
		ev := baseEvent(src, raw)
		ev.Actor = actor
		ev.Kind = kind
		ev.NativeType = eventType
		ev.Portability = port
		ev.Reason = reason
		ev.Blocks = []capsule.Block{RefBlock("opaque:" + contentDigest(raw))}
		return ev, true, true
	}
}

func codexMapEventMsg(src capsule.SourcePointer, payload map[string]any, raw []byte) (capsule.Event, bool, bool) {
	payloadType := strings.ToLower(mapString(payload, "type"))
	ev := baseEvent(src, raw)
	ev.NativeType = "event_msg/" + payloadType

	switch payloadType {
	case "user_message":
		text := codexEventMessageText(payload)
		ev.Actor = capsule.ActorUser
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		ev.Blocks, ev.Truncated = codexTextBlocks(text)
		ev.ContentHash = contentDigest([]byte(text))
		return ev, true, false
	case "agent_message":
		text := codexEventMessageText(payload)
		ev.Actor = capsule.ActorAssistant
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		ev.Blocks, ev.Truncated = codexTextBlocks(text)
		ev.ContentHash = contentDigest([]byte(text))
		return ev, true, false
	default:
		actor, kind, port, reason := ClassifyUnknown("event_msg/" + payloadType)
		ev.Actor = actor
		ev.Kind = kind
		ev.Portability = port
		ev.Reason = reason
		ev.Blocks = []capsule.Block{RefBlock("opaque:" + contentDigest(raw))}
		return ev, true, true
	}
}

func codexMapResponseItem(src capsule.SourcePointer, payload map[string]any, raw []byte) (capsule.Event, bool, bool) {
	payloadType := strings.ToLower(mapString(payload, "type"))
	ev := baseEvent(src, raw)
	ev.NativeType = "response_item/" + payloadType

	switch payloadType {
	case "message":
		role := strings.ToLower(mapString(payload, "role"))
		text := codexExtractTextContent(payload["content"])
		ev.Blocks, ev.Truncated = codexTextBlocks(text)
		ev.ContentHash = contentDigest([]byte(text))
		switch role {
		case "user":
			ev.Actor = capsule.ActorUser
			ev.Kind = capsule.KindMessage
			ev.Portability = capsule.PortabilityExact
			return ev, true, false
		case "assistant":
			ev.Actor = capsule.ActorAssistant
			ev.Kind = capsule.KindMessage
			ev.Portability = capsule.PortabilityExact
			return ev, true, false
		default:
			actor, kind, port, reason := ClassifyUnknown("response_item/message/" + role)
			ev.Actor = actor
			ev.Kind = kind
			ev.Portability = port
			ev.Reason = reason
			return ev, true, true
		}

	case "function_call", "custom_tool_call", "tool_call":
		name := mapString(payload, "name")
		callID := firstNonEmpty(
			mapString(payload, "call_id"),
			mapString(payload, "callId"),
			mapString(payload, "id"),
		)
		args := firstNonEmpty(
			mapString(payload, "arguments"),
			mapString(payload, "input"),
		)
		if args == "" {
			if rawArgs, err := json.Marshal(payload["arguments"]); err == nil && string(rawArgs) != "null" {
				args = string(rawArgs)
			}
		}
		ev.Actor = capsule.ActorAssistant
		ev.Kind = capsule.KindToolCall
		ev.NativeName = name
		ev.CallID = callID
		ev.Portability = capsule.PortabilityNormalized
		ev.Reason = "normalized_tool_call"
		block := capsule.Block{
			Type: capsule.BlockTypeToolInput,
			Text: args,
			Size: int64(len(args)),
		}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = contentDigest([]byte(name + "\n" + callID + "\n" + args))
		return ev, true, false

	case "function_call_output", "custom_tool_call_output", "function_output", "tool_result", "tool_output":
		callID := firstNonEmpty(
			mapString(payload, "call_id"),
			mapString(payload, "callId"),
			mapString(payload, "id"),
		)
		output := firstNonEmpty(
			mapString(payload, "output"),
			mapString(payload, "content"),
			mapString(payload, "result"),
		)
		if output == "" {
			output = codexExtractTextContent(payload["output"])
		}
		ev.Actor = capsule.ActorTool
		ev.Kind = capsule.KindToolResult
		ev.LinkedCallID = callID
		ev.Portability = capsule.PortabilityNormalized
		ev.Reason = "normalized_tool_result"
		block := capsule.Block{
			Type: capsule.BlockTypeToolOutput,
			Text: output,
			Size: int64(len(output)),
		}
		if isTruthy(payload["is_error"]) || isTruthy(payload["isError"]) {
			block.IsError = true
		}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = contentDigest([]byte(callID + "\n" + output))
		return ev, true, false

	case "reasoning":
		// R4: all reasoning / encrypted reasoning items are vendor-opaque.
		return codexOmittedReasoning(src, raw, payloadType), true, false

	default:
		// Encrypted or opaque reasoning may also appear under adjacent type names.
		if isCodexOpaqueReasoning(payloadType, payload) {
			return codexOmittedReasoning(src, raw, payloadType), true, false
		}
		actor, kind, port, reason := ClassifyUnknown("response_item/" + payloadType)
		ev.Actor = actor
		ev.Kind = kind
		ev.Portability = port
		ev.Reason = reason
		ev.Blocks = []capsule.Block{RefBlock("opaque:" + contentDigest(raw))}
		return ev, true, true
	}
}

func codexMapBareMessage(src capsule.SourcePointer, event map[string]any, raw []byte) (capsule.Event, bool, bool) {
	role := strings.ToLower(mapString(event, "role"))
	text := codexExtractTextContent(event["content"])
	if text == "" {
		text = mapString(event, "message", "text")
	}
	ev := baseEvent(src, raw)
	ev.NativeType = "message"
	ev.Blocks, ev.Truncated = codexTextBlocks(text)
	ev.ContentHash = contentDigest([]byte(text))
	switch role {
	case "user":
		ev.Actor = capsule.ActorUser
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		return ev, true, false
	case "assistant":
		ev.Actor = capsule.ActorAssistant
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		return ev, true, false
	default:
		actor, kind, port, reason := ClassifyUnknown("message/" + role)
		ev.Actor = actor
		ev.Kind = kind
		ev.Portability = port
		ev.Reason = reason
		return ev, true, true
	}
}

func codexOmittedReasoning(src capsule.SourcePointer, raw []byte, payloadType string) capsule.Event {
	ev := baseEvent(src, raw)
	ev.Actor = capsule.ActorAssistant
	ev.Kind = capsule.KindUnknown
	ev.NativeType = "response_item/" + payloadType
	ev.Portability = capsule.PortabilityOmitted
	ev.Reason = "vendor_opaque_state"
	ev.Blocks = nil
	// Digest the raw record for stability without retaining opaque bodies.
	ev.ContentHash = contentDigest(raw)
	return ev
}

// isCodexOpaqueReasoning reports R4 shapes that must never be translated.
func isCodexOpaqueReasoning(payloadType string, payload map[string]any) bool {
	if payloadType == "reasoning" {
		return true
	}
	if mapString(payload, "encrypted_content") != "" || mapString(payload, "encryptedContent") != "" {
		return true
	}
	if _, ok := payload["encrypted_content"]; ok {
		return true
	}
	if _, ok := payload["encryptedContent"]; ok {
		return true
	}
	return false
}

func codexEventMessageText(payload map[string]any) string {
	if message := mapString(payload, "message", "text"); message != "" {
		return message
	}
	return codexExtractTextContent(payload["content"])
}

func codexExtractTextContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var texts []string
		for _, raw := range typed {
			if text, ok := raw.(string); ok {
				texts = append(texts, text)
				continue
			}
			block, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			blockType := strings.ToLower(mapString(block, "type"))
			if blockType != "" && blockType != "text" && blockType != "input_text" && blockType != "output_text" {
				continue
			}
			if text := mapString(block, "text"); text != "" {
				texts = append(texts, text)
			}
		}
		return strings.Join(texts, "\n")
	case map[string]any:
		return mapString(typed, "text")
	default:
		return ""
	}
}

func codexTextBlocks(text string) ([]capsule.Block, bool) {
	block := TextBlock(text)
	if len(block.Text) <= capsule.MaxTextBlockBytes {
		return []capsule.Block{block}, false
	}
	return []capsule.Block{TruncateBlock(block, capsule.MaxTextBlockBytes)}, true
}

func baseEvent(src capsule.SourcePointer, raw []byte) capsule.Event {
	src.RecordKey = contentDigest(raw)[:16]
	return capsule.Event{
		ID:          capsule.EventID(src),
		Timestamp:   codexTimestamp(src, raw),
		ContentHash: contentDigest(raw),
		Source:      src,
	}
}

func codexTimestamp(_ capsule.SourcePointer, raw []byte) time.Time {
	var event map[string]any
	if json.Unmarshal(raw, &event) != nil {
		return time.Time{}
	}
	for _, key := range []string{"timestamp", "created_at", "createdAt"} {
		if s, ok := event[key].(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t.UTC()
			}
		}
		if payload, ok := event["payload"].(map[string]any); ok {
			if s, ok := payload[key].(string); ok && s != "" {
				if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
					return t.UTC()
				}
			}
		}
	}
	return time.Time{}
}

// codexSessionIDFromFilename mirrors sessionindex.codexSessionIDFromFilename:
// the trailing UUID of a rollout filename is the authoritative session id.
func codexSessionIDFromFilename(path string) string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fields := strings.Split(stem, "-")
	if len(fields) < 5 {
		return ""
	}
	candidate := strings.Join(fields[len(fields)-5:], "-")
	if !codexLooksLikeUUID(candidate) {
		return ""
	}
	return candidate
}

func codexLooksLikeUUID(value string) bool {
	groups := strings.Split(value, "-")
	if len(groups) != 5 {
		return false
	}
	for index, width := range [5]int{8, 4, 4, 4, 12} {
		if len(groups[index]) != width {
			return false
		}
		for _, char := range groups[index] {
			isDigit := char >= '0' && char <= '9'
			isHex := (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
			if !isDigit && !isHex {
				return false
			}
		}
	}
	return true
}

func mapString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func isTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "y":
			return true
		}
	case float64:
		return typed != 0
	}
	return false
}

func contentDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}
