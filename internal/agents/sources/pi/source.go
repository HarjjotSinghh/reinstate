// Package pi discovers Pi coding-agent sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-17 dual-platform probes:
//
//	~/.pi/agent/sessions/<slug>/<slug>-<uuid-v4>.jsonl
//
// First-line keys on both platforms: cwd, id, timestamp, type, version.
package pi

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// SessionGlob matches one Pi session JSONL file.
const SessionGlob = "sessions/**/*.jsonl"

var requiredKeys = []string{"cwd", "id", "timestamp", "type"}

// Excluded keeps credentials, caches, packages, and HTML exports out of the walk.
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

// Fingerprint summarises the source without opening any file, so an
// unchanged refresh can skip parsing entirely.
func (s *Source) Fingerprint(ctx context.Context) (string, bool, error) {
	return hometree.Fingerprint(ctx, s.config())
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

func parseSession(file hometree.File) (sessionindex.Record, error) {
	parsed, err := readConversation(file.Path)
	if err != nil {
		return sessionindex.Record{}, err
	}
	id := parsed.id
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	}
	workspace := parsed.cwd
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
		Key:            sessionindex.CompositeReference(sessionindex.AgentPi, id),
		ID:             id,
		Agent:          sessionindex.AgentPi,
		Title:          sessionindex.SafePreview(title),
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updated,
		SizeBytes:      file.Size,
		MessageCount:   parsed.messages,
		PromptPreview:  sessionindex.SafePreview(parsed.firstPrompt),
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.PiReadOnlyReason,
		SourcePath:     file.Path,
		SourceModTime:  file.ModTime.UnixNano(),
		SourceSize:     file.Size,
		SearchText: sessionindex.BuildSearchText(
			id, title, project, workspace, parsed.prompts.String(),
		),
	}, nil
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
					return fmt.Errorf("unknown pi layout: missing %s", key)
				}
			}
			first = false
		}
		if out.id == "" {
			out.id = strings.TrimSpace(sources.FirstString(item, "id"))
		}
		if out.cwd == "" {
			out.cwd = strings.TrimSpace(sources.FirstString(item, "cwd"))
		}
		if stamp := sources.EventTimestamp(item); stamp != 0 {
			out.updated = time.Unix(stamp, 0).UTC()
		}
		role, text := turnRoleAndText(item)
		switch role {
		case "user", "human":
			out.messages++
			out.prompts.Add(text)
			if out.firstPrompt == "" && text != "" {
				out.firstPrompt = sessionindex.SafePreview(text)
			}
		case "assistant", "ai":
			out.messages++
		}
		return nil
	})
	if err != nil {
		return conversation{}, err
	}
	if first {
		return conversation{}, fmt.Errorf("pi conversation is empty")
	}
	return out, nil
}

// turnRoleAndText resolves the speaker role and readable text for one JSONL
// item.
//
// Real (version 3) turn items look like:
//
//	{"type":"message","id":"...","message":{"role":"user","content":[{"type":"text","text":"..."}]}}
//
// The top-level "type" is "message" for both user and assistant turns, so
// the role lives on the nested message object, and the readable text is
// message.content — an array of typed parts (text/input_text plus
// tool/thinking parts that ExtractTextContent already ignores) — never the
// message wrapper itself.
//
// The legacy (version 1) shape has no "message" wrapper: the role is the
// top-level "type" directly and the text is a top-level "text" string.
//
//	{"type":"user","text":"..."}
func turnRoleAndText(item map[string]any) (role, text string) {
	if message, ok := item["message"].(map[string]any); ok {
		role = strings.ToLower(sources.FirstString(message, "role"))
		text = sources.ExtractTextContent(message["content"])
		if role != "" {
			return role, text
		}
	}
	// No usable nested message: fall back to the legacy top-level shape.
	role = strings.ToLower(sources.FirstString(item, "type"))
	if text == "" {
		text = sources.FirstString(item, "text")
	}
	return role, text
}
