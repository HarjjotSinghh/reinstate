package conformance

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/catalog"

	_ "modernc.org/sqlite"
)

// TestOpenCodeSyncConformanceAtT5 exercises OpenCode's encrypted-sync adapter
// through the shipped catalog's NewSyncAdapter — the same constructor the CLI
// registry uses — and asserts a full extract/apply round trip against the
// committed synthetic store fixture. The generic conformance checks cover the
// index source; this is the T5 surface, which only OpenCode reaches among the
// embedded-store agents, so it is exercised here rather than left implicit in
// the capability check.
//
// Nothing here touches a real store: the store is hydrated from
// testdata/adapters/opencode/macos/store.sql under a throwaway XDG root.
func TestOpenCodeSyncConformanceAtT5(t *testing.T) {
	desc := catalog.OpenCode()
	if desc.Tier < agents.TierSync {
		t.Fatalf("OpenCode tier %s is below T5; sync conformance would prove nothing", desc.Tier)
	}
	if desc.NewSyncAdapter == nil {
		t.Fatal("OpenCode declares T5 but NewSyncAdapter is not wired")
	}

	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(root, "testdata", "adapters", "opencode", "macos", "store.sql")

	xdg := t.TempDir()
	storeDir := filepath.Join(xdg, "opencode")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(storeDir, "opencode.db")
	hydrateStoreFromSeed(t, dbPath, seed)

	// The adapter resolves $XDG_DATA_HOME/opencode itself, exactly as in
	// production; Home drives the portable path remapping.
	t.Setenv("XDG_DATA_HOME", xdg)
	a, err := desc.NewSyncAdapter(agents.Env{Home: "/Users/fixture-user", LookupEnv: os.Getenv})
	if err != nil || a == nil {
		t.Fatalf("NewSyncAdapter: %v", err)
	}

	ctx := context.Background()
	sessions, err := a.Discover(ctx, adapter.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_fixture001" {
		t.Fatalf("discover = %+v", sessions)
	}
	sess := sessions[0]
	// Discover's project identity is the portable token, and filtering by it
	// returns the same session — the round-trip identity the sync engine needs.
	if !strings.HasPrefix(sess.ProjectID, "${") {
		t.Fatalf("project identity %q is not portable", sess.ProjectID)
	}

	plan, err := a.PlanExport(ctx, sess, adapter.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err := a.Export(ctx, plan, &archive); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(archive.Bytes(), []byte("not-a-real-token")) ||
		bytes.Contains(archive.Bytes(), []byte("access-token")) {
		t.Fatal("export leaked a credential or account value")
	}

	// Simulate the destination device: the store exists but the session has not
	// arrived yet.
	clearSessionRows(t, dbPath, "ses_fixture001")

	snap := adapter.Snapshot{
		Agent: "opencode", SessionID: sess.ID, ProjectID: sess.ProjectID,
		RelativePath: sess.RelativePath,
	}
	rplan, err := a.PlanRestore(ctx, snap, adapter.RestoreOptions{
		CompatibilityOK: true, BackupRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rplan.Refuse != "" {
		t.Fatalf("restore refused: %s", rplan.Refuse)
	}
	if err := a.Restore(ctx, rplan, bytes.NewReader(archive.Bytes())); err != nil {
		t.Fatal(err)
	}

	back, err := a.Discover(ctx, adapter.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || back[0].ID != "ses_fixture001" {
		t.Fatalf("session did not round-trip through the catalog adapter: %+v", back)
	}
	if got := sessionMessageCount(t, dbPath, "ses_fixture001"); got != 2 {
		t.Fatalf("restored message count = %d, want 2", got)
	}
}

func hydrateStoreFromSeed(t *testing.T, dbPath, seed string) {
	t.Helper()
	body, err := os.ReadFile(seed)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, stmt := range strings.Split(string(body), ";") {
		s := strings.TrimSpace(stmt)
		if s == "" || (strings.HasPrefix(s, "--") && !strings.Contains(s, "CREATE") && !strings.Contains(s, "INSERT")) {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("hydrate: %v", err)
		}
	}
}

func clearSessionRows(t *testing.T, dbPath, id string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, q := range []string{
		"DELETE FROM part WHERE session_id = ?",
		"DELETE FROM message WHERE session_id = ?",
		"DELETE FROM session WHERE id = ?",
	} {
		if _, err := db.Exec(q, id); err != nil {
			t.Fatalf("clear: %v", err)
		}
	}
}

func sessionMessageCount(t *testing.T, dbPath, id string) int {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM message WHERE session_id = ?", id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
