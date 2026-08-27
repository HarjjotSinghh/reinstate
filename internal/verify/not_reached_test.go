package verify

import (
	"errors"
	"io/fs"
	"strings"
	"syscall"
	"testing"
)

// TestANotStartedBYOReportDescribesNoReferenceLocker closes the other route
// into a step-4 sentence.
//
// The running report's isolation step was given a BYO-specific "What was
// done" so a reader of a BYO report is not told about a control plane it
// does not have. NotReached is the second route, reached by
// notStartedReport whenever the profile is not a Hop one, and it went on
// describing a reference locker: "step 1 could not list this account's
// locker, so a refusal from another bucket would show nothing about bucket
// scope". There is no other bucket in a BYO profile's verification.
func TestANotStartedBYOReportDescribesNoReferenceLocker(t *testing.T) {
	r := NotReached(Options{Storage: StorageBYO, ClientVersion: "rein test"}, errors.New("dial tcp: connection refused"))
	step := stepOf(t, r, StepIsolation)
	// It may say a reference locker is what a Hop profile would have had;
	// what it may not do is describe this profile as having one, or blame
	// step 1 for a comparison this profile never makes.
	for _, phrase := range []string{"step 1 could not list", "would show nothing about bucket scope"} {
		if strings.Contains(step.Observed, phrase) {
			t.Fatalf("a BYO report that did not start says %q: %q", phrase, step.Observed)
		}
	}
	for _, phrase := range []string{"bucket you configured yourself", "No request was made."} {
		if !strings.Contains(step.Observed, phrase) {
			t.Fatalf("step 4 does not say %q: %q", phrase, step.Observed)
		}
	}

	// The Hop wording is unchanged: there the reference locker is real and
	// naming it is the point.
	hopReport := NotReached(Options{Storage: StorageHop, ClientVersion: "rein test"}, errors.New("dial tcp: connection refused"))
	if hopStep := stepOf(t, hopReport, StepIsolation); !strings.Contains(hopStep.Observed, "bucket scope") {
		t.Fatalf("the Hop wording changed too: %q", hopStep.Observed)
	}
}

// TestUnreachableIsAboutTheEndpointAndNotTheFilesystem pins the predicate
// against the one error shape that reads as a network failure and is not
// one.
//
// syscall.Errno has Timeout() and Temporary(), so it satisfies net.Error;
// an *fs.PathError wrapping ENOENT or EACCES therefore matched the
// net.Error case and an unreadable local file was classified as a storage
// endpoint that gave no answer. The exported predicate's own documentation
// says the opposite, and the retry advice a caller prints on the strength
// of it — wait for the endpoint — is advice about the wrong machine.
func TestUnreachableIsAboutTheEndpointAndNotTheFilesystem(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "a missing local file", err: &fs.PathError{Op: "open", Path: "account.json", Err: syscall.ENOENT}, want: false},
		{name: "a local file this user may not read", err: &fs.PathError{Op: "open", Path: "device.key", Err: syscall.EACCES}, want: false},
		{name: "a missing local file, wrapped", err: errors.Join(errors.New("load the device key"), &fs.PathError{Op: "open", Path: "device.key", Err: syscall.ENOENT}), want: false},
		{name: "a request that timed out", err: timeout, want: true},
		{name: "a connection that dropped", err: dropped, want: true},
		{name: "a refusal the endpoint gave", err: nil, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Unreachable(tc.err); got != tc.want {
				t.Fatalf("Unreachable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
