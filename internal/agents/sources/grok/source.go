// Package grok ports Grok Build's F1 index source onto the shared hometree scanner.
package grok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const sessionGlob = "sessions/**/summary.json"

// Excluded skips Grok subagent trees.
var Excluded = []string{"subagents"}

// Source discovers Grok Build sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Grok index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentGrok }

// Scan walks summary.json files and maps each session directory to a record.
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
		sessionDir := filepath.Dir(file.Path)
		record, warnings, parseErr := parseSession(sessionDir)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentGrok,
				Source:  sessionDir,
				Code:    "session_read_failed",
				Message: "Grok session could not be read; other sessions remain available",
			})
			continue
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
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
		RootEnv:     "GROK_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "sessions",
		SessionGlob: sessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".grok")}
	}
	return cfg
}

type summary struct {
	Info struct {
		ID  string `json:"id"`
		CWD string `json:"cwd"`
	} `json:"info"`
	SessionSummary    string `json:"session_summary"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	NumMessages       int    `json:"num_messages"`
	NumChatMessages   int    `json:"num_chat_messages"`
	ChatFormatVersion *int   `json:"chat_format_version"`
}

func parseSession(sessionDir string) (sessionindex.Record, []sessionindex.Warning, error) {
	summaryPath := filepath.Join(sessionDir, "summary.json")
	item, err := readSummary(summaryPath)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	var warnings []sessionindex.Warning
	if item.ChatFormatVersion != nil {
		version := *item.ChatFormatVersion
		if version != 0 && version != 1 {
			return sessionindex.Record{}, nil, fmt.Errorf("unsupported chat_format_version %d", version)
		}
	}

	id := strings.TrimSpace(item.Info.ID)
	if id == "" {
		id = filepath.Base(sessionDir)
	}
	workspace := strings.TrimSpace(item.Info.CWD)
	if workspace == "" {
		workspace = decodeWorkspace(filepath.Base(filepath.Dir(sessionDir)), sessionDir)
	}
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}

	_, authorityInfo, err := authorityFile(sessionDir)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}

	prompts, messageCount, files, extractWarnings := extractIndexContent(sessionDir)
	warnings = append(warnings, extractWarnings...)
	if item.NumChatMessages > 0 {
		messageCount = item.NumChatMessages
	} else if item.NumMessages > 0 && messageCount == 0 {
		messageCount = item.NumMessages
	}

	updatedAt := parseTime(item.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = authorityInfo.ModTime().UTC()
	}
	title := sessionindex.SafePreview(item.SessionSummary)
	if title == "" {
		title = id
	}
	preview := firstPromptPreview(sessionDir)

	// `grok --resume` and `grok --resume --fork-session` address a session by
	// UUID. A recorded id of any other shape would be matched as a title, so
	// such a session stays read-only instead of being resumed by name.
	resumable := sessionindex.IsGrokSessionID(id)
	readOnlyReason := ""
	if !resumable {
		readOnlyReason = sessionindex.GrokTitleAddressableReason
	}

	for index := range warnings {
		warnings[index].Agent = sessionindex.AgentGrok
		warnings[index].SessionID = id
		warnings[index].Source = sessionDir
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentGrok, id),
		ID:             id,
		Agent:          sessionindex.AgentGrok,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updatedAt,
		SizeBytes:      authorityInfo.Size(),
		MessageCount:   messageCount,
		PromptPreview:  preview,
		Files:          files,
		CanResume:      resumable,
		CanFork:        resumable,
		ReadOnlyReason: readOnlyReason,
		SourcePath:     sessionDir,
		SourceModTime:  authorityInfo.ModTime().UnixNano(),
		SourceSize:     authorityInfo.Size(),
		SearchText: sessionindex.BuildSearchText(
			id, title, project, workspace, prompts.String(), strings.Join(files, " "),
		),
	}, warnings, nil
}

func readSummary(path string) (summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return summary{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return summary{}, err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return summary{}, fmt.Errorf("grok summary exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var item summary
	if err := json.Unmarshal(data, &item); err != nil {
		return summary{}, err
	}
	return item, nil
}

func authorityFile(sessionDir string) (string, os.FileInfo, error) {
	for _, name := range []string{"updates.jsonl", "chat_history.jsonl", "summary.json"} {
		path := filepath.Join(sessionDir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, info, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", nil, err
		}
	}
	info, err := os.Stat(sessionDir)
	if err != nil {
		return "", nil, err
	}
	return sessionDir, info, nil
}

func extractIndexContent(sessionDir string) (sources.BoundedText, int, []string, []sessionindex.Warning) {
	var prompts sources.BoundedText
	fileSet := make(map[string]struct{})
	messageCount := 0
	var warnings []sessionindex.Warning

	historyPath := filepath.Join(sessionDir, "chat_history.jsonl")
	if _, err := os.Stat(historyPath); err == nil {
		_, _ = hometree.ReadJSONL(historyPath, func(_ int, line []byte) error {
			var item map[string]any
			if json.Unmarshal(line, &item) != nil {
				return nil
			}
			kind := strings.ToLower(sources.FirstString(item, "type"))
			switch kind {
			case "user":
				messageCount++
				if sources.FirstString(item, "synthetic_reason") != "" {
					return nil
				}
				prompts.Add(sources.ExtractTextContent(item["content"]))
			case "assistant":
				messageCount++
				collectToolFiles(item, fileSet)
			case "tool_result", "backend_tool_call":
				collectToolFiles(item, fileSet)
			}
			return nil
		})
	}

	requestsDir := filepath.Join(sessionDir, "compaction_requests")
	entries, err := os.ReadDir(requestsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				continue
			}
			path := filepath.Join(requestsDir, entry.Name())
			items, readErr := readCompactionRequestHistory(path)
			if readErr != nil {
				warnings = append(warnings, sessionindex.Warning{
					Code:    "compaction_request_read_failed",
					Message: "Grok compaction request could not be read; indexed post-compact history only",
				})
				continue
			}
			for _, item := range items {
				kind := strings.ToLower(sources.FirstString(item, "type"))
				if kind != "user" || sources.FirstString(item, "synthetic_reason") != "" {
					continue
				}
				prompts.Add(sources.ExtractTextContent(item["content"]))
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warnings = append(warnings, sessionindex.Warning{
			Code:    "compaction_requests_unreadable",
			Message: "Grok compaction_requests directory could not be listed",
		})
	}

	return prompts, messageCount, sources.NormalizedFileMap(fileSet), warnings
}

func firstPromptPreview(sessionDir string) string {
	historyPath := filepath.Join(sessionDir, "chat_history.jsonl")
	preview := ""
	_, _ = hometree.ReadJSONL(historyPath, func(_ int, line []byte) error {
		if preview != "" {
			return nil
		}
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			return nil
		}
		if strings.ToLower(sources.FirstString(item, "type")) != "user" {
			return nil
		}
		if sources.FirstString(item, "synthetic_reason") != "" {
			return nil
		}
		preview = sessionindex.SafePreview(sources.ExtractTextContent(item["content"]))
		return nil
	})
	if preview != "" {
		return preview
	}
	requestsDir := filepath.Join(sessionDir, "compaction_requests")
	entries, err := os.ReadDir(requestsDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		items, readErr := readCompactionRequestHistory(filepath.Join(requestsDir, entry.Name()))
		if readErr != nil {
			continue
		}
		for _, item := range items {
			if strings.ToLower(sources.FirstString(item, "type")) != "user" {
				continue
			}
			if sources.FirstString(item, "synthetic_reason") != "" {
				continue
			}
			return sessionindex.SafePreview(sources.ExtractTextContent(item["content"]))
		}
	}
	return ""
}

func readCompactionRequestHistory(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return nil, fmt.Errorf("compaction request exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	raw, ok := payload["chat_history"].([]any)
	if !ok {
		return nil, nil
	}
	return sources.MapsFromAny(raw), nil
}

func collectToolFiles(item map[string]any, files map[string]struct{}) {
	sources.CollectToolFiles(item, files)
	if calls, ok := item["tool_calls"].([]any); ok {
		for _, raw := range calls {
			if call, ok := raw.(map[string]any); ok {
				sources.CollectToolFiles(call, files)
			}
		}
	}
}

func decodeWorkspace(encodedDir, sessionDir string) string {
	if decoded, err := url.PathUnescape(encodedDir); err == nil && looksAbsolute(decoded) {
		return decoded
	}
	for _, candidate := range []string{
		filepath.Join(sessionDir, ".cwd"),
		filepath.Join(filepath.Dir(sessionDir), ".cwd"),
	} {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(data))
		if value != "" {
			return value
		}
	}
	return ""
}

func looksAbsolute(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/") {
		return true
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	return strings.HasPrefix(path, `\\`)
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
