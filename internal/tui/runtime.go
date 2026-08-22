// Package tui holds the interactive Reinstate surfaces built on Bubble Tea.
//
// The layering rule is absolute: tui is a view. It reads through the engine
// packages (sessionindex, preflight, handoff, sync) and returns an Intent
// describing what the user chose. It never launches a vendor process, never
// writes into a vendor store, and never decides policy. Engine packages never
// import tui or ui.
//
// The reason a surface returns an Intent instead of acting is the terminal.
// Resuming a session hands the terminal to a vendor CLI that draws its own
// full-screen UI. That can only happen after Bubble Tea has restored the
// terminal completely, which means after the program has exited — so the
// program exits first, and the caller acts second.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package tui

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Action is what an interactive surface decided to do.
type Action string

const (
	// ActionNone means the user cancelled. The caller exits 0 silently.
	ActionNone Action = ""
	// ActionResume continues a session in its own vendor agent.
	ActionResume Action = "resume"
	// ActionFork branches a session through its vendor's native fork path.
	ActionFork Action = "fork"
	// ActionHandoff starts a new session in a different agent from a capsule.
	ActionHandoff Action = "handoff"
	// ActionInspect prints the full metadata report for a session.
	ActionInspect Action = "inspect"
	// ActionCommand runs another rein subcommand, named in Intent.Command.
	// It is how the command palette reaches verbs that are not session actions.
	ActionCommand Action = "command"
)

// Intent is the result of an interactive surface: what to do, to what, and
// with which acknowledgements. It is deliberately flat and free of engine
// types so the boundary stays narrow.
type Intent struct {
	Action Action
	// Reference is the canonical agent:native-session-id.
	Reference string
	// Destination is the handoff target agent key. Empty unless ActionHandoff.
	Destination string
	// Policy is the handoff projection policy. Empty unless ActionHandoff.
	Policy string
	// Command is the rein subcommand to run. Set only for ActionCommand, and
	// always a fixed identifier chosen from a table, never user-typed text.
	Command string
	// AcknowledgedWarnings holds exact warning IDs the user ticked. These are
	// passed straight through to the same authorization call the flag path
	// uses; the TUI grants nothing the flags could not.
	AcknowledgedWarnings []string
}

// Chosen reports whether the user picked an action rather than cancelling.
func (i Intent) Chosen() bool { return i.Action != ActionNone }

// Surface is a Bubble Tea model that resolves to an Intent.
//
// Implementations must return a zero Intent when the user cancels, and must
// never perform the action themselves.
type Surface interface {
	tea.Model
	// Intent returns the decision after the program has exited.
	Intent() Intent
	// Err returns a fatal error encountered while running, if any.
	Err() error
}

// ErrNotInteractive is returned when Run is called without a usable terminal.
// Callers translate it into the usage-coded refusal with a --json hint.
var ErrNotInteractive = errors.New("interactive surface requires a terminal")

// RunOptions configure one interactive program.
type RunOptions struct {
	In         io.Reader
	Out        io.Writer
	Capability ui.Capability
	// AltScreen puts the program on the alternate buffer, leaving the user's
	// scrollback untouched. Full-screen surfaces set it; inline prompts do not.
	AltScreen bool
	// teaOptions is an escape hatch for tests to inject an input driver.
	teaOptions []tea.ProgramOption
}

// Run starts an interactive surface and returns the user's intent.
//
// It refuses non-interactive capabilities rather than degrading, because the
// caller has already chosen this path by consulting the same capability.
func Run(ctx context.Context, surface Surface, opts RunOptions) (Intent, error) {
	if !opts.Capability.Mode.Interactive() {
		return Intent{}, ErrNotInteractive
	}
	programOptions := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithInput(opts.In),
		tea.WithOutput(opts.Out),
	}
	if opts.AltScreen {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	programOptions = append(programOptions, opts.teaOptions...)

	program := tea.NewProgram(surface, programOptions...)
	finalModel, err := program.Run()
	if err != nil {
		// A cancelled context is the user pressing ctrl-c or the parent command
		// shutting down. Neither is a failure to report.
		if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, context.Canceled) {
			return Intent{}, nil
		}
		return Intent{}, fmt.Errorf("interactive surface failed: %w", err)
	}
	resolved, ok := finalModel.(Surface)
	if !ok {
		return Intent{}, fmt.Errorf("interactive surface returned unexpected model %T", finalModel)
	}
	if surfaceErr := resolved.Err(); surfaceErr != nil {
		return Intent{}, surfaceErr
	}
	return resolved.Intent(), nil
}
