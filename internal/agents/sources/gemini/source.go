// Package gemini ports Gemini CLI's F1 index source onto the shared hometree scanner.
package gemini

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
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	sessionGlob     = "tmp/*/chats/session-*.json*"
	readOnlyReason  = "Gemini CLI sessions are read-only in Phase 2"
	errSubagentText = "gemini subagent session"
)

var errSubagent = errors.New(errSubagentText)

// Excluded skips Gemini subagent trees.
var Excluded = []string{"subagents"}

// maxProjectsBytes bounds the projects.json read. The file is a flat path
// index; anything larger is not it.
const maxProjectsBytes = 4 << 20

// Source discovers Gemini CLI chat records through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Gemini index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentGemini }

// Scan walks Gemini chat files and maps each to a record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	tmpRoot := filepath.Join(root, "tmp")
	workspaces := projectWorkspaces(root)
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		if !geminiSessionFile(file.Path) {
			continue
		}
		record, warnings, parseErr := parseSession(file.Path, tmpRoot, workspaces)
		if errors.Is(parseErr, errSubagent) {
			continue
		}
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentGemini,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Gemini session could not be read; other sessions remain available",
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
		RootEnv:     "GEMINI_CLI_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "tmp",
		SessionGlob: sessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".gemini")}
	}
	return cfg
}

func geminiSessionFile(path string) bool {
	name := filepath.Base(path)
	extension := strings.ToLower(filepath.Ext(name))
	return strings.HasPrefix(strings.ToLower(name), "session-") &&
		(extension == ".json" || extension == ".jsonl") &&
		strings.EqualFold(filepath.Base(filepath.Dir(path)), "chats")
}

type message struct {
	id     string
	kind   string
	prompt string
	files  []string
}

type state struct {
	id          string
	projectHash string
	title       string
	workspace   string
	kind        string
	latest      int64
	messages    []message
}

func parseSession(path, tmpRoot string, workspaces map[string]string) (sessionindex.Record, []sessionindex.Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	st := state{
		id:          strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		projectHash: projectHash(path, tmpRoot),
	}
	var warnings []sessionindex.Warning
	if strings.EqualFold(filepath.Ext(path), ".jsonl") {
		warnings, err = parseJSONL(path, &st)
	} else {
		err = parseLegacyJSON(path, &st)
	}
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	if strings.EqualFold(st.kind, "subagent") {
		return sessionindex.Record{}, nil, errSubagent
	}

	var prompts sources.BoundedText
	firstPrompt := ""
	fileSet := make(map[string]struct{})
	messageCount := 0
	for _, item := range st.messages {
		if item.kind == "user" || item.kind == "gemini" {
			messageCount++
		}
		if item.kind == "user" {
			prompts.Add(item.prompt)
			if firstPrompt == "" {
				firstPrompt = sessionindex.SafePreview(item.prompt)
			}
		}
		for _, file := range item.files {
			sources.AddFilePath(file, fileSet)
		}
	}
	if st.latest == 0 {
		st.latest = info.ModTime().Unix()
	}
	// A chat record often carries only projectHash. Gemini derives that hash
	// from the absolute project path, and projects.json lists those paths, so
	// the workspace is recoverable instead of surfacing a bare digest as the
	// project name.
	if st.workspace == "" && st.projectHash != "" {
		if resolved, ok := workspaces[strings.ToLower(st.projectHash)]; ok {
			st.workspace = resolved
		}
	}
	project := st.projectHash
	if st.workspace != "" {
		project = sources.PortableBase(st.workspace)
	}
	if project == "" {
		project = "unknown"
	}
	preview := firstPrompt
	title := sessionindex.SafePreview(st.title)
	if title == "" {
		title = st.id
	}
	files := sources.NormalizedFileMap(fileSet)
	for index := range warnings {
		warnings[index].Agent = sessionindex.AgentGemini
		warnings[index].SessionID = st.id
		warnings[index].Source = path
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentGemini, st.id),
		ID:             st.id,
		Agent:          sessionindex.AgentGemini,
		Title:          title,
		Project:        project,
		Workspace:      st.workspace,
		UpdatedAt:      time.Unix(st.latest, 0).UTC(),
		SizeBytes:      info.Size(),
		MessageCount:   messageCount,
		PromptPreview:  preview,
		Files:          files,
		ReadOnlyReason: readOnlyReason,
		SourcePath:     path,
		SourceModTime:  info.ModTime().UnixNano(),
		SourceSize:     info.Size(),
		SearchText:     sessionindex.BuildSearchText(st.id, title, project, st.workspace, prompts.String(), strings.Join(files, " ")),
	}, warnings, nil
}

func parseJSONL(path string, st *state) ([]sessionindex.Warning, error) {
	return hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return nil
		}
		if update, ok := event["$set"].(map[string]any); ok {
			applyMetadata(update, st)
			return nil
		}
		if rewindID := sources.FirstString(event, "$rewindTo"); rewindID != "" {
			for index := len(st.messages) - 1; index >= 0; index-- {
				if st.messages[index].id == rewindID {
					st.messages = st.messages[:index+1]
					break
				}
			}
			return nil
		}
		if eventType := strings.ToLower(sources.FirstString(event, "type")); eventType != "" {
			st.messages = append(st.messages, messageFromMap(event))
			st.latest = sources.MaxInt64(st.latest, sources.EventTimestamp(event))
			return nil
		}
		applyMetadata(event, st)
		return nil
	})
}

func parseLegacyJSON(path string, st *state) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return fmt.Errorf("gemini session exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var conversation map[string]any
	if err := json.Unmarshal(data, &conversation); err != nil {
		return err
	}
	applyMetadata(conversation, st)
	if messages, ok := conversation["messages"].([]any); ok {
		for _, raw := range messages {
			if item, ok := raw.(map[string]any); ok {
				st.messages = append(st.messages, messageFromMap(item))
			}
		}
	}
	return nil
}

func applyMetadata(values map[string]any, st *state) {
	if value := sources.FirstString(values, "sessionId", "session_id"); value != "" {
		st.id = value
	}
	if value := sources.FirstString(values, "projectHash", "project_hash"); value != "" {
		st.projectHash = value
	}
	if value := sources.FirstString(values, "summary", "title", "name"); value != "" {
		st.title = value
	}
	if value := sources.FirstString(values, "kind"); value != "" {
		st.kind = value
	}
	if directories, ok := values["directories"].([]any); ok {
		for _, raw := range directories {
			if directory, ok := raw.(string); ok && directory != "" {
				st.workspace = directory
				break
			}
		}
	}
	if value := sources.FirstString(values, "cwd", "directory", "workspace"); value != "" {
		st.workspace = value
	}
	st.latest = sources.MaxInt64(st.latest, sources.EventTimestamp(values))
}

func messageFromMap(values map[string]any) message {
	item := message{
		id:   sources.FirstString(values, "id"),
		kind: strings.ToLower(sources.FirstString(values, "type", "role")),
	}
	if item.kind == "user" {
		item.prompt = sources.ExtractTextContent(values["content"])
	}
	if item.kind == "gemini" || item.kind == "assistant" || item.kind == "model" {
		if calls, ok := values["toolCalls"].([]any); ok {
			fileSet := make(map[string]struct{})
			for _, raw := range calls {
				if call, ok := raw.(map[string]any); ok {
					collectGeminiArgs(call["args"], fileSet)
				}
			}
			item.files = sources.NormalizedFileMap(fileSet)
		}
		item.kind = "gemini"
	}
	return item
}

func collectGeminiArgs(value any, files map[string]struct{}) {
	// Reuse the same structured-file walk as the original source by wrapping
	// args in a synthetic event so CollectToolFiles sees them as input.
	sources.CollectToolFiles(map[string]any{"input": value}, files)
}

// projectWorkspaces maps a Gemini project hash to the absolute path it was
// derived from. Gemini records the hash on each chat but keeps the paths in
// projects.json, so the two are joined here. A missing or malformed file
// yields no mappings rather than an error: this only enriches a record.
func projectWorkspaces(root string) map[string]string {
	path := filepath.Join(root, "projects.json")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxProjectsBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.Projects) == 0 {
		return nil
	}
	out := make(map[string]string, len(doc.Projects))
	for workspace := range doc.Projects {
		if strings.TrimSpace(workspace) == "" {
			continue
		}
		sum := sha256.Sum256([]byte(workspace))
		out[hex.EncodeToString(sum[:])] = workspace
	}
	return out
}

func projectHash(path, tmpRoot string) string {
	relative, err := filepath.Rel(tmpRoot, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}
