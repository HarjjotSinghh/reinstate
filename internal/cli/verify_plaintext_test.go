package cli

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
)

// TestSyncVerifyRefusesAPlaintextEndpointThroughTheRealClient drives the
// isolation step's plaintext refusal end to end: the real `rein sync
// verify`, the real S3 client, and a locker the control plane really
// listed — at an address that is not loopback.
//
// That last part is the whole point of the test. The refusal exempts
// loopback, and every fake locker in this package is an httptest server,
// which listens on 127.0.0.1 — the one address the carve-out lets
// through. So every CLI-level run of this code took the exempt path, and
// the refusal itself was covered only by a unit test of the predicate.
// Here the fake locker is bound to a non-loopback address of this
// machine, the control plane advertises it over plain `http` for both the
// locker and its reference locker, and the step has to refuse.
//
// It skips where the machine has no non-loopback address it can bind and
// dial (a container with only `lo`, a laptop with every interface down).
// The step's other outcomes are covered without one; this one is not, and
// a skip says so rather than passing quietly.
func TestSyncVerifyRefusesAPlaintextEndpointThroughTheRealClient(t *testing.T) {
	ln, ok := s3test.NonLoopbackListener(t)
	if !ok {
		t.Skip("no non-loopback local address to bind: the plaintext refusal cannot be driven through the real client here")
	}
	j := newLockerJourney(t)
	j.plane.s3 = s3test.NewPlainOn(t, "lk-0000000000000000000000test", ln)
	j, _ = hostedVerifyJourneyOn(t, j)
	endpoint := j.plane.s3.URL()
	if strings.Contains(endpoint, "127.0.0.1") || strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "[::1]") {
		t.Fatalf("the locker is at a loopback address after all: %s", endpoint)
	}
	if !strings.HasPrefix(endpoint, "http://") {
		t.Fatalf("the locker is not a plaintext endpoint: %s", endpoint)
	}

	// The push proves the whole client path works at this address: the
	// refusal below is the isolation step's decision, not a locker nobody
	// could reach.
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	before := len(j.plane.s3.RequestLog())

	out, errb, code := j.run("sync", "verify", "--json", "--post=false")
	if code != ExitSafety {
		t.Fatalf("exit=%d, want %d (safety):\n%s\n%s", code, ExitSafety, out, errb)
	}
	v := decodeVerify(t, out)
	step := stepByID(t, v.Report, "isolation")
	if step.Status != "fail" {
		t.Fatalf("isolation step %+v", step)
	}
	for _, want := range []string{
		"which is plaintext http",
		"it sends those over an unencrypted connection to nothing but this machine's own loopback address",
		"no request was made and nothing about bucket scope was shown",
	} {
		if !strings.Contains(step.Observed, want) {
			t.Fatalf("isolation step observed %q, want it to contain %q", step.Observed, want)
		}
	}
	if v.Report.Outcome != "fail" || v.Report.IsolationChecked() {
		t.Fatalf("outcome %s, isolation checked %v", v.Report.Outcome, v.Report.IsolationChecked())
	}
	// The refusal is worth nothing unless the credential stayed home. The
	// fake records every request it is offered, and marks the ones that
	// named a bucket other than the locker's.
	for _, entry := range j.plane.s3.RequestLog()[before:] {
		if strings.Contains(entry, "(foreign bucket)") {
			t.Fatalf("the credential was signed for the reference bucket anyway: %q", entry)
		}
	}
}

// TestSyncVerifyExemptsLoopbackFromThePlaintextRefusal is the other half
// of the carve-out, stated as a test rather than only as a sentence: the
// same plain `http` endpoint passes when it is loopback, which is why
// every other journey in this package can run at all. If the exemption is
// ever removed, this test fails and the docs that state it have to change
// with it.
func TestSyncVerifyExemptsLoopbackFromThePlaintextRefusal(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	if endpoint := j.plane.s3.URL(); !strings.HasPrefix(endpoint, "http://127.0.0.1") {
		t.Fatalf("this journey's locker is not a plaintext loopback endpoint: %s", endpoint)
	}
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	out, errb, code := j.run("sync", "verify", "--json", "--post=false")
	if code != ExitOK {
		t.Fatalf("exit=%d, want %d:\n%s\n%s", code, ExitOK, out, errb)
	}
	v := decodeVerify(t, out)
	step := stepByID(t, v.Report, "isolation")
	if step.Status != "pass" || strings.Contains(step.Observed, "plaintext http") {
		t.Fatalf("a loopback endpoint was refused: %+v", step)
	}
}
