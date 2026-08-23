package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// --- locker endpoints of the fake control plane ---

func (f *fakeControlPlane) authed(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := f.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]; !ok {
		writeFakeError(w, 401, "unknown or revoked device token")
		return false
	}
	return true
}

func (f *fakeControlPlane) lockerView() map[string]any {
	view := map[string]any{
		"endpoint": f.s3.URL(), "bucket": f.locker.bucket, "region": "auto", "prefix": "",
		"location_hint": "apac", "plan": "hop", "created_at": "2026-08-23T12:02:00Z",
		"devices": len(f.tokens),
		"usage":   map[string]any{"bytes": f.usageBytes, "objects": len(f.mints), "observed_at": "2026-08-23T12:02:00Z"},
		"quota":   map[string]any{"storage_bytes": 5 << 30, "devices": 5, "mints_per_hour": 60},
	}
	if f.locker.firstPushAt != "" {
		view["first_push_at"] = f.locker.firstPushAt
	}
	return view
}

func (f *fakeControlPlane) provisionLocker(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	f.provisions++
	if f.locker == nil {
		f.locker = &fakeLocker{bucket: f.s3.Bucket}
	}
	_ = json.NewEncoder(w).Encode(f.lockerView())
}

func (f *fakeControlPlane) lockerStatus(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	if f.locker == nil {
		writeFakeErrorCode(w, 404, "no_locker", "no locker has been provisioned for this account yet; the first push creates it")
		return
	}
	_ = json.NewEncoder(w).Encode(f.lockerView())
}

func (f *fakeControlPlane) mintCredentials(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	if f.locker == nil {
		writeFakeErrorCode(w, 404, "no_locker", "no locker has been provisioned for this account yet; call POST /v1/locker first")
		return
	}
	switch f.refuse {
	case "quota_storage":
		writeFakeErrorCode(w, 403, f.refuse, "the locker holds 5.0 GB, the hop plan's limit is 5.0 GB. Delete old snapshots or upgrade the plan")
		return
	case "quota_devices":
		writeFakeErrorCode(w, 403, f.refuse, "this account has 6 enrolled devices; the hop plan allows 5. Revoke a device or upgrade the plan")
		return
	case "quota_push_rate":
		writeFakeErrorCode(w, 429, f.refuse, "this account requested storage credentials 60 times in the last hour; the hop plan allows 60. Wait before pushing again")
		return
	case "storage_unavailable":
		writeFakeErrorCode(w, 502, f.refuse, "the storage provider could not issue credentials; try again shortly")
		return
	}
	akid := fmt.Sprintf("AKIAHOP%03d", len(f.mints)+1)
	f.mints = append(f.mints, akid)
	// Only the newest credential is accepted: an older one is expired the
	// moment its successor exists, which is how R2 behaves once a TTL ends.
	f.s3.Accept(akid)
	ttl := f.credTTL
	if ttl == 0 {
		ttl = time.Hour
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_key_id": akid, "secret_access_key": "secret-" + akid, "session_token": "session-" + akid,
		"expires_at": time.Now().Add(ttl).UTC().Format(time.RFC3339),
		"endpoint":   f.s3.URL(), "bucket": f.locker.bucket, "region": "auto",
	})
}

func (f *fakeControlPlane) firstPush(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	f.firstPushes++
	first := f.locker.firstPushAt == ""
	if first {
		f.locker.firstPushAt = "2026-08-23T12:05:00Z"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"first": first, "first_push_at": f.locker.firstPushAt})
}

func writeFakeErrorCode(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code})
}

// --- journey harness ---

// lockerJourney drives the real CLI entrypoint against the fake control
// plane and a fake S3 that only accepts the credential the plane minted
// last. Every seam is in-process; nothing leaves the loopback interface.
type lockerJourney struct {
	t       *testing.T
	plane   *fakeControlPlane
	tokens  *credentials.MemoryDeviceTokenStore
	secrets credentials.SecretStore
	home    string
	// shownCode captures the recovery code from the account init prompt.
	shownCode string
}

func newLockerJourney(t *testing.T) *lockerJourney {
	t.Helper()
	j := &lockerJourney{t: t, plane: newFakeControlPlane(t), tokens: &credentials.MemoryDeviceTokenStore{}, secrets: credentials.NewMemorySecrets(), home: t.TempDir()}
	j.plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000000test")
	t.Setenv(hopURLEnv, j.plane.srv.URL)
	t.Setenv("REINSTATE_BACKEND", "")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "")
	t.Setenv("REINSTATE_PASSPHRASE_FD", "")
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", "")
	t.Setenv("REINSTATE_HOP_LOCATION", "")
	t.Setenv("TZ", "Asia/Kolkata")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	return j
}

const hopURLEnv = "REINSTATE_HOP_URL"

func (j *lockerJourney) run(args ...string) (stdout, stderr string, code int) {
	j.t.Helper()
	j.t.Setenv("REINSTATE_HOME", j.home)
	var out, errb bytes.Buffer
	code = Execute(Options{
		Name: "rein", Stdout: &out, Stderr: &errb, Args: args,
		AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		DeviceTokenStore:    j.tokens,
		DeviceSecrets:       j.secrets,
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

// writeClaudeFixture plants one synthetic Claude session under a fresh HOME
// so `rein push` has something to upload.
func writeClaudeFixture(t *testing.T) string {
	t.Helper()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)
	project := filepath.Join(userHome, "Projects", "locker-source")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"synthetic locker journey"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-locker.jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return project
}

// TestLockerJourneyLoginInitPushStatus is the primary-seam journey for the
// hosted backend: sign in, init for Hop, generate the root key, and push.
// The first push provisions the locker and mints credentials; the push
// survives a credential expiring halfway through by minting again; the
// locker then holds only ciphertext written with the minted keys.
func TestLockerJourneyLoginInitPushStatus(t *testing.T) {
	j := newLockerJourney(t)
	project := writeClaudeFixture(t)

	// Not signed in yet: the hosted backend says so before touching anything.
	out, errb, code := j.run("hop", "status")
	if code != ExitAuthStorage || !strings.Contains(errb, "not signed in to Reinstate Hop; run rein login") {
		t.Fatalf("status before login: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, errb, code := j.run("init", "--hop"); code != ExitAuthStorage || !strings.Contains(errb, "rein login") {
		t.Fatalf("init --hop before login: exit=%d err=%q", code, errb)
	}

	if out, errb, code := j.run("login"); code != ExitOK {
		t.Fatalf("login exit=%d out=%q err=%q", code, out, errb)
	}
	if got := j.plane.hints; len(got) != 1 || got[0] != "apac" {
		t.Fatalf("location hints sent at sign-in: %v", got)
	}

	// Nothing is provisioned by signing in alone.
	out, _, code = j.run("hop", "status")
	if code != ExitOK || !strings.Contains(out, "not provisioned yet") {
		t.Fatalf("status after login: exit=%d out=%q", code, out)
	}
	if j.plane.locker != nil {
		t.Fatal("login provisioned a locker")
	}

	out, errb, code = j.run("init", "--hop", "--project", "local/locker="+project)
	if code != ExitOK {
		t.Fatalf("init --hop exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(out, "storage.type=hop") || !strings.Contains(out, "locker lk-0000000000000000000000test at "+j.plane.s3.URL()) {
		t.Fatalf("init output %q", out)
	}
	cfg, err := config.LoadConfig(j.home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Type != schema.StorageHop || cfg.Storage.Bucket != "" || cfg.Storage.CredentialRef != "" || cfg.ProfileID != "acct-1" || cfg.DeviceID != "dev-sess-1" {
		t.Fatalf("config %+v", cfg)
	}
	raw, _ := os.ReadFile(filepath.Join(j.home, "config.toml"))
	for _, secret := range []string{"hop_sess", "AKIA", "secret-"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("config.toml holds %q:\n%s", secret, raw)
		}
	}
	if j.plane.locker == nil || len(j.plane.mints) != 0 {
		t.Fatalf("init --hop must provision without minting: locker=%v mints=%v", j.plane.locker, j.plane.mints)
	}

	// A pairing code is a BYO concept; hosted profiles are joined by login.
	if _, errb, code := j.run("init", "--link"); code != ExitConfig || !strings.Contains(errb, "rein account recover") {
		t.Fatalf("init --link on hop: exit=%d err=%q", code, errb)
	}

	// The root key lives on the device; the keyring lands in the locker
	// through the first minted credential.
	out, errb, code = j.run("account", "init")
	if code != ExitOK {
		t.Fatalf("account init exit=%d out=%q err=%q", code, out, errb)
	}
	if len(j.plane.mints) != 1 {
		t.Fatalf("mints after account init: %v", j.plane.mints)
	}
	if _, _, err := j.plane.s3.Store.Get(context.Background(), "keyring.v1.json"); err != nil {
		t.Fatalf("keyring not in the locker: %v", err)
	}

	// Expire the credential mid-push: the snapshot PUT goes out on the first
	// key, everything after it on a freshly minted one.
	var expiredAt int
	j.plane.s3.Mu.Lock()
	j.plane.s3.Hook = func(n int) {
		if expiredAt == 0 && strings.HasPrefix(j.plane.s3.Requests[n-1], "PUT snapshots/") {
			expiredAt = n
			j.plane.s3.AcceptLocked()
		}
	}
	j.plane.s3.Mu.Unlock()

	out, errb, code = j.run("push", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
	}
	var pushed struct {
		Snapshots []string `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(out), &pushed); err != nil || len(pushed.Snapshots) != 1 {
		t.Fatalf("push output %q: %v", out, err)
	}
	if expiredAt == 0 {
		t.Fatal("the hook never expired the credential")
	}
	// account init minted one, the push minted one, the expiry forced a third.
	if len(j.plane.mints) != 3 {
		t.Fatalf("mints after push: %v", j.plane.mints)
	}
	// init --hop provisioned once; every later command read the locker.
	if j.plane.provisions != 1 {
		t.Fatalf("POST /v1/locker was called %d times; once is enough", j.plane.provisions)
	}
	log := j.plane.s3.RequestLog()
	if signedBy(log, "AKIAHOP002") == 0 || signedBy(log, "AKIAHOP003") == 0 {
		t.Fatalf("push did not refresh its credential:\n%s", strings.Join(log, "\n"))
	}
	for _, l := range log[expiredAt:] {
		if strings.HasSuffix(l, " as AKIAHOP002") {
			t.Fatalf("request signed with the expired key after it was refused:\n%s", strings.Join(log, "\n"))
		}
	}
	if j.plane.firstPushes != 1 || j.plane.locker.firstPushAt == "" {
		t.Fatalf("first push reported %d times", j.plane.firstPushes)
	}

	// The locker holds ciphertext only.
	objects, _ := j.plane.s3.Store.List(context.Background(), "")
	var keys []string
	for _, o := range objects {
		keys = append(keys, o.Key)
		rc, _, err := j.plane.s3.Store.Get(context.Background(), o.Key)
		if err != nil {
			t.Fatal(err)
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(rc)
		rc.Close()
		if strings.Contains(body.String(), "synthetic locker journey") {
			t.Fatalf("plaintext in the locker at %s", o.Key)
		}
	}
	if len(keys) < 3 {
		t.Fatalf("locker objects %v", keys)
	}

	// A second push is a no-op and does not re-report.
	out, errb, code = j.run("push", "--all")
	if code != ExitOK || !strings.Contains(out, "skipped 1 unchanged") {
		t.Fatalf("second push exit=%d out=%q err=%q", code, out, errb)
	}
	if j.plane.firstPushes != 1 {
		t.Fatalf("first push reported %d times", j.plane.firstPushes)
	}

	out, errb, code = j.run("hop", "status")
	if code != ExitOK {
		t.Fatalf("hop status exit=%d err=%q", code, errb)
	}
	for _, want := range []string{"Locker:   lk-0000000000000000000000test at " + j.plane.s3.URL(), "location apac", "Plan:     hop", "of 5.0 GB", "Devices:  1 of 5", "60 credential mints per hour", "first push: 2026-08-23T12:05:00Z"} {
		if !strings.Contains(out, want) {
			t.Fatalf("hop status %q missing %q", out, want)
		}
	}
	out, _, code = j.run("hop", "status", "--json")
	var status struct {
		ControlPlane string `json:"control_plane"`
		Locker       struct {
			Bucket string `json:"bucket"`
			Quota  struct {
				StorageBytes int64 `json:"storage_bytes"`
			} `json:"quota"`
		} `json:"locker"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil || code != ExitOK || status.ControlPlane != j.plane.srv.URL || status.Locker.Bucket != "lk-0000000000000000000000test" || status.Locker.Quota.StorageBytes != 5<<30 {
		t.Fatalf("hop status --json %q err=%v", out, err)
	}
	out, _, code = j.run("whoami")
	if code != ExitOK || !strings.Contains(out, "Plan:    hop (locker location apac)") {
		t.Fatalf("whoami %q", out)
	}

	// A revoked token surfaces as a sign-in problem, not a storage one.
	tok, _ := j.tokens.GetDeviceToken()
	j.plane.revoke(tok.Token)
	_, errb, code = j.run("push", "--all")
	if code != ExitAuthStorage || !strings.Contains(errb, "rejected by the control plane") || !strings.Contains(errb, "rein login") {
		t.Fatalf("push with revoked token: exit=%d err=%q", code, errb)
	}
}

// TestLockerJourneyQuotaRefusals runs the same journey up to the first push
// and has the control plane refuse the mint for each quota kind; the
// message names the quota and the fix, and the locker is left untouched.
func TestLockerJourneyQuotaRefusals(t *testing.T) {
	tests := []struct {
		code     string
		exit     int
		contains []string
	}{
		{code: "quota_storage", exit: ExitAuthStorage, contains: []string{"locker over quota (storage)", "5.0 GB", "rein hop status"}},
		{code: "quota_devices", exit: ExitAuthStorage, contains: []string{"locker over quota (devices)", "6 enrolled devices", "Revoke a device"}},
		{code: "quota_push_rate", exit: ExitAuthStorage, contains: []string{"locker over quota (push-rate)", "60 times in the last hour", "Wait before pushing"}},
		{code: "storage_unavailable", exit: ExitRuntime, contains: []string{"could not reach the storage provider"}},
	}
	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			j := newLockerJourney(t)
			writeClaudeFixture(t)
			if _, errb, code := j.run("login"); code != ExitOK {
				t.Fatalf("login exit=%d err=%q", code, errb)
			}
			if _, errb, code := j.run("init", "--hop"); code != ExitOK {
				t.Fatalf("init --hop exit=%d err=%q", code, errb)
			}
			if _, errb, code := j.run("account", "init"); code != ExitOK {
				t.Fatalf("account init exit=%d err=%q", code, errb)
			}
			before := len(j.plane.s3.RequestLog())
			j.plane.mu.Lock()
			j.plane.refuse = tc.code
			j.plane.mu.Unlock()

			out, errb, code := j.run("push", "--all")
			if code != tc.exit {
				t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
			}
			for _, want := range tc.contains {
				if !strings.Contains(errb, want) {
					t.Fatalf("push stderr %q missing %q", errb, want)
				}
			}
			if strings.Contains(errb, "operation error") || strings.Contains(errb, "failed to retrieve credentials") {
				t.Fatalf("SDK prose leaked into the message: %q", errb)
			}
			if after := len(j.plane.s3.RequestLog()); after != before {
				t.Fatalf("locker was touched while refused: %d requests", after-before)
			}
			if j.plane.firstPushes != 0 {
				t.Fatal("a refused push was reported as the first push")
			}

			// Status still answers while pushes are refused.
			j.plane.mu.Lock()
			j.plane.refuse = ""
			j.plane.mu.Unlock()
			if out, _, code := j.run("hop", "status"); code != ExitOK || !strings.Contains(out, "Plan:     hop") {
				t.Fatalf("hop status after refusal: exit=%d out=%q", code, out)
			}
		})
	}
}

// TestByoStorageIgnoresHop makes sure a BYO profile never consults the
// device token or the control plane.
func TestByoStorageIgnoresHop(t *testing.T) {
	j := newLockerJourney(t)
	writeClaudeFixture(t)
	locker := t.TempDir()
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	if _, errb, code := j.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "byo", "--yes"); code != ExitOK {
		t.Fatalf("byo init exit=%d err=%q", code, errb)
	}
	if _, errb, code := j.run("account", "init"); code != ExitOK {
		t.Fatalf("account init exit=%d err=%q", code, errb)
	}
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	if j.plane.locker != nil || len(j.plane.mints) != 0 || len(j.plane.sessions) != 0 {
		t.Fatalf("BYO profile reached the control plane: locker=%v mints=%v sessions=%d", j.plane.locker, j.plane.mints, len(j.plane.sessions))
	}
}

func signedBy(log []string, akid string) int {
	n := 0
	for _, l := range log {
		if strings.HasSuffix(l, " as "+akid) {
			n++
		}
	}
	return n
}
