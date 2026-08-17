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
	reasonKimiHarnessMeta    = "harness_meta_record"
	reasonKimiSourcePrompt   = "source_instruction_referenced"
	reasonKimiToolNormalized = "kimi_tool_call"
)

func init() {
	if err := Register(NewKimiReader()); err != nil {
		panic("transcript: register kimi reader: " + err.Error())
	}
}

// KimiReader converts Kimi Code CLI agents/main/wire.jsonl into capsule events.
//
// Conversation content lives on turn.prompt and context.append_message. Every
// other observed type is harness metadata and is classified omitted or
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

		produced, unknown, truncated := kimiEventsFromRecord(boundary, item, line, order, recordStart, index)
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
	events = LinkToolResults(events)
	report.Events = len(events)
	return events, report, ctx.Err()
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
	boundary Boundary,
	item map[string]any,
	raw []byte,
	order int,
	byteOffset int64,
	index int,
) (events []capsule.Event, unknown, truncated bool) {
	nativeType := firstString(item, "type")
	switch nativeType {
	case "turn.prompt":
		text := extractKimiText(item["input"])
		block, cut := kimiTextBlock(text)
		truncated = cut
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorUser, capsule.KindMessage, nativeType, "",
			[]capsule.Block{block}, capsule.PortabilityExact, "", byteOffset, index, nativeType,
		)}, false, truncated

	case "context.append_message":
		return kimiAppendMessage(boundary, item, order, byteOffset, index)

	case "profile.bind":
		// systemPrompt lives here. Reference the record; never copy the prompt.
		return []capsule.Event{kimiEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata, nativeType, "",
			[]capsule.Block{RefBlock("opaque:" + contentDigest(raw))},
			capsule.PortabilityReferenced, reasonKimiSourcePrompt, byteOffset, index, nativeType,
		)}, false, false

	case "metadata", "permission.set_mode", "plugin.session_start",
		"llm.tools_snapshot", "llm.request", "usage.record", "turn.ended",
		"context.append_loop_event":
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

func kimiAppendMessage(
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
	// turn.prompt already emitted the user side of the turn.
	if role == "user" {
		return nil, false, false
	}

	text := extractKimiText(message["content"])
	if text != "" && role != "tool" {
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
