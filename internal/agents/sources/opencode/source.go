// Package opencode ports OpenCode's F2 index source onto the shared cliquery scanner.
package opencode

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/cliquery"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

const readOnlyReason = "OpenCode sessions are read-only in Phase 2"

// Source lists OpenCode sessions through the documented JSON CLI.
type Source struct {
	env    agents.Env
	runner cliquery.Runner
}

// New constructs an OpenCode index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// NewWithRunner injects a command runner for tests.
func NewWithRunner(env agents.Env, runner cliquery.Runner) *Source {
	return &Source{env: env, runner: runner}
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentOpenCode }

// Scan runs `opencode session list --format json` and maps each entry.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	output, err := cliquery.Run(ctx, "opencode", []string{"session", "list", "--format", "json"}, cliquery.Config{
		Runner: s.runner,
	})
	if err != nil {
		if commandNotInstalled(err) {
			return sessionindex.ScanResult{Warnings: []sessionindex.Warning{{
				Agent:   sessionindex.AgentOpenCode,
				Source:  "opencode session list",
				Code:    "agent_not_installed",
				Message: "OpenCode executable was not found; OpenCode sessions were not indexed",
			}}}, nil
		}
		return sessionindex.ScanResult{}, fmt.Errorf("list OpenCode sessions: %w", err)
	}

	sessions, err := decodeSessions(output)
	if err != nil {
		return sessionindex.ScanResult{}, fmt.Errorf("decode OpenCode session list: %w", err)
	}
	result := sessionindex.ScanResult{Records: make([]sessionindex.Record, 0, len(sessions))}
	for index, session := range sessions {
		record, ok := recordFrom(session)
		if !ok {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentOpenCode,
				Source:  "opencode session list",
				Code:    "missing_session_id",
				Message: fmt.Sprintf("ignored OpenCode session entry %d without an ID", index+1),
			})
			continue
		}
		result.Records = append(result.Records, record)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

func commandNotInstalled(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var exitError interface{ ExitCode() int }
	return errors.As(err, &exitError) && exitError.ExitCode() == 127
}

func decodeSessions(output []byte) ([]map[string]any, error) {
	var raw any
	if err := cliquery.DecodeJSON(output, &raw); err != nil {
		return nil, err
	}
	switch typed := raw.(type) {
	case []any:
		return sources.MapsFromAny(typed), nil
	case map[string]any:
		for _, key := range []string{"sessions", "data", "results"} {
			if entries, ok := typed[key].([]any); ok {
				return sources.MapsFromAny(entries), nil
			}
		}
		if sources.FirstString(typed, "id", "sessionID", "sessionId") != "" {
			return []map[string]any{typed}, nil
		}
		return nil, errors.New("JSON response does not contain a session array")
	default:
		return nil, errors.New("JSON response is not an object or array")
	}
}

func recordFrom(values map[string]any) (sessionindex.Record, bool) {
	id := sources.FirstString(values, "id", "sessionID", "sessionId", "session_id")
	if id == "" {
		return sessionindex.Record{}, false
	}
	title := sources.FirstString(values, "title", "name", "summary")
	workspace := sources.FirstString(values, "directory", "cwd", "workspace", "workingDirectory")
	branch := sources.FirstString(values, "branch", "gitBranch")
	// OpenCode reports projectId as an opaque 40-hex digest. Every other
	// source names a project after its directory, and Matrix C2 compares the
	// project against what the agent shows, so prefer a human name and keep
	// the vendor digest as the last resort.
	project := ""
	if mapped, ok := values["project"].(map[string]any); ok {
		project = sources.FirstString(mapped, "name")
	}
	if project == "" && workspace != "" {
		project = sources.PortableBase(workspace)
	}
	if project == "" {
		project = sources.FirstString(values, "project", "projectID", "projectId", "project_id")
	}
	if project == "" {
		project = "unknown"
	}
	updatedAt := sources.EventTimestamp(values)
	updatedAt = sources.MaxInt64(updatedAt, sources.ParseTimestamp(values["updated"]))
	updatedAt = sources.MaxInt64(updatedAt, sources.ParseTimestamp(values["created"]))
	if timing, ok := values["time"].(map[string]any); ok {
		updatedAt = sources.MaxInt64(updatedAt, sources.EventTimestamp(timing))
		updatedAt = sources.MaxInt64(updatedAt, sources.ParseTimestamp(timing["updated"]))
		updatedAt = sources.MaxInt64(updatedAt, sources.ParseTimestamp(timing["created"]))
	}
	messageCount := sources.FirstInt(values, "messageCount", "message_count", "messages")
	title = sessionindex.SafePreview(title)
	if title == "" {
		title = id
	}

	sourcePath := "opencode://session/" + id
	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentOpenCode, id),
		ID:             id,
		Agent:          sessionindex.AgentOpenCode,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		Branch:         branch,
		UpdatedAt:      time.Unix(updatedAt, 0).UTC(),
		MessageCount:   messageCount,
		ReadOnlyReason: readOnlyReason,
		SourcePath:     sourcePath,
		SourceModTime:  time.Unix(updatedAt, 0).UnixNano(),
		SearchText:     sessionindex.BuildSearchText(id, title, project, workspace, branch),
	}, true
}

// DataRoot resolves the OpenCode XDG data root from env, for descriptors and tests.
func DataRoot(env agents.Env) (string, error) {
	if env.FixtureRoot != "" {
		return env.FixtureRoot, nil
	}
	home := func() (string, error) {
		dir, err := env.HomeDir()
		return dir.String(), err
	}
	return transcript.ResolveOpenCodeDataRoot(env.Lookup, home)
}
