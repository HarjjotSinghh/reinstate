package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
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

// deniedOn is what a bucket-scoped credential sees on the wire at host: two
// signed 403s, one per request the step makes. Every test that expects the
// isolation step to pass has to state this, because the step is pinned to
// the response and not to the endpoint strings alone.
func deniedOn(host string) []Exchange {
	return []Exchange{
		{Host: host, Status: 403, ErrorCode: "AccessDenied"},
		{Host: host, Status: 403, ErrorCode: "AccessDenied"},
	}
}

func openRef(b backend.Backend, akid string, seen ...Exchange) func(context.Context, hop.Reference) (Probe, error) {
	return func(_ context.Context, ref hop.Reference) (Probe, error) {
		record := seen
		if record == nil {
			record = deniedOn(endpointHost(ref.Endpoint))
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
	if !r.IsolationChecked() || !strings.Contains(r.Summary(), "refused by a bucket that is not its own") {
		t.Fatalf("summary %q", r.Summary())
	}
	// The summary claims only what was fetched and says nothing about
	// which device sealed the objects.
	if sum := r.Summary(); !strings.Contains(sum, "The objects checked (the index and the newest snapshot in the index) are ciphertext this device can open. No other object is in the locker.") ||
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
		{"wrong key", func(o *Options, _ *memory.Store) { o.Keys = other }, StepDecrypt, "did not decrypt with this device's key"},
		{"no key", func(o *Options, _ *memory.Store) { o.Keys = nil }, StepDecrypt, "No key is available"},
		{"nothing pushed", func(_ *Options, s *memory.Store) {
			_ = s.Delete(context.Background(), "manifest.age")
		}, StepList, "nothing has been pushed"},
		{"reference reachable", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(leaky, "AKIAHOP1",
				Exchange{Host: "s3.example", Status: 200},
				Exchange{Host: "s3.example", Status: 200})
		}, StepIsolation, "SUCCEEDED"},
		{"credential rotated between steps", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(denied, "AKIAHOP2")
		}, StepIsolation, "changed between step 1 (AKIAHOP1) and this step (AKIAHOP2)"},
		{"reference refused for another reason", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(refusing{err: errors.New("dial tcp: connection refused")}, "AKIAHOP1",
				Exchange{Host: "s3.example"}, Exchange{Host: "s3.example"})
		}, StepIsolation, "neither succeeded nor was refused as access denied"},
		{"reference unknown error", func(o *Options, _ *memory.Store) {
			o.Reference, o.ReferenceErr = nil, errors.New("control plane down")
		}, StepIsolation, "control plane down"},
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
// bucket scope when it lives at the endpoint step 1 actually listed;
// scheme and trailing-slash differences are the same endpoint.
func TestIsolationEndpointMustMatchStepOne(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name     string
		locker   string
		ref      string
		status   Status
		observed string
	}{
		{"same endpoint modulo scheme and slash", "http://s3.example/", "https://s3.example", Pass, "refused as access denied"},
		{"different host", "https://s3.example", "https://always-403.example", Fail, "pointed this step at https://always-403.example, but step 1 listed this account's locker at https://s3.example"},
		{"same host, different port", "https://s3.example:9000", "https://s3.example", Fail, "pointed this step at https://s3.example, but step 1 listed this account's locker at https://s3.example:9000"},
		{"scheme case only", "HTTPS://S3.example", "https://s3.example", Pass, "refused as access denied"},
		// Without the locker's own endpoint there is nothing to pin the
		// reference against, and an unpinnable refusal decides nothing: it
		// is not-applicable, never a pass.
		{"unknown locker endpoint", "", "https://always-403.example", NotApplicable, "the storage endpoint this account's locker was listed at is not known here"},
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
	sum := r.Summary()
	if !strings.Contains(sum, "The object checked (the index) is ciphertext this device can open.") {
		t.Fatalf("summary does not name only the index: %q", sum)
	}
	if strings.Contains(sum, "newest snapshot") {
		t.Fatalf("summary names a snapshot that was never fetched: %q", sum)
	}
	if got := r.CheckedObjects(); got != "the index" {
		t.Fatalf("CheckedObjects() = %q", got)
	}
}

// TestIsolationFailsWhenTheCredentialItselfIsRejected: a refusal that says
// the credential is dead (unknown key id, bad signature, expired token) is
// refused by every bucket, so it proves nothing about scope and must not
// pass the isolation step. Only AccessDenied (or a bodiless 403) does.
func TestIsolationFailsWhenTheCredentialItselfIsRejected(t *testing.T) {
	keys := rootKeys(t)
	for _, code := range []string{"InvalidAccessKeyId", "SignatureDoesNotMatch", "ExpiredToken", "ExpiredTokenException", "InvalidToken", "TokenRefreshRequired"} {
		t.Run(code, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				Locker:    LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
				Reference: &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(refusing{err: &backend.Refusal{Code: code, Credential: true}}, "AKIAHOP1",
					Exchange{Host: "s3.example", Status: 403, ErrorCode: code},
					Exchange{Host: "s3.example", Status: 403, ErrorCode: code}),
				CredentialID: credential("AKIAHOP1")})
			step := r.Steps[3]
			if r.Outcome != Fail || step.Status != Fail || r.IsolationChecked() {
				t.Fatalf("%s passed isolation: %+v", code, step)
			}
			for _, want := range []string{"the credential itself was rejected (", code, "so nothing about bucket scope was shown"} {
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
					Exchange{Host: "s3.example", Status: 403, ErrorCode: code},
					Exchange{Host: "s3.example", Status: 403, ErrorCode: code}),
				CredentialID: credential("AKIAHOP1")})
			if r.Outcome != Pass || !r.IsolationChecked() {
				t.Fatalf("%s did not pass isolation: %+v", code, r.Steps[3])
			}
		})
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
	}{
		{"the listing was refused", func() *memory.Store { return nil }},
		{"nothing has been pushed yet", func() *memory.Store { return memory.New() }},
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
			if r.Steps[0].Status != Fail {
				t.Fatalf("step 1 %+v; this test needs it to fail", r.Steps[0])
			}
			step := r.Steps[3]
			if step.Status != NotApplicable || r.IsolationChecked() {
				t.Fatalf("isolation claimed on a failed step 1: %+v", step)
			}
			if !strings.Contains(step.Observed, "step 1 did not list this account's locker") {
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
	sum := r.Summary()
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
			if got := r.CheckedObjects(); got != "the index and one snapshot" {
				t.Fatalf("CheckedObjects() = %q; a snapshot chosen without the index must not be called the newest", got)
			}
			if strings.Contains(r.Summary(), "newest") {
				t.Fatalf("summary claims recency nothing observed: %q", r.Summary())
			}
		})
	}
}
