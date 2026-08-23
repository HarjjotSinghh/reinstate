package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	syncengine "github.com/HarjjotSinghh/reinstate/internal/sync"
)

// AgentProcessChecker reports whether the selected coding agent is holding the
// session file at sessionPath.
//
// scoped reports whether the answer was specific to sessionPath. When the host
// cannot enumerate open file handles the implementation falls back to a
// host-wide check and reports scoped=false, so callers can explain the refusal
// accurately instead of claiming precision they do not have.
type AgentProcessChecker func(ctx context.Context, agent string, target processcheck.Target) (busy bool, scoped bool, err error)

// Options configure root command construction.
type Options struct {
	// Name is the binary name shown in help (rein or reinstate).
	Name string
	// Context carries cancellation through refresh and native child execution.
	Context context.Context
	// Stdout/Stderr override streams for tests.
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	// Args overrides os.Args[1:] when non-nil.
	Args []string
	// AgentProcessChecker overrides process detection in deterministic tests.
	AgentProcessChecker AgentProcessChecker
	// EnvelopeCodec overrides envelope crypto in deterministic tests.
	// The production command leaves it nil.
	EnvelopeCodec syncengine.EnvelopeCodec
	// SessionSources overrides config-independent local discovery in tests.
	// A nil slice selects the production Claude, Codex, Gemini, OpenCode, and
	// Grok sources.
	SessionSources []sessionindex.Source
	// SessionLaunchRunner overrides native vendor process execution in tests.
	SessionLaunchRunner sessionindex.LaunchRunner
	// PreflightVerifier overrides verified-resume environment inspection in
	// deterministic tests. Production uses the bounded local verifier service.
	PreflightVerifier preflight.Verifier
	// TerminalChecker overrides TTY detection in switcher tests.
	TerminalChecker func(io.Reader, io.Writer) bool
	// HandoffDestinationCompat overrides the destination agent's compatibility
	// answer in deterministic tests. Production leaves it empty and detection
	// resolves it by running `<agent> --version` as a child process under a
	// hard deadline.
	//
	// That probe was the last non-deterministic boundary in this struct with no
	// injection point, and it is a real one: under a saturated parallel run the
	// child does not reliably start inside the two-second bound, detection
	// correctly reports UNTESTED, and a test that asserts nothing about
	// versions fails for a reason it never meant to measure. The bound itself
	// is deliberate — a hanging vendor binary must not stall handoff planning —
	// so the seam belongs here rather than in the timeout.
	HandoffDestinationCompat adapter.Compatibility
	// DeviceTokenStore overrides the OS keyring holding the Hop device token
	// in deterministic tests. Production uses the native keyring.
	DeviceTokenStore credentials.DeviceTokenStore
	// OpenBrowser overrides launching the system browser for `rein login`.
	OpenBrowser func(url string) error
	// LoginPollSleep overrides the wait between login polls in tests.
	LoginPollSleep func(context.Context, time.Duration) error
	// DeviceName overrides the hostname `rein login` reports in tests.
	DeviceName string
	// DeviceSecrets overrides the OS keyring that holds this device's hosted
	// key in deterministic tests. Production leaves it nil.
	DeviceSecrets credentials.SecretStore
	// RecoveryCodePrompt overrides hidden recovery-code entry in deterministic
	// tests (both the forced re-entry at init and the prompt at recover).
	// Production leaves it nil and reads a terminal or
	// REINSTATE_RECOVERY_CODE_FD.
	RecoveryCodePrompt func(prompt string) ([]byte, error)
	// PairingCodePrompt overrides hidden pairing-code entry on the approving
	// device in deterministic tests. Production leaves it nil and reads a
	// terminal or REINSTATE_PAIRING_CODE_FD.
	PairingCodePrompt func(prompt string) ([]byte, error)
}

type envelopeCodecContextKey struct{}

// Execute builds and runs the root command, returning a process exit code.
func Execute(opts Options) int {
	if opts.Name == "" {
		opts.Name = "reinstate"
	}
	root := NewRoot(opts)
	if opts.Args != nil {
		root.SetArgs(opts.Args)
	}
	executed, err := root.ExecuteC()
	if err == nil {
		return ExitOK
	}
	if ee, ok := err.(*ExitError); ok {
		WriteError(root.ErrOrStderr(), commandJSONMode(executed, root), ee)
		return ee.Code
	}
	// Cobra usage errors
	if strings.Contains(err.Error(), "unknown command") ||
		strings.Contains(err.Error(), "unknown flag") ||
		strings.Contains(err.Error(), "required flag") ||
		strings.Contains(err.Error(), "requires at least") ||
		strings.Contains(err.Error(), "accepts") ||
		strings.Contains(err.Error(), "invalid argument") ||
		strings.Contains(err.Error(), "unknown shorthand") {
		return ExitUsage
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
		processChecker = processcheck.SessionBusy
	}
	local := localCommandOptions{
		sources:        opts.SessionSources,
		launchRunner:   opts.SessionLaunchRunner,
		verifier:       opts.PreflightVerifier,
		terminalCheck:  opts.TerminalChecker,
		processChecker: processChecker,
	}
	if local.verifier == nil {
		local.verifier = preflight.DefaultService()
	}
	if local.terminalCheck == nil {
		local.terminalCheck = func(in io.Reader, out io.Writer) bool {
			inputFile, inputOK := in.(*os.File)
			outputFile, outputOK := out.(*os.File)
			return inputOK && outputOK && term.IsTerminal(int(inputFile.Fd())) &&
				term.IsTerminal(int(outputFile.Fd()))
		}
	}
	hopOpts := hopCommandOptions{
		tokens:      opts.DeviceTokenStore,
		openBrowser: opts.OpenBrowser,
		sleep:       opts.LoginPollSleep,
		deviceName:  opts.DeviceName,
	}
	var jsonGlobal bool
	root := &cobra.Command{
		Use:           name,
		Short:         "Find, resume, and sync coding-agent sessions",
		Long:          "Reinstate is a continuity layer for coding-agent work: search and resume local sessions, or sync supported sessions with end-to-end encryption and bring-your-own storage.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessionPicker(cmd, local)
		},
	}
	rootContext := opts.Context
	if rootContext == nil {
		rootContext = context.Background()
	}
	if opts.EnvelopeCodec != nil {
		rootContext = context.WithValue(rootContext, envelopeCodecContextKey{}, opts.EnvelopeCodec)
	}
	rootContext = context.WithValue(rootContext, hopSeamsContextKey{}, hopOpts)
	rootContext = context.WithValue(rootContext, hostedHolderContextKey{}, &hostedHolder{})
	if opts.DeviceSecrets != nil || opts.RecoveryCodePrompt != nil || opts.PairingCodePrompt != nil {
		rootContext = context.WithValue(rootContext, accountSeamsContextKey{}, accountSeams{
			secrets:        opts.DeviceSecrets,
			recoveryPrompt: opts.RecoveryCodePrompt,
			pairingPrompt:  opts.PairingCodePrompt,
		})
	}
	root.SetContext(rootContext)
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
	root.PersistentFlags().Bool(
		plainFlagName,
		false,
		"force plain non-interactive output on a terminal that could show the interactive UI",
	)
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
		newLoginCmd(hopOpts),
		newWhoamiCmd(hopOpts),
		newHopCmd(),
		newDevicesCmd(),
		newInitCmd(),
		newAccountCmd(),
		newListCmd(),
		newSessionsCmd(local),
		newSearchCmd(local),
		newInspectCmd(local),
		newLastCmd(local),
		newHandoffCmd(handoffCommandOptions{
			local: local, processChecker: processChecker,
			destinationCompat: opts.HandoffDestinationCompat,
		}),
		newResumeCmd(local),
		newForkCmd(local),
		newStatusCmd(),
		newDiffCmd(),
		newPushCmd(),
		newPullCmd(processChecker),
		newSyncCmd(),
		newConflictsCmd(processChecker),
		newCompletionCmd(),
	)
	// Map missing-command and usage to ExitError for consistent codes.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return NewExitError(ExitUsage, err.Error())
	})
	return root
}

func commandJSONMode(executed, root *cobra.Command) bool {
	if executed != nil {
		if flag := executed.Flags().Lookup("json"); flag != nil && flag.Changed {
			value, err := executed.Flags().GetBool("json")
			if err == nil {
				return value
			}
		}
	}
	if root != nil {
		if flag := root.PersistentFlags().Lookup("json"); flag != nil {
			value, err := root.PersistentFlags().GetBool("json")
			return err == nil && value
		}
	}
	return false
}

func RunContext(ctx context.Context, opts Options) int {
	opts.Context = ctx
	return Execute(opts)
}
