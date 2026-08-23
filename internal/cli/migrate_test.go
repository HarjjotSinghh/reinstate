package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// migrateJourney extends the locker journey with what leaving needs: a
// cancellable context (to interrupt a run), a scripted stdin (to answer the
// offers), the fast envelope codec, and a disk-backed destination shared
// with a second, BYO-only home.
type migrateJourney struct {
	*lockerJourney
	ctx     context.Context
	stdin   io.Reader
	destDir string
	codec   *fastAgeEnvelopeCodec
}

func newMigrateJourney(t *testing.T) *migrateJourney {
	t.Helper()
	j := &migrateJourney{lockerJourney: newLockerJourney(t), ctx: context.Background(), destDir: t.TempDir(), codec: &fastAgeEnvelopeCodec{}}
	return j
}

// runIn drives the CLI for one home. The destination passphrase travels on
// REINSTATE_PASSPHRASE_FD; the destination bucket is the disk-backed memory
// store at destDir (REINSTATE_BACKEND=memory only applies to the
// destination of a migration and to a BYO home).
func (j *migrateJourney) runIn(home string, memoryBackend bool, passphrase string, args ...string) (stdout, stderr string, code int) {
	j.t.Helper()
	j.t.Setenv("REINSTATE_HOME", home)
	if memoryBackend {
		j.t.Setenv("REINSTATE_BACKEND", "memory")
		j.t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", j.destDir)
	} else {
		j.t.Setenv("REINSTATE_BACKEND", "")
	}
	if passphrase != "" {
		f, err := os.CreateTemp(j.t.TempDir(), "passphrase-*")
		if err != nil {
			j.t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		_, _ = f.WriteString(passphrase + "\n")
		_, _ = f.Seek(0, 0)
		j.t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.FormatUint(uint64(f.Fd()), 10))
	} else {
		j.t.Setenv("REINSTATE_PASSPHRASE_FD", "")
	}
	var out, errb bytes.Buffer
	stdin := j.stdin
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	code = Execute(Options{
		Context: j.ctx, Name: "rein", Stdout: &out, Stderr: &errb, Stdin: stdin, Args: args,
		AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		DeviceTokenStore:    j.tokens,
		DeviceSecrets:       j.secrets,
		EnvelopeCodec:       j.codec,
		OpenBrowser: func(u string) error {
			resp, err := http.Get(u)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		},
		LoginPollSleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
		DeviceName:     "laptop",
		RecoveryCodePrompt: func(prompt string) ([]byte, error) {
			if !strings.Contains(prompt, "Re-enter") {
				return nil, errors.New("unexpected prompt " + prompt)
			}
			j.shownCode = recoveryCodePattern.FindString(errb.String())
			if j.shownCode == "" {
				return nil, errors.New("recovery code was not shown before the confirmation prompt")
			}
			return []byte(j.shownCode), nil
		},
	})
	return out.String(), errb.String(), code
}

// seedLocker signs in, initialises for Hop, creates the root key, plants
// three Claude sessions (one with a Windows-shaped project directory) and
// pushes them, then makes the locker read-only as a lapsed account's is.
func (j *migrateJourney) seedLocker(t *testing.T) (project string, sessions map[string][]byte) {
	t.Helper()
	project = writeClaudeFixture(t)
	root := filepath.Join(os.Getenv("HOME"), ".claude", "projects", claudeProjectDirectoryForTest(project))
	sessions = map[string][]byte{}
	first, _ := os.ReadFile(filepath.Join(root, "session-locker.jsonl"))
	sessions["session-locker"] = first
	for name, text := range map[string]string{
		"session-two":     "second synthetic session",
		"session-windows": `windows shaped: C:\\Users\\dev\\Projects\\locker-source`,
	} {
		meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
		content := append(meta, '\n')
		content = append(content, []byte(`{"type":"user","message":{"content":"`+text+`"}}`+"\n")...)
		if err := os.WriteFile(filepath.Join(root, name+".jsonl"), content, 0o600); err != nil {
			t.Fatal(err)
		}
		sessions[name] = content
	}
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := j.runIn(j.home, false, "", args...); code != ExitOK {
			t.Fatalf("%v exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	j.plane.s3.Mu.Lock()
	j.plane.s3.ReadOnly = true
	j.plane.s3.Mu.Unlock()
	return project, sessions
}

func lockerRequestsSince(j *migrateJourney, from int) []string {
	log := j.plane.s3.RequestLog()
	return log[from:]
}

func destinationObjects(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	disk, err := memory.NewDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	objects, err := disk.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, o := range objects {
		rc, _, err := disk.Get(context.Background(), o.Key)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(rc)
		_ = rc.Close()
		out[o.Key] = raw
	}
	return out
}

// TestMigrateJourneyLeaveHopForBYO is the primary-seam journey for leaving:
// a read-only locker is migrated to a bucket under a new passphrase, the
// device switches to it and forgets its sign-in, and a second device that
// never had a Hop account pulls and resumes the history from the bucket.
func TestMigrateJourneyLeaveHopForBYO(t *testing.T) {
	j := newMigrateJourney(t)
	project, sessions := j.seedLocker(t)
	mark := len(j.plane.s3.RequestLog())

	// Nothing to migrate on a BYO profile, and the destination is required.
	_, errb, code := j.runIn(j.home, true, "new-byo-passphrase", "sync", "migrate", "--to", "s3")
	if code != ExitUsage || !strings.Contains(errb, "--to must be byo") {
		t.Fatalf("--to s3: exit=%d err=%q", code, errb)
	}
	_, errb, code = j.runIn(j.home, true, "new-byo-passphrase", "sync", "migrate", "--to", "byo", "--json", "--switch")
	if code != ExitUsage || !strings.Contains(errb, "endpoint and a bucket") {
		t.Fatalf("no destination: exit=%d err=%q", code, errb)
	}
	if len(j.plane.s3.RequestLog()) != mark {
		t.Fatal("a refused migration touched the locker")
	}

	// Interactive run: answer the two offers with yes.
	j.stdin = strings.NewReader("y\ny\n")
	out, errb, code := j.runIn(j.home, true, "new-byo-passphrase", "sync", "migrate", "--to", "byo", "--endpoint", "https://byo.example.test", "--bucket", "my-own-bucket")
	if code != ExitOK {
		t.Fatalf("migrate exit=%d out=%q err=%q", code, out, errb)
	}
	for _, want := range []string{"Migrated 3 snapshots", "3 sessions to my-own-bucket at https://byo.example.test under profiles/", "root key was not written", "This device now syncs to the destination", "Hop sign-in was forgotten"} {
		if !strings.Contains(out, want) {
			t.Fatalf("migrate output %q missing %q", out, want)
		}
	}
	for _, want := range []string{"[1/3]", "[3/3]", "manifest written and verified (3 snapshots"} {
		if !strings.Contains(errb, want) {
			t.Fatalf("progress %q missing %q", errb, want)
		}
	}
	// The locker was only read: every request after seeding is a GET, HEAD,
	// or LIST, and the control plane saw no first-push report.
	for _, req := range lockerRequestsSince(j, mark) {
		if strings.HasPrefix(req, "PUT ") || strings.HasPrefix(req, "DELETE ") {
			t.Fatalf("migration wrote to the locker: %s", req)
		}
	}
	if j.plane.firstPushes != 1 {
		t.Fatalf("first push reported %d times", j.plane.firstPushes)
	}

	// The device switched: BYO config under a new profile, previous config
	// backed up, sign-in forgotten, no migration state left behind.
	cfg, err := config.LoadConfig(j.home)
	if err != nil {
		t.Fatal(err)
	}
	profileID := strings.TrimSpace(strings.Split(strings.Split(out, "profile_id=")[1], " ")[0])
	if cfg.Storage.Type != schema.StorageS3 || cfg.Storage.Bucket != "my-own-bucket" || cfg.Storage.Endpoint != "https://byo.example.test" || cfg.Storage.Prefix != "profiles/"+profileID || cfg.Encryption.Type != schema.EncryptionPassphrase || cfg.ProfileID != profileID || !cfg.RemoteProfileRequired || len(cfg.Projects) != 1 {
		t.Fatalf("switched config %+v", cfg)
	}
	raw, _ := os.ReadFile(filepath.Join(j.home, "config.toml"))
	for _, secret := range []string{"new-byo-passphrase", "hop_sess", "AKIA"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("config.toml holds %q", secret)
		}
	}
	if backups, _ := filepath.Glob(filepath.Join(j.home, "backups", "*migrate-byo*", "config.toml")); len(backups) != 1 {
		t.Fatalf("hosted config backup: %v", backups)
	}
	if _, err := j.tokens.GetDeviceToken(); err == nil {
		t.Fatal("device token survived --forget")
	}
	if _, err := os.Stat(filepath.Join(j.home, migrateStateFile)); !os.IsNotExist(err) {
		t.Fatalf("migration state left behind: %v", err)
	}

	// The destination holds only passphrase-sealed envelopes under the new
	// profile: no keyring, no X25519 stanza, none of the locker's root key.
	objects := destinationObjects(t, j.destDir)
	if len(objects) != 4 {
		t.Fatalf("destination objects: %d", len(objects))
	}
	deviceSecret, _ := j.secrets.GetSecret(deviceSecretRef("acct-1", "dev-sess-1"))
	for key, body := range objects {
		if !strings.HasPrefix(key, "profiles/"+profileID+"/") || strings.Contains(key, "keyring") {
			t.Fatalf("unexpected destination object %s", key)
		}
		header, _, _ := bytes.Cut(body, []byte("\n---"))
		if !bytes.Contains(header, []byte("-> scrypt")) || bytes.Contains(header, []byte("X25519")) {
			t.Fatalf("%s is not sealed to the passphrase alone", key)
		}
		for name, content := range sessions {
			if bytes.Contains(body, content) {
				t.Fatalf("plaintext of %s in %s", name, key)
			}
		}
		if len(deviceSecret) != 0 && bytes.Contains(body, bytes.TrimSpace(deviceSecret)) {
			t.Fatalf("device key in %s", key)
		}
	}

	// A second device that never signed in joins the BYO profile and pulls
	// every session back byte for byte, then resumes one.
	secondHome := t.TempDir()
	secondUser := t.TempDir()
	t.Setenv("HOME", secondUser)
	t.Setenv("USERPROFILE", secondUser)
	if err := os.MkdirAll(filepath.Join(secondUser, ".claude", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(secondUser, ".claude", "version"), []byte("2.1.219\n"), 0o600)
	secondProject := filepath.Join(secondUser, "Projects", "locker-source")
	_ = os.MkdirAll(secondProject, 0o700)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA-second")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "secret-second")
	out, errb, code = j.runIn(secondHome, true, "", "init", "--yes", "--endpoint", "https://byo.example.test", "--bucket", "my-own-bucket", "--profile-id", profileID, "--project", "local/locker="+secondProject)
	if code != ExitOK {
		t.Fatalf("second init exit=%d out=%q err=%q", code, out, errb)
	}
	// The wrong passphrase opens nothing.
	if _, errb, code := j.runIn(secondHome, true, "wrong-passphrase", "pull", "--all"); code == ExitOK || !strings.Contains(errb, "decrypt") {
		t.Fatalf("pull with wrong passphrase: exit=%d err=%q", code, errb)
	}
	out, errb, code = j.runIn(secondHome, true, "new-byo-passphrase", "pull", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("second pull exit=%d out=%q err=%q", code, out, errb)
	}
	var pulled struct {
		Plans []struct {
			SessionID    string   `json:"session_id"`
			Destinations []string `json:"destinations"`
		} `json:"plans"`
	}
	if err := json.Unmarshal([]byte(out), &pulled); err != nil || len(pulled.Plans) != 3 {
		t.Fatalf("pull output %q: %v", out, err)
	}
	for _, r := range pulled.Plans {
		if len(r.Destinations) != 1 {
			t.Fatalf("restored %+v", r)
		}
		// Restore remaps the meta line's cwd to this device's project root
		// (path remapping is first-class) and re-serialises each line; the
		// conversation itself must be the same records.
		got, err := os.ReadFile(r.Destinations[0])
		if err != nil || !bytes.Contains(got, []byte(secondProject)) {
			t.Fatalf("restored %s (err=%v):\n%s", r.SessionID, err, got)
		}
		gotLines, wantLines := jsonLines(t, got), jsonLines(t, sessions[r.SessionID])
		if len(gotLines) != len(wantLines) || len(gotLines) != 2 {
			t.Fatalf("restored %s has %d lines, want %d", r.SessionID, len(gotLines), len(wantLines))
		}
		if gotLines[1] != wantLines[1] {
			t.Fatalf("restored %s conversation differs:\n%s\n%s", r.SessionID, gotLines[1], wantLines[1])
		}
		if !strings.HasPrefix(r.Destinations[0], secondUser) {
			t.Fatalf("restored outside the second device's home: %s", r.Destinations[0])
		}
	}
	out, errb, code = j.runIn(secondHome, true, "", "resume", "claude:session-windows", "--dry-run", "--json")
	if code != ExitOK || !strings.Contains(out, "session-windows") {
		t.Fatalf("resume on the second device exit=%d out=%q err=%q", code, out, errb)
	}
	_ = project
}

// TestMigrateJourneyInterruptedRunResumes cancels the migration after the
// first snapshot reached the destination and reruns it: the finished
// snapshot is skipped, nothing is written twice, and the manifest appears
// only at the end. A --keep-hop-config run leaves the device on the locker.
func TestMigrateJourneyInterruptedRunResumes(t *testing.T) {
	j := newMigrateJourney(t)
	j.seedLocker(t)
	ctx, cancel := context.WithCancel(context.Background())
	j.ctx = ctx
	var snapshotReads int
	j.plane.s3.Mu.Lock()
	j.plane.s3.Hook = func(n int) {
		if strings.HasPrefix(j.plane.s3.Requests[n-1], "GET snapshots/") {
			snapshotReads++
			if snapshotReads == 2 {
				cancel()
			}
		}
	}
	j.plane.s3.Mu.Unlock()
	env := []string{"sync", "migrate", "--to", "byo", "--endpoint", "https://byo.example.test", "--bucket", "my-own-bucket", "--keep-hop-config", "--json"}
	out, errb, code := j.runIn(j.home, true, "resume-passphrase", env...)
	if code == ExitOK || !strings.Contains(errb, "rerun the same command to resume") {
		t.Fatalf("interrupted run exit=%d out=%q err=%q", code, out, errb)
	}
	before := destinationObjects(t, j.destDir)
	if len(before) != 1 {
		t.Fatalf("destination after interruption: %d objects", len(before))
	}
	for key := range before {
		if strings.HasSuffix(key, "manifest.age") {
			t.Fatal("manifest written before every snapshot was verified")
		}
	}
	stateRaw, err := os.ReadFile(filepath.Join(j.home, migrateStateFile))
	if err != nil || strings.Contains(string(stateRaw), "resume-passphrase") {
		t.Fatalf("migration state: %q err=%v", stateRaw, err)
	}
	var st migrateState
	_ = json.Unmarshal(stateRaw, &st)
	if len(st.Done) != 1 || st.Destination.Bucket != "my-own-bucket" {
		t.Fatalf("state %+v", st)
	}

	// A different destination cannot be mixed into an in-progress migration.
	j.ctx = context.Background()
	j.plane.s3.Mu.Lock()
	j.plane.s3.Hook = nil
	j.plane.s3.Mu.Unlock()
	if _, errb, code := j.runIn(j.home, true, "resume-passphrase", "sync", "migrate", "--to", "byo", "--endpoint", "https://byo.example.test", "--bucket", "other-bucket", "--keep-hop-config", "--json"); code != ExitUsage || !strings.Contains(errb, "in progress") {
		t.Fatalf("other bucket: exit=%d err=%q", code, errb)
	}
	// Another passphrase cannot continue either: the finished snapshot is
	// re-read and refuses to open.
	if _, errb, code := j.runIn(j.home, true, "another-passphrase", env...); code == ExitOK || !strings.Contains(errb, "verified by an earlier run") {
		t.Fatalf("resume under another passphrase: exit=%d err=%q", code, errb)
	}

	out, errb, code = j.runIn(j.home, true, "resume-passphrase", env...)
	if code != ExitOK {
		t.Fatalf("resumed run exit=%d out=%q err=%q", code, out, errb)
	}
	var result struct {
		Migrated struct {
			Snapshots, Written, Verified, Skipped int
		} `json:"migrated"`
		ProfileID string `json:"profile_id"`
		Switched  bool   `json:"switched"`
		ForgotHop bool   `json:"forgot_hop"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json %q: %v", out, err)
	}
	if result.Migrated.Snapshots != 3 || result.Migrated.Skipped != 1 || result.Migrated.Written != 2 || result.Migrated.Verified != 0 || result.Switched || result.ForgotHop {
		t.Fatalf("resumed result %+v", result)
	}
	after := destinationObjects(t, j.destDir)
	if len(after) != 4 {
		t.Fatalf("destination after resume: %d objects", len(after))
	}
	for key, body := range before {
		if !bytes.Equal(after[key], body) {
			t.Fatalf("%s was rewritten on resume", key)
		}
	}
	cfg, _ := config.LoadConfig(j.home)
	if cfg.Storage.Type != schema.StorageHop {
		t.Fatalf("--keep-hop-config switched the config: %+v", cfg.Storage)
	}
	if _, err := j.tokens.GetDeviceToken(); err != nil {
		t.Fatal("device token lost without --forget-hop")
	}
	stateRaw, _ = os.ReadFile(filepath.Join(j.home, migrateStateFile))
	if !strings.Contains(string(stateRaw), "completed_at") {
		t.Fatalf("finished migration not recorded: %s", stateRaw)
	}
	// A finished migration re-run keeps the profile and writes nothing
	// twice; a second copy under a new profile would double the bucket.
	out, errb, code = j.runIn(j.home, true, "resume-passphrase", env...)
	if code != ExitOK {
		t.Fatalf("rerun exit=%d out=%q err=%q", code, out, errb)
	}
	previous := result.ProfileID
	_ = json.Unmarshal([]byte(out), &result)
	if result.Migrated.Skipped != 3 || result.Migrated.Written != 0 || result.ProfileID != previous {
		t.Fatalf("rerun result %+v (previous profile %s)", result, previous)
	}
	if again := destinationObjects(t, j.destDir); len(again) != 4 {
		t.Fatalf("destination after rerun: %d objects", len(again))
	}
}

// TestMigrateRefusesNonHopProfile covers the BYO-only home.
func TestMigrateRefusesNonHopProfile(t *testing.T) {
	j := newMigrateJourney(t)
	home := t.TempDir()
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "secret")
	if out, errb, code := j.runIn(home, true, "", "init", "--yes", "--endpoint", "https://byo.example.test", "--bucket", "b"); code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	_, errb, code := j.runIn(home, true, "p", "sync", "migrate", "--to", "byo", "--endpoint", "x", "--bucket", "y", "--switch")
	if code != ExitConfig || !strings.Contains(errb, "nothing to migrate") {
		t.Fatalf("exit=%d err=%q", code, errb)
	}
}

// jsonLines re-encodes each JSONL record canonically so key order does not
// matter.
func jsonLines(t *testing.T, raw []byte) []string {
	t.Helper()
	var out []string
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var v map[string]any
		if err := json.Unmarshal(line, &v); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		canon, _ := json.Marshal(v)
		out = append(out, string(canon))
	}
	return out
}

var _ backend.Backend = (*memory.DiskStore)(nil)
