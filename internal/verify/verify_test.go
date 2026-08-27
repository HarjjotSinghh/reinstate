package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func put(t *testing.T, b backend.Backend, key string, body []byte) {
	t.Helper()
	if _, err := b.Put(context.Background(), key, bytes.NewReader(body), int64(len(body)), backend.PutOptions{}); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, b backend.Backend, key string) []byte {
	t.Helper()
	rc, _, err := b.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func seal(t *testing.T, keys crypto.KeyProvider, plain []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := crypto.Seal(bytes.NewReader(plain), &out, keys); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// lockerWith fills a store with a sealed manifest and one snapshot under
// prefix, the way a push leaves them.
func lockerWith(t *testing.T, keys crypto.KeyProvider, prefix string) *memory.Store {
	t.Helper()
	store := memory.New()
	payload := []byte("synthetic session payload\n")
	sum := crypto.SHA256Hex(payload)
	env := schema.Envelope{SchemaVersion: 1, Kind: "snapshot", SnapshotID: "snap-1", Agent: "claude", SessionID: "sess-1", ProjectID: "local/p",
		CreatedAt: "2026-08-23T12:00:00Z", SourcePlatform: "darwin-arm64",
		Files: []schema.EnvelopeFile{{Path: "s.jsonl", Mode: 0o600, Size: int64(len(payload)), SHA256: sum}}}
	meta, _ := json.Marshal(env)
	man := schema.NewManifest("snap-1")
	man.Sessions["claude:sess-1"] = schema.ManifestSession{Agent: "claude", SessionID: "sess-1", SnapshotID: "snap-1", ProjectID: "local/p", UpdatedAt: "2026-08-23T12:00:00Z"}
	manRaw, _ := json.Marshal(man)
	p := strings.Trim(prefix, "/")
	if p != "" {
		p += "/"
	}
	put(t, store, p+"manifest.age", seal(t, keys, manRaw))
	put(t, store, p+"snapshots/snap-1.age", seal(t, keys, append(append(meta, '\n'), payload...)))
	put(t, store, "elsewhere/manifest.age", []byte("not ours"))
	return store
}

func rootKeys(t *testing.T) *crypto.RootKeyProvider {
	t.Helper()
	rk, err := crypto.NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := crypto.NewRootKeyProvider(rk)
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

// refusing is a reference locker that answers every request with the
// given refusal, the way R2 answers a credential scoped to another bucket
// (AccessDenied) or a dead credential (InvalidAccessKeyId and friends).
type refusing struct {
	backend.Backend
	err error
}

func (r refusing) List(context.Context, string) ([]backend.ObjectMeta, error) {
	return nil, r.err
}

func (r refusing) Get(context.Context, string) (io.ReadCloser, backend.ObjectMeta, error) {
	return nil, backend.ObjectMeta{}, r.err
}

// denied is the reference locker as R2 presents it to a bucket-scoped key.
var denied = refusing{err: &backend.Refusal{Code: "AccessDenied"}}

// deniedOn is what a bucket-scoped credential sees on the wire at
// endpoint: two signed 403s, one per request the step makes. Every test
// that expects the isolation step to pass has to state this, because the
// step is pinned to the response and not to the endpoint strings alone.
// The scheme is part of it — a probe that reached the right host over
// http went out in the clear — so the whole endpoint is given here, not
// the host alone.
func deniedOn(endpoint string) []Exchange {
	scheme, host := splitEndpoint(endpoint)
	return []Exchange{
		{Scheme: scheme, Host: host, Status: 403, ErrorCode: "AccessDenied"},
		{Scheme: scheme, Host: host, Status: 403, ErrorCode: "AccessDenied"},
	}
}

// splitEndpoint is what the probe transport records for a request to
// endpoint: its scheme and host, as the URL carried them.
func splitEndpoint(endpoint string) (scheme, host string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", ""
	}
	return strings.ToLower(u.Scheme), strings.ToLower(u.Host)
}

func openRef(b backend.Backend, akid string, seen ...Exchange) func(context.Context, hop.Reference) (Probe, error) {
	return func(_ context.Context, ref hop.Reference) (Probe, error) {
		record := seen
		if record == nil {
			record = deniedOn(ref.Endpoint)
		}
		return Probe{Backend: b, AccessKeyID: akid, Exchanges: func() []Exchange { return record }}, nil
	}
}

func credential(akid string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return akid, nil }
}

func TestRunPassesAndStripsDetailForUpload(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "team/a")
	ref := &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Region: "auto", Key: "reference/probe.txt"}
	r := Run(context.Background(), Options{
		Backend: store, Prefix: "team/a", Keys: keys, Storage: StorageHop, Reference: ref,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1"),
		ClientVersion: "rein test", Now: func() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) },
	})
	if r.Outcome != Pass || len(r.Steps) != 4 {
		t.Fatalf("report %+v", r)
	}
	for i, id := range []string{StepList, StepCiphertext, StepDecrypt, StepIsolation} {
		if r.Steps[i].ID != id || r.Steps[i].Status != Pass {
			t.Fatalf("step %d %+v", i, r.Steps[i])
		}
	}
	if got := r.Steps[0].Observed; !strings.Contains(got, "2 object(s)") || strings.Contains(got, "other object") {
		t.Fatalf("listing crossed the prefix: %q", got)
	}
	detail := strings.Join(r.Steps[2].Detail, "\n")
	if !strings.Contains(detail, "1 session(s) (claude 1)") || !strings.Contains(detail, "session sess-1") {
		t.Fatalf("decrypt step detail %+v", r.Steps[2])
	}
	// Steps 1 and 4 record the same access key id, so the report shows the
	// credential the locker accepted is the one the reference refused.
	for _, i := range []int{0, 3} {
		if !strings.Contains(strings.Join(r.Steps[i].Detail, "\n"), "signed with access key id AKIAHOP1") {
			t.Fatalf("step %d detail lacks the access key id: %+v", i+1, r.Steps[i].Detail)
		}
	}
	// The uploaded form reveals exactly these step summaries and nothing
	// about the index: no agent names, counts, revision, or sizes.
	u := r.ForUpload()
	wantObserved := []string{
		"2 object(s): manifest.age (the encrypted index); 1 snapshot(s) under snapshots/ named by opaque ids.",
		"manifest.age (" + strconv.Itoa(len(get(t, store, "team/a/manifest.age"))) + " bytes): begins with the age v1 header (recipient X25519 (root key)); no plaintext field name appears anywhere in the body. " +
			"snapshots/snap-1.age (" + strconv.Itoa(len(get(t, store, "team/a/snapshots/snap-1.age"))) + " bytes): begins with the age v1 header (recipient X25519 (root key)); no plaintext field name appears anywhere in the body.",
		"manifest.age decrypted into a schema v1 index. snapshots/snap-1.age decrypted into a snapshot envelope whose payload sha256 matches the envelope.",
		"Listing the reference locker was refused as access denied. Reading the probe object was refused as access denied.",
	}
	for i, want := range wantObserved {
		if u.Steps[i].Observed != want {
			t.Fatalf("upload step %d observed\n got %q\nwant %q", i+1, u.Steps[i].Observed, want)
		}
	}
	raw, _ := json.Marshal(u)
	for _, s := range []string{"sess-1", "local/p", "detail", "lk-ref", "s3.example", "claude", "session", "revision", "AKIAHOP1"} {
		if bytes.Contains(raw, []byte(s)) {
			t.Fatalf("upload carries %q: %s", s, raw)
		}
	}
	if !r.IsolationChecked() || !strings.Contains(r.Summary, "refused by a bucket that is not its own") {
		t.Fatalf("summary %q", r.Summary)
	}
	// The summary claims only what was fetched and says nothing about
	// which device sealed the objects.
	if sum := r.Summary; !strings.Contains(sum, "The objects checked (the index and the newest snapshot in the index) are ciphertext this device can open. No other object is in the locker.") ||
		strings.Contains(sum, "Everything in the locker") || strings.Contains(sum, "sealed on this device") {
		t.Fatalf("summary over-claims: %q", sum)
	}
	var human bytes.Buffer
	r.WriteHuman(&human)
	if !strings.Contains(human.String(), "OUTCOME: PASS") || strings.Count(human.String(), "Result:         PASS") != 4 {
		t.Fatalf("human report:\n%s", human.String())
	}
}

func TestRunFailures(t *testing.T) {
	keys := rootKeys(t)
	other := rootKeys(t)
	leaky := memory.New()
	put(t, leaky, "reference/probe.txt", []byte("probe"))
	tests := []struct {
		name   string
		mutate func(o *Options, store *memory.Store)
		step   string
		want   string
	}{
		{"plaintext manifest", func(_ *Options, s *memory.Store) {
			put(t, s, "manifest.age", []byte(`{"schema_version":1,"sessions":{}}`))
		}, StepCiphertext, "does NOT begin with the age v1 header"},
		{"age header then plaintext", func(_ *Options, s *memory.Store) {
			put(t, s, "manifest.age", []byte("age-encryption.org/v1\n-> X25519 x\n{\"sessions\":{}}"))
		}, StepCiphertext, `plaintext field "sessions" appears`},
		{"wrong key", func(o *Options, _ *memory.Store) { o.Keys = other }, StepDecrypt, "the key held on this device is not one of the object's recipients"},
		{"no key", func(o *Options, _ *memory.Store) { o.Keys = nil }, StepDecrypt, "No key is available"},
		{"listing refused", func(o *Options, _ *memory.Store) {
			o.Backend = refusing{err: &backend.Refusal{Code: "AccessDenied"}}
		}, StepList, "the storage endpoint recognised the credential and refused the request anyway (AccessDenied)"},
		{"reference reachable", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(leaky, "AKIAHOP1",
				Exchange{Scheme: "https", Host: "s3.example", Status: 200},
				Exchange{Scheme: "https", Host: "s3.example", Status: 200})
		}, StepIsolation, "SUCCEEDED"},
		// A control plane could point step 4 at any host it likes — every
		// bucket answers a foreign credential with 403 — so a reference
		// locker at a different endpoint than the one step 1 listed fails
		// the step even though the probe is refused as access denied.
		{"reference at a different endpoint", func(o *Options, _ *memory.Store) {
			o.Reference = &hop.Reference{Endpoint: "https://always-403.example", Bucket: "lk-ref", Key: "reference/probe.txt"}
		}, StepIsolation, "pointed this step at https://always-403.example, but step 1 listed this account's locker at https://s3.example"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")}
			tc.mutate(&o, store)
			r := Run(context.Background(), o)
			if r.Outcome != Fail {
				t.Fatalf("outcome %s", r.Outcome)
			}
			var found bool
			for _, s := range r.Steps {
				if s.ID == tc.step {
					found = true
					if s.Status != Fail || !strings.Contains(s.Observed, tc.want) {
						t.Fatalf("step %+v; want fail containing %q", s, tc.want)
					}
				}
			}
			if !found {
				t.Fatal("step missing")
			}
		})
	}
}

// TestIsolationCouldNotRunIsNotAFailure: everything that stops step 4
// without showing anything about this account's credentials is a check
// that could not run. None of it is the account's doing — an operator
// misconfiguration, a reference bucket that has been deleted, an endpoint
// that answered 500, a dropped connection, an hourly credential that died
// mid-command — and reporting any of it as `OUTCOME: FAIL` tells a
// customer their locker failed a security check when it did not. The
// report must say so, exit 0, and claim nothing about bucket scope.
func TestIsolationCouldNotRunIsNotAFailure(t *testing.T) {
	keys := rootKeys(t)
	// unusable is a reference locker the step must never be handed. It is
	// wide open, so a step that probed it anyway would report the loudest
	// possible false alarm instead of quietly passing this test.
	unusable := memory.New()
	put(t, unusable, "reference/probe.txt", []byte("probe"))
	tests := []struct {
		name   string
		mutate func(o *Options)
		want   string
	}{
		{
			// The integration case: the control plane answers 500 on
			// GET /v1/verify/reference. Before this every account's
			// trust-establishing command reported a failed security check
			// because of one operator-side fault.
			name: "the control plane answered an error",
			mutate: func(o *Options) {
				o.Reference, o.ReferenceErr = nil, &hop.Error{Status: 500, Code: "internal", Message: "internal error"}
			},
			want: "Could not run: the control plane did not say where its reference locker is",
		},
		{
			name: "the control plane answered something that is not a reference locker",
			mutate: func(o *Options) {
				o.Reference, o.ReferenceErr = nil, errors.New("control plane returned an incomplete reference locker")
			},
			want: "control plane returned an incomplete reference locker",
		},
		{
			name: "the reference bucket has been deleted",
			mutate: func(o *Options) {
				o.OpenReference = openRef(refusing{err: backend.ErrNotFound}, "AKIAHOP1",
					Exchange{Scheme: "https", Host: "s3.example", Status: 404},
					Exchange{Scheme: "https", Host: "s3.example", Status: 404})
			},
			want: "Could not run: listing the reference locker neither succeeded nor was refused as access denied",
		},
		{
			name: "the connection dropped",
			mutate: func(o *Options) {
				o.OpenReference = openRef(refusing{err: errors.New("dial tcp: connection refused")}, "AKIAHOP1",
					Exchange{Scheme: "https", Host: "s3.example"}, Exchange{Scheme: "https", Host: "s3.example"})
			},
			want: "Could not run: listing the reference locker neither succeeded nor was refused as access denied",
		},
		{
			name: "the locker credential died between step 1 and step 4",
			mutate: func(o *Options) {
				o.OpenReference = openRef(refusing{err: &backend.Refusal{Code: "ExpiredToken", Credential: true}}, "AKIAHOP1",
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: "ExpiredToken"},
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: "ExpiredToken"})
			},
			want: "Could not run: listing the reference locker failed because the credential itself was rejected",
		},
		{
			name:   "the locker credential was rotated between step 1 and step 4",
			mutate: func(o *Options) { o.OpenReference = openRef(denied, "AKIAHOP2") },
			want:   "Could not run: the locker credential changed between step 1 (AKIAHOP1) and this step (AKIAHOP2)",
		},
		{
			// A reference row naming the account's own bucket would be
			// answered by that bucket, and the answer read as credentials
			// reaching a bucket that is not their own — exactly backwards.
			name: "the reference locker is this account's own bucket",
			mutate: func(o *Options) {
				o.Reference = &hop.Reference{Endpoint: "https://s3.example", Bucket: "LK-1", Key: "reference/probe.txt"}
				o.OpenReference = openRef(unusable, "AKIAHOP1")
			},
			want: "Could not run: the control plane named this account's own bucket (lk-1) as its reference locker",
		},
		{
			name:   "the bucket step 1 listed is not known here",
			mutate: func(o *Options) { o.Locker.Bucket = "" },
			want:   "Could not run: the bucket step 1 listed is not known here",
		},
		{
			name:   "no way to open the reference locker",
			mutate: func(o *Options) { o.OpenReference = nil },
			want:   "Could not run: no way to open the reference locker was configured on this device",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")}
			tc.mutate(&o)
			r := Run(context.Background(), o)
			step := r.Steps[3]
			if step.ID != StepIsolation || step.Status != NotApplicable || !strings.Contains(step.Observed, tc.want) {
				t.Fatalf("isolation step %+v; want not-applicable containing %q", step, tc.want)
			}
			// The first three steps ran, so the report passes and exits 0;
			// what it must not do is claim the isolation it never observed.
			if r.Outcome != Pass || r.Failed() || r.IsolationChecked() {
				t.Fatalf("outcome %s failed=%t isolation=%t", r.Outcome, r.Failed(), r.IsolationChecked())
			}
			if !strings.Contains(r.Summary, "was not checked (step 4 above says why)") || strings.Contains(r.Summary, "refused by a bucket") {
				t.Fatalf("summary %q", r.Summary)
			}
			// The same distinction has to survive into the document a
			// console or a script reads.
			if u := r.ForUpload(); u.Outcome != Pass || u.Steps[3].Status != NotApplicable {
				t.Fatalf("upload %+v", u.Steps[3])
			}
		})
	}
}

// TestIsolationRefusesToProbeTheAccountsOwnBucket: the step must not send
// the probe at all when the control plane names this account's own bucket.
// Sending it would reach the bucket these credentials are for, and the
// success would be reported as credentials reaching a bucket that is not
// their own.
func TestIsolationRefusesToProbeTheAccountsOwnBucket(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	opened := 0
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:    LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference: &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-1", Key: "reference/probe.txt"},
		OpenReference: func(ctx context.Context, ref hop.Reference) (Probe, error) {
			opened++
			return openRef(store, "AKIAHOP1")(ctx, ref)
		},
		CredentialID: credential("AKIAHOP1")})
	if opened != 0 {
		t.Fatalf("the probe was sent to this account's own bucket %d time(s)", opened)
	}
	if step := r.Steps[3]; step.Status != NotApplicable || !strings.Contains(step.Observed, "named this account's own bucket") {
		t.Fatalf("isolation step %+v", step)
	}
	if r.Failed() || strings.Contains(r.Summary, "not its own") {
		t.Fatalf("the account's own bucket was reported as a foreign one: %q", r.Summary)
	}
}

func TestRunNotApplicable(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	byo := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageBYO})
	if byo.Outcome != Pass || byo.Steps[3].Status != NotApplicable || !strings.Contains(byo.Steps[3].Observed, "BYO storage") {
		t.Fatalf("byo %+v", byo.Steps[3])
	}
	none := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop, ReferenceErr: hop.ErrNoReference})
	if none.Outcome != Pass || none.Steps[3].Status != NotApplicable || !strings.Contains(none.Steps[3].Observed, "no reference locker") {
		t.Fatalf("no reference %+v", none.Steps[3])
	}
	// A passing report whose isolation step did not run must not claim
	// isolation in its summary.
	for _, r := range []*Report{byo, none} {
		var human bytes.Buffer
		r.WriteHuman(&human)
		if r.IsolationChecked() || !strings.Contains(human.String(), "OUTCOME: PASS. ") ||
			!strings.Contains(human.String(), "was not checked (step 4 above says why)") ||
			strings.Contains(human.String(), "refused by a bucket") {
			t.Fatalf("summary claims more than observed:\n%s", human.String())
		}
	}
}

// TestIsolationEndpointMustMatchStepOne: the reference locker only proves
// bucket scope when it lives at the endpoint step 1 actually listed —
// scheme, host and port. Case, a trailing slash, a trailing dot on the
// host and an implicit default port are not differences; the scheme is.
//
// The scheme case is the one this table was written for. `http://<the same
// host>` used to satisfy the pin, and the step then signed a request to it
// with the live secret key and session token this device pushes with. The
// probe is never sent over an unencrypted connection, whatever the pin
// says.
func TestIsolationEndpointMustMatchStepOne(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name     string
		locker   string
		ref      string
		status   Status
		observed string
	}{
		{"http where step 1 listed https", "https://s3.example", "http://s3.example", Fail,
			"pointed this step at http://s3.example, but step 1 listed this account's locker at https://s3.example"},
		{"https where step 1 listed http", "http://s3.example/", "https://s3.example", Fail,
			"pointed this step at https://s3.example, but step 1 listed this account's locker at http://s3.example/"},
		{"plaintext on both sides is refused outright", "http://s3.example", "http://s3.example", Fail,
			"which is plaintext http. This step signs its request with the same temporary credentials this device pushes with"},
		{"scheme case only", "HTTPS://S3.example", "https://s3.example", Pass, "refused as access denied"},
		{"trailing slash and host case only", "https://S3.example/", "https://s3.example", Pass, "refused as access denied"},
		{"an explicit default port is the same endpoint", "https://s3.example:443", "https://s3.example", Pass, "refused as access denied"},
		{"a trailing dot on the host is the same endpoint", "https://s3.example.", "https://s3.example", Pass, "refused as access denied"},
		{"an IPv6 literal is the same endpoint", "https://[2606:4700::1111]:443/", "https://[2606:4700::1111]", Pass, "refused as access denied"},
		{"different host", "https://s3.example", "https://always-403.example", Fail, "pointed this step at https://always-403.example, but step 1 listed this account's locker at https://s3.example"},
		{"same host, different port", "https://s3.example:9000", "https://s3.example", Fail, "pointed this step at https://s3.example, but step 1 listed this account's locker at https://s3.example:9000"},
		{"a reference endpoint that is not a URL", "https://s3.example", "s3.example", Fail, "pointed this step at s3.example, but step 1 listed this account's locker at https://s3.example"},
		// Without the locker's own endpoint there is nothing to pin the
		// reference against, and an unpinnable refusal decides nothing: it
		// is not-applicable, never a pass.
		{"unknown locker endpoint", "", "https://always-403.example", NotApplicable, "the storage endpoint this account's locker was listed at is not known here"},
		{"unparseable locker endpoint", "s3.example:9000", "https://always-403.example", NotApplicable, `("s3.example:9000" is not an absolute http or https URL)`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: tc.locker, Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: tc.ref, Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
			step := r.Steps[3]
			if step.ID != StepIsolation || step.Status != tc.status || !strings.Contains(step.Observed, tc.observed) {
				t.Fatalf("isolation step %+v; want %s containing %q", step, tc.status, tc.observed)
			}
			// The reference endpoint is on record either way, so a reader
			// can see where the step was pointed.
			if !strings.Contains(strings.Join(step.Detail, "\n"), "at "+tc.ref) {
				t.Fatalf("detail does not record the endpoint: %+v", step.Detail)
			}
		})
	}
}

// TestIsolationNeverSendsTheCredentialOverPlaintext is the fix's whole
// point: the pin compares two strings the control plane supplied, so a
// plaintext endpoint that satisfies it must still not receive this
// account's live secret key and session token. Nothing is sent, and the
// step says why.
func TestIsolationNeverSendsTheCredentialOverPlaintext(t *testing.T) {
	keys := rootKeys(t)
	for _, endpoint := range []string{"http://s3.example", "HTTP://S3.example/", "http://s3.example:80", "http://198.51.100.7:9000"} {
		t.Run(endpoint, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			opened := 0
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:    LockerInfo{Endpoint: endpoint, Bucket: "lk-1"},
				Reference: &hop.Reference{Endpoint: endpoint, Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: func(ctx context.Context, ref hop.Reference) (Probe, error) {
					opened++
					return openRef(denied, "AKIAHOP1")(ctx, ref)
				},
				CredentialID: credential("AKIAHOP1")})
			if opened != 0 {
				t.Fatalf("the credential was offered to a plaintext endpoint %d time(s)", opened)
			}
			step := r.Steps[3]
			if step.Status != Fail || !strings.Contains(step.Observed, "which is plaintext http") || r.IsolationChecked() {
				t.Fatalf("isolation step %+v", step)
			}
		})
	}
	// Loopback is the exception, and only loopback: the request never
	// reaches a network, which is what the fakes in these tests and a
	// locally run control plane rely on.
	for _, endpoint := range []string{"http://127.0.0.1:8080", "http://localhost:8080", "http://[::1]:8080"} {
		t.Run(endpoint, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: endpoint, Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: endpoint, Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
			if step := r.Steps[3]; step.Status != Pass {
				t.Fatalf("a loopback probe was refused: %+v", step)
			}
		})
	}
}

// TestSummaryClaimsOnlyWhatWasFetched: a manifest-only locker fetched only
// the index, and the outcome sentence must not name a snapshot it never
// read.
func TestSummaryClaimsOnlyWhatWasFetched(t *testing.T) {
	keys := rootKeys(t)
	store := memory.New()
	man := schema.NewManifest("r1")
	manRaw, _ := json.Marshal(man)
	put(t, store, "manifest.age", seal(t, keys, manRaw))
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	if !r.Passed() {
		t.Fatalf("report %+v", r)
	}
	sum := r.Summary
	if !strings.Contains(sum, "The object checked (the index) is ciphertext this device can open.") {
		t.Fatalf("summary does not name only the index: %q", sum)
	}
	if strings.Contains(sum, "newest snapshot") {
		t.Fatalf("summary names a snapshot that was never fetched: %q", sum)
	}
	if got := r.CheckedPhrase(); got != "the index" {
		t.Fatalf("CheckedObjects() = %q", got)
	}
}

// TestIsolationDoesNotPassWhenTheCredentialItselfIsRejected: a refusal
// that says the credential is dead (unknown key id, bad signature, expired
// token) is what every bucket answers, so it proves nothing about scope and
// must not pass the isolation step. It is not a failed check either — a
// locker credential lasts an hour, and one that ran out between step 1 and
// step 4 says nothing about the account — so it is a check that could not
// run. Only AccessDenied (or a bodiless 403) passes.
func TestIsolationDoesNotPassWhenTheCredentialItselfIsRejected(t *testing.T) {
	keys := rootKeys(t)
	for _, code := range []string{"InvalidAccessKeyId", "SignatureDoesNotMatch", "ExpiredToken", "ExpiredTokenException", "InvalidToken", "TokenRefreshRequired"} {
		t.Run(code, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:    LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference: &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(refusing{err: &backend.Refusal{Code: code, Credential: true}}, "AKIAHOP1",
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: code},
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: code}),
				CredentialID: credential("AKIAHOP1")})
			step := r.Steps[3]
			if r.Failed() || step.Status != NotApplicable || r.IsolationChecked() {
				t.Fatalf("%s: outcome %s, isolation %+v", code, r.Outcome, step)
			}
			for _, want := range []string{"Could not run: ", "the credential itself was rejected — ", code, "a credential no bucket accepts is refused everywhere, so nothing about bucket scope was shown"} {
				if strings.Count(step.Observed, want) < 1 {
					t.Fatalf("%s: observed %q lacks %q", code, step.Observed, want)
				}
			}
		})
	}
	for _, code := range []string{"AccessDenied", "Forbidden"} {
		t.Run(code, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:    LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference: &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(refusing{err: &backend.Refusal{Code: code}}, "AKIAHOP1",
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: code},
					Exchange{Scheme: "https", Host: "s3.example", Status: 403, ErrorCode: code}),
				CredentialID: credential("AKIAHOP1")})
			if r.Outcome != Pass || !r.IsolationChecked() {
				t.Fatalf("%s did not pass isolation: %+v", code, r.Steps[3])
			}
		})
	}
}

// reading is a backend that records every object fetched through it.
type reading struct {
	backend.Backend
	got *[]string
}

func (r reading) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	*r.got = append(*r.got, key)
	return r.Backend.Get(ctx, key)
}

// TestVerifyNeverOpensTheKeyring pins a claim docs/hop/threat-model.md
// makes in the sentence that matters most on the page: the checks do not
// examine `keyring.v1.json`, so nothing here detects a planted keyring,
// and the report says so by naming the keyring among the objects it
// judged by name only. A step 2 that quietly started fetching it would
// make that paragraph false.
func TestVerifyNeverOpensTheKeyring(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	put(t, store, keyring.ObjectName, []byte(`{"schema_version":1,"profile_id":"p","current_generation":1,"generations":[]}`))
	var fetched []string
	r := Run(context.Background(), Options{Backend: reading{Backend: store, got: &fetched}, Keys: keys, Storage: StorageBYO})
	if !r.Passed() {
		t.Fatalf("report %+v", r)
	}
	for _, key := range fetched {
		if strings.Contains(key, keyring.ObjectName) {
			t.Fatalf("the checks fetched %s; the threat model says they never do", key)
		}
	}
	if !strings.Contains(r.Unopened, "the wrapped keyring") || !strings.Contains(r.Summary, "the wrapped keyring") {
		t.Fatalf("the report does not name the keyring as unopened: %q / %q", r.Unopened, r.Summary)
	}
}

// TestIsolationNeedsStepOneToHavePassed: an always-403 host answers the
// probe with a perfectly well-formed AccessDenied whether or not it has
// ever seen this credential before. The step only means something when
// step 1 showed a locker accepting that same credential, so a step 1 that
// did not pass makes the step not applicable rather than a green tick
// posted to the account console.
func TestIsolationNeedsStepOneToHavePassed(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name  string
		store func() *memory.Store
		step1 Status
		why   string
	}{
		{"the listing was refused", func() *memory.Store { return nil }, Fail, "step 1 did not list this account's locker"},
		{"nothing has been pushed yet", func() *memory.Store { return memory.New() }, NotApplicable, "nothing has been pushed from this profile yet"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b backend.Backend = refusing{err: &backend.Refusal{Code: "AccessDenied"}}
			if store := tc.store(); store != nil {
				b = store
			}
			r := Run(context.Background(), Options{Backend: b, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
			if r.Steps[0].Status != tc.step1 {
				t.Fatalf("step 1 %+v; this test needs it %s", r.Steps[0], tc.step1)
			}
			step := r.Steps[3]
			if step.Status != NotApplicable || r.IsolationChecked() {
				t.Fatalf("isolation claimed without a passing step 1: %+v", step)
			}
			if !strings.Contains(step.Observed, tc.why) {
				t.Fatalf("the step does not say why it did not apply: %q", step.Observed)
			}
		})
	}
}

// TestCiphertextStepFetchesTheSnapshotTheIndexCallsNewest pins which
// snapshot step 2 reads. Snapshot ids are random uuids, so the id that
// sorts last is not the newest object; the index is the only thing that
// knows. The ids here are chosen so that neither the first nor the last in
// sort order is the newest, which is what makes the pick observable.
func TestCiphertextStepFetchesTheSnapshotTheIndexCallsNewest(t *testing.T) {
	keys := rootKeys(t)
	store := memory.New()
	man := schema.NewManifest("r1")
	updated := map[string]string{"a": "2026-08-20T09:00:00Z", "m": "2026-08-23T18:30:00Z", "z": "2026-08-21T11:00:00Z"}
	for _, id := range []string{"a", "m", "z"} {
		payload := []byte("session " + id + "\n")
		env := schema.Envelope{SchemaVersion: 1, Kind: "snapshot", SnapshotID: id, Agent: "claude", SessionID: "sess-" + id, ProjectID: "local/p",
			CreatedAt: updated[id], SourcePlatform: "darwin-arm64",
			Files: []schema.EnvelopeFile{{Path: "s.jsonl", Mode: 0o600, Size: int64(len(payload)), SHA256: crypto.SHA256Hex(payload)}}}
		meta, _ := json.Marshal(env)
		put(t, store, "snapshots/"+id+".age", seal(t, keys, append(append(meta, '\n'), payload...)))
		man.Sessions["claude:sess-"+id] = schema.ManifestSession{Agent: "claude", SessionID: "sess-" + id, SnapshotID: id, ProjectID: "local/p", UpdatedAt: updated[id]}
	}
	manRaw, _ := json.Marshal(man)
	put(t, store, "manifest.age", seal(t, keys, manRaw))

	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	if !r.Passed() {
		t.Fatalf("report %+v", r)
	}
	for _, s := range []Step{r.Steps[1], r.Steps[2]} {
		if !strings.Contains(s.Observed, "snapshots/m.age") {
			t.Fatalf("step %q read a snapshot other than the newest the index names: %q", s.ID, s.Observed)
		}
		for _, other := range []string{"snapshots/a.age", "snapshots/z.age"} {
			if strings.Contains(s.Observed, other) {
				t.Fatalf("step %q read %s as well: %q", s.ID, other, s.Observed)
			}
		}
	}
	// Having chosen by recency, the report may say so — and must count the
	// two it left alone without calling them older than anything.
	sum := r.Summary
	if !strings.Contains(sum, "The objects checked (the index and the newest snapshot in the index) are ciphertext this device can open.") ||
		!strings.Contains(sum, "2 other age-named snapshot(s)") {
		t.Fatalf("summary %q", sum)
	}
}

// TestCiphertextStepClaimsNoRecencyItCannotSee: when the index cannot be
// opened here, or names none of the snapshots in the locker, the snapshot
// step 2 falls back to is just one snapshot — and the report says exactly
// that rather than calling it the newest.
func TestCiphertextStepClaimsNoRecencyItCannotSee(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name   string
		mutate func(store *memory.Store, o *Options)
	}{
		{"the index names no snapshot in the locker", func(store *memory.Store, _ *Options) {
			man := schema.NewManifest("r1")
			man.Sessions["claude:gone"] = schema.ManifestSession{Agent: "claude", SessionID: "gone", SnapshotID: "not-here", UpdatedAt: "2026-08-23T12:00:00Z"}
			raw, _ := json.Marshal(man)
			put(t, store, "manifest.age", seal(t, keys, raw))
		}},
		{"the index does not open with this device's key", func(store *memory.Store, o *Options) {
			put(t, store, "manifest.age", seal(t, rootKeys(t), []byte(`{"schema_version":1}`)))
			o.Keys = keys
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")}
			tc.mutate(store, &o)
			r := Run(context.Background(), o)
			if got := r.CheckedPhrase(); got != "the index and one snapshot" {
				t.Fatalf("CheckedObjects() = %q; a snapshot chosen without the index must not be called the newest", got)
			}
			if strings.Contains(r.Summary, "newest") {
				t.Fatalf("summary claims recency nothing observed: %q", r.Summary)
			}
		})
	}
}
