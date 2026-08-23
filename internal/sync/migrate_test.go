package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// migrateFixture is a hosted-style source (root key, keyring object, several
// snapshots including an orphan and a Windows-recorded one) ready to leave.
type migrateFixture struct {
	source  *memory.Store
	src     *Engine
	rootKey []byte
	// plain holds every snapshot's plaintext payload by snapshot id.
	plain map[string][]byte
	ids   []string
}

func newMigrateFixture(t *testing.T) *migrateFixture {
	t.Helper()
	rootKey := make([]byte, 32)
	if _, err := rand.Read(rootKey); err != nil {
		t.Fatal(err)
	}
	keys, err := crypto.NewRootKeyProvider(rootKey)
	if err != nil {
		t.Fatal(err)
	}
	source := memory.New()
	src := testEngine(&Engine{Backend: source, Keys: keys, Platform: "windows-amd64"})
	ctx := context.Background()
	// A keyring object stands in the source the way a locker holds one.
	ring := []byte(`{"schema_version":1,"profile_id":"acct-1","current_generation":1,"wrapped":"` + base64.StdEncoding.EncodeToString(rootKey) + `"}`)
	if _, err := source.Put(ctx, "keyring.v1.json", bytes.NewReader(ring), int64(len(ring)), backend.PutOptions{IfNoneMatch: true}); err != nil {
		t.Fatal(err)
	}
	f := &migrateFixture{source: source, src: src, rootKey: rootKey, plain: map[string][]byte{}}
	dir := t.TempDir()
	push := func(agent, session, relative, body string) string {
		t.Helper()
		local := filepath.Join(dir, agent+"-"+session+".jsonl")
		if err := os.WriteFile(local, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		id, err := src.PushSession(ctx, PushItem{Agent: agent, SessionID: session, ProjectID: "github.com/example/app", LocalPath: local, RelativePath: relative}, false)
		if err != nil {
			t.Fatal(err)
		}
		f.plain[id] = []byte(body)
		f.ids = append(f.ids, id)
		return id
	}
	// Two revisions of one session: the first becomes an orphan the
	// manifest no longer points at, and it must still move.
	push("claude", "s-1", "projects/C--Users-dev-Projects-app/s-1.jsonl", `{"type":"user","cwd":"C:\\Users\\dev\\Projects\\app","text":"first"}`)
	push("claude", "s-1", "projects/C--Users-dev-Projects-app/s-1.jsonl", `{"type":"user","cwd":"C:\\Users\\dev\\Projects\\app","text":"second"}`)
	push("codex", "s-2", "sessions/2026/08/23/rollout-s-2.jsonl", strings.Repeat(`{"type":"assistant","text":"codex synthetic"}`+"\n", 200))
	return f
}

// destPair returns a destination engine over a fresh in-memory store with
// the BYO passphrase model under a profile prefix.
func destPair(passphrase string) (*memory.Store, *Engine) {
	store := memory.New()
	return store, testEngine(&Engine{Backend: store, Keys: testKeys(passphrase), Prefix: "profiles/11111111-2222-3333-4444-555555555555"})
}

func TestMigrateMovesEverySnapshotAndManifestWithFidelity(t *testing.T) {
	f := newMigrateFixture(t)
	destStore, dest := destPair("new-byo-passphrase")
	ctx := context.Background()
	var steps []MigrateProgress
	m := &Migration{Source: f.src, Destination: dest, Progress: func(p MigrateProgress) { steps = append(steps, p) }}
	report, err := m.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Snapshots != 3 || report.Written != 3 || report.Verified != 0 || report.Skipped != 0 || report.ManifestSessions != 2 || report.Bytes == 0 {
		t.Fatalf("report %+v", report)
	}
	if len(steps) != 4 || steps[3].SnapshotID != "" || steps[2].Completed != 3 || steps[2].Total != 3 {
		t.Fatalf("progress %+v", steps)
	}

	// Fidelity: a BYO-only reader pulls each head and every orphan back
	// byte for byte, Windows path shapes included.
	reader := testEngine(&Engine{Backend: destStore, Keys: testKeys("new-byo-passphrase"), Prefix: dest.Prefix, RequireRemoteManifest: true})
	man, err := reader.FetchManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	srcMan, _ := f.src.FetchManifest(ctx)
	if man.Revision != srcMan.Revision || len(man.Sessions) != 2 || man.Sessions["claude:s-1"].SnapshotID != f.ids[1] || man.Sessions["codex:s-2"].SnapshotID != f.ids[2] {
		t.Fatalf("manifest %+v", man)
	}
	for _, id := range f.ids {
		env, payload, err := reader.PullSession(ctx, PullItem{SnapshotID: id}, t.TempDir(), false)
		if err != nil {
			t.Fatalf("pull %s: %v", id, err)
		}
		if !bytes.Equal(payload, f.plain[id]) {
			t.Fatalf("snapshot %s payload differs", id)
		}
		if env.SourcePlatform != "windows-amd64" || env.SnapshotID != id {
			t.Fatalf("envelope %+v", env)
		}
		if env.Agent == "claude" && env.Files[0].Path != "projects/C--Users-dev-Projects-app/s-1.jsonl" {
			t.Fatalf("windows-recorded path changed: %q", env.Files[0].Path)
		}
	}

	// The root key is absent from the destination: no keyring object, no
	// object anywhere that carries the key in any encoding, and every
	// envelope is sealed to scrypt only.
	objects, _ := destStore.List(ctx, "")
	if len(objects) != 4 {
		t.Fatalf("destination objects %+v", objects)
	}
	forms := [][]byte{f.rootKey, []byte(base64.StdEncoding.EncodeToString(f.rootKey)), []byte(hex.EncodeToString(f.rootKey))}
	for _, o := range objects {
		if strings.HasSuffix(o.Key, "keyring.v1.json") || !strings.HasPrefix(o.Key, dest.Prefix+"/") {
			t.Fatalf("unexpected destination object %s", o.Key)
		}
		rc, _, _ := destStore.Get(ctx, o.Key)
		raw, _ := io.ReadAll(rc)
		_ = rc.Close()
		for _, form := range forms {
			if bytes.Contains(raw, form) {
				t.Fatalf("root key material in %s", o.Key)
			}
		}
		header, _, _ := bytes.Cut(raw, []byte("\n---"))
		if !bytes.Contains(header, []byte("-> scrypt")) || bytes.Contains(header, []byte("-> X25519")) {
			t.Fatalf("%s is not sealed to the passphrase alone:\n%s", o.Key, header)
		}
	}
	if has, _ := dest.ContainsKeyringObject(ctx); has {
		t.Fatal("keyring reported at destination")
	}
	// The source was only read.
	srcObjects, _ := f.source.List(ctx, "")
	if len(srcObjects) != 5 {
		t.Fatalf("source objects changed: %+v", srcObjects)
	}
	if _, err := (&Migration{Source: f.src, Destination: testEngine(&Engine{Backend: destStore, Keys: testKeys("wrong"), Prefix: dest.Prefix})}).Run(ctx); err == nil {
		t.Fatal("a second run under another passphrase must fail verification, not overwrite")
	}
}

// failingPuts is a destination that fails after n successful snapshot puts,
// standing in for an interrupted run.
type failingPuts struct {
	backend.Backend
	remaining int
	puts      int
}

var errInterrupted = errors.New("connection reset by peer")

func (f *failingPuts) Put(ctx context.Context, key string, r io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	if strings.Contains(key, "/snapshots/") {
		if f.remaining == 0 {
			return backend.ObjectMeta{}, errInterrupted
		}
		f.remaining--
	}
	f.puts++
	return f.Backend.Put(ctx, key, r, size, opts)
}

func TestMigrateResumesAfterInterruptionWithoutDuplicates(t *testing.T) {
	f := newMigrateFixture(t)
	destStore, dest := destPair("resume-passphrase")
	ctx := context.Background()
	flaky := &failingPuts{Backend: destStore, remaining: 1}
	dest.Backend = flaky
	done := map[string]string{}
	progress := func(p MigrateProgress) {
		if p.SnapshotID != "" {
			done[p.SnapshotID] = p.Digest
		}
	}
	_, err := (&Migration{Source: f.src, Destination: dest, Progress: progress}).Run(ctx)
	if !errors.Is(err, errInterrupted) {
		t.Fatalf("first run err=%v", err)
	}
	if len(done) != 1 || flaky.puts != 1 {
		t.Fatalf("after interruption done=%v puts=%d", done, flaky.puts)
	}
	if _, _, err := destStore.Get(ctx, dest.key("manifest.age")); !errors.Is(err, backend.ErrNotFound) {
		t.Fatalf("manifest written before every snapshot was verified: %v", err)
	}

	// Second run: the finished snapshot is skipped from Done, the rest are
	// written, and nothing is written twice.
	flaky.remaining = 10
	flaky.puts = 0
	report, err := (&Migration{Source: f.src, Destination: dest, Done: done, Progress: progress}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Skipped != 1 || report.Written != 2 || report.Verified != 0 || flaky.puts != 3 {
		t.Fatalf("resume report %+v puts=%d", report, flaky.puts)
	}
	objects, _ := destStore.List(ctx, "")
	if len(objects) != 4 {
		t.Fatalf("destination objects after resume: %+v", objects)
	}

	// A crash between the put and recording Done: the object exists but is
	// not in Done, so it is verified in place, not rewritten.
	delete(done, f.ids[2])
	flaky.puts = 0
	report, err = (&Migration{Source: f.src, Destination: dest, Done: done}).Run(ctx)
	if err != nil || report.Verified != 1 || report.Skipped != 2 || report.Written != 0 || flaky.puts != 1 {
		t.Fatalf("third run report %+v puts=%d err=%v", report, flaky.puts, err)
	}
	// Corrupt an existing destination object: verification refuses.
	bad := []byte("not an envelope")
	_ = destStore.Delete(ctx, dest.key("snapshots/"+f.ids[0]+".age"))
	_, _ = destStore.Put(ctx, dest.key("snapshots/"+f.ids[0]+".age"), bytes.NewReader(bad), int64(len(bad)), backend.PutOptions{})
	_, err = (&Migration{Source: f.src, Destination: dest}).Run(ctx)
	if !errors.Is(err, ErrMigrateVerify) {
		t.Fatalf("corrupt destination: err=%v", err)
	}
}

func TestMigrateRefusesDestinationInUse(t *testing.T) {
	f := newMigrateFixture(t)
	destStore, dest := destPair("busy")
	ctx := context.Background()
	other := testEngine(&Engine{Backend: destStore, Keys: testKeys("busy"), Prefix: dest.Prefix})
	local := filepath.Join(t.TempDir(), "x.jsonl")
	_ = os.WriteFile(local, []byte("x"), 0o600)
	if _, err := other.PushSession(ctx, PushItem{Agent: "claude", SessionID: "elsewhere", LocalPath: local}, false); err != nil {
		t.Fatal(err)
	}
	_, err := (&Migration{Source: f.src, Destination: dest}).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "in use") {
		t.Fatalf("err=%v", err)
	}
}

func TestListSnapshotsIgnoresOtherObjects(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	for _, key := range []string{"p/snapshots/a.age", "p/snapshots/b.age", "p/snapshots/nested/c.age", "p/manifest.age", "p/keyring.v1.json", "q/snapshots/d.age"} {
		_, _ = store.Put(ctx, key, strings.NewReader("x"), 1, backend.PutOptions{})
	}
	ids, err := (&Engine{Backend: store, Prefix: "p"}).ListSnapshots(ctx)
	if err != nil || fmt.Sprint(ids) != "[a b]" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}
