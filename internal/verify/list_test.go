package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// TestListStepFollowsEveryListingPage: step 1 tells the reader it asked for
// every object "following every listing page", and a locker with a long
// history holds more than the 1000 keys one ListObjectsV2 answer carries.
// The memory backend has no paging at all, so the claim is only worth
// something against a real S3 client and an endpoint that truncates —
// here, three pages for seven snapshots.
func TestListStepFollowsEveryListingPage(t *testing.T) {
	keys := rootKeys(t)
	const pageSize = 3
	f := s3test.New(t, "lk-0000000000000000000000test")
	f.Accept("AKIA1")
	f.Mu.Lock()
	f.PageSize = pageSize
	f.Mu.Unlock()
	ctx := context.Background()
	client, err := s3.New(ctx, s3.Config{
		Endpoint: f.Srv.URL, Region: "auto", Bucket: f.Bucket,
		Credentials: s3.Static("AKIA1", "secret"), HTTPClient: f.Srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	write := func(key string, body []byte) {
		t.Helper()
		if _, err := client.Put(ctx, key, bytes.NewReader(body), int64(len(body)), backend.PutOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	const snapshots = 7
	man := schema.NewManifest("r1")
	for i := 0; i < snapshots; i++ {
		id := fmt.Sprintf("snap-%02d", i)
		payload := []byte("synthetic session payload " + id + "\n")
		env := schema.Envelope{SchemaVersion: 1, Kind: "snapshot", SnapshotID: id, Agent: "claude", SessionID: id, ProjectID: "local/p",
			CreatedAt: fmt.Sprintf("2026-08-%02dT12:00:00Z", i+1), SourcePlatform: "darwin-arm64",
			Files: []schema.EnvelopeFile{{Path: "s.jsonl", Mode: 0o600, Size: int64(len(payload)), SHA256: crypto.SHA256Hex(payload)}}}
		meta, _ := json.Marshal(env)
		write("snapshots/"+id+".age", seal(t, keys, append(append(meta, '\n'), payload...)))
		man.Sessions["claude:"+id] = schema.ManifestSession{Agent: "claude", SessionID: id, SnapshotID: id, ProjectID: "local/p",
			UpdatedAt: fmt.Sprintf("2026-08-%02dT12:00:00Z", i+1)}
	}
	manRaw, _ := json.Marshal(man)
	write("manifest.age", seal(t, keys, manRaw))

	r := Run(ctx, Options{Backend: client, Keys: keys, Storage: StorageBYO})
	if !r.Passed() {
		t.Fatalf("report %+v", r)
	}
	if got := r.Steps[0].Observed; !strings.Contains(got, fmt.Sprintf("%d object(s)", snapshots+1)) ||
		!strings.Contains(got, fmt.Sprintf("%d snapshot(s)", snapshots)) {
		t.Fatalf("step 1 stopped at a page boundary: %q", got)
	}
	lists := 0
	for _, entry := range f.RequestLog() {
		if strings.HasPrefix(entry, "GET  as ") {
			lists++
		}
	}
	if want := (snapshots + 1 + pageSize - 1) / pageSize; lists != want {
		t.Fatalf("issued %d listing request(s), want %d; nothing here exercises paging", lists, want)
	}
}
