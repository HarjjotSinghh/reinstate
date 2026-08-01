package sessionindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const geminiReadOnlyReason = "Gemini CLI sessions are read-only in Phase 2"

var errGeminiSubagent = errors.New("gemini subagent session")

// GeminiSource reads Gemini CLI's project chat records. Phase 2 intentionally
// exposes discovery/search/inspection only.
type GeminiSource struct {
	root string
}

func NewGeminiSource(root string) *GeminiSource {
	return &GeminiSource{root: root}
}

func (s *GeminiSource) Name() string { return AgentGemini }

func (s *GeminiSource) Scan(ctx context.Context) (ScanResult, error) {
	root, err := resolveGeminiRoot(s.root)
	if err != nil {
		return ScanResult{}, err
	}
	if root == "" {
		return ScanResult{}, nil
	}
	tmpRoot := filepath.Join(root, "tmp")
	if _, err := os.Stat(tmpRoot); errors.Is(err, os.ErrNotExist) {
		return ScanResult{}, nil
	} else if err != nil {
		return ScanResult{}, fmt.Errorf("inspect Gemini session directory: %w", err)
	}

	var result ScanResult
	err = filepath.WalkDir(tmpRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.EqualFold(entry.Name(), "subagents") {
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if !strings.HasPrefix(strings.ToLower(entry.Name()), "session-") ||
			(extension != ".json" && extension != ".jsonl") ||
			!strings.EqualFold(filepath.Base(filepath.Dir(path)), "chats") {
			return nil
		}

		record, warnings, parseErr := parseGeminiSession(path, tmpRoot)
		if errors.Is(parseErr, errGeminiSubagent) {
			return nil
		}
		if parseErr != nil {
			result.Warnings = append(result.Warnings, Warning{
				Agent:   AgentGemini,
				Source:  path,
				Code:    "session_read_failed",
				Message: "Gemini session could not be read; other sessions remain available",
			})
			return nil
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
		return nil
	})
	if err != nil {
		return ScanResult{}, fmt.Errorf("scan Gemini sessions: %w", err)
	}
	sortRecordsBySourcePath(result.Records)
	return result, nil
}

func resolveGeminiRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if configured := os.Getenv("GEMINI_CLI_HOME"); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	candidate := filepath.Join(home, ".gemini")
	if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
		return candidate, nil
	}
	return "", nil
}

type geminiMessage struct {
	id     string
	kind   string
	prompt string
	files  []string
}

type geminiState struct {
	id          string
	projectHash string
	title       string
	workspace   string
	kind        string
	latest      int64
	messages    []geminiMessage
}

func parseGeminiSession(path, tmpRoot string) (Record, []Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Record{}, nil, err
	}
	state := geminiState{
		id:          strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		projectHash: geminiProjectHash(path, tmpRoot),
	}
	var warnings []Warning

	if strings.EqualFold(filepath.Ext(path), ".jsonl") {
		warnings, err = parseGeminiJSONL(path, &state)
	} else {
		err = parseGeminiLegacyJSON(path, &state)
	}
	if err != nil {
		return Record{}, nil, err
	}
	if strings.EqualFold(state.kind, "subagent") {
		return Record{}, nil, errGeminiSubagent
	}

	var prompts boundedText
	firstPrompt := ""
	fileSet := make(map[string]struct{})
	messageCount := 0
	for _, message := range state.messages {
		if message.kind == "user" || message.kind == "gemini" {
			messageCount++
		}
		if message.kind == "user" {
			prompts.Add(message.prompt)
			if firstPrompt == "" {
				firstPrompt = SafePreview(message.prompt)
			}
		}
		for _, file := range message.files {
			addFilePath(file, fileSet)
		}
	}
	if state.latest == 0 {
		state.latest = info.ModTime().Unix()
	}
	project := state.projectHash
	if state.workspace != "" {
		project = portableBase(state.workspace)
	}
	if project == "" {
		project = "unknown"
	}
	preview := firstPrompt
	title := SafePreview(state.title)
	if title == "" {
		title = state.id
	}
	files := normalizedFileMap(fileSet)
	for index := range warnings {
		warnings[index].Agent = AgentGemini
		warnings[index].SessionID = state.id
		warnings[index].Source = path
	}

	return Record{
		Key:            CompositeReference(AgentGemini, state.id),
		ID:             state.id,
		Agent:          AgentGemini,
		Title:          title,
		Project:        project,
		Workspace:      state.workspace,
		UpdatedAt:      time.Unix(state.latest, 0).UTC(),
		SizeBytes:      info.Size(),
		MessageCount:   messageCount,
		PromptPreview:  preview,
		Files:          files,
		ReadOnlyReason: geminiReadOnlyReason,
		SourcePath:     path,
		SourceModTime:  info.ModTime().UnixNano(),
		SourceSize:     info.Size(),
		SearchText:     BuildSearchText(state.id, title, project, state.workspace, prompts.String(), strings.Join(files, " ")),
	}, warnings, nil
}

func parseGeminiJSONL(path string, state *geminiState) ([]Warning, error) {
	return visitJSONL(path, func(line []byte) {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return
		}
		if update, ok := event["$set"].(map[string]any); ok {
			applyGeminiMetadata(update, state)
			return
		}
		if rewindID := firstString(event, "$rewindTo"); rewindID != "" {
			for index := len(state.messages) - 1; index >= 0; index-- {
				if state.messages[index].id == rewindID {
					state.messages = state.messages[:index+1]
					break
				}
			}
			return
		}
		if eventType := strings.ToLower(firstString(event, "type")); eventType != "" {
			state.messages = append(state.messages, geminiMessageFromMap(event))
			state.latest = maxInt64(state.latest, eventTimestamp(event))
			return
		}
		applyGeminiMetadata(event, state)
	})
}

func parseGeminiLegacyJSON(path string, state *geminiState) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > MaxJSONLineBytes {
		return fmt.Errorf("gemini session exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var conversation map[string]any
	if err := json.Unmarshal(data, &conversation); err != nil {
		return err
	}
	applyGeminiMetadata(conversation, state)
	if messages, ok := conversation["messages"].([]any); ok {
		for _, raw := range messages {
			if message, ok := raw.(map[string]any); ok {
				state.messages = append(state.messages, geminiMessageFromMap(message))
			}
		}
	}
	return nil
}

func applyGeminiMetadata(values map[string]any, state *geminiState) {
	if value := firstString(values, "sessionId", "session_id"); value != "" {
		state.id = value
	}
	if value := firstString(values, "projectHash", "project_hash"); value != "" {
		state.projectHash = value
	}
	if value := firstString(values, "summary", "title", "name"); value != "" {
		state.title = value
	}
	if value := firstString(values, "kind"); value != "" {
		state.kind = value
	}
	if directories, ok := values["directories"].([]any); ok {
		for _, raw := range directories {
			if directory, ok := raw.(string); ok && directory != "" {
				state.workspace = directory
				break
			}
		}
	}
	if value := firstString(values, "cwd", "directory", "workspace"); value != "" {
		state.workspace = value
	}
	state.latest = maxInt64(state.latest, eventTimestamp(values))
}

func geminiMessageFromMap(values map[string]any) geminiMessage {
	message := geminiMessage{
		id:   firstString(values, "id"),
		kind: strings.ToLower(firstString(values, "type", "role")),
	}
	if message.kind == "user" {
		message.prompt = extractTextContent(values["content"])
	}
	if message.kind == "gemini" || message.kind == "assistant" || message.kind == "model" {
		if calls, ok := values["toolCalls"].([]any); ok {
			fileSet := make(map[string]struct{})
			for _, raw := range calls {
				if call, ok := raw.(map[string]any); ok {
					collectStructuredFileFields(call["args"], fileSet)
				}
			}
			message.files = normalizedFileMap(fileSet)
		}
		message.kind = "gemini"
	}
	return message
}

func geminiProjectHash(path, tmpRoot string) string {
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
