// Package qwen discovers Qwen Code sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 dual-platform probes:
//
//	~/.qwen/projects/<slug>/chats/<uuid-v4>.jsonl
//
// First-line keys on both platforms: cwd, message, parentUuid, provenance,
// sessionId, timestamp, type, uuid, version. macOS also writes
// <uuid-v4>-runtime.json sidecars; those are not conversations.
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

// Excluded keeps credentials, config, and the self-updater npm tree out of the walk.
var Excluded = []string{
	"settings.json",
	".env",
	"**/.env",
	"skills",
	"extension-store",
	"updates",
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
		text := ""
		if message, ok := item["message"].(map[string]any); ok {
			text = sources.ExtractTextContent(message["content"])
		}
		if text == "" {
			text = sources.ExtractTextContent(item["message"])
		}
		if text == "" {
			text = sources.ExtractTextContent(item["content"])
		}
		switch kind {
		case "user", "human":
			out.messages++
			out.prompts.Add(text)
			if out.firstPrompt == "" {
				out.firstPrompt = sessionindex.SafePreview(text)
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

func collectFiles(item map[string]any, files map[string]struct{}) {
	if len(files) >= sessionindex.MaxFileReferences {
		return
	}
	for _, key := range []string{"file", "path", "filePath"} {
		if value := strings.TrimSpace(sources.FirstString(item, key)); value != "" && !strings.ContainsAny(value, "\r\n\x00") {
			files[value] = struct{}{}
		}
	}
}
