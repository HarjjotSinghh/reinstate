// Package cursor discovers Cursor CLI sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 dual-platform probes:
//
//	~/.cursor/chats/<32-hex>/<uuid-v4>/meta.json
//
// First-line keys on both platforms: createdAtMs, cwd, hasConversation,
// schemaVersion, updatedAtMs. The editor tree under projects/ is excluded.
package cursor

import (
	"context"
	"encoding/json"
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

// SessionGlob matches one CLI session metadata file.
const SessionGlob = "chats/**/meta.json"

var requiredKeys = []string{"createdAtMs", "cwd", "hasConversation", "schemaVersion", "updatedAtMs"}

// Excluded keeps the editor agent, extensions, and skills out of the CLI walk.
var Excluded = []string{
	"projects",
	"extensions",
	"plugins",
	"skills",
	"skills-cursor",
	"plans",
	"agents",
	"rules",
	"ai-tracking",
	"sandbox-policies",
	"worktrees",
	"cli-config.json",
	"mcp.json",
	"**/mcp.json",
	"ide_state.json",
	"argv.json",
	"hooks.json",
	"hooks.json.bak",
	"agent-cli-state.json",
	"statsig-cache.json",
}

// Source discovers Cursor CLI sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Cursor index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentCursor }

// Scan maps every readable CLI meta.json to one record.
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
				Agent:   sessionindex.AgentCursor,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Cursor CLI session could not be read; other sessions remain available",
			})
			continue
		}
		if record.ID == "" {
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
		LookupEnv:   s.env.LookupEnv,
		Marker:      "chats",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".cursor")}
	}
	return cfg
}

type meta struct {
	CreatedAtMs     json.Number `json:"createdAtMs"`
	UpdatedAtMs     json.Number `json:"updatedAtMs"`
	CWD             string      `json:"cwd"`
	HasConversation bool        `json:"hasConversation"`
	SchemaVersion   *int        `json:"schemaVersion"`
}

func parseSession(file hometree.File) (sessionindex.Record, error) {
	item, err := readMeta(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	if item.SchemaVersion == nil {
		return sessionindex.Record{}, fmt.Errorf("unknown cursor layout: missing schemaVersion")
	}
	if !item.HasConversation {
		return sessionindex.Record{}, nil
	}
	id := filepath.Base(filepath.Dir(file.Path))
	workspace := strings.TrimSpace(item.CWD)
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}
	updated := unixMs(item.UpdatedAtMs)
	if updated.IsZero() {
		updated = unixMs(item.CreatedAtMs)
	}
	if updated.IsZero() {
		updated = file.ModTime.UTC()
	}
	title := project
	if title == "unknown" {
		title = id
	}
	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentCursor, id),
		ID:             id,
		Agent:          sessionindex.AgentCursor,
		Title:          sessionindex.SafePreview(title),
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      file.Size,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.CursorReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  file.ModTime.UnixNano(),
		SourceSize:     file.Size,
		SearchText:     sessionindex.BuildSearchText(id, title, project, workspace),
	}, nil
}

func readMeta(path string) (meta, error) {
	file, err := os.Open(path)
	if err != nil {
		return meta{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return meta{}, err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return meta{}, fmt.Errorf("cursor meta.json exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return meta{}, err
	}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			return meta{}, fmt.Errorf("unknown cursor layout: missing %s", key)
		}
	}
	var item meta
	if err := json.Unmarshal(data, &item); err != nil {
		return meta{}, err
	}
	return item, nil
}

func unixMs(value json.Number) time.Time {
	n, err := value.Int64()
	if err != nil || n <= 0 {
		return time.Time{}
	}
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}
