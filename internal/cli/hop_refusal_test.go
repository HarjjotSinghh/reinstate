package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// The device-quota sentence the control plane records and renders, copied
// from `internal/server`. Nothing here parses it: it is printed verbatim,
// and the point of holding the real one is that the assertions run against
// backticks and figures rather than against a tidy stand-in.
const quotaDevicesSentence = "This account already has the 5 devices the hop plan allows. " +
	"Revoke a device you no longer use, or move to a larger plan, and then run `rein login` again."

// browsedPage runs the sign-in link the way the harness's browser does and
// keeps the page body, so a test can hold the browser's words and the
// terminal's words side by side.
func (h *hopHarness) browsedPage(page *string) func(string) error {
	return func(url string) error {
		h.browsed = append(h.browsed, url)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		*page = string(body)
		return nil
	}
}

func (f *fakeControlPlane) pollCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.polls
}

// TestALoginRefusalReachesTheTerminal is the journey this work exists for.
//
// The browser half of a sign-in can end without enrolling a device, and it
// renders a helpful page when it does. The CLI is polling. Before the poll
// route could say "refused", the session stayed pending: the person read
// what to do in one window while `rein login` polled on to its deadline in
// the other and then reported an expiry — wrong about what happened and
// silent about what to do.
//
// Every row here is a refusal the control plane can send, including one
// with a code this build has never heard of, and one that arrives with no
// code and no sentence at all. All of them must stop the poll at the first
// answer, print what the browser printed, and exit non-zero.
func TestALoginRefusalReachesTheTerminal(t *testing.T) {
	tests := []struct {
		name       string
		refusal    fakeSignInRefusal
		wantExit   int
		wantStderr []string
	}{
		{
			name:     "device quota",
			refusal:  fakeSignInRefusal{code: hop.CodeQuotaDevices, reason: quotaDevicesSentence, pageStatus: 403},
			wantExit: ExitAuthStorage,
			// The sentence, then the commands that act on it. `rein devices
			// revoke` is named only where this build has it, which is what
			// TestTheQuotaRemedyNamesRevokeOnlyWhenThisBuildHasIt covers.
			wantStderr: []string{"quota_devices", quotaDevicesSentence, "rein devices", "run rein login here again"},
		},
		{
			name:       "link opened too late",
			refusal:    fakeSignInRefusal{code: hop.CodeLoginExpired, reason: "This sign-in link has expired. Run `rein login` again.", pageStatus: 410},
			wantExit:   ExitRuntime,
			wantStderr: []string{"login_expired", "This sign-in link has expired."},
		},
		{
			name:       "github authorization cancelled",
			refusal:    fakeSignInRefusal{code: hop.CodeGitHubNoCode, reason: "GitHub did not return an authorization code, which is what a cancelled or denied sign-in looks like. Run `rein login` again to start over.", pageStatus: 400},
			wantExit:   ExitAuthStorage,
			wantStderr: []string{"github_no_code", "cancelled or denied sign-in"},
		},
		{
			name:       "github rejected the exchange",
			refusal:    fakeSignInRefusal{code: hop.CodeGitHubRejected, reason: "GitHub rejected the authorization. Run `rein login` again.", pageStatus: 502},
			wantExit:   ExitAuthStorage,
			wantStderr: []string{"github_rejected", "rein login --email <address>"},
		},
		{
			name:       "address linked to another github identity",
			refusal:    fakeSignInRefusal{code: hop.CodeAccountLinked, reason: "This email address already belongs to a Reinstate Hop account linked to a different GitHub identity. Sign in by email instead, or contact support.", pageStatus: 409},
			wantExit:   ExitAuthStorage,
			wantStderr: []string{"account_linked", "Sign in by email instead", "rein login --email <address>"},
		},
		{
			name:       "control plane fault",
			refusal:    fakeSignInRefusal{code: hop.CodeInternalError, reason: "Something went wrong on the control plane and this sign-in did not finish. Run `rein login` again; if it keeps happening, write to support@reinstate.dev.", pageStatus: 500},
			wantExit:   ExitRuntime,
			wantStderr: []string{"internal_error", "did not finish"},
		},
		{
			// A control plane newer than this CLI. The code means nothing
			// here; the sentence still means everything to the person, and
			// the poll still has to stop.
			name:       "a code this build has never heard of",
			refusal:    fakeSignInRefusal{code: "account_suspended", reason: "This account is suspended. Write to support@reinstate.dev.", pageStatus: 403},
			wantExit:   ExitAuthStorage,
			wantStderr: []string{"account_suspended", "This account is suspended."},
		},
		{
			// Neither should ever be empty on the wire. If one is, the
			// person still gets a sentence rather than a blank line.
			name:       "no code and no sentence",
			refusal:    fakeSignInRefusal{pageStatus: 403},
			wantExit:   ExitAuthStorage,
			wantStderr: []string{hop.GenericRefusalReason},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHopHarness(t)
			refusal := test.refusal
			h.plane.refuseSignIn = &refusal
			var page string
			h.browser = h.browsedPage(&page)

			out, errb, code := h.run("login")
			if code != test.wantExit {
				t.Fatalf("exit=%d want %d\nstdout=%q\nstderr=%q", code, test.wantExit, out, errb)
			}
			for _, want := range test.wantStderr {
				if !strings.Contains(errb, want) {
					t.Errorf("stderr does not carry %q:\n%s", want, errb)
				}
			}
			// The browser and the terminal say the same thing. That is the
			// property, not just "the terminal says something".
			if refusal.reason != "" && (!strings.Contains(page, refusal.reason) || !strings.Contains(errb, refusal.reason)) {
				t.Errorf("page and terminal disagree.\npage:   %q\nstderr: %q", page, errb)
			}
			// Stopped at the first answer rather than running out the clock.
			if got := h.plane.pollCount(); got != 1 {
				t.Errorf("polled %d times; a refusal is terminal and must stop the loop at once", got)
			}
			// And was not reported as the thing it is not.
			if strings.Contains(errb, "expired before it was used") {
				t.Errorf("a refusal was reported as an expiry:\n%s", errb)
			}
			if _, err := h.tokens.GetDeviceToken(); err == nil {
				t.Error("a refused sign-in stored a device token")
			}
			if !strings.Contains(errb, "This device was not enrolled, the sign-in link is spent, and no device token was stored.") {
				t.Errorf("stderr does not say the device was not enrolled:\n%s", errb)
			}
		})
	}
}

// TestALoginRefusalCarriesStructuredJSON holds `--json` to the same
// standard as the prose: a script must be able to read the code, the
// sentence, whether this build recognised the code, and the commands the
// CLI would have named — without parsing an English message.
func TestALoginRefusalCarriesStructuredJSON(t *testing.T) {
	h := newHopHarness(t)
	h.plane.refuseSignIn = &fakeSignInRefusal{code: hop.CodeQuotaDevices, reason: quotaDevicesSentence, pageStatus: 403}
	var page string
	h.browser = h.browsedPage(&page)

	out, errb, code := h.run("login", "--json")
	if code != ExitAuthStorage {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	var envelope ErrorJSON
	if err := json.Unmarshal([]byte(errb), &envelope); err != nil {
		t.Fatalf("stderr is not the JSON error envelope (%v): %s", err, errb)
	}
	if envelope.Code != "auth_storage" || envelope.SafeToRetry {
		t.Errorf("envelope %+v: a refusal is terminal and belongs to the sign-in family", envelope)
	}
	refusal, ok := envelope.Details["refusal"].(map[string]any)
	if !ok {
		t.Fatalf("details carry no refusal document: %+v", envelope.Details)
	}
	if refusal["code"] != hop.CodeQuotaDevices || refusal["reason"] != quotaDevicesSentence {
		t.Errorf("refusal document %+v", refusal)
	}
	if refusal["known"] != true || refusal["terminal"] != true {
		t.Errorf("refusal document %+v", refusal)
	}
	commands, ok := refusal["commands"].([]any)
	if !ok || len(commands) == 0 {
		t.Fatalf("refusal document names no commands: %+v", refusal)
	}
	if commands[0] != "rein devices" {
		t.Errorf("first command %v, want the one that shows what is enrolled", commands[0])
	}
	if strings.Contains(out, "{") {
		t.Errorf("a refused sign-in wrote a document to stdout: %q", out)
	}
}

// TestEverySignInRefusalCodeHasGuidance is the gate that makes the next
// refusal cheap instead of a repeat of this bug. A code declared in the hop
// package with no row here would reach a person with an exit code nobody
// chose; a row here for a code nobody declared is guidance that can never
// fire.
func TestEverySignInRefusalCodeHasGuidance(t *testing.T) {
	stable := map[int]bool{ExitRuntime: true, ExitUsage: true, ExitConfig: true, ExitAuthStorage: true,
		ExitCompatibility: true, ExitConflict: true, ExitSafety: true}
	declared := map[string]bool{}
	for _, code := range hop.SignInRefusalCodes() {
		declared[code] = true
		entry, ok := loginRefusals[code]
		if !ok {
			t.Errorf("hop.%s has no row in loginRefusals: `rein login` would report it with an exit code nobody chose", code)
			continue
		}
		if !stable[entry.exit] {
			t.Errorf("%s exits %d, which is not one of the stable exit codes", code, entry.exit)
		}
	}
	for code := range loginRefusals {
		if !declared[code] {
			t.Errorf("loginRefusals has a row for %q, which hop.SignInRefusalCodes() does not declare", code)
		}
	}
	if unknownLoginRefusal.exit == ExitOK {
		t.Error("a refusal this build does not know must not exit 0")
	}
}

// TestALoginRefusalIsNeverRenderedEmpty is the unknown-code guarantee
// stated as a property rather than as a list. Every code this build knows,
// a code from a control plane newer than this one, and the degenerate
// answers that should never arrive: each has to produce a message a person
// can act on, a non-zero exit, and a refusal document.
func TestALoginRefusalIsNeverRenderedEmpty(t *testing.T) {
	codes := append(hop.SignInRefusalCodes(), "", "a_code_from_a_newer_control_plane")
	reasons := []string{
		"This account is suspended. Write to support@reinstate.dev.",
		"",
		"   ",
	}
	root := NewRoot(Options{Name: "rein"})
	for _, code := range codes {
		for _, reason := range reasons {
			t.Run(code+"/"+reason, func(t *testing.T) {
				err := loginRefusalError(root, &hop.RefusedError{Code: code, Reason: reason})
				if err.Code == ExitOK {
					t.Fatalf("code %q exits 0", code)
				}
				if strings.TrimSpace(err.Message) == "" {
					t.Fatalf("code %q renders an empty message", code)
				}
				want := strings.TrimSpace(reason)
				if want == "" {
					want = hop.GenericRefusalReason
				}
				if !strings.Contains(err.Message, want) {
					t.Fatalf("code %q does not print the sentence %q:\n%s", code, want, err.Message)
				}
				if !strings.Contains(err.Message, "rein login") {
					t.Fatalf("code %q leaves a person with no way on:\n%s", code, err.Message)
				}
				refusal, ok := err.Details["refusal"].(map[string]any)
				if !ok {
					t.Fatalf("code %q carries no refusal document", code)
				}
				if refusal["reason"] == "" {
					t.Fatalf("code %q carries an empty reason in JSON", code)
				}
				// "known" means exactly one thing: this build has a row for
				// the code and so had something to add. It is read from the
				// table rather than from a list repeated here, so that a
				// code added to one and not the other is caught by
				// TestEverySignInRefusalCodeHasGuidance rather than by a
				// puzzling failure in this one.
				_, wantKnown := loginRefusals[code]
				if refusal["known"] != wantKnown {
					t.Fatalf("code %q reported known=%v, want %v", code, refusal["known"], wantKnown)
				}
			})
		}
	}
}

// TestALoginRefusalNamesOnlyCommandsThisBuildHas keeps the remedy runnable.
// Telling somebody over their device quota to run a subcommand that answers
// "unknown command" is worse than telling them nothing: it costs them a
// round trip to find out the CLI was wrong.
func TestALoginRefusalNamesOnlyCommandsThisBuildHas(t *testing.T) {
	root := NewRoot(Options{Name: "rein"})
	// Every code this build knows, plus the two that exercise the fallback:
	// a code from a newer control plane, and no code at all. The rendered
	// document is what is checked, not the table, so the fallback's own
	// command is held to the same rule as the table's.
	for _, code := range append(hop.SignInRefusalCodes(), "", "a_code_from_a_newer_control_plane") {
		err := loginRefusalError(root, &hop.RefusedError{Code: code, Reason: "A sentence the control plane wrote."})
		refusal, ok := err.Details["refusal"].(map[string]any)
		if !ok {
			t.Fatalf("%q carries no refusal document", code)
		}
		commands, ok := refusal["commands"].([]string)
		if !ok || len(commands) == 0 {
			t.Errorf("%q names no command, so the terminal ends with nothing to run: %v", code, refusal["commands"])
			continue
		}
		for _, command := range commands {
			path := commandPath(command)
			if len(path) == 0 {
				t.Errorf("%q names %q, which is not a rein command", code, command)
				continue
			}
			if !hasCommand(root, path...) {
				t.Errorf("%q names %q, which this build does not have", code, command)
			}
			// A command listed for a script but absent from the prose is a
			// command the person reading the terminal never sees.
			if !strings.Contains(err.Message, command) {
				t.Errorf("%q lists %q among its commands but the message a person reads does not name it:\n%s", code, command, err.Message)
			}
		}
	}
}

// commandPath reduces "rein devices revoke <device-id>" to
// {"devices", "revoke"}: the leading binary name dropped, and everything
// from the first flag or placeholder onwards.
func commandPath(command string) []string {
	fields := strings.Fields(command)
	if len(fields) == 0 || fields[0] != "rein" {
		return nil
	}
	var path []string
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "-") || strings.HasPrefix(field, "<") {
			break
		}
		path = append(path, field)
	}
	return path
}

// TestTheQuotaRemedyNamesRevokeOnlyWhenThisBuildHasIt covers both sides of
// the one guidance line that depends on what is merged. `rein devices
// revoke` arrives with device revocation; on a tree without it the same
// refusal has to point at what does exist rather than invent a subcommand.
func TestTheQuotaRemedyNamesRevokeOnlyWhenThisBuildHasIt(t *testing.T) {
	withoutRevoke := NewRoot(Options{Name: "rein"})
	line, commands := freeADeviceSlot(withoutRevoke)
	if hasCommand(withoutRevoke, "devices", "revoke") {
		// Device revocation is merged: this is the other half of the test.
		if !strings.Contains(line, "rein devices revoke <device-id>") {
			t.Fatalf("this build has rein devices revoke and the quota remedy does not name it:\n%s", line)
		}
	} else {
		if strings.Contains(line, "revoke") {
			t.Fatalf("this build has no rein devices revoke and the quota remedy names it:\n%s", line)
		}
		for _, command := range commands {
			if strings.Contains(command, "revoke") {
				t.Fatalf("commands name a subcommand this build does not have: %v", commands)
			}
		}
	}
	if !strings.Contains(line, "still signed in to this account") {
		t.Errorf("the quota remedy does not say where to run it; the refused machine holds no token for the account:\n%s", line)
	}
	if strings.Contains(strings.ToLower(line), "http") {
		t.Errorf("the quota remedy invents an upgrade link:\n%s", line)
	}

	withRevoke := NewRoot(Options{Name: "rein"})
	for _, child := range withRevoke.Commands() {
		if child.Name() == "devices" {
			child.AddCommand(&cobra.Command{Use: "revoke DEVICE-ID"})
		}
	}
	line, commands = freeADeviceSlot(withRevoke)
	if !strings.Contains(line, "rein devices revoke <device-id>") {
		t.Fatalf("with rein devices revoke present the remedy still does not name it:\n%s", line)
	}
	if len(commands) != 3 || commands[1] != "rein devices revoke <device-id>" {
		t.Fatalf("commands %v", commands)
	}
}
