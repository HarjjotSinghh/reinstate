package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// securityReport is the sentence that must appear only when something the
// operator did is actually wrong.
const securityReport = "security@reinstate.dev"

// TestSummaryTellsTheFailuresApart is the blocker this file exists for.
// One outcome sentence for every way a run can end tells a first-time user
// with nothing pushed, and a user who mistyped a passphrase, to report a
// security incident. That teaches both of them that this command's alarm
// means nothing — and it is the one alarm that has to be believed the
// single time it fires for real.
func TestSummaryTellsTheFailuresApart(t *testing.T) {
	keys := rootKeys(t)
	other := rootKeys(t)
	leaky := memoryWithProbe(t)
	tests := []struct {
		name    string
		storage string
		mutate  func(o *Options, s *memory.Store)
		outcome Status
		want    []string
		notWant []string
	}{
		{
			name:    "nothing pushed yet",
			storage: StorageHop,
			mutate:  func(_ *Options, s *memory.Store) { _ = s.Delete(context.Background(), "manifest.age") },
			outcome: NotApplicable,
			want: []string{
				"NOT YET VERIFIABLE.",
				"Nothing has been pushed from this profile yet",
				"Run rein push, then rein sync verify again.",
			},
			notWant: []string{securityReport, "FAIL"},
		},
		{
			name:    "a passphrase that does not open a BYO locker",
			storage: StorageBYO,
			mutate:  func(o *Options, _ *memory.Store) { o.Keys = other },
			outcome: Fail,
			want: []string{
				"The objects in the locker are ciphertext",
				"a different passphrase than the one given at rein init",
				"If you are certain it is the right one",
			},
			notWant: []string{"An object in the locker is not encrypted"},
		},
		{
			name:    "a root key that does not open a Hop locker",
			storage: StorageHop,
			mutate:  func(o *Options, _ *memory.Store) { o.Keys = other },
			outcome: Fail,
			want: []string{
				"the root key this device holds did not open them",
				"rein account recover",
				"If this device has opened these objects before",
			},
			notWant: []string{"passphrase"},
		},
		{
			name:    "plaintext in the locker",
			storage: StorageHop,
			mutate: func(_ *Options, s *memory.Store) {
				put(t, s, "manifest.age", []byte(`{"schema_version":1,"revision":"r1","sessions":{}}`))
			},
			outcome: Fail,
			want: []string{
				"An object in the locker is not encrypted",
				"This is what these checks exist to catch.",
				securityReport,
			},
		},
		{
			name:    "the credentials reach another bucket",
			storage: StorageHop,
			mutate: func(o *Options, _ *memory.Store) {
				o.OpenReference = openRef(leaky, "AKIAHOP1",
					Exchange{Host: "s3.example", Status: 200},
					Exchange{Host: "s3.example", Status: 200})
			},
			outcome: Fail,
			want: []string{
				"reached a bucket that is not its own",
				"This is what these checks exist to catch.",
				securityReport,
			},
		},
		{
			name:    "a listing nobody could make",
			storage: StorageHop,
			mutate: func(o *Options, _ *memory.Store) {
				o.Backend = refusing{err: &backend.Refusal{Code: "AccessDenied"}}
			},
			outcome: Fail,
			want: []string{
				"At least one step did not hold.",
				"The failed step above names what was seen and what to do about it.",
			},
			notWant: []string{securityReport},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: tc.storage,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")}
			if tc.storage == StorageHop {
				o.Reference = &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"}
			}
			tc.mutate(&o, store)
			r := Run(context.Background(), o)
			if r.Outcome != tc.outcome {
				t.Fatalf("outcome %s, want %s; summary %q", r.Outcome, tc.outcome, r.Summary)
			}
			for _, want := range tc.want {
				if !strings.Contains(r.Summary, want) {
					t.Fatalf("summary %q lacks %q", r.Summary, want)
				}
			}
			for _, unwanted := range tc.notWant {
				if strings.Contains(r.Summary, unwanted) {
					t.Fatalf("summary %q still says %q", r.Summary, unwanted)
				}
			}
			// The human output ends with exactly the summary, so a reader and
			// a decoder cannot be told two different things.
			var human bytes.Buffer
			r.WriteHuman(&human)
			if !strings.Contains(human.String(), "OUTCOME: "+r.Summary) {
				t.Fatalf("human report does not end with the summary:\n%s", human.String())
			}
		})
	}
}

// TestNothingPushedRunsNoFailedStep: with nothing to check, no step is a
// failure. Each says what it could not do and why, and the report as a
// whole is neither a pass nor a failure.
func TestNothingPushedRunsNoFailedStep(t *testing.T) {
	keys := rootKeys(t)
	r := Run(context.Background(), Options{Backend: memory.New(), Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	if r.Outcome != NotApplicable || r.Passed() || r.Failed() {
		t.Fatalf("outcome %s (passed=%t failed=%t)", r.Outcome, r.Passed(), r.Failed())
	}
	for i, s := range r.Steps {
		if s.Status != NotApplicable {
			t.Fatalf("step %d is %s, want not-applicable: %+v", i+1, s.Status, s)
		}
	}
	for i, want := range []string{
		"nothing has been pushed from this profile yet, so there is nothing to check",
		"the locker holds no index yet",
		"no object was fetched",
		"nothing has been pushed from this profile yet",
	} {
		if !strings.Contains(r.Steps[i].Observed, want) {
			t.Fatalf("step %d observed %q, want it to contain %q", i+1, r.Steps[i].Observed, want)
		}
	}
}

// TestNotRunReportsAnUnreachableControlPlane: a Hop locker is listed with
// credentials the control plane mints, so an outage stops every check
// before it starts. The reader still gets a report saying which four
// checks did not run and why, instead of a bare dial error they cannot
// tell from a finding.
func TestNotRunReportsAnUnreachableControlPlane(t *testing.T) {
	cause := &url.Error{Op: "Get", URL: "https://hop.reinstate.dev/v1/locker", Err: errors.New("dial tcp 10.0.0.1:443: connectex: no such host")}
	r := NotRun(Options{Storage: StorageHop, ClientVersion: "rein test"}, cause)
	if r.Outcome != NotApplicable || r.Failed() || len(r.Steps) != 4 {
		t.Fatalf("report %+v", r)
	}
	for i, s := range r.Steps {
		if s.Status != NotApplicable {
			t.Fatalf("step %d is %s, want not-applicable", i+1, s.Status)
		}
		if !strings.HasPrefix(s.Observed, "Could not run: ") {
			t.Fatalf("step %d does not say it could not run: %q", i+1, s.Observed)
		}
	}
	for _, want := range []string{"NOT VERIFIED.", "the control plane could not be reached", cause.Error(), "No step failed"} {
		if !strings.Contains(r.Summary, want) {
			t.Fatalf("summary %q lacks %q", r.Summary, want)
		}
	}
	if strings.Contains(r.Summary, securityReport) {
		t.Fatalf("an outage asks for a security report: %q", r.Summary)
	}
}

// TestIsolationDoesNotFailOnAnUnreachableControlPlane: when the locker
// opened but the reference lookup did not, the three checks that ran
// against storage and the local key still stand, and step 4 says it could
// not run rather than reporting a failure of a property it never tested.
func TestIsolationDoesNotFailOnAnUnreachableControlPlane(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	cause := &url.Error{Op: "Get", URL: "https://hop.reinstate.dev/v1/verify/reference", Err: errors.New("dial tcp: connection refused")}
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker: LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"}, ReferenceErr: cause,
		CredentialID: credential("AKIAHOP1")})
	if r.Outcome != Pass || r.Failed() {
		t.Fatalf("outcome %s: %+v", r.Outcome, r.Steps[3])
	}
	step := r.Steps[3]
	if step.Status != NotApplicable || r.IsolationChecked() {
		t.Fatalf("isolation step %+v", step)
	}
	for _, want := range []string{"Could not run: the control plane could not be reached", cause.Error(), "The first three steps above ran entirely against storage"} {
		if !strings.Contains(step.Observed, want) {
			t.Fatalf("isolation step %q lacks %q", step.Observed, want)
		}
	}
	if !strings.Contains(r.Summary, "was not checked (step 4 above says why)") {
		t.Fatalf("summary claims isolation: %q", r.Summary)
	}
	// A control plane that answered, and answered something unexpected, is
	// still a failure: only a service nobody could reach is excused.
	answered := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:       LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		ReferenceErr: &hop.Error{Status: 500, Message: "internal error"}, CredentialID: credential("AKIAHOP1")})
	if answered.Outcome != Fail || answered.Steps[3].Status != Fail {
		t.Fatalf("a control plane answering 500 was excused: %+v", answered.Steps[3])
	}
}

// TestReportJSONCarriesWhatTheHumanOutputCarries: a consumer that decodes
// the document has to be able to rebuild the sentence a person reads.
// Without the summary and the two lists behind it, a decoder sees
// outcome:pass and can render "everything verified" — the exact
// over-claim the human text was written to avoid.
func TestReportJSONCarriesWhatTheHumanOutputCarries(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	put(t, store, "keyring.v1.json", []byte(`{"schema_version":1}`))
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	if !strings.Contains(r.Summary, "the wrapped keyring") {
		t.Fatalf("the fixture does not exercise the unopened clause: %q", r.Summary)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Summary != r.Summary {
		t.Fatalf("decoded summary %q, want %q", decoded.Summary, r.Summary)
	}
	if got, want := strings.Join(decoded.CheckedObjects, "|"), strings.Join(r.CheckedObjects, "|"); got != want {
		t.Fatalf("decoded checked_objects %q, want %q", got, want)
	}
	if decoded.Unopened != r.Unopened || decoded.Unopened == "" {
		t.Fatalf("decoded unopened %q, want %q", decoded.Unopened, r.Unopened)
	}
	// The upload form is unchanged by any of this: the control plane's
	// schema is fixed and holds step results only.
	upload, err := json.Marshal(r.ForUpload())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"summary", "checked_objects", "unopened", "locker"} {
		if bytes.Contains(upload, []byte(`"`+forbidden+`"`)) {
			t.Fatalf("the upload form grew a %q field: %s", forbidden, upload)
		}
	}
}

func memoryWithProbe(t *testing.T) *memory.Store {
	t.Helper()
	store := memory.New()
	put(t, store, "reference/probe.txt", []byte("probe"))
	return store
}
