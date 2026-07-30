package sessionindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ClaudeSource discovers Claude Code's local project JSONL sessions without
// depending on Reinstate project mappings or sync configuration.
type ClaudeSource struct {
	root string
}

func NewClaudeSource(root string) *ClaudeSource {
	return &ClaudeSource{root: root}
}

func (s *ClaudeSource) Name() string { return "claude" }

func (s *ClaudeSource) Scan(ctx context.Context) (ScanResult, error) {
	root, err := resolveClaudeRoot(s.root)
	if err != nil {
		return ScanResult{}, err
	}
	if root == "" {
		return ScanResult{}, nil
	}

	projectsRoot := filepath.Join(root, "projects")
	if _, err := os.Stat(projectsRoot); errors.Is(err, os.ErrNotExist) {
		return ScanResult{}, nil
	} else if err != nil {
		return ScanResult{}, fmt.Errorf("inspect Claude projects directory: %w", err)
	}

	var result ScanResult
	err = filepath.WalkDir(projectsRoot, func(path string, entry os.DirEntry, walkErr error) error {
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
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".jsonl") {
			return nil
		}

		record, warnings, parseErr := parseClaudeSession(path, projectsRoot)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, Warning{
				Agent:   AgentClaude,
				Source:  path,
				Code:    "session_read_failed",
				Message: "Claude session could not be read; other sessions remain available",
			})
			return nil
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
		return nil
	})
	if err != nil {
		return ScanResult{}, fmt.Errorf("scan Claude sessions: %w", err)
	}
	sortRecordsBySourcePath(result.Records)
	return result, nil
}

func resolveClaudeRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if configured := os.Getenv("CLAUDE_CONFIG_DIR"); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	for _, candidate := range []string{
		filepath.Join(home, ".claude"),
		filepath.Join(home, ".config", "claude"),
	} {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
}

func parseClaudeSession(path, projectsRoot string) (Record, []Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Record{}, nil, err
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	project := claudeProjectName(path, projectsRoot)
	var (
		workspace    string
		branch       string
		title        string
		latest       int64
		messageCount int
		prompts      boundedText
		firstPrompt  string
		files        = make(map[string]struct{})
	)

	warnings, err := visitJSONL(path, func(line []byte) {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return
		}
		if value := firstString(event, "sessionId", "session_id"); value != "" {
			id = value
		}
		if value := firstString(event, "cwd", "workingDirectory", "workdir"); value != "" {
			workspace = value
		}
		if value := firstString(event, "gitBranch", "branch"); value != "" {
			branch = value
		}
		eventType := strings.ToLower(firstString(event, "type"))
		if value := firstString(event, "customTitle"); value != "" {
			title = SafePreview(value)
		} else if eventType == "summary" || eventType == "session_meta" || eventType == "metadata" {
			if value := firstString(event, "title", "summary", "name"); value != "" {
				title = SafePreview(value)
			}
		}
		latest = maxInt64(latest, eventTimestamp(event))

		message, _ := event["message"].(map[string]any)
		role := strings.ToLower(firstString(message, "role"))
		if eventType == "user" || eventType == "assistant" || role == "user" || role == "assistant" {
			messageCount++
		}
		if (eventType == "user" || role == "user") && !boolValue(event["isMeta"]) {
			prompt := extractTextContent(message["content"])
			prompts.Add(prompt)
			if firstPrompt == "" {
				firstPrompt = SafePreview(prompt)
			}
		}
		collectToolFiles(event, files)
	})
	if err != nil {
		return Record{}, nil, err
	}

	if latest == 0 {
		latest = info.ModTime().Unix()
	}
	if workspace != "" {
		project = portableBase(workspace)
	}
	preview := firstPrompt
	if title == "" {
		title = id
	}
	fileList := normalizedFileMap(files)
	for index := range warnings {
		warnings[index].Agent = AgentClaude
		warnings[index].SessionID = id
		warnings[index].Source = path
	}

	return Record{
		Key:           CompositeReference(AgentClaude, id),
		ID:            id,
		Agent:         AgentClaude,
		Title:         title,
		Project:       project,
		Workspace:     workspace,
		Branch:        branch,
		UpdatedAt:     time.Unix(latest, 0).UTC(),
		SizeBytes:     info.Size(),
		MessageCount:  messageCount,
		PromptPreview: preview,
		Files:         fileList,
		CanResume:     true,
		CanFork:       true,
		SourcePath:    path,
		SourceModTime: info.ModTime().UnixNano(),
		SourceSize:    info.Size(),
		SearchText:    BuildSearchText(id, title, project, workspace, branch, prompts.String(), strings.Join(fileList, " ")),
	}, warnings, nil
}

func claudeProjectName(path, projectsRoot string) string {
	relative, err := filepath.Rel(projectsRoot, path)
	if err != nil {
		return "unknown"
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 || parts[0] == "" || parts[0] == "." {
		return "unknown"
	}
	return parts[0]
}

func visitJSONL(path string, visit func([]byte)) ([]Warning, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ScanJSONLines(file, MaxJSONLineBytes, func(_ int, line []byte) error {
		visit(line)
		return nil
	})
}

type boundedText struct {
	value string
}

func (b *boundedText) Add(value string) {
	b.value = BuildSearchText(b.value, value)
}

func (b *boundedText) String() string { return b.value }

func extractTextContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var texts []string
		for _, raw := range typed {
			if text, ok := raw.(string); ok {
				texts = append(texts, text)
				continue
			}
			block, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			blockType := strings.ToLower(firstString(block, "type"))
			if blockType != "" && blockType != "text" && blockType != "input_text" {
				continue
			}
			if text := firstString(block, "text"); text != "" {
				texts = append(texts, text)
			}
		}
		return strings.Join(texts, "\n")
	case map[string]any:
		return firstString(typed, "text")
	default:
		return ""
	}
}

func collectToolFiles(event map[string]any, files map[string]struct{}) {
	collectStructuredFileFields(event["input"], files)
	collectStructuredFileFields(event["arguments"], files)
	if message, ok := event["message"].(map[string]any); ok {
		if blocks, ok := message["content"].([]any); ok {
			for _, raw := range blocks {
				block, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				if strings.EqualFold(firstString(block, "type"), "tool_use") {
					collectStructuredFileFields(block["input"], files)
				}
			}
		}
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		collectStructuredFileFields(payload["input"], files)
		collectStructuredFileFields(payload["arguments"], files)
	}
}

func collectStructuredFileFields(value any, files map[string]struct{}) {
	if len(files) >= MaxFileReferences || value == nil {
		return
	}
	if encoded, ok := value.(string); ok {
		var decoded any
		if json.Unmarshal([]byte(encoded), &decoded) != nil {
			return
		}
		value = decoded
	}
	collectStructuredFileValue(value, files)
}

func collectStructuredFileValue(value any, files map[string]struct{}) {
	if len(files) >= MaxFileReferences || value == nil {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isFileField(key) {
				switch fileValue := nested.(type) {
				case string:
					addFilePath(fileValue, files)
				case []any:
					for _, item := range fileValue {
						if path, ok := item.(string); ok {
							addFilePath(path, files)
						}
					}
				}
			}
			collectStructuredFileValue(nested, files)
		}
	case []any:
		for _, nested := range typed {
			collectStructuredFileValue(nested, files)
		}
	}
}

func isFileField(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
	switch normalized {
	case "path", "file", "filename", "filepath", "files", "paths", "targetpath", "sourcepath", "destinationpath":
		return true
	default:
		return false
	}
}

func addFilePath(value string, files map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") || len(files) >= MaxFileReferences {
		return
	}
	files[value] = struct{}{}
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

func eventTimestamp(values map[string]any) int64 {
	for _, key := range []string{"timestamp", "updatedAt", "updated_at", "lastUpdated", "createdAt", "created_at"} {
		if timestamp := parseTimestamp(values[key]); timestamp != 0 {
			return timestamp
		}
	}
	return 0
}

func parseTimestamp(value any) int64 {
	switch typed := value.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.Unix()
		}
		if number, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return normalizeUnix(number)
		}
	case float64:
		return normalizeUnix(int64(typed))
	case json.Number:
		if number, err := typed.Int64(); err == nil {
			return normalizeUnix(number)
		}
	case int64:
		return normalizeUnix(typed)
	case int:
		return normalizeUnix(int64(typed))
	}
	return 0
}

func normalizeUnix(value int64) int64 {
	switch {
	case value > 1_000_000_000_000_000:
		return value / 1_000_000_000
	case value > 1_000_000_000_000:
		return value / 1_000
	default:
		return value
	}
}

func portableBase(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, `\`, "/"), "/")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		path = path[index+1:]
	}
	if path == "" {
		return "unknown"
	}
	return path
}

func normalizedFileMap(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return NormalizeFiles(result)
}

func maxInt64(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}

func sortRecordsBySourcePath(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].SourcePath < records[j].SourcePath
	})
}
