package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// revoke runs `rein devices revoke <target>` on d, answering the recovery
// prompt with code.
func (d *pairDevice) revoke(target, code string) (string, string, int) {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	out, errb := &syncBuffer{}, &syncBuffer{}
	exit := d.execute(runOptions{stdout: out, stderr: errb, recovery: code}, "devices", "revoke", target)
	return out.String(), errb.String(), exit
}

// recover runs `rein account recover` on d with the recovery code.
func (d *pairDevice) recover(code string) (string, string, int) {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	out, errb := &syncBuffer{}, &syncBuffer{}
	exit := d.execute(runOptions{stdout: out, stderr: errb, recovery: code}, "account", "recover")
	return out.String(), errb.String(), exit
}

// enrol signs d in, inits it for Hop against project, and enrols it from
// the recovery code.
func (d *pairDevice) enrol(project, code string) {
	d.t.Helper()
	if _, errb, exit := d.run("login"); exit != ExitOK {
		d.t.Fatalf("%s login: %d %q", d.name, exit, errb)
	}
	if _, errb, exit := d.run("init", "--hop", "--project", "local/locker="+project); exit != ExitOK {
		d.t.Fatalf("%s init --hop: %d %q", d.name, exit, errb)
	}
	if out, errb, exit := d.recover(code); exit != ExitOK {
		d.t.Fatalf("%s recover: %d out=%q err=%q", d.name, exit, out, errb)
	}
}

// keyringState loads the keyring as stored and this device's generation
// keys, for assertions that bypass the CLI.
func (d *pairDevice) keyringState(t *testing.T, plane *fakeControlPlane) (*keyring.Keyring, map[int][]byte) {
	t.Helper()
	ctx := context.Background()
	cfg, err := config.LoadConfig(d.home)
	if err != nil {
		t.Fatal(err)
	}
	objects, err := plane.s3.Store.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var key string
	for _, meta := range objects {
		if strings.HasSuffix(meta.Key, keyring.ObjectName) {
			key = meta.Key
		}
	}
	ring, _, err := keyring.Load(ctx, plane.s3.Store, key)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := d.secrets.GetSecret(deviceSecretRef(cfg.ProfileID, cfg.DeviceID))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := age.ParseX25519Identity(strings.TrimSpace(string(secret)))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := ring.UnwrapGenerations(cfg.DeviceID, identity)
	if err != nil {
		t.Fatalf("%s cannot open the keyring: %v", d.name, err)
	}
	return ring, keys
}

func objectKeys(t *testing.T, plane *fakeControlPlane) map[string]bool {
	t.Helper()
	objects, err := plane.s3.Store.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, meta := range objects {
		keys[meta.Key] = true
	}
	return keys
}

// opensWith reports whether the object at key is an envelope the provider
// can open.
func opensWith(t *testing.T, plane *fakeControlPlane, key string, keys crypto.KeyProvider) bool {
	t.Helper()
	rc, _, err := plane.s3.Store.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	raw, _ := io.ReadAll(rc)
	var out bytes.Buffer
	return crypto.Open(bytes.NewReader(raw), &out, keys) == nil
}

func deviceID(t *testing.T, d *pairDevice) string {
	t.Helper()
	tok, err := d.tokens.GetDeviceToken()
	if err != nil {
		t.Fatal(err)
	}
	return tok.DeviceID
}

// TestRevocationJourney is the primary-seam journey for device revocation
// and key generations: A and B are enrolled; A revokes B, which starts key
// generation 2; B can neither mint credentials nor open anything pushed
// afterwards, while A reads the whole history; the recovery code enrols a
// new device after the rollover and that device reads everything too; a
// revocation racing an approval, in either order, converges on one
// keyring where the approved device holds a wrap and the revoked one does
// not.
func TestRevocationJourney(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-000000000000000000000revoke")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	// A: first device, pushes the first session.
	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	// B: joins by approval and pulls.
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	join := b.startJoin()
	if out, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("A approves B: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("B pull: exit=%d out=%q err=%q", code, out, errb)
	}
	bID := deviceID(t, b)
	_, bKeys := b.keyringState(t, plane)
	if len(bKeys) != 1 || bKeys[1] == nil {
		t.Fatalf("B holds generations %v, want only 1", bKeys)
	}
	before := objectKeys(t, plane)

	// Refusals: an unknown device, this device, a wrong recovery code.
	// None of them changes the keyring or the control plane.
	if out, errb, code := a.revoke("nobody", a.shownCode); code != ExitUsage || !strings.Contains(errb, `no device "nobody"`) {
		t.Fatalf("revoke unknown: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.revoke("macbook", a.shownCode); code != ExitUsage || !strings.Contains(errb, "is this device") {
		t.Fatalf("revoke self: exit=%d out=%q err=%q", code, out, errb)
	}
	wrongCode, err := keyring.GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	if out, errb, code := a.revoke("desktop", wrongCode); code != ExitAuthStorage || !strings.Contains(errb, "does not match") || !strings.Contains(errb, "nothing was revoked") {
		t.Fatalf("revoke with wrong code: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.revoke("desktop", "not-a-code"); code != ExitUsage {
		t.Fatalf("revoke with malformed code: exit=%d out=%q err=%q", code, out, errb)
	}
	ring, _ := a.keyringState(t, plane)
	if ring.CurrentGeneration != 1 || ring.DeviceCount() != 2 || len(plane.events) != 0 {
		t.Fatalf("a refused revocation changed state: gen=%d devices=%d events=%v", ring.CurrentGeneration, ring.DeviceCount(), plane.events)
	}
	// B's token still mints.
	if out, errb, code := b.run("hop", "status"); code != ExitOK {
		t.Fatalf("B hop status before revocation: exit=%d out=%q err=%q", code, out, errb)
	}

	// A revokes B by name, typing the code loosely.
	out, errb, code := a.revoke("Desktop", strings.ToLower(strings.ReplaceAll(a.shownCode, "-", " ")))
	if code != ExitOK || !strings.Contains(out, `revoked device "desktop"`) || !strings.Contains(out, "key generation 2 started with 1 enrolled device") {
		t.Fatalf("revoke: exit=%d out=%q err=%q", code, out, errb)
	}
	// The success message must not leave the operator believing the
	// revoked device is locked out of the bucket the moment this returns:
	// a credential it already minted keeps working until it expires.
	if !strings.Contains(out, "until it expires") || !strings.Contains(out, "up to an hour") {
		t.Fatalf("the revocation message does not name the credential window: %q", out)
	}
	if strings.Contains(out, a.shownCode) || strings.Contains(errb, a.shownCode) {
		t.Fatal("the recovery code was echoed")
	}
	plane.mu.Lock()
	events := append([]string(nil), plane.events...)
	_, bStillKnown := plane.revoked[bID]
	plane.mu.Unlock()
	if len(events) != 1 || events[0] != "device_revoked:"+bID || !bStillKnown {
		t.Fatalf("control plane events %v, revoked=%v", events, bStillKnown)
	}
	ring, aKeys := a.keyringState(t, plane)
	if ring.CurrentGeneration != 2 || len(ring.Generations) != 2 || ring.DeviceCount() != 1 || ring.HasDevice(bID) || !ring.RevokedDevice(bID) {
		t.Fatalf("keyring after revocation: gen=%d gens=%d devices=%d b=%v", ring.CurrentGeneration, len(ring.Generations), ring.DeviceCount(), ring.HasDevice(bID))
	}
	if len(aKeys) != 2 || !bytes.Equal(aKeys[1], bKeys[1]) {
		t.Fatalf("A holds generations %v after rollover", aKeys)
	}
	// Generation 1 is untouched: B's old wrap still opens it (B already
	// had that key), which is exactly why nothing new is written under it.
	if !bytes.Equal(aKeys[1], bKeys[1]) {
		t.Fatal("generation 1 was rewritten")
	}

	// Visible on both sides.
	out, _, code = a.run("devices")
	if code != ExitOK || !strings.Contains(out, "desktop") || !strings.Contains(out, ", revoked 20") || !strings.Contains(out, "holds a root-key wrap (key generation 2)") {
		t.Fatalf("A devices after revocation: exit=%d out=%q", code, out)
	}
	out, _, code = a.run("devices", "--json")
	var listed struct {
		Devices []struct {
			ID        string `json:"id"`
			RevokedAt string `json:"revoked_at"`
			InKeyring *bool  `json:"in_keyring"`
		} `json:"devices"`
		KeyGeneration int `json:"key_generation"`
	}
	_ = json.Unmarshal([]byte(out), &listed)
	if code != ExitOK || listed.KeyGeneration != 2 || len(listed.Devices) != 2 {
		t.Fatalf("A devices --json: exit=%d out=%q", code, out)
	}
	for _, d := range listed.Devices {
		if d.ID == bID && (d.RevokedAt == "" || d.InKeyring == nil || *d.InKeyring) {
			t.Fatalf("B row after revocation: %+v", d)
		}
		if d.ID != bID && (d.RevokedAt != "" || d.InKeyring == nil || !*d.InKeyring) {
			t.Fatalf("A row after revocation: %+v", d)
		}
	}
	// B: the token is refused everywhere, so nothing mints and nothing
	// pulls; the session it already pulled stays on disk.
	if out, errb, code := b.run("devices"); code != ExitAuthStorage || !strings.Contains(errb, "token was rejected") {
		t.Fatalf("B devices after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("pull", "--all", "--json"); code == ExitOK || !strings.Contains(errb, "token was rejected") {
		t.Fatalf("B pull after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("push", "--all", "--json"); code == ExitOK {
		t.Fatalf("B push after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("account", "join"); code != ExitSafety || !strings.Contains(errb, "already enrolled") {
		t.Fatalf("B join after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	if content, err := os.ReadFile(filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(filepath.Join(userHome, "Projects", "desktop-target")), "session-locker.jsonl")); err != nil || !bytes.Contains(content, []byte("synthetic locker journey")) {
		t.Fatalf("revocation touched B's local copy: %v", err)
	}

	// A pushes a new session. B's only key (generation 1) opens every
	// object written before the revocation and none written after; A
	// opens all of them and pulls its whole history.
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"written after the revocation"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-after.jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, errb, code := a.run("push", "--all", "--json"); code != ExitOK {
		t.Fatalf("A push after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	bProvider, err := crypto.NewRootKeyProvider(bKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	aProvider, err := crypto.NewRootKeyProvider(aKeys[2], aKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	after := objectKeys(t, plane)
	newObjects, oldEnvelopes := 0, 0
	for key := range after {
		if strings.HasSuffix(key, keyring.ObjectName) {
			continue
		}
		switch {
		case before[key] && !strings.HasSuffix(key, "manifest.age"):
			oldEnvelopes++
			if !opensWith(t, plane, key, bProvider) {
				t.Fatalf("pre-revocation object %s no longer opens under generation 1", key)
			}
		case !before[key] || strings.HasSuffix(key, "manifest.age"):
			newObjects++
			if opensWith(t, plane, key, bProvider) {
				t.Fatalf("object %s written after the revocation opens with the revoked device's key", key)
			}
		}
		if !opensWith(t, plane, key, aProvider) {
			t.Fatalf("A cannot open %s", key)
		}
	}
	if newObjects == 0 || oldEnvelopes == 0 {
		t.Fatalf("expected objects on both sides of the revocation: new=%d old=%d keys=%v", newObjects, oldEnvelopes, after)
	}
	if out, errb, code := a.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("A pull after revocation: exit=%d out=%q err=%q", code, out, errb)
	}

	// Revoking again is harmless on both sides.
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK || !strings.Contains(out, "already revoked") {
		t.Fatalf("second revoke: exit=%d out=%q err=%q", code, out, errb)
	}
	if ring, _ := a.keyringState(t, plane); ring.CurrentGeneration != 2 || len(ring.Generations) != 2 {
		t.Fatalf("second revoke changed the keyring: gen=%d gens=%d", ring.CurrentGeneration, len(ring.Generations))
	}
	if len(plane.events) != 1 {
		t.Fatalf("second revoke recorded another event: %v", plane.events)
	}

	// The recovery code still works after the rollover: a new device
	// enrols from it into every generation and reads the whole locker,
	// including what was written before the revocation.
	d := newPairDevice(t, plane, "workstation")
	d.enrol(filepath.Join(userHome, "Projects", "workstation-target"), a.shownCode)
	out, errb, code = d.run("account", "status", "--json")
	if code != ExitOK {
		t.Fatalf("D status: %d %q", code, errb)
	}
	var status map[string]any
	_ = json.Unmarshal([]byte(out), &status)
	if status["key_generation"] != float64(2) || status["enrolled_devices"] != float64(2) || status["device_in_keyring"] != true {
		t.Fatalf("D status = %v", status)
	}
	if p := status["account_path"].(string); strings.Contains(p, `\`) {
		t.Fatalf("account_path must be slash-normalized on every host: %v", p)
	}
	if _, dKeys := d.keyringState(t, plane); len(dKeys) != 2 || !bytes.Equal(dKeys[1], aKeys[1]) || !bytes.Equal(dKeys[2], aKeys[2]) {
		t.Fatalf("D holds generations %v, want both", dKeys)
	}
	if out, errb, code := d.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("D pull: exit=%d out=%q err=%q", code, out, errb)
	}
	dRoot := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(filepath.Join(userHome, "Projects", "workstation-target")))
	for file, want := range map[string]string{"session-locker.jsonl": "synthetic locker journey", "session-after.jsonl": "written after the revocation"} {
		got, err := os.ReadFile(filepath.Join(dRoot, file))
		if err != nil || !bytes.Contains(got, []byte(want)) {
			t.Fatalf("D did not restore %s: %v %q", file, err, got)
		}
	}
	// B, revoked, enrols again by the recipe docs/hop.md documents, run
	// verbatim and with nothing removed by hand: sign in again, rein init
	// --hop --force, then rein account recover.
	if _, err := os.Stat(config.AccountPath(b.home)); err != nil {
		t.Fatal(err)
	}
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login again: %d %q", code, errb)
	}
	bReID := deviceID(t, b)
	if bReID == bID {
		t.Fatal("login again reused the revoked device id")
	}
	// The revoked machine still carries the record of the enrolment it
	// lost, and nothing but init --force removes it. Both enrolment
	// commands refuse while it is there, and both name the way out.
	for _, attempt := range []struct {
		name string
		run  func() (string, string, int)
	}{
		{"recover", func() (string, string, int) { return b.recover(a.shownCode) }},
		{"join", func() (string, string, int) { return b.run("account", "join") }},
	} {
		out, errb, code := attempt.run()
		if code != ExitSafety || !strings.Contains(errb, "already enrolled") || !strings.Contains(errb, "rein init --hop --force") {
			t.Fatalf("B %s before reinitializing: exit=%d out=%q err=%q", attempt.name, code, out, errb)
		}
	}
	if _, errb, code := b.run("init", "--hop", "--force", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target-2")); code != ExitOK {
		t.Fatalf("B init --hop again: %d %q", code, errb)
	}
	// init --force is what makes the documented recipe run: it copies the
	// stale enrolment record into a backup set and takes it off the home.
	if _, err := config.LoadAccount(b.home); !os.IsNotExist(err) {
		t.Fatalf("rein init --force left the stale enrolment record in place: %v", err)
	}
	backedUp, err := filepath.Glob(filepath.Join(b.home, "backups", "*", "account.json"))
	if err != nil || len(backedUp) == 0 {
		t.Fatalf("rein init --force removed the enrolment record without backing it up: %v %v", backedUp, err)
	}
	if out, errb, code := b.recover(a.shownCode); code != ExitOK || !strings.Contains(out, "1 earlier one") {
		t.Fatalf("B recover after revocation: exit=%d out=%q err=%q", code, out, errb)
	}
	// The other gate is still there for a home that carries no enrolment
	// record but names a device the account no longer signs in as.
	g := newPairDevice(t, plane, "unenrolled")
	if _, errb, code := g.run("login"); code != ExitOK {
		t.Fatalf("G login: %d %q", code, errb)
	}
	if _, errb, code := g.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "unenrolled-target")); code != ExitOK {
		t.Fatalf("G init --hop: %d %q", code, errb)
	}
	if _, errb, code := g.run("login"); code != ExitOK {
		t.Fatalf("G login again: %d %q", code, errb)
	}
	if out, errb, code := g.recover(a.shownCode); code != ExitConfig || !strings.Contains(errb, "is not the signed-in device") {
		t.Fatalf("G recover with a stale device_id: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, bAgain := b.keyringState(t, plane); len(bAgain) != 2 {
		t.Fatalf("re-enrolled B holds generations %v, want both", bAgain)
	}
	if out, errb, code := b.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("B pull after re-enrolment: exit=%d out=%q err=%q", code, out, errb)
	}
	if got, err := os.ReadFile(filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(filepath.Join(userHome, "Projects", "desktop-target-2")), "session-after.jsonl")); err != nil || !bytes.Contains(got, []byte("written after the revocation")) {
		t.Fatalf("re-enrolled B did not restore the post-revocation session: %v", err)
	}

	// A revocation racing an approval, revocation first: C joins, A's
	// prompt is open, D revokes B's new enrolment meanwhile. A's approval
	// lands on the rolled-over keyring and enrols C into generation 3 and
	// both earlier ones; C's join names generation 3 and succeeds.
	c := newPairDevice(t, plane, "laptop-2")
	if _, errb, code := c.run("login"); code != ExitOK {
		t.Fatalf("C login: %d %q", code, errb)
	}
	if _, errb, code := c.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "laptop-target")); code != ExitOK {
		t.Fatalf("C init --hop: %d %q", code, errb)
	}
	joinC := c.startJoin()
	out, errb, code = a.approveWhilePrompting(joinC.code, false, func() {
		if out, errb, code := d.revoke(bReID, a.shownCode); code != ExitOK || !strings.Contains(out, "key generation 3") {
			t.Errorf("D revokes B during A's prompt: exit=%d out=%q err=%q", code, out, errb)
		}
		// REINSTATE_HOME is process-wide; hand it back to A's command.
		t.Setenv("REINSTATE_HOME", a.home)
	})
	if code != ExitOK || !strings.Contains(out, "key generation 3") {
		t.Fatalf("A approves C across a rollover: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := joinC.finish(t); code != ExitOK || !strings.Contains(out, "key_generation=3") {
		t.Fatalf("C join across a rollover: exit=%d out=%q err=%q", code, out, errb)
	}
	ring, _ = a.keyringState(t, plane)
	cID := deviceID(t, c)
	if ring.CurrentGeneration != 3 || ring.DeviceCount() != 3 || !ring.HasDevice(cID) || ring.HasDevice(bReID) {
		t.Fatalf("keyring after approve-across-rollover: gen=%d devices=%d c=%v b=%v", ring.CurrentGeneration, ring.DeviceCount(), ring.HasDevice(cID), ring.HasDevice(bReID))
	}
	if _, cKeys := c.keyringState(t, plane); len(cKeys) != 3 {
		t.Fatalf("C holds generations %v, want all three", cKeys)
	}
	if out, errb, code := c.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("C pull: exit=%d out=%q err=%q", code, out, errb)
	}

	// Approval first, then a revocation before the joiner collects: the
	// rollover carries the just-approved wrap into generation 4, the
	// joiner refuses a payload naming generation 3 (fail closed, no
	// record), and its retry is approved at generation 4.
	e := newPairDevice(t, plane, "tablet")
	if _, errb, code := e.run("login"); code != ExitOK {
		t.Fatalf("E login: %d %q", code, errb)
	}
	if _, errb, code := e.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "tablet-target")); code != ExitOK {
		t.Fatalf("E init --hop: %d %q", code, errb)
	}
	joinE := e.startJoin()
	if out, errb, code := a.approve(joinE.code, false); code != ExitOK || !strings.Contains(out, "key generation 3") {
		t.Fatalf("A approves E: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := d.revoke(cID, a.shownCode); code != ExitOK || !strings.Contains(out, "key generation 4") {
		t.Fatalf("D revokes C after E's approval: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := joinE.finish(t); code != ExitSafety || !strings.Contains(errb, "generation 4 but the approval named 3") {
		t.Fatalf("E join after a rollover: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := config.LoadAccount(e.home); !os.IsNotExist(err) {
		t.Fatalf("E's refused join wrote an account record: %v", err)
	}
	eID := deviceID(t, e)
	ring, _ = a.keyringState(t, plane)
	if ring.CurrentGeneration != 4 || !ring.HasDevice(eID) || ring.HasDevice(cID) {
		t.Fatalf("keyring after revoke-after-approve: gen=%d e=%v c=%v", ring.CurrentGeneration, ring.HasDevice(eID), ring.HasDevice(cID))
	}
	retryE := e.startJoin()
	if out, errb, code := a.approve(retryE.code, false); code != ExitOK || !strings.Contains(out, "key generation 4") {
		t.Fatalf("A re-approves E: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := retryE.finish(t); code != ExitOK || !strings.Contains(out, "key_generation=4") {
		t.Fatalf("E join retry: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := e.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("E pull: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := c.run("pull", "--all", "--json"); code == ExitOK || !strings.Contains(errb, "token was rejected") {
		t.Fatalf("C pull after revocation: exit=%d out=%q err=%q", code, out, errb)
	}

	// Two devices revoke the same device at the same moment: both finish,
	// exactly one generation is started, and the control plane records
	// one event.
	f := newPairDevice(t, plane, "spare")
	f.enrol(filepath.Join(userHome, "Projects", "spare-target"), a.shownCode)
	fID := deviceID(t, f)
	ring, _ = a.keyringState(t, plane)
	generationsBefore := len(ring.Generations)
	eventsBefore := len(plane.events)
	var wg sync.WaitGroup
	results := make([]struct {
		out, errb string
		code      int
	}, 2)
	// REINSTATE_HOME is process-wide, so both revocations run from A's
	// home: two shells on the same machine racing each other, which is
	// the same compare-and-swap race on the keyring.
	t.Setenv("REINSTATE_HOME", a.home)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, errb := &syncBuffer{}, &syncBuffer{}
			code := a.execute(runOptions{stdout: out, stderr: errb, recovery: a.shownCode}, "devices", "revoke", fID)
			results[i].out, results[i].errb, results[i].code = out.String(), errb.String(), code
		}(i)
	}
	wg.Wait()
	revokedCount, alreadyCount := 0, 0
	for i, r := range results {
		if r.code != ExitOK {
			t.Fatalf("concurrent revoke %d: exit=%d out=%q err=%q", i, r.code, r.out, r.errb)
		}
		switch {
		case strings.Contains(r.out, "revoked device"):
			revokedCount++
		case strings.Contains(r.out, "already revoked"), strings.Contains(r.out, "had no wrap"):
			alreadyCount++
		}
	}
	if revokedCount != 1 || alreadyCount != 1 {
		t.Fatalf("concurrent revocations: %d revoked, %d already (%+v)", revokedCount, alreadyCount, results)
	}
	ring, _ = a.keyringState(t, plane)
	if len(ring.Generations) != generationsBefore+1 || ring.HasDevice(fID) || len(plane.events) != eventsBefore+1 {
		t.Fatalf("concurrent revocations: gens %d->%d, f=%v, events %d->%d", generationsBefore, len(ring.Generations), ring.HasDevice(fID), eventsBefore, len(plane.events))
	}
	for _, dev := range []*pairDevice{a, d, e} {
		if out, errb, code := dev.run("pull", "--all", "--json"); code != ExitOK {
			t.Fatalf("%s pull at the end: exit=%d out=%q err=%q", dev.name, code, out, errb)
		}
	}
	_ = ctx
}

func TestResolveDevice(t *testing.T) {
	devices := []hop.Device{
		{ID: "dev-1", Name: "macbook", Platform: "darwin-arm64", CreatedAt: "2026-08-01T00:00:00Z"},
		{ID: "dev-2", Name: "DESKTOP-7Q2", Platform: "windows-amd64", CreatedAt: "2026-08-02T00:00:00Z"},
		{ID: "dev-3", Name: "desktop-7q2", Platform: "windows-amd64", CreatedAt: "2026-08-03T00:00:00Z", RevokedAt: "2026-08-04T00:00:00Z"},
	}
	cases := map[string]struct {
		target  string
		wantID  string
		wantErr string
	}{
		"by id":                  {"dev-1", "dev-1", ""},
		"by name":                {"macbook", "dev-1", ""},
		"by name any case":       {"MacBook", "dev-1", ""},
		"padded":                 {"  dev-2 ", "dev-2", ""},
		"ambiguous windows name": {"desktop-7q2", "", "2 devices are named"},
		"unknown":                {"nobody", "", `no device "nobody"`},
		"empty":                  {"", "", "required"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := resolveDevice(devices, tc.target)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %v, want %q", err, tc.wantErr)
				}
				if name == "ambiguous windows name" && (!strings.Contains(err.Error(), "revoked 2026-08-04") || !strings.Contains(err.Error(), "enrolled 2026-08-02")) {
					t.Fatalf("ambiguity message lacks state: %v", err)
				}
				return
			}
			if err != nil || got.ID != tc.wantID {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
}

// keyringObject returns the stored keyring's key and bytes.
func keyringObject(t *testing.T, plane *fakeControlPlane) (string, []byte) {
	t.Helper()
	ctx := context.Background()
	objects, err := plane.s3.Store.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, meta := range objects {
		if strings.HasSuffix(meta.Key, keyring.ObjectName) {
			rc, _, err := plane.s3.Store.Get(ctx, meta.Key)
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			raw, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			return meta.Key, raw
		}
	}
	t.Fatal("no keyring in storage")
	return "", nil
}

// TestRevocationRefusesRolledBackKeyring is the rollback probe: a revoked
// device that still holds locker credentials puts its pre-rollover copy of
// the keyring back. Every remaining device has pinned the generation it
// saw, so push, pull, approve and revoke all fail closed instead of
// writing under the generation the revoked device holds.
func TestRevocationRefusesRolledBackKeyring(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000rollback")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	join := b.startJoin()
	if out, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("A approves B: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	_, bKeys := b.keyringState(t, plane)

	// B snapshots the keyring while still enrolled (it holds credentials
	// that last until expiry, so it can write the locker for a while).
	key, snapshot := keyringObject(t, plane)
	bID := deviceID(t, b)
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK {
		t.Fatalf("revoke: exit=%d out=%q err=%q", code, out, errb)
	}
	account, err := config.LoadAccount(a.home)
	if err != nil || account.KeyGeneration != 2 {
		t.Fatalf("A did not pin the generation it created: %+v %v", account, err)
	}
	before := objectKeys(t, plane)

	// B puts the generation-1 keyring back.
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(snapshot), int64(len(snapshot)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"written after the rollback"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-rollback.jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	want := "rolled back"
	if out, errb, code := a.run("push", "--all", "--json"); code != ExitSafety || !strings.Contains(errb, want) || !strings.Contains(errb, "current_generation 1 is below the 2") {
		t.Fatalf("A push on a rolled-back keyring: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("pull", "--all", "--json"); code != ExitSafety || !strings.Contains(errb, want) {
		t.Fatalf("A pull on a rolled-back keyring: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitSafety || !strings.Contains(errb, want) {
		t.Fatalf("A revoke on a rolled-back keyring: exit=%d out=%q err=%q", code, out, errb)
	}
	c := newPairDevice(t, plane, "laptop-2")
	if _, errb, code := c.run("login"); code != ExitOK {
		t.Fatalf("C login: %d %q", code, errb)
	}
	if _, errb, code := c.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "laptop-target")); code != ExitOK {
		t.Fatalf("C init --hop: %d %q", code, errb)
	}
	joinC := c.startJoin()
	if out, errb, code := a.approve(joinC.code, false); code != ExitSafety || !strings.Contains(errb, want) {
		t.Fatalf("A approve on a rolled-back keyring: exit=%d out=%q err=%q", code, out, errb)
	}
	// The request is left pending; expire it so C's join returns.
	plane.mu.Lock()
	for _, p := range plane.pairings {
		if p.status == "pending" {
			p.expired = true
		}
	}
	plane.mu.Unlock()
	if out, errb, code := joinC.finish(t); code == ExitOK {
		t.Fatalf("C join succeeded without an approval: out=%q err=%q", out, errb)
	}

	// Nothing new was written, the generation-1 key opens nothing it did
	// not already open, and the keyring on disk is the rolled-back copy.
	after := objectKeys(t, plane)
	bProvider, err := crypto.NewRootKeyProvider(bKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	for k := range after {
		if strings.HasSuffix(k, keyring.ObjectName) {
			continue
		}
		if !before[k] {
			t.Fatalf("object %s was written on a rolled-back keyring", k)
		}
		if !strings.HasSuffix(k, "manifest.age") && !opensWith(t, plane, k, bProvider) {
			t.Fatalf("pre-revocation object %s no longer opens under generation 1", k)
		}
	}
	if _, raw := keyringObject(t, plane); !bytes.Equal(raw, snapshot) {
		t.Fatal("a refused command rewrote the keyring")
	}
	if account, err := config.LoadAccount(a.home); err != nil || account.KeyGeneration != 2 {
		t.Fatalf("A's pinned generation moved: %+v %v", account, err)
	}
	if len(plane.events) != 1 {
		t.Fatalf("control plane events %v", plane.events)
	}
}

func backendPutOptions() backend.PutOptions { return backend.PutOptions{} }

// TestExpiredApprovalRollsBackEveryGeneration: on a multi-generation
// keyring, an approval enrols the joining device into every generation the
// approver can read (EnrolAll). When the relay then refuses — the request
// expired while the approver's prompt was open — the rollback must sweep
// every generation: removing only the current generation's wrap would
// leave the refused device able to unwrap pre-revocation history.
func TestExpiredApprovalRollsBackEveryGeneration(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000rollback")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")

	// A: first device. B: joins by approval. A revokes B, which starts
	// key generation 2 — the keyring now has history worth protecting.
	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	joinB := b.startJoin()
	if out, errb, code := a.approve(joinB.code, false); code != ExitOK {
		t.Fatalf("A approves B: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := joinB.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.revoke("desktop", a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}

	// C asks to join; the request expires while A's prompt is open. The
	// approval had already written C's wrap into both generations.
	c := newPairDevice(t, plane, "laptop-2")
	if _, errb, code := c.run("login"); code != ExitOK {
		t.Fatalf("C login: %d %q", code, errb)
	}
	if _, errb, code := c.run("init", "--hop"); code != ExitOK {
		t.Fatalf("C init --hop: %d %q", code, errb)
	}
	cID := deviceID(t, c)
	joinC := c.startJoin()
	out, errb, code := a.approveWhilePrompting(joinC.code, false, func() {
		plane.mu.Lock()
		defer plane.mu.Unlock()
		for _, p := range plane.pairings {
			if p.status == "pending" {
				p.expired = true
			}
		}
	})
	if code != ExitAuthStorage || !strings.Contains(errb, "expired") || !strings.Contains(errb, "was removed again") {
		t.Fatalf("approve expiring at the prompt: exit=%d out=%q err=%q", code, out, errb)
	}

	// NO generation lists the refused device: the current one and every
	// earlier one, or C could still unwrap pre-revocation history.
	ring, aKeys := a.keyringState(t, plane)
	keyring.ZeroGenerations(aKeys)
	if ring.CurrentGeneration != 2 || len(ring.Generations) != 2 {
		t.Fatalf("keyring shape changed: gen=%d gens=%d", ring.CurrentGeneration, len(ring.Generations))
	}
	for _, g := range ring.Generations {
		for _, d := range g.Devices {
			if d.DeviceID == cID {
				t.Fatalf("generation %d still lists refused device %s", g.Number, cID)
			}
		}
	}
	if out, errb, code := joinC.finish(t); code != ExitAuthStorage || !strings.Contains(errb, "expired") {
		t.Fatalf("C join expired at the prompt: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := config.LoadAccount(c.home); !os.IsNotExist(err) {
		t.Fatalf("expired join wrote an account record: %v", err)
	}
}

// TestRevokeHelpNamesTheCredentialWindow: the command's own help text and
// docs/hop.md must agree. A revoked device is refused new credentials
// instantly, but one it already minted keeps working against the bucket
// until it expires, and storage.Provider has no way to withdraw it. Help
// that says only "cannot push" tells the operator the window does not exist.
func TestRevokeHelpNamesTheCredentialWindow(t *testing.T) {
	long := strings.Join(strings.Fields(newDevicesRevokeCmd().Long), " ")
	for _, want := range []string{"cannot mint new locker credentials", "until it expires", "up to an hour"} {
		if !strings.Contains(long, want) {
			t.Fatalf("rein devices revoke --help does not say %q:\n%s", want, long)
		}
	}
	if strings.Contains(long, "and it cannot push") {
		t.Fatalf("rein devices revoke --help still claims the revoked device cannot push:\n%s", long)
	}
}

// mutateGeneration rewrites one generation inside a marshalled keyring, by
// index, leaving every other byte of the object alone. It is how the
// forgeries here are built: a party with bucket write access edits the
// published object, it does not rebuild it.
func mutateGeneration(t *testing.T, raw []byte, index int, f func(map[string]any)) []byte {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	f(obj["generations"].([]any)[index].(map[string]any))
	out, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// forgeKeyring builds the object a party with write access to the locker,
// and no root key at all, can put in the keyring's place: a whole keyring
// for the same profile, wrapping a root key of its own to the public keys
// the genuine keyring published, signed under an account key it derived
// from a recovery code it drew itself, and rolled forward to generation so
// it clears the floor every remaining device has pinned.
//
// This is the residual attack signature verification alone cannot stop — the
// object is internally perfect — and the one the locally pinned account key
// exists to catch.
func forgeKeyring(t *testing.T, genuine *keyring.Keyring, generation int) []byte {
	t.Helper()
	attackerKey, err := crypto.NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	attackerDevice, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	attackerCode, err := keyring.GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	forged, err := keyring.New(genuine.ProfileID, attackerKey, attackerCode, "attacker-device", attackerDevice, now)
	if err != nil {
		t.Fatal(err)
	}
	// Every device the genuine keyring lists gets a wrap of the attacker's
	// key, sealed to the public key the genuine keyring published.
	for _, id := range currentDeviceIDs(genuine) {
		recipient, err := age.ParseX25519Recipient(genuine.DevicePublicKey(id))
		if err != nil {
			t.Fatal(err)
		}
		if err := forged.Enrol(attackerKey, id, recipient, now); err != nil {
			t.Fatal(err)
		}
	}
	current := attackerKey
	for forged.CurrentGeneration < generation {
		spare, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatal(err)
		}
		id := fmt.Sprintf("spare-%d", forged.CurrentGeneration)
		if err := forged.Enrol(current, id, spare.Recipient(), now); err != nil {
			t.Fatal(err)
		}
		next, err := forged.Rollover(current, attackerCode, []string{id}, "attacker-device", now)
		if err != nil {
			t.Fatal(err)
		}
		current = next
	}
	raw, err := forged.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// currentDeviceIDs lists the devices the current generation wraps for.
func currentDeviceIDs(k *keyring.Keyring) []string {
	var ids []string
	for _, g := range k.Generations {
		if g.Number != k.CurrentGeneration {
			continue
		}
		for _, d := range g.Devices {
			ids = append(ids, d.DeviceID)
		}
	}
	return ids
}

// TestKeyringForgeryIsRefusedOnEveryReadPath is the blocker probe from the
// 2026-08-27 verification round, driven through the real CLI.
//
// A revoked device keeps working locker credentials for the rest of their
// TTL, so it can write the keyring object. Two forgeries follow from that,
// and every command that acts on the keyring must refuse both: writing a
// generation nobody but a holder of the recovery code could sign, and
// replacing the whole object with a self-consistent keyring signed under an
// account key of its own. Refusing on push alone would not be a fix — pull,
// approve, revoke and recover all load the keyring too.
func TestKeyringForgeryIsRefusedOnEveryReadPath(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000000forge")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	join := b.startJoin()
	if out, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("A approves B: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	bID := deviceID(t, b)
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}
	key, genuine := keyringObject(t, plane)
	ring, err := keyring.Parse(genuine)
	if err != nil {
		t.Fatal(err)
	}
	if ring.CurrentGeneration != 2 {
		t.Fatalf("expected generation 2 after the revocation, got %d", ring.CurrentGeneration)
	}
	account, err := config.LoadAccount(a.home)
	if err != nil || account.KeyGeneration != 2 || account.KeyRecipient != ring.CurrentRecipient() {
		t.Fatalf("A did not pin the generation it created: %+v %v", account, err)
	}
	before := objectKeys(t, plane)

	// A refusal must happen at the public keyring gate, before pull fetches
	// ciphertext to decrypt or push writes any locker object. The fake's S3
	// request log makes that ordering observable without reaching into the
	// command implementation.
	assertNoLockerDataAccess := func(t *testing.T, requests []string) {
		t.Helper()
		for _, request := range requests {
			if strings.HasPrefix(request, "PUT ") || strings.HasPrefix(request, "DELETE ") {
				t.Fatalf("a refused command changed the locker: %s", request)
			}
			for object := range before {
				if object != key && strings.Contains(request, " "+object+" as ") {
					t.Fatalf("a refused command fetched locker data before authenticating the keyring: %s", request)
				}
			}
		}
	}

	// Something for a push to have to write.
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"written after the forgery"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-forged.jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	put := func(raw []byte) {
		t.Helper()
		if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(raw), int64(len(raw)), backendPutOptions()); err != nil {
			t.Fatal(err)
		}
	}
	// refuseEverywhere drives every command that loads the keyring and
	// requires each one to fail closed for the same stated reason.
	refuseEverywhere := func(t *testing.T, label, want string) {
		t.Helper()
		for _, args := range [][]string{{"push", "--all", "--json"}, {"pull", "--all", "--json"}} {
			logStart := len(plane.s3.RequestLog())
			out, errb, code := a.run(args...)
			if code != ExitSafety || !strings.Contains(errb, want) {
				t.Fatalf("A %v on a forged keyring: exit=%d out=%q err=%q", args, code, out, errb)
			}
			assertNoLockerDataAccess(t, plane.s3.RequestLog()[logStart:])
		}
		if out, errb, code := a.revoke(bID, a.shownCode); code != ExitSafety || !strings.Contains(errb, want) {
			t.Fatalf("A revoke on a forged keyring: exit=%d out=%q err=%q", code, out, errb)
		}
		c := newPairDevice(t, plane, "laptop-"+label)
		if _, errb, code := c.run("login"); code != ExitOK {
			t.Fatalf("C login: %d %q", code, errb)
		}
		if _, errb, code := c.run("init", "--hop"); code != ExitOK {
			t.Fatalf("C init --hop: %d %q", code, errb)
		}
		// A joining device refuses the object on its first load, before it
		// publishes anything; only when it gets past that does the
		// approving device's refusal come into it. Both are failures
		// closed, and both must name the same reason.
		joinC, out, errb, code := c.tryStartJoin()
		switch {
		case joinC == nil:
			if code != ExitSafety || !strings.Contains(errb, want) {
				t.Fatalf("C join on a forged keyring: exit=%d out=%q err=%q", code, out, errb)
			}
		default:
			out, errb, code := a.approve(joinC.code, false)
			if code != ExitSafety || !strings.Contains(errb, want) {
				t.Fatalf("A approve on a forged keyring: exit=%d out=%q err=%q", code, out, errb)
			}
			plane.mu.Lock()
			for _, p := range plane.pairings {
				if p.status == "pending" {
					p.expired = true
				}
			}
			plane.mu.Unlock()
			if out, errb, code := joinC.finish(t); code == ExitOK {
				t.Fatalf("C join succeeded against a forged keyring: out=%q err=%q", out, errb)
			}
		}
		for k := range objectKeys(t, plane) {
			if !before[k] && !strings.HasSuffix(k, keyring.ObjectName) {
				t.Fatalf("object %s was written on a forged keyring", k)
			}
		}
		// The two diagnostics hold no keys, exit 0, and must report the
		// keyring as refused rather than as the account's key-model truth.
		// They reach the refusal by different routes — an object that does
		// not verify fails to load at all, while a replacement loads and
		// is caught by the anchor — so both forgeries are driven through
		// them, not only the one that parses.
		for _, args := range [][]string{{"account", "status"}, {"devices"}} {
			out, errb, code := a.run(args...)
			if code != ExitOK || !strings.Contains(out, "this device refuses it") {
				t.Fatalf("A %v on a forged keyring: exit=%d out=%q err=%q", args, code, out, errb)
			}
		}
	}

	// Forgery 1: a generation whose signature does not verify. A party with
	// bucket write access has everything else — the profile id, the
	// generation numbers, the account key, every device's public key — and
	// cannot produce the one value only the recovery code derives.
	// Corrupting the signature the genuine rollover produced is what any
	// forged value looks like to a reader.
	t.Run("a generation nothing signed", func(t *testing.T) {
		put(mutateGeneration(t, genuine, 1, func(g map[string]any) {
			forged := make([]byte, ed25519.SignatureSize)
			forged[0] = 1
			g["signature"] = base64.StdEncoding.EncodeToString(forged)
		}))
		refuseEverywhere(t, "signature", "is not signed by this account's key")
	})

	// Forgery 1b: the same, one generation back. A reader that accepted a
	// keyring because its *current* generation verified would sail past
	// this one, which is how the version 3 chain failed open.
	t.Run("an earlier generation nothing signed", func(t *testing.T) {
		put(mutateGeneration(t, genuine, 0, func(g map[string]any) {
			g["signature"] = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		}))
		refuseEverywhere(t, "earlier", "is not signed by this account's key")
	})

	// Forgery 2: the whole object replaced, signed end to end under an
	// account key the attacker derived from a recovery code of its own. It
	// verifies against itself, so only the account key this device pinned
	// at enrolment tells it from the account's own keyring.
	t.Run("the whole keyring replaced", func(t *testing.T) {
		forgedRaw := forgeKeyring(t, ring, ring.CurrentGeneration)
		forged, err := keyring.Parse(forgedRaw)
		if err != nil {
			t.Fatal(err)
		}
		if forged.CurrentGeneration != ring.CurrentGeneration || forged.CurrentRecipient() == account.KeyRecipient {
			t.Fatalf("the replacement is not the current-generation recipient forgery this probe requires: genuine=%s forged=%s generation=%d", account.KeyRecipient, forged.CurrentRecipient(), forged.CurrentGeneration)
		}
		put(forgedRaw)
		refuseEverywhere(t, "rewrite", "signed by a different account key")
	})

	// A forgery must not brick anything either: with the genuine object
	// back, every command works, the pinned generation never moved, and
	// the recovery code still enrols a device that reads everything.
	put(genuine)
	if account, err := config.LoadAccount(a.home); err != nil || account.KeyGeneration != 2 || account.KeyRecipient != ring.CurrentRecipient() {
		t.Fatalf("A's anchor moved while the forgeries were in place: %+v %v", account, err)
	}
	for _, args := range [][]string{{"push", "--all", "--json"}, {"pull", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v after the genuine keyring came back: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	d := newPairDevice(t, plane, "workstation")
	d.enrol(filepath.Join(userHome, "Projects", "workstation-target"), a.shownCode)
	if _, dKeys := d.keyringState(t, plane); len(dKeys) != 2 {
		t.Fatalf("the recovered device holds generations %v, want both", dKeys)
	}
	if out, errb, code := d.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("D pull: exit=%d out=%q err=%q", code, out, errb)
	}
	if got, err := os.ReadFile(filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(filepath.Join(userHome, "Projects", "workstation-target")), "session-forged.jsonl")); err != nil || !bytes.Contains(got, []byte("written after the forgery")) {
		t.Fatalf("D did not restore the session pushed after the forgeries: %v", err)
	}
}
