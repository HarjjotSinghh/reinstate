package verify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// noAnswer is a locker whose endpoint says nothing at all: the request
// times out, the connection drops, the name does not resolve. It is the
// difference between a check that failed and a check that could not run,
// and before this file steps 1 and 2 called it a failure — which is what
// the shipped promise "a check that could not run is never reported as a
// check that failed" said they did not.
type noAnswer struct {
	backend.Backend
	// on names the object whose fetch gives no answer; empty means the
	// listing itself gives none.
	on  string
	err error
}

func (n noAnswer) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	if n.on != "" {
		return n.Backend.List(ctx, prefix)
	}
	return nil, n.err
}

func (n noAnswer) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	if n.on != "" && !strings.HasSuffix(key, n.on) {
		return n.Backend.Get(ctx, key)
	}
	return nil, backend.ObjectMeta{}, n.err
}

// timeout is what a request that never came back looks like to the S3
// client: a net.Error the backend could not map to any refusal.
var timeout = fmt.Errorf("backend: list: %w", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("i/o timeout")})

// dropped is the other shape: the connection went away mid-response.
var dropped = fmt.Errorf("backend: get: %w", io.ErrUnexpectedEOF)

// TestStepsOneAndTwoDoNotFailOnAnEndpointThatDidNotAnswer holds steps 1
// and 2 to the rule step 4 already followed. A refusal is an answer and
// fails the step; no answer at all is a check that could not run, and the
// run says NOT VERIFIED rather than passing on the steps that did run or
// failing on the ones that could not.
func TestStepsOneAndTwoDoNotFailOnAnEndpointThatDidNotAnswer(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name        string
		backend     func(*memory.Store) backend.Backend
		step        string
		status      Status
		observed    string
		outcome     Status
		summary     string
		notVerified bool
	}{
		{
			name:        "the listing times out",
			backend:     func(s *memory.Store) backend.Backend { return noAnswer{Backend: s, err: timeout} },
			step:        StepList,
			status:      NotApplicable,
			observed:    "Could not run: the storage endpoint gave no answer to the listing",
			outcome:     NotApplicable,
			summary:     "NOT VERIFIED. The locker could not be listed:",
			notVerified: true,
		},
		{
			// One of the two objects still answered, so the run reaches a
			// verdict on that one — and says, in the outcome sentence, that
			// the other was not checked.
			name:     "the index fetch drops the connection",
			backend:  func(s *memory.Store) backend.Backend { return noAnswer{Backend: s, on: "manifest.age", err: dropped} },
			step:     StepCiphertext,
			status:   NotApplicable,
			observed: "Could not run: the storage endpoint gave no answer for manifest.age",
			outcome:  Pass,
			summary:  "manifest.age could not be fetched:",
		},
		{
			name: "the listing is refused, which is an answer",
			backend: func(s *memory.Store) backend.Backend {
				return noAnswer{Backend: s, err: &backend.Refusal{Code: "AccessDenied"}}
			},
			step:     StepList,
			status:   Fail,
			observed: "The listing was refused or failed:",
			outcome:  Fail,
			summary:  "FAIL. At least one step did not hold.",
		},
		{
			name: "the index fetch is refused, which is an answer",
			backend: func(s *memory.Store) backend.Backend {
				return noAnswer{Backend: s, on: "manifest.age", err: &backend.Refusal{Code: "AccessDenied"}}
			},
			step:     StepCiphertext,
			status:   Fail,
			observed: "manifest.age could not be fetched:",
			outcome:  Fail,
			summary:  "FAIL. At least one step did not hold.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := lockerWith(t, keys, "team/a")
			r := Run(context.Background(), Options{
				Backend: tt.backend(store), Prefix: "team/a", Keys: keys, Storage: StorageBYO,
				ClientVersion: "rein test", Now: func() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) },
			})
			step := stepOf(t, r, tt.step)
			if !strings.Contains(step.Observed, tt.observed) {
				t.Fatalf("step %s observed %q, want it to contain %q", tt.step, step.Observed, tt.observed)
			}
			if step.Status != tt.status {
				t.Fatalf("step %s is %s, want %s", tt.step, step.Status, tt.status)
			}
			if r.Outcome != tt.outcome {
				t.Fatalf("outcome %s, want %s: %q", r.Outcome, tt.outcome, r.Summary)
			}
			if !strings.Contains(r.Summary, tt.summary) {
				t.Fatalf("summary %q, want it to contain %q", r.Summary, tt.summary)
			}
			if r.NotVerified() != tt.notVerified {
				t.Fatalf("NotVerified() = %v, want %v", r.NotVerified(), tt.notVerified)
			}
		})
	}
}

// TestAnUnansweredRunIsNotTheSameAsAnEmptyLocker keeps the two reports
// that end with no verdict apart. Both are outcome not-applicable and
// neither is a failure, but one tells the reader to push and the other
// tells them the locker was never reached — and only the second is worth
// a non-zero exit.
func TestAnUnansweredRunIsNotTheSameAsAnEmptyLocker(t *testing.T) {
	keys := rootKeys(t)
	empty := Run(context.Background(), Options{
		Backend: memory.New(), Keys: keys, Storage: StorageBYO, ClientVersion: "rein test",
	})
	if empty.Outcome != NotApplicable || empty.NotVerified() {
		t.Fatalf("an empty locker reads as unreachable: %+v", empty)
	}
	if !strings.HasPrefix(empty.Summary, "NOT YET VERIFIABLE.") {
		t.Fatalf("empty-locker summary %q", empty.Summary)
	}
	unreachable := Run(context.Background(), Options{
		Backend: noAnswer{Backend: memory.New(), err: timeout}, Keys: keys, Storage: StorageBYO, ClientVersion: "rein test",
	})
	if unreachable.Outcome != NotApplicable || !unreachable.NotVerified() {
		t.Fatalf("an unreachable locker reads as an empty one: %+v", unreachable)
	}
	if strings.Contains(unreachable.Summary, "Run rein push") {
		t.Fatalf("an unreachable locker was blamed on not having pushed: %q", unreachable.Summary)
	}
}

// TestAFetchThatGotNoAnswerIsNotAPass is the over-claim the could-not-run
// rule could have introduced: with steps 2 and 3 unable to run, step 1
// alone would have carried the report to PASS and the summary would have
// called objects it never fetched ciphertext.
func TestAFetchThatGotNoAnswerIsNotAPass(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "team/a")
	r := Run(context.Background(), Options{
		Backend: noAnswer{Backend: store, on: ".age", err: dropped}, Prefix: "team/a", Keys: keys,
		Storage: StorageBYO, ClientVersion: "rein test",
	})
	if stepOf(t, r, StepList).Status != Pass {
		t.Fatalf("step 1 did not pass: %+v", r.Steps)
	}
	if r.Outcome != NotApplicable || !r.NotVerified() {
		t.Fatalf("a run that opened nothing is outcome %s: %q", r.Outcome, r.Summary)
	}
	if strings.Contains(r.Summary, "ciphertext") {
		t.Fatalf("the summary called unfetched objects ciphertext: %q", r.Summary)
	}
}

// TestTheForeignBucketAlarmWaitsForThePin is the ordering the report used
// to get backwards. The List and fetch observations were turned into a
// Fail and into "credentials reached a bucket that is not its own" before
// the pin ran, and degrade cannot lift a Fail — so a report could carry
// the alarm, and tell the reader to mail security@, on the strength of an
// observation the pin then invalidated.
//
// The step still fails: a probe that answered a credential it should have
// refused is a contradiction wherever it happened. What it must not do is
// name the finding.
func TestTheForeignBucketAlarmWaitsForThePin(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name      string
		exchanges []Exchange
		alarm     bool
	}{
		{
			name:      "the probe landed at the pinned endpoint",
			exchanges: []Exchange{{Scheme: "https", Host: "s3.example", Status: 200}},
			alarm:     true,
		},
		{
			name:      "the probe landed somewhere else",
			exchanges: []Exchange{{Scheme: "https", Host: "other.example", Status: 200}},
			alarm:     false,
		},
		{
			name:      "the endpoint offered a redirect",
			exchanges: []Exchange{{Scheme: "https", Host: "s3.example", RedirectedTo: "https://other.example/"}},
			alarm:     false,
		},
		{
			// The case a single boolean over the whole probe got wrong.
			// The credential was answered at the pinned endpoint and only
			// then redirected somewhere else; withholding the alarm here
			// would hand a reference locker a way to read this account's
			// objects and suppress the finding by redirecting the next
			// request.
			name: "the endpoint answered and then offered a redirect",
			exchanges: []Exchange{
				{Scheme: "https", Host: "s3.example", Status: 200},
				{Scheme: "https", Host: "s3.example", RedirectedTo: "https://other.example/"},
			},
			alarm: true,
		},
		{
			name:      "the transport recorded nothing",
			exchanges: []Exchange{},
			alarm:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := lockerWith(t, keys, "team/a")
			ref := &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Region: "auto", Key: "reference/probe.txt"}
			r := Run(context.Background(), Options{
				Backend: store, Prefix: "team/a", Keys: keys, Storage: StorageHop, Reference: ref,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				OpenReference: openRef(answeringLocker{}, "AKIAHOP1", tt.exchanges...), CredentialID: credential("AKIAHOP1"),
				ClientVersion: "rein test",
			})
			if r.Outcome != Fail {
				t.Fatalf("a probe that answered the credential did not fail the run: %q", r.Summary)
			}
			step := stepOf(t, r, StepIsolation)
			named := strings.Contains(r.Summary, "reached a bucket that is not its own")
			if named != tt.alarm {
				t.Fatalf("summary %q\n names the foreign bucket = %v, want %v\nstep: %q", r.Summary, named, tt.alarm, step.Observed)
			}
			if reported := strings.Contains(r.Summary, "security@reinstate.dev"); reported != tt.alarm {
				t.Fatalf("summary %q asks for a security report = %v, want %v", r.Summary, reported, tt.alarm)
			}
			// Whatever the pin decided, the step says what the probe
			// answered; only the conclusion drawn from it is withheld.
			if !strings.Contains(step.Observed, "SUCCEEDED") {
				t.Fatalf("the step hid the successful probe: %q", step.Observed)
			}
			if !tt.alarm && strings.Contains(step.Observed, "credentials reach a bucket that is not its own") {
				t.Fatalf("the step concluded from an unplaced request: %q", step.Observed)
			}
		})
	}
}

// answeringLocker is a reference locker that answers this account's
// credential instead of refusing it: the one thing step 4 exists to
// catch.
type answeringLocker struct{ backend.Backend }

func (answeringLocker) List(context.Context, string) ([]backend.ObjectMeta, error) {
	return []backend.ObjectMeta{{Key: "reference/probe.txt", Size: 5}}, nil
}

func (answeringLocker) Get(context.Context, string) (io.ReadCloser, backend.ObjectMeta, error) {
	return io.NopCloser(strings.NewReader("probe")), backend.ObjectMeta{Key: "reference/probe.txt", Size: 5}, nil
}

func stepOf(t *testing.T, r *Report, id string) Step {
	t.Helper()
	for _, s := range r.Steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("report has no step %q", id)
	return Step{}
}
