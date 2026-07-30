package sessionindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodexSource discovers local Codex CLI rollout files without requiring a
// configured Reinstate project.
type CodexSource struct {
	root string
}

func NewCodexSource(root string) *CodexSource {
	return &CodexSource{root: root}
}

func (s *CodexSource) Name() string { return AgentCodex }

func (s *CodexSource) Scan(ctx context.Context) (ScanResult, error) {
	root, err := resolveCodexRoot(s.root)
	if err != nil {
		return ScanResult{}, err
	}
	if root == "" {
		return ScanResult{}, nil
	}
	sessionsRoot := filepath.Join(root, "sessions")
	if _, err := os.Stat(sessionsRoot); errors.Is(err, os.ErrNotExist) {
		return ScanResult{}, nil
	} else if err != nil {
		return ScanResult{}, fmt.Errorf("inspect Codex sessions directory: %w", err)
	}

	var result ScanResult
	err = filepath.WalkDir(sessionsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".jsonl") {
			return nil
		}
		record, warnings, parseErr := parseCodexSession(path)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, Warning{
				Agent:   AgentCodex,
				Source:  path,
				Code:    "session_read_failed",
				Message: "Codex session could not be read; other sessions remain available",
			})
			return nil
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
		return nil
	})
	if err != nil {
		return ScanResult{}, fmt.Errorf("scan Codex sessions: %w", err)
	}
	sortRecordsBySourcePath(result.Records)
	return result, nil
}

func resolveCodexRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if configured := os.Getenv("CODEX_HOME"); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	for _, candidate := range []string{
		filepath.Join(home, ".codex"),
		filepath.Join(home, ".config", "codex"),
	} {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
}

func parseCodexSession(path string) (Record, []Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Record{}, nil, err
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var (
		workspace       string
		branch          string
		title           string
		latest          int64
		messageCount    int
		directPrompts   boundedText
		fallbackPrompts boundedText
		directPreview   string
		fallbackPreview string
		files           = make(map[string]struct{})
	)

	warnings, err := visitJSONL(path, func(line []byte) {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return
		}
		payload, _ := event["payload"].(map[string]any)
		eventType := strings.ToLower(firstString(event, "type"))
		for index, values := range []map[string]any{event, payload} {
			if values == nil {
				continue
			}
			if value := firstString(values, "id", "sessionId", "session_id"); value != "" &&
				(strings.EqualFold(firstString(event, "type"), "session_meta") || index == 0) {
				id = value
			}
			if value := firstString(values, "cwd", "workingDirectory", "workdir"); value != "" {
				workspace = value
			}
			if value := firstString(values, "gitBranch", "branch"); value != "" {
				branch = value
			}
			if eventType == "session_meta" || eventType == "metadata" || eventType == "summary" {
				if value := firstString(values, "title", "summary"); value != "" {
					title = SafePreview(value)
				}
			}
			latest = maxInt64(latest, eventTimestamp(values))
		}
		if git, ok := payload["git"].(map[string]any); ok {
			if value := firstString(git, "branch"); value != "" {
				branch = value
			}
		}

		if prompt, counted := codexUserPrompt(event, payload); counted {
			messageCount++
			if strings.EqualFold(firstString(event, "type"), "event_msg") &&
				strings.EqualFold(firstString(payload, "type"), "user_message") {
				directPrompts.Add(prompt)
				if directPreview == "" {
					directPreview = SafePreview(prompt)
				}
			} else {
				fallbackPrompts.Add(prompt)
				if fallbackPreview == "" {
					fallbackPreview = SafePreview(prompt)
				}
			}
		} else if codexAssistantMessage(event, payload) {
			messageCount++
		}
		collectToolFiles(event, files)
	})
	if err != nil {
		return Record{}, nil, err
	}

	if latest == 0 {
		latest = info.ModTime().Unix()
	}
	project := portableBase(workspace)
	if workspace == "" {
		project = "unknown"
	}
	promptText := directPrompts.String()
	preview := directPreview
	if promptText == "" {
		// Current rollouts emit the human-authored text as event_msg /
		// user_message. Older layouts used role=user message records. Prefer
		// the direct event stream so synthesized instruction/environment
		// records cannot enter the index when both representations exist.
		promptText = fallbackPrompts.String()
		preview = fallbackPreview
	}
	if title == "" {
		title = id
	}
	fileList := normalizedFileMap(files)
	for index := range warnings {
		warnings[index].Agent = AgentCodex
		warnings[index].SessionID = id
		warnings[index].Source = path
	}

	return Record{
		Key:           CompositeReference(AgentCodex, id),
		ID:            id,
		Agent:         AgentCodex,
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
		SearchText:    BuildSearchText(id, title, project, workspace, branch, promptText, strings.Join(fileList, " ")),
	}, warnings, nil
}

func codexUserPrompt(event, payload map[string]any) (string, bool) {
	eventType := strings.ToLower(firstString(event, "type"))
	payloadType := strings.ToLower(firstString(payload, "type"))

	switch eventType {
	case "response_item":
		if payloadType == "message" && strings.EqualFold(firstString(payload, "role"), "user") {
			return extractTextContent(payload["content"]), true
		}
	case "event_msg":
		if payloadType == "user_message" {
			if message := firstString(payload, "message", "text"); message != "" {
				return message, true
			}
			return extractTextContent(payload["content"]), true
		}
	case "message":
		if strings.EqualFold(firstString(event, "role"), "user") {
			return extractTextContent(event["content"]), true
		}
	}
	if strings.EqualFold(firstString(event, "role"), "user") {
		return extractTextContent(event["content"]), true
	}
	return "", false
}

func codexAssistantMessage(event, payload map[string]any) bool {
	eventType := strings.ToLower(firstString(event, "type"))
	if eventType == "response_item" {
		return strings.EqualFold(firstString(payload, "type"), "message") &&
			strings.EqualFold(firstString(payload, "role"), "assistant")
	}
	if eventType == "event_msg" {
		return strings.EqualFold(firstString(payload, "type"), "agent_message")
	}
	return (eventType == "message" || firstString(event, "role") != "") &&
		strings.EqualFold(firstString(event, "role"), "assistant")
}
