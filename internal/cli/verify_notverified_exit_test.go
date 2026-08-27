package cli

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
)

// TestSyncVerifyExitsNotVerifiedWhenAStepGotNoAnswer drives the exit code
// three shipped pages name and no test reached.
//
// There are two routes to NOT VERIFIED and only one of them was covered.
// TestSyncVerifyOnAStorageEndpointThatAnswersNothing drives the route where
// the profile cannot be opened at all, which returns through
// notStartedReport. The other route is a run that started: the profile
// opened, step 1 asked the endpoint, and the endpoint said nothing. That
// one ends at `if report.NotVerified()` inside the command, and deleting
// that branch left the whole suite green while docs/hop.md,
// docs/hop/threat-model.md and docs/cli-reference.md all state the code.
//
// A BYO profile is the way to reach it, because it opens with a passphrase
// held on this device and never reads the locker to do so. On a Hop profile
// the keyring fetch fails first and the run takes the other route, which is
// exactly what the other test asserts.
func TestSyncVerifyExitsNotVerifiedWhenAStepGotNoAnswer(t *testing.T) {
	fake := s3test.NewPlain(t, "reinstate-byo-outage")
	fake.AcceptPrefix = "AKIA"
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_BYO_OUTAGE")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_BYO_OUTAGE")
	t.Setenv("REINSTATE_HOP_URL", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	project := writeClaudeFixture(t)

	run := func(args ...string) (string, string, int) {
		t.Helper()
		passphraseFile, err := os.CreateTemp(t.TempDir(), "passphrase-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = passphraseFile.Close() }()
		if _, err := passphraseFile.WriteString("verify-outage-passphrase-not-real\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := passphraseFile.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.FormatUint(uint64(passphraseFile.Fd()), 10))
		var out, errb bytes.Buffer
		code := Execute(Options{
			Name: "rein", Stdout: &out, Stderr: &errb, Args: args, Context: context.Background(),
			AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		})
		return out.String(), errb.String(), code
	}

	if out, errb, code := run("init", "--endpoint", fake.Srv.URL, "--bucket", "reinstate-byo-outage", "--prefix", "team/a", "--project", "local/outage="+project, "--yes"); code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := run("sync", "verify"); code != ExitOK {
		t.Fatalf("a healthy BYO verify exit=%d out=%q err=%q", code, out, errb)
	}

	// The bucket's endpoint stops answering. The passphrase is on this
	// device, so the profile still opens and the run still starts.
	fake.Srv.Close()

	out, errb, code := run("sync", "verify")
	if code != ExitRuntime {
		t.Fatalf("a run whose steps got no answer exited %d, want %d (runtime); an outage must not read as a pass and must not read as a failed check:\n%s\n%s", code, ExitRuntime, out, errb)
	}
	for _, want := range []string{"OUTCOME: NOT VERIFIED", "gave no answer"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the report does not contain %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"OUTCOME: FAIL", "OUTCOME: PASS", "security@reinstate.dev"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("a storage outage was reported as %q:\n%s", unwanted, out)
		}
	}
}
