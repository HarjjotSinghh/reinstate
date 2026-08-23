package opencode

import (
	"archive/tar"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"

	_ "modernc.org/sqlite"
)

// hydrateStore builds an opencode.db under a fresh root from a committed
// store.sql seed and returns the root directory.
func hydrateStore(t *testing.T, sqlPath string) string {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, DatabaseName)
	execScript(t, dbPath, readSeed(t, sqlPath))
	return root
}

// schemaOnlyStore builds an opencode.db with the fixture schema but no rows, so
// a restore into it proves row creation rather than update.
func schemaOnlyStore(t *testing.T, sqlPath string) string {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, DatabaseName)
	execScript(t, dbPath, readSeed(t, sqlPath))
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, tbl := range []string{"part", "message", "session", "project", "credential"} {
		if _, err := db.Exec("DELETE FROM " + tbl); err != nil {
			t.Fatalf("clear %s: %v", tbl, err)
		}
	}
	return root
}

func readSeed(t *testing.T, sqlPath string) string {
	t.Helper()
	body, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func execScript(t *testing.T, dbPath, script string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, stmt := range strings.Split(script, ";") {
		s := strings.TrimSpace(stmt)
		if s == "" || strings.HasPrefix(s, "--") && !strings.Contains(s, "CREATE") && !strings.Contains(s, "INSERT") {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", firstLine(s), err)
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

const (
	macosSeed   = "../../../testdata/adapters/opencode/macos/store.sql"
	windowsSeed = "../../../testdata/adapters/opencode/windows/store.sql"
)

func TestDetect(t *testing.T) {
	root := hydrateStore(t, macosSeed)
	a := &Adapter{Root: root, Home: "/Users/fixture-user"}
	inst, compat, err := a.Detect(context.Background())
	if err != nil || compat != adapter.CompatibilitySupported {
		t.Fatalf("detect = %+v %s %v", inst, compat, err)
	}
	if inst.Layout != "embedded-sqlite-session-store" {
		t.Fatalf("layout = %q", inst.Layout)
	}

	empty := &Adapter{Root: t.TempDir()}
	_, compat, err = empty.Detect(context.Background())
	if err != nil || compat != adapter.CompatibilityNotInstalled {
		t.Fatalf("empty detect = %s %v", compat, err)
	}
}

func TestDiscover(t *testing.T) {
	root := hydrateStore(t, macosSeed)
	a := &Adapter{Root: root, Home: "/Users/fixture-user"}
	sessions, err := a.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d", len(sessions))
	}
	s := sessions[0]
	if s.ID != "ses_fixture001" || s.Agent != "opencode" {
		t.Fatalf("session = %+v", s)
	}
	if s.RelativePath != "sessions/ses_fixture001.json" {
		t.Fatalf("relative = %q", s.RelativePath)
	}
	if s.ProjectID != "/Users/fixture-user/code/demo" {
		t.Fatalf("project = %q", s.ProjectID)
	}
	// Project filter that matches nothing.
	none, err := a.Discover(context.Background(), adapter.DiscoverOptions{ProjectID: "/nope"})
	if err != nil || len(none) != 0 {
		t.Fatalf("filtered = %+v %v", none, err)
	}
}

func exportOne(t *testing.T, a *Adapter, id string) []byte {
	t.Helper()
	plan, err := a.PlanExport(context.Background(), adapter.Session{ID: id, Agent: "opencode"}, adapter.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := a.Export(context.Background(), plan, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExportRestoreRoundTrip(t *testing.T) {
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user"}
	archive := exportOne(t, src, "ses_fixture001")

	// The portable document must not carry the credential token.
	if bytes.Contains(archive, []byte("synthetic-not-a-real-token")) {
		t.Fatal("export leaked a credential value")
	}
	// Paths must be normalised to tokens, not the source home.
	if bytes.Contains(archive, []byte("/Users/fixture-user")) {
		t.Fatal("export left an absolute source path")
	}
	if !bytes.Contains(archive, []byte("${HOME}")) && !bytes.Contains(archive, []byte("${REPO:")) {
		t.Fatal("export did not tokenise paths")
	}

	destRoot := schemaOnlyStore(t, macosSeed)
	dest := &Adapter{Root: destRoot, Home: "/Users/other-user"}
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "", archive)

	got, err := dest.readSessionDocument(context.Background(), filepath.Join(destRoot, DatabaseName), "ses_fixture001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || len(got.Parts) != 2 {
		t.Fatalf("restored messages=%d parts=%d", len(got.Messages), len(got.Parts))
	}
	// The destination home is different, so the directory denormalises onto it.
	dbDir := destSessionDirectory(t, destRoot, "ses_fixture001")
	if dbDir != "/Users/other-user/code/demo" {
		t.Fatalf("restored directory = %q", dbDir)
	}
}

// restore drives Restore with a RestorePlan analogous to what the CLI builds.
func restore(t *testing.T, a *Adapter, archivePath, sourceID, forkID string, archive []byte) {
	backupRoot := t.TempDir()
	t.Helper()
	dbPath := filepath.Join(a.Root, DatabaseName)
	destinationRelative := archivePath
	destinationID := sourceID
	if forkID != "" {
		destinationID = forkID
		destinationRelative = "sessions/" + forkID + ".json"
	}
	plan := adapter.RestorePlan{
		Session: adapter.Session{
			ID: destinationID, Agent: "opencode",
			Path: dbPath, RelativePath: destinationRelative,
		},
		Files:           []string{dbPath},
		BackupRoot:      backupRoot,
		ArchivePath:     archivePath,
		SourceSessionID: sourceID,
	}
	if err := a.Restore(context.Background(), plan, bytes.NewReader(archive)); err != nil {
		t.Fatalf("restore: %v", err)
	}
}

func destSessionDirectory(t *testing.T, root, id string) string {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(root, DatabaseName)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var dir string
	if err := db.QueryRow(`SELECT directory FROM session WHERE id = ?`, id).Scan(&dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCrossOSRemapping(t *testing.T) {
	// Export on macOS, restore on Windows: the flagship multi-device case.
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user", GOOS: "darwin"}
	archive := exportOne(t, src, "ses_fixture001")

	destRoot := schemaOnlyStore(t, windowsSeed)
	dest := &Adapter{Root: destRoot, Home: `C:\Users\fixture-user`, GOOS: "windows"}
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "", archive)

	dir := destSessionDirectory(t, destRoot, "ses_fixture001")
	if dir != `C:\Users\fixture-user\code\demo` {
		t.Fatalf("windows directory = %q", dir)
	}
	// The assistant message's embedded path must also land on Windows.
	got, err := dest.readSessionDocument(context.Background(), filepath.Join(destRoot, DatabaseName), "ses_fixture001")
	if err != nil {
		t.Fatal(err)
	}
	// readSessionDocument re-normalises through the windows mapper, so tokens
	// come back; assert the stored blob directly instead.
	raw := storedMessageData(t, destRoot, "msg_fixtureasst001")
	if !strings.Contains(raw, `C:\\Users\\fixture-user\\code\\demo`) {
		t.Fatalf("assistant path not remapped to windows: %s", raw)
	}
	_ = got
}

func storedMessageData(t *testing.T, root, id string) string {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(root, DatabaseName)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var data string
	if err := db.QueryRow(`SELECT data FROM message WHERE id = ?`, id).Scan(&data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestForkKeepBoth(t *testing.T) {
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user"}
	archive := exportOne(t, src, "ses_fixture001")

	// Destination already holds the original session; a fork must land beside it.
	destRoot := hydrateStore(t, macosSeed)
	dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user"}
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "ses_fork0001", archive)

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(destRoot, DatabaseName)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var sessions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM session WHERE id IN ('ses_fixture001','ses_fork0001')`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 2 {
		t.Fatalf("expected original and fork, got %d", sessions)
	}
	// Original messages are untouched; the fork owns its own message ids.
	var origMsgs, forkMsgs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE session_id = 'ses_fixture001'`).Scan(&origMsgs); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE session_id = 'ses_fork0001'`).Scan(&forkMsgs); err != nil {
		t.Fatal(err)
	}
	if origMsgs != 2 || forkMsgs != 2 {
		t.Fatalf("orig=%d fork=%d", origMsgs, forkMsgs)
	}
	// The fork's parts reference the fork's derived message ids, not the originals.
	var danglingParts int
	if err := db.QueryRow(`
SELECT COUNT(*) FROM part p
 WHERE p.session_id = 'ses_fork0001'
   AND NOT EXISTS (SELECT 1 FROM message m WHERE m.id = p.message_id AND m.session_id = 'ses_fork0001')`).Scan(&danglingParts); err != nil {
		t.Fatal(err)
	}
	if danglingParts != 0 {
		t.Fatalf("fork has %d parts with no fork message", danglingParts)
	}
	// The fork is self-contained: its assistant message answers the fork's own
	// user message, not the original session's.
	forkAsst := derivedID("msg", "ses_fork0001", "msg_fixtureasst001")
	forkUser := derivedID("msg", "ses_fork0001", "msg_fixtureuser001")
	var asstData string
	if err := db.QueryRow(`SELECT data FROM message WHERE id = ?`, forkAsst).Scan(&asstData); err != nil {
		t.Fatal(err)
	}
	var asst struct {
		ParentID string `json:"parentID"`
	}
	if err := json.Unmarshal([]byte(asstData), &asst); err != nil {
		t.Fatal(err)
	}
	if asst.ParentID != forkUser {
		t.Fatalf("fork assistant parentID = %q, want fork user %q", asst.ParentID, forkUser)
	}
	var crossRefs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE session_id = 'ses_fork0001' AND data LIKE '%msg_fixture%'`).Scan(&crossRefs); err != nil {
		t.Fatal(err)
	}
	if crossRefs != 0 {
		t.Fatalf("%d fork messages still reference original message ids", crossRefs)
	}
}

func TestRewriteIDRefs(t *testing.T) {
	idMap := map[string]string{"msg_a": "msg_fork_a", "prt_a": "prt_fork_a"}
	tests := []struct {
		name string
		in   string
		keys map[string]bool
		want string
	}{
		{"parent and own id", `{"id":"msg_a","parentID":"msg_a","role":"assistant"}`, messageIDKeys, `{"id":"msg_fork_a","parentID":"msg_fork_a","role":"assistant"}`},
		{"part message ref", `{"messageID":"msg_a","type":"text"}`, messageIDKeys, `{"messageID":"msg_fork_a","type":"text"}`},
		{"part own id only", `{"id":"prt_a","messageID":"msg_a"}`, partIDKeys, `{"id":"prt_fork_a","messageID":"msg_a"}`},
		{"prose is untouched", `{"text":"see msg_a","id":"other"}`, messageIDKeys, `{"text":"see msg_a","id":"other"}`},
		{"unknown id untouched", `{"parentID":"msg_z"}`, messageIDKeys, `{"parentID":"msg_z"}`},
		{"unparseable verbatim", `{not json`, messageIDKeys, `{not json`},
		{"nested ids are not identity", `{"meta":{"id":"msg_a"}}`, messageIDKeys, `{"meta":{"id":"msg_a"}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(rewriteIDRefs(json.RawMessage(tc.in), idMap, tc.keys))
			if got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestArchiveSessionID(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"sessions/ses_x.json", "ses_x", false},
		{"sessions/.json", "", true},
		{"sessions/ .json", "", true},
		{`sessions\ses_x.json`, "", true},
		{"sessions/../ses_x.json", "", true},
		{"sessions/sub/ses_x.json", "", true},
		{"", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := archiveSessionID(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("id=%q want %q", got, tc.want)
			}
		})
	}
}

// TestRestoreRefusesEmptyOrMismatchedSessionID pins the fail-closed contract:
// a document with no session id, or one whose id is not the planned source,
// never reaches the vendor store.
func TestRestoreRefusesEmptyOrMismatchedSessionID(t *testing.T) {
	tests := []struct {
		name      string
		docID     string
		planID    string
		sourceID  string
		wantInErr string
	}{
		{"empty everywhere", "", "", "", "no session id"},
		{"empty destination", "ses_x", "", "ses_x", "destination session id"},
		{"document is another session", "ses_other", "ses_x", "ses_x", "not the planned"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			destRoot := schemaOnlyStore(t, macosSeed)
			dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user"}
			var buf bytes.Buffer
			body := fmt.Sprintf(`{"schema":%q,"session":{"id":%q,"project_id":"","slug":"s","directory":"/tmp","title":"t","version":"1","time_created":1,"time_updated":1},"messages":[],"parts":[]}`, exportSchema, tc.docID)
			writeTarEntry(t, &buf, "sessions/x.json", []byte(body))
			dbPath := filepath.Join(destRoot, DatabaseName)
			plan := adapter.RestorePlan{
				Session:         adapter.Session{ID: tc.planID, Agent: "opencode", Path: dbPath, RelativePath: "sessions/x.json"},
				Files:           []string{dbPath},
				BackupRoot:      t.TempDir(),
				ArchivePath:     "sessions/x.json",
				SourceSessionID: tc.sourceID,
			}
			err := dest.Restore(context.Background(), plan, &buf)
			if err == nil || !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("expected refusal containing %q, got %v", tc.wantInErr, err)
			}
			db, oerr := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
			if oerr != nil {
				t.Fatal(oerr)
			}
			defer func() { _ = db.Close() }()
			var rows int
			if err := db.QueryRow(`SELECT COUNT(*) FROM session`).Scan(&rows); err != nil {
				t.Fatal(err)
			}
			if rows != 0 {
				t.Fatalf("refused restore still wrote %d session rows", rows)
			}
		})
	}
}

func TestForkIsIdempotent(t *testing.T) {
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user"}
	archive := exportOne(t, src, "ses_fixture001")
	destRoot := hydrateStore(t, macosSeed)
	dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user"}
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "ses_fork0001", archive)
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "ses_fork0001", archive)

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(destRoot, DatabaseName)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var forkMsgs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE session_id = 'ses_fork0001'`).Scan(&forkMsgs); err != nil {
		t.Fatal(err)
	}
	if forkMsgs != 2 {
		t.Fatalf("re-restore duplicated fork messages: %d", forkMsgs)
	}
}

func TestSessionRevisionStableAndDeviceIndependent(t *testing.T) {
	macRoot := hydrateStore(t, macosSeed)
	winRoot := hydrateStore(t, windowsSeed)
	mac := &Adapter{Root: macRoot, Home: "/Users/fixture-user", GOOS: "darwin"}
	win := &Adapter{Root: winRoot, Home: `C:\Users\fixture-user`, GOOS: "windows"}

	s := adapter.Session{ID: "ses_fixture001"}
	r1, err := mac.SessionRevision(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := mac.SessionRevision(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Fatal("revision not stable across two reads")
	}
	rw, err := win.SessionRevision(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != rw {
		t.Fatalf("revision differs across devices: %s vs %s", r1, rw)
	}
}

// TestSessionRevisionIgnoresDestinationProjectTimes pins that a pulled session
// has the same revision as its source even though the destination keeps its
// own project-row timestamps; otherwise every pull would look like a local edit.
func TestSessionRevisionIgnoresDestinationProjectTimes(t *testing.T) {
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user"}
	archive := exportOne(t, src, "ses_fixture001")
	srcRev, err := src.SessionRevision(context.Background(), adapter.Session{ID: "ses_fixture001"})
	if err != nil {
		t.Fatal(err)
	}

	destRoot := schemaOnlyStore(t, macosSeed)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(destRoot, DatabaseName)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO project VALUES ('proj_fixture','/Users/fixture-user/code/demo','demo','[]',1,2)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user"}
	restore(t, dest, "sessions/ses_fixture001.json", "ses_fixture001", "", archive)
	destRev, err := dest.SessionRevision(context.Background(), adapter.Session{ID: "ses_fixture001"})
	if err != nil {
		t.Fatal(err)
	}
	if srcRev != destRev {
		t.Fatalf("revision changed across pull: src=%s dest=%s", srcRev, destRev)
	}
}

func TestUnsafeSnapshotPathRefused(t *testing.T) {
	root := hydrateStore(t, macosSeed)
	a := &Adapter{Root: root, Home: "/Users/fixture-user"}
	for _, bad := range []string{"../escape.json", "sessions/../../etc/passwd", "other/x.json", "sessions/x.jsonl"} {
		_, err := a.PlanRestore(context.Background(), adapter.Snapshot{SessionID: "x", RelativePath: bad}, adapter.RestoreOptions{})
		if err == nil {
			t.Fatalf("accepted unsafe path %q", bad)
		}
	}
}

func TestRestoreRefusesUninitialisedStore(t *testing.T) {
	srcRoot := hydrateStore(t, macosSeed)
	src := &Adapter{Root: srcRoot, Home: "/Users/fixture-user"}
	archive := exportOne(t, src, "ses_fixture001")

	destRoot := t.TempDir() // no opencode.db
	dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user", ForceCompat: adapter.CompatibilitySupported}
	plan := adapter.RestorePlan{
		Session:     adapter.Session{ID: "ses_fixture001", Agent: "opencode", Path: filepath.Join(destRoot, DatabaseName), RelativePath: "sessions/ses_fixture001.json"},
		Files:       []string{filepath.Join(destRoot, DatabaseName)},
		ArchivePath: "sessions/ses_fixture001.json",
	}
	err := dest.Restore(context.Background(), plan, bytes.NewReader(archive))
	if err == nil || !strings.Contains(err.Error(), "not initialised") {
		t.Fatalf("expected uninitialised refusal, got %v", err)
	}
}

func TestRestoreRejectsWrongSchemaArchive(t *testing.T) {
	destRoot := schemaOnlyStore(t, macosSeed)
	dest := &Adapter{Root: destRoot, Home: "/Users/fixture-user"}
	// A tar with a single JSON entry carrying the wrong schema.
	var buf bytes.Buffer
	writeTarEntry(t, &buf, "sessions/ses_x.json", []byte(`{"schema":"bogus/9"}`))
	plan := adapter.RestorePlan{
		Session:     adapter.Session{ID: "ses_x", Agent: "opencode", Path: filepath.Join(destRoot, DatabaseName), RelativePath: "sessions/ses_x.json"},
		Files:       []string{filepath.Join(destRoot, DatabaseName)},
		ArchivePath: "sessions/ses_x.json",
	}
	err := dest.Restore(context.Background(), plan, &buf)
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema rejection, got %v", err)
	}
}

func TestExclusionsNameCredentialTables(t *testing.T) {
	a := &Adapter{}
	var haveCred bool
	for _, ex := range a.Exclusions() {
		if strings.Contains(ex.Pattern, "credential") {
			haveCred = true
		}
	}
	if !haveCred {
		t.Fatal("exclusions do not name the credential table")
	}
}

func writeTarEntry(t *testing.T, w *bytes.Buffer, name string, body []byte) {
	t.Helper()
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestCheckpointedCopyStaysBesideStore pins the working copy to the store's own
// directory: the final os.Rename cannot cross filesystems, so a copy under the
// system temp directory would fail after the backup had been taken.
func TestCheckpointedCopyStaysBesideStore(t *testing.T) {
	root := hydrateStore(t, macosSeed)
	store := filepath.Join(root, DatabaseName)
	working, cleanup, err := checkpointedCopy(store)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(working)) != root {
		t.Fatalf("working copy %q is not staged beside %q", working, store)
	}
	if filepath.Dir(working) == root {
		t.Fatalf("working copy %q collides with the live store's directory entries", working)
	}
	cleanup()
	if _, err := os.Stat(filepath.Dir(working)); !os.IsNotExist(err) {
		t.Fatalf("cleanup left %q behind (err=%v)", filepath.Dir(working), err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".reinstate-opencode-restore-") {
			t.Fatalf("stale working directory %q left beside the store", e.Name())
		}
	}
}

// TestDenormalizerLeavesUntokenisedPathsAlone guards that a path outside every
// known root (no ${HOME}/${REPO:} token) is restored verbatim rather than
// having its separators flipped into a path that exists nowhere.
func TestDenormalizerLeavesUntokenisedPathsAlone(t *testing.T) {
	tests := []struct {
		name string
		goos string
		home string
		in   string
		want string
	}{
		{"windows keeps posix path", "windows", `C:\Users\fixture-user`, "/opt/tools", "/opt/tools"},
		{"darwin keeps windows path", "darwin", "/Users/fixture-user", `D:\tools`, `D:\tools`},
		{"windows home token", "windows", `C:\Users\fixture-user`, "${HOME}/code/demo", `C:\Users\fixture-user\code\demo`},
		{"darwin home token", "darwin", "/Users/fixture-user", "${HOME}/code/demo", "/Users/fixture-user/code/demo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Adapter{Root: t.TempDir(), Home: tt.home, GOOS: tt.goos}
			if got := a.denormalizer()(tt.in); got != tt.want {
				t.Fatalf("denormalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestRewriteJSONPathsKeepsLargeIntegers proves a message body survives the
// path rewrite with integers above 2^53 intact instead of rounded via float64.
func TestRewriteJSONPathsKeepsLargeIntegers(t *testing.T) {
	in := []byte(`{"id":9007199254740993,"path":{"cwd":"/Users/fixture-user/code/demo"}}`)
	out := rewriteJSONPaths(in, func(string) string { return "${HOME}/code/demo" })
	if !strings.Contains(string(out), `9007199254740993`) {
		t.Fatalf("large integer lost precision: %s", out)
	}
	if !strings.Contains(string(out), `"cwd":"${HOME}/code/demo"`) {
		t.Fatalf("path not rewritten: %s", out)
	}
}
