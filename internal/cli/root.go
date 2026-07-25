package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
)

// AgentProcessChecker reports whether the selected coding agent is active.
type AgentProcessChecker func(context.Context, string) (bool, error)

// Options configure root command construction.
type Options struct {
	// Name is the binary name shown in help (rein or reinstate).
	Name string
	// Stdout/Stderr override streams for tests.
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	// Args overrides os.Args[1:] when non-nil.
	Args []string
	// AgentProcessChecker overrides process detection in deterministic tests.
	AgentProcessChecker AgentProcessChecker
}

// Execute builds and runs the root command, returning a process exit code.
func Execute(opts Options) int {
	if opts.Name == "" {
		opts.Name = "reinstate"
	}
	root := NewRoot(opts)
	if opts.Args != nil {
		root.SetArgs(opts.Args)
	}
	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	// Cobra usage errors
	if strings.Contains(err.Error(), "unknown command") ||
		strings.Contains(err.Error(), "unknown flag") ||
		strings.Contains(err.Error(), "required flag") ||
		strings.Contains(err.Error(), "accepts") ||
		strings.Contains(err.Error(), "invalid argument") ||
		strings.Contains(err.Error(), "unknown shorthand") {
		return ExitUsage
	}
	if ee, ok := err.(*ExitError); ok {
		jsonMode := false
		if f := root.PersistentFlags().Lookup("json"); f != nil {
			jsonMode, _ = root.PersistentFlags().GetBool("json")
		}
		// Prefer command-local --json when present on the leaf.
		WriteError(root.ErrOrStderr(), jsonMode, ee)
		return ee.Code
	}
	_, _ = fmt.Fprintln(root.ErrOrStderr(), err.Error())
	return ExitCodeFrom(err)
}

// NewRoot constructs the Cobra root command tree.
func NewRoot(opts Options) *cobra.Command {
	name := opts.Name
	if name == "" {
		name = "reinstate"
	}
	processChecker := opts.AgentProcessChecker
	if processChecker == nil {
		processChecker = processcheck.AgentActive
	}
	var jsonGlobal bool
	root := &cobra.Command{
		Use:           name,
		Short:         "Encrypted multi-agent session sync for AI coding tools",
		Long:          "Reinstate syncs Claude Code and Codex sessions across devices with end-to-end encryption and bring-your-own storage.",
		SilenceErrors: true,
		SilenceUsage:  true,
		// No args → usage exit 2
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return NewExitError(ExitUsage, "missing command")
		},
	}
	if opts.Stdout != nil {
		root.SetOut(opts.Stdout)
	}
	if opts.Stderr != nil {
		root.SetErr(opts.Stderr)
	}
	if opts.Stdin != nil {
		root.SetIn(opts.Stdin)
	}
	root.PersistentFlags().BoolVar(&jsonGlobal, "json", false, "prefer JSON output where supported")
	root.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Root().Printf("Unknown help topic %#q\n", args)
				_ = c.Root().Usage()
				return
			}
			_ = cmd.Help()
		},
	})
	// Help flag success path handled by Cobra (exit 0).
	root.AddCommand(
		newVersionCmd(),
		newDoctorCmd(),
		newSetupCmd(),
		newInitCmd(),
		newListCmd(),
		newStatusCmd(),
		newDiffCmd(),
		newPushCmd(),
		newPullCmd(processChecker),
		newConflictsCmd(processChecker),
		newCompletionCmd(),
	)
	// Map missing-command and usage to ExitError for consistent codes.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return NewExitError(ExitUsage, err.Error())
	})
	return root
}

// RunContext is reserved for future cancellation wiring.
func RunContext(ctx context.Context, opts Options) int {
	_ = ctx
	return Execute(opts)
}
