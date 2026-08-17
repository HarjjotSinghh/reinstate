// Package pi discovers Pi coding-agent sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 macOS and native Windows probes
// (docs/testing/results/agent-probes/2026-08-17-macos-pi.json and
// 2026-08-17-windows-pi.json):
//
//	~/.pi/agent/sessions/<slug>/<slug>-<uuid-v4>.jsonl
//
// The first complete JSONL line is a type=session header with keys
// cwd, id, timestamp, type, version. Later lines are a tree; this T1
// source does not parse them. MessageCount stays 0 until a T2 reader
// exists. Resume and fork stay refused.
package pi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// SessionGlob matches one session JSONL file below the config root.
const SessionGlob = "sessions/**/*.jsonl"

// Excluded keeps credentials, caches, packages, skills, and HTML exports
// out of the walk. PI_CODING_AGENT_SESSION_DIR is a separate override; it
// is not RootEnv because the default session tree lives under the config
// root and that override has not been probed on a device.
var Excluded = []string{
	"auth.json",
	"**/auth.json",
	"npm",
	"git",
	"extensions",
	"skills",
	"prompts",
	"themes",
	"models-store.json",
	"**/*.html",
}

// Source discovers Pi sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Pi index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentPi }

// Scan maps every readable session JSONL file to one record.
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
				Agent:   sessionindex.AgentPi,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Pi session could not be read; other sessions remain available",
			})
			continue
		}
		result.Records = append(result.Records, record)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

func (s *Source) config() hometree.Config {
	cfg := hometree.Config{
		Explicit:    s.env.FixtureRoot,
		RootEnv:     "PI_CODING_AGENT_DIR",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "sessions",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".pi", "agent")}
	}
	return cfg
}

type header struct {
	Type      string
	ID        string
	CWD       string
	Timestamp time.Time
}

func parseSession(file hometree.File) (sessionindex.Record, error) {
	item, err := readHeader(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	id := strings.TrimSpace(item.ID)
	if id == "" || !utf8.ValidString(id) {
		return sessionindex.Record{}, fmt.Errorf("pi session header missing id")
	}
	if item.Type != "session" {
		return sessionindex.Record{}, fmt.Errorf("pi session header type %q is not session", item.Type)
	}

	workspace := strings.TrimSpace(item.CWD)
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}

	updatedAt := item.Timestamp
	if updatedAt.IsZero() {
		updatedAt = file.ModTime.UTC()
	}

	title := sessionindex.SafePreview(id)
	preview := sessionindex.SafePreview(project)
	if preview == "" || preview == "unknown" {
		preview = title
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentPi, id),
		ID:             id,
		Agent:          sessionindex.AgentPi,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updatedAt,
		SizeBytes:      file.Size,
		MessageCount:   0,
		PromptPreview:  preview,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.PiReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  file.ModTime.UnixNano(),
		SourceSize:     file.Size,
		SearchText:     sessionindex.BuildSearchText(id, title, project, workspace, "", ""),
	}, nil
}

func readHeader(path string) (header, error) {
	var item header
	var sawLine bool
	_, err := hometree.ReadJSONL(path, func(_ int, line []byte) error {
		if sawLine {
			return nil
		}
		sawLine = true
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			return fmt.Errorf("pi session header is not JSON")
		}
		item.Type = strings.TrimSpace(sources.FirstString(raw, "type"))
		item.ID = strings.TrimSpace(sources.FirstString(raw, "id"))
		item.CWD = strings.TrimSpace(sources.FirstString(raw, "cwd"))
		if ts := sources.ParseTimestamp(raw["timestamp"]); ts != 0 {
			item.Timestamp = time.Unix(ts, 0).UTC()
		}
		return nil
	})
	if err != nil {
		return header{}, err
	}
	if !sawLine {
		return header{}, fmt.Errorf("pi session file has no complete JSON line")
	}
	return item, nil
}
