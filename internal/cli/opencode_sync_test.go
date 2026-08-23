package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	opencodeadapter "github.com/HarjjotSinghh/reinstate/internal/adapter/opencode"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"

	_ "modernc.org/sqlite"
)

// TestCLIOpenCodeSyncJourney drives the real CLI (init → list → push → pull)
// for OpenCode against the disk-backed memory backend. OpenCode keeps every
// session in one embedded SQLite store, so this exercises the embedded-store
// sync path end to end: the per-session revision, the extract-to-snapshot
// export, and the restore that writes a session back into a second store.
//
// The store is hydrated from the committed synthetic seed
// testdata/adapters/opencode/macos/store.sql; nothing touches a real store.
//
// The shipped catalog keeps OpenCode at T4 until its native Windows T5 journey
// is recorded, so the adapter is registered through the test-only hook rather
// than the descriptor; the rest of the path is the production code.
func TestCLIOpenCodeSyncJourney(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	_ = os.Unsetenv("REINSTATE_PASSPHRASE")

	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)

	// The OpenCode store resolves under $XDG_DATA_HOME/opencode.
	xdgData := filepath.Join(userHome, "xdgdata")
	t.Setenv("XDG_DATA_HOME", xdgData)
	storeDir := filepath.Join(xdgData, "opencode")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(storeDir, "opencode.db")
	hydrateOpenCodeStore(t, dbPath, "../../testdata/adapters/opencode/macos/store.sql")

	extraSyncAdapters = func() []adapter.Adapter {
		return []adapter.Adapter{&opencodeadapter.Adapter{Home: userHome}}
	}
	t.Cleanup(func() { extraSyncAdapters = nil })

	testCodec := &fastAgeEnvelopeCodec{}
	inactive := func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil }
	run := func(args ...string) (string, string, int) {
		passphraseFile, err := os.CreateTemp(t.TempDir(), "passphrase-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = passphraseFile.Close() }()
		if _, err := passphraseFile.WriteString("opencode-journey-passphrase-not-real\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := passphraseFile.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.FormatUint(uint64(passphraseFile.Fd()), 10))
		var out, errb bytes.Buffer
		code := Execute(Options{
			Name: "reinstate", Stdout: &out, Stderr: &errb, Args: args,
			AgentProcessChecker: inactive,
			EnvelopeCodec:       testCodec,
		})
		return out.String(), errb.String(), code
	}

	out, errb, code := run(
		"init",
		"--endpoint", "https://example.r2.cloudflarestorage.com",
		"--bucket", "reinstate-test",
		"--yes",
	)
	if code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}

	out, errb, code = run("list", "--agent", "opencode", "--json")
	if code != ExitOK {
		t.Fatalf("list exit=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "ses_fixture001") {
		t.Fatalf("list missing opencode session: %q", out)
	}

	// Real push.
	out, errb, code = run("push", "--agent", "opencode", "--session", "ses_fixture001", "--json")
	if code != ExitOK {
		t.Fatalf("push exit=%d err=%q out=%q", code, errb, out)
	}
	var pushRes map[string]any
	if err := json.Unmarshal([]byte(out), &pushRes); err != nil {
		t.Fatalf("push json: %v %q", err, out)
	}
	snaps, _ := pushRes["snapshots"].([]any)
	if len(snaps) != 1 {
		t.Fatalf("push snapshots = %#v, want one", pushRes["snapshots"])
	}

	// A second push of the unchanged session is skipped — the embedded-store
	// revision is stable even though the shared database file changes.
	out, _, code = run("push", "--agent", "opencode", "--session", "ses_fixture001", "--json")
	if code != ExitOK {
		t.Fatalf("second push exit=%d out=%q", code, out)
	}
	var secondPush map[string]any
	if err := json.Unmarshal([]byte(out), &secondPush); err != nil {
		t.Fatalf("second push json: %v %q", err, out)
	}
	if secondPush["skipped"] != float64(1) {
		t.Fatalf("unchanged opencode session was not skipped: %v", secondPush)
	}

	// Simulate the destination device: the store still exists (schema intact)
	// but the session has not arrived yet.
	deleteOpenCodeSession(t, dbPath, "ses_fixture001")
	if openCodeSessionCount(t, dbPath, "ses_fixture001") != 0 {
		t.Fatal("failed to clear session before pull")
	}

	out, errb, code = run("pull", "--agent", "opencode", "--session", "ses_fixture001", "--json")
	if code != ExitOK {
		t.Fatalf("pull exit=%d err=%q out=%q", code, errb, out)
	}
	var pullRes map[string]any
	if err := json.Unmarshal([]byte(out), &pullRes); err != nil {
		t.Fatalf("pull json: %v %q", err, out)
	}
	if pullRes["pulled"] != float64(1) {
		t.Fatalf("pull did not restore the session: %v", pullRes)
	}

	// The session, its messages and parts are back in the vendor's store.
	if openCodeSessionCount(t, dbPath, "ses_fixture001") != 1 {
		t.Fatal("pull did not write the session back into the store")
	}
	if n := openCodeMessageCount(t, dbPath, "ses_fixture001"); n != 2 {
		t.Fatalf("restored message count = %d, want 2", n)
	}
}

func hydrateOpenCodeStore(t *testing.T, dbPath, sqlPath string) {
	t.Helper()
	body, err := os.ReadFile(sqlPath)
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
			t.Fatalf("hydrate exec: %v", err)
		}
	}
}

func deleteOpenCodeSession(t *testing.T, dbPath, id string) {
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
			t.Fatalf("delete: %v", err)
		}
	}
}

func openCodeSessionCount(t *testing.T, dbPath, id string) int {
	return openCodeCount(t, dbPath, "SELECT COUNT(*) FROM session WHERE id = ?", id)
}

func openCodeMessageCount(t *testing.T, dbPath, id string) int {
	return openCodeCount(t, dbPath, "SELECT COUNT(*) FROM message WHERE session_id = ?", id)
}

func openCodeCount(t *testing.T, dbPath, query, id string) int {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(query, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
