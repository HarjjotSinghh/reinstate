package sessionindex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	openCodeListTimeout   = 5 * time.Second
	maxOpenCodeListOutput = 16 << 20
	openCodeReadOnly      = "OpenCode sessions are read-only in Phase 2"
)

var ErrCommandOutputTooLarge = errors.New("local command output exceeds safe limit")

// CommandRunner is the small injectable command surface used for OpenCode's
// documented JSON session-list command.
type CommandRunner interface {
	Run(ctx context.Context, executable string, args ...string) ([]byte, error)
}

// CommandRunnerFunc adapts a function to CommandRunner.
type CommandRunnerFunc func(context.Context, string, ...string) ([]byte, error)

func (runner CommandRunnerFunc) Run(ctx context.Context, executable string, args ...string) ([]byte, error) {
	return runner(ctx, executable, args...)
}

// ExecCommandRunner runs a local executable without a shell.
type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, executable string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, executable, args...)
	output := boundedCommandOutput{remaining: maxOpenCodeListOutput}
	command.Stdout = &output
	err := command.Run()
	if output.exceeded {
		return nil, fmt.Errorf(
			"%w: maximum is %d bytes",
			ErrCommandOutputTooLarge,
			maxOpenCodeListOutput,
		)
	}
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type boundedCommandOutput struct {
	bytes.Buffer
	remaining int
	exceeded  bool
}

func (output *boundedCommandOutput) Write(value []byte) (int, error) {
	length := len(value)
	if length > output.remaining {
		output.exceeded = true
		value = value[:max(output.remaining, 0)]
	}
	if len(value) > 0 {
		_, _ = output.Buffer.Write(value)
		output.remaining -= len(value)
	}
	// Report the original length so os/exec continues draining the child's
	// stdout without retaining bytes beyond the cap.
	return length, nil
}

// OpenCodeSource obtains local session metadata through OpenCode's supported
// read-only JSON command rather than inspecting its private storage.
type OpenCodeSource struct {
	runner CommandRunner
}

func NewOpenCodeSource(runner CommandRunner) *OpenCodeSource {
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	return &OpenCodeSource{runner: runner}
}

func (s *OpenCodeSource) Name() string { return AgentOpenCode }

func (s *OpenCodeSource) Scan(ctx context.Context) (ScanResult, error) {
	runContext, cancel := context.WithTimeout(ctx, openCodeListTimeout)
	defer cancel()
	output, err := s.runner.Run(runContext, "opencode", "session", "list", "--format", "json")
	if err != nil {
		if commandNotInstalled(err) {
			return ScanResult{Warnings: []Warning{{
				Agent:   AgentOpenCode,
				Source:  "opencode session list",
				Code:    "agent_not_installed",
				Message: "OpenCode executable was not found; OpenCode sessions were not indexed",
			}}}, nil
		}
		return ScanResult{}, fmt.Errorf("list OpenCode sessions: %w", err)
	}
	if len(output) > maxOpenCodeListOutput {
		return ScanResult{}, fmt.Errorf(
			"list OpenCode sessions: %w: maximum is %d bytes",
			ErrCommandOutputTooLarge,
			maxOpenCodeListOutput,
		)
	}

	sessions, err := decodeOpenCodeSessions(output)
	if err != nil {
		return ScanResult{}, fmt.Errorf("decode OpenCode session list: %w", err)
	}
	result := ScanResult{Records: make([]Record, 0, len(sessions))}
	for index, session := range sessions {
		record, ok := openCodeRecord(session)
		if !ok {
			result.Warnings = append(result.Warnings, Warning{
				Agent:   AgentOpenCode,
				Source:  "opencode session list",
				Code:    "missing_session_id",
				Message: fmt.Sprintf("ignored OpenCode session entry %d without an ID", index+1),
			})
			continue
		}
		result.Records = append(result.Records, record)
	}
	sortRecordsBySourcePath(result.Records)
	return result, nil
}

func commandNotInstalled(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var exitError interface{ ExitCode() int }
	return errors.As(err, &exitError) && exitError.ExitCode() == 127
}

func decodeOpenCodeSessions(output []byte) ([]map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	switch typed := raw.(type) {
	case []any:
		return mapsFromAny(typed), nil
	case map[string]any:
		for _, key := range []string{"sessions", "data", "results"} {
			if entries, ok := typed[key].([]any); ok {
				return mapsFromAny(entries), nil
			}
		}
		// Be defensive around a single-session response while still requiring a
		// recognizable identity field.
		if firstString(typed, "id", "sessionID", "sessionId") != "" {
			return []map[string]any{typed}, nil
		}
		return nil, errors.New("JSON response does not contain a session array")
	default:
		return nil, errors.New("JSON response is not an object or array")
	}
}

func mapsFromAny(values []any) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if mapped, ok := value.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func openCodeRecord(values map[string]any) (Record, bool) {
	id := firstString(values, "id", "sessionID", "sessionId", "session_id")
	if id == "" {
		return Record{}, false
	}
	title := firstString(values, "title", "name", "summary")
	workspace := firstString(values, "directory", "cwd", "workspace", "workingDirectory")
	branch := firstString(values, "branch", "gitBranch")
	// OpenCode reports projectId as an opaque 40-hex digest. Every other
	// source names a project after its directory, and Matrix C2 compares the
	// project against what the agent shows, so prefer a human name and keep
	// the vendor digest as the last resort. Kept identical to the catalog
	// source in internal/agents/sources/opencode.
	project := ""
	if mapped, ok := values["project"].(map[string]any); ok {
		project = firstString(mapped, "name")
	}
	if project == "" && workspace != "" {
		project = portableBase(workspace)
	}
	if project == "" {
		project = firstString(values, "project", "projectID", "projectId", "project_id")
	}
	if project == "" {
		project = "unknown"
	}
	updatedAt := eventTimestamp(values)
	updatedAt = maxInt64(updatedAt, parseTimestamp(values["updated"]))
	updatedAt = maxInt64(updatedAt, parseTimestamp(values["created"]))
	if timing, ok := values["time"].(map[string]any); ok {
		updatedAt = maxInt64(updatedAt, eventTimestamp(timing))
		updatedAt = maxInt64(updatedAt, parseTimestamp(timing["updated"]))
		updatedAt = maxInt64(updatedAt, parseTimestamp(timing["created"]))
	}
	messageCount := firstInt(values, "messageCount", "message_count", "messages")
	title = SafePreview(title)
	if title == "" {
		title = id
	}

	sourcePath := "opencode://session/" + id
	return Record{
		Key:            CompositeReference(AgentOpenCode, id),
		ID:             id,
		Agent:          AgentOpenCode,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		Branch:         branch,
		UpdatedAt:      time.Unix(updatedAt, 0).UTC(),
		MessageCount:   messageCount,
		ReadOnlyReason: openCodeReadOnly,
		SourcePath:     sourcePath,
		SourceModTime:  time.Unix(updatedAt, 0).UnixNano(),
		SearchText:     BuildSearchText(id, title, project, workspace, branch),
	}, true
}

func firstInt(values map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := values[key].(type) {
		case int:
			return value
		case float64:
			return int(value)
		case json.Number:
			number, err := strconv.Atoi(value.String())
			if err == nil {
				return number
			}
		case string:
			number, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil {
				return number
			}
		case []any:
			return len(value)
		}
	}
	return 0
}
