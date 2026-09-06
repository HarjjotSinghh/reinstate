package cli

import (
	"strings"
	"testing"
)

// TestSyncVerifyOnAStorageEndpointThatAnswersNothing drives, through the
// real CLI, the case the shipped sentence "a check that could not run is
// never reported as a check that failed" promised and steps 1 and 2 did
// not honour: the control plane is up, the credentials mint, and the
// storage endpoint is simply not there.
//
// Before this, the run ended with a bare SDK dial error and exit 4
// (`AuthStorage`) — the code for a device that is not authorised, for a
// socket that was refused — and printed no report at all. It now prints
// the four checks that did not run, says NOT VERIFIED, and exits 1: the
// same code an unreachable control plane uses, so a script cannot read an
// outage as a pass either.
//
// A Hop profile fetches the keyring from the locker while it opens, so a
// storage outage stops the command before step 1. The other route — the
// profile opens and a step gets no answer — is where BYO storage lands,
// and is covered in internal/verify by
// TestStepsOneAndTwoDoNotFailOnAnEndpointThatDidNotAnswer.
func TestSyncVerifyOnAStorageEndpointThatAnswersNothing(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	// The locker's endpoint stops answering. The control plane still does,
	// so this is a storage outage and nothing else.
	j.plane.s3.Srv.Close()

	out, errb, code := j.run("sync", "verify", "--post=false")
	if code != ExitRuntime {
		t.Fatalf("exit=%d, want %d (runtime):\n%s\n%s", code, ExitRuntime, out, errb)
	}
	for _, want := range []string{
		"Could not run: the storage endpoint gave no answer",
		"so the locker could not be opened or listed",
		"OUTCOME: NOT VERIFIED. Nothing was checked, because the storage endpoint gave no answer",
		"Run rein sync verify again when the storage endpoint answers.",
		"Result:         NOT APPLICABLE",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("report does not contain %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"OUTCOME: FAIL", "security@reinstate.dev", "NOT YET VERIFIABLE", "Run rein push"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("a storage outage was reported as %q:\n%s", unwanted, out)
		}
	}
	// Nothing was checked, so nothing is posted to the account console.
	j.plane.mu.Lock()
	posted := len(j.plane.reports)
	j.plane.mu.Unlock()
	if posted != 1 {
		t.Fatalf("reports posted = %d, want only the one the first push sent", posted)
	}
}
