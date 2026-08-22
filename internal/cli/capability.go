// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// plainFlagName is the global opt-out that forces the frozen non-interactive
// output on a terminal that could host the TUI.
const plainFlagName = "plain"

// resolveCapability applies the rendering degradation ladder for one command
// run, threading the command's own streams and the test-injectable terminal
// check through to internal/ui.
//
// Every interactive entry point calls this and branches on the result. Nothing
// else in internal/cli is allowed to ask whether it is on a terminal, so the
// ladder has exactly one implementation and one place to test.
func resolveCapability(cmd *cobra.Command, options localCommandOptions, asJSON bool) ui.Capability {
	return ui.Detect(cmd.InOrStdin(), cmd.OutOrStdout(), ui.Options{
		JSON:          asJSON || rootJSONFlag(cmd),
		Plain:         plainRequested(cmd),
		Getenv:        os.Getenv,
		TerminalCheck: interactiveStreamCheck(options.terminalCheck),
		Size:          capabilitySizeProbe(),
	})
}

// interactiveStreamCheck requires both a terminal and real console handles.
//
// A full-screen program puts the terminal into raw mode, which needs a file
// descriptor; it cannot drive a pipe, a buffer, or a string. The injected
// terminal check answers a different question — "should this behave as a TTY" —
// and tests set it true while passing in-memory streams in order to exercise
// the line-oriented prompts. Honouring that alone would start a Bubble Tea
// program against a bytes.Buffer, which blocks forever.
//
// Both conditions are genuinely required, so both are checked.
func interactiveStreamCheck(injected func(io.Reader, io.Writer) bool) func(io.Reader, io.Writer) bool {
	return func(in io.Reader, out io.Writer) bool {
		if injected != nil && !injected(in, out) {
			return false
		}
		_, inputIsFile := in.(*os.File)
		_, outputIsFile := out.(*os.File)
		return inputIsFile && outputIsFile
	}
}

// plainRequested reads the persistent --plain flag from anywhere in the tree.
func plainRequested(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	flag := cmd.Flags().Lookup(plainFlagName)
	if flag == nil {
		return false
	}
	value, err := cmd.Flags().GetBool(plainFlagName)
	return err == nil && value
}

// rootJSONFlag reports the persistent --json flag regardless of which
// subcommand declared it.
func rootJSONFlag(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		return false
	}
	value, err := cmd.Flags().GetBool("json")
	return err == nil && value
}

// capabilitySizeProbe returns a size function bound to the command's output.
// When the stream is not a real file — every test — it reports failure so the
// ladder falls back to its fixed default rather than reading the host console.
func capabilitySizeProbe() func(io.Writer) (int, int, error) {
	return func(w io.Writer) (int, int, error) {
		if _, ok := w.(*os.File); !ok {
			return 0, 0, errNoSize
		}
		return ui.TerminalSize(w)
	}
}

type capabilityError string

func (e capabilityError) Error() string { return string(e) }

const errNoSize = capabilityError("output stream has no measurable size")
