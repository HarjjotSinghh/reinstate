//go:build hopacceptance

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
)

// TestHopFirstPushJourneyStaging runs the first-push journey of
// TestHopFirstPushJourney against a real control plane and a real locker.
// It is built only with -tags hopacceptance and skips unless the
// environment names a staging control plane and a device token:
//
//	HOP_STAGING_URL     control plane base URL (https://hop-staging.example)
//	HOP_DEVICE_TOKEN    a device token issued by that control plane
//	                    (`rein login` needs a browser, so sign in once by
//	                    hand and export the token for the run)
//	HOP_ACCOUNT_ID      optional; recorded in the token for `rein whoami`
//
// The account must be a disposable staging account: the journey pushes
// three synthetic sessions into its locker, and the first_push check is
// only exact when the locker has never seen a push before. It never runs
// in CI and never against production.
func TestHopFirstPushJourneyStaging(t *testing.T) {
	stagingURL := strings.TrimSpace(os.Getenv("HOP_STAGING_URL"))
	token := strings.TrimSpace(os.Getenv("HOP_DEVICE_TOKEN"))
	if stagingURL == "" || token == "" {
		t.Skip("HOP_STAGING_URL and HOP_DEVICE_TOKEN are not set; the staging first-push journey is skipped")
	}
	if strings.Contains(stagingURL, "hop.reinstate.dev") && !strings.Contains(stagingURL, "staging") {
		t.Fatalf("refusing to run the first-push journey against %s; use a staging control plane", stagingURL)
	}
	t.Setenv(hopURLEnv, stagingURL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(env, "")
	}
	home := plantJourneyHome(t, t.TempDir())

	signIn := func(d *hopDevice) {
		t.Helper()
		// The token stands in for the browser round-trip of `rein login`.
		if err := d.tokens.SetDeviceToken(credentials.DeviceToken{Token: token, ControlPlaneURL: stagingURL, AccountID: os.Getenv("HOP_ACCOUNT_ID")}); err != nil {
			t.Fatal(err)
		}
		d.mustRun("whoami", "whoami")
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
	signIn(laptop)
	before := firstPushAt(laptop)
	if before != "" {
		t.Logf("the locker already records a first push at %s; this run can only check it does not change", before)
	}
	laptop.mustRun("init --hop", "init", "--hop", "--project", "local/first-push="+home.project)
	laptop.mustRun("account init", "account", "init")
	recoveryCode := laptop.shownCode
	pushOut := laptop.mustRun("push --all", "push", "--all", "--json")
	elapsed := time.Since(start)
	var pushed struct {
		Snapshots []string `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(pushOut), &pushed); err != nil || len(pushed.Snapshots) != 3 {
		t.Fatalf("push output %q: %v", pushOut, err)
	}
	t.Logf("sign-in to first successful push against %s: %s (budget %s)", stagingURL, elapsed.Round(time.Millisecond), firstPushBudget)
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
	fresh := newHopDevice(t, nil, "acceptance-laptop")
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
