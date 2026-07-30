package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	syncengine "github.com/HarjjotSinghh/reinstate/internal/sync"
)

func TestConcreteConflictResolutionStrategies(t *testing.T) {
	t.Run("keep remote", func(t *testing.T) {
		fixture := newConflictFixture(t)
		if err := resolveKeepRemote(context.Background(), fixture.engine, fixture.home, fixture.registry, fixture.conflict, false); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(fixture.sessionPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(got, []byte("remote")) {
			t.Fatalf("remote revision was not restored: %s", got)
		}
		backups, err := filepath.Glob(filepath.Join(fixture.home, "backups", "*", "projects", "p", "s.jsonl"))
		if err != nil || len(backups) != 1 {
			t.Fatalf("backups=%v err=%v", backups, err)
		}
	})

	t.Run("keep both", func(t *testing.T) {
		fixture := newConflictFixture(t)
		if err := resolveKeepRemote(context.Background(), fixture.engine, fixture.home, fixture.registry, fixture.conflict, true); err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(fixture.sessionPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(original, []byte("local")) {
			t.Fatalf("keep-both replaced local revision: %s", original)
		}
		forkID := forkSessionID("remote", "s", fixture.snapshotID)
		if _, err := discoverSession(context.Background(), fixture.registry, "claude", forkID); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("keep local", func(t *testing.T) {
		fixture := newConflictFixture(t)
		if err := resolveKeepLocal(context.Background(), fixture.engine, fixture.home, fixture.registry, fixture.conflict); err != nil {
			t.Fatal(err)
		}
		manifest, err := fixture.engine.FetchManifest(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		head := manifest.Sessions["claude:s"].SnapshotID
		if head == fixture.snapshotID || head == "" {
			t.Fatalf("keep-local did not publish a new remote revision: %q", head)
		}
	})
}

type conflictFixture struct {
	home        string
	sessionPath string
	snapshotID  string
	engine      *syncengine.Engine
	registry    *adapter.Registry
	conflict    syncengine.Conflict
}

func newConflictFixture(t *testing.T) conflictFixture {
	t.Helper()
	home := t.TempDir()
	if err := config.EnsureLayout(home); err != nil {
		t.Fatal(err)
	}
	cfg := schema.DefaultConfig("profile", "device")
	if err := config.SaveConfig(home, cfg); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveState(home, schema.NewState()); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), ".claude")
	project := filepath.Join(root, "projects", "p")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(project, "s.jsonl")
	if err := os.WriteFile(sessionPath, []byte(`{"type":"user","message":{"content":"remote"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := &claude.Adapter{Root: root}
	registry := adapter.NewRegistry()
	if err := registry.Register(selected); err != nil {
		t.Fatal(err)
	}
	sessions, err := selected.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(sessions) != 1 {
		t.Fatalf("sessions=%v err=%v", sessions, err)
	}
	plan, err := selected.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := os.CreateTemp(home, "remote-*.tar")
	if err != nil {
		t.Fatal(err)
	}
	if err := selected.Export(context.Background(), plan, artifact); err != nil {
		t.Fatal(err)
	}
	artifactPath := artifact.Name()
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(artifactPath) })
	engine := &syncengine.Engine{
		Backend: memory.New(), Passphrase: "test-passphrase",
		Codec: &fastAgeEnvelopeCodec{},
	}
	snapshotID, err := engine.PushSession(context.Background(), syncengine.PushItem{
		Agent: "claude", SessionID: "s", ProjectID: "p",
		LocalPath: artifactPath, RelativePath: sessions[0].RelativePath,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte(`{"type":"user","message":{"content":"local"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return conflictFixture{
		home: home, sessionPath: sessionPath, snapshotID: snapshotID,
		engine: engine, registry: registry,
		conflict: syncengine.Conflict{
			ID: "conflict", Agent: "claude", SessionID: "s", ProjectID: "p",
			RemoteRevision: snapshotID, RemoteSnapshot: snapshotID,
		},
	}
}
