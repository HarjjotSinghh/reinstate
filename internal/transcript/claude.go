package transcript

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	claudeAgentName           = "claude"
	claudeMinVerifiedVersion  = "2.1.219"
	claudeMaxVerifiedVersion  = "2.1.227"
	claudeLayoutProjectsJSONL = "projects-jsonl"

	reasonHarnessMeta           = "harness_meta_record"
	reasonHarnessMetadata       = "harness_metadata"
	reasonSourceInstruction     = "source_instruction_referenced"
	reasonToolUseNormalized     = "tool_use_normalized"
	reasonToolResultNormalized  = "tool_result_normalized"
	reasonVendorCompaction      = "vendor_compaction_summary"
	reasonAttachmentUnavailable = "attachment_unavailable"
	reasonAttachmentReferenced  = "attachment_path_referenced"
)

func init() {
	if err := Register(&ClaudeReader{}); err != nil {
		panic("transcript: register claude reader: " + err.Error())
	}
}

// ClaudeReader converts Claude Code project JSONL transcripts into canonical
// capsule events. It never writes vendor files and never reads a live
// contributor ~/.claude tree — callers pass an indexed session record.
type ClaudeReader struct{}

// Name returns the stable agent key "claude".
func (r *ClaudeReader) Name() string { return claudeAgentName }

// Probe reports layout/version support for a Claude session record.
// Unrecognized layouts are UNSUPPORTED; recognizable layouts outside the
// docs/compatibility.md range are UNTESTED.
func (r *ClaudeReader) Probe(_ context.Context, rec sessionindex.Record) (Compatibility, error) {
	if !claudeRecognizedLayout(rec.SourcePath) {
		return CompatibilityUnsupported, nil
	}
	version := claudeVersionAt(rec)
	if !adapter.StableVersionInRange(version, claudeMinVerifiedVersion, claudeMaxVerifiedVersion) {
		return CompatibilityUntested, nil
	}
	return CompatibilitySupported, nil
}

// Snapshot freezes the last complete JSONL record boundary for the session.
func (r *ClaudeReader) Snapshot(_ context.Context, rec sessionindex.Record) (Boundary, error) {
	if rec.SourcePath == "" {
		return Boundary{}, fmt.Errorf("transcript: claude snapshot requires source path")
	}
	sessionID := rec.ID
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(rec.SourcePath), filepath.Ext(rec.SourcePath))
	}
	return SnapshotJSONL(rec.SourcePath, claudeAgentName, sessionID, 0)
}

// Parse converts a frozen Claude boundary into canonical events.
func (r *ClaudeReader) Parse(_ context.Context, b Boundary) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	if b.Agent != "" && b.Agent != claudeAgentName {
		return nil, report, fmt.Errorf("transcript: claude parse got agent %q", b.Agent)
	}
	sessionDir := filepath.Dir(b.Path())

	var events []capsule.Event
	order := 0
	warnings, err := VisitCompleteJSONL(b, 0, func(lineNumber int, line []byte) error {
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			report.MalformedLines++
			report.Warnings = append(report.Warnings, Warning{
				Agent:   claudeAgentName,
				Code:    "malformed_record",
				Message: fmt.Sprintf("ignored malformed Claude JSONL record %d", lineNumber),
			})
			return nil
		}

		emitted, truncated, unknown, parseWarnings := parseClaudeRecord(b, sessionDir, lineNumber, raw, &order)
		report.TruncatedBlocks += truncated
		report.UnknownRecords += unknown
		report.Warnings = append(report.Warnings, parseWarnings...)
		events = append(events, emitted...)
		return nil
	})
	if err != nil {
		return nil, report, err
	}
	report.Warnings = append(report.Warnings, warnings...)

	events = LinkToolResults(events)
	for i := range events {
		report.ByKind[events[i].Kind]++
	}
	report.Events = len(events)
	return events, report, nil
}

func claudeRecognizedLayout(path string) bool {
	if path == "" {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	lower := strings.ToLower(clean)
	if !strings.HasSuffix(lower, ".jsonl") {
		return false
	}
	if strings.Contains(lower, "/subagents/") {
		return false
	}
	return strings.Contains(lower, "/projects/")
}

func claudeVersionAt(rec sessionindex.Record) string {
	root := sessionindex.AgentRoot(rec)
	if root == "" {
		return "unknown"
	}
	data, err := os.ReadFile(filepath.Join(root, "version"))
	if err != nil {
		return "unknown"
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "unknown"
	}
	return version
}

func parseClaudeRecord(
	b Boundary,
	sessionDir string,
	lineNumber int,
	raw map[string]any,
	order *int,
) (events []capsule.Event, truncated, unknown int, warnings []Warning) {
	nativeType := strings.ToLower(strings.TrimSpace(asString(raw["type"])))
	ts := parseClaudeTime(raw["timestamp"])
	recordKey := firstNonEmpty(asString(raw["uuid"]), asString(raw["id"]))
	isMeta := asBool(raw["isMeta"])

	switch nativeType {
	case "user":
		if isMeta {
			ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
			ev.Actor = capsule.ActorHarness
			ev.Kind = capsule.KindMetadata
			ev.Portability = capsule.PortabilityReferenced
			ev.Reason = reasonHarnessMeta
			ev.Blocks = claudeTextBlocks(raw, &truncated)
			ev.ContentHash = hashClaudeContent(ev)
			return []capsule.Event{ev}, truncated, 0, nil
		}
		return parseClaudeUserContent(b, sessionDir, lineNumber, raw, order, ts, recordKey, nativeType)

	case "assistant":
		return parseClaudeAssistantContent(b, sessionDir, lineNumber, raw, order, ts, recordKey, nativeType)

	case "summary":
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorHarness
		ev.Kind = capsule.KindSummary
		ev.Portability = capsule.PortabilitySummarized
		ev.Reason = reasonVendorCompaction
		text := firstNonEmpty(
			asString(raw["summary"]),
			asString(raw["title"]),
			asString(raw["leafUuid"]),
			extractClaudePlainText(messageMap(raw)["content"]),
		)
		if text != "" {
			block := TextBlock(text)
			if len(block.Text) > capsule.MaxTextBlockBytes {
				block = TruncateBlock(block, capsule.MaxTextBlockBytes)
				truncated++
				ev.Truncated = true
			}
			ev.Blocks = []capsule.Block{block}
		}
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, truncated, 0, nil

	case "system", "developer":
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorHarness
		ev.Kind = capsule.KindMetadata
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = reasonSourceInstruction
		ev.Blocks = claudeTextBlocks(raw, &truncated)
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, truncated, 0, nil

	case "session_meta", "metadata", "meta":
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorHarness
		ev.Kind = capsule.KindMetadata
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = reasonHarnessMetadata
		ev.Blocks = claudeTextBlocks(raw, &truncated)
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, truncated, 0, nil

	default:
		// message.role system/developer without a typed wrapper
		if msg := messageMap(raw); msg != nil {
			role := strings.ToLower(asString(msg["role"]))
			if role == "system" || role == "developer" {
				ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, firstNonEmpty(nativeType, role))
				ev.Actor = capsule.ActorHarness
				ev.Kind = capsule.KindMetadata
				ev.Portability = capsule.PortabilityReferenced
				ev.Reason = reasonSourceInstruction
				ev.NativeType = firstNonEmpty(nativeType, role)
				ev.Blocks = claudeTextBlocks(raw, &truncated)
				ev.ContentHash = hashClaudeContent(ev)
				return []capsule.Event{ev}, truncated, 0, nil
			}
		}
		actor, kind, port, reason := ClassifyUnknown(nativeType)
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = actor
		ev.Kind = kind
		ev.Portability = port
		ev.Reason = reason
		// Opaque hash of the raw record — never guess semantics.
		sum := sha256.Sum256(mustJSON(raw))
		ev.Blocks = []capsule.Block{RefBlock("opaque:" + hex.EncodeToString(sum[:8]))}
		ev.ContentHash = hex.EncodeToString(sum[:16])
		return []capsule.Event{ev}, truncated, 1, nil
	}
}

func parseClaudeUserContent(
	b Boundary,
	sessionDir string,
	lineNumber int,
	raw map[string]any,
	order *int,
	ts time.Time,
	recordKey, nativeType string,
) (events []capsule.Event, truncated, unknown int, warnings []Warning) {
	msg := messageMap(raw)
	content := any(nil)
	if msg != nil {
		content = msg["content"]
	}
	if content == nil {
		content = raw["content"]
	}

	switch typed := content.(type) {
	case string:
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorUser
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		block := TextBlock(typed)
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated++
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, truncated, 0, nil
	case []any:
		return emitClaudeBlocks(b, sessionDir, lineNumber, typed, order, ts, recordKey, nativeType, capsule.ActorUser)
	default:
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorUser
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, 0, 0, nil
	}
}

func parseClaudeAssistantContent(
	b Boundary,
	sessionDir string,
	lineNumber int,
	raw map[string]any,
	order *int,
	ts time.Time,
	recordKey, nativeType string,
) (events []capsule.Event, truncated, unknown int, warnings []Warning) {
	msg := messageMap(raw)
	content := any(nil)
	if msg != nil {
		content = msg["content"]
	}
	if content == nil {
		content = raw["content"]
	}

	switch typed := content.(type) {
	case string:
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorAssistant
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		block := TextBlock(typed)
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated++
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, truncated, 0, nil
	case []any:
		return emitClaudeBlocks(b, sessionDir, lineNumber, typed, order, ts, recordKey, nativeType, capsule.ActorAssistant)
	default:
		ev := baseClaudeEvent(b, lineNumber, 0, order, ts, recordKey, nativeType)
		ev.Actor = capsule.ActorAssistant
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		ev.ContentHash = hashClaudeContent(ev)
		return []capsule.Event{ev}, 0, 0, nil
	}
}

func emitClaudeBlocks(
	b Boundary,
	sessionDir string,
	lineNumber int,
	blocks []any,
	order *int,
	ts time.Time,
	recordKey, nativeType string,
	defaultActor capsule.Actor,
) (events []capsule.Event, truncated, unknown int, warnings []Warning) {
	var textParts []string
	subIndex := 0
	flushText := func() {
		if len(textParts) == 0 {
			return
		}
		ev := baseClaudeEvent(b, lineNumber, subIndex, order, ts, recordKey, nativeType)
		subIndex++
		ev.Actor = defaultActor
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		block := TextBlock(strings.Join(textParts, "\n"))
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated++
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = hashClaudeContent(ev)
		events = append(events, ev)
		textParts = nil
	}

	for _, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]any)
		if !ok {
			continue
		}
		blockType := strings.ToLower(asString(block["type"]))
		switch blockType {
		case "text", "input_text", "":
			if text := asString(block["text"]); text != "" {
				textParts = append(textParts, text)
			} else if s, ok := rawBlock.(string); ok && s != "" {
				textParts = append(textParts, s)
			}
		case "tool_use":
			flushText()
			ev := baseClaudeEvent(b, lineNumber, subIndex, order, ts, recordKey, "tool_use")
			subIndex++
			ev.Actor = capsule.ActorAssistant
			ev.Kind = capsule.KindToolCall
			ev.Portability = capsule.PortabilityNormalized
			ev.Reason = reasonToolUseNormalized
			ev.NativeName = asString(block["name"])
			ev.CallID = asString(block["id"])
			inputBlock := capsule.Block{
				Type: capsule.BlockTypeToolInput,
				Text: compactJSON(block["input"]),
			}
			inputBlock.Size = int64(len(inputBlock.Text))
			if len(inputBlock.Text) > capsule.MaxTextBlockBytes {
				inputBlock = TruncateBlock(inputBlock, capsule.MaxTextBlockBytes)
				truncated++
				ev.Truncated = true
			}
			ev.Blocks = []capsule.Block{inputBlock}
			ev.ContentHash = hashClaudeContent(ev)
			events = append(events, ev)
		case "tool_result":
			flushText()
			ev := baseClaudeEvent(b, lineNumber, subIndex, order, ts, recordKey, "tool_result")
			subIndex++
			ev.Actor = capsule.ActorTool
			ev.Kind = capsule.KindToolResult
			ev.Portability = capsule.PortabilityNormalized
			ev.Reason = reasonToolResultNormalized
			ev.LinkedCallID = firstNonEmpty(asString(block["tool_use_id"]), asString(block["toolUseId"]))
			out := capsule.Block{
				Type:    capsule.BlockTypeToolOutput,
				Text:    claudeToolResultText(block["content"]),
				IsError: asBool(block["is_error"]) || asBool(block["isError"]),
			}
			out.Size = int64(len(out.Text))
			if out.IsError {
				if out.Meta == nil {
					out.Meta = map[string]string{}
				}
				out.Meta["is_error"] = "true"
			}
			if len(out.Text) > capsule.MaxTextBlockBytes {
				out = TruncateBlock(out, capsule.MaxTextBlockBytes)
				truncated++
				ev.Truncated = true
			}
			ev.Blocks = []capsule.Block{out}
			ev.ContentHash = hashClaudeContent(ev)
			events = append(events, ev)
		case "image":
			flushText()
			ev, t := parseClaudeImageBlock(b, sessionDir, lineNumber, subIndex, order, ts, recordKey, block, defaultActor)
			subIndex++
			truncated += t
			events = append(events, ev)
		default:
			flushText()
			actor, kind, port, reason := ClassifyUnknown(blockType)
			ev := baseClaudeEvent(b, lineNumber, subIndex, order, ts, recordKey, blockType)
			subIndex++
			ev.Actor = actor
			ev.Kind = kind
			ev.Portability = port
			ev.Reason = reason
			sum := sha256.Sum256(mustJSON(block))
			ev.Blocks = []capsule.Block{RefBlock("opaque:" + hex.EncodeToString(sum[:8]))}
			ev.ContentHash = hex.EncodeToString(sum[:16])
			events = append(events, ev)
			unknown++
		}
	}
	flushText()
	return events, truncated, unknown, warnings
}

func parseClaudeImageBlock(
	b Boundary,
	sessionDir string,
	lineNumber, subIndex int,
	order *int,
	ts time.Time,
	recordKey string,
	block map[string]any,
	actor capsule.Actor,
) (capsule.Event, int) {
	ev := baseClaudeEvent(b, lineNumber, subIndex, order, ts, recordKey, "image")
	ev.Actor = actor
	ev.Kind = capsule.KindAttachment
	truncated := 0

	source, _ := block["source"].(map[string]any)
	mime := firstNonEmpty(
		asString(block["media_type"]),
		asString(block["mime"]),
		asString(source["media_type"]),
		asString(source["mime_type"]),
		"application/octet-stream",
	)
	localPath := firstNonEmpty(
		asString(block["path"]),
		asString(block["file"]),
		asString(source["path"]),
		asString(source["file"]),
		asString(source["file_path"]),
	)
	sourceType := strings.ToLower(asString(source["type"]))
	inlineData := asString(source["data"])

	if localPath != "" {
		resolved := localPath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(sessionDir, localPath)
		}
		if info, err := os.Stat(resolved); err == nil && info.Mode().IsRegular() {
			sum, size, hashErr := hashFileBounded(resolved, capsule.MaxTextBlockBytes)
			if hashErr == nil {
				ev.Portability = capsule.PortabilityReferenced
				ev.Reason = reasonAttachmentReferenced
				ev.Blocks = []capsule.Block{{
					Type:   capsule.BlockTypeAttachment,
					MIME:   mime,
					SHA256: sum,
					Size:   size,
					Ref:    "attachment:" + sum[:16],
					// Never emit absolute filesystem paths into the capsule.
					Path: "",
					Meta: map[string]string{
						"source":      "path",
						"layout":      claudeLayoutProjectsJSONL,
						"name":        filepath.Base(localPath),
						"media_type":  mime,
						"source_type": sourceType,
					},
				}}
				ev.ContentHash = hashClaudeContent(ev)
				return ev, truncated
			}
		}
		// Path claimed but unavailable on disk.
		ev.Portability = capsule.PortabilityOmitted
		ev.Reason = reasonAttachmentUnavailable
		ev.Blocks = []capsule.Block{{
			Type: capsule.BlockTypeAttachment,
			MIME: mime,
			Meta: map[string]string{"source": "path", "unavailable": "true"},
		}}
		ev.ContentHash = hashClaudeContent(ev)
		return ev, truncated
	}

	// Inline base64 (or any non-path image payload) is not re-embedded.
	_ = inlineData
	ev.Portability = capsule.PortabilityOmitted
	ev.Reason = reasonAttachmentUnavailable
	meta := map[string]string{"source": "inline", "unavailable": "true"}
	if sourceType != "" {
		meta["source_type"] = sourceType
	}
	if inlineData != "" {
		sum := sha256.Sum256([]byte(inlineData))
		meta["inline_sha256_prefix"] = hex.EncodeToString(sum[:8])
	}
	ev.Blocks = []capsule.Block{{
		Type: capsule.BlockTypeAttachment,
		MIME: mime,
		Meta: meta,
	}}
	ev.ContentHash = hashClaudeContent(ev)
	return ev, truncated
}

func baseClaudeEvent(
	b Boundary,
	lineNumber, subIndex int,
	order *int,
	ts time.Time,
	recordKey, nativeType string,
) capsule.Event {
	src := capsule.SourcePointer{
		Agent:      claudeAgentName,
		SessionID:  b.SessionID,
		RecordKey:  recordKey,
		ByteOffset: int64(lineNumber),
		Index:      subIndex,
	}
	*order++
	return capsule.Event{
		ID:         capsule.EventID(src),
		Order:      *order,
		Timestamp:  ts,
		NativeType: nativeType,
		Source:     src,
	}
}

func claudeTextBlocks(raw map[string]any, truncated *int) []capsule.Block {
	text := extractClaudePlainText(nil)
	if msg := messageMap(raw); msg != nil {
		text = extractClaudePlainText(msg["content"])
	}
	if text == "" {
		text = firstNonEmpty(asString(raw["summary"]), asString(raw["title"]), asString(raw["content"]))
	}
	if text == "" {
		return nil
	}
	block := TextBlock(text)
	if len(block.Text) > capsule.MaxTextBlockBytes {
		block = TruncateBlock(block, capsule.MaxTextBlockBytes)
		*truncated++
	}
	return []capsule.Block{block}
}

func extractClaudePlainText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var parts []string
		for _, raw := range typed {
			if s, ok := raw.(string); ok {
				parts = append(parts, s)
				continue
			}
			block, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			blockType := strings.ToLower(asString(block["type"]))
			if blockType != "" && blockType != "text" && blockType != "input_text" {
				continue
			}
			if text := asString(block["text"]); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return asString(typed["text"])
	default:
		return ""
	}
}

func claudeToolResultText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		return extractClaudePlainText(typed)
	case map[string]any:
		if text := asString(typed["text"]); text != "" {
			return text
		}
		return compactJSON(typed)
	default:
		return compactJSON(value)
	}
}

func messageMap(raw map[string]any) map[string]any {
	if msg, ok := raw["message"].(map[string]any); ok {
		return msg
	}
	return nil
}

func hashClaudeContent(ev capsule.Event) string {
	payload := map[string]any{
		"actor":          string(ev.Actor),
		"kind":           string(ev.Kind),
		"native_type":    ev.NativeType,
		"native_name":    ev.NativeName,
		"call_id":        ev.CallID,
		"linked_call_id": ev.LinkedCallID,
		"portability":    string(ev.Portability),
		"reason":         ev.Reason,
		"blocks":         ev.Blocks,
	}
	sum := sha256.Sum256(mustJSON(payload))
	return hex.EncodeToString(sum[:16])
}

func hashFileBounded(path string, maxBytes int) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	if maxBytes <= 0 {
		maxBytes = capsule.MaxTextBlockBytes
	}
	h := sha256.New()
	buf := make([]byte, 32<<10)
	var total int64
	for total < int64(maxBytes) {
		toRead := len(buf)
		if rem := int64(maxBytes) - total; int64(toRead) > rem {
			toRead = int(rem)
		}
		n, readErr := f.Read(buf[:toRead])
		if n > 0 {
			_, _ = h.Write(buf[:n])
			total += int64(n)
		}
		if readErr != nil {
			break
		}
	}
	return hex.EncodeToString(h.Sum(nil)), info.Size(), nil
}

func compactJSON(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return data
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func parseClaudeTime(v any) time.Time {
	switch typed := v.(type) {
	case string:
		if typed == "" {
			return time.Time{}
		}
		if ts, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return ts.UTC()
		}
		if ts, err := time.Parse(time.RFC3339, typed); err == nil {
			return ts.UTC()
		}
	case float64:
		return time.Unix(int64(typed), 0).UTC()
	}
	return time.Time{}
}
