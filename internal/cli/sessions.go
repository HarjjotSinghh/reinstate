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

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/doctor"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func init() {
	sessionindex.SetNativeLaunchLookup(catalogNativeLaunch)
	agentcheck.SetDefinitions(catalogAgentDefinitions())
	processcheck.SetSpecs(catalogProcessSpecs())
}

type localCommandOptions struct {
	sources       []sessionindex.Source
	launchRunner  sessionindex.LaunchRunner
	verifier      preflight.Verifier
	terminalCheck func(io.Reader, io.Writer) bool
	// processChecker reports whether a vendor still holds a session file. The
	// handoff pipeline requires it, and the interactive surfaces reach the
	// pipeline without going through the handoff command's own options.
	processChecker AgentProcessChecker
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
			index, refresh, err := openRefreshedLocalIndex(cmd, options, agent)
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
	cmd.Flags().StringVar(&agent, "agent", "all", "agent filter: "+agentFilterHelp(agents.TierDiscover, true))
	registerAgentCompletion(cmd, "agent", agents.TierDiscover, true)
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
			index, refresh, err := openRefreshedLocalIndex(cmd, options, filter.Agent)
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
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: "+agentFilterHelp(agents.TierDiscover, true))
	registerAgentCompletion(cmd, "agent", agents.TierDiscover, true)
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
			index, refresh, err := openRefreshedLocalIndex(cmd, options, filter.Agent)
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
	cmd.Flags().StringVar(&filter.Agent, "agent", "all", "agent filter: "+agentFilterHelp(agents.TierResume, true))
	registerAgentCompletion(cmd, "agent", agents.TierResume, true)
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
				// Matrix A7 wants one answer for every key below T3, whether or
				// not the session exists. A resolved record refuses later with
				// its own read-only reason, which is more specific, so this only
				// covers the case where resolution could not produce one.
				if tierErr := refuseBelowResumeTier(args[0], action); tierErr != nil {
					return tierErr
				}
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
		cmd.Flags().StringVar(&withAgent, "with", "", "continue the same task through a structured handoff to "+agentFilterHelp(agents.TierHandoffTo, false))
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
	return runHandoffWith(cmd, handoffAliasOptions{
		Session:         session,
		Agent:           agent,
		DryRun:          dryRun,
		JSON:            asJSON,
		AllowedWarnings: allowedWarnings,
	})
}

// handoffAliasOptions is one invocation of the real handoff command from
// another surface.
type handoffAliasOptions struct {
	Session         string
	Agent           string
	Policy          string
	DryRun          bool
	JSON            bool
	NoLaunch        bool
	AllowedWarnings []string
}

// runHandoffWith re-enters the real `rein handoff` command with flags set.
//
// Going through the command itself rather than calling the pipeline directly is
// deliberate: every caller, interactive or not, then gets the identical
// validation, refusal, and output path. An interactive surface cannot acquire a
// capability the flags do not already have.
func runHandoffWith(cmd *cobra.Command, options handoffAliasOptions) error {
	handoffCmd, _, err := cmd.Root().Find([]string{"handoff"})
	if err != nil || handoffCmd == nil || handoffCmd.RunE == nil {
		return localRuntimeError("prepare structured handoff", errors.New("handoff command is unavailable"))
	}
	handoffCmd.SetContext(cmd.Context())
	flags := []struct {
		name  string
		value string
	}{
		{name: "to", value: options.Agent},
		{name: "dry-run", value: strconv.FormatBool(options.DryRun)},
		{name: "json", value: strconv.FormatBool(options.JSON)},
	}
	if options.Policy != "" {
		flags = append(flags, struct {
			name  string
			value string
		}{name: "policy", value: options.Policy})
	}
	if options.NoLaunch {
		flags = append(flags, struct {
			name  string
			value string
		}{name: "no-launch", value: "true"})
	}
	for _, flag := range flags {
		if err := handoffCmd.Flags().Set(flag.name, flag.value); err != nil {
			return localRuntimeError("prepare structured handoff", err)
		}
	}
	for _, warning := range options.AllowedWarnings {
		if err := handoffCmd.Flags().Set("allow-warning", warning); err != nil {
			return localRuntimeError("prepare structured handoff", err)
		}
	}
	return handoffCmd.RunE(handoffCmd, []string{options.Session})
}

func defaultLocalSources() []sessionindex.Source {
	var sources []sessionindex.Source
	for _, descriptor := range agents.Capable(agents.CapabilityIndex) {
		if descriptor.NewIndexSource == nil {
			continue
		}
		source, err := descriptor.NewIndexSource(agents.Env{})
		if err != nil || source == nil {
			continue
		}
		sources = append(sources, source)
	}
	return sources
}

// openRefreshedLocalIndex refreshes only the selected agent's source when the
// caller narrowed the request. Scanning every vendor to answer a question about
// one of them made a single-agent query cost the same as a full refresh, and
// let one slow source delay a request that never concerned it.
func openRefreshedLocalIndex(
	cmd *cobra.Command,
	options localCommandOptions,
	agent string,
) (*sessionindex.Index, sessionindex.RefreshResult, error) {
	index, err := openLocalIndex(options)
	if err != nil {
		return nil, sessionindex.RefreshResult{}, err
	}
	var refresh sessionindex.RefreshResult
	if selected := strings.ToLower(strings.TrimSpace(agent)); selected != "" && selected != "all" {
		refresh, err = index.RefreshAgent(cmd.Context(), selected)
	} else {
		refresh, err = index.Refresh(cmd.Context())
	}
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

	// On a terminal that can host the interactive surfaces the report is
	// summarized rather than dumped. A clean launch does not need eleven
	// passing checks printed at it — that reads as an error wall — and a launch
	// with warnings is about to show them again in the acknowledgement
	// checklist. Plain and non-TTY output is unchanged.
	if capability := resolveCapability(cmd, options, false); capability.Mode.Interactive() {
		writeEnvironmentSummaryHuman(cmd.OutOrStdout(), ui.NewTheme(capability), report)
	} else {
		writeEnvironmentReportHuman(cmd.OutOrStdout(), report)
	}
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
		SessionID:   record.ID,
		SessionPath: record.SourcePath,
		ProjectRoot: record.Workspace,
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
	// An interactive terminal gets the checklist, where acknowledging a warning
	// is a spacebar rather than an identifier retyped by hand. Everything else
	// keeps the frozen yes/no prompt.
	if capability := resolveCapability(cmd, options, false); capability.Mode.Interactive() {
		return authorizeEnvironmentInteractively(cmd, capability, report, plan)
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

// runSessionPicker is bare `rein`. Whether this is a terminal at all, and
// whether it can host a full-screen program, are different questions. The first
// decides the refusal below and is unchanged. The second only chooses which
// switcher to draw.
func runSessionPicker(cmd *cobra.Command, options localCommandOptions) error {
	if !options.terminalCheck(cmd.InOrStdin(), cmd.OutOrStdout()) {
		_ = cmd.Help()
		return NewExitError(
			ExitUsage,
			"interactive session picker requires a terminal; use `rein sessions --json`",
		)
	}
	if capability := resolveCapability(cmd, options, false); capability.Mode.Interactive() {
		return runInteractiveSwitcher(cmd, options, capability)
	}
	return runPlainSessionPicker(cmd, options)
}

// runPlainSessionPicker is the numbered switcher. Its output is frozen: it is
// what scripts, recorded acceptance evidence, and every terminal that cannot
// host a full-screen program still see.
func runPlainSessionPicker(cmd *cobra.Command, options localCommandOptions) error {
	index, _, err := openRefreshedLocalIndex(cmd, options, "")
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
			PrintHuman(cmd.OutOrStdout(), "Destination agent (%s):", agentChoiceProse(agents.TierHandoffTo))
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

// writeEnvironmentSummaryHuman condenses the environment report for a terminal
// that is about to hand itself to a vendor CLI.
//
// It prints what the reader can act on and nothing else: one line when the
// environment verified cleanly, and only the checks that are not informational
// when something needs attention. The full report remains one `rein inspect`
// away, and every non-interactive caller still gets it in full.
func writeEnvironmentSummaryHuman(writer io.Writer, theme ui.Theme, report preflight.Report) {
	actionable := make([]preflight.Check, 0, len(report.Checks))
	for _, check := range report.Checks {
		if check.Severity != preflight.SeverityInfo {
			actionable = append(actionable, check)
		}
	}
	if len(actionable) == 0 {
		PrintHuman(
			writer,
			"%s environment verified — %s passed",
			theme.Ready.Render(theme.Glyphs.ReadyMark),
			pluralCount(len(report.Checks), "check", "checks"),
		)
		return
	}
	// Warnings are about to be listed by the acknowledgement checklist, so
	// repeating them here would ask the reader to read the same thing twice.
	if report.Decision == preflight.DecisionConfirmationRequired {
		return
	}
	for _, check := range actionable {
		PrintHuman(
			writer,
			"%s %s — %s",
			theme.Blocked.Render(theme.Glyphs.BlockedMark),
			check.ID,
			ui.Sanitize(check.Message),
		)
		if repair := ui.Sanitize(check.Repair); repair != "" {
			PrintHuman(writer, "  %s %s", theme.Glyphs.TrailLink, repair)
		}
	}
}

// pluralCount renders "1 check" or "3 checks".
func pluralCount(count int, singular, many string) string {
	word := many
	if count == 1 {
		word = singular
	}
	return fmt.Sprintf("%d %s", count, word)
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
	return validateCatalogAgent(agent, allowAll, agents.TierDiscover, "invalid agent; expected %s")
}

func validateNativeAgent(agent string, allowAll bool) error {
	return validateCatalogAgent(agent, allowAll, agents.TierResume, "invalid native agent; expected %s")
}

// validateDestinationAgent gates `--to` / `--with` on T4, which is the tier
// that actually has a handoff destination. It used to gate on T3, matching
// neither the flag's own help text nor the ladder: a T3 agent that is not yet
// a destination would pass usage validation and then fail deep in the pipeline
// with "unknown destination agent", instead of being told which agents can
// receive a handoff.
func validateDestinationAgent(agent string) error {
	return validateCatalogAgent(agent, false, agents.TierHandoffTo, "invalid destination agent; expected %s")
}

func validateCatalogAgent(agent string, allowAll bool, min agents.Tier, message string) error {
	agent = strings.ToLower(strings.TrimSpace(agent))
	keys := catalogKeysAtLeast(min)
	if agent == "all" && allowAll {
		return nil
	}
	for _, key := range keys {
		if agent == key {
			return nil
		}
	}
	return NewExitError(ExitUsage, fmt.Sprintf(message, agentChoiceProseWithAll(keys, allowAll)))
}

// registerAgentCompletion offers exactly the catalog keys a flag accepts, so
// shell completion agrees with the flag's own validation instead of leaving
// the user to guess. Matrix H6 requires the correct keys per command.
func registerAgentCompletion(cmd *cobra.Command, flag string, min agents.Tier, allowAll bool) {
	_ = cmd.RegisterFlagCompletionFunc(
		flag,
		func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			keys := catalogKeysAtLeast(min)
			if allowAll {
				keys = append([]string{"all"}, keys...)
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		},
	)
}

func catalogKeysAtLeast(min agents.Tier) []string {
	descriptors := agents.AtLeast(min)
	keys := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		keys = append(keys, descriptor.Key)
	}
	return keys
}

func agentFilterHelp(min agents.Tier, includeAll bool) string {
	keys := catalogKeysAtLeast(min)
	if includeAll {
		keys = append(keys, "all")
	}
	return strings.Join(keys, "|")
}

func agentChoiceProse(min agents.Tier) string {
	return joinChoiceProse(catalogKeysAtLeast(min))
}

func agentChoiceProseWithAll(keys []string, includeAll bool) string {
	if includeAll {
		keys = append(append([]string{}, keys...), "all")
	}
	return joinChoiceProse(keys)
}

func joinChoiceProse(keys []string) string {
	switch len(keys) {
	case 0:
		return ""
	case 1:
		return keys[0]
	case 2:
		return keys[0] + " or " + keys[1]
	default:
		return strings.Join(keys[:len(keys)-1], ", ") + ", or " + keys[len(keys)-1]
	}
}

func catalogAgentDefinitions() map[string]agentcheck.Definition {
	out := map[string]agentcheck.Definition{}
	for _, descriptor := range agents.All() {
		if descriptor.Native == nil || descriptor.Version == nil {
			continue
		}
		desc := descriptor
		out[desc.Key] = agentcheck.Definition{
			Executable:            desc.Native.Executable,
			Layout:                desc.Storage.Layout,
			Marker:                desc.Storage.Marker,
			RootEnvironment:       desc.Storage.RootEnv,
			RootEnvironmentSuffix: desc.Storage.RootEnvSuffix,
			Roots: func(home string) []string {
				if desc.Storage.Roots == nil {
					return nil
				}
				var roots []string
				for _, root := range desc.Storage.Roots(agents.HomeDir(home)) {
					if root.Matches(agents.CurrentOS()) {
						roots = append(roots, root.Path)
					}
				}
				return roots
			},
			Parse: func(output agentcheck.VersionOutput) (string, bool) {
				if desc.Version.Parse == nil {
					return "", false
				}
				return desc.Version.Parse(agents.VersionOutput{Stdout: output.Stdout, Stderr: output.Stderr})
			},
			Min: desc.Version.Min,
			Max: desc.Version.Max,
		}
	}
	return out
}

func catalogProcessSpecs() map[string]processcheck.Spec {
	out := map[string]processcheck.Spec{}
	for _, descriptor := range agents.All() {
		identities := make([]processcheck.Identity, 0, len(descriptor.Process.Identify))
		for _, identity := range descriptor.Process.Identify {
			identities = append(identities, processcheck.Identity{Name: identity.Name, Value: identity.Value})
		}
		out[descriptor.Key] = processcheck.Spec{
			Images:      append([]string(nil), descriptor.Process.Images...),
			NodeMarkers: append([]string(nil), descriptor.Process.NodeMarkers...),
			Identify:    identities,
		}
	}
	return out
}

func catalogNativeLaunch(agent, operation, sessionID string) (string, []string, bool) {
	descriptor, ok := agents.Get(strings.ToLower(strings.TrimSpace(agent)))
	if !ok || descriptor.Native == nil {
		return "", nil, false
	}
	var template []string
	switch operation {
	case sessionindex.OperationResume:
		template = descriptor.Native.Resume
	case sessionindex.OperationFork:
		template = descriptor.Native.Fork
	default:
		return "", nil, false
	}
	if descriptor.Native.Executable == "" || len(template) == 0 {
		return "", nil, false
	}
	// Last gate before a session identifier becomes argv. A vendor whose
	// resume flag also accepts something other than an ID — Grok Build's
	// `--resume [<SESSION_ID_OR_TITLE>]` resolves any non-UUID value as a
	// title — declares the shape it requires, and a value of any other shape
	// never reaches the command line. The indexed record is already refused
	// upstream; this backstop covers every other route to PlanLaunch.
	if !descriptor.Native.SessionIDAllowed(sessionID) {
		return "", nil, false
	}
	return descriptor.Native.Executable, sessionindex.ApplyArgvTemplate(template, sessionID), true
}

// refuseBelowResumeTier rejects a native resume or fork for a catalog agent
// below TierResume with the compatibility exit code and a stated reason,
// independent of whether the referenced session happens to exist.
func refuseBelowResumeTier(reference, operation string) error {
	agent, _, qualified := sessionindex.ParseCompositeReference(reference)
	if !qualified {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(agent))
	for _, descriptor := range agents.All() {
		if descriptor.Key != key {
			continue
		}
		if descriptor.Tier >= agents.TierResume {
			return nil
		}
		reason := fmt.Sprintf(
			"%s is tier %s; native %s is unsupported",
			descriptor.DisplayName, descriptor.Tier, operation,
		)
		if descriptor.Tier == agents.TierKnown && descriptor.T0Reason != "" {
			reason = fmt.Sprintf(
				"%s is tier %s (%s); native %s is unsupported",
				descriptor.DisplayName, descriptor.Tier, descriptor.T0Reason, operation,
			)
		}
		return NewExitError(ExitCompatibility, fmt.Sprintf(
			"%s: %s", sessionindex.ErrNativeActionUnsupported, reason,
		))
	}
	return nil
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
