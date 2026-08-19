// Package copilot discovers GitHub Copilot CLI sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-16 macOS and 2026-08-17 native Windows probes:
//
//	~/.copilot/session-state/<uuid-v4>/events.jsonl
//
// First-line keys on both platforms: data, id, parentId, timestamp, type.
// Windows also writes session-store.db and per-session session.db; those are
// not parsed. A rename-aside probe showed an old session ID did not return,
// so this is a local file tree, not a rebuild-from-account store.
package copilot

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

// SessionGlob matches one Copilot CLI event log. SQLite sidecars do not match.
const SessionGlob = "session-state/**/events.jsonl"

var requiredKeys = []string{"data", "id", "parentId", "timestamp", "type"}

// Excluded keeps credentials, OAuth fallbacks, and the global SQLite index
// out of the walk.
var Excluded = []string{
	"config.json",
	"mcp-oauth-config",
	"mcp-secrets",
	"session-store.db",
	"session-store.db-shm",
	"session-store.db-wal",
	"logs",
	"command-history-state.json",
	"command-history-state",
}

// Source discovers GitHub Copilot CLI sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Copilot index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentCopilot }

// Scan maps every readable events.jsonl file to one record.
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
		record, parseErr := parseSession(file)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentCopilot,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "GitHub Copilot CLI session could not be read; other sessions remain available",
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
		RootEnv:     "COPILOT_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "session-state",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".copilot")}
	}
	return cfg
}

func parseSession(file hometree.File) (sessionindex.Record, error) {
	parsed, err := readConversation(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	id := strings.TrimSpace(filepath.Base(filepath.Dir(file.Path)))
	if id == "" || id == "." || id == string(filepath.Separator) {
		id = parsed.id
	}
	if id == "" {
		return sessionindex.Record{}, fmt.Errorf("copilot session has no id")
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
		Key:            sessionindex.CompositeReference(sessionindex.AgentCopilot, id),
		ID:             id,
		Agent:          sessionindex.AgentCopilot,
		Title:          sessionindex.SafePreview(title),
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      file.Size,
		MessageCount:   parsed.messages,
		PromptPreview:  sessionindex.SafePreview(parsed.firstPrompt),
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.CopilotReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  file.ModTime.UnixNano(),
		SourceSize:     file.Size,
		SearchText: sessionindex.BuildSearchText(
			id, title, project, workspace, parsed.prompts.String(),
		),
	}, nil
}

type conversation struct {
	id          string
	cwd         string
	firstPrompt string
	messages    int
	updated     time.Time
	prompts     sources.BoundedText
}

func readConversation(path string) (conversation, error) {
	var out conversation
	first := true
	_, err := hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			return nil
		}
		if first {
			for _, key := range requiredKeys {
				if _, ok := item[key]; !ok {
					return fmt.Errorf("unknown copilot layout: missing %s", key)
				}
			}
			first = false
		}
		if out.id == "" {
			out.id = strings.TrimSpace(sources.FirstString(item, "id"))
		}
		if stamp := sources.EventTimestamp(item); stamp != 0 {
			out.updated = time.Unix(stamp, 0).UTC()
		}
		if cwd := eventCWD(item); cwd != "" && out.cwd == "" {
			out.cwd = cwd
		}
		kind := strings.ToLower(sources.FirstString(item, "type"))
		text := eventText(item)
		if strings.Contains(kind, "user") || kind == "human" {
			out.messages++
			out.prompts.Add(text)
			if out.firstPrompt == "" && text != "" {
				out.firstPrompt = sessionindex.SafePreview(text)
			}
		} else if strings.Contains(kind, "assistant") || strings.Contains(kind, "model") {
			out.messages++
		}
		return nil
	})
	if err != nil {
		return conversation{}, err
	}
	if first {
		return conversation{}, fmt.Errorf("copilot conversation is empty")
	}
	return out, nil
}

func eventText(item map[string]any) string {
	data, ok := item["data"].(map[string]any)
	if !ok {
		return sources.ExtractTextContent(item["data"])
	}
	if text := sources.ExtractTextContent(data["content"]); text != "" {
		return text
	}
	if text := sources.FirstString(data, "text", "content", "prompt", "message"); text != "" {
		return text
	}
	return sources.ExtractTextContent(data)
}

func eventCWD(item map[string]any) string {
	data, ok := item["data"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(sources.FirstString(data, "cwd", "workingDirectory", "workspace"))
}
