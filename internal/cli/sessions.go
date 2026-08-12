package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/doctor"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

type localCommandOptions struct {
	sources       []sessionindex.Source
	launchRunner  sessionindex.LaunchRunner
	verifier      preflight.Verifier
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
	Session     sessionindex.Record `json:"session"`
	Environment preflight.Report    `json:"environment"`
	Warnings    []localWarning      `json:"warnings,omitempty"`
}

type localLaunchOutput struct {
	sessionindex.LaunchPlan
	Environment preflight.Report `json:"environment"`
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
	cmd.Flags().StringVar(&agent, "agent", "all", "agent filter: claude|codex|gemini|opencode|grok|all")
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
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: claude|codex|gemini|opencode|grok|all")
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
			index, err := openLocalIndex(options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, refresh, fresh, err := index.RefreshAndResolve(cmd.Context(), args[0])
			if err != nil {
				return localResolveError(err)
			}
			report, err := verifyLocalRecord(cmd.Context(), options, index, record, fresh)
			if err != nil {
				return err
			}
			return writeLocalInspect(cmd, record, report, refresh.Warnings, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func newLastCmd(options localCommandOptions) *cobra.Command {
	var asJSON bool
	var dryRun bool
	var allowedWarnings []string
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
			index, refresh, err := openRefreshedLocalIndex(cmd, options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, err := index.Last(cmd.Context(), filter)
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(
				cmd, options, index, record, refresh.SourceFresh(record.Agent),
				sessionindex.OperationResume, dryRun, asJSON, allowedWarnings, nil,
			)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the native launch plan without starting the agent")
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: claude|codex|all")
	cmd.Flags().StringVar(&filter.Project, "project", "", "project or workspace fragment")
	cmd.Flags().StringArrayVar(
		&allowedWarnings,
		"allow-environment-warning",
		nil,
		"acknowledge one exact current environment warning ID (repeatable)",
	)
	return cmd
}

func newResumeCmd(options localCommandOptions) *cobra.Command {
	return newNativeActionCmd(options, sessionindex.OperationResume, true)
}

func newForkCmd(options localCommandOptions) *cobra.Command {
	return newNativeActionCmd(options, sessionindex.OperationFork, false)
}

func newNativeActionCmd(options localCommandOptions, operation string, allowHandoff bool) *cobra.Command {
	var asJSON bool
	var dryRun bool
	var fork bool
	var withAgent string
	var allowedWarnings []string
	cmd := &cobra.Command{
		Use:   operation + " SESSION",
		Short: strings.ToUpper(operation[:1]) + operation[1:] + " a session through its native coding agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(withAgent) != "" && fork {
				return NewExitError(ExitUsage, "--with and --fork are mutually exclusive")
			}
			if strings.TrimSpace(withAgent) != "" {
				PrintHuman(cmd.ErrOrStderr(), "%s.", handoffHumanPrefix(withAgent))
				return runHandoffAlias(cmd, args[0], withAgent, dryRun, asJSON, allowedWarnings)
			}
			action := operation
			if fork {
				action = sessionindex.OperationFork
			}
			if asJSON && !dryRun {
				return NewExitError(ExitUsage, "--json requires --dry-run for native agent launches")
			}
			index, err := openLocalIndex(options)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()
			record, _, fresh, err := index.RefreshAndResolve(cmd.Context(), args[0])
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(
				cmd, options, index, record, fresh, action, dryRun, asJSON,
				allowedWarnings, nil,
			)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the native launch plan without starting the agent")
	if allowHandoff {
		cmd.Flags().StringVar(&withAgent, "with", "", "continue the same task through a structured handoff to claude|codex")
		cmd.Flags().BoolVar(&fork, "fork", false, "fork through the native agent instead of resuming")
	}
	cmd.Flags().StringArrayVar(
		&allowedWarnings,
		"allow-environment-warning",
		nil,
		"acknowledge one exact current environment warning ID (repeatable)",
	)
	return cmd
}

func runHandoffAlias(cmd *cobra.Command, session, agent string, dryRun, asJSON bool, allowedWarnings []string) error {
	handoffCmd, _, err := cmd.Root().Find([]string{"handoff"})
	if err != nil || handoffCmd == nil || handoffCmd.RunE == nil {
		return localRuntimeError("prepare structured handoff", errors.New("handoff command is unavailable"))
	}
	handoffCmd.SetContext(cmd.Context())
	for _, flag := range []struct {
		name  string
		value string
	}{
		{name: "to", value: agent},
		{name: "dry-run", value: strconv.FormatBool(dryRun)},
		{name: "json", value: strconv.FormatBool(asJSON)},
	} {
		if err := handoffCmd.Flags().Set(flag.name, flag.value); err != nil {
			return localRuntimeError("prepare structured handoff", err)
		}
	}
	for _, warning := range allowedWarnings {
		if err := handoffCmd.Flags().Set("allow-warning", warning); err != nil {
			return localRuntimeError("prepare structured handoff", err)
		}
	}
	return handoffCmd.RunE(handoffCmd, []string{session})
}

func defaultLocalSources() []sessionindex.Source {
	return []sessionindex.Source{
		sessionindex.NewClaudeSource(""),
		sessionindex.NewCodexSource(""),
		sessionindex.NewGeminiSource(""),
		sessionindex.NewOpenCodeSource(nil),
		sessionindex.NewGrokSource(""),
	}
}

func openRefreshedLocalIndex(
	cmd *cobra.Command,
	options localCommandOptions,
) (*sessionindex.Index, sessionindex.RefreshResult, error) {
	index, err := openLocalIndex(options)
	if err != nil {
		return nil, sessionindex.RefreshResult{}, err
	}
	refresh, err := index.Refresh(cmd.Context())
	if err != nil {
		_ = index.Close()
		return nil, refresh, localRuntimeError("refresh local session index", err)
	}
	return index, refresh, nil
}

func openLocalIndex(options localCommandOptions) (*sessionindex.Index, error) {
	home, err := config.Home()
	if err != nil {
		return nil, NewExitError(ExitConfig, err.Error())
	}
	sources := options.sources
	if sources == nil {
		sources = defaultLocalSources()
	}
	index, err := sessionindex.OpenIndex(home, sources...)
	if err != nil {
		return nil, localRuntimeError("open local session index", err)
	}
	return index, nil
}

func launchLocalRecord(
	cmd *cobra.Command,
	options localCommandOptions,
	index *sessionindex.Index,
	record sessionindex.Record,
	sourceFresh bool,
	operation string,
	dryRun bool,
	asJSON bool,
	allowedWarnings []string,
	readLine lineReader,
) error {
	plan, err := sessionindex.PlanLaunch(record, operation)
	if err != nil {
		return localLaunchError(err)
	}
	report, err := verifyLocalRecord(cmd.Context(), options, index, record, sourceFresh)
	if err != nil {
		return err
	}
	if dryRun {
		if asJSON {
			if report.Decision == preflight.DecisionBlocked {
				return blockedEnvironmentError(report, plan)
			}
			if err := validateDryRunWarningIDs(report, plan, allowedWarnings); err != nil {
				return err
			}
			return WriteJSON(cmd.OutOrStdout(), localLaunchOutput{LaunchPlan: plan, Environment: report})
		}
		PrintHuman(
			cmd.OutOrStdout(),
			"%s %q in %s",
			plan.Executable,
			plan.Args,
			doctor.RedactPath(plan.Dir),
		)
		writeEnvironmentReportHuman(cmd.OutOrStdout(), report)
		if report.Decision == preflight.DecisionBlocked {
			return blockedEnvironmentError(report, plan)
		}
		if err := validateDryRunWarningIDs(report, plan, allowedWarnings); err != nil {
			return err
		}
		return nil
	}

	writeEnvironmentReportHuman(cmd.OutOrStdout(), report)
	authorizedWarnings, err := authorizeEnvironment(
		cmd, options, report, plan, allowedWarnings, readLine,
	)
	if err != nil {
		return err
	}

	// Re-read the exact qualified identity and repeat every observation after
	// authorization. A plan or report change invalidates that authorization.
	secondRecord, _, secondFresh, err := index.RefreshAndResolve(cmd.Context(), record.Reference())
	if err != nil {
		if errors.Is(err, sessionindex.ErrNotFound) || errors.Is(err, sessionindex.ErrAmbiguous) {
			return environmentReportError(report, plan, ExitSafety, "selected session changed after environment authorization")
		}
		return localRuntimeError("refresh selected session before launch", err)
	}
	secondPlan, err := sessionindex.PlanLaunch(secondRecord, operation)
	if err != nil {
		return environmentReportError(report, plan, ExitSafety, "selected session changed after environment authorization")
	}
	secondReport, err := verifyLocalRecord(cmd.Context(), options, index, secondRecord, secondFresh)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(plan, secondPlan) || !reflect.DeepEqual(report, secondReport) {
		return environmentReportError(secondReport, secondPlan, ExitSafety, "environment changed after authorization; review and retry")
	}
	secondAuthorization, err := preflight.Authorize(secondReport, authorizedWarnings)
	if err != nil || !secondAuthorization.Allowed {
		return environmentReportError(secondReport, secondPlan, authorizationExitCode(secondAuthorization), authorizationMessage(err))
	}

	candidate, err := preflight.BaselineFromReport(secondReport, time.Now().UTC())
	if err != nil {
		return localRuntimeError("build prelaunch environment baseline", err)
	}
	runner := options.launchRunner
	if runner == nil {
		runner = sessionindex.ExecLaunchRunner{
			Stdin:              cmd.InOrStdin(),
			Stdout:             cmd.OutOrStdout(),
			Stderr:             cmd.ErrOrStderr(),
			Executable:         secondReport.Agent.ExecutablePath,
			ExecutableIdentity: secondReport.Agent.ExecutableIdentity,
			WorkspaceIdentity:  secondReport.Workspace.Workspace.Identity,
			BeforeExec: finalEnvironmentGuard(
				cmd, options, index, secondRecord, secondPlan, secondReport, authorizedWarnings,
			),
		}
	}
	if err := sessionindex.RunLaunch(cmd.Context(), secondPlan, runner); err != nil {
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			return exitErr
		}
		return localLaunchError(err)
	}
	if err := index.Store().PutPrelaunchBaseline(cmd.Context(), candidate); err != nil {
		return localRuntimeError("persist successful prelaunch environment baseline", err)
	}
	return nil
}

func finalEnvironmentGuard(
	cmd *cobra.Command,
	options localCommandOptions,
	index *sessionindex.Index,
	record sessionindex.Record,
	plan sessionindex.LaunchPlan,
	report preflight.Report,
	authorizedWarnings []string,
) func(context.Context, sessionindex.LaunchPlan) error {
	return func(ctx context.Context, runnerPlan sessionindex.LaunchPlan) error {
		if !reflect.DeepEqual(plan, runnerPlan) {
			return environmentReportError(report, plan, ExitSafety, "native launch plan changed at the execution boundary")
		}
		latest, _, fresh, err := index.RefreshAndResolve(ctx, record.Reference())
		if err != nil {
			if errors.Is(err, sessionindex.ErrNotFound) || errors.Is(err, sessionindex.ErrAmbiguous) {
				return environmentReportError(report, plan, ExitSafety, "selected session changed at the execution boundary")
			}
			return localRuntimeError("refresh selected session at the execution boundary", err)
		}
		latestPlan, err := sessionindex.PlanLaunch(latest, plan.Operation)
		if err != nil {
			return environmentReportError(report, plan, ExitSafety, "selected session changed at the execution boundary")
		}
		latestReport, err := verifyLocalRecord(ctx, options, index, latest, fresh)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(plan, latestPlan) || !reflect.DeepEqual(report, latestReport) {
			return environmentReportError(latestReport, latestPlan, ExitSafety, "environment changed at the execution boundary; review and retry")
		}
		authorization, err := preflight.Authorize(latestReport, authorizedWarnings)
		if err != nil || !authorization.Allowed {
			return environmentReportError(latestReport, latestPlan, authorizationExitCode(authorization), authorizationMessage(err))
		}
		return nil
	}
}

func blockedEnvironmentError(report preflight.Report, plan sessionindex.LaunchPlan) error {
	authorization, err := preflight.Authorize(report, nil)
	return environmentReportError(report, plan, authorizationExitCode(authorization), authorizationMessage(err))
}

type lineReader func() (line string, ok bool, err error)

func scannerLineReader(scanner *bufio.Scanner) lineReader {
	return func() (string, bool, error) {
		if scanner.Scan() {
			return scanner.Text(), true, nil
		}
		return "", false, scanner.Err()
	}
}

func verifyLocalRecord(
	ctx context.Context,
	options localCommandOptions,
	index *sessionindex.Index,
	record sessionindex.Record,
	sourceFresh bool,
) (preflight.Report, error) {
	if options.verifier == nil {
		return preflight.Report{}, localRuntimeError("verify native environment", errors.New("environment verifier is unavailable"))
	}
	var baselinePointer = (*environment.PrelaunchBaseline)(nil)
	baseline, err := index.Store().GetPrelaunchBaseline(ctx, record.Reference())
	switch {
	case err == nil:
		baselinePointer = &baseline
	case errors.Is(err, sessionindex.ErrPrelaunchBaselineNotFound):
	case err != nil:
		return preflight.Report{}, localRuntimeError("read prelaunch environment baseline", err)
	}
	report, err := options.verifier.Verify(ctx, preflight.Input{
		SessionRef:  record.Reference(),
		Agent:       record.Agent,
		Workspace:   record.Workspace,
		AgentRoot:   sessionindex.AgentRoot(record),
		Recorded:    record.RecordedEnvironment,
		Baseline:    baselinePointer,
		SourceFresh: sourceFresh,
	})
	if err != nil {
		return preflight.Report{}, localRuntimeError("verify native environment", err)
	}
	if report.SessionRef != record.Reference() {
		return preflight.Report{}, localRuntimeError("verify native environment", errors.New("environment report identity does not match selected session"))
	}
	if err := preflight.ValidateReport(report); err != nil {
		return preflight.Report{}, localRuntimeError("verify native environment", err)
	}
	return report, nil
}

func validateDryRunWarningIDs(
	report preflight.Report,
	plan sessionindex.LaunchPlan,
	allowed []string,
) error {
	if len(allowed) == 0 {
		return nil
	}
	current := make(map[string]struct{})
	for _, check := range report.Checks {
		if check.Severity == preflight.SeverityWarning {
			current[check.ID] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(allowed))
	for _, raw := range allowed {
		id := strings.TrimSpace(raw)
		if id == "" || strings.ContainsAny(id, "*?[]") {
			return environmentReportError(report, plan, ExitUsage, fmt.Sprintf("invalid environment warning ID %q", id))
		}
		if _, duplicate := seen[id]; duplicate {
			return environmentReportError(report, plan, ExitUsage, fmt.Sprintf("duplicate environment warning ID %q", id))
		}
		seen[id] = struct{}{}
		if _, exists := current[id]; !exists {
			return environmentReportError(report, plan, ExitUsage, fmt.Sprintf("environment warning ID %q is not a current warning", id))
		}
	}
	return nil
}

func authorizeEnvironment(
	cmd *cobra.Command,
	options localCommandOptions,
	report preflight.Report,
	plan sessionindex.LaunchPlan,
	allowed []string,
	readLine lineReader,
) ([]string, error) {
	if report.Decision != preflight.DecisionConfirmationRequired || len(allowed) != 0 ||
		!options.terminalCheck(cmd.InOrStdin(), cmd.OutOrStdout()) {
		authorization, err := preflight.Authorize(report, allowed)
		if err != nil || !authorization.Allowed {
			return nil, environmentReportError(report, plan, authorizationExitCode(authorization), authorizationMessage(err))
		}
		return preflight.WarningIDs(report), nil
	}
	promptInput := environmentPromptInput{
		readLine: readLine,
		restore:  func() error { return nil },
	}
	if promptInput.readLine == nil {
		var err error
		promptInput, err = newEnvironmentPromptInput(cmd.InOrStdin(), cmd.OutOrStdout())
		if err != nil {
			return nil, localRuntimeError("prepare environment confirmation", err)
		}
	}
	confirmed, confirmErr := confirmEnvironmentWarnings(cmd.Context(), cmd.OutOrStdout(), promptInput.readLine)
	restoreErr := promptInput.restore()
	if confirmErr != nil {
		return nil, confirmErr
	}
	if restoreErr != nil {
		return nil, localRuntimeError("restore terminal after environment confirmation", restoreErr)
	}
	if !confirmed {
		return nil, environmentReportError(report, plan, ExitSafety, "environment warning confirmation declined")
	}
	warnings := preflight.WarningIDs(report)
	authorization, err := preflight.Authorize(report, warnings)
	if err != nil || !authorization.Allowed {
		return nil, environmentReportError(report, plan, authorizationExitCode(authorization), authorizationMessage(err))
	}
	return warnings, nil
}

func confirmEnvironmentWarnings(ctx context.Context, writer io.Writer, readLine lineReader) (bool, error) {
	promptContext, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	for {
		if err := promptContext.Err(); err != nil {
			return false, nil
		}
		_, _ = fmt.Fprint(writer, "Continue with these environment warnings? Type yes or no [no]: ")
		line, ok, err, canceled := readLineWithContext(promptContext, readLine)
		if canceled {
			return false, nil
		}
		if promptContext.Err() != nil {
			return false, nil
		}
		if err != nil {
			return false, localRuntimeError("read environment confirmation", err)
		}
		if !ok {
			return false, nil
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "yes":
			return true, nil
		case "", "no":
			return false, nil
		default:
			PrintHuman(writer, "Enter exactly yes or no.")
		}
	}
}

type lineReadResult struct {
	line string
	ok   bool
	err  error
}

func readLineWithContext(ctx context.Context, readLine lineReader) (string, bool, error, bool) {
	result := make(chan lineReadResult, 1)
	go func() {
		line, ok, err := readLine()
		result <- lineReadResult{line: line, ok: ok, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", false, nil, true
	case value := <-result:
		return value.line, value.ok, value.err, false
	}
}

func environmentReportError(
	report preflight.Report,
	plan sessionindex.LaunchPlan,
	code int,
	message string,
) *ExitError {
	if code == 0 {
		code = ExitRuntime
	}
	err := NewExitError(code, message)
	err.Details["environment"] = report
	err.Details["launch_plan"] = plan
	return err
}

func authorizationExitCode(authorization preflight.Authorization) int {
	if authorization.ExitCode != 0 {
		return authorization.ExitCode
	}
	return ExitRuntime
}

func authorizationMessage(err error) string {
	if err == nil {
		return "environment authorization failed"
	}
	return err.Error()
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

	reader := scannerLineReader(bufio.NewScanner(cmd.InOrStdin()))
	filter := sessionindex.Filter{Limit: sessionindex.DefaultLimit}
	for {
		records, err := index.Search(cmd.Context(), filter)
		if err != nil {
			return localRuntimeError("read local session index", err)
		}
		printPicker(cmd.OutOrStdout(), records, filter.Query)
		input, ok, err := reader()
		if !ok {
			if err != nil {
				return localRuntimeError("read picker input", err)
			}
			return nil
		}
		input = strings.TrimSpace(input)
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
			record, refresh, fresh, err := index.RefreshAndResolve(cmd.Context(), records[indexValue].Reference())
			if err != nil {
				return localResolveError(err)
			}
			report, err := verifyLocalRecord(cmd.Context(), options, index, record, fresh)
			if err != nil {
				return err
			}
			if err := writeLocalInspect(cmd, record, report, refresh.Warnings, false); err != nil {
				return err
			}
			continue
		case strings.HasPrefix(strings.ToLower(input), "f "):
			indexValue, ok := pickerIndex(input[2:], len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Invalid session number.")
				continue
			}
			record, _, fresh, err := index.RefreshAndResolve(cmd.Context(), records[indexValue].Reference())
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(
				cmd,
				options,
				index,
				record,
				fresh,
				sessionindex.OperationFork,
				false,
				false,
				nil,
				reader,
			)
		case strings.HasPrefix(strings.ToLower(input), "h "):
			indexValue, ok := pickerIndex(input[2:], len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Invalid session number.")
				continue
			}
			PrintHuman(cmd.OutOrStdout(), "Destination agent (claude or codex):")
			agent, ok, err := reader()
			if err != nil {
				return localRuntimeError("read picker input", err)
			}
			if !ok {
				return nil
			}
			agent = strings.TrimSpace(agent)
			PrintHuman(cmd.ErrOrStderr(), "%s.", handoffHumanPrefix(agent))
			return runHandoffAlias(cmd, records[indexValue].Reference(), agent, false, false, nil)
		default:
			indexValue, ok := pickerIndex(input, len(records))
			if !ok {
				PrintHuman(cmd.OutOrStdout(), "Enter a number, /text, i NUMBER, f NUMBER, h NUMBER, or q.")
				continue
			}
			record, _, fresh, err := index.RefreshAndResolve(cmd.Context(), records[indexValue].Reference())
			if err != nil {
				return localResolveError(err)
			}
			return launchLocalRecord(
				cmd,
				options,
				index,
				record,
				fresh,
				sessionindex.OperationResume,
				false,
				false,
				nil,
				reader,
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
	PrintHuman(writer, "Choose NUMBER, /text, i NUMBER, f NUMBER, h NUMBER (hand off to another agent), or q:")
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
	report preflight.Report,
	warnings []sessionindex.Warning,
	asJSON bool,
) error {
	if asJSON {
		return WriteJSON(cmd.OutOrStdout(), localInspectOutput{
			Session:     record,
			Environment: report,
			Warnings:    publicLocalWarnings(warnings),
		})
	}
	writeLocalWarnings(cmd.ErrOrStderr(), warnings)
	PrintHuman(cmd.OutOrStdout(), "Session: %s", record.Reference())
	PrintHuman(cmd.OutOrStdout(), "Agent: %s", record.Agent)
	PrintHuman(cmd.OutOrStdout(), "Title: %s", record.Title)
	PrintHuman(cmd.OutOrStdout(), "Project: %s", record.Project)
	// Human output never prints absolute workspace paths (RC3 privacy finding).
	PrintHuman(cmd.OutOrStdout(), "Workspace: %s", doctor.RedactPath(record.Workspace))
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
	writeEnvironmentReportHuman(cmd.OutOrStdout(), report)
	return nil
}

func writeEnvironmentReportHuman(writer io.Writer, report preflight.Report) {
	PrintHuman(writer, "Environment decision: %s", report.Decision)
	if report.BlockExitCode != 0 {
		PrintHuman(writer, "Environment block exit code: %d", report.BlockExitCode)
	}
	for _, check := range report.Checks {
		PrintHuman(
			writer,
			"Environment check: %s status=%s severity=%s provenance=%s — %s",
			check.ID,
			check.Status,
			check.Severity,
			check.Provenance,
			check.Message,
		)
		if value, ok := environmentReportValueHuman(check.Expected); ok {
			PrintHuman(writer, "  Expected: %s", value)
		}
		if value, ok := environmentReportValueHuman(check.Actual); ok {
			PrintHuman(writer, "  Actual: %s", value)
		}
		if check.Repair != "" {
			PrintHuman(writer, "  Repair: %s", check.Repair)
		}
	}
}

func environmentReportValueHuman(value any) (string, bool) {
	switch current := value.(type) {
	case nil:
		return "", false
	case bool:
		return strconv.FormatBool(current), true
	case string:
		// Redact absolute paths that may appear in expected/actual string fields.
		return strconv.Quote(doctor.RedactPath(current)), true
	case workspace.WorkingTreeState:
		return strconv.Quote(string(current)), true
	case []string:
		quoted := make([]string, 0, len(current))
		for _, item := range current {
			quoted = append(quoted, strconv.Quote(doctor.RedactPath(item)))
		}
		return "[" + strings.Join(quoted, ", ") + "]", true
	default:
		return "", false
	}
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
		sessionindex.AgentOpenCode,
		sessionindex.AgentGrok:
		return nil
	case "all":
		if allowAll {
			return nil
		}
	}
	return NewExitError(
		ExitUsage,
		"invalid agent; expected claude, codex, gemini, opencode, grok, or all",
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
	if errors.Is(err, sessionindex.ErrLaunchBoundaryChanged) {
		return NewExitError(ExitSafety, err.Error())
	}
	if errors.Is(err, sessionindex.ErrNonInteractiveLaunch) {
		return NewExitError(ExitSafety, err.Error())
	}
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
