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
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	openCodeAdapterVersion         = "1"
	openCodeKnownSessionVersion    = "1"
	reasonSourceBodiesUnavailable  = "source_bodies_unavailable"
	openCodeMetadataBoundaryPrefix = "opencode://metadata/"
	openCodeMaxJSONFileBytes       = sessionindex.MaxJSONLineBytes
	openCodeConversationComponent  = "conversation"
)

func init() {
	_ = Register(NewOpenCodeReader(nil))
}

// OpenCodeReader parses OpenCode MessageV2 storage into canonical events.
//
// Two-tier order: (1) on-disk storage under <data>/storage/message/… plus
// parts; (2) metadata fallback via `opencode session list --format json` when
// the message tree is absent or unrecognized. SQLite-only installs fail closed
// to the metadata path — this reader never invents a SQL schema.
//
// Windows uses the same XDG data root as Unix
// (%USERPROFILE%\.local\share\opencode), not %LOCALAPPDATA%.
type OpenCodeReader struct {
	// DataRoot is Global.Path.data (…/opencode). Empty resolves from the
	// environment. Tests inject a fixture root.
	DataRoot string
	// Runner lists sessions for the metadata fallback. Nil uses
	// sessionindex.ExecCommandRunner.
	Runner sessionindex.CommandRunner
	Getenv func(string) string
	Home   func() (string, error)
}

// NewOpenCodeReader returns a reader with optional command runner injection.
func NewOpenCodeReader(runner sessionindex.CommandRunner) *OpenCodeReader {
	return &OpenCodeReader{Runner: runner}
}

func (r *OpenCodeReader) Name() string { return sessionindex.AgentOpenCode }

// Probe reports whether OpenCode storage or metadata fallback can serve record.
func (r *OpenCodeReader) Probe(ctx context.Context, record sessionindex.Record) (Compatibility, error) {
	if record.ID == "" {
		return CompatibilityUnsupported, nil
	}
	storageRoot := r.storageRoot()
	messageDir := filepath.Join(storageRoot, "message", record.ID)
	if info, err := os.Stat(messageDir); err == nil && info.IsDir() {
		if err := r.validateStorageSession(storageRoot, record.ID); err != nil {
			// Unrecognized shape → metadata fallback remains available.
			if r.canListMetadata(ctx) {
				return CompatibilitySupported, nil
			}
			return CompatibilityUnsupported, nil
		}
		return CompatibilitySupported, nil
	}
	// No message tree (SQLite-only / absent storage): metadata fallback.
	if r.canListMetadata(ctx) {
		return CompatibilitySupported, nil
	}
	if storageRoot == "" {
		return CompatibilityNotInstalled, nil
	}
	if _, err := os.Stat(storageRoot); err != nil {
		return CompatibilityNotInstalled, nil
	}
	return CompatibilitySupported, nil
}

// Snapshot freezes a storage digest boundary, or a metadata-only sentinel.
func (r *OpenCodeReader) Snapshot(ctx context.Context, record sessionindex.Record) (Boundary, error) {
	if record.ID == "" {
		return Boundary{}, errors.New("transcript: opencode session id is empty")
	}
	storageRoot := r.storageRoot()
	messageDir := filepath.Join(storageRoot, "message", record.ID)
	if info, err := os.Stat(messageDir); err == nil && info.IsDir() {
		if err := r.validateStorageSession(storageRoot, record.ID); err == nil {
			return r.snapshotStorage(record.ID, storageRoot, messageDir)
		}
	}
	// Fail closed on unrecognized / missing message bodies → metadata boundary.
	_ = ctx
	return Boundary{
		Agent:      sessionindex.AgentOpenCode,
		SessionID:  record.ID,
		ByteOffset: 0,
		SizeBytes:  0,
		SHA256:     metadataBoundaryDigest(record.ID),
		Partial:    false,
		path:       openCodeMetadataBoundaryPrefix + record.ID,
	}, nil
}

// Parse converts a storage boundary into events, or returns no events for
// metadata-only boundaries (bodies unavailable).
func (r *OpenCodeReader) Parse(_ context.Context, b Boundary) ([]capsule.Event, ParseReport, error) {
	report := ParseReport{ByKind: map[capsule.Kind]int{}}
	if isOpenCodeMetadataBoundary(b) {
		report.Warnings = append(report.Warnings, Warning{
			Agent:   sessionindex.AgentOpenCode,
			Source:  b.Path(),
			Code:    reasonSourceBodiesUnavailable,
			Message: "OpenCode message bodies unavailable; conversation omitted",
		})
		return nil, report, nil
	}
	if b.Path() == "" {
		return nil, report, errors.New("transcript: opencode boundary path is empty")
	}
	events, err := r.parseStorageMessages(b)
	if err != nil {
		return nil, report, err
	}
	report.Events = len(events)
	for _, ev := range events {
		report.ByKind[ev.Kind]++
		if ev.Kind == capsule.KindUnknown {
			report.UnknownRecords++
		}
		if ev.Truncated {
			report.TruncatedBlocks++
		}
	}
	return events, report, nil
}

// MetadataFallbackCapsule builds a minimal validatable capsule whose
// conversation is omitted with reason source_bodies_unavailable.
func (r *OpenCodeReader) MetadataFallbackCapsule(sessionID string) capsule.Capsule {
	if sessionID == "" {
		sessionID = "unknown"
	}
	digest := metadataBoundaryDigest(sessionID)
	fidelity := capsule.Fidelity{
		Overall: capsule.PortabilityOmitted,
		Mode:    capsule.FidelityModeStructuredHandoff,
		Components: []capsule.Component{{
			Name:        openCodeConversationComponent,
			Portability: capsule.PortabilityOmitted,
			Count:       0,
			Reason:      reasonSourceBodiesUnavailable,
		}},
	}
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			Parent: capsule.Parent{
				Agent:          sessionindex.AgentOpenCode,
				SessionID:      sessionID,
				ArtifactSHA256: digest,
				AdapterVersion: openCodeAdapterVersion,
			},
			SchemaVer: capsule.SchemaVersion,
		},
		RawSource: capsule.RawSource{
			Agent:          sessionindex.AgentOpenCode,
			SessionID:      sessionID,
			ArtifactSHA256: digest,
			AdapterVersion: openCodeAdapterVersion,
		},
		Task: capsule.Task{
			Constraints:        capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Decisions:          capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			RejectedApproaches: capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
		},
		Conversation: capsule.Conversation{},
		Capabilities: capsule.CapabilityDiff{
			Source:      map[string]any{},
			Destination: map[string]any{},
		},
		Security: capsule.Security{
			SourceInstructionsAreUntrustedHistory: true,
		},
		Fidelity:   fidelity,
		Projection: capsule.Projection{Policy: "checkpoint"},
	}
}

// ResolveOpenCodeDataRoot returns Global.Path.data for OpenCode on every OS:
// $XDG_DATA_HOME/opencode when set, else <home>/.local/share/opencode.
// Windows matches vendor XDG layout (not %LOCALAPPDATA%).
func ResolveOpenCodeDataRoot(getenv func(string) string, home func() (string, error)) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if home == nil {
		home = os.UserHomeDir
	}
	if xdg := strings.TrimSpace(getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "opencode"), nil
	}
	homeDir, err := home()
	if err != nil {
		return "", err
	}
	if homeDir == "" {
		return "", errors.New("transcript: empty home directory")
	}
	return filepath.Join(homeDir, ".local", "share", "opencode"), nil
}

func (r *OpenCodeReader) storageRoot() string {
	data := strings.TrimSpace(r.DataRoot)
	if data == "" {
		resolved, err := ResolveOpenCodeDataRoot(r.Getenv, r.Home)
		if err != nil {
			return ""
		}
		data = resolved
	}
	return filepath.Join(data, "storage")
}

func (r *OpenCodeReader) runner() sessionindex.CommandRunner {
	if r.Runner != nil {
		return r.Runner
	}
	return sessionindex.ExecCommandRunner{}
}

func (r *OpenCodeReader) canListMetadata(ctx context.Context) bool {
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.runner().Run(runCtx, "opencode", "session", "list", "--format", "json")
	return err == nil
}

func (r *OpenCodeReader) validateStorageSession(storageRoot, sessionID string) error {
	sessionPath, err := findOpenCodeSessionFile(storageRoot, sessionID)
	if err != nil {
		return err
	}
	raw, err := readBoundedJSONFile(sessionPath)
	if err != nil {
		return err
	}
	var session openCodeSessionFile
	if err := json.Unmarshal(raw, &session); err != nil {
		return fmt.Errorf("decode session: %w", err)
	}
	if session.ID != "" && session.ID != sessionID {
		return fmt.Errorf("session id mismatch: %q != %q", session.ID, sessionID)
	}
	version := strings.TrimSpace(session.Version)
	if version == "" {
		version = openCodeKnownSessionVersion
	}
	if version != openCodeKnownSessionVersion {
		return fmt.Errorf("unrecognized OpenCode session version %q", version)
	}

	entries, err := os.ReadDir(filepath.Join(storageRoot, "message", sessionID))
	if err != nil {
		return err
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		found = true
		path := filepath.Join(storageRoot, "message", sessionID, entry.Name())
		msgRaw, err := readBoundedJSONFile(path)
		if err != nil {
			return err
		}
		var msg openCodeMessageInfo
		if err := json.Unmarshal(msgRaw, &msg); err != nil {
			return fmt.Errorf("decode message %s: %w", entry.Name(), err)
		}
		if err := msg.validate(); err != nil {
			return fmt.Errorf("message %s: %w", entry.Name(), err)
		}
		if msg.SessionID != "" && msg.SessionID != sessionID {
			return fmt.Errorf("message %s sessionID mismatch", entry.Name())
		}
	}
	if !found {
		return errors.New("no message files in storage/message tree")
	}
	return nil
}

func (r *OpenCodeReader) snapshotStorage(sessionID, storageRoot, messageDir string) (Boundary, error) {
	paths, total, err := openCodeArtifactPaths(storageRoot, sessionID)
	if err != nil {
		return Boundary{}, err
	}
	digest, err := digestOrderedFiles(paths)
	if err != nil {
		return Boundary{}, err
	}
	info, err := os.Stat(messageDir)
	if err != nil {
		return Boundary{}, err
	}
	return Boundary{
		Agent:      sessionindex.AgentOpenCode,
		SessionID:  sessionID,
		ByteOffset: total,
		SizeBytes:  total,
		SHA256:     digest,
		ModTimeNS:  info.ModTime().UnixNano(),
		path:       messageDir,
	}, nil
}

func (r *OpenCodeReader) parseStorageMessages(b Boundary) ([]capsule.Event, error) {
	messageDir := b.Path()
	sessionID := b.SessionID
	entries, err := os.ReadDir(messageDir)
	if err != nil {
		return nil, fmt.Errorf("transcript: read opencode messages: %w", err)
	}
	type msgFile struct {
		id   string
		path string
	}
	files := make([]msgFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		files = append(files, msgFile{id: id, path: filepath.Join(messageDir, entry.Name())})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].id < files[j].id })

	storageRoot := filepath.Dir(filepath.Dir(messageDir)) // …/storage
	events := make([]capsule.Event, 0, len(files))
	for index, file := range files {
		raw, err := readBoundedJSONFile(file.path)
		if err != nil {
			return nil, err
		}
		var msg openCodeMessageInfo
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, fmt.Errorf("decode message %s: %w", file.id, err)
		}
		if err := msg.validate(); err != nil {
			return nil, fmt.Errorf("message %s: %w", file.id, err)
		}
		parts, err := readOpenCodeParts(storageRoot, msg.ID)
		if err != nil {
			return nil, err
		}
		ev, ok := openCodeMessageEvent(msg, parts, sessionID, index)
		if !ok {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

type openCodeSessionFile struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type openCodeMessageInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Time      struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
	ParentID   string `json:"parentID"`
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
	Mode       string `json:"mode"`
	Agent      string `json:"agent"`
	Model      *struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model"`
}

func (m openCodeMessageInfo) validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return errors.New("missing id")
	}
	switch strings.ToLower(strings.TrimSpace(m.Role)) {
	case "user", "assistant":
		return nil
	default:
		return fmt.Errorf("unrecognized role %q", m.Role)
	}
}

type openCodePart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"`
	Text      string `json:"text"`
}

func openCodeMessageEvent(msg openCodeMessageInfo, parts []openCodePart, sessionID string, index int) (capsule.Event, bool) {
	actor := NormalizeActor(msg.Role)
	blocks := make([]capsule.Block, 0, len(parts))
	nativeType := msg.Role
	kind := capsule.KindMessage
	port := capsule.PortabilityExact
	reason := ""

	sort.Slice(parts, func(i, j int) bool { return parts[i].ID < parts[j].ID })
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "text":
			blocks = append(blocks, TextBlock(part.Text))
		default:
			// Unknown part shapes stay referenced — never guess tool schemas.
			actor, kind, port, reason = ClassifyUnknown(part.Type)
			blocks = append(blocks, RefBlock("opencode-part:"+part.ID))
			nativeType = part.Type
		}
	}
	if len(blocks) == 0 && port == capsule.PortabilityExact {
		// Message Info without parts: still emit an empty exact message so
		// ordering stays visible; empty text is honest.
		blocks = append(blocks, TextBlock(""))
	}

	src := capsule.SourcePointer{
		Agent:     sessionindex.AgentOpenCode,
		SessionID: sessionID,
		RecordKey: msg.ID,
		Index:     index,
	}
	ev := capsule.Event{
		ID:          capsule.EventID(src),
		Order:       index,
		Actor:       actor,
		Kind:        kind,
		NativeType:  nativeType,
		Blocks:      blocks,
		Portability: port,
		Reason:      reason,
		ContentHash: openCodeContentHash(blocks),
		Source:      src,
	}
	if msg.Time.Created > 0 {
		ev.Timestamp = time.UnixMilli(msg.Time.Created).UTC()
	}
	return ev, true
}

func readOpenCodeParts(storageRoot, messageID string) ([]openCodePart, error) {
	dir := filepath.Join(storageRoot, "part", messageID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	parts := make([]openCodePart, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := readBoundedJSONFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var part openCodePart
		if err := json.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("decode part %s: %w", entry.Name(), err)
		}
		parts = append(parts, part)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].ID < parts[j].ID })
	return parts, nil
}

func findOpenCodeSessionFile(storageRoot, sessionID string) (string, error) {
	sessionRoot := filepath.Join(storageRoot, "session")
	entries, err := os.ReadDir(sessionRoot)
	if err != nil {
		return "", err
	}
	name := sessionID + ".json"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(sessionRoot, entry.Name(), name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("opencode session file for %q not found", sessionID)
}

func openCodeArtifactPaths(storageRoot, sessionID string) ([]string, int64, error) {
	var paths []string
	var total int64
	add := func(path string) error {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, path)
		total += info.Size()
		return nil
	}

	if sessionPath, err := findOpenCodeSessionFile(storageRoot, sessionID); err == nil {
		if err := add(sessionPath); err != nil {
			return nil, 0, err
		}
	}

	messageDir := filepath.Join(storageRoot, "message", sessionID)
	msgEntries, err := os.ReadDir(messageDir)
	if err != nil {
		return nil, 0, err
	}
	msgIDs := make([]string, 0, len(msgEntries))
	for _, entry := range msgEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(messageDir, entry.Name())
		if err := add(path); err != nil {
			return nil, 0, err
		}
		msgIDs = append(msgIDs, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(msgIDs)
	for _, msgID := range msgIDs {
		partDir := filepath.Join(storageRoot, "part", msgID)
		partEntries, err := os.ReadDir(partDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, 0, err
		}
		for _, entry := range partEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			if err := add(filepath.Join(partDir, entry.Name())); err != nil {
				return nil, 0, err
			}
		}
	}
	sort.Strings(paths)
	return paths, total, nil
}

func digestOrderedFiles(paths []string) (string, error) {
	h := sha256.New()
	for _, path := range paths {
		raw, err := readBoundedJSONFile(path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.Base(path)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(raw)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readBoundedJSONFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(openCodeMaxJSONFileBytes)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(raw) > openCodeMaxJSONFileBytes {
		return nil, fmt.Errorf("transcript: opencode file %s exceeds %d-byte read limit", filepath.Base(path), openCodeMaxJSONFileBytes)
	}
	return raw, nil
}

func isOpenCodeMetadataBoundary(b Boundary) bool {
	return strings.HasPrefix(b.Path(), openCodeMetadataBoundaryPrefix)
}

func metadataBoundaryDigest(sessionID string) string {
	sum := sha256.Sum256([]byte("opencode-metadata\x00" + sessionID))
	return hex.EncodeToString(sum[:])
}

func openCodeContentHash(blocks []capsule.Block) string {
	h := sha256.New()
	for _, block := range blocks {
		_, _ = h.Write([]byte(block.Type))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(block.Text))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(block.Ref))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}
