package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

type localCommandOptions struct {
	sources       []sessionindex.Source
	launchRunner  sessionindex.LaunchRunner
	terminalCheck func(io.Reader, io.Writer) bool
}

type localSessionSummary struct {
	Key            string                    `json:"key"`
	ID             string                    `json:"id"`
	Agent          string                    `json:"agent"`
	Title          string                    `json:"title,omitempty"`
	Project        string                    `json:"project,omitempty"`
	Workspace      string                    `json:"workspace,omitempty"`
	Branch         string                    `json:"branch,omitempty"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	SizeBytes      int64                     `json:"size_bytes"`
	MessageCount   int                       `json:"message_count"`
	Files          []string                  `json:"files,omitempty"`
	Capabilities   sessionindex.Capabilities `json:"capabilities"`
	ReadOnlyReason string                    `json:"read_only_reason,omitempty"`
}

type localWarning struct {
	Agent     string `json:"agent,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type localSessionsOutput struct {
	Sessions []localSessionSummary `json:"sessions"`
	Warnings []localWarning        `json:"warnings,omitempty"`
}

type localInspectOutput struct {
	Session  sessionindex.Record `json:"session"`
	Warnings []localWarning      `json:"warnings,omitempty"`
}

func newSessionsCmd(options localCommandOptions) *cobra.Command {
	var asJSON bool
	var agent string
	var limit int
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List indexed local coding-agent sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateLocalAgent(agent, true); err != nil {
				return err
			}
			index, refresh, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			records, err := index.Search(cmd.Context(), sessionindex.Filter{
				Agent: agent,
				Limit: limit,
			})
			if err != nil {
				return localRuntimeError("read local session index", err)
			}
			return writeLocalSessions(cmd, records, refresh.Warnings, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().StringVar(&agent, "agent", "all", "agent filter: claude|codex|gemini|opencode|all")
	cmd.Flags().IntVar(&limit, "limit", sessionindex.DefaultLimit, "maximum sessions to return")
	return cmd
}

func newSearchCmd(options localCommandOptions) *cobra.Command {
	var asJSON bool
	var filter sessionindex.Filter
	cmd := &cobra.Command{
		Use:   "search QUERY [QUERY...]",
		Short: "Search bounded local session metadata and user prompts",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLocalAgent(filter.Agent, true); err != nil {
				return err
			}
			filter.Query = strings.Join(args, " ")
			index, refresh, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			records, err := index.Search(cmd.Context(), filter)
			if err != nil {
				return localRuntimeError("search local session index", err)
			}
			return writeLocalSessions(cmd, records, refresh.Warnings, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: claude|codex|gemini|opencode|all")
	cmd.Flags().StringVar(&filter.Project, "project", "", "project or workspace fragment")
	cmd.Flags().StringVar(&filter.Branch, "branch", "", "branch fragment")
	cmd.Flags().StringVar(&filter.File, "file", "", "known file fragment")
	cmd.Flags().IntVar(&filter.Limit, "limit", sessionindex.DefaultLimit, "maximum matches to return")
	return cmd
}

func newInspectCmd(options localCommandOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "inspect SESSION",
		Short: "Inspect safe metadata for one local session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, refresh, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, err := index.Resolve(cmd.Context(), args[0])
			if err != nil {
				return localResolveError(err)
			}
			return writeLocalInspect(cmd, record, refresh.Warnings, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func newLastCmd(options localCommandOptions) *cobra.Command {
	var asJSON bool
	var dryRun bool
	var filter sessionindex.Filter
	cmd := &cobra.Command{
		Use:   "last",
		Short: "Resume the newest matching local session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateNativeAgent(filter.Agent, true); err != nil {
				return err
			}
			if asJSON && !dryRun {
				return NewExitError(ExitUsage, "--json requires --dry-run for native agent launches")
			}
			filter.ResumableOnly = true
			index, _, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, err := index.Last(cmd.Context(), filter)
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(cmd, options, record, sessionindex.OperationResume, dryRun, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the native launch plan without starting the agent")
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: claude|codex|all")
	cmd.Flags().StringVar(&filter.Project, "project", "", "project or workspace fragment")
	return cmd
}

func newResumeCmd(options localCommandOptions) *cobra.Command {
	return newNativeActionCmd(options, sessionindex.OperationResume)
}

func newForkCmd(options localCommandOptions) *cobra.Command {
	return newNativeActionCmd(options, sessionindex.OperationFork)
}

func newNativeActionCmd(options localCommandOptions, operation string) *cobra.Command {
	var asJSON bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   operation + " SESSION",
		Short: strings.ToUpper(operation[:1]) + operation[1:] + " a session through its native coding agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON && !dryRun {
				return NewExitError(ExitUsage, "--json requires --dry-run for native agent launches")
			}
			index, _, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, err := index.Resolve(cmd.Context(), args[0])
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(cmd, options, record, operation, dryRun, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the native launch plan without starting the agent")
	return cmd
}

func defaultLocalSources() []sessionindex.Source {
	return []sessionindex.Source{
		sessionindex.NewClaudeSource(""),
		sessionindex.NewCodexSource(""),
		sessionindex.NewGeminiSource(""),
		sessionindex.NewOpenCodeSource(nil),
	}
}

func openRefreshedLocalIndex(
	cmd *cobra.Command,
	options localCommandOptions,
) (*sessionindex.Index, sessionindex.RefreshResult, error) {
	home, err := config.Home()
	if err != nil {
		return nil, sessionindex.RefreshResult{}, NewExitError(ExitConfig, err.Error())
	}
	sources := options.sources
	if sources == nil {
		sources = defaultLocalSources()
	}
	index, err := sessionindex.OpenIndex(home, sources...)
	if err != nil {
		return nil, sessionindex.RefreshResult{}, localRuntimeError("open local session index", err)
	}
	refresh, err := index.Refresh(cmd.Context())
	if err != nil {
		_ = index.Close()
		return nil, refresh, localRuntimeError("refresh local session index", err)
	}
	return index, refresh, nil
}

func launchLocalRecord(
	cmd *cobra.Command,
	options localCommandOptions,
	record sessionindex.Record,
	operation string,
	dryRun bool,
	asJSON bool,
) error {
	plan, err := sessionindex.PlanLaunch(record, operation)
	if err != nil {
		return localLaunchError(err)
	}
	if dryRun {
		if asJSON {
			return WriteJSON(cmd.OutOrStdout(), plan)
		}
		PrintHuman(
			cmd.OutOrStdout(),
			"%s %q in %s",
			plan.Executable,
			plan.Args,
			plan.Dir,
		)
		return nil
	}
	runner := options.launchRunner
	if runner == nil {
		runner = sessionindex.ExecLaunchRunner{
			Stdin:  cmd.InOrStdin(),
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
		}
	}
	if err := sessionindex.RunLaunch(cmd.Context(), plan, runner); err != nil {
		return localLaunchError(err)
	}
	return nil
}

func runSessionPicker(cmd *cobra.Command, options localCommandOptions) error {
	if !options.terminalCheck(cmd.InOrStdin(), cmd.OutOrStdout()) {
		_ = cmd.Help()
		return NewExitError(
			ExitUsage,
			"interactive session picker requires a terminal; use `rein sessions --json`",
		)
	}
	index, _, err := openRefreshedLocalIndex(cmd, options)
	if err != nil {
		return err
	}
	defer func() { _ = index.Close() }()

	reader := bufio.NewScanner(cmd.InOrStdin())
	filter := sessionindex.Filter{Limit: sessionindex.DefaultLimit}
	for {
		records, err := index.Search(cmd.Context(), filter)
		if err != nil {
			return localRuntimeError("read local session index", err)
		}
		printPicker(cmd.OutOrStdout(), records, filter.Query)
		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				return localRuntimeError("read picker input", err)
			}
			return nil
		}
		input := strings.TrimSpace(reader.Text())
		switch {
		case input == "" || strings.EqualFold(input, "q"):
			return nil
		case strings.HasPrefix(input, "/"):
			filter.Query = strings.TrimSpace(strings.TrimPrefix(input, "/"))
			continue
		case strings.HasPrefix(strings.ToLower(input), "i "):
			indexValue, ok := pickerIndex(input[2:], len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Invalid session number.")
				continue
			}
			if err := writeLocalInspect(cmd, records[indexValue], nil, false); err != nil {
				return err
			}
			continue
		case strings.HasPrefix(strings.ToLower(input), "f "):
			indexValue, ok := pickerIndex(input[2:], len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Invalid session number.")
				continue
			}
			return launchLocalRecord(
				cmd,
				options,
				records[indexValue],
				sessionindex.OperationFork,
				false,
				false,
			)
		default:
			indexValue, ok := pickerIndex(input, len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Enter a number, /text, i NUMBER, f NUMBER, or q.")
				continue
			}
			return launchLocalRecord(
				cmd,
				options,
				records[indexValue],
				sessionindex.OperationResume,
				false,
				false,
			)
		}
	}
}

func pickerIndex(value string, length int) (int, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 1 || number > length {
		return 0, false
	}
	return number - 1, true
}

func printPicker(writer io.Writer, records []sessionindex.Record, query string) {
	if query == "" {
		PrintHuman(writer, "Local sessions")
	} else {
		PrintHuman(writer, "Local sessions matching %q", query)
	}
	if len(records) == 0 {
		PrintHuman(writer, "  No matching sessions.")
	} else {
		for index, record := range records {
			PrintHuman(
				writer,
				"%3d  %-8s  %-24s  %s",
				index+1,
				record.Agent,
				record.Project,
				record.Title,
			)
		}
	}
	PrintHuman(writer, "Choose NUMBER, /text, i NUMBER, f NUMBER, or q:")
}

func writeLocalSessions(
	cmd *cobra.Command,
	records []sessionindex.Record,
	warnings []sessionindex.Warning,
	asJSON bool,
) error {
	summaries := make([]localSessionSummary, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, summarizeLocalRecord(record))
	}
	if asJSON {
		return WriteJSON(cmd.OutOrStdout(), localSessionsOutput{
			Sessions: summaries,
			Warnings: publicLocalWarnings(warnings),
		})
	}
	writeLocalWarnings(cmd.ErrOrStderr(), warnings)
	if len(summaries) == 0 {
		PrintHuman(cmd.OutOrStdout(), "No matching local sessions.")
		return nil
	}
	for _, summary := range summaries {
		PrintHuman(
			cmd.OutOrStdout(),
			"%s\t%s\t%s\t%s",
			summary.Key,
			summary.Project,
			summary.Branch,
			summary.Title,
		)
	}
	return nil
}

func writeLocalInspect(
	cmd *cobra.Command,
	record sessionindex.Record,
	warnings []sessionindex.Warning,
	asJSON bool,
) error {
	if asJSON {
		return WriteJSON(cmd.OutOrStdout(), localInspectOutput{
			Session:  record,
			Warnings: publicLocalWarnings(warnings),
		})
	}
	writeLocalWarnings(cmd.ErrOrStderr(), warnings)
	PrintHuman(cmd.OutOrStdout(), "Session: %s", record.Reference())
	PrintHuman(cmd.OutOrStdout(), "Agent: %s", record.Agent)
	PrintHuman(cmd.OutOrStdout(), "Title: %s", record.Title)
	PrintHuman(cmd.OutOrStdout(), "Project: %s", record.Project)
	PrintHuman(cmd.OutOrStdout(), "Workspace: %s", record.Workspace)
	PrintHuman(cmd.OutOrStdout(), "Branch: %s", record.Branch)
	PrintHuman(cmd.OutOrStdout(), "Updated: %s", record.UpdatedAt.Format(time.RFC3339))
	PrintHuman(cmd.OutOrStdout(), "Messages: %d", record.MessageCount)
	PrintHuman(cmd.OutOrStdout(), "Files: %s", strings.Join(record.Files, ", "))
	PrintHuman(
		cmd.OutOrStdout(),
		"Capabilities: resume=%t fork=%t",
		record.CanResume,
		record.CanFork,
	)
	if record.ReadOnlyReason != "" {
		PrintHuman(cmd.OutOrStdout(), "Read only: %s", record.ReadOnlyReason)
	}
	if record.PromptPreview != "" {
		PrintHuman(cmd.OutOrStdout(), "User prompt preview: %s", record.PromptPreview)
	}
	return nil
}

func summarizeLocalRecord(record sessionindex.Record) localSessionSummary {
	return localSessionSummary{
		Key:            record.Reference(),
		ID:             record.ID,
		Agent:          record.Agent,
		Title:          record.Title,
		Project:        record.Project,
		Workspace:      record.Workspace,
		Branch:         record.Branch,
		UpdatedAt:      record.UpdatedAt,
		SizeBytes:      record.SizeBytes,
		MessageCount:   record.MessageCount,
		Files:          append([]string(nil), record.Files...),
		Capabilities:   record.Capabilities(),
		ReadOnlyReason: record.ReadOnlyReason,
	}
}

func publicLocalWarnings(warnings []sessionindex.Warning) []localWarning {
	result := make([]localWarning, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, localWarning{
			Agent:     warning.Agent,
			SessionID: warning.SessionID,
			Code:      warning.Code,
			Message:   warning.Message,
		})
	}
	return result
}

func writeLocalWarnings(writer io.Writer, warnings []sessionindex.Warning) {
	for _, warning := range publicLocalWarnings(warnings) {
		if warning.Code == "agent_not_installed" {
			continue
		}
		if warning.Agent == "" {
			PrintHuman(writer, "warning: %s", warning.Message)
			continue
		}
		PrintHuman(writer, "warning [%s]: %s", warning.Agent, warning.Message)
	}
}

func validateLocalAgent(agent string, allowAll bool) error {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case sessionindex.AgentClaude,
		sessionindex.AgentCodex,
		sessionindex.AgentGemini,
		sessionindex.AgentOpenCode:
		return nil
	case "all":
		if allowAll {
			return nil
		}
	}
	return NewExitError(
		ExitUsage,
		"invalid agent; expected claude, codex, gemini, opencode, or all",
	)
}

func validateNativeAgent(agent string, allowAll bool) error {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case sessionindex.AgentClaude, sessionindex.AgentCodex:
		return nil
	case "all":
		if allowAll {
			return nil
		}
	}
	return NewExitError(ExitUsage, "invalid native agent; expected claude, codex, or all")
}

func localResolveError(err error) error {
	switch {
	case errors.Is(err, sessionindex.ErrNotFound),
		errors.Is(err, sessionindex.ErrAmbiguous):
		return NewExitError(ExitUsage, err.Error())
	default:
		return localRuntimeError("resolve local session", err)
	}
}

func localLaunchError(err error) error {
	if errors.Is(err, sessionindex.ErrNativeActionUnsupported) ||
		errors.Is(err, sessionindex.ErrExecutableNotFound) ||
		errors.Is(err, sessionindex.ErrWorkspaceUnavailable) {
		return NewExitError(ExitCompatibility, err.Error())
	}
	return localRuntimeError("launch native agent", err)
}

func localRuntimeError(action string, err error) error {
	if errors.Is(err, context.Canceled) {
		return NewExitError(ExitRuntime, action+": canceled")
	}
	return NewExitError(ExitRuntime, fmt.Sprintf("%s: %v", action, err))
}
