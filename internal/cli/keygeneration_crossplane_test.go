//go:build hopacceptance

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// The account key generation floor is the one thing in tickets #11 and #12
// that no single repository can exercise. The client half lives here and
// the server half lives in the private control plane, and every test on
// either side talks to a fake of the other: the CLI journeys drive a fake
// control plane written in this package, and hopd's own tests drive its
// store directly. A wire-shape mistake between the two -- a renamed field,
// a status code read the other way round -- passes both suites and fails on
// the first real device.
//
// This is the journey against the real thing: the real rein code in
// process, against the real hopd binary over HTTP, with a shared
// disk-backed locker standing in for the bucket. It is built only with
// -tags hopacceptance and skips unless REINSTATE_HOPD_BIN names a hopd
// built from the control plane repository, because that repository is
// private and a public checkout has no way to produce one.
//
// What the shared locker directory stands in for is stated plainly: it is
// the bucket, and writing to it directly is what a revoked device does
// inside the hour its already-minted credential keeps working. Nothing here
// stands in for R2 itself; see the bench record for what that leaves.
const hopdBinEnv = "REINSTATE_HOPD_BIN"

// hopdLab is a running hopd: its base URL, its log output (where the log
// email sender writes magic links), and the knobs a test needs.
type hopdLab struct {
	t       *testing.T
	base    string
	logs    *syncBuffer
	client  *http.Client
	linksAt int
}

var magicLink = regexp.MustCompile(`https?://[^\s]+/login/email/[A-Za-z0-9_\-]+`)

// startHopd runs the real control plane on a free loopback port with a
// throwaway database, the in-memory locker provider, and the log email
// sender, and waits for /healthz.
func startHopd(t *testing.T) *hopdLab {
	t.Helper()
	bin := strings.TrimSpace(os.Getenv(hopdBinEnv))
	if bin == "" {
		t.Skipf("%s is not set; the cross-plane key-generation journey needs a hopd built from the control plane repository", hopdBinEnv)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("%s=%s: %v", hopdBinEnv, bin, err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	base := "http://" + addr
	logs := &syncBuffer{}
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"HOPD_ADDR="+addr,
		"HOPD_BASE_URL="+base,
		"HOPD_DB_PATH="+filepath.Join(t.TempDir(), "hopd.db"),
		"HOPD_EMAIL_SENDER=log",
		"HOPD_STORAGE=fake",
		// The locker record has to name an endpoint or the client refuses
		// it as incomplete. Nothing in this journey talks to it: the bucket
		// is stood in for by REINSTATE_BACKEND=memory, which is also what
		// leaves the control-plane floor lookup as the only live call the
		// keyring path makes.
		"HOPD_S3_ENDPOINT=https://locker.invalid",
	)
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", bin, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		if t.Failed() {
			t.Logf("hopd log:\n%s", logs.String())
		}
	})
	lab := &hopdLab{t: t, base: base, logs: logs, client: &http.Client{Timeout: 10 * time.Second}}
	deadline := time.Now().Add(20 * time.Second)
	for {
		resp, err := lab.client.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return lab
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("hopd did not answer /healthz within 20s; log:\n%s", logs.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// approveNextLink finds the magic link hopd has mailed since the last call
// and posts it, which is what a person clicking "Approve this device?"
// does. It reports whether a new link was there to approve.
func (l *hopdLab) approveNextLink() bool {
	l.t.Helper()
	links := magicLink.FindAllString(l.logs.String(), -1)
	if len(links) <= l.linksAt {
		return false
	}
	link := links[l.linksAt]
	l.linksAt++
	resp, err := l.client.PostForm(link, url.Values{})
	if err != nil {
		l.t.Fatalf("approve %s: %v", link, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		l.t.Fatalf("approve %s: %d %s", link, resp.StatusCode, body)
	}
	return true
}

// keyGenerationFloor reads the account's floor the way a device does.
func (l *hopdLab) keyGenerationFloor(token string) int {
	l.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, l.base+"/v1/account/key-generation", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := l.client.Do(req)
	if err != nil {
		l.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		l.t.Fatalf("GET /v1/account/key-generation: %d %s", resp.StatusCode, body)
	}
	var out struct {
		Generation int `json:"generation"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		l.t.Fatalf("decode %s: %v", body, err)
	}
	return out.Generation
}

// signIn runs the real sign-in against hopd, approving the mailed link from
// inside the poll wait so the journey needs no second goroutine.
func (l *hopdLab) signIn(d *hopDevice, address string) {
	l.t.Helper()
	approved := false
	d.loginSleep = func(ctx context.Context, _ time.Duration) error {
		if !approved {
			approved = l.approveNextLink()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	d.ctx = ctx
	d.mustRun("login "+d.name, "login", "--email", address, "--no-browser")
	d.ctx = nil
	d.loginSleep = nil
	if !approved {
		l.t.Fatalf("%s signed in without a link being approved", d.name)
	}
}

// deviceIDNamed returns the id of the device hopd knows by name.
func deviceIDNamed(t *testing.T, d *hopDevice, name string) string {
	t.Helper()
	out := d.mustRun("devices --json", "devices", "--json")
	var report struct {
		Devices []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"devices"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("devices --json %q: %v", out, err)
	}
	for _, row := range report.Devices {
		if row.Name == name {
			return row.ID
		}
	}
	t.Fatalf("no device named %q in %q", name, out)
	return ""
}

// copyTree copies src to dst, which is how the journey keeps a genuine
// pre-revocation copy of the locker and puts one object back afterwards.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.RemoveAll(dst); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// lockerKeyringObject locates the keyring inside a disk-backed locker. The store
// is keyed by the sha256 of the object key and indexed by a file under
// keys/, so the index is what says which object is the keyring and the
// content lives beside it under objects/.
func lockerKeyringObject(t *testing.T, root string) (body, meta string) {
	t.Helper()
	var found []string
	err := filepath.WalkDir(filepath.Join(root, "keys"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".key") {
			return nil
		}
		name, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.HasSuffix(string(name), "keyring.v1.json") {
			found = append(found, string(name))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("keyring keys under %s: %v; want exactly one", root, found)
	}
	sum := sha256.Sum256([]byte(found[0]))
	body = filepath.Join(root, "objects", hex.EncodeToString(sum[:]))
	return body, body + ".meta.json"
}

// restoreKeyring writes the keyring from one locker directory over another,
// which is what a revoked device does to the bucket with the credential it
// was already given.
func restoreKeyring(t *testing.T, from, to string) {
	t.Helper()
	srcBody, srcMeta := lockerKeyringObject(t, from)
	dstBody, dstMeta := lockerKeyringObject(t, to)
	for _, pair := range [][2]string{{srcBody, dstBody}, {srcMeta, dstMeta}} {
		raw, err := os.ReadFile(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pair[1], raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// mustToken reads a device's Hop token out of its own token store.
func mustToken(t *testing.T, d *hopDevice) string {
	t.Helper()
	tok, err := d.tokens.GetDeviceToken()
	if err != nil {
		t.Fatal(err)
	}
	return tok.Token
}

// TestKeyGenerationFloorAgainstRealHopd is the lagging-device attack, driven
// against the real control plane rather than a fake of it.
//
//	A enrols, writes the keyring, pushes.
//	B enrols from the recovery code.
//	C enrols from the recovery code and pulls once, so it records key
//	  generation 1 and whatever floor hopd served then, which is 0.
//	The locker is copied: that copy is the genuine generation-1 keyring,
//	  correctly signed, which is what makes this attack different from
//	  planting a forgery.
//	A revokes B. The keyring rolls to generation 2 and rein devices revoke
//	  reports the rollover, so the account's floor at hopd becomes 2.
//	The genuine generation-1 keyring is written back over the current one,
//	  which is what a revoked device can still do for up to an hour with
//	  the locker credential it was already given.
//	C, which has run nothing since generation 1, is asked to work.
//
// Without the floor, C accepts the restored keyring -- its own anchor says
// generation 1 and the object is genuine -- and keeps sealing to the root
// key the revoked device holds. With it, C asks hopd first, is told 2, and
// refuses.
func TestKeyGenerationFloorAgainstRealHopd(t *testing.T) {
	lab := startHopd(t)
	locker := t.TempDir()
	t.Setenv(hopURLEnv, lab.base)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	for _, env := range []string{"REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(env, "")
	}
	home := plantJourneyHome(t, t.TempDir())
	const address = "floor@example.test"

	a := newHopDevice(t, nil, "device-a")
	lab.signIn(a, address)
	a.mustRun("init --hop", "init", "--hop", "--project", "local/floor="+home.project)
	a.mustRun("account init", "account", "init")
	code := a.shownCode
	if code == "" {
		t.Fatal("account init showed no recovery code")
	}
	a.mustRun("push --all", "push", "--all")

	b := newHopDevice(t, nil, "device-b")
	lab.signIn(b, address)
	b.mustRun("init --hop", "init", "--hop")
	b.typedCode = code
	b.mustRun("account recover", "account", "recover")

	c := newHopDevice(t, nil, "device-c")
	lab.signIn(c, address)
	c.mustRun("init --hop", "init", "--hop")
	c.typedCode = code
	c.mustRun("account recover", "account", "recover")
	// One ordinary command, which is all it takes to record what this
	// device saw: key generation 1 in its own anchor, and whatever floor
	// the control plane served then. After this C goes quiet.
	c.mustRun("account status", "account", "status")

	if floor := lab.keyGenerationFloor(mustToken(t, a)); floor != 0 {
		t.Fatalf("the account's floor is %d before any revocation; want 0, which every keyring satisfies", floor)
	}

	// The genuine pre-revocation locker, kept aside.
	before := filepath.Join(t.TempDir(), "generation-1")
	copyTree(t, locker, before)
	genuineBody, _ := lockerKeyringObject(t, before)
	genuine, err := os.ReadFile(genuineBody)
	if err != nil {
		t.Fatal(err)
	}

	bID := deviceIDNamed(t, a, "device-b")
	a.typedCode = code
	a.mustRun("devices revoke", "devices", "revoke", bID)

	if floor := lab.keyGenerationFloor(mustToken(t, a)); floor != 2 {
		t.Fatalf("the account's floor is %d after a revocation to key generation 2; want 2", floor)
	}

	// The revoked device puts the genuine earlier keyring back.
	currentBody, _ := lockerKeyringObject(t, locker)
	rolled, err := os.ReadFile(currentBody)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(rolled, genuine) {
		t.Fatal("the revocation did not change the keyring object; the attack would prove nothing")
	}
	restoreKeyring(t, before, locker)

	// Every route C has that reads the keyring must refuse, and say why.
	for _, tc := range []struct {
		step string
		args []string
		// diagnostics report the refusal and still exit 0.
		diagnostic bool
	}{
		{step: "push", args: []string{"push", "--all"}},
		{step: "pull", args: []string{"pull", "--all"}},
		{step: "sync verify", args: []string{"sync", "verify"}},
		// devices approve is in the same class and is driven by
		// TestControlPlaneFloorReachesADeviceThatLagsBehind against the
		// in-package fake; it is left out here because it answers "no
		// pending pairing requests" before it reads a keyring, and opening
		// one needs a fourth device sitting in an interactive poll.
		{step: "account status", args: []string{"account", "status"}, diagnostic: true},
		{step: "devices", args: []string{"devices"}, diagnostic: true},
	} {
		out, errb, exit := c.run(tc.args...)
		said := out + errb
		if !strings.Contains(said, "control plane") {
			t.Errorf("%s on the lagging device said %q; the refusal has to name the control plane it came from", tc.step, said)
		}
		if tc.diagnostic {
			if exit != ExitOK {
				t.Errorf("%s exited %d; a diagnostic reports the refusal rather than becoming one", tc.step, exit)
			}
			continue
		}
		if exit != ExitSafety {
			t.Errorf("%s on the lagging device exited %d, want %d (ExitSafety): out=%q err=%q", tc.step, exit, ExitSafety, out, errb)
		}
	}

	// The device that did the revoking refuses the same object for a
	// different reason: it holds generation 2 in its own anchor. Naming
	// both matters, because it is the floor that reaches C and the local
	// anchor that reaches A.
	if _, errb, exit := a.run("push", "--all"); exit != ExitSafety {
		t.Errorf("the revoking device accepted the rolled-back keyring: exit=%d err=%q", exit, errb)
	}
}

// TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd is the other half of
// what the documentation claims, driven the same way: a control plane that
// does not carry the floor at all.
//
// docs/hop.md says two things about that case and they are different. A
// device that has already confirmed a floor keeps it, because the floor a
// command uses is the higher of the live answer and its own record. A
// device that has never confirmed one has nothing to be higher than, and
// accepts the restored keyring. Both are exercised here, because the second
// is a residual and a residual that is not pinned quietly becomes a claim.
func TestKeyGenerationFloorWithoutTheRouteAgainstRealHopd(t *testing.T) {
	lab := startHopd(t)
	// A proxy in front of the real hopd that answers the two floor routes
	// 404, exactly as a deployment older than them does.
	upstream, err := url.Parse(lab.base)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	floorReads := 0
	older := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/account/key-generation" {
			mu.Lock()
			floorReads++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"not found"}`)
			return
		}
		proxied := *r.URL
		proxied.Scheme, proxied.Host = upstream.Scheme, upstream.Host
		req, err := http.NewRequest(r.Method, proxied.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(older.Close)

	locker := t.TempDir()
	t.Setenv(hopURLEnv, older.URL)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	for _, env := range []string{"REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "CLAUDE_CONFIG_DIR"} {
		t.Setenv(env, "")
	}
	home := plantJourneyHome(t, t.TempDir())
	const address = "nofloor@example.test"

	a := newHopDevice(t, nil, "older-a")
	lab.signIn(a, address)
	a.mustRun("init --hop", "init", "--hop", "--project", "local/floor="+home.project)
	a.mustRun("account init", "account", "init")
	code := a.shownCode
	a.mustRun("push --all", "push", "--all")

	c := newHopDevice(t, nil, "older-c")
	lab.signIn(c, address)
	c.mustRun("init --hop", "init", "--hop")
	c.typedCode = code
	c.mustRun("account recover", "account", "recover")
	// One ordinary command, which is all it takes to record what this
	// device saw: key generation 1 in its own anchor, and whatever floor
	// the control plane served then. After this C goes quiet.
	c.mustRun("account status", "account", "status")

	b := newHopDevice(t, nil, "older-b")
	lab.signIn(b, address)
	b.mustRun("init --hop", "init", "--hop")
	b.typedCode = code
	b.mustRun("account recover", "account", "recover")

	mu.Lock()
	reads := floorReads
	mu.Unlock()
	if reads == 0 {
		t.Fatal("no command asked for the floor; the 404 path was never taken and this test proves nothing")
	}
	t.Logf("the floor route was asked for %d times and answered 404 every time", reads)

	before := filepath.Join(t.TempDir(), "generation-1")
	copyTree(t, locker, before)

	bID := deviceIDNamed(t, a, "older-b")
	a.typedCode = code
	out, errb, exit := a.run("devices", "revoke", bID)
	if exit != ExitOK {
		t.Fatalf("devices revoke against a control plane with no floor route: exit=%d out=%q err=%q", exit, out, errb)
	}
	// The revocation succeeds and says what it could not get, rather than
	// reporting a protection it did not obtain.
	if !strings.Contains(out+errb, "key generation") {
		t.Errorf("devices revoke said %q; on a control plane that carries no floor it has to say so", out+errb)
	}

	restoreKeyring(t, before, locker)

	// C has never confirmed a floor: nothing on this control plane and
	// nothing in its own record contradicts the restored generation-1
	// keyring, so it opens it and reports it as current. This is the
	// documented residual, pinned here so it cannot quietly become a
	// protection nobody re-tested. account status is the probe rather than
	// a sync command, because a sync also has sessions to disagree about
	// and its exit code would then be about them.
	cOut, cErr, cExit := c.run("account", "status")
	said := cOut + cErr
	switch {
	case cExit != ExitOK || !strings.Contains(said, "key generation 1"):
		t.Logf("the lagging device did not accept the restored keyring: exit=%d said=%q", cExit, said)
		t.Log("docs/hop.md's residual about a control plane that carries no floor is then weaker than the code, and should be revisited")
	default:
		t.Log("documented residual reproduced: on a control plane that carries no floor, a device that has confirmed none reads the restored generation-1 keyring as current")
	}

	// A holds generation 2 in its own anchor, so the local half still bites
	// even with no floor anywhere. This is what the residual leaves in place.
	if _, errb, exit := a.run("push", "--all"); exit != ExitSafety {
		t.Errorf("with no control-plane floor the local anchor still has to refuse a rollback on the device that saw it: exit=%d err=%q", exit, errb)
	}
}
