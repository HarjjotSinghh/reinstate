// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package cli

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/pairing"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/wizard"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// runInitWizard collects storage coordinates interactively.
//
// It returns the collected values and true, or false when the user cancelled.
// Credentials are deliberately not collected here; see the wizard package
// comment for why. The caller reads those through the existing hidden-input
// path once the terminal has been restored.
func runInitWizard(
	cmd *cobra.Command,
	capability ui.Capability,
	defaults wizard.Result,
) (wizard.Result, bool, error) {
	model := wizard.New(wizard.Options{
		Theme:      ui.NewTheme(capability),
		Capability: capability,
		Defaults:   defaults,
	})
	if _, err := tui.Run(cmd.Context(), model, tui.RunOptions{
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		Capability: capability,
		AltScreen:  true,
	}); err != nil {
		return wizard.Result{}, false, NewExitError(ExitRuntime, "run setup: "+err.Error())
	}
	if !model.Confirmed() {
		return wizard.Result{}, false, nil
	}
	return model.Result(), true, nil
}

// printPairingCode writes the pairing block for this device's profile.
func printPairingCode(cmd *cobra.Command) error {
	home, err := config.Home()
	if err != nil {
		return NewExitError(ExitConfig, err.Error())
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return NewExitError(ExitConfig, "this device is not initialized yet; run rein init first")
	}
	if cfg.Storage.Type == schema.StorageHop {
		return NewExitError(ExitConfig, "this profile syncs to a Reinstate Hop locker; another device joins it with rein login, rein init --hop, and rein account join, not init --link")
	}
	if strings.TrimSpace(cfg.Storage.Bucket) == "" {
		return NewExitError(ExitConfig, "this device has no storage configured; run rein init first")
	}
	code, err := pairing.Encode(pairing.Payload{
		Endpoint:  cfg.Storage.Endpoint,
		Bucket:    cfg.Storage.Bucket,
		Region:    cfg.Storage.Region,
		Prefix:    cfg.Storage.Prefix,
		ProfileID: cfg.ProfileID,
	})
	if err != nil {
		return NewExitError(ExitRuntime, err.Error())
	}
	PrintHuman(cmd.OutOrStdout(), "Pairing code for this profile:")
	PrintHuman(cmd.OutOrStdout(), "")
	for _, line := range pairing.Format(code, 56) {
		PrintHuman(cmd.OutOrStdout(), "  %s", line)
	}
	PrintHuman(cmd.OutOrStdout(), "")
	PrintHuman(cmd.OutOrStdout(), "On the other device run:  rein init --paste")
	PrintHuman(cmd.OutOrStdout(), "")
	// Saying what the code is not is as important as saying what it is. Someone
	// who believes it is a secret will handle it too carefully to be useful;
	// someone who believes it is a credential may expect it to grant access.
	PrintHuman(
		cmd.OutOrStdout(),
		"This code carries no keys and no passphrase. The other device still asks for both.",
	)
	return nil
}

// pairingDefaults converts a decoded pairing code into wizard defaults.
func pairingDefaults(payload pairing.Payload) wizard.Result {
	return wizard.Result{
		Provider:     providerForEndpoint(payload.Endpoint),
		Endpoint:     payload.Endpoint,
		Bucket:       payload.Bucket,
		Region:       payload.Region,
		Prefix:       payload.Prefix,
		ProfileID:    payload.ProfileID,
		JoinExisting: true,
	}
}

// providerForEndpoint guesses the preset from an endpoint so the review screen
// names the provider the reader recognises. A wrong guess costs nothing: the
// preset only seeds placeholders, and every value is already filled in.
func providerForEndpoint(endpoint string) string {
	lowered := strings.ToLower(endpoint)
	switch {
	case strings.Contains(lowered, "r2.cloudflarestorage.com"):
		return "r2"
	case strings.Contains(lowered, "amazonaws.com"):
		return "s3"
	case strings.Contains(lowered, "backblazeb2.com"):
		return "b2"
	default:
		return "other"
	}
}

// wizardApplies decides whether interactive setup should run.
//
// It runs only when the terminal can host it, the caller did not ask for
// non-interactive mode, and the coordinates were not already supplied by flags
// or environment. Anyone who passed --endpoint and --bucket has already said
// what they want; putting a form in front of them would be noise. A pasted
// pairing code always opens the wizard, because reviewing what was decoded
// before committing to it is the point.
func wizardApplies(capability ui.Capability, nonInteractive, paste bool, endpoint, bucket string) bool {
	if nonInteractive || !capability.Mode.Interactive() {
		return false
	}
	if paste {
		return true
	}
	return strings.TrimSpace(endpoint) == "" || strings.TrimSpace(bucket) == ""
}

// readPairingCode reads a pairing code from stdin.
func readPairingCode(cmd *cobra.Command) (pairing.Payload, error) {
	PrintHuman(cmd.ErrOrStderr(), "Paste the pairing code from your other device, then press enter:")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return pairing.Payload{}, NewExitError(ExitUsage, "read pairing code: "+err.Error())
	}
	payload, err := pairing.Decode(line)
	if err != nil {
		return pairing.Payload{}, NewExitError(ExitUsage, err.Error())
	}
	return payload, nil
}

// defaultInitTerminalCheck reports whether init has a usable terminal. init has
// no injected terminal check of its own, so it uses the same probe the rest of
// the CLI does.
func defaultInitTerminalCheck(in io.Reader, out io.Writer) bool {
	inputFile, inputOK := in.(*os.File)
	outputFile, outputOK := out.(*os.File)
	if !inputOK || !outputOK {
		return false
	}
	return term.IsTerminal(int(inputFile.Fd())) && term.IsTerminal(int(outputFile.Fd()))
}
