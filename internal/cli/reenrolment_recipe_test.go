package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
)

// clearHopEnv blanks the environment a Hop journey must not inherit from
// the developer's shell.
func clearHopEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
}

// TestReEnrolmentRecipeEndsInAWorkingPush runs the recipe `docs/hop.md`
// prints for a revoked machine, verbatim and to the end.
//
// `rein init --hop --force` is in it for one reason: it is the only thing in
// the CLI that removes the enrolment record, and both `rein account join`
// and `rein account recover` refuse to run where one exists. Clearing
// state.json was collateral, and it cost the recipe its last step — with the
// session records gone, `rein push` sees a local revision and a remote
// snapshot that differ with no shared base, records a conflict, and exits 6.
// A recovery path whose final command fails on state the path itself threw
// away is not a recovery path.
//
// Falsified by making initHosted write schema.NewState() unconditionally
// again: the final push then exits 6 with "session(s) diverged".
func TestReEnrolmentRecipeEndsInAWorkingPush(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-000000000000000000000recipe")
	t.Setenv(hopURLEnv, plane.srv.URL)
	clearHopEnv(t)
	project := writeClaudeFixture(t)

	// A holds the account; B is the machine that will be revoked and
	// re-enrolled, and it is the one that pushes this session.
	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	b.enrol(project, a.shownCode)
	if out, errb, code := b.run("push", "--all", "--json"); code != ExitOK {
		t.Fatalf("B push: exit=%d out=%q err=%q", code, out, errb)
	}
	beforeState, err := config.LoadState(b.home)
	if err != nil || len(beforeState.Sessions) == 0 {
		t.Fatalf("B recorded no sync state: %+v %v", beforeState, err)
	}

	bID := deviceID(t, b)
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}

	// The agent kept working on the machine after it was revoked, so the
	// local session has moved past the snapshot in the locker. This is the
	// ordinary case, not a corner: a machine is revoked because it was lost
	// or retired, and the one being re-enrolled has been in use.
	appendToClaudeSession(t, project, `{"type":"user","message":{"content":"written after the revocation"}}`)

	// The recipe, verbatim from docs/hop.md.
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--force", "--project", "local/locker=" + project}} {
		if out, errb, code := b.run(args...); code != ExitOK {
			t.Fatalf("B %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	if out, errb, code := b.recover(a.shownCode); code != ExitOK {
		t.Fatalf("B recover: exit=%d out=%q err=%q", code, out, errb)
	}
	out, errb, code := b.run("push", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("the last step of the documented recovery recipe failed: exit=%d out=%q err=%q", code, out, errb)
	}
	if strings.Contains(out+errb, "diverged") {
		t.Fatalf("the recipe ended in a conflict: out=%q err=%q", out, errb)
	}
	var pushed struct {
		Snapshots []string `json:"snapshots"`
		Conflicts []string `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(out), &pushed); err != nil {
		t.Fatalf("push --json: %v (%q)", err, out)
	}
	if len(pushed.Conflicts) != 0 || len(pushed.Snapshots) == 0 {
		t.Fatalf("push after re-enrolment: %+v", pushed)
	}

	// The re-initialization still backed the old state up, and the enrolment
	// record it had to remove is gone from the live home.
	backups, err := filepath.Glob(filepath.Join(b.home, "backups", "*", "state.json"))
	if err != nil || len(backups) == 0 {
		t.Fatalf("no state.json in the backup set: %v %v", backups, err)
	}
}

// TestInitForceStartsCleanOnADifferentProfile is the other half of the rule:
// the sync state is carried only when the home is being pointed at the same
// profile. A different profile is a different locker, whose snapshot ids
// mean nothing here, so that state is dropped.
func TestInitForceStartsCleanOnADifferentProfile(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000newprofile")
	t.Setenv(hopURLEnv, plane.srv.URL)
	clearHopEnv(t)
	project := writeClaudeFixture(t)

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	before, err := config.LoadState(a.home)
	if err != nil || len(before.Sessions) == 0 {
		t.Fatalf("A recorded no sync state: %+v %v", before, err)
	}
	// Rewrite the config to name a different profile, as pointing the home
	// at another account does.
	cfg, err := config.LoadConfig(a.home)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ProfileID = "99999999-2222-4333-8444-555555555555"
	if err := config.SaveConfig(a.home, cfg); err != nil {
		t.Fatal(err)
	}
	if out, errb, code := a.run("init", "--hop", "--force", "--project", "local/locker="+project); code != ExitOK {
		t.Fatalf("A re-init: exit=%d out=%q err=%q", code, out, errb)
	}
	after, err := config.LoadState(a.home)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Sessions) != 0 || after.LastManifestRev != "" {
		t.Fatalf("state from another profile was carried forward: %+v", after)
	}
}

// appendToClaudeSession adds a line to the fixture session, so the local
// revision moves past what is in the locker.
func appendToClaudeSession(t *testing.T, project, line string) {
	t.Helper()
	userHome := os.Getenv("HOME")
	path := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project), "session-locker.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}
