// Package codex ports Codex CLI's F1 index source onto the shared hometree scanner.
package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const sessionGlob = "sessions/**/*.jsonl"

// Excluded is the Codex adapter credential/cache set.
var Excluded = []string{
	"**/auth.json",
	"**/.codex/auth.json",
	"**/.env",
	"**/cache/**",
}

// Source discovers Codex CLI rollout files through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Codex index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentCodex }

// Scan walks Codex rollout JSONL files and maps each to a record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	_, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		record, warnings, parseErr := parseSession(file.Path)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentCodex,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Codex session could not be read; other sessions remain available",
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
		RootEnv:     "CODEX_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "sessions",
		SessionGlob: sessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".codex"), home.Join(".config", "codex")}
	}
	return cfg
}

func parseSession(path string) (sessionindex.Record, []sessionindex.Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	identityPinned := false
	if fromName := sessionIDFromFilename(path); fromName != "" {
		id = fromName
		identityPinned = true
	}
	var (
		workspace           string
		branch              string
		title               string
		latest              int64
		messageCount        int
		directPrompts       sources.BoundedText
		fallbackPrompts     sources.BoundedText
		directPreview       string
		fallbackPreview     string
		files               = make(map[string]struct{})
		recordedEnvironment environment.RecordedEnvironment
	)

	warnings, err := hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return nil
		}
		payload, _ := event["payload"].(map[string]any)
		eventType := strings.ToLower(sources.FirstString(event, "type"))
		for index, values := range []map[string]any{event, payload} {
			if values == nil {
				continue
			}
			if value := sources.FirstString(values, "id", "sessionId", "session_id"); value != "" &&
				!identityPinned &&
				(strings.EqualFold(sources.FirstString(event, "type"), "session_meta") || index == 0) {
				id = value
			}
			if value := sources.FirstString(values, "cwd", "workingDirectory", "workdir"); value != "" {
				workspace = value
			}
			if value := sources.FirstString(values, "gitBranch", "branch"); value != "" {
				branch = value
			}
			if eventType == "session_meta" || eventType == "metadata" || eventType == "summary" {
				if value := sources.FirstString(values, "title", "summary"); value != "" {
					title = sessionindex.SafePreview(value)
				}
			}
			latest = sources.MaxInt64(latest, sources.EventTimestamp(values))
		}
		if git, ok := payload["git"].(map[string]any); ok {
			if value := sources.FirstString(git, "branch"); value != "" {
				branch = value
				if eventType == "session_meta" {
					recordedEnvironment.Branch = environment.RecordedField{
						Value:      value,
						Provenance: "codex.session_meta.git.branch",
					}
				}
			}
			if eventType == "session_meta" {
				if value := sources.FirstString(git, "repository_url"); value != "" {
					if repositoryID := environment.NormalizeRepositoryID(value); repositoryID != "" {
						recordedEnvironment.RepositoryID = environment.RecordedField{
							Value:      repositoryID,
							Provenance: "codex.session_meta.git.repository_url",
						}
					}
				}
				if value := sources.FirstString(git, "commit_hash"); value != "" {
					if gitHead := environment.NormalizeGitHead(value); gitHead != "" {
						recordedEnvironment.GitHead = environment.RecordedField{
							Value:      gitHead,
							Provenance: "codex.session_meta.git.commit_hash",
						}
					}
				}
			}
		}

		if prompt, counted := userPrompt(event, payload); counted {
			messageCount++
			if strings.EqualFold(sources.FirstString(event, "type"), "event_msg") &&
				strings.EqualFold(sources.FirstString(payload, "type"), "user_message") {
				directPrompts.Add(prompt)
				if directPreview == "" {
					directPreview = sessionindex.SafePreview(prompt)
				}
			} else {
				fallbackPrompts.Add(prompt)
				if fallbackPreview == "" {
					fallbackPreview = sessionindex.SafePreview(prompt)
				}
			}
		} else if assistantMessage(event, payload) {
			messageCount++
		}
		sources.CollectToolFiles(event, files)
		return nil
	})
	if err != nil {
		return sessionindex.Record{}, nil, err
	}

	if latest == 0 {
		latest = info.ModTime().Unix()
	}
	project := sources.PortableBase(workspace)
	if workspace == "" {
		project = "unknown"
	}
	promptText := directPrompts.String()
	preview := directPreview
	if promptText == "" {
		promptText = fallbackPrompts.String()
		preview = fallbackPreview
	}
	if title == "" {
		title = id
	}
	fileList := sources.NormalizedFileMap(files)
	for index := range warnings {
		warnings[index].Agent = sessionindex.AgentCodex
		warnings[index].SessionID = id
		warnings[index].Source = path
	}

	return sessionindex.Record{
		Key:                 sessionindex.CompositeReference(sessionindex.AgentCodex, id),
		ID:                  id,
		Agent:               sessionindex.AgentCodex,
		Title:               title,
		Project:             project,
		Workspace:           workspace,
		Branch:              branch,
		UpdatedAt:           time.Unix(latest, 0).UTC(),
		SizeBytes:           info.Size(),
		MessageCount:        messageCount,
		PromptPreview:       preview,
		Files:               fileList,
		CanResume:           true,
		CanFork:             true,
		RecordedEnvironment: recordedEnvironment,
		SourcePath:          path,
		SourceModTime:       info.ModTime().UnixNano(),
		SourceSize:          info.Size(),
		SearchText:          sessionindex.BuildSearchText(id, title, project, workspace, branch, promptText, strings.Join(fileList, " ")),
	}, warnings, nil
}

// sessionIDFromFilename is the filename-wins identity rule: a rollout named
// after its own UUID stays addressable even when session_meta replays a fork
// source id.
func sessionIDFromFilename(path string) string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fields := strings.Split(stem, "-")
	if len(fields) < 5 {
		return ""
	}
	candidate := strings.Join(fields[len(fields)-5:], "-")
	if !looksLikeUUID(candidate) {
		return ""
	}
	return candidate
}

func looksLikeUUID(value string) bool {
	groups := strings.Split(value, "-")
	if len(groups) != 5 {
		return false
	}
	for index, width := range [5]int{8, 4, 4, 4, 12} {
		if len(groups[index]) != width {
			return false
		}
		for _, char := range groups[index] {
			isDigit := char >= '0' && char <= '9'
			isHex := (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
			if !isDigit && !isHex {
				return false
			}
		}
	}
	return true
}

func userPrompt(event, payload map[string]any) (string, bool) {
	eventType := strings.ToLower(sources.FirstString(event, "type"))
	payloadType := strings.ToLower(sources.FirstString(payload, "type"))

	switch eventType {
	case "response_item":
		if payloadType == "message" && strings.EqualFold(sources.FirstString(payload, "role"), "user") {
			return sources.ExtractTextContent(payload["content"]), true
		}
	case "event_msg":
		if payloadType == "user_message" {
			if message := sources.FirstString(payload, "message", "text"); message != "" {
				return message, true
			}
			return sources.ExtractTextContent(payload["content"]), true
		}
	case "message":
		if strings.EqualFold(sources.FirstString(event, "role"), "user") {
			return sources.ExtractTextContent(event["content"]), true
		}
	}
	if strings.EqualFold(sources.FirstString(event, "role"), "user") {
		return sources.ExtractTextContent(event["content"]), true
	}
	return "", false
}

func assistantMessage(event, payload map[string]any) bool {
	eventType := strings.ToLower(sources.FirstString(event, "type"))
	if eventType == "response_item" {
		return strings.EqualFold(sources.FirstString(payload, "type"), "message") &&
			strings.EqualFold(sources.FirstString(payload, "role"), "assistant")
	}
	if eventType == "event_msg" {
		return strings.EqualFold(sources.FirstString(payload, "type"), "agent_message")
	}
	return (eventType == "message" || sources.FirstString(event, "role") != "") &&
		strings.EqualFold(sources.FirstString(event, "role"), "assistant")
}
