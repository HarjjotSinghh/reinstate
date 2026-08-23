// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package cli

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/config"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/readiness"
	"github.com/HarjjotSinghh/reinstate/internal/tui/switcher"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// switcherDaemonLine is the daemon summary for the switcher's status line,
// read from the status file the daemon writes; empty when none has run.
func switcherDaemonLine() string {
	home, err := config.Home()
	if err != nil {
		return ""
	}
	return daemonSummaryLine(home, time.Now())
}

// runInteractiveSwitcher opens the full-screen session switcher and then
// performs whatever the user chose.
//
// The two steps are strictly ordered. Bubble Tea owns the terminal while the
// list is on screen; a vendor CLI owns it afterwards. Resuming from inside the
// program would hand a half-restored terminal to the vendor, so the program
// exits first and the launch happens here, against a clean terminal.
func runInteractiveSwitcher(
	cmd *cobra.Command,
	options localCommandOptions,
	capability ui.Capability,
) error {
	index, refresh, err := openRefreshedLocalIndex(cmd, options, "")
	if err != nil {
		return err
	}
	defer func() { _ = index.Close() }()

	// Readiness is computed in the background for the rows on screen. The
	// verifier is the same one every launch path uses, so a green dot means
	// exactly what a clean `rein resume` would mean.
	prober := newReadinessProber(options, index, refresh)
	scope := currentProjectScope()
	model := switcher.New(switcher.Options{
		Theme:        ui.NewTheme(capability),
		Capability:   capability,
		Loader:       indexLoader{index: index, ctx: cmd.Context(), workspace: scope.root},
		Readiness:    prober,
		Context:      cmd.Context(),
		Project:      scope.filter,
		ProjectLabel: scope.label,
		Limit:        sessionindex.DefaultLimit,
		Clipboard:    tui.OSC52Clipboard(cmd.OutOrStdout()),
		Daemon:       switcherDaemonLine(),
	})

	intent, err := tui.Run(cmd.Context(), model, tui.RunOptions{
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		Capability: capability,
		AltScreen:  true,
	})
	if err != nil {
		return localRuntimeError("run the session switcher", err)
	}
	return performIntent(cmd, options, index, intent, capability)
}

// performIntent executes a decision made in an interactive surface by calling
// exactly the same code paths the flag-driven commands call. Nothing an
// interactive surface can do is reachable only from an interactive surface.
func performIntent(
	cmd *cobra.Command,
	options localCommandOptions,
	index *sessionindex.Index,
	intent tui.Intent,
	capability ui.Capability,
) error {
	if !intent.Chosen() {
		return nil
	}
	switch intent.Action {
	case tui.ActionResume, tui.ActionFork:
		operation := sessionindex.OperationResume
		if intent.Action == tui.ActionFork {
			operation = sessionindex.OperationFork
		}
		record, _, fresh, err := index.RefreshAndResolve(cmd.Context(), intent.Reference)
		if err != nil {
			return localResolveError(err)
		}
		return launchLocalRecord(
			cmd,
			options,
			index,
			record,
			fresh,
			operation,
			false,
			false,
			intent.AcknowledgedWarnings,
			scannerLineReader(bufio.NewScanner(cmd.InOrStdin())),
		)

	case tui.ActionInspect:
		record, refresh, fresh, err := index.RefreshAndResolve(cmd.Context(), intent.Reference)
		if err != nil {
			return localResolveError(err)
		}
		report, err := verifyLocalRecord(cmd.Context(), options, index, record, fresh)
		if err != nil {
			return err
		}
		return writeLocalInspect(cmd, record, report, refresh.Warnings, false)

	case tui.ActionCommand:
		// The palette reaches other rein verbs by name. The name comes from a
		// fixed table in the switcher, never from typed text, so this resolves a
		// known subcommand rather than interpreting user input as one.
		return runNamedCommand(cmd, intent.Command)

	case tui.ActionHandoff:
		// The switcher chooses only the source. Destination and projection
		// policy are chosen in the studio, where their consequences are
		// measured and shown before anything is written.
		if intent.Destination == "" {
			return runHandoffStudio(
				cmd,
				options,
				capability,
				intent.Reference,
				handoffSourceAgent(intent.Reference),
			)
		}
		PrintHuman(cmd.ErrOrStderr(), "%s.", handoffHumanPrefix(intent.Destination))
		return runHandoffWith(cmd, handoffAliasOptions{
			Session:         intent.Reference,
			Agent:           intent.Destination,
			Policy:          intent.Policy,
			AllowedWarnings: intent.AcknowledgedWarnings,
		})
	}
	return nil
}

// indexLoader adapts the session index to the switcher's Loader.
type indexLoader struct {
	index *sessionindex.Index
	ctx   context.Context
	// workspace narrows results to one repository root. It is applied here
	// rather than in the index because the index matches Filter.Project as a
	// literal substring, and a workspace path cannot be compared literally
	// across platforms — see normalizedWorkspace.
	workspace string
}

// Load implements switcher.Loader.
func (l indexLoader) Load(filter sessionindex.Filter) ([]sessionindex.Record, error) {
	records, err := l.index.Search(l.ctx, filter)
	if err != nil || l.workspace == "" || filter.Project == "" {
		return records, err
	}
	root := normalizedWorkspace(l.workspace)
	kept := records[:0]
	for _, record := range records {
		if strings.HasPrefix(normalizedWorkspace(record.Workspace), root) {
			kept = append(kept, record)
		}
	}
	return kept, nil
}

// normalizedWorkspace puts a workspace path into a comparable form.
//
// Vendors record the working directory in whatever form their own runtime
// produced, and the index stores it verbatim. On Windows that means one session
// file can hold a forward-slash path while the process standing in that same
// directory reports a backslash one. Comparing those literally finds nothing,
// which is how the switcher came to show an empty list for every project on
// Windows.
//
// The separator is folded with an explicit replacement rather than
// filepath.ToSlash, and cleaned with the slash-based path package, because
// filepath is compiled for the host: on Unix it does not treat a backslash as a
// separator at all. A session recorded on Windows and read on macOS is the
// ordinary case for this product, so the normalization cannot depend on which
// machine is doing the reading.
//
// Case is folded too. That is wrong in principle for a case-sensitive
// filesystem, but this narrows a list rather than resolving identity, and a
// drive letter or home directory differing only in case is far more common than
// two sibling checkouts differing only in case.
func normalizedWorkspace(value string) string {
	folded := strings.ReplaceAll(value, `\`, "/")
	return strings.ToLower(path.Clean(folded))
}

// projectScope is how the switcher is narrowed to the repository the user is
// standing in.
type projectScope struct {
	// filter goes to the index. It is the repository's directory name, which
	// matches the project column and contains no path separator, so it behaves
	// identically on every platform.
	filter string
	// label is what the header shows.
	label string
	// root is the absolute repository root, used to narrow the index's broader
	// name match down to this exact checkout.
	root string
}

// currentProjectScope resolves the scope for the current working directory.
//
// The index is asked for the repository *name* rather than its path: Filter
// .Project is a literal substring match, and a path is not portable enough to
// match literally. The exact checkout is then selected from those results by
// comparing normalized workspace paths, so two repositories that happen to
// share a directory name do not bleed into each other.
func currentProjectScope() projectScope {
	working, err := os.Getwd()
	if err != nil {
		return projectScope{}
	}
	root := gitRoot(working)
	if root == "" {
		return projectScope{}
	}
	name := filepath.Base(root)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return projectScope{}
	}
	return projectScope{filter: name, label: name, root: root}
}

// gitRoot walks up from start looking for a repository root. It stops at the
// filesystem root and returns empty when there is none, which puts the switcher
// into all-projects scope rather than inventing a project.
func gitRoot(start string) string {
	current := filepath.Clean(start)
	for {
		// A worktree records .git as a file rather than a directory, so this
		// deliberately accepts either.
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current || strings.TrimSpace(parent) == "" {
			return ""
		}
		current = parent
	}
}

// authorizeEnvironmentInteractively runs the warning checklist and converts the
// result into the exact acknowledgement list the flag path would have produced.
//
// It grants nothing extra: the identifiers it collects go straight into
// preflight.Authorize, the same call the --allow-environment-warning flags feed.
// The only thing that changes is how the user says yes.
func authorizeEnvironmentInteractively(
	cmd *cobra.Command,
	capability ui.Capability,
	report preflight.Report,
	plan sessionindex.LaunchPlan,
) ([]string, error) {
	checklist := readiness.NewChecklist(readiness.Options{
		Theme:      ui.NewTheme(capability),
		Capability: capability,
		Report:     report,
		Reference:  report.SessionRef,
		Operation:  plan.Operation,
		Clipboard:  tui.OSC52Clipboard(cmd.OutOrStdout()),
	})
	if _, err := tui.Run(cmd.Context(), checklist, tui.RunOptions{
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		Capability: capability,
		AltScreen:  true,
	}); err != nil {
		return nil, localRuntimeError("confirm environment warnings", err)
	}
	if !checklist.Confirmed() {
		return nil, environmentReportError(report, plan, ExitSafety, "environment warning confirmation declined")
	}
	acknowledged := checklist.Acknowledged()
	authorization, err := preflight.Authorize(report, acknowledged)
	if err != nil || !authorization.Allowed {
		return nil, environmentReportError(report, plan, authorizationExitCode(authorization), authorizationMessage(err))
	}
	return acknowledged, nil
}

// runNamedCommand executes another rein subcommand chosen from the palette.
//
// It runs the real command, so an interactive route to `rein doctor` behaves
// exactly like typing `rein doctor`, including its refusals and exit code.
func runNamedCommand(cmd *cobra.Command, name string) error {
	switch name {
	case "doctor", "status", "push", "pull":
	default:
		return localRuntimeError("run command", errors.New("unknown command "+name))
	}
	target, _, err := cmd.Root().Find([]string{name})
	if err != nil || target == nil || target.RunE == nil {
		return localRuntimeError("run command", errors.New(name+" is unavailable"))
	}
	target.SetContext(cmd.Context())
	return target.RunE(target, nil)
}

// newReadinessProber builds the switcher's readiness provider.
//
// The freshness argument is load-bearing, and is the reason this is a named
// function rather than a closure written inline. `source.fresh` is a *blocking*
// check: a report built with SourceFresh false is DecisionBlocked no matter how
// healthy the environment actually is, so passing a constant there makes every
// row in the list read "cannot resume". Freshness is per-agent and comes from
// the refresh the caller already performed.
func newReadinessProber(
	options localCommandOptions,
	index *sessionindex.Index,
	refresh sessionindex.RefreshResult,
) *readiness.Prober {
	return readiness.New(func(ctx context.Context, record sessionindex.Record) (preflight.Report, error) {
		return verifyLocalRecord(ctx, options, index, record, refresh.SourceFresh(record.Agent))
	})
}
