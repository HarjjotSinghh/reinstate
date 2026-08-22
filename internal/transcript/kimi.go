package transcript

import (
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

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Observed on both platform probes. Anything else is a layout change.
const (
	kimiStateSchemaVersion = 2
	kimiWireProtocolMajor  = "1"
)

const (
	reasonKimiHarnessMeta      = "harness_meta_record"
	reasonKimiSourcePrompt     = "source_instruction_referenced"
	reasonKimiToolNormalized   = "kimi_tool_call"
	reasonKimiToolResult       = "kimi_tool_result"
	reasonKimiHarnessInjection = "harness_injection"
	reasonKimiVendorPart       = "vendor_opaque_state"
)

// codeKimiContextRewritten is raised when the wire log contains a record that
// rewrites conversation history in place. The reader replays records in file
// order, so a capsule built from such a session can carry messages the vendor
// later dropped. Reported, never silently applied.
const codeKimiContextRewritten = "kimi_context_rewritten"

// kimiContextRewriteTypes are the wire ops whose reducers discard or replace
// already-appended context. Taken from the vendor's own op registry.
var kimiContextRewriteTypes = map[string]bool{
	"context.clear":            true,
	"context.undo":             true,
	"context.apply_compaction": true,
	"context.spliced":          true,
	"full_compaction.complete": true,
}

func init() {
	if err := Register(NewKimiReader()); err != nil {
		panic("transcript: register kimi reader: " + err.Error())
	}
}

// KimiReader converts Kimi Code CLI agents/main/wire.jsonl into capsule events.
//
// Two on-disk shapes carry conversation content, and both are read:
//
//   - Native (wire protocol 1.4/1.5, what the CLI writes today). The user side
//     is turn.prompt / turn.steer plus a role "user" context.append_message.
//     The assistant side is context.append_loop_event: step.begin opens a
//     partial assistant message, content.part appends its content, tool.call
//     appends a call, tool.result closes one, step.end settles. Assistant text
//     never appears as a role "assistant" context.append_message.
//   - Migrated legacy (wire protocol 1.0, written by `kimi migrate` from the
//     old kimi-cli store). Every message, including the assistant's, is a
//     context.append_message and there is no turn.prompt at all.
//
// Every other observed type is harness metadata and is classified omitted or
// referenced with no payload body. Unknown types are referenced, never guessed.
type KimiReader struct{}

// NewKimiReader returns the Kimi Code CLI transcript reader.
func NewKimiReader() *KimiReader { return &KimiReader{} }

func (r *KimiReader) Name() string { return sessionindex.AgentKimi }

func (r *KimiReader) Probe(_ context.Context, record sessionindex.Record) (Compatibility, error) {
	sessionDir, err := kimiSessionDir(record)
	if err != nil {
		return CompatibilityUnsupported, err
	}
	state, err := readKimiState(filepath.Join(sessionDir, "state.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CompatibilityUnsupported, nil
		}
		return CompatibilityUnsupported, err
	}
	if state.Version == nil || *state.Version != kimiStateSchemaVersion {
		return CompatibilityUnsupported, nil
	}
	wire := kimiWirePath(sessionDir)
	if _, err := os.Stat(wire); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CompatibilityUnsupported, nil
		}
		return CompatibilityUnsupported, err
	}
	if err := kimiCheckWireProtocol(wire); err != nil {
		return CompatibilityUnsupported, nil
	}
	return CompatibilitySupported, nil
}

func (r *KimiReader) Snapshot(_ context.Context, record sessionindex.Record) (Boundary, error) {
	sessionDir, err := kimiSessionDir(record)
	if err != nil {
		return Boundary{}, err
	}
	id := strings.TrimSpace(record.ID)
	if id == "" {
		id = filepath.Base(sessionDir)
	}
	boundary, err := SnapshotJSONL(kimiWirePath(sessionDir), sessionindex.AgentKimi, id, MaxJSONLineBytes)
	if err != nil {
		return Boundary{}, err
	}
	return boundary.WithPathContext(PathContextFor(record)), nil
}

func (r *KimiReader) Parse(ctx context.Context, boundary Boundary) ([]capsule.Event, ParseReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, ParseReport{}, err
	}
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	var events []capsule.Event
	order := 0
	offset := int64(0)
	first := true
	state := &kimiParseState{pendingPrompts: map[string]int{}}

	warnings, err := VisitCompleteJSONL(boundary, MaxJSONLineBytes, func(index int, line []byte) error {
		recordStart := offset
		offset += int64(len(line)) + 1

		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			report.MalformedLines++
			return nil
		}
		kind := firstString(item, "type")
		if first {
			first = false
			if kind == "metadata" {
				version := firstString(item, "protocol_version")
				if major, _, _ := strings.Cut(version, "."); major != kimiWireProtocolMajor {
					return fmt.Errorf("transcript: unsupported kimi wire protocol_version %q", version)
				}
			}
		}

		if kimiContextRewriteTypes[kind] {
			state.contextRewrites++
			state.rewriteType = kind
		}

		produced, unknown, truncated := kimiEventsFromRecord(state, boundary, item, line, order, recordStart, index)
		if truncated {
			report.TruncatedBlocks++
		}
		if unknown {
			report.UnknownRecords++
		}
		for _, ev := range produced {
			events = append(events, ev)
			report.ByKind[ev.Kind]++
			order++
		}
		return nil
	})
	report.Warnings = append(report.Warnings, warnings...)
	if err != nil {
		return nil, report, err
	}
	if state.contextRewrites > 0 {
		report.Warnings = append(report.Warnings, Warning{
			Agent:     firstNonEmpty(boundary.Agent, sessionindex.AgentKimi),
			SessionID: boundary.SessionID,
			Code:      codeKimiContextRewritten,
			Message: fmt.Sprintf(
				"kimi wire log rewrites context history %d time(s) (last: %s); the capsule replays every appended message in file order and may include messages the session later dropped",
				state.contextRewrites, state.rewriteType,
			),
		})
	}
	events = LinkToolResults(events)
	report.Events = len(events)
	return events, report, ctx.Err()
}

// kimiParseState is the per-Parse bookkeeping the wire log needs. It never
// leaves Parse and never holds vendor payload bodies beyond the prompt text
// already emitted as a user event.
type kimiParseState struct {
	// pendingPrompts counts user texts already emitted from turn.prompt /
	// turn.steer, so the role "user" context.append_message the CLI writes
	// immediately afterwards is not emitted a second time. A migrated legacy
	// wire has no turn.prompt, so nothing is pending and its user messages are
	// emitted from context.append_message instead.
	pendingPrompts  map[string]int
	contextRewrites int
	rewriteType     string
}

func (s *kimiParseState) notePrompt(text string) {
	if text == "" {
		return
	}
	s.pendingPrompts[text]++
}

// consumePrompt reports whether text was already emitted by a preceding
// turn.prompt / turn.steer, consuming the pending entry when it was.
func (s *kimiParseState) consumePrompt(text string) bool {
	if text == "" {
		return false
	}
	if s.pendingPrompts[text] == 0 {
		return false
	}
	s.pendingPrompts[text]--
	if s.pendingPrompts[text] == 0 {
		delete(s.pendingPrompts, text)
	}
	return true
}

type kimiStateFile struct {
	Version *int `json:"version"`
}

func readKimiState(path string) (kimiStateFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return kimiStateFile{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return kimiStateFile{}, err
	}
	if len(data) > MaxJSONLineBytes {
		return kimiStateFile{}, fmt.Errorf("transcript: kimi state.json exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var item kimiStateFile
	if err := json.Unmarshal(data, &item); err != nil {
		return kimiStateFile{}, err
	}
	return item, nil
}

func kimiSessionDir(record sessionindex.Record) (string, error) {
	path := strings.TrimSpace(record.SourcePath)
	if path == "" {
		return "", errors.New("transcript: kimi session source path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Clean(path), nil
	}
	switch filepath.Base(path) {
	case "wire.jsonl":
		// .../session/agents/main/wire.jsonl
		return filepath.Clean(filepath.Dir(filepath.Dir(filepath.Dir(path)))), nil
	case "state.json":
		return filepath.Clean(filepath.Dir(path)), nil
	default:
		return filepath.Clean(filepath.Dir(path)), nil
	}
}

func kimiWirePath(sessionDir string) string {
	return filepath.Join(sessionDir, "agents", "main", "wire.jsonl")
}

func kimiCheckWireProtocol(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, int64(MaxJSONLineBytes)+1))
	if err != nil {
		return err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return errors.New("transcript: kimi wire.jsonl is empty")
	}
	var item map[string]any
	if json.Unmarshal([]byte(line), &item) != nil {
		return errors.New("transcript: kimi wire.jsonl first record is not JSON")
	}
	if firstString(item, "type") != "metadata" {
		return errors.New("transcript: kimi wire.jsonl does not start with metadata")
	}
	version := firstString(item, "protocol_version")
	major, _, _ := strings.Cut(version, ".")
	if major != kimiWireProtocolMajor {
		return fmt.Errorf("transcript: unsupported kimi wire protocol_version %q", version)
	}
	return nil
}

func kimiEventsFromRecord(
	state *kimiParseState,
	boundary Boundary,
	item map[string]any,
	raw []byte,
	order int,
	byteOffset int64,
	index int,
) (events []capsule.Event, unknown, truncated bool) {
	nativeType := firstString(item, "type")
	switch nativeType {
	case "turn.prompt", "turn.steer":
		text := extractKimiText(item["input"])
		state.notePrompt(text)
		block, cut := kimiTextBlock(text)
		truncated = cut
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorUser, capsule.KindMessage, nativeType, "",
			[]capsule.Block{block}, capsule.PortabilityExact, "", byteOffset, index, nativeType,
		)}, false, truncated

	case "context.append_message":
		return kimiAppendMessage(state, boundary, item, order, byteOffset, index)

	case "context.append_loop_event":
		return kimiLoopEvent(boundary, item, raw, order, byteOffset, index)

	case "profile.bind":
		// systemPrompt lives here. Reference the record; never copy the prompt.
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata, nativeType, "",
			[]capsule.Block{RefBlock("opaque:" + contentDigest(raw))},
			capsule.PortabilityReferenced, reasonKimiSourcePrompt, byteOffset, index, nativeType,
		)}, false, false

	case "metadata", "permission.set_mode", "plugin.session_start",
		"llm.tools_snapshot", "llm.request", "usage.record", "turn.ended",
		"turn.cancel":
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata, nativeType, "",
			nil, capsule.PortabilityOmitted, reasonKimiHarnessMeta, byteOffset, index, nativeType,
		)}, false, false

	default:
		actor, kind, portability, reason := ClassifyUnknown(nativeType)
		return []capsule.Event{kimiEvent(
			boundary, order, actor, kind, nativeType, "",
			[]capsule.Block{RefBlock("opaque:" + contentDigest(raw))},
			portability, reason, byteOffset, index, nativeType,
		)}, true, false
	}
}

// kimiLoopEvent reads a context.append_loop_event record. This is where the
// CLI puts the assistant's side of a turn at wire protocol 1.4/1.5: step.begin
// opens a partial assistant message, content.part appends its content,
// tool.call and tool.result carry the tool round-trip, and step.end settles.
func kimiLoopEvent(
	boundary Boundary,
	item map[string]any,
	raw []byte,
	order int,
	byteOffset int64,
	index int,
) (events []capsule.Event, unknown, truncated bool) {
	const nativeType = "context.append_loop_event"
	event, ok := item["event"].(map[string]any)
	if !ok {
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata, nativeType, "",
			nil, capsule.PortabilityOmitted, reasonKimiHarnessMeta, byteOffset, index, nativeType,
		)}, false, false
	}

	loopType := firstString(event, "type")
	// tool.result carries parentUuid rather than its own uuid, so fall back to
	// the call id before the record type. A record key must stay unique inside
	// one wire log: it feeds the event id.
	recordKey := firstNonEmpty(
		firstString(event, "uuid"),
		firstString(event, "toolCallId", "tool_call_id"),
		nativeType+"/"+loopType,
	)

	switch loopType {
	case "content.part":
		part, _ := event["part"].(map[string]any)
		partType := strings.ToLower(firstString(part, "type"))
		if partType != "" && partType != "text" && partType != "input_text" {
			// Thinking and other vendor-private parts are referenced, never copied.
			return []capsule.Event{kimiEvent(
				boundary, order, capsule.ActorAssistant, capsule.KindMetadata,
				nativeType+"/"+loopType, partType,
				[]capsule.Block{RefBlock("opaque:" + contentDigest(raw))},
				capsule.PortabilityReferenced, reasonKimiVendorPart, byteOffset, index, recordKey,
			)}, false, false
		}
		text := extractKimiText(event["part"])
		if text == "" {
			return nil, false, false
		}
		block, cut := kimiTextBlock(text)
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindMessage,
			nativeType+"/"+loopType, "",
			[]capsule.Block{block}, capsule.PortabilityExact, "", byteOffset, index, recordKey,
		)}, false, cut

	case "tool.call":
		name := firstString(event, "name")
		callID := firstNonEmpty(firstString(event, "toolCallId", "tool_call_id", "id"), recordKey)
		args := kimiToolArgs(event)
		args = boundary.PathContext().TokenizeJSONText(args)
		block := capsule.Block{Type: capsule.BlockTypeToolInput, Text: args, Size: int64(len(args))}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := kimiEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindToolCall,
			nativeType+"/"+loopType, name,
			[]capsule.Block{block}, capsule.PortabilityNormalized, reasonKimiToolNormalized,
			byteOffset, index, callID,
		)
		ev.CallID = callID
		return []capsule.Event{ev}, false, truncated

	case "tool.result":
		result, _ := event["result"].(map[string]any)
		text := extractKimiText(result["output"])
		text = boundary.PathContext().TokenizeJSONText(text)
		block := capsule.Block{
			Type:    capsule.BlockTypeToolOutput,
			Text:    text,
			Size:    int64(len(text)),
			IsError: boolValue(result["isError"]),
		}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := kimiEvent(
			boundary, order, capsule.ActorTool, capsule.KindToolResult,
			nativeType+"/"+loopType, firstString(event, "name"),
			[]capsule.Block{block}, capsule.PortabilityNormalized, reasonKimiToolResult,
			byteOffset, index, recordKey,
		)
		ev.LinkedCallID = firstString(event, "toolCallId", "tool_call_id")
		return []capsule.Event{ev}, false, truncated

	case "step.begin", "step.end":
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata,
			nativeType+"/"+loopType, "",
			nil, capsule.PortabilityOmitted, reasonKimiHarnessMeta, byteOffset, index, recordKey,
		)}, false, false

	default:
		actor, kind, portability, reason := ClassifyUnknown(loopType)
		return []capsule.Event{kimiEvent(
			boundary, order, actor, kind, nativeType+"/"+loopType, "",
			[]capsule.Block{RefBlock("opaque:" + contentDigest(raw))},
			portability, reason, byteOffset, index, recordKey,
		)}, true, false
	}
}

func kimiAppendMessage(
	state *kimiParseState,
	boundary Boundary,
	item map[string]any,
	order int,
	byteOffset int64,
	index int,
) (events []capsule.Event, unknown, truncated bool) {
	message, ok := item["message"].(map[string]any)
	if !ok {
		return nil, false, false
	}
	role := strings.ToLower(firstString(message, "role"))
	text := extractKimiText(message["content"])

	if role == "user" {
		origin, _ := message["origin"].(map[string]any)
		if kind := strings.ToLower(firstString(origin, "kind")); kind != "" && kind != "user" {
			// Harness-injected user turns (permission-mode reminders and the
			// like) are the harness talking to itself, not the operator.
			return []capsule.Event{kimiEvent(
				boundary, order, capsule.ActorHarness, capsule.KindMetadata,
				"context.append_message", kind,
				nil, capsule.PortabilityOmitted, reasonKimiHarnessInjection,
				byteOffset, index, firstNonEmpty(firstString(message, "id"), "context.append_message"),
			)}, false, false
		}
		// A native wire already emitted this turn from turn.prompt. A migrated
		// legacy wire has no turn.prompt, so nothing is pending and the user
		// message is emitted from here instead.
		if state.consumePrompt(text) {
			return nil, false, false
		}
		if text == "" {
			return nil, false, false
		}
		block, cut := kimiTextBlock(text)
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorUser, capsule.KindMessage, "context.append_message", "",
			[]capsule.Block{block}, capsule.PortabilityExact, "", byteOffset, index,
			firstNonEmpty(firstString(message, "id"), "context.append_message"),
		)}, false, cut
	}

	if role == "tool" {
		text = boundary.PathContext().TokenizeJSONText(text)
		block := capsule.Block{
			Type:    capsule.BlockTypeToolOutput,
			Text:    text,
			Size:    int64(len(text)),
			IsError: boolValue(message["isError"]),
		}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := kimiEvent(
			boundary, order, capsule.ActorTool, capsule.KindToolResult, "context.append_message", "",
			[]capsule.Block{block}, capsule.PortabilityNormalized, reasonKimiToolResult,
			byteOffset, index, firstNonEmpty(firstString(message, "toolCallId", "tool_call_id"), "context.append_message"),
		)
		ev.LinkedCallID = firstString(message, "toolCallId", "tool_call_id")
		return []capsule.Event{ev}, false, truncated
	}

	if text != "" {
		block, cut := kimiTextBlock(text)
		truncated = cut
		events = append(events, kimiEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindMessage, "context.append_message", "",
			[]capsule.Block{block}, capsule.PortabilityExact, "", byteOffset, index, firstString(message, "id"),
		))
		order++
	}

	calls := kimiToolCalls(message)
	for i, call := range calls {
		name := firstString(call, "name")
		callID := firstString(call, "id", "callId", "call_id")
		if callID == "" {
			callID = fmt.Sprintf("%s:%d", firstNonEmpty(firstString(message, "id"), "kimi-tool"), i)
		}
		args := kimiToolArgs(call)
		args = boundary.PathContext().TokenizeJSONText(args)
		block := capsule.Block{Type: capsule.BlockTypeToolInput, Text: args, Size: int64(len(args))}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := kimiEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindToolCall, "context.append_message", name,
			[]capsule.Block{block}, capsule.PortabilityNormalized, reasonKimiToolNormalized,
			byteOffset, index, callID,
		)
		ev.CallID = callID
		events = append(events, ev)
		order++
	}
	return events, false, truncated
}

func kimiToolCalls(message map[string]any) []map[string]any {
	var out []map[string]any
	if calls, ok := message["toolCalls"].([]any); ok {
		for _, raw := range calls {
			if call, ok := raw.(map[string]any); ok {
				out = append(out, call)
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	content, ok := message["content"].([]any)
	if !ok {
		return nil
	}
	for _, raw := range content {
		block, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(firstString(block, "type"), "tool_use") {
			out = append(out, block)
		}
	}
	return out
}

func kimiToolArgs(call map[string]any) string {
	for _, key := range []string{"arguments", "input", "args"} {
		if call[key] == nil {
			continue
		}
		if text, ok := call[key].(string); ok {
			return text
		}
		encoded, err := json.Marshal(call[key])
		if err != nil {
			continue
		}
		if string(encoded) == "null" {
			continue
		}
		return string(encoded)
	}
	return ""
}

func kimiTextBlock(text string) (capsule.Block, bool) {
	block := TextBlock(text)
	if len(block.Text) > capsule.MaxTextBlockBytes {
		return TruncateBlock(block, capsule.MaxTextBlockBytes), true
	}
	return block, false
}

func kimiEvent(
	boundary Boundary,
	order int,
	actor capsule.Actor,
	kind capsule.Kind,
	nativeType, nativeName string,
	blocks []capsule.Block,
	portability capsule.Portability,
	reason string,
	byteOffset int64,
	index int,
	recordKey string,
) capsule.Event {
	source := capsule.SourcePointer{
		Agent:      firstNonEmpty(boundary.Agent, sessionindex.AgentKimi),
		SessionID:  boundary.SessionID,
		RecordKey:  recordKey,
		ByteOffset: byteOffset,
		Index:      index,
	}
	ev := capsule.Event{
		ID:          capsule.EventID(source),
		Order:       order,
		Actor:       actor,
		Kind:        kind,
		NativeType:  nativeType,
		NativeName:  nativeName,
		Blocks:      blocks,
		Portability: portability,
		Reason:      reason,
		ContentHash: hashKimiContent(nativeType, nativeName, blocks),
		Source:      source,
	}
	for _, block := range blocks {
		if strings.Contains(block.Text, TruncationMarker) {
			ev.Truncated = true
			break
		}
	}
	return ev
}

func hashKimiContent(nativeType, nativeName string, blocks []capsule.Block) string {
	h := sha256.New()
	_, _ = io.WriteString(h, nativeType+"\n"+nativeName+"\n")
	for _, block := range blocks {
		_, _ = io.WriteString(h, string(block.Type)+"\n"+block.Text+"\n"+block.Ref+"\n")
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:16])
}

func extractKimiText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var parts []string
		for _, raw := range typed {
			switch item := raw.(type) {
			case string:
				parts = append(parts, item)
			case map[string]any:
				blockType := strings.ToLower(firstString(item, "type"))
				if blockType != "" && blockType != "text" && blockType != "input_text" {
					continue
				}
				if text := firstString(item, "text"); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return firstString(typed, "text", "content")
	default:
		return ""
	}
}
