package transcript

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	qwenAgentName          = sessionindex.AgentQwen
	qwenLayoutProjectChats = "projects-slug-chats-jsonl"

	reasonQwenHarnessMetadata  = "harness_metadata"
	reasonQwenToolCall         = "vendor_tool_call_normalized"
	reasonQwenToolResult       = "vendor_tool_result_normalized"
	reasonQwenRewindMarker     = "vendor_rewind_marker"
	reasonQwenCompaction       = "vendor_compaction_summary"
	reasonQwenReasoningOmitted = "vendor_opaque_state"
	reasonQwenInlineAttachment = "attachment_unavailable"
	reasonQwenFileAttachment   = "attachment_path_referenced"
	reasonQwenUnknownPart      = "unrecognized_part_type"

	warningQwenMalformed       = "malformed_qwen_record"
	warningQwenRewound         = "qwen_rewound_records_excluded"
	warningQwenChainIncomplete = "qwen_uuid_chain_incomplete"
)

func init() {
	if err := Register(NewQwenReader()); err != nil {
		panic("transcript: register qwen reader: " + err.Error())
	}
}

// QwenReader converts Qwen Code chat JSONL into canonical capsule events.
//
// It is source-only: Snapshot and Parse open the frozen record path read-only
// and never write a vendor file. No real ~/.qwen tree is ever consulted — the
// path comes from an indexed sessionindex.Record.
//
// Qwen's top-level record keys match Claude Code's (uuid / parentUuid /
// sessionId / cwd / type), which is exactly why the Claude reader must not be
// reused: the message body is a Gemini Content value —
// {"role":"user"|"model","parts":[…]} — not a Claude content-block array, so
// the Claude reader would find no text at all and emit empty messages.
//
// Rewind is the other reason this reader exists. The vendor's
// ChatRecordingService re-points lastRecordUuid at the record before the
// rewound turn and appends a subtype:"rewind" system record there; the
// discarded turns stay on disk on a dead branch of the uuid tree, and the
// vendor's own resume path walks the parentUuid chain back from the last
// record to decide what is live. Parse does the same, so a handoff never
// replays turns the user explicitly threw away.
type QwenReader struct {
	// ResolveVersion overrides installed-version resolution. Nil resolves
	// through internal/agentcheck; tests inject a fake so no unit test depends
	// on the contributor's installed agents.
	ResolveVersion VersionResolver
}

// NewQwenReader returns the Qwen Code transcript reader.
func NewQwenReader() *QwenReader { return &QwenReader{} }

// Name returns the stable agent key "qwen".
func (r *QwenReader) Name() string { return qwenAgentName }

// Probe applies the shared reader compatibility contract in compat.go: layout
// decides support, and a version is only consulted when the installed agent
// can actually report one.
func (r *QwenReader) Probe(ctx context.Context, rec sessionindex.Record) (Compatibility, error) {
	return probeCompatibility(ctx, rec, qwenRecognizedLayout(rec), r.ResolveVersion), nil
}

// Snapshot freezes the last complete JSONL record boundary for the session.
func (r *QwenReader) Snapshot(_ context.Context, rec sessionindex.Record) (Boundary, error) {
	path := strings.TrimSpace(rec.SourcePath)
	if path == "" {
		return Boundary{}, errors.New("transcript/qwen: snapshot requires a source path")
	}
	sessionID := strings.TrimSpace(rec.ID)
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	boundary, err := SnapshotJSONL(path, qwenAgentName, sessionID, MaxJSONLineBytes)
	if err != nil {
		return Boundary{}, err
	}
	return boundary.WithPathContext(PathContextFor(rec)), nil
}

// Parse converts a frozen Qwen boundary into canonical events.
func (r *QwenReader) Parse(ctx context.Context, b Boundary) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	if err := ctx.Err(); err != nil {
		return nil, report, err
	}
	if b.Agent != "" && b.Agent != qwenAgentName {
		return nil, report, fmt.Errorf("transcript/qwen: parse got agent %q", b.Agent)
	}

	records, warnings, err := readQwenRecords(b, &report)
	report.Warnings = append(report.Warnings, warnings...)
	if err != nil {
		return nil, report, err
	}

	live, excluded, chainComplete := qwenLiveChain(records)
	if !chainComplete {
		report.Warnings = append(report.Warnings, Warning{
			Agent:     qwenAgentName,
			SessionID: b.SessionID,
			Code:      warningQwenChainIncomplete,
			Message:   "Qwen parentUuid chain did not reach the first record; every record was kept",
		})
	}
	if excluded > 0 {
		report.Warnings = append(report.Warnings, Warning{
			Agent:     qwenAgentName,
			SessionID: b.SessionID,
			Code:      warningQwenRewound,
			Message: fmt.Sprintf(
				"%d Qwen records sit on a rewound branch and are not part of the live conversation", excluded,
			),
		})
	}

	var events []capsule.Event
	order := 0
	for _, rec := range records {
		if live != nil {
			if _, ok := live[rec.uuid]; !ok {
				continue
			}
		}
		emitted, truncated, unknown := parseQwenRecord(b, rec, &order)
		report.TruncatedBlocks += truncated
		report.UnknownRecords += unknown
		events = append(events, emitted...)
	}

	events = LinkToolResults(events)
	for i := range events {
		// Backstop for the path contract in paths.go: no structural value
		// leaves the reader as an absolute path.
		if b.paths.TokenizeBlocks(events[i].Blocks) {
			events[i].ContentHash = hashQwenContent(events[i])
		}
		report.ByKind[events[i].Kind]++
	}
	report.Events = len(events)
	return events, report, ctx.Err()
}

// qwenRecord is one decoded conversation record plus its file position.
type qwenRecord struct {
	raw        map[string]any
	uuid       string
	parentUUID string
	lineNumber int
}

func qwenRecognizedLayout(rec sessionindex.Record) bool {
	if rec.Agent != "" && !strings.EqualFold(strings.TrimSpace(rec.Agent), qwenAgentName) {
		return false
	}
	path := strings.TrimSpace(rec.SourcePath)
	if path == "" {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	if !strings.HasSuffix(lower, ".jsonl") {
		return false
	}
	// Subagent transcripts live at projects/<slug>/subagents/<id>/ and are not
	// sessions; only the chats/ tree under a project bucket is a conversation.
	if !strings.Contains(lower, "/projects/") || !strings.Contains(lower, "/chats/") {
		return false
	}
	return true
}

func readQwenRecords(b Boundary, report *ParseReport) ([]qwenRecord, []Warning, error) {
	var records []qwenRecord
	warnings, err := VisitCompleteJSONL(b, MaxJSONLineBytes, func(lineNumber int, line []byte) error {
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			report.MalformedLines++
			report.Warnings = append(report.Warnings, Warning{
				Agent:   qwenAgentName,
				Code:    warningQwenMalformed,
				Message: fmt.Sprintf("ignored malformed Qwen JSONL record %d", lineNumber),
			})
			return nil
		}
		uuid := strings.TrimSpace(asString(raw["uuid"]))
		if uuid == "" {
			// Without a uuid the record cannot be placed in the parentUuid
			// chain, so it cannot be judged live or rewound. Counting it as
			// unknown is honest; guessing its position is not.
			report.UnknownRecords++
			report.Warnings = append(report.Warnings, Warning{
				Agent:   qwenAgentName,
				Code:    warningQwenMalformed,
				Message: fmt.Sprintf("ignored Qwen record %d with no uuid", lineNumber),
			})
			return nil
		}
		records = append(records, qwenRecord{
			raw:        raw,
			uuid:       uuid,
			parentUUID: strings.TrimSpace(asString(raw["parentUuid"])),
			lineNumber: lineNumber,
		})
		return nil
	})
	if err != nil {
		return nil, warnings, err
	}
	return records, warnings, nil
}

// qwenLiveChain returns the set of uuids reachable by walking parentUuid back
// from the last conversation record, the number of records excluded by that
// walk, and whether the walk terminated cleanly at a root record.
//
// A nil set means "keep everything": either the file has no usable chain, or
// the walk could not reach a root, and dropping records on a guess would lose
// real conversation.
func qwenLiveChain(records []qwenRecord) (live map[string]struct{}, excluded int, complete bool) {
	if len(records) == 0 {
		return nil, 0, true
	}
	byUUID := make(map[string]qwenRecord, len(records))
	for _, rec := range records {
		if _, seen := byUUID[rec.uuid]; !seen {
			byUUID[rec.uuid] = rec
		}
	}

	leaf := ""
	for i := len(records) - 1; i >= 0; i-- {
		if qwenIsConversationRecord(records[i].raw) {
			leaf = records[i].uuid
			break
		}
	}
	if leaf == "" {
		return nil, 0, true
	}

	live = map[string]struct{}{}
	for uuid := leaf; uuid != ""; {
		if _, seen := live[uuid]; seen {
			// A cycle is corruption, not a rewind.
			return nil, 0, false
		}
		live[uuid] = struct{}{}
		rec, ok := byUUID[uuid]
		if !ok {
			return nil, 0, false
		}
		uuid = rec.parentUUID
	}
	excluded = len(records) - len(live)
	return live, excluded, true
}

// qwenIsConversationRecord mirrors the vendor's isTranscriptConversationRecord:
// every record except the two session-artifact system subtypes participates in
// the uuid chain that resume replays.
func qwenIsConversationRecord(raw map[string]any) bool {
	if strings.ToLower(asString(raw["type"])) != "system" {
		return true
	}
	switch strings.ToLower(asString(raw["subtype"])) {
	case "session_artifact_event", "session_artifact_snapshot":
		return false
	default:
		return true
	}
}

func parseQwenRecord(b Boundary, rec qwenRecord, order *int) (events []capsule.Event, truncated, unknown int) {
	raw := rec.raw
	nativeType := strings.ToLower(strings.TrimSpace(asString(raw["type"])))
	ts := parseQwenTime(raw["timestamp"])

	switch nativeType {
	case "user":
		return parseQwenMessageParts(b, rec, order, ts, nativeType, qwenUserActor(raw))
	case "assistant":
		return parseQwenMessageParts(b, rec, order, ts, nativeType, capsule.ActorAssistant)
	case "tool_result":
		return parseQwenMessageParts(b, rec, order, ts, nativeType, capsule.ActorTool)
	case "system":
		ev, t := parseQwenSystemRecord(b, rec, order, ts)
		return []capsule.Event{ev}, t, 0
	default:
		ev := baseQwenEvent(b, rec, 0, order, ts, nativeType)
		actor, kind, portability, reason := ClassifyUnknown(nativeType)
		ev.Actor = actor
		ev.Kind = kind
		ev.Portability = portability
		ev.Reason = reason
		sum := sha256.Sum256(mustJSON(raw))
		ev.Blocks = []capsule.Block{RefBlock("opaque:" + hex.EncodeToString(sum[:8]))}
		ev.ContentHash = hex.EncodeToString(sum[:16])
		return []capsule.Event{ev}, 0, 1
	}
}

// qwenUserActor keeps the vendor's own provenance as the authority for whether
// a user-role record is a person or the harness. Qwen records cron prompts,
// notifications, and goal-runtime messages as type "user" with provenance
// "system"; treating those as user turns would put harness text in the capsule
// as if the operator had typed it.
func qwenUserActor(raw map[string]any) capsule.Actor {
	if strings.ToLower(asString(raw["provenance"])) == "real_user" {
		return capsule.ActorUser
	}
	if asString(raw["provenance"]) == "" && asString(raw["subtype"]) == "" {
		return capsule.ActorUser
	}
	if strings.ToLower(asString(raw["subtype"])) == "mid_turn_user_message" {
		return capsule.ActorUser
	}
	return capsule.ActorHarness
}

func parseQwenMessageParts(
	b Boundary,
	rec qwenRecord,
	order *int,
	ts time.Time,
	nativeType string,
	actor capsule.Actor,
) (events []capsule.Event, truncated, unknown int) {
	parts := qwenParts(rec.raw)
	subIndex := 0
	var textParts []string

	flushText := func() {
		if len(textParts) == 0 {
			return
		}
		ev := baseQwenEvent(b, rec, subIndex, order, ts, nativeType)
		subIndex++
		ev.Actor = actor
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		if actor == capsule.ActorHarness {
			ev.Kind = capsule.KindMetadata
			ev.Portability = capsule.PortabilityReferenced
			ev.Reason = reasonQwenHarnessMetadata
		}
		block := TextBlock(strings.Join(textParts, "\n"))
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated++
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
		ev.ContentHash = hashQwenContent(ev)
		events = append(events, ev)
		textParts = nil
	}

	for _, part := range parts {
		switch {
		case asBool(part["thought"]):
			flushText()
			ev := baseQwenEvent(b, rec, subIndex, order, ts, "thought")
			subIndex++
			ev.Actor = capsule.ActorAssistant
			ev.Kind = capsule.KindMetadata
			ev.Portability = capsule.PortabilityOmitted
			ev.Reason = reasonQwenReasoningOmitted
			ev.ContentHash = hashQwenContent(ev)
			events = append(events, ev)

		case part["functionCall"] != nil:
			flushText()
			ev, t := qwenToolCallEvent(b, rec, subIndex, order, ts, part)
			subIndex++
			truncated += t
			events = append(events, ev)

		case part["functionResponse"] != nil:
			flushText()
			ev, t := qwenToolResultEvent(b, rec, subIndex, order, ts, part)
			subIndex++
			truncated += t
			events = append(events, ev)

		case part["inlineData"] != nil || part["fileData"] != nil:
			flushText()
			ev := qwenAttachmentEvent(b, rec, subIndex, order, ts, part)
			subIndex++
			events = append(events, ev)

		case part["text"] != nil:
			if text := asString(part["text"]); text != "" {
				textParts = append(textParts, text)
			}

		default:
			flushText()
			ev := baseQwenEvent(b, rec, subIndex, order, ts, nativeType)
			subIndex++
			ev.Actor = capsule.ActorUnknown
			ev.Kind = capsule.KindUnknown
			ev.Portability = capsule.PortabilityReferenced
			ev.Reason = reasonQwenUnknownPart
			sum := sha256.Sum256(mustJSON(part))
			ev.Blocks = []capsule.Block{RefBlock("opaque:" + hex.EncodeToString(sum[:8]))}
			ev.ContentHash = hex.EncodeToString(sum[:16])
			events = append(events, ev)
			unknown++
		}
	}
	flushText()

	if len(events) == 0 {
		// A record with a message but no usable part still happened.
		ev := baseQwenEvent(b, rec, 0, order, ts, nativeType)
		ev.Actor = actor
		ev.Kind = capsule.KindMessage
		ev.Portability = capsule.PortabilityExact
		ev.ContentHash = hashQwenContent(ev)
		events = append(events, ev)
	}
	return events, truncated, unknown
}

func qwenToolCallEvent(
	b Boundary,
	rec qwenRecord,
	subIndex int,
	order *int,
	ts time.Time,
	part map[string]any,
) (capsule.Event, int) {
	call, _ := part["functionCall"].(map[string]any)
	ev := baseQwenEvent(b, rec, subIndex, order, ts, "functionCall")
	ev.Actor = capsule.ActorAssistant
	ev.Kind = capsule.KindToolCall
	ev.Portability = capsule.PortabilityNormalized
	ev.Reason = reasonQwenToolCall
	ev.NativeName = asString(call["name"])
	ev.CallID = asString(call["id"])

	// Tool arguments carry vendor paths (path, file_path, directory). Tokenize
	// inside the decoded JSON so each value is judged on its own.
	tokenized, _ := b.paths.TokenizeJSON(call["args"])
	block := capsule.Block{Type: capsule.BlockTypeToolInput, Text: compactJSON(tokenized)}
	block.Size = int64(len(block.Text))
	truncated := 0
	if len(block.Text) > capsule.MaxTextBlockBytes {
		block = TruncateBlock(block, capsule.MaxTextBlockBytes)
		truncated++
		ev.Truncated = true
	}
	ev.Blocks = []capsule.Block{block}
	ev.ContentHash = hashQwenContent(ev)
	return ev, truncated
}

func qwenToolResultEvent(
	b Boundary,
	rec qwenRecord,
	subIndex int,
	order *int,
	ts time.Time,
	part map[string]any,
) (capsule.Event, int) {
	response, _ := part["functionResponse"].(map[string]any)
	ev := baseQwenEvent(b, rec, subIndex, order, ts, "functionResponse")
	ev.Actor = capsule.ActorTool
	ev.Kind = capsule.KindToolResult
	ev.Portability = capsule.PortabilityNormalized
	ev.Reason = reasonQwenToolResult
	ev.NativeName = asString(response["name"])
	ev.LinkedCallID = asString(response["id"])

	payload, _ := response["response"].(map[string]any)
	text := ""
	isError := false
	switch {
	case payload == nil:
		text = compactJSON(response["response"])
	case payload["error"] != nil:
		text = qwenResponseText(payload["error"])
		isError = true
	default:
		text = qwenResponseText(firstNonNil(payload["output"], payload["result"], payload["content"]))
		if text == "" {
			text = compactJSON(payload)
		}
	}
	if result, ok := rec.raw["toolCallResult"].(map[string]any); ok {
		if strings.EqualFold(asString(result["status"]), "error") ||
			strings.EqualFold(asString(result["executionStatus"]), "error") {
			isError = true
		}
	}

	block := capsule.Block{
		Type:    capsule.BlockTypeToolOutput,
		Text:    b.paths.Tokenize(text),
		IsError: isError,
	}
	block.Size = int64(len(block.Text))
	if isError {
		block.Meta = map[string]string{"is_error": "true"}
	}
	truncated := 0
	if len(block.Text) > capsule.MaxTextBlockBytes {
		block = TruncateBlock(block, capsule.MaxTextBlockBytes)
		truncated++
		ev.Truncated = true
	}
	ev.Blocks = []capsule.Block{block}
	ev.ContentHash = hashQwenContent(ev)
	return ev, truncated
}

func qwenAttachmentEvent(
	b Boundary,
	rec qwenRecord,
	subIndex int,
	order *int,
	ts time.Time,
	part map[string]any,
) capsule.Event {
	ev := baseQwenEvent(b, rec, subIndex, order, ts, "attachment")
	ev.Actor = capsule.ActorUser
	ev.Kind = capsule.KindAttachment

	if fileData, ok := part["fileData"].(map[string]any); ok {
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = reasonQwenFileAttachment
		mime := firstNonEmpty(asString(fileData["mimeType"]), "application/octet-stream")
		ev.Blocks = []capsule.Block{{
			Type: capsule.BlockTypeAttachment,
			MIME: mime,
			Ref:  b.paths.Tokenize(asString(fileData["fileUri"])),
			Meta: map[string]string{"source": "file_data", "layout": qwenLayoutProjectChats},
		}}
		ev.ContentHash = hashQwenContent(ev)
		return ev
	}

	// Inline base64 is never re-embedded into a capsule.
	inline, _ := part["inlineData"].(map[string]any)
	ev.Portability = capsule.PortabilityOmitted
	ev.Reason = reasonQwenInlineAttachment
	meta := map[string]string{"source": "inline", "unavailable": "true"}
	if data := asString(inline["data"]); data != "" {
		sum := sha256.Sum256([]byte(data))
		meta["inline_sha256_prefix"] = hex.EncodeToString(sum[:8])
	}
	ev.Blocks = []capsule.Block{{
		Type: capsule.BlockTypeAttachment,
		MIME: firstNonEmpty(asString(inline["mimeType"]), "application/octet-stream"),
		Meta: meta,
	}}
	ev.ContentHash = hashQwenContent(ev)
	return ev
}

// parseQwenSystemRecord maps a system record to a harness event. System
// payloads are telemetry, attribution snapshots, and UI state; their bodies are
// never copied into a capsule, because they carry workspace file paths and no
// conversational value. Only a custom title and a compaction summary have text
// worth keeping.
func parseQwenSystemRecord(
	b Boundary,
	rec qwenRecord,
	order *int,
	ts time.Time,
) (capsule.Event, int) {
	subtype := strings.ToLower(strings.TrimSpace(asString(rec.raw["subtype"])))
	ev := baseQwenEvent(b, rec, 0, order, ts, firstNonEmpty(subtype, "system"))
	ev.Actor = capsule.ActorHarness
	payload, _ := rec.raw["systemPayload"].(map[string]any)
	truncated := 0

	addText := func(text string) {
		if text == "" {
			return
		}
		block := TextBlock(text)
		if len(block.Text) > capsule.MaxTextBlockBytes {
			block = TruncateBlock(block, capsule.MaxTextBlockBytes)
			truncated++
			ev.Truncated = true
		}
		ev.Blocks = []capsule.Block{block}
	}

	switch subtype {
	case "chat_compression":
		ev.Kind = capsule.KindSummary
		ev.Portability = capsule.PortabilitySummarized
		ev.Reason = reasonQwenCompaction
		addText(asString(payload["summary"]))
	case "rewind":
		ev.Kind = capsule.KindCheckpoint
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = reasonQwenRewindMarker
	case "custom_title":
		ev.Kind = capsule.KindMetadata
		ev.Portability = capsule.PortabilityReferenced
		ev.Reason = reasonQwenHarnessMetadata
		addText(asString(payload["customTitle"]))
	default:
		if _, known := qwenKnownSubtypes[subtype]; known || subtype == "" {
			ev.Kind = capsule.KindMetadata
			ev.Portability = capsule.PortabilityReferenced
			ev.Reason = reasonQwenHarnessMetadata
		} else {
			actor, kind, portability, reason := ClassifyUnknown(subtype)
			ev.Actor = actor
			ev.Kind = kind
			ev.Portability = portability
			ev.Reason = reason
		}
	}
	ev.ContentHash = hashQwenContent(ev)
	return ev, truncated
}

// qwenKnownSubtypes mirrors the vendor's KNOWN_RECORD_SUBTYPES. A subtype
// outside it comes from a Qwen release this reader has not seen, and is
// classified unknown rather than silently folded into metadata.
var qwenKnownSubtypes = map[string]struct{}{
	"agent_bootstrap":           {},
	"agent_launch_prompt":       {},
	"agent_retry":               {},
	"at_command":                {},
	"attribution_snapshot":      {},
	"chat_compression":          {},
	"cron":                      {},
	"custom_title":              {},
	"file_history_snapshot":     {},
	"goal_runtime":              {},
	"goal_state":                {},
	"mid_turn_user_message":     {},
	"notification":              {},
	"parent_session":            {},
	"realtime_message":          {},
	"rewind":                    {},
	"session_artifact_event":    {},
	"session_artifact_snapshot": {},
	"session_source":            {},
	"slash_command":             {},
	"ui_telemetry":              {},
}

func qwenParts(raw map[string]any) []map[string]any {
	message, ok := raw["message"].(map[string]any)
	if !ok {
		return nil
	}
	values, ok := message["parts"].([]any)
	if !ok {
		return nil
	}
	parts := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if part, ok := value.(map[string]any); ok {
			parts = append(parts, part)
		}
	}
	return parts
}

func qwenResponseText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return compactJSON(typed)
	}
}

func baseQwenEvent(
	b Boundary,
	rec qwenRecord,
	subIndex int,
	order *int,
	ts time.Time,
	nativeType string,
) capsule.Event {
	src := capsule.SourcePointer{
		Agent:      qwenAgentName,
		SessionID:  b.SessionID,
		RecordKey:  rec.uuid,
		ByteOffset: int64(rec.lineNumber),
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

func hashQwenContent(ev capsule.Event) string {
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

func parseQwenTime(value any) time.Time {
	switch typed := value.(type) {
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

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
