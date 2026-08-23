package sync

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// x25519KeyProvider is a test-only second key model: a fixed age X25519
// identity standing in for the hosted root key. Swapping it in must not
// touch sync, manifest, or conflict code.
type x25519KeyProvider struct{ identity *age.X25519Identity }

func newX25519KeyProvider(t *testing.T) *x25519KeyProvider {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	return &x25519KeyProvider{identity: identity}
}

func (p *x25519KeyProvider) Recipients() ([]age.Recipient, error) {
	return []age.Recipient{p.identity.Recipient()}, nil
}

func (p *x25519KeyProvider) Identities() ([]age.Identity, error) {
	return []age.Identity{p.identity}, nil
}

func TestEngineJourneyByKeyProvider(t *testing.T) {
	tests := []struct {
		name  string
		keys  crypto.KeyProvider
		other crypto.KeyProvider
	}{
		{name: "passphrase", keys: testKeys("test-pass-phrase-32"), other: testKeys("different-passphrase")},
		{name: "x25519", keys: newX25519KeyProvider(t), other: newX25519KeyProvider(t)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			session := filepath.Join(dir, "session-001.jsonl")
			plain := []byte(`{"type":"user","text":"synthetic fixture only"}`)
			if err := os.WriteFile(session, plain, 0o600); err != nil {
				t.Fatal(err)
			}
			store := memory.New()
			eng := testEngine(&Engine{Backend: store, Keys: tc.keys, Platform: "darwin-arm64"})
			item := PushItem{Agent: "claude", SessionID: "s1", ProjectID: "p1", LocalPath: session, RelativePath: "projects/p1/session-001.jsonl", BaseKnown: true}
			id, err := eng.PushSession(context.Background(), item, false)
			if err != nil {
				t.Fatalf("push: %v", err)
			}
			for _, key := range []string{"manifest.age", "snapshots/" + id + ".age"} {
				rc, _, err := store.Get(context.Background(), key)
				if err != nil {
					t.Fatal(err)
				}
				raw := new(bytes.Buffer)
				_, _ = raw.ReadFrom(rc)
				_ = rc.Close()
				if bytes.Contains(raw.Bytes(), plain) || bytes.Contains(raw.Bytes(), []byte("session-001")) {
					t.Fatalf("%s stored plaintext", key)
				}
			}

			env, got, err := eng.PullSession(context.Background(), PullItem{Agent: "claude", SessionID: "s1", SnapshotID: id, ProjectID: "p1"}, filepath.Join(dir, "pull"), false)
			if err != nil {
				t.Fatalf("pull: %v", err)
			}
			if !bytes.Equal(got, plain) || env.SnapshotID != id {
				t.Fatalf("pull mismatch: env=%+v payload=%q", env, got)
			}
			manifest, err := eng.FetchManifest(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Sessions[SessionKey("claude", "s1")].SnapshotID != id {
				t.Fatalf("manifest head = %+v", manifest.Sessions)
			}

			// Conflict detection is unchanged by the key model.
			if _, err := eng.PushSession(context.Background(), item, false); !errors.Is(err, ErrConflict) {
				t.Fatalf("stale base push error = %v, want ErrConflict", err)
			}

			// A different key of the same model cannot read the profile.
			wrong := testEngine(&Engine{Backend: store, Keys: tc.other, RequireRemoteManifest: true})
			if _, err := wrong.FetchManifest(context.Background()); err == nil {
				t.Fatal("different key read the manifest")
			}
			if _, _, err := wrong.PullSession(context.Background(), PullItem{SnapshotID: id}, filepath.Join(dir, "wrong"), false); err == nil {
				t.Fatal("different key read the snapshot")
			}
		})
	}
}

func TestEngineRequiresKeyProvider(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "s.jsonl")
	if err := os.WriteFile(session, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	eng := testEngine(&Engine{Backend: memory.New()})
	_, err := eng.PushSession(context.Background(), PushItem{Agent: "claude", SessionID: "s", LocalPath: session}, false)
	if err == nil || !strings.Contains(err.Error(), "key provider required") {
		t.Fatalf("push without keys error = %v", err)
	}
	if _, _, err := eng.PullSession(context.Background(), PullItem{SnapshotID: "none"}, dir, true); err == nil {
		t.Fatal("pull without keys succeeded")
	}
}

// TestGoldenPreSeamProfileReadsUnchanged loads a manifest and snapshot written
// by the engine before the key-provider seam (testdata/crypto/pre-seam) and
// requires the passphrase provider to read them with identical plaintext.
func TestGoldenPreSeamProfileReadsUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("decrypts at age's default scrypt cost")
	}
	fixture := filepath.Join("..", "..", "testdata", "crypto", "pre-seam")
	read := func(name string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	snapshotID := strings.TrimSpace(string(read("snapshot-id.txt")))
	store := memory.New()
	for key, name := range map[string]string{"manifest.age": "manifest.age", "snapshots/" + snapshotID + ".age": "snapshot.age"} {
		raw := read(name)
		if _, err := store.Put(context.Background(), key, bytes.NewReader(raw), int64(len(raw)), backend.PutOptions{IfNoneMatch: true}); err != nil {
			t.Fatal(err)
		}
	}
	// Production codec and default work factor: this is the real decrypt path.
	eng := &Engine{Backend: store, Keys: crypto.NewPassphraseProvider("golden-passphrase-not-real"), RequireRemoteManifest: true}
	manifest, err := eng.FetchManifest(context.Background())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	head, ok := manifest.Sessions[SessionKey("claude", "golden-session")]
	if !ok || head.SnapshotID != snapshotID || head.ProjectID != "golden-project" {
		t.Fatalf("manifest head = %+v", manifest.Sessions)
	}
	env, payload, err := eng.PullSession(context.Background(), PullItem{Agent: "claude", SessionID: "golden-session", SnapshotID: snapshotID, ProjectID: "golden-project"}, t.TempDir(), false)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if want := read("session-golden.jsonl"); !bytes.Equal(payload, want) {
		t.Fatalf("golden payload mismatch: got %q want %q", payload, want)
	}
	if env.Files[0].Path != "projects/golden/session-golden.jsonl" || env.SourcePlatform != "darwin-arm64" {
		t.Fatalf("golden envelope = %+v", env)
	}
}
