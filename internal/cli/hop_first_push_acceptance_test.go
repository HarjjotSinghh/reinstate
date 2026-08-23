//go:build hopacceptance

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
)

// TestHopFirstPushJourneyStaging runs the first-push journey of
// TestHopFirstPushJourney against a real control plane and a real locker.
// It is built only with -tags hopacceptance and skips unless the
// environment names a staging control plane and a way to sign in:
//
//	HOP_STAGING_URL      control plane base URL (https://hop-staging.example)
//
// and one of
//
//	HOP_LOGIN_EMAIL      the journey runs `rein login --email` for each of
//	                     its two devices and waits (HOP_LOGIN_TIMEOUT,
//	                     default 5m) for each link to be approved, so the
//	                     sign-in is the real one; approve the two links by
//	                     hand (or from the control plane's log sender)
//	HOP_DEVICE_TOKEN and HOP_DEVICE_TOKEN_2
//	                     tokens already issued by that control plane for two
//	                     distinct devices of one account; the journey fills
//	                     in each device id from /v1/whoami. Two are needed
//	                     because the wiped device signs in again as a new
//	                     device, exactly as the in-process journey does; a
//	                     keyring that already lists a device id for which
//	                     this machine holds no key is refused by
//	                     `rein account recover`.
//
// The account must be a disposable staging account: the journey pushes
// three synthetic sessions into its locker, and the first_push check is
// only exact when the locker has never seen a push before. It never runs
// in CI and never against production. A lab run against hopd and the
// fake locker is recorded in docs/testing/results/2026-08-24-first-push-acceptance-lab.md.
func TestHopFirstPushJourneyStaging(t *testing.T) {
	stagingURL := strings.TrimSpace(os.Getenv("HOP_STAGING_URL"))
	loginEmail := strings.TrimSpace(os.Getenv("HOP_LOGIN_EMAIL"))
	tokens := []string{strings.TrimSpace(os.Getenv("HOP_DEVICE_TOKEN")), strings.TrimSpace(os.Getenv("HOP_DEVICE_TOKEN_2"))}
	switch {
	case stagingURL == "":
		t.Skip("HOP_STAGING_URL is not set; the staging first-push journey is skipped")
	case loginEmail == "" && (tokens[0] == "" || tokens[1] == ""):
		t.Skip("neither HOP_LOGIN_EMAIL nor both HOP_DEVICE_TOKEN and HOP_DEVICE_TOKEN_2 are set; the staging first-push journey is skipped")
	}
	if strings.Contains(stagingURL, "hop.reinstate.dev") && !strings.Contains(stagingURL, "staging") {
		t.Fatalf("refusing to run the first-push journey against %s; use a staging control plane", stagingURL)
	}
	loginTimeout := 5 * time.Minute
	if v := strings.TrimSpace(os.Getenv("HOP_LOGIN_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			t.Fatalf("HOP_LOGIN_TIMEOUT %q: %v", v, err)
		}
		loginTimeout = d
	}
	t.Setenv(hopURLEnv, stagingURL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(env, "")
	}
	home := plantJourneyHome(t, t.TempDir())

	// Pre-issued tokens are checked before anything is pushed: each must
	// resolve to a device (`rein init --hop` refuses a token without a device
	// id) and the two must be different devices of one account.
	var identities []hop.Identity
	if loginEmail == "" {
		for i, token := range tokens {
			id, err := hop.New(stagingURL).Whoami(context.Background(), token)
			if err != nil {
				t.Fatalf("whoami for device token %d: %v", i+1, err)
			}
			if id.Device.ID == "" {
				t.Fatalf("device token %d resolves to no device id", i+1)
			}
			identities = append(identities, id)
		}
		switch {
		case identities[0].Device.ID == identities[1].Device.ID:
			t.Fatalf("HOP_DEVICE_TOKEN and HOP_DEVICE_TOKEN_2 both belong to device %s; the wiped device needs its own device, see the test comment", identities[0].Device.ID)
		case identities[0].Account.ID != identities[1].Account.ID:
			t.Fatalf("HOP_DEVICE_TOKEN (account %s) and HOP_DEVICE_TOKEN_2 (account %s) must belong to one account", identities[0].Account.ID, identities[1].Account.ID)
		}
	}

	// signIn enrols d as a new device of the staging account and returns
	// the time spent waiting for a person to approve the link (zero when a
	// pre-issued token stands in for the browser round-trip).
	nextToken := 0
	seenDevices := map[string]string{}
	signIn := func(d *hopDevice) time.Duration {
		t.Helper()
		if loginEmail != "" {
			d.loginSleep = func(ctx context.Context, wait time.Duration) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
					return nil
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
			defer cancel()
			t.Logf("device %q: approve the sign-in link sent to %s within %s", d.name, loginEmail, loginTimeout)
			waited := time.Now()
			d.ctx = ctx
			d.mustRun("login --email", "login", "--email", loginEmail)
			d.ctx = nil
			approval := time.Since(waited)
			tok, err := d.tokens.GetDeviceToken()
			if err != nil {
				t.Fatal(err)
			}
			if prev, dup := seenDevices[tok.DeviceID]; dup {
				t.Fatalf("the control plane enrolled %q under the device id of %q; each sign-in must mint a new device", d.name, prev)
			}
			seenDevices[tok.DeviceID] = d.name
			t.Logf("device %q signed in as device %s after %s", d.name, tok.DeviceID, approval.Round(time.Millisecond))
			return approval
		}
		token, id := tokens[nextToken], identities[nextToken]
		nextToken++
		if err := d.tokens.SetDeviceToken(credentials.DeviceToken{Token: token, ControlPlaneURL: stagingURL, AccountID: id.Account.ID, DeviceID: id.Device.ID}); err != nil {
			t.Fatal(err)
		}
		d.mustRun("whoami", "whoami")
		return 0
	}
	firstPushAt := func(d *hopDevice) string {
		t.Helper()
		var status struct {
			Locker *struct {
				FirstPushAt string `json:"first_push_at"`
			} `json:"locker"`
		}
		out := d.mustRun("hop status --json", "hop", "status", "--json")
		if err := json.Unmarshal([]byte(out), &status); err != nil {
			t.Fatalf("hop status %q: %v", out, err)
		}
		if status.Locker == nil {
			return ""
		}
		return status.Locker.FirstPushAt
	}

	// --- day one ---
	laptop := newHopDevice(t, nil, "acceptance-laptop")
	laptop.verifier = preflight.DefaultService()
	start := time.Now()
	approvalWait := signIn(laptop)
	before := firstPushAt(laptop)
	if before != "" {
		t.Logf("the locker already records a first push at %s; this run can only check it does not change", before)
	}
	laptop.mustRun("init --hop", "init", "--hop", "--project", "local/first-push="+home.project)
	laptop.mustRun("account init", "account", "init")
	recoveryCode := laptop.shownCode
	pushOut := laptop.mustRun("push --all", "push", "--all", "--json")
	// The approval wait is a person clicking a link, not the product; the
	// budget covers everything from the signed-in device to its first push.
	elapsed := time.Since(start) - approvalWait
	var pushed struct {
		Snapshots []string `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(pushOut), &pushed); err != nil || len(pushed.Snapshots) != 3 {
		t.Fatalf("push output %q: %v", pushOut, err)
	}
	t.Logf("sign-in to first successful push against %s: %s (budget %s; %s more waiting for the sign-in link to be approved)", stagingURL, elapsed.Round(time.Millisecond), firstPushBudget, approvalWait.Round(time.Millisecond))
	if elapsed > firstPushBudget {
		t.Fatalf("sign-in to first push took %s, over the %s budget", elapsed, firstPushBudget)
	}
	after := firstPushAt(laptop)
	switch {
	case after == "":
		t.Fatal("the control plane records no first push after a successful push")
	case before != "" && after != before:
		t.Fatalf("first_push_at moved from %s to %s; the event must be recorded once", before, after)
	case before == "":
		t.Logf("first push recorded at %s", after)
	}
	if out := laptop.mustRun("second push", "push", "--all"); !strings.Contains(out, "skipped 3 unchanged") {
		t.Fatalf("second push %q", out)
	}
	if again := firstPushAt(laptop); again != after {
		t.Fatalf("a no-op push moved first_push_at from %s to %s", after, again)
	}

	// --- the device is wiped ---
	home.wipe(t)
	fresh := newHopDevice(t, nil, "acceptance-laptop-wiped")
	fresh.verifier = preflight.DefaultService()
	signIn(fresh)
	fresh.mustRun("init --hop again", "init", "--hop", "--project", "local/first-push="+home.project)
	if _, errb, code := fresh.run("pull", "--all"); code == ExitOK || !strings.Contains(errb, "rein account recover") {
		t.Fatalf("pull before recover: exit=%d err=%q", code, errb)
	}
	fresh.typedCode = recoveryCode
	fresh.mustRun("account recover", "account", "recover")
	fresh.typedCode = ""
	home.reinstallAgents(t)
	home.reinstallOpenCode(t)
	pullOut := fresh.mustRun("pull --all", "pull", "--all", "--json")
	var pulled struct {
		Pulled int `json:"pulled"`
	}
	if err := json.Unmarshal([]byte(pullOut), &pulled); err != nil || pulled.Pulled != 3 {
		t.Fatalf("pull output %q: %v", pullOut, err)
	}
	if restored, err := os.ReadFile(home.claude); err != nil || !bytes.Contains(restored, []byte("synthetic first push claude")) {
		t.Fatalf("claude session not restored: %v", err)
	}
	if restored, err := os.ReadFile(home.codex); err != nil || !bytes.Contains(restored, []byte("synthetic first push codex")) {
		t.Fatalf("codex session not restored: %v", err)
	}
	if n := openCodeMessageCount(t, home.db, "ses_fixture001"); n != 2 {
		t.Fatalf("opencode session restored with %d messages, want 2", n)
	}

	// Verified resume with the real environment verifier. A host without a
	// vendor binary is blocked on agent.executable by design; that is
	// recorded, not failed, so the locker journey stays the subject.
	for _, ref := range []string{"claude:session-first-push", "codex:rollout-first-push", "opencode:ses_fixture001"} {
		out, errb, code := fresh.run("resume", ref, "--dry-run", "--json")
		switch {
		case code == ExitOK:
			t.Logf("resume %s: verified, ready", ref)
		case strings.Contains(errb, "agent.executable"):
			t.Logf("resume %s: blocked only because the vendor executable is not installed here", ref)
		default:
			t.Fatalf("resume %s: exit=%d out=%q err=%q", ref, code, out, errb)
		}
	}
	if final := firstPushAt(fresh); final != after {
		t.Fatalf("first_push_at moved from %s to %s over the journey", after, final)
	}
}
