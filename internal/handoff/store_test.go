package handoff

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
)

func TestOpenStoreModesAndLayout(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	store, err := OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	root := store.Root()
	wantRoot := filepath.Join(home, "handoffs")
	if root != wantRoot {
		t.Fatalf("root = %q, want %q", root, wantRoot)
	}
	assertOwnerOnlyDir(t, home)
	assertOwnerOnlyDir(t, root)

	id, err := store.Put(testCapsule("aabbccddeeff00112233445566778899"), Artifacts{
		ProjectionMD:  []byte("# projection\n"),
		Bootstrap:     []byte("bootstrap prompt"),
		FidelityJSON:  []byte(`{"overall":"normalized","mode":"structured_handoff","components":[]}`),
		SidecarEvents: []byte("{\"id\":\"e1\"}\n"),
		SidecarBlobs:  map[string][]byte{"deadbeef": []byte("blob")},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, id)
	assertOwnerOnlyDir(t, dir)
	for _, name := range []string{"capsule.json", "projection.md", "bootstrap.txt", "fidelity.json"} {
		assertOwnerOnlyFile(t, filepath.Join(dir, name))
	}
	assertOwnerOnlyFile(t, filepath.Join(dir, "sidecar", "events.jsonl"))
	assertOwnerOnlyFile(t, filepath.Join(dir, "sidecar", "blobs", "deadbeef"))
}

func TestPutGetRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := testCapsule("11223344556677889900aabbccddeeff")
	arts := Artifacts{
		ProjectionMD: []byte("projection body"),
		Bootstrap:    []byte("boot"),
		FidelityJSON: []byte(`{"overall":"exact","mode":"structured_handoff","components":[]}`),
	}
	id, err := store.Put(c, arts)
	if err != nil {
		t.Fatal(err)
	}
	if id != c.Identity.ID {
		t.Fatalf("id = %q, want %q", id, c.Identity.ID)
	}
	got, gotArts, err := store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Identity.ID != id || got.Schema != capsule.Schema {
		t.Fatalf("capsule mismatch: %+v", got.Identity)
	}
	if string(gotArts.ProjectionMD) != "projection body" || string(gotArts.Bootstrap) != "boot" {
		t.Fatalf("artifacts mismatch: %+v", gotArts)
	}
}

func TestGetMissingTypedNotFound(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Get("missing-handoff-id-0123456789abcdef")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}
}

func TestAppendLineageConcurrentAndListSkipsPartial(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		n := i
		go func() {
			defer wg.Done()
			id := "concurrent-handoff-" + string(rune('a'+n))
			errs <- store.AppendLineage(LineageEntry{
				HandoffID:   id,
				LineageRoot: id,
				CreatedAt:   time.Date(2026, 8, 12, 10, n, 0, 0, time.UTC),
				Source:      LineageEndpoint{Agent: "claude", SessionID: "s1"},
				Destination: LineageEndpoint{Agent: "codex", State: "resolved"},
				Policy:      "balanced",
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(store.Root(), "lineage.jsonl")
	partial, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := partial.WriteString(`{"handoff_id":"partial`); err != nil {
		_ = partial.Close()
		t.Fatal(err)
	}
	if err := partial.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2 well-formed lines", len(entries))
	}
	for _, e := range entries {
		if _, marshalErr := json.Marshal(e); marshalErr != nil || e.HandoffID == "" {
			t.Fatalf("malformed list entry: %+v (%v)", e, marshalErr)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `{"handoff_id":"partial`) {
		t.Fatal("partial line was rewritten away from lineage.jsonl")
	}
	if strings.HasSuffix(body, "\n") {
		t.Fatal("expected trailing partial line without terminating newline")
	}
}

func TestOpenStoreRejectsRepositoryPath(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := OpenStore(home)
	if !errors.Is(err, ErrInsideRepository) {
		t.Fatalf("OpenStore inside repo = %v, want ErrInsideRepository", err)
	}
}

func TestStoreRootNeverInsideModuleRepository(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	store, err := OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(store.Root(), home+string(filepath.Separator)) && store.Root() != filepath.Join(home, "handoffs") {
		t.Fatalf("store root %q is not under reinstate home %q", store.Root(), home)
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(store.Root(), moduleRoot+string(filepath.Separator)) {
		t.Fatalf("store root %q must not live inside module repository %q", store.Root(), moduleRoot)
	}
}

func TestListRecoversArtifactDirWithoutLineage(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.Put(testCapsule("aabbccddeeff00112233445566778899"), Artifacts{
		ProjectionMD: []byte("# p\n"),
		Bootstrap:    []byte("boot"),
		FidelityJSON: []byte(`{"overall":"normalized","mode":"structured_handoff","components":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].HandoffID != id || entries[0].Source.Agent != "claude" {
		t.Fatalf("recovered list = %+v", entries)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), lineageFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dir recovery must not create lineage.jsonl")
	}
}

func TestListKeepsRepeatedLineageRowsForSameHandoffID(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.Put(testCapsule("aabbccddeeff00112233445566778899"), Artifacts{
		ProjectionMD: []byte("# p\n"),
		Bootstrap:    []byte("boot"),
		FidelityJSON: []byte(`{"overall":"normalized","mode":"structured_handoff","components":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := store.AppendLineage(LineageEntry{
			HandoffID:   id,
			LineageRoot: id,
			CreatedAt:   time.Date(2026, 8, 12, 10, i, 0, 0, time.UTC),
			Source:      LineageEndpoint{Agent: "claude", SessionID: "s1"},
			Destination: LineageEndpoint{Agent: "codex", State: "resolved"},
			Policy:      "balanced",
			Launched:    true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("List = %d, want 3 lineage rows (no HandoffID dedupe)", len(entries))
	}
}

func testCapsule(id string) capsule.Capsule {
	return capsule.Capsule{
		Schema: capsule.Schema,
		Identity: capsule.Identity{
			ID:          id,
			LineageRoot: id,
			Parent: capsule.Parent{
				Agent:          "claude",
				SessionID:      "sess-1",
				ArtifactSHA256: "deadbeef",
				AdapterVersion: "1",
			},
			SchemaVer: capsule.SchemaVersion,
		},
		RawSource: capsule.RawSource{
			Agent:          "claude",
			SessionID:      "sess-1",
			ArtifactSHA256: "deadbeef",
			AdapterVersion: "1",
			ByteOffset:     0,
			SizeBytes:      8,
		},
		Task: capsule.Task{
			Goal:             capsule.TextField{Text: "ship", Portability: capsule.PortabilityNormalized, Label: "derived_deterministic"},
			LatestUserIntent: capsule.TextField{Text: "continue", Portability: capsule.PortabilityExact},
			Constraints:      capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Decisions:        capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "requires_optional_summarizer"},
			RejectedApproaches: capsule.ListField{
				Portability: capsule.PortabilityOmitted,
				Reason:      "requires_optional_summarizer",
			},
			Pending: capsule.ListField{Portability: capsule.PortabilityOmitted, Reason: "interrupted_not_replayed"},
		},
		Workspace: capsule.Workspace{
			ProjectID: "github.com/example/demo",
			Root:      "${REPO:github.com/example/demo}",
		},
		Capabilities: capsule.CapabilityDiff{
			Source:      map[string]any{"mcp_count": 0},
			Destination: map[string]any{"mcp_count": 0},
		},
		Security: capsule.Security{SourceInstructionsAreUntrustedHistory: true},
		Fidelity: capsule.Fidelity{
			Overall: capsule.PortabilityNormalized,
			Mode:    capsule.FidelityModeStructuredHandoff,
			Components: []capsule.Component{
				{Name: "goal", Portability: capsule.PortabilityNormalized, Count: 1, Bytes: 4},
			},
		},
		Projection: capsule.Projection{Policy: "balanced"},
	}
}

func assertOwnerOnlyDir(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("%s mode = %04o, want 0700", path, got)
	}
}

func assertOwnerOnlyFile(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("%s mode = %04o, want 0600", path, got)
	}
}
