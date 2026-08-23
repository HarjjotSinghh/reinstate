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

// refusing is a reference locker that behaves like R2: access denied.
type refusing struct{ backend.Backend }

func (refusing) List(context.Context, string) ([]backend.ObjectMeta, error) {
	return nil, backend.ErrUnauthorized
}

func (refusing) Get(context.Context, string) (io.ReadCloser, backend.ObjectMeta, error) {
	return nil, backend.ObjectMeta{}, backend.ErrUnauthorized
}

func TestRunPassesAndStripsDetailForUpload(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "team/a")
	ref := &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Region: "auto", Key: "reference/probe.txt"}
	r := Run(context.Background(), Options{
		Backend: store, Prefix: "team/a", Keys: keys, Storage: StorageHop, Reference: ref,
		OpenReference: func(context.Context, hop.Reference) (backend.Backend, error) { return refusing{}, nil },
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
	// The uploaded form reveals exactly these step summaries and nothing
	// about the index: no agent names, counts, revision, or sizes.
	u := r.ForUpload()
	wantObserved := []string{
		"2 object(s): manifest.age (the encrypted index); 1 snapshot(s) under snapshots/ named by opaque ids.",
		"manifest.age (" + strconv.Itoa(len(get(t, store, "team/a/manifest.age"))) + " bytes): begins with the age v1 header (recipient X25519 (root key)); no plaintext field name appears anywhere in the body. " +
			"snapshots/snap-1.age (" + strconv.Itoa(len(get(t, store, "team/a/snapshots/snap-1.age"))) + " bytes): begins with the age v1 header (recipient X25519 (root key)); no plaintext field name appears anywhere in the body.",
		"manifest.age decrypted into a schema v1 index. snapshots/snap-1.age decrypted into a snapshot envelope whose payload sha256 matches the envelope.",
		"Listing the reference locker was refused as unauthorized. Reading the probe object was refused as unauthorized.",
	}
	for i, want := range wantObserved {
		if u.Steps[i].Observed != want {
			t.Fatalf("upload step %d observed\n got %q\nwant %q", i+1, u.Steps[i].Observed, want)
		}
	}
	raw, _ := json.Marshal(u)
	for _, s := range []string{"sess-1", "local/p", "detail", "lk-ref", "s3.example", "claude", "session", "revision"} {
		if bytes.Contains(raw, []byte(s)) {
			t.Fatalf("upload carries %q: %s", s, raw)
		}
	}
	if !r.IsolationChecked() || !strings.Contains(r.Summary(), "refused by a bucket that is not its own") {
		t.Fatalf("summary %q", r.Summary())
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
			o.OpenReference = func(context.Context, hop.Reference) (backend.Backend, error) { return leaky, nil }
		}, StepIsolation, "SUCCEEDED"},
		{"reference unknown error", func(o *Options, _ *memory.Store) {
			o.Reference, o.ReferenceErr = nil, errors.New("control plane down")
		}, StepIsolation, "control plane down"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := lockerWith(t, keys, "")
			o := Options{Backend: store, Keys: keys, Storage: StorageHop,
				Reference:     &hop.Reference{Endpoint: "e", Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: func(context.Context, hop.Reference) (backend.Backend, error) { return refusing{}, nil }}
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
