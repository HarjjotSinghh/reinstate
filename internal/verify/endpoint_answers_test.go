package verify

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

// oversizedLocker serves an object in full and there is more of it than the
// check reads. Nothing about the endpoint is wrong; the limit is this
// package's own.
type oversizedLocker struct {
	backend.Backend
	on string
}

func (o oversizedLocker) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	if !strings.HasSuffix(key, o.on) {
		return o.Backend.Get(ctx, key)
	}
	body := strings.Repeat("x", maxObjectBytes+1)
	return io.NopCloser(strings.NewReader(body)), backend.ObjectMeta{Key: key, Size: int64(len(body))}, nil
}

// answeringWith is a locker whose endpoint answers with an API error this
// build has no particular case for. NoSuchBucket is the plainest one: the
// bucket named in the request does not exist. It is unambiguously something
// the endpoint said.
type answeringWith struct {
	backend.Backend
	err error
}

func (a answeringWith) List(context.Context, string) ([]backend.ObjectMeta, error) {
	return nil, a.err
}

// TestAnEndpointThatAnsweredIsNotAnEndpointThatSaidNothing is the rule the
// no-answer classifier has to follow in the direction that costs something.
//
// `answered` began as an allowlist of the error shapes somebody had thought
// of, which meant that every answer outside the list — every S3 API code
// the backend's switch has no case for, and this package's own size limit —
// was reported to a person as "the storage endpoint gave no answer". Two
// things go wrong at once. The sentence is false: the endpoint answered.
// And the run stops failing: a locker that will not serve its index in a
// readable size, or whose bucket is gone, used to fail step 1 or 2 and exit
// 7, and instead reported a check that could not run.
func TestAnEndpointThatAnsweredIsNotAnEndpointThatSaidNothing(t *testing.T) {
	keys := rootKeys(t)

	t.Run("an API code this build has no case for", func(t *testing.T) {
		store := lockerWith(t, keys, "team/a")
		gone := &backend.APIAnswer{Code: "NoSuchBucket", Err: errors.New("api error NoSuchBucket: The specified bucket does not exist")}
		r := Run(context.Background(), Options{
			Backend: answeringWith{Backend: store, err: gone}, Prefix: "team/a", Keys: keys,
			Storage: StorageBYO, ClientVersion: "rein test",
		})
		step := stepOf(t, r, StepList)
		if step.Status != Fail {
			t.Fatalf("a listing the endpoint refused with NoSuchBucket is step 1 = %q, want %q: %q", step.Status, Fail, step.Observed)
		}
		if strings.Contains(step.Observed, "gave no answer") {
			t.Fatalf("step 1 says the endpoint gave no answer, and it gave one: %q", step.Observed)
		}
		if r.Outcome != Fail {
			t.Fatalf("outcome %q, want %q: %q", r.Outcome, Fail, r.Summary)
		}
	})

	t.Run("an object larger than the check reads", func(t *testing.T) {
		store := lockerWith(t, keys, "team/a")
		r := Run(context.Background(), Options{
			Backend: oversizedLocker{Backend: store, on: manifestObject}, Prefix: "team/a", Keys: keys,
			Storage: StorageBYO, ClientVersion: "rein test",
		})
		step := stepOf(t, r, StepCiphertext)
		if strings.Contains(step.Observed, "gave no answer") {
			t.Fatalf("step 2 says the endpoint gave no answer about an object it served in full: %q", step.Observed)
		}
		if r.Outcome == Pass {
			t.Fatalf("a locker that will not serve its index in a readable size passed: %q", r.Summary)
		}
		if !strings.Contains(step.Observed, "64 MiB") {
			t.Fatalf("step 2 does not say what the limit was: %q", step.Observed)
		}
	})
}

// TestAnsweredIsNotAnAllowlistOfNames pins the classifier itself, so the
// two cases above cannot be fixed by special-casing them at the call site
// and leaving the next unnamed answer to be reported as silence.
func TestAnsweredIsNotAnAllowlistOfNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: true},
		{name: "a scope refusal", err: &backend.Refusal{Code: "AccessDenied"}, want: true},
		{name: "a rejected credential", err: &backend.Refusal{Code: "InvalidAccessKeyId", Credential: true}, want: true},
		{name: "not found", err: backend.ErrNotFound, want: true},
		{name: "an API code with no case", err: &backend.APIAnswer{Code: "AccountProblem"}, want: true},
		{name: "an API code wrapped by a caller", err: errors.Join(errors.New("backend: list"), &backend.APIAnswer{Code: "RequestTimeTooSkewed"}), want: true},
		{name: "this package's own size limit", err: ErrObjectTooLarge, want: true},
		{name: "a request that timed out", err: timeout, want: false},
		{name: "a connection that dropped", err: dropped, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := answered(tc.err); got != tc.want {
				t.Fatalf("answered(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
