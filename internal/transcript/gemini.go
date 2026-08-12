package transcript

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	geminiAgentName = sessionindex.AgentGemini

	warningRewindTargetMissing = "rewind_target_not_found"
	warningGeminiSubagent      = "gemini_subagent_excluded"
	warningMalformedGemini     = "malformed_gemini_record"

	reasonGeminiToolCallNormalized   = "vendor_tool_call_normalized"
	reasonGeminiToolResultNormalized = "vendor_tool_result_normalized"
)

func init() {
	if err := Register(&GeminiReader{}); err != nil {
		panic(err)
	}
}

// GeminiReader converts Gemini CLI chat JSON/JSONL into canonical capsule events.
//
// It is source-only: Snapshot and Parse open records read-only and never write
// vendor session files. Real ~/.gemini trees are never consulted — only the
// path frozen into a sessionindex.Record / Boundary.
//
// $rewindTo semantics match vendor ChatRecordingService.rewindTo: on-disk lines
// are preserved (append marker), but replay truncates the active conversation
// from and including the target message id before events are emitted.
type GeminiReader struct{}

// Name returns the stable agent key "gemini".
func (GeminiReader) Name() string { return geminiAgentName }

// Probe reports layout support for a Gemini session record.
func (GeminiReader) Probe(_ context.Context, record sessionindex.Record) (Compatibility, error) {
	if !strings.EqualFold(strings.TrimSpace(record.Agent), geminiAgentName) {
		return CompatibilityUnsupported, nil
	}
	path := strings.TrimSpace(record.SourcePath)
	if path == "" {
		return CompatibilityUnsupported, nil
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jsonl", ".json":
		return CompatibilitySupported, nil
	default:
		return CompatibilityUnsupported, nil
	}
}

// Snapshot freezes a complete-record boundary for a Gemini session artifact.
func (r GeminiReader) Snapshot(_ context.Context, record sessionindex.Record) (Boundary, error) {
	path := strings.TrimSpace(record.SourcePath)
	if path == "" {
		return Boundary{}, errors.New("transcript/gemini: empty source path")
	}
	sessionID := strings.TrimSpace(record.ID)
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl":
		return SnapshotJSONL(path, geminiAgentName, sessionID, 0)
	case ".json":
		return snapshotGeminiLegacyJSON(path, sessionID)
	default:
		return Boundary{}, fmt.Errorf("transcript/gemini: unsupported layout %q", filepath.Ext(path))
	}
}

// Parse converts a frozen Gemini boundary into canonical events.
//
// Rewind markers are applied before emission. kind:"subagent" sessions are
// rejected. toolCalls[] become tool_call (and optional tool_result) events with
// normalized portability.
func (GeminiReader) Parse(_ context.Context, b Boundary) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	if b.Path() == "" {
		return nil, report, errors.New("transcript/gemini: boundary path is empty")
	}

	state, warnings, err := loadGeminiBoundary(b, &report)
	report.Warnings = append(report.Warnings, warnings...)
	if err != nil {
		return nil, report, err
	}
	if strings.EqualFold(state.kind, "subagent") {
		report.Warnings = append(report.Warnings, Warning{
			Agent:     geminiAgentName,
			SessionID: b.SessionID,
			Source:    b.Path(),
			Code:      warningGeminiSubagent,
			Message:   "Gemini kind:subagent sessions are excluded from handoff capsules",
		})
		return nil, report, errGeminiSubagentSession
	}

	events := emitGeminiEvents(b, state, &report)
	report.Events = len(events)
	return events, report, nil
}

var errGeminiSubagentSession = errors.New("transcript/gemini: kind subagent is excluded")

type geminiParseState struct {
	kind     string
	messages []geminiPendingMessage
}

type geminiPendingMessage struct {
	id         string
	nativeType string
	content    string
	toolCalls  []geminiToolCall
	timestamp  time.Time
	byteOffset int64
	index      int
}

type geminiToolCall struct {
	ID        string
	Name      string
	Args      json.RawMessage
	Result    string
	HasResult bool
}

func loadGeminiBoundary(b Boundary, report *ParseReport) (geminiParseState, []Warning, error) {
	ext := strings.ToLower(filepath.Ext(b.Path()))
	switch ext {
	case ".jsonl":
		return parseGeminiJSONLBoundary(b, report)
	case ".json":
		return parseGeminiLegacyBoundary(b, report)
	default:
		// Boundaries from SnapshotJSONL may point at a path without an
		// extension in tests; sniff the prefix.
		reader, err := PrefixReader(b)
		if err != nil {
			return geminiParseState{}, nil, err
		}
		defer func() { _ = reader.Close() }()
		head := make([]byte, 1)
		n, _ := io.ReadFull(reader, head)
		if n == 1 && head[0] == '[' {
			return geminiParseState{}, nil, errors.New("transcript/gemini: unexpected JSON array session")
		}
		// Re-parse as JSONL (Gemini chats are newline-delimited objects).
		return parseGeminiJSONLBoundary(b, report)
	}
}

func snapshotGeminiLegacyJSON(path, sessionID string) (Boundary, error) {
	file, err := os.Open(path) // read-only
	if err != nil {
		return Boundary{}, fmt.Errorf("transcript/gemini: open legacy snapshot: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return Boundary{}, fmt.Errorf("transcript/gemini: stat legacy snapshot: %w", err)
	}
	size := info.Size()
	if size > int64(MaxJSONLineBytes) {
		return Boundary{}, fmt.Errorf("transcript/gemini: legacy session exceeds %d-byte read limit", MaxJSONLineBytes)
	}

	limited := io.LimitReader(file, size)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Boundary{}, fmt.Errorf("transcript/gemini: read legacy snapshot: %w", err)
	}
	if int64(len(data)) != size {
		return Boundary{}, fmt.Errorf("transcript/gemini: short legacy read %d, want %d", len(data), size)
	}
	if size > 0 && !json.Valid(bytes.TrimSpace(data)) {
		return Boundary{}, errors.New("transcript/gemini: legacy session is not valid JSON")
	}
	sum := sha256.Sum256(data)
	return Boundary{
		Agent:      geminiAgentName,
		SessionID:  sessionID,
		ByteOffset: size,
		SizeBytes:  size,
		SHA256:     hex.EncodeToString(sum[:]),
		ModTimeNS:  info.ModTime().UnixNano(),
		Partial:    false,
		path:       path,
	}, nil
}

func parseGeminiLegacyBoundary(b Boundary, report *ParseReport) (geminiParseState, []Warning, error) {
	reader, err := PrefixReader(b)
	if err != nil {
		return geminiParseState{}, nil, err
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(io.LimitReader(reader, int64(MaxJSONLineBytes)+1))
	if err != nil {
		return geminiParseState{}, nil, err
	}
	if len(data) > MaxJSONLineBytes {
		return geminiParseState{}, nil, fmt.Errorf("transcript/gemini: legacy session exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var conversation map[string]any
	if err := json.Unmarshal(data, &conversation); err != nil {
		report.MalformedLines++
		return geminiParseState{}, []Warning{{
			Agent:     geminiAgentName,
			SessionID: b.SessionID,
			Source:    b.Path(),
			Code:      warningMalformedGemini,
			Message:   "Gemini legacy JSON session could not be parsed",
		}}, fmt.Errorf("transcript/gemini: legacy JSON: %w", err)
	}

	var state geminiParseState
	applyGeminiSessionMeta(conversation, &state)
	messages, ok := conversation["messages"].([]any)
	if !ok {
		return state, nil, nil
	}
	for i, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			report.UnknownRecords++
			continue
		}
		pending, ok := geminiMessageFromRecord(msg, 0, i)
		if !ok {
			report.UnknownRecords++
			continue
		}
		state.messages = append(state.messages, pending)
	}
	return state, nil, nil
}

func parseGeminiJSONLBoundary(b Boundary, report *ParseReport) (geminiParseState, []Warning, error) {
	reader, err := PrefixReader(b)
	if err != nil {
		return geminiParseState{}, nil, err
	}
	defer func() { _ = reader.Close() }()

	var (
		state    geminiParseState
		warnings []Warning
		offset   int64
		msgIndex int
	)

	scanWarnings, err := sessionindex.ScanJSONLines(reader, MaxJSONLineBytes, func(lineNumber int, line []byte) error {
		// ScanJSONLines trims the line; recover the on-disk span by counting
		// the trimmed payload plus a terminating newline when present in the
		// frozen prefix. Exact offsets for SourcePointer use the start of the
		// JSON object within the cumulative scan position.
		recordStart := offset
		// Approximate: each visited record ends at prior consumed bytes. We
		// track by summing len(line)+1 for the newline that ScanJSONLines saw
		// when the record was complete. Trailing spaces were trimmed from
		// `line`, so this is a stable lower bound suitable for EventID.
		_ = lineNumber

		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			report.MalformedLines++
			return nil
		}

		if update, ok := event["$set"].(map[string]any); ok {
			applyGeminiSessionMeta(update, &state)
			offset += int64(len(line)) + 1
			return nil
		}
		if rewindID, ok := event["$rewindTo"].(string); ok && strings.TrimSpace(rewindID) != "" {
			warnings = append(warnings, applyGeminiRewind(&state, strings.TrimSpace(rewindID), b)...)
			offset += int64(len(line)) + 1
			return nil
		}
		if eventType := strings.ToLower(firstStringMap(event, "type")); eventType != "" {
			pending, ok := geminiMessageFromRecord(event, recordStart, msgIndex)
			if !ok {
				report.UnknownRecords++
				offset += int64(len(line)) + 1
				return nil
			}
			state.messages = append(state.messages, pending)
			msgIndex++
			offset += int64(len(line)) + 1
			return nil
		}
		// Session header / metadata object without type.
		if _, hasSession := event["sessionId"]; hasSession || firstStringMap(event, "session_id") != "" {
			applyGeminiSessionMeta(event, &state)
			offset += int64(len(line)) + 1
			return nil
		}
		if _, hasKind := event["kind"]; hasKind {
			applyGeminiSessionMeta(event, &state)
			offset += int64(len(line)) + 1
			return nil
		}
		report.UnknownRecords++
		offset += int64(len(line)) + 1
		return nil
	})
	warnings = append(warnings, scanWarnings...)
	if err != nil {
		return state, warnings, err
	}
	return state, warnings, nil
}

func applyGeminiSessionMeta(values map[string]any, state *geminiParseState) {
	if value := firstStringMap(values, "kind"); value != "" {
		state.kind = value
	}
}

// applyGeminiRewind truncates pending messages from and including the target
// id, matching vendor rewindTo (exclusive of the target — the target never
// reaches the capsule). An unknown id is a no-op with a warning.
func applyGeminiRewind(state *geminiParseState, rewindID string, b Boundary) []Warning {
	for index := len(state.messages) - 1; index >= 0; index-- {
		if state.messages[index].id == rewindID {
			state.messages = state.messages[:index]
			return nil
		}
	}
	return []Warning{{
		Agent:     geminiAgentName,
		SessionID: b.SessionID,
		Source:    b.Path(),
		Code:      warningRewindTargetMissing,
		Message:   fmt.Sprintf("Gemini $rewindTo target %q was not found; rewind is a no-op", rewindID),
	}}
}

func geminiMessageFromRecord(values map[string]any, byteOffset int64, index int) (geminiPendingMessage, bool) {
	nativeType := strings.ToLower(firstStringMap(values, "type", "role"))
	if nativeType == "" {
		return geminiPendingMessage{}, false
	}
	msg := geminiPendingMessage{
		id:         firstStringMap(values, "id"),
		nativeType: nativeType,
		content:    extractGeminiText(values["content"]),
		timestamp:  parseGeminiTimestamp(values),
		byteOffset: byteOffset,
		index:      index,
	}
	if calls, ok := values["toolCalls"].([]any); ok {
		for i, raw := range calls {
			call, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			tc := geminiToolCall{
				Name: firstStringMap(call, "name"),
				ID:   firstStringMap(call, "id", "callId", "call_id"),
			}
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("%s-tool-%d", msg.id, i)
				if msg.id == "" {
					tc.ID = fmt.Sprintf("gemini-tool-%d-%d", index, i)
				}
			}
			if args := call["args"]; args != nil {
				if encoded, err := json.Marshal(args); err == nil {
					tc.Args = encoded
				}
			} else if args := call["arguments"]; args != nil {
				if encoded, err := json.Marshal(args); err == nil {
					tc.Args = encoded
				}
			}
			if result, ok := call["result"]; ok && result != nil {
				tc.HasResult = true
				switch typed := result.(type) {
				case string:
					tc.Result = typed
				default:
					if encoded, err := json.Marshal(typed); err == nil {
						tc.Result = string(encoded)
					}
				}
			}
			msg.toolCalls = append(msg.toolCalls, tc)
		}
	}
	return msg, true
}

func emitGeminiEvents(b Boundary, state geminiParseState, report *ParseReport) []capsule.Event {
	events := make([]capsule.Event, 0, len(state.messages))
	order := 0
	for _, msg := range state.messages {
		switch msg.nativeType {
		case "user", "human":
			ev := geminiTextEvent(b, msg, order, capsule.ActorUser, capsule.PortabilityExact, "")
			events = append(events, ev)
			report.ByKind[ev.Kind]++
			order++
		case "gemini", "model", "assistant":
			if strings.TrimSpace(msg.content) != "" || len(msg.toolCalls) == 0 {
				ev := geminiTextEvent(b, msg, order, capsule.ActorAssistant, capsule.PortabilityExact, "")
				events = append(events, ev)
				report.ByKind[ev.Kind]++
				order++
			}
			for callIndex, call := range msg.toolCalls {
				callEvent := geminiToolCallEvent(b, msg, call, callIndex, order)
				events = append(events, callEvent)
				report.ByKind[callEvent.Kind]++
				order++
				if call.HasResult {
					resultEvent := geminiToolResultEvent(b, msg, call, callIndex, order)
					events = append(events, resultEvent)
					report.ByKind[resultEvent.Kind]++
					order++
				}
			}
		default:
			actor, kind, portability, reason := ClassifyUnknown(msg.nativeType)
			ev := geminiTextEvent(b, msg, order, actor, portability, reason)
			ev.Kind = kind
			events = append(events, ev)
			report.ByKind[ev.Kind]++
			report.UnknownRecords++
			order++
		}
	}
	return events
}

func geminiTextEvent(
	b Boundary,
	msg geminiPendingMessage,
	order int,
	actor capsule.Actor,
	portability capsule.Portability,
	reason string,
) capsule.Event {
	blocks := []capsule.Block{}
	if msg.content != "" {
		blocks = append(blocks, TextBlock(msg.content))
	}
	src := capsule.SourcePointer{
		Agent:      geminiAgentName,
		SessionID:  b.SessionID,
		RecordKey:  msg.id,
		ByteOffset: msg.byteOffset,
		Index:      msg.index,
	}
	ev := capsule.Event{
		ID:          capsule.EventID(src),
		Order:       order,
		Timestamp:   msg.timestamp,
		Actor:       actor,
		Kind:        capsule.KindMessage,
		NativeType:  msg.nativeType,
		Blocks:      blocks,
		Portability: portability,
		Reason:      reason,
		ContentHash: geminiContentHash(actor, capsule.KindMessage, msg.nativeType, blocks, "", ""),
		Source:      src,
	}
	return ev
}

func geminiToolCallEvent(
	b Boundary,
	msg geminiPendingMessage,
	call geminiToolCall,
	callIndex, order int,
) capsule.Event {
	block := capsule.Block{
		Type: capsule.BlockTypeToolInput,
		Text: string(call.Args),
		Size: int64(len(call.Args)),
	}
	if block.Text == "" {
		block.Text = "{}"
		block.Size = 2
	}
	blocks := []capsule.Block{block}
	src := capsule.SourcePointer{
		Agent:      geminiAgentName,
		SessionID:  b.SessionID,
		RecordKey:  msg.id,
		ByteOffset: msg.byteOffset,
		Index:      msg.index*1000 + callIndex*2,
	}
	return capsule.Event{
		ID:          capsule.EventID(src),
		Order:       order,
		Timestamp:   msg.timestamp,
		Actor:       capsule.ActorAssistant,
		Kind:        capsule.KindToolCall,
		NativeType:  "toolCalls",
		NativeName:  call.Name,
		Blocks:      blocks,
		CallID:      call.ID,
		Portability: capsule.PortabilityNormalized,
		Reason:      reasonGeminiToolCallNormalized,
		ContentHash: geminiContentHash(capsule.ActorAssistant, capsule.KindToolCall, call.Name, blocks, call.ID, ""),
		Source:      src,
	}
}

func geminiToolResultEvent(
	b Boundary,
	msg geminiPendingMessage,
	call geminiToolCall,
	callIndex, order int,
) capsule.Event {
	blocks := []capsule.Block{{
		Type: capsule.BlockTypeToolOutput,
		Text: call.Result,
		Size: int64(len(call.Result)),
	}}
	src := capsule.SourcePointer{
		Agent:      geminiAgentName,
		SessionID:  b.SessionID,
		RecordKey:  msg.id,
		ByteOffset: msg.byteOffset,
		Index:      msg.index*1000 + callIndex*2 + 1,
	}
	return capsule.Event{
		ID:           capsule.EventID(src),
		Order:        order,
		Timestamp:    msg.timestamp,
		Actor:        capsule.ActorTool,
		Kind:         capsule.KindToolResult,
		NativeType:   "toolCalls.result",
		NativeName:   call.Name,
		Blocks:       blocks,
		LinkedCallID: call.ID,
		Portability:  capsule.PortabilityNormalized,
		Reason:       reasonGeminiToolResultNormalized,
		ContentHash:  geminiContentHash(capsule.ActorTool, capsule.KindToolResult, call.Name, blocks, "", call.ID),
		Source:       src,
	}
}

func geminiContentHash(
	actor capsule.Actor,
	kind capsule.Kind,
	native string,
	blocks []capsule.Block,
	callID, linked string,
) string {
	h := sha256.New()
	_, _ = io.WriteString(h, string(actor))
	_, _ = io.WriteString(h, "\x00")
	_, _ = io.WriteString(h, string(kind))
	_, _ = io.WriteString(h, "\x00")
	_, _ = io.WriteString(h, native)
	_, _ = io.WriteString(h, "\x00")
	_, _ = io.WriteString(h, callID)
	_, _ = io.WriteString(h, "\x00")
	_, _ = io.WriteString(h, linked)
	_, _ = io.WriteString(h, "\x00")
	for _, block := range blocks {
		_, _ = io.WriteString(h, string(block.Type))
		_, _ = io.WriteString(h, "\x00")
		_, _ = io.WriteString(h, block.Text)
		_, _ = io.WriteString(h, "\x00")
		_, _ = io.WriteString(h, block.Ref)
		_, _ = io.WriteString(h, "\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
}

func extractGeminiText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var texts []string
		for _, raw := range typed {
			switch item := raw.(type) {
			case string:
				texts = append(texts, item)
			case map[string]any:
				if text := firstStringMap(item, "text"); text != "" {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "\n")
	case map[string]any:
		return firstStringMap(typed, "text")
	default:
		return ""
	}
}

func parseGeminiTimestamp(values map[string]any) time.Time {
	for _, key := range []string{"timestamp", "lastUpdated", "updatedAt", "createdAt"} {
		raw, ok := values[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case string:
			if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
				return parsed.UTC()
			}
			if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
				return parsed.UTC()
			}
		case float64:
			return time.Unix(int64(typed), 0).UTC()
		case json.Number:
			if n, err := typed.Int64(); err == nil {
				return time.Unix(n, 0).UTC()
			}
		}
	}
	return time.Time{}
}

func firstStringMap(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
