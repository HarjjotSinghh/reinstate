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
	"os"
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

// maxWorkspaceManifestBytes bounds the sibling workspace.yaml read. The file
// Copilot writes is a few hundred bytes; anything larger is not the manifest.
const maxWorkspaceManifestBytes = 64 << 10

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

// Fingerprint summarises the source without opening any file, so an
// unchanged refresh can skip parsing entirely.
func (s *Source) Fingerprint(ctx context.Context) (string, bool, error) {
	return hometree.Fingerprint(ctx, s.config())
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
	// Copilot CLI 1.0.80 stopped emitting cwd inside events.jsonl. The
	// sibling workspace.yaml still records it, so fall back to that manifest
	// before giving up on workspace truth.
	manifest := readWorkspaceManifest(filepath.Dir(file.Path))
	workspace := parsed.cwd
	if workspace == "" {
		workspace = manifest.cwd
	}
	if workspace == "" {
		workspace = manifest.gitRoot
	}
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
		Branch:         manifest.branch,
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

// workspaceManifest is the flat workspace.yaml Copilot writes beside
// events.jsonl. Only the scalar keys this source needs are read; the file is
// never executed and unknown keys are ignored.
type workspaceManifest struct {
	cwd     string
	gitRoot string
	branch  string
}

// readWorkspaceManifest reads the bounded flat "key: value" manifest. A
// missing, oversized, or malformed file yields a zero manifest, never an
// error: workspace truth is a best-effort enrichment, not a gate.
func readWorkspaceManifest(sessionDir string) workspaceManifest {
	path := filepath.Join(sessionDir, "workspace.yaml")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxWorkspaceManifestBytes {
		return workspaceManifest{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return workspaceManifest{}
	}
	var manifest workspaceManifest
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "-") {
			continue // nested or list content is out of scope
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "\r"))
		value = strings.Trim(value, "'\"")
		if value == "" {
			continue
		}
		switch strings.TrimSpace(key) {
		case "cwd":
			manifest.cwd = value
		case "git_root":
			manifest.gitRoot = value
		case "branch":
			manifest.branch = value
		}
	}
	return manifest
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
