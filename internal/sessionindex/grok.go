package sessionindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GrokSessionIDPattern is the anchored shape a Grok Build session identifier
// must have before Reinstate will put it on a `grok` command line.
//
// `grok --resume [<SESSION_ID_OR_TITLE>]` accepts either an ID or a title, and
// resolves any value that is not UUID-shaped as a title. Titles are neither
// unique nor stable — the vendor documents that duplicates are an ambiguity
// error — so a non-UUID value in that position can address a different session
// than the one Reinstate resolved. Sessions whose recorded ID does not match
// this shape stay read-only rather than being resumed by name.
const GrokSessionIDPattern = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`

// GrokTitleAddressableReason is the read-only contract for a Grok session whose
// recorded identifier is not UUID-shaped.
const GrokTitleAddressableReason = "Grok Build session id is not a UUID; --resume would address it by title"

var grokSessionID = regexp.MustCompile(GrokSessionIDPattern)

// IsGrokSessionID reports whether id is addressable as a Grok session ID rather
// than as a session title.
func IsGrokSessionID(id string) bool { return grokSessionID.MatchString(id) }

// GrokSource discovers Grok Build CLI sessions under <root>/sessions/.
// Sessions are indexed for search/inspect/handoff-from, and resumed natively
// through `grok --resume <uuid>` when the recorded id is UUID-shaped.
type GrokSource struct {
	root string
}

// NewGrokSource constructs a Grok Build local source. An empty root resolves
// via GROK_HOME or ~/.grok.
func NewGrokSource(root string) *GrokSource {
	return &GrokSource{root: root}
}

func (s *GrokSource) Name() string { return AgentGrok }

func (s *GrokSource) Scan(ctx context.Context) (ScanResult, error) {
	root, err := resolveGrokRoot(s.root)
	if err != nil {
		return ScanResult{}, err
	}
	if root == "" {
		return ScanResult{}, nil
	}
	if abs, absErr := filepath.Abs(root); absErr == nil {
		root = abs
	}
	sessionsRoot := filepath.Join(root, "sessions")
	if _, err := os.Stat(sessionsRoot); errors.Is(err, os.ErrNotExist) {
		return ScanResult{}, nil
	} else if err != nil {
		return ScanResult{}, fmt.Errorf("inspect Grok sessions directory: %w", err)
	}

	var result ScanResult
	err = filepath.WalkDir(sessionsRoot, func(path string, entry os.DirEntry, walkErr error) error {
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
		if !strings.EqualFold(entry.Name(), "summary.json") {
			return nil
		}
		sessionDir := filepath.Dir(path)
		record, warnings, parseErr := parseGrokSession(sessionDir)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, Warning{
				Agent:   AgentGrok,
				Source:  sessionDir,
				Code:    "session_read_failed",
				Message: "Grok session could not be read; other sessions remain available",
			})
			return nil
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
		return nil
	})
	if err != nil {
		return ScanResult{}, fmt.Errorf("scan Grok sessions: %w", err)
	}
	sortRecordsBySourcePath(result.Records)
	return result, nil
}

func resolveGrokRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	if configured := os.Getenv("GROK_HOME"); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	candidate := filepath.Join(home, ".grok")
	if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
		return candidate, nil
	}
	return "", nil
}

type grokSummary struct {
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

func parseGrokSession(sessionDir string) (Record, []Warning, error) {
	summaryPath := filepath.Join(sessionDir, "summary.json")
	summary, err := readGrokSummary(summaryPath)
	if err != nil {
		return Record{}, nil, err
	}

	var warnings []Warning
	if summary.ChatFormatVersion != nil {
		version := *summary.ChatFormatVersion
		if version != 0 && version != 1 {
			return Record{}, nil, fmt.Errorf("unsupported chat_format_version %d", version)
		}
	}

	id := strings.TrimSpace(summary.Info.ID)
	if id == "" {
		id = filepath.Base(sessionDir)
	}
	workspace := strings.TrimSpace(summary.Info.CWD)
	if workspace == "" {
		workspace = decodeGrokWorkspace(filepath.Base(filepath.Dir(sessionDir)), sessionDir)
	}
	project := "unknown"
	if workspace != "" {
		project = portableBase(workspace)
	}

	_, authorityInfo, err := grokAuthorityFile(sessionDir)
	if err != nil {
		return Record{}, nil, err
	}

	prompts, messageCount, files, extractWarnings := extractGrokIndexContent(sessionDir)
	warnings = append(warnings, extractWarnings...)
	if summary.NumChatMessages > 0 {
		messageCount = summary.NumChatMessages
	} else if summary.NumMessages > 0 && messageCount == 0 {
		messageCount = summary.NumMessages
	}

	updatedAt := parseGrokTime(summary.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = authorityInfo.ModTime().UTC()
	}
	title := SafePreview(summary.SessionSummary)
	if title == "" {
		title = id
	}
	preview := firstGrokPromptPreview(sessionDir)

	// `grok --resume` and `grok --resume --fork-session` address a session by
	// UUID. A recorded id of any other shape would be matched as a title, so
	// such a session stays read-only instead of being resumed by name.
	resumable := IsGrokSessionID(id)
	readOnlyReason := ""
	if !resumable {
		readOnlyReason = GrokTitleAddressableReason
	}

	for index := range warnings {
		warnings[index].Agent = AgentGrok
		warnings[index].SessionID = id
		warnings[index].Source = sessionDir
	}

	return Record{
		Key:            CompositeReference(AgentGrok, id),
		ID:             id,
		Agent:          AgentGrok,
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
		SearchText: BuildSearchText(
			id, title, project, workspace, prompts.String(), strings.Join(files, " "),
		),
	}, warnings, nil
}

func readGrokSummary(path string) (grokSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return grokSummary{}, err
	}
	defer func() { _ = file.Close() }()

	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return grokSummary{}, err
	}
	if len(data) > MaxJSONLineBytes {
		return grokSummary{}, fmt.Errorf("grok summary exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var summary grokSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return grokSummary{}, err
	}
	return summary, nil
}

func grokAuthorityFile(sessionDir string) (string, os.FileInfo, error) {
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

func extractGrokIndexContent(sessionDir string) (boundedText, int, []string, []Warning) {
	var prompts boundedText
	fileSet := make(map[string]struct{})
	messageCount := 0
	var warnings []Warning

	// Prefer model-facing chat_history for searchable user prompts. Compaction
	// may rewrite it; pre-compact turns are recovered from compaction_requests/.
	historyPath := filepath.Join(sessionDir, "chat_history.jsonl")
	if _, err := os.Stat(historyPath); err == nil {
		_, _ = visitJSONL(historyPath, func(line []byte) {
			var item map[string]any
			if json.Unmarshal(line, &item) != nil {
				return
			}
			kind := strings.ToLower(firstString(item, "type"))
			switch kind {
			case "user":
				messageCount++
				if firstString(item, "synthetic_reason") != "" {
					return
				}
				prompts.Add(extractTextContent(item["content"]))
			case "assistant":
				messageCount++
				collectGrokToolFiles(item, fileSet)
			case "tool_result", "backend_tool_call":
				collectGrokToolFiles(item, fileSet)
			}
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
			items, readErr := readGrokCompactionRequestHistory(path)
			if readErr != nil {
				warnings = append(warnings, Warning{
					Code:    "compaction_request_read_failed",
					Message: "Grok compaction request could not be read; indexed post-compact history only",
				})
				continue
			}
			for _, item := range items {
				kind := strings.ToLower(firstString(item, "type"))
				if kind != "user" || firstString(item, "synthetic_reason") != "" {
					continue
				}
				prompts.Add(extractTextContent(item["content"]))
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warnings = append(warnings, Warning{
			Code:    "compaction_requests_unreadable",
			Message: "Grok compaction_requests directory could not be listed",
		})
	}

	return prompts, messageCount, normalizedFileMap(fileSet), warnings
}

func firstGrokPromptPreview(sessionDir string) string {
	historyPath := filepath.Join(sessionDir, "chat_history.jsonl")
	preview := ""
	_, _ = visitJSONL(historyPath, func(line []byte) {
		if preview != "" {
			return
		}
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			return
		}
		if strings.ToLower(firstString(item, "type")) != "user" {
			return
		}
		if firstString(item, "synthetic_reason") != "" {
			return
		}
		preview = SafePreview(extractTextContent(item["content"]))
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
		items, readErr := readGrokCompactionRequestHistory(filepath.Join(requestsDir, entry.Name()))
		if readErr != nil {
			continue
		}
		for _, item := range items {
			if strings.ToLower(firstString(item, "type")) != "user" {
				continue
			}
			if firstString(item, "synthetic_reason") != "" {
				continue
			}
			return SafePreview(extractTextContent(item["content"]))
		}
	}
	return ""
}

func readGrokCompactionRequestHistory(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxJSONLineBytes {
		return nil, fmt.Errorf("compaction request exceeds %d-byte read limit", MaxJSONLineBytes)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	raw, ok := payload["chat_history"].([]any)
	if !ok {
		return nil, nil
	}
	return mapsFromAny(raw), nil
}

func collectGrokToolFiles(item map[string]any, files map[string]struct{}) {
	collectStructuredFileFields(item["arguments"], files)
	collectStructuredFileFields(item["input"], files)
	if calls, ok := item["tool_calls"].([]any); ok {
		for _, raw := range calls {
			if call, ok := raw.(map[string]any); ok {
				collectStructuredFileFields(call["arguments"], files)
				collectStructuredFileFields(call["input"], files)
				if name := firstString(call, "name"); name != "" {
					_ = name
				}
			}
		}
	}
}

func decodeGrokWorkspace(encodedDir, sessionDir string) string {
	if decoded, err := url.PathUnescape(encodedDir); err == nil && grokLooksAbsolute(decoded) {
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

func grokLooksAbsolute(path string) bool {
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

func parseGrokTime(value string) time.Time {
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
