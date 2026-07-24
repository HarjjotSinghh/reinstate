package sync

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

func TestPushPullRoundTripMemory(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session-001.jsonl")
	plain := []byte(`{"type":"user","text":"synthetic fixture only"}`)
	if err := os.WriteFile(session, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	eng := &Engine{Backend: store, Passphrase: "test-pass-phrase-32", Platform: "darwin-arm64"}
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
	if err := Resolve(home, "c1", KeepLocal); err != nil {
		t.Fatal(err)
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

func TestRefuseCredentialPush(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	_ = os.WriteFile(p, []byte(`{"token":"x"}`), 0o600)
	eng := &Engine{Backend: memory.New(), Passphrase: "p"}
	if _, err := eng.PushSession(context.Background(), PushItem{Agent: "claude", SessionID: "s", LocalPath: p}, false); err == nil {
		t.Fatal("expected refuse")
	}
}
