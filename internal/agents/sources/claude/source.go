// Package claude ports Claude Code's F1 index source onto the shared hometree scanner.
package claude

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

const sessionGlob = "projects/**/*.jsonl"

// Excluded is the Claude subtree set: subagents plus adapter credential/cache exclusions.
var Excluded = []string{
	"subagents",
	"**/auth.json",
	"**/.credentials.json",
	"**/credentials.json",
	"**/.env",
	"**/cache/**",
}

// Source discovers Claude Code project JSONL sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Claude index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentClaude }

// Scan walks the Claude projects tree and maps each JSONL file to a record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	projectsRoot := filepath.Join(root, "projects")
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		record, warnings, parseErr := parseSession(file.Path, projectsRoot)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentClaude,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Claude session could not be read; other sessions remain available",
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
		RootEnv:     "CLAUDE_CONFIG_DIR",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "projects",
		SessionGlob: sessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".claude"), home.Join(".config", "claude")}
	}
	return cfg
}

func parseSession(path, projectsRoot string) (sessionindex.Record, []sessionindex.Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	project := projectName(path, projectsRoot)
	var (
		workspace           string
		branch              string
		title               string
		latest              int64
		messageCount        int
		prompts             sources.BoundedText
		firstPrompt         string
		files               = make(map[string]struct{})
		recordedEnvironment environment.RecordedEnvironment
	)

	warnings, err := hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return nil
		}
		if value := sources.FirstString(event, "sessionId", "session_id"); value != "" {
			id = value
		}
		if value := sources.FirstString(event, "cwd", "workingDirectory", "workdir"); value != "" {
			workspace = value
		}
		if value := sources.FirstString(event, "gitBranch"); value != "" {
			branch = value
			recordedEnvironment.Branch = environment.RecordedField{
				Value:      value,
				Provenance: "claude.event.gitBranch",
			}
		} else if value := sources.FirstString(event, "branch"); value != "" {
			branch = value
			recordedEnvironment.Branch = environment.RecordedField{
				Value:      value,
				Provenance: "claude.event.branch",
			}
		}
		eventType := strings.ToLower(sources.FirstString(event, "type"))
		if value := sources.FirstString(event, "customTitle"); value != "" {
			title = sessionindex.SafePreview(value)
		} else if eventType == "summary" || eventType == "session_meta" || eventType == "metadata" {
			if value := sources.FirstString(event, "title", "summary", "name"); value != "" {
				title = sessionindex.SafePreview(value)
			}
		}
		latest = sources.MaxInt64(latest, sources.EventTimestamp(event))

		message, _ := event["message"].(map[string]any)
		role := strings.ToLower(sources.FirstString(message, "role"))
		if eventType == "user" || eventType == "assistant" || role == "user" || role == "assistant" {
			messageCount++
		}
		if (eventType == "user" || role == "user") && !sources.BoolValue(event["isMeta"]) {
			prompt := sources.ExtractTextContent(message["content"])
			prompts.Add(prompt)
			if firstPrompt == "" {
				firstPrompt = sessionindex.SafePreview(prompt)
			}
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
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}
	preview := firstPrompt
	if title == "" {
		title = id
	}
	fileList := sources.NormalizedFileMap(files)
	for index := range warnings {
		warnings[index].Agent = sessionindex.AgentClaude
		warnings[index].SessionID = id
		warnings[index].Source = path
	}

	return sessionindex.Record{
		Key:                 sessionindex.CompositeReference(sessionindex.AgentClaude, id),
		ID:                  id,
		Agent:               sessionindex.AgentClaude,
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
		SearchText:          sessionindex.BuildSearchText(id, title, project, workspace, branch, prompts.String(), strings.Join(fileList, " ")),
	}, warnings, nil
}

func projectName(path, projectsRoot string) string {
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
