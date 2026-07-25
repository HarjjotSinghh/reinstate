package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func TestPushPullRoundTripMemory(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session-001.jsonl")
	plain := []byte(`{"type":"user","text":"synthetic fixture only"}`)
	if err := os.WriteFile(session, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	eng := testEngine(&Engine{Backend: store, Passphrase: "test-pass-phrase-32", Platform: "darwin-arm64"})
	ctx := context.Background()
	id, err := eng.PushSession(ctx, PushItem{
		Agent: "claude", SessionID: "session-001", ProjectID: "github.com/example/demo", LocalPath: session,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	rc, _, err := store.Get(ctx, "snapshots/"+id+".age")
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(cipher, plain) {
		t.Fatal("plaintext leaked to remote")
	}
	var trash bytes.Buffer
	if err := crypto.Decrypt(bytes.NewReader(cipher), &trash, "wrong"); err == nil {
		t.Fatal("wrong pass should fail")
	}

	dest := filepath.Join(dir, "restored")
	env, payload, err := eng.PullSession(ctx, PullItem{
		Agent: "claude", SessionID: "session-001", SnapshotID: id, ProjectID: "github.com/example/demo",
	}, dest, false)
	if err != nil {
		t.Fatal(err)
	}
	if env.Agent != "claude" || !bytes.Equal(payload, plain) {
		t.Fatalf("env=%+v payload=%q", env, payload)
	}
	man, err := eng.FetchManifest(ctx)
	if err != nil || man.Sessions[SessionKey("claude", "session-001")].SnapshotID != id {
		t.Fatalf("manifest %+v %v", man, err)
	}
}

func TestConflictMetadataOnly(t *testing.T) {
	home := t.TempDir()
	c := Conflict{
		ID: "c1", Agent: "claude", SessionID: "s1",
		LocalRevision: "l", RemoteRevision: "r", RemoteSnapshot: "snap",
	}
	if err := SaveConflict(home, c); err != nil {
		t.Fatal(err)
	}
	list, err := ListConflicts(home)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %v", list, err)
	}
	b, _ := os.ReadFile(filepath.Join(home, "conflicts", "c1.json"))
	if bytes.Contains(b, []byte("transcript")) {
		t.Fatal("unexpected")
	}
	applied := false
	if err := Resolve(home, "c1", KeepLocal, func(got Conflict, how Resolution) error {
		applied = got.ID == "c1" && how == KeepLocal
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("resolution executor was not called")
	}
}

func TestFailedConflictResolutionPreservesRecord(t *testing.T) {
	home := t.TempDir()
	if err := SaveConflict(home, Conflict{ID: "c1", Agent: "claude", SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("apply failed")
	if err := Resolve(home, "c1", KeepRemote, func(Conflict, Resolution) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("got %v want %v", err, wantErr)
	}
	if _, err := GetConflict(home, "c1"); err != nil {
		t.Fatalf("failed resolution deleted conflict: %v", err)
	}
}

func TestDetectConflict(t *testing.T) {
	if !DetectConflict("a", "b", "base") {
		t.Fatal("expected conflict")
	}
	if DetectConflict("a", "a", "a") {
		t.Fatal("same")
	}
}

func TestAtomicRestorePreservesOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.jsonl")
	if err := fsx.WriteFileAtomic(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.AtomicRestoreFile(path, []byte("new"), true); err == nil {
		t.Fatal("expected fail")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "old" {
		t.Fatalf("got %q", b)
	}
}

func TestPullSessionUsesAtomicWrite(t *testing.T) {
	// PullSession must write via AtomicRestoreFile so a failed write leaves prior content.
	dir := t.TempDir()
	session := filepath.Join(dir, "session-001.jsonl")
	plain := []byte(`{"type":"user","text":"synthetic"}`)
	if err := os.WriteFile(session, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	eng := testEngine(&Engine{Backend: store, Passphrase: "test-pass-phrase-32", Platform: "darwin-arm64"})
	ctx := context.Background()
	id, err := eng.PushSession(ctx, PushItem{
		Agent: "claude", SessionID: "session-001", ProjectID: "p", LocalPath: session,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "restored")
	// seed prior content that must survive a failed atomic restore path check
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	prior := filepath.Join(dest, "session-001.jsonl")
	if err := os.WriteFile(prior, []byte("prior"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, payload, err := eng.PullSession(ctx, PullItem{
		Agent: "claude", SessionID: "session-001", SnapshotID: id, ProjectID: "p",
	}, dest, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, plain) {
		t.Fatalf("payload %q", payload)
	}
	got, err := os.ReadFile(prior)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("restored file %q", got)
	}
}

func TestRefuseCredentialPush(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	_ = os.WriteFile(p, []byte(`{"token":"x"}`), 0o600)
	eng := testEngine(&Engine{Backend: memory.New(), Passphrase: "p"})
	if _, err := eng.PushSession(context.Background(), PushItem{Agent: "claude", SessionID: "s", LocalPath: p}, false); err == nil {
		t.Fatal("expected refuse")
	}
	safeName := filepath.Join(dir, "session.jsonl")
	_ = os.WriteFile(safeName, []byte(`{"type":"user"}`), 0o600)
	if _, err := eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", LocalPath: safeName,
		RelativePath: "projects/p/.env.production",
	}, false); err == nil {
		t.Fatal("expected relative credential path refusal")
	}
}

func TestPullDryRunStillAuthenticatesAndValidates(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(session, []byte(`{"type":"user"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	good := testEngine(&Engine{Backend: store, Passphrase: "correct-passphrase"})
	id, err := good.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", ProjectID: "p", LocalPath: session,
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	wrong := testEngine(&Engine{Backend: store, Passphrase: "wrong-passphrase"})
	if _, _, err := wrong.PullSession(context.Background(), PullItem{
		Agent: "claude", SessionID: "s", SnapshotID: id, ProjectID: "p",
	}, filepath.Join(dir, "dry-run"), true); err == nil {
		t.Fatal("dry-run accepted an unauthenticated snapshot")
	}
}

func TestPullRejectsPayloadHashMismatch(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(session, []byte(`{"type":"user","text":"original"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	eng := testEngine(&Engine{Backend: store, Passphrase: "correct-passphrase"})
	id, err := eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", ProjectID: "p", LocalPath: session,
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	key := "snapshots/" + id + ".age"
	rc, meta, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	var plain bytes.Buffer
	if err := crypto.Decrypt(rc, &plain, eng.Passphrase); err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
	plain.WriteString("tampered")
	var cipher bytes.Buffer
	if err := (fastAgeEnvelopeCodec{}).Encrypt(bytes.NewReader(plain.Bytes()), &cipher, eng.Passphrase); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), key, bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), backend.PutOptions{IfMatch: meta.ETag}); err != nil {
		t.Fatal(err)
	}

	if _, _, err := eng.PullSession(context.Background(), PullItem{
		Agent: "claude", SessionID: "s", SnapshotID: id, ProjectID: "p",
	}, filepath.Join(dir, "restore"), false); err == nil {
		t.Fatal("pull accepted a payload whose size/hash did not match the envelope")
	}
}

func TestManifestCASRetryMergesUnrelatedConcurrentUpdate(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")
	if err := os.WriteFile(first, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	base := testEngine(&Engine{Backend: store, Passphrase: "passphrase"})
	if _, err := base.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "first", ProjectID: "p", LocalPath: first,
	}, false); err != nil {
		t.Fatal(err)
	}

	racing := &manifestRaceBackend{base: store, passphrase: "passphrase"}
	eng := testEngine(&Engine{Backend: racing, Passphrase: "passphrase"})
	if _, err := eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "second", ProjectID: "p", LocalPath: second,
	}, false); err != nil {
		t.Fatal(err)
	}

	manifest, err := eng.FetchManifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"claude:first", "claude:second", "codex:concurrent"} {
		if _, ok := manifest.Sessions[key]; !ok {
			t.Fatalf("manifest lost %s during CAS retry: %+v", key, manifest.Sessions)
		}
	}
}

func TestPushRejectsRemoteHeadBeyondLocalBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	eng := testEngine(&Engine{Backend: store, Passphrase: "passphrase"})
	first, err := eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", LocalPath: path,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", LocalPath: path,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("expected immutable snapshots")
	}
	if err := os.WriteFile(path, []byte("stale-device"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = eng.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "s", LocalPath: path,
		BaseKnown: true, BaseRevision: first,
	}, false)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale local base was accepted: %v", err)
	}
}

func TestProfilePrefixesIsolateSharedBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("session"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	first := testEngine(&Engine{Backend: store, Passphrase: "passphrase", Prefix: "profiles/one"})
	second := testEngine(&Engine{Backend: store, Passphrase: "passphrase", Prefix: "profiles/two"})
	if _, err := first.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "one", LocalPath: path,
	}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := second.PushSession(context.Background(), PushItem{
		Agent: "claude", SessionID: "two", LocalPath: path,
	}, false); err != nil {
		t.Fatal(err)
	}
	firstManifest, err := first.FetchManifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secondManifest, err := second.FetchManifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := firstManifest.Sessions["claude:two"]; ok {
		t.Fatal("profile one observed profile two")
	}
	if _, ok := secondManifest.Sessions["claude:one"]; ok {
		t.Fatal("profile two observed profile one")
	}
}

type manifestRaceBackend struct {
	base       backend.Backend
	passphrase string
	once       sync.Once
	err        error
}

func (r *manifestRaceBackend) Put(ctx context.Context, key string, body io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	if key == "manifest.age" && opts.IfMatch != "" {
		r.once.Do(func() {
			rc, meta, err := r.base.Get(ctx, key)
			if err != nil {
				r.err = err
				return
			}
			defer func() { _ = rc.Close() }()
			var plain bytes.Buffer
			if err := crypto.Decrypt(rc, &plain, r.passphrase); err != nil {
				r.err = err
				return
			}
			var manifest schema.Manifest
			if err := json.Unmarshal(plain.Bytes(), &manifest); err != nil {
				r.err = err
				return
			}
			manifest.Sessions["codex:concurrent"] = schema.ManifestSession{
				Agent: "codex", SessionID: "concurrent", SnapshotID: "other-snapshot", ProjectID: "other",
			}
			raw, err := json.Marshal(&manifest)
			if err != nil {
				r.err = err
				return
			}
			var cipher bytes.Buffer
			if err := (fastAgeEnvelopeCodec{}).Encrypt(bytes.NewReader(raw), &cipher, r.passphrase); err != nil {
				r.err = err
				return
			}
			_, r.err = r.base.Put(ctx, key, bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), backend.PutOptions{IfMatch: meta.ETag})
		})
		if r.err != nil {
			return backend.ObjectMeta{}, r.err
		}
	}
	return r.base.Put(ctx, key, body, size, opts)
}

func (r *manifestRaceBackend) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	return r.base.Get(ctx, key)
}

func (r *manifestRaceBackend) Head(ctx context.Context, key string) (backend.ObjectMeta, error) {
	return r.base.Head(ctx, key)
}

func (r *manifestRaceBackend) Delete(ctx context.Context, key string) error {
	return r.base.Delete(ctx, key)
}

func (r *manifestRaceBackend) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	return r.base.List(ctx, prefix)
}

type fastAgeEnvelopeCodec struct{}

func (fastAgeEnvelopeCodec) Encrypt(source io.Reader, dest io.Writer, passphrase string) error {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	recipient.SetWorkFactor(1)
	writer, err := age.Encrypt(dest, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, source); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func (fastAgeEnvelopeCodec) DecryptReader(source io.Reader, passphrase string) (io.Reader, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	return age.Decrypt(source, identity)
}

func testEngine(engine *Engine) *Engine {
	engine.codec = fastAgeEnvelopeCodec{}
	return engine
}
