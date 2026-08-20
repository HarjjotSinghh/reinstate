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
	"strconv"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// DestinationWarningGrok is the capsule security warning for Grok Build sources.
// It names the documented mid-2026 repository-upload history.
const DestinationWarningGrok = "grok_source_upload_history"

// ErrNoRedactRefused is returned when --no-redact is requested for a Grok source.
// CLI maps this to exit code 2 (usage).
var ErrNoRedactRefused = errors.New("no-redact refused for Grok Build source")

func init() {
	_ = Register(NewGrokReader())
}

// GrokReader converts Grok Build session directories into canonical capsule events.
//
// Authority: prefer updates.jsonl (append-only restore log). chat_history.jsonl
// is model-facing and may be rewritten by /compact; pre-compact turns live under
// compaction_requests/.
type GrokReader struct{}

// NewGrokReader returns the Grok Build transcript reader.
func NewGrokReader() *GrokReader {
	return &GrokReader{}
}

func (r *GrokReader) Name() string { return sessionindex.AgentGrok }

// ForcedSecurity returns the Phase 4 Grok source security policy.
// Redaction is unconditional; destination warning is always set.
func (r *GrokReader) ForcedSecurity() capsule.Security {
	return ForcedGrokSecurity()
}

// ForcedGrokSecurity is the capsule Security block for any Grok-sourced handoff.
func ForcedGrokSecurity() capsule.Security {
	return capsule.Security{
		DestinationWarning:                    DestinationWarningGrok,
		RedactionForced:                       true,
		SourceInstructionsAreUntrustedHistory: true,
	}
}

// RefuseNoRedact returns ErrNoRedactRefused when agent is Grok Build.
// Callers map the error to process exit 2.
func RefuseNoRedact(agent string) error {
	if strings.EqualFold(strings.TrimSpace(agent), sessionindex.AgentGrok) {
		return fmt.Errorf(
			"%w: redaction is forced for Grok Build sources (%s)",
			ErrNoRedactRefused,
			DestinationWarningGrok,
		)
	}
	return nil
}

func (r *GrokReader) Probe(_ context.Context, record sessionindex.Record) (Compatibility, error) {
	sessionDir, err := grokSessionDir(record)
	if err != nil {
		return CompatibilityUnsupported, err
	}
	summaryPath := filepath.Join(sessionDir, "summary.json")
	if _, err := os.Stat(summaryPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CompatibilityUnsupported, nil
		}
		return CompatibilityUnsupported, err
	}
	summary, err := readGrokSummaryFile(summaryPath)
	if err != nil {
		return CompatibilityUnsupported, err
	}
	if summary.ChatFormatVersion != nil {
		version := *summary.ChatFormatVersion
		if version != 0 && version != 1 {
			return CompatibilityUnsupported, nil
		}
	}
	if _, _, err := grokAuthorityPath(sessionDir); err != nil {
		return CompatibilityUnsupported, nil
	}
	return CompatibilitySupported, nil
}

func (r *GrokReader) Snapshot(_ context.Context, record sessionindex.Record) (Boundary, error) {
	sessionDir, err := grokSessionDir(record)
	if err != nil {
		return Boundary{}, err
	}
	path, _, err := grokAuthorityPath(sessionDir)
	if err != nil {
		return Boundary{}, err
	}
	id := strings.TrimSpace(record.ID)
	if id == "" {
		id = filepath.Base(sessionDir)
	}
	boundary, err := SnapshotJSONL(path, sessionindex.AgentGrok, id, MaxJSONLineBytes)
	if err != nil {
		return Boundary{}, err
	}
	return boundary.WithPathContext(PathContextFor(record)), nil
}

func (r *GrokReader) Parse(ctx context.Context, boundary Boundary) ([]capsule.Event, ParseReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, ParseReport{}, err
	}
	sessionDir := filepath.Dir(boundary.Path())
	if sessionDir == "" || sessionDir == "." {
		return nil, ParseReport{}, errors.New("transcript: grok boundary path missing session directory")
	}

	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	var events []capsule.Event
	order := 0

	// Prefer updates.jsonl markers for compaction checkpoints (authority).
	checkpoints, updateWarnings, malformed, unknown := parseGrokUpdates(boundary)
	report.Warnings = append(report.Warnings, updateWarnings...)
	report.MalformedLines += malformed
	report.UnknownRecords += unknown

	// Pre-compact turns from compaction_requests/ when checkpoints exist.
	if len(checkpoints) > 0 {
		preEvents, preReport, err := parseGrokCompactionRequests(sessionDir, boundary, &order)
		if err != nil {
			return nil, report, err
		}
		events = append(events, preEvents...)
		mergeParseReport(&report, preReport)
	}

	for _, checkpoint := range checkpoints {
		ev := grokEvent(
			boundary,
			order,
			capsule.ActorHarness,
			capsule.KindCheckpoint,
			"compaction_checkpoint",
			"",
			[]capsule.Block{RefBlock(checkpoint.Ref)},
			capsule.PortabilityReferenced,
			"grok_compaction_checkpoint",
			checkpoint.Offset,
			checkpoint.Index,
		)
		events = append(events, ev)
		report.ByKind[capsule.KindCheckpoint]++
		order++
	}

	historyPath := filepath.Join(sessionDir, "chat_history.jsonl")
	historyEvents, historyReport, err := parseGrokChatHistory(historyPath, boundary, &order)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, report, err
	}
	if err == nil {
		events = append(events, historyEvents...)
		mergeParseReport(&report, historyReport)
	}

	events = LinkToolResults(events)
	for i := range events {
		// Backstop for the path contract in paths.go: no structural value
		// leaves the reader as an absolute path. Without this a Grok block
		// carrying a vendor path in its text failed capsule validation and
		// made the whole handoff unusable.
		if boundary.paths.TokenizeBlocks(events[i].Blocks) {
			events[i].ContentHash = hashGrokContent(
				events[i].NativeType, events[i].NativeName, events[i].Blocks,
			)
		}
	}
	report.Events = len(events)
	return events, report, ctx.Err()
}

type grokCheckpoint struct {
	Ref    string
	Offset int64
	Index  int
}

type grokSummaryFile struct {
	ChatFormatVersion *int `json:"chat_format_version"`
}

func grokSessionDir(record sessionindex.Record) (string, error) {
	path := strings.TrimSpace(record.SourcePath)
	if path == "" {
		return "", errors.New("transcript: grok session source path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Clean(path), nil
	}
	// SourcePath may point at updates.jsonl / summary.json inside the session dir.
	return filepath.Clean(filepath.Dir(path)), nil
}

func grokAuthorityPath(sessionDir string) (string, os.FileInfo, error) {
	for _, name := range []string{"updates.jsonl", "chat_history.jsonl"} {
		path := filepath.Join(sessionDir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, info, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("transcript: grok session missing updates.jsonl and chat_history.jsonl")
}

func readGrokSummaryFile(path string) (grokSummaryFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return grokSummaryFile{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return grokSummaryFile{}, err
	}
	if len(data) > MaxJSONLineBytes {
		return grokSummaryFile{}, fmt.Errorf("transcript: grok summary exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var summary grokSummaryFile
	if err := json.Unmarshal(data, &summary); err != nil {
		return grokSummaryFile{}, err
	}
	return summary, nil
}

func parseGrokUpdates(boundary Boundary) ([]grokCheckpoint, []Warning, int, int) {
	var (
		checkpoints []grokCheckpoint
		warnings    []Warning
		malformed   int
		unknown     int
		offset      int64
		index       int
	)
	visitWarnings, err := VisitCompleteJSONL(boundary, MaxJSONLineBytes, func(lineNumber int, line []byte) error {
		start := offset
		offset += int64(len(line)) + 1 // account for newline consumed by scanner framing
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			malformed++
			return nil
		}
		nativeType := strings.ToLower(firstString(item, "type"))
		switch nativeType {
		case "compaction_checkpoint":
			ref := firstString(item, "checkpoint_file", "checkpoint_id")
			checkpoints = append(checkpoints, grokCheckpoint{
				Ref:    ref,
				Offset: start,
				Index:  index,
			})
			index++
		case "":
			malformed++
		default:
			// Opaque update variants are preserved as unknown counts; conversation
			// bodies come from chat_history / compaction_requests.
			unknown++
			index++
		}
		_ = lineNumber
		return nil
	})
	if err != nil {
		warnings = append(warnings, Warning{
			Agent:   sessionindex.AgentGrok,
			Code:    "updates_parse_failed",
			Message: "Grok updates.jsonl could not be fully parsed",
		})
	}
	warnings = append(warnings, visitWarnings...)
	return checkpoints, warnings, malformed, unknown
}

func parseGrokCompactionRequests(
	sessionDir string,
	boundary Boundary,
	order *int,
) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	dir := filepath.Join(sessionDir, "compaction_requests")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, report, nil
	}
	if err != nil {
		return nil, report, err
	}
	var events []capsule.Event
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		items, readErr := readGrokRequestHistory(path)
		if readErr != nil {
			report.Warnings = append(report.Warnings, Warning{
				Agent:   sessionindex.AgentGrok,
				Code:    "compaction_request_read_failed",
				Message: "Grok compaction request could not be read",
			})
			continue
		}
		for itemIndex, item := range items {
			ev, ok, truncated := grokConversationItemEvent(boundary, *order, item, 0, itemIndex, "compaction_request:"+entry.Name())
			if !ok {
				report.UnknownRecords++
				continue
			}
			if truncated {
				report.TruncatedBlocks++
			}
			events = append(events, ev)
			report.ByKind[ev.Kind]++
			*order++
		}
	}
	return events, report, nil
}

func parseGrokChatHistory(
	path string,
	boundary Boundary,
	order *int,
) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	file, err := os.Open(path)
	if err != nil {
		return nil, report, err
	}
	defer func() { _ = file.Close() }()

	var (
		events []capsule.Event
		offset int64
		index  int
	)
	warnings, err := sessionindex.ScanJSONLines(file, MaxJSONLineBytes, func(_ int, line []byte) error {
		start := offset
		offset += int64(len(line)) + 1
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			report.MalformedLines++
			return nil
		}
		ev, ok, truncated := grokConversationItemEvent(boundary, *order, item, start, index, "chat_history")
		index++
		if !ok {
			report.UnknownRecords++
			return nil
		}
		if truncated {
			report.TruncatedBlocks++
		}
		events = append(events, ev)
		report.ByKind[ev.Kind]++
		*order++
		return nil
	})
	report.Warnings = append(report.Warnings, warnings...)
	if err != nil {
		return events, report, err
	}
	return events, report, nil
}

func readGrokRequestHistory(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxJSONLineBytes {
		return nil, fmt.Errorf("transcript: compaction request exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	raw, ok := payload["chat_history"].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		if mapped, ok := value.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out, nil
}

func grokConversationItemEvent(
	boundary Boundary,
	order int,
	item map[string]any,
	byteOffset int64,
	index int,
	recordKey string,
) (capsule.Event, bool, bool) {
	nativeType := strings.ToLower(firstString(item, "type"))
	truncated := false

	switch nativeType {
	case "system":
		text := extractGrokText(item["content"])
		block := TextBlock(text)
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		return grokEvent(
			boundary, order, capsule.ActorHarness, capsule.KindMetadata, nativeType, "",
			[]capsule.Block{block}, capsule.PortabilityReferenced, "source_system_instruction",
			byteOffset, index, recordKey,
		), true, truncated

	case "user":
		text := extractGrokText(item["content"])
		block := TextBlock(text)
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		actor := capsule.ActorUser
		kind := capsule.KindMessage
		portability := capsule.PortabilityExact
		reason := ""
		if firstString(item, "synthetic_reason") != "" {
			actor = capsule.ActorHarness
			kind = capsule.KindSummary
			portability = capsule.PortabilitySummarized
			reason = "grok_compaction_summary"
		}
		return grokEvent(
			boundary, order, actor, kind, nativeType, "",
			[]capsule.Block{block}, portability, reason,
			byteOffset, index, recordKey,
		), true, truncated

	case "assistant":
		text := extractGrokText(item["content"])
		blocks := []capsule.Block{}
		if text != "" {
			block := TextBlock(text)
			if len(block.Text) > capsule.MaxTextBlockBytes {
				block = TruncateBlock(block, capsule.MaxTextBlockBytes)
				truncated = true
			}
			blocks = append(blocks, block)
		}
		ev := grokEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindMessage, nativeType, "",
			blocks, capsule.PortabilityExact, "",
			byteOffset, index, recordKey,
		)
		// Tool calls on the assistant item are represented as separate events by
		// the caller via expanding; here we attach call metadata when a single
		// call is present, otherwise leave message text and let unknown tooling
		// remain in native payload as normalized sidecar refs.
		if calls, ok := item["tool_calls"].([]any); ok {
			for callIndex, raw := range calls {
				call, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				name := firstString(call, "name", "tool_name")
				callID := firstString(call, "id", "call_id", "tool_call_id")
				if callID == "" {
					callID = fmt.Sprintf("%s:%d", recordKey, callIndex)
				}
				args := firstString(call, "arguments")
				if args == "" {
					if encoded, err := json.Marshal(call["arguments"]); err == nil {
						args = string(encoded)
					}
				}
				argBlock := capsule.Block{
					Type: capsule.BlockTypeToolInput,
					Text: args,
					Size: int64(len(args)),
				}
				if len(argBlock.Text) > capsule.MaxTextBlockBytes {
					argBlock = TruncateBlock(argBlock, capsule.MaxTextBlockBytes)
					truncated = true
				}
				// Prefer emitting tool_call as the primary event when there is
				// no assistant text; otherwise keep the message and note tools.
				if text == "" && len(calls) == 1 {
					ev = grokEvent(
						boundary, order, capsule.ActorAssistant, capsule.KindToolCall, "backend_tool_call", name,
						[]capsule.Block{argBlock}, capsule.PortabilityNormalized, "grok_tool_call",
						byteOffset, index, recordKey,
					)
					ev.CallID = callID
				} else {
					ev.Blocks = append(ev.Blocks, argBlock)
					ev.Portability = capsule.PortabilityNormalized
					if ev.Reason == "" {
						ev.Reason = "grok_inline_tool_calls"
					}
					if ev.CallID == "" {
						ev.CallID = callID
					}
					ev.NativeName = name
				}
			}
		}
		return ev, true, truncated

	case "tool_result":
		text := extractGrokText(item["content"])
		if text == "" {
			text = extractGrokText(item["output"])
		}
		block := capsule.Block{
			Type:    capsule.BlockTypeToolOutput,
			Text:    text,
			Size:    int64(len(text)),
			IsError: boolValue(item["is_error"]) || boolValue(item["isError"]),
		}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := grokEvent(
			boundary, order, capsule.ActorTool, capsule.KindToolResult, nativeType,
			firstString(item, "name", "tool_name"),
			[]capsule.Block{block}, capsule.PortabilityNormalized, "grok_tool_result",
			byteOffset, index, recordKey,
		)
		ev.LinkedCallID = firstString(item, "tool_call_id", "call_id", "id")
		return ev, true, truncated

	case "backend_tool_call":
		name := firstString(item, "name", "tool_name")
		callID := firstString(item, "id", "call_id", "tool_call_id")
		args := ""
		if encoded, err := json.Marshal(item["arguments"]); err == nil && string(encoded) != "null" {
			args = string(encoded)
		}
		block := capsule.Block{Type: capsule.BlockTypeToolInput, Text: args, Size: int64(len(args))}
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated = true
		}
		ev := grokEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindToolCall, nativeType, name,
			[]capsule.Block{block}, capsule.PortabilityNormalized, "grok_tool_call",
			byteOffset, index, recordKey,
		)
		ev.CallID = callID
		return ev, true, truncated

	case "reasoning":
		ev := grokEvent(
			boundary, order, capsule.ActorAssistant, capsule.KindMetadata, nativeType, "",
			nil, capsule.PortabilityOmitted, "vendor_opaque_state",
			byteOffset, index, recordKey,
		)
		return ev, true, false

	default:
		actor, kind, portability, reason := ClassifyUnknown(nativeType)
		ev := grokEvent(
			boundary, order, actor, kind, nativeType, "",
			nil, portability, reason,
			byteOffset, index, recordKey,
		)
		return ev, true, false
	}
}

func grokEvent(
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
	recordKey ...string,
) capsule.Event {
	key := ""
	if len(recordKey) > 0 {
		key = recordKey[0]
	}
	source := capsule.SourcePointer{
		Agent:      boundary.Agent,
		SessionID:  boundary.SessionID,
		RecordKey:  key,
		ByteOffset: byteOffset,
		Index:      index,
	}
	if source.Agent == "" {
		source.Agent = sessionindex.AgentGrok
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
		ContentHash: hashGrokContent(nativeType, nativeName, blocks),
		Source:      source,
		Truncated:   false,
	}
	for _, block := range blocks {
		if strings.Contains(block.Text, TruncationMarker) {
			ev.Truncated = true
			break
		}
	}
	return ev
}

func hashGrokContent(nativeType, nativeName string, blocks []capsule.Block) string {
	h := sha256.New()
	_, _ = io.WriteString(h, nativeType)
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, nativeName)
	_, _ = io.WriteString(h, "\n")
	for _, block := range blocks {
		_, _ = io.WriteString(h, string(block.Type))
		_, _ = io.WriteString(h, "\n")
		_, _ = io.WriteString(h, block.Text)
		_, _ = io.WriteString(h, "\n")
		_, _ = io.WriteString(h, block.Ref)
		_, _ = io.WriteString(h, "\n")
		_, _ = io.WriteString(h, strconv.FormatBool(block.IsError))
		_, _ = io.WriteString(h, "\n")
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:16])
}

func extractGrokText(value any) string {
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

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func mergeParseReport(dst *ParseReport, src ParseReport) {
	dst.MalformedLines += src.MalformedLines
	dst.UnknownRecords += src.UnknownRecords
	dst.TruncatedBlocks += src.TruncatedBlocks
	dst.Warnings = append(dst.Warnings, src.Warnings...)
	if dst.ByKind == nil {
		dst.ByKind = map[capsule.Kind]int{}
	}
	for kind, count := range src.ByKind {
		dst.ByKind[kind] += count
	}
}
