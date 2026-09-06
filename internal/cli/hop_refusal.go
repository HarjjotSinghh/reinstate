package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/safetext"
)

// Reporting a refused sign-in.
//
// `rein login` starts a session, opens a browser, and polls. When the
// browser half refuses — the account is at its device quota, the link was
// opened too late, GitHub cancelled — the control plane records the refusal
// and the poll answers `refused` with a code and the exact sentence the
// browser page showed (docs/hop.md, "When a sign-in is refused"). This file
// turns that answer into what a person sees in the terminal.
//
// Three rules, in order of who is right:
//
//  1. The control plane's sentence is printed verbatim, always, whether or
//     not this build knows the code. It is the sentence the person read in
//     the browser and it carries this account's own figures.
//  2. This client adds a command line only where that sentence names an
//     action but not the command that performs it — "revoke a device you no
//     longer use", "sign in by email instead". Where the sentence already
//     names the command, repeating it would be noise.
//  3. A command is named only if this build actually has it. `rein devices
//     revoke` ships with device revocation; until that is merged the same
//     refusal points at `rein devices` alone rather than at a subcommand
//     that would answer "unknown command".
//
// An unknown code — a control plane newer than this CLI — keeps rule 1 and
// loses rules 2 and 3: the sentence is printed, the poll still stops, and
// the exit code is the sign-in family's. TestALoginRefusalIsNeverRendered-
// Empty holds that path to a non-empty, actionable message.

// loginRefusal is what this client knows to do about one refusal code
// beyond printing the control plane's sentence.
type loginRefusal struct {
	// exit is the process exit code. Most refusals are a sign-in that did
	// not authenticate, which is exit 4 (`auth_storage`); the two that are
	// "nothing is wrong with you, try again" — an expired link and a
	// control-plane fault — keep exit 1, which is what an expired sign-in
	// and an unreachable control plane already returned.
	exit int
	// guidance returns the extra line to print and the commands it names,
	// or "" and nil when the control plane's sentence is complete on its
	// own. root is the command tree, so guidance never names a subcommand
	// this build does not carry.
	guidance func(root *cobra.Command) (string, []string)
}

// loginRefusals is the guidance table, one row per code in
// hop.SignInRefusalCodes(). TestEverySignInRefusalCodeHasGuidance fails if
// a code is added to the hop package without a row here, so a new refusal
// cannot reach a person with an undecided exit code.
var loginRefusals = map[string]loginRefusal{
	hop.CodeQuotaDevices: {exit: ExitAuthStorage, guidance: freeADeviceSlot},
	// "This sign-in link has expired. Run `rein login` again." — complete.
	hop.CodeLoginExpired: {exit: ExitRuntime},
	// "GitHub did not return an authorization code ... Run `rein login`
	// again to start over." — complete, and a cancel is a cancel.
	hop.CodeGitHubNoCode: {exit: ExitAuthStorage},
	// GitHub refused the exchange, so running the same sign-in again asks
	// the same GitHub. The other door is worth naming.
	hop.CodeGitHubRejected: {exit: ExitAuthStorage, guidance: signInByEmail},
	// "Sign in by email instead, or contact support." — names the action,
	// not the command.
	hop.CodeAccountLinked: {exit: ExitAuthStorage, guidance: signInByEmail},
	// "Something went wrong on the control plane ... Run `rein login`
	// again; if it keeps happening, write to support@reinstate.dev." —
	// complete, and nothing this client can do makes it likelier to work.
	hop.CodeInternalError: {exit: ExitRuntime},
}

// unknownLoginRefusal is the row for a code this build has never heard of:
// the sign-in family's exit code, and no invented guidance.
var unknownLoginRefusal = loginRefusal{exit: ExitAuthStorage}

// freeADeviceSlot is the quota remedy. The control plane says to revoke a
// device or move to a larger plan; this names the commands for the first
// half and says nothing about the second, because this CLI has no upgrade
// command and there is no self-serve upgrade URL to send anyone to.
//
// Two things it has to be honest about. The revoking has to happen on a
// machine that is still signed in to the account — this one has just been
// refused and holds no token for it — and `rein devices revoke` exists only
// once device revocation is merged.
func freeADeviceSlot(root *cobra.Command) (string, []string) {
	if hasCommand(root, "devices", "revoke") {
		return "To free a slot: on a device that is still signed in to this account, run rein devices to see what is enrolled and rein devices revoke <device-id> to remove one. Then run rein login here again.",
			[]string{"rein devices", "rein devices revoke <device-id>", "rein login"}
	}
	return "To see what is enrolled: run rein devices on a device that is still signed in to this account. Then run rein login here again once the account is back under its device limit.",
		[]string{"rein devices", "rein login"}
}

// signInByEmail names the one-time-link sign-in, which is the way past a
// GitHub that will not authorize and past an address already linked to
// another GitHub identity.
func signInByEmail(*cobra.Command) (string, []string) {
	return "To sign in without GitHub: run rein login --email <address> and open the one-time link it sends.",
		[]string{"rein login --email <address>"}
}

// maxRefusalSentenceRunes bounds the control plane's sentence. The longest
// one it sends today is under 200 runes; the bound exists so a refusal
// cannot fill the terminal, not to truncate anything real.
const maxRefusalSentenceRunes = 400

// loginRefusalError turns a refused sign-in into the error `rein login`
// exits with: the control plane's sentence, what it means for this machine,
// the commands this build can offer, and — under --json — the same as a
// refusal document under `details.refusal`.
func loginRefusalError(root *cobra.Command, refused *hop.RefusedError) *ExitError {
	code := strings.TrimSpace(refused.Code)
	entry, known := loginRefusals[code]
	if !known {
		entry = unknownLoginRefusal
	}
	// The sentence is prose the control plane chose, printed to a terminal.
	// Left verbatim it carries newlines and ANSI sequences, so a hostile or
	// intercepted control plane can forge lines that read like this command's
	// own output — clear the screen, set the window title, or announce a
	// sign-in that never happened. safetext flattens it to one bounded line
	// and keeps the backticked command text the sentences rely on.
	reason := safetext.Text(refused.Sentence(), maxRefusalSentenceRunes)

	head := "Reinstate Hop refused this sign-in"
	if code != "" {
		head += " (" + code + ")"
	}
	lines := []string{
		fmt.Sprintf("%s: %s", head, reason),
		// True of every refusal, not only the ones this build knows: the
		// refusal is recorded instead of the approval, so no device was
		// enrolled, and the session is never approved afterwards. The
		// token is stored only after an approval, so there is nothing to
		// undo either.
		"This device was not enrolled, the sign-in link is spent, and no device token was stored.",
	}
	var guidance string
	var commands []string
	if entry.guidance != nil {
		guidance, commands = entry.guidance(root)
	}
	if guidance != "" {
		lines = append(lines, guidance)
	} else {
		// Every refusal leaves the terminal with a command, including one
		// whose code this build does not know and whose sentence names no
		// command of its own. This one promises nothing about the outcome:
		// starting over is what there is, and whether it succeeds depends
		// on the sentence above.
		lines = append(lines, "Nothing more will arrive on this session; starting over means running rein login again.")
		commands = []string{"rein login"}
	}

	err := NewExitError(entry.exit, strings.Join(lines, "\n"))
	err.Details["refusal"] = map[string]any{
		"code":     code,
		"reason":   reason,
		"known":    known,
		"terminal": true,
		"commands": commands,
	}
	return err
}

// hasCommand reports whether the command tree under root carries the named
// path, so guidance can name `rein devices revoke` where it exists and stay
// quiet where it does not.
func hasCommand(root *cobra.Command, path ...string) bool {
	if root == nil || len(path) == 0 {
		return false
	}
	current := root
	for _, name := range path {
		var next *cobra.Command
		for _, child := range current.Commands() {
			if child.Name() == name {
				next = child
				break
			}
		}
		if next == nil {
			return false
		}
		current = next
	}
	return true
}
