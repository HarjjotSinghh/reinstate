// Package cline discovers Cline CLI sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-19 dual-platform probes:
//
//	~/.cline/data/sessions/<slug>/<slug>.json
//	~/.cline/data/sessions/<slug>/<slug>.messages.json
//
// Session metadata is pretty-printed JSON. Both platforms also write
// db/sessions.db; that file is not parsed. *.messages.json is skipped.
package cline

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

// SessionGlob matches session metadata and message sidecars. Scan skips the latter.
const SessionGlob = "sessions/*/*.json"

var requiredKeys = []string{"cwd", "session_id", "started_at", "status"}

// Excluded keeps credentials, locks, logs, and caches out of the walk.
var Excluded = []string{
	"settings/providers.json",
	"**/providers.json",
	"locks",
	"logs",
	"cache",
}

// Source discovers Cline sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Cline index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentCline }

// Scan maps every readable session metadata file to one record.
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
		if strings.HasSuffix(file.Path, ".messages.json") {
			continue
		}
		record, parseErr := parseSession(file)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentCline,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Cline session could not be read; other sessions remain available",
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
		RootEnv:     "CLINE_DATA_DIR",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "sessions",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".cline", "data")}
	}
	return cfg
}

type meta struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	WorkspaceRoot string `json:"workspace_root"`
	Prompt        string `json:"prompt"`
	StartedAt     string `json:"started_at"`
	EndedAt       string `json:"ended_at"`
	Status        string `json:"status"`
}

func parseSession(file hometree.File) (sessionindex.Record, error) {
	item, err := readMeta(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	id := strings.TrimSpace(item.SessionID)
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	}
	workspace := strings.TrimSpace(item.CWD)
	if workspace == "" {
		workspace = strings.TrimSpace(item.WorkspaceRoot)
	}
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}
	title := sessionindex.SafePreview(item.Prompt)
	if title == "" {
		title = id
	}
	updated := parseTime(item.EndedAt)
	if updated.IsZero() {
		updated = parseTime(item.StartedAt)
	}
	if updated.IsZero() {
		updated = file.ModTime.UTC()
	}
	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentCline, id),
		ID:             id,
		Agent:          sessionindex.AgentCline,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      file.Size,
		PromptPreview:  title,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.ClineReadOnlyReason,
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
		return meta{}, fmt.Errorf("cline session JSON exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return meta{}, err
	}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			return meta{}, fmt.Errorf("unknown cline layout: missing %s", key)
		}
	}
	var item meta
	if err := json.Unmarshal(data, &item); err != nil {
		return meta{}, err
	}
	return item, nil
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
