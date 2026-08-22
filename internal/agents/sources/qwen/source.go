// Package qwen discovers Qwen Code sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 dual-platform probes:
//
//	~/.qwen/projects/<slug>/chats/<uuid-v4>.jsonl
//
// First-line keys on both platforms: cwd, message, parentUuid, provenance,
// sessionId, timestamp, type, uuid, version. Runtime status sidecars are
// written beside the conversation as JSON, not JSONL, and never match the glob.
//
// The keys match Claude Code's; the body does not. `message` is a Gemini
// Content value, {"role":"user"|"model","parts":[…]}, so text lives under
// parts[].text and tool arguments under parts[].functionCall.args.
package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// SessionGlob matches one conversation JSONL file. Runtime sidecars do not match.
const SessionGlob = "projects/**/chats/*.jsonl"

// requiredKeys must appear on the first complete record or the file is unknown layout.
var requiredKeys = []string{"cwd", "sessionId", "timestamp", "type", "uuid"}

// Excluded keeps credentials, config, and the self-updater npm tree out of the
// walk. `subagents` holds per-subagent transcripts under a project bucket; they
// are not sessions and must never be indexed or handed off as one.
var Excluded = []string{
	"settings.json",
	".env",
	"**/.env",
	"skills",
	"extension-store",
	"updates",
	"subagents",
}

// Source discovers Qwen Code sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Qwen index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentQwen }

// Scan maps every readable conversation JSONL file to one record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		if strings.HasSuffix(file.Path, "-runtime.json") {
			continue
		}
		record, parseErr := parseSession(file)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentQwen,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Qwen session could not be read; other sessions remain available",
			})
			continue
		}
		result.Records = append(result.Records, record)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

// Fingerprint summarises the source without opening any file, so an
// unchanged refresh can skip parsing entirely.
func (s *Source) Fingerprint(ctx context.Context) (string, bool, error) {
	return hometree.Fingerprint(ctx, s.config())
}

func (s *Source) config() hometree.Config {
	cfg := hometree.Config{
		Explicit:    s.env.FixtureRoot,
		RootEnv:     "QWEN_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "projects",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".qwen")}
	}
	return cfg
}

func parseSession(file hometree.File) (sessionindex.Record, error) {
	parsed, err := readConversation(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	id := parsed.id
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	}
	workspace := parsed.cwd
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}
	title := parsed.firstPrompt
	if title == "" {
		title = id
	}
	updated := parsed.updated
	if updated.IsZero() {
		updated = file.ModTime.UTC()
	}
	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentQwen, id),
		ID:             id,
		Agent:          sessionindex.AgentQwen,
		Title:          sessionindex.SafePreview(title),
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      file.Size,
		MessageCount:   parsed.messages,
		PromptPreview:  sessionindex.SafePreview(parsed.firstPrompt),
		Files:          parsed.files,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.QwenReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  file.ModTime.UnixNano(),
		SourceSize:     file.Size,
		SearchText: sessionindex.BuildSearchText(
			id, title, project, workspace, parsed.prompts.String(), strings.Join(parsed.files, " "),
		),
	}, nil
}

type conversation struct {
	id          string
	cwd         string
	firstPrompt string
	messages    int
	updated     time.Time
	files       []string
	prompts     sources.BoundedText
}

func readConversation(path string) (conversation, error) {
	var out conversation
	first := true
	fileSet := map[string]struct{}{}
	_, err := hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			return nil
		}
		if first {
			for _, key := range requiredKeys {
				if _, ok := item[key]; !ok {
					return fmt.Errorf("unknown qwen layout: missing %s", key)
				}
			}
			first = false
		}
		if out.id == "" {
			out.id = strings.TrimSpace(sources.FirstString(item, "sessionId"))
		}
		if out.cwd == "" {
			out.cwd = strings.TrimSpace(sources.FirstString(item, "cwd"))
		}
		if stamp := sources.EventTimestamp(item); stamp != 0 {
			out.updated = time.Unix(stamp, 0).UTC()
		}
		kind := strings.ToLower(sources.FirstString(item, "type"))
		text := partsText(item)
		switch kind {
		case "user", "human":
			out.messages++
			if isRealUserTurn(item) {
				out.prompts.Add(text)
				if out.firstPrompt == "" {
					out.firstPrompt = sessionindex.SafePreview(text)
				}
			}
		case "assistant", "ai":
			out.messages++
		}
		collectFiles(item, fileSet)
		return nil
	})
	if err != nil {
		return conversation{}, err
	}
	if first {
		return conversation{}, fmt.Errorf("qwen conversation is empty")
	}
	out.files = sources.NormalizedFileMap(fileSet)
	return out, nil
}

// messageParts returns the Gemini Content parts Qwen records under `message`.
//
// Qwen's top-level record keys match Claude Code's, but the body does not: it
// is {"role":"user"|"model","parts":[…]}, never a Claude content-block array.
// Reading it as Claude's shape finds no text at all, which is how every real
// Qwen session was indexed with an empty title, an empty prompt preview, and
// search text that matched nothing the operator had typed.
func messageParts(item map[string]any) []map[string]any {
	message, ok := item["message"].(map[string]any)
	if !ok {
		return nil
	}
	values, ok := message["parts"].([]any)
	if !ok {
		return nil
	}
	return sources.MapsFromAny(values)
}

// partsText joins the plain-text parts of a record. functionCall and
// functionResponse parts are structure, not prose, and stay out of the preview.
func partsText(item map[string]any) string {
	var texts []string
	for _, part := range messageParts(item) {
		if sources.BoolValue(part["thought"]) {
			continue
		}
		if text := sources.FirstString(part, "text"); text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}

// isRealUserTurn reports whether a type:"user" record is a person speaking.
// Qwen also writes cron prompts, notifications, and goal-runtime messages as
// type:"user" with provenance:"system"; those are harness text and must not
// become a session's title or prompt preview.
func isRealUserTurn(item map[string]any) bool {
	switch strings.ToLower(sources.FirstString(item, "provenance")) {
	case "", "real_user", "user":
		return true
	}
	return strings.EqualFold(sources.FirstString(item, "subtype"), "mid_turn_user_message")
}

func collectFiles(item map[string]any, files map[string]struct{}) {
	if len(files) >= sessionindex.MaxFileReferences {
		return
	}
	for _, key := range []string{"file", "path", "filePath"} {
		if value := strings.TrimSpace(sources.FirstString(item, key)); value != "" && !strings.ContainsAny(value, "\r\n\x00") {
			files[value] = struct{}{}
		}
	}
	// Real file references live in tool arguments: {"functionCall":{"args":{…}}}.
	for _, part := range messageParts(item) {
		call, ok := part["functionCall"].(map[string]any)
		if !ok {
			continue
		}
		sources.CollectToolFiles(map[string]any{"arguments": call["args"]}, files)
	}
}
