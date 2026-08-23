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

func openRef(b backend.Backend, akid string) func(context.Context, hop.Reference) (backend.Backend, string, error) {
	return func(context.Context, hop.Reference) (backend.Backend, string, error) { return b, akid, nil }
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
	if sum := r.Summary(); !strings.Contains(sum, "The objects checked (the index and the newest snapshot) are ciphertext this device can open. No other object is in the locker.") ||
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
			o.OpenReference = openRef(leaky, "AKIAHOP1")
		}, StepIsolation, "SUCCEEDED"},
		{"credential rotated between steps", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(denied, "AKIAHOP2")
		}, StepIsolation, "changed between step 1 (AKIAHOP1) and this step (AKIAHOP2)"},
		{"reference refused for another reason", func(o *Options, _ *memory.Store) {
			o.OpenReference = openRef(refusing{err: errors.New("dial tcp: connection refused")}, "AKIAHOP1")
		}, StepIsolation, "neither succeeded nor was refused as access denied"},
		{"reference unknown error", func(o *Options, _ *memory.Store) {
			o.Reference, o.ReferenceErr = nil, errors.New("control plane down")
		}, StepIsolation, "control plane down"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: StorageHop,
				Reference:     &hop.Reference{Endpoint: "e", Bucket: "lk-ref", Key: "reference/probe.txt"},
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
			!strings.Contains(human.String(), "was not checked (no reference locker)") ||
			strings.Contains(human.String(), "refused by a bucket") {
			t.Fatalf("summary claims more than observed:\n%s", human.String())
		}
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
				Reference:     &hop.Reference{Endpoint: "e", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(refusing{err: &backend.Refusal{Code: code, Credential: true}}, "AKIAHOP1"),
				CredentialID:  credential("AKIAHOP1")})
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
				Reference:     &hop.Reference{Endpoint: "e", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openRef(refusing{err: &backend.Refusal{Code: code}}, "AKIAHOP1"),
				CredentialID:  credential("AKIAHOP1")})
			if r.Outcome != Pass || !r.IsolationChecked() {
				t.Fatalf("%s did not pass isolation: %+v", code, r.Steps[3])
			}
		})
	}
}
