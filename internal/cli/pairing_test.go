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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
)

var pairingCodePattern = regexp.MustCompile(`\b(?:[0-9A-Z]{4}-){3}[0-9A-Z]{4}\b`)

// syncBuffer is a bytes.Buffer safe to read while another goroutine's
// command is still writing to it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// pairDevice is one machine in a pairing journey: its own Reinstate home,
// device-token store, and OS-keyring stand-in, sharing the fake control
// plane and the fake locker with the other machines.
type pairDevice struct {
	t       *testing.T
	plane   *fakeControlPlane
	name    string
	home    string
	tokens  *credentials.MemoryDeviceTokenStore
	secrets credentials.SecretStore
	// shownCode captures the recovery code at account init.
	shownCode string
}

type runOptions struct {
	sleep         func(context.Context, time.Duration) error
	pairingPrompt func(string) ([]byte, error)
	stdout        *syncBuffer
	stderr        *syncBuffer
}

func newPairDevice(t *testing.T, plane *fakeControlPlane, name string) *pairDevice {
	return &pairDevice{t: t, plane: plane, name: name, home: t.TempDir(), tokens: &credentials.MemoryDeviceTokenStore{}, secrets: credentials.NewMemorySecrets()}
}

// run executes one command for this device from the test goroutine.
func (d *pairDevice) run(args ...string) (string, string, int) {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	out, errb := &syncBuffer{}, &syncBuffer{}
	code := d.execute(runOptions{stdout: out, stderr: errb}, args...)
	return out.String(), errb.String(), code
}

// execute runs the CLI with this device's seams; REINSTATE_HOME must
// already point at d.home (it is read once, when the command starts).
func (d *pairDevice) execute(ro runOptions, args ...string) int {
	sleep := ro.sleep
	if sleep == nil {
		sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	}
	return Execute(Options{
		Name: "rein", Stdout: ro.stdout, Stderr: ro.stderr, Args: args,
		AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		DeviceTokenStore:    d.tokens,
		DeviceSecrets:       d.secrets,
		OpenBrowser: func(u string) error {
			resp, err := http.Get(u)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		},
		LoginPollSleep: sleep,
		DeviceName:     d.name,
		RecoveryCodePrompt: func(prompt string) ([]byte, error) {
			if !strings.Contains(prompt, "Re-enter") {
				return nil, errors.New("unexpected prompt " + prompt)
			}
			d.shownCode = recoveryCodePattern.FindString(ro.stderr.String())
			if d.shownCode == "" {
				return nil, errors.New("recovery code was not shown before the confirmation prompt")
			}
			return []byte(d.shownCode), nil
		},
		PairingCodePrompt: ro.pairingPrompt,
	})
}

// joinInProgress is a `rein account join` blocked in its poll loop: the
// code it shows is readable, and release lets it poll again.
type joinInProgress struct {
	code    string
	stdout  *syncBuffer
	stderr  *syncBuffer
	release chan struct{}
	done    chan int
}

// startJoin runs `rein account join` on d in the background and returns
// once the device has published its pairing request and shown the code.
// The join polls once, then waits for release before polling again, so the
// test decides exactly when the approval becomes visible to it.
func (d *pairDevice) startJoin() *joinInProgress {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	j := &joinInProgress{stdout: &syncBuffer{}, stderr: &syncBuffer{}, release: make(chan struct{}), done: make(chan int, 1)}
	requested := make(chan struct{})
	var once sync.Once
	sleep := func(ctx context.Context, _ time.Duration) error {
		once.Do(func() { close(requested) })
		select {
		case <-j.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	go func() {
		j.done <- d.execute(runOptions{sleep: sleep, stdout: j.stdout, stderr: j.stderr}, "account", "join")
	}()
	select {
	case <-requested:
	case code := <-j.done:
		d.t.Fatalf("join exited before publishing a request: exit=%d out=%q err=%q", code, j.stdout.String(), j.stderr.String())
	case <-time.After(30 * time.Second):
		d.t.Fatal("join never published a pairing request")
	}
	j.code = pairingCodePattern.FindString(j.stderr.String())
	if j.code == "" {
		d.t.Fatalf("join showed no pairing code: %q", j.stderr.String())
	}
	return j
}

// finish lets the join poll again and returns its result.
func (j *joinInProgress) finish(t *testing.T) (string, string, int) {
	t.Helper()
	close(j.release)
	select {
	case code := <-j.done:
		return j.stdout.String(), j.stderr.String(), code
	case <-time.After(30 * time.Second):
		t.Fatal("join did not finish")
		return "", "", -1
	}
}

// approve runs `rein devices approve` on d with a code answered at the
// hidden prompt (or delivered through REINSTATE_PAIRING_CODE_FD when fd is
// true, the automation path), optionally for one request id.
func (d *pairDevice) approve(code string, fd bool, extra ...string) (string, string, int) {
	d.t.Helper()
	return d.approveWhilePrompting(code, fd, nil, extra...)
}

// approveWhilePrompting is approve with a hook that runs while the hidden
// prompt is open (before the code is answered), standing in for whatever
// happens on the other machine while the user walks over to read the code.
func (d *pairDevice) approveWhilePrompting(code string, fd bool, atPrompt func(), extra ...string) (string, string, int) {
	d.t.Helper()
	d.t.Setenv("REINSTATE_HOME", d.home)
	out, errb := &syncBuffer{}, &syncBuffer{}
	ro := runOptions{stdout: out, stderr: errb}
	if fd {
		withSecretFD(d.t, "REINSTATE_PAIRING_CODE_FD", code)
	} else {
		ro.pairingPrompt = func(prompt string) ([]byte, error) {
			if !strings.Contains(prompt, "Pairing code") {
				return nil, errors.New("unexpected prompt " + prompt)
			}
			if atPrompt != nil {
				atPrompt()
			}
			return []byte(code), nil
		}
	}
	args := append([]string{"devices", "approve"}, extra...)
	exit := d.execute(ro, args...)
	if fd {
		d.t.Setenv("REINSTATE_PAIRING_CODE_FD", "")
	}
	return out.String(), errb.String(), exit
}

// withSecretFD hands secret to the CLI through a descriptor, as automation
// would, and names it in env.
func withSecretFD(t *testing.T, env, secret string) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "secret-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.WriteString(secret + "\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Setenv(env, strconv.FormatUint(uint64(file.Fd()), 10))
}

func (d *pairDevice) keyringDevices() int {
	d.t.Helper()
	out, errb, code := d.run("account", "status", "--json")
	if code != ExitOK {
		d.t.Fatalf("account status: exit=%d out=%q err=%q", code, out, errb)
	}
	var status struct {
		EnrolledDevices int `json:"enrolled_devices"`
	}
	_ = json.Unmarshal([]byte(out), &status)
	return status.EnrolledDevices
}

// TestPairingJourneyJoinApprovePull is the primary-seam journey for device
// approval: device A initialises the account and pushes; device B signs in
// and joins, showing a code; A approves by entering that code; B pulls and
// reads what A wrote. Along the way: a wrong code approves nothing and
// leaves no wrap behind, an expired request fails closed, and two pending
// requests approved back to back both end up in one converged keyring.
func TestPairingJourneyJoinApprovePull(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000000pair")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	t.Setenv("TZ", "Asia/Kolkata")
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")

	// Device A: sign in, init for Hop, generate the root key, push.
	a := newPairDevice(t, plane, "macbook")
	if out, errb, code := a.run("login"); code != ExitOK {
		t.Fatalf("A login: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("init", "--hop", "--project", "local/locker="+project); code != ExitOK {
		t.Fatalf("A init --hop: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("account", "init"); code != ExitOK {
		t.Fatalf("A account init: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.run("push", "--all", "--json"); code != ExitOK {
		t.Fatalf("A push: exit=%d out=%q err=%q", code, out, errb)
	}
	// Nothing to approve yet.
	if out, errb, code := a.approve("0000-0000-0000-0000", false); code != ExitUsage || !strings.Contains(errb, "no pending pairing requests") {
		t.Fatalf("approve with nothing pending: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, _, code := a.run("devices"); code != ExitOK || !strings.Contains(out, "macbook") || !strings.Contains(out, "holds a root-key wrap") || !strings.Contains(out, "no pending pairing requests") {
		t.Fatalf("A devices: exit=%d out=%q", code, out)
	}

	// Device B: a different machine (a Windows desktop) with a different
	// HOME, signs in to the same account and asks to join.
	b := newPairDevice(t, plane, "desktop")
	if out, errb, code := b.run("account", "join"); code != ExitAuthStorage || !strings.Contains(errb, "not signed in") {
		t.Fatalf("B join before login: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := b.run("pull", "--all", "--json"); code == ExitOK {
		t.Fatalf("B pull before join succeeded: out=%q err=%q", out, errb)
	}
	join := b.startJoin()
	if strings.Contains(join.stdout.String(), join.code) {
		t.Fatalf("pairing code leaked to stdout: %q", join.stdout.String())
	}
	if strings.Count(join.stderr.String(), join.code) != 1 {
		t.Fatalf("pairing code must be shown exactly once: %q", join.stderr.String())
	}
	assertPairingCodeNeverSent(t, plane, join.code)

	// A sees the request.
	out, _, code := a.run("devices")
	if code != ExitOK || !strings.Contains(out, "desktop") || !strings.Contains(out, "pending approval") || !strings.Contains(out, "no root-key wrap yet") {
		t.Fatalf("A devices with pending request: exit=%d out=%q", code, out)
	}

	// A wrong code (valid checksum) approves nothing and appends no wrap.
	wrong, err := keyring.NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	out, errb, code := a.approve(wrong.Code, false)
	if code != ExitSafety || !strings.Contains(errb, "does not match this pairing request") || !strings.Contains(errb, "nothing was approved") {
		t.Fatalf("wrong code: exit=%d out=%q err=%q", code, out, errb)
	}
	if n := a.keyringDevices(); n != 1 {
		t.Fatalf("wrong code left %d device wraps, want 1", n)
	}
	plane.mu.Lock()
	if p := plane.pairings["pair-1"]; p.status != "pending" || p.payload != "" {
		t.Fatalf("wrong code changed the request: %+v", p)
	}
	plane.mu.Unlock()
	// A typo is caught by the checksum before any key derivation.
	typo := join.code[:len(join.code)-1] + map[bool]string{true: "2", false: "3"}[strings.HasSuffix(join.code, "3")]
	if out, errb, code := a.approve(typo, false); code != ExitUsage || !strings.Contains(errb, "checksum") {
		t.Fatalf("typo: exit=%d out=%q err=%q", code, out, errb)
	}

	// The right code, delivered through the automation descriptor, typed
	// casually (lower case, spaces).
	out, errb, code = a.approve(strings.ToLower(strings.ReplaceAll(join.code, "-", " ")), true)
	if code != ExitOK || !strings.Contains(out, `approved device "desktop"`) || !strings.Contains(out, "key generation 1") {
		t.Fatalf("approve: exit=%d out=%q err=%q", code, out, errb)
	}
	if n := a.keyringDevices(); n != 2 {
		t.Fatalf("keyring lists %d devices after approval, want 2", n)
	}
	assertPairingCodeNeverSent(t, plane, join.code)

	// B collects the root key and finishes.
	out, errb, code = join.finish(t)
	if code != ExitOK || !strings.Contains(out, `approved by "macbook"`) || !strings.Contains(out, "this device can now read the locker") {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	accountB, err := config.LoadAccount(b.home)
	if err != nil || accountB.EnrolledVia != "join" || accountB.RecoveryCodeConfirmed || accountB.KeyGeneration != 1 {
		t.Fatalf("B account record: %+v %v", accountB, err)
	}
	out, errb, code = b.run("account", "status", "--json")
	if code != ExitOK {
		t.Fatalf("B status: exit=%d out=%q err=%q", code, out, errb)
	}
	var status map[string]any
	_ = json.Unmarshal([]byte(out), &status)
	if status["device_in_keyring"] != true || status["enrolled_devices"] != float64(2) || status["recovery_code_confirmed"] != false || status["enrolled_via"] != "join" {
		t.Fatalf("B status = %v", status)
	}
	if p := status["account_path"].(string); strings.Contains(p, `\`) {
		t.Fatalf("account_path must be slash-normalized on every host: %v", p)
	}
	// The request is single use.
	if out, errb, code := a.approve(join.code, false); code != ExitUsage || !strings.Contains(errb, "no pending pairing requests") {
		t.Fatalf("approve after completion: exit=%d out=%q err=%q", code, out, errb)
	}

	// B pulls and reads what A wrote.
	out, errb, code = b.run("pull", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("B pull: exit=%d out=%q err=%q", code, out, errb)
	}
	restored := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(filepath.Join(userHome, "Projects", "desktop-target")), "session-locker.jsonl")
	content, err := os.ReadFile(restored)
	if err != nil || !bytes.Contains(content, []byte("synthetic locker journey")) {
		t.Fatalf("B did not restore A's session: %v %q", err, content)
	}
	// Neither the code nor the root key reached the locker or the relay.
	objects, err := plane.s3.Store.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, meta := range objects {
		rc, _, err := plane.s3.Store.Get(context.Background(), meta.Key)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(rc)
		_ = rc.Close()
		if bytes.Contains(raw, []byte(join.code)) || bytes.Contains(raw, []byte(strings.ReplaceAll(join.code, "-", ""))) {
			t.Fatalf("locker object %s holds the pairing code", meta.Key)
		}
	}
	if out, _, code := b.run("devices", "--json"); code != ExitOK || !strings.Contains(out, `"in_keyring": true`) || strings.Contains(out, `"in_keyring": false`) {
		t.Fatalf("B devices --json: exit=%d out=%q", code, out)
	}
	if out, errb, code := b.run("account", "join"); code != ExitSafety || !strings.Contains(errb, "already enrolled") {
		t.Fatalf("B second join: exit=%d out=%q err=%q", code, out, errb)
	}

	// An expired request fails closed on both sides.
	c := newPairDevice(t, plane, "laptop-2")
	if _, errb, code := c.run("login"); code != ExitOK {
		t.Fatalf("C login: %d %q", code, errb)
	}
	if _, errb, code := c.run("init", "--hop"); code != ExitOK {
		t.Fatalf("C init --hop: %d %q", code, errb)
	}
	expiredJoin := c.startJoin()
	plane.mu.Lock()
	for _, p := range plane.pairings {
		if p.status == "pending" {
			p.expired = true
		}
	}
	plane.mu.Unlock()
	if out, errb, code := a.approve(expiredJoin.code, false); code != ExitUsage || !strings.Contains(errb, "no pending pairing requests") {
		t.Fatalf("approve expired: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := expiredJoin.finish(t); code != ExitAuthStorage || !strings.Contains(errb, "expired") {
		t.Fatalf("C expired join: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := config.LoadAccount(c.home); !os.IsNotExist(err) {
		t.Fatalf("expired join wrote an account record: %v", err)
	}
	if n := a.keyringDevices(); n != 2 {
		t.Fatalf("expired join changed the keyring: %d devices", n)
	}

	// The request expires while A's prompt is open (the user walked over
	// to read the code): the relay refuses, and the wrap A had already
	// appended is removed again so C's next join is a fresh request, not
	// a silent enrolment without an approval behind it.
	promptJoin := c.startJoin()
	out, errb, code = a.approveWhilePrompting(promptJoin.code, false, func() {
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
	if n := a.keyringDevices(); n != 2 {
		t.Fatalf("expiry at the prompt left a wrap behind: %d devices", n)
	}
	if out, errb, code := promptJoin.finish(t); code != ExitAuthStorage || !strings.Contains(errb, "expired") {
		t.Fatalf("C join expired at the prompt: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := config.LoadAccount(c.home); !os.IsNotExist(err) {
		t.Fatalf("join expired at the prompt wrote an account record: %v", err)
	}

	// A request listed as pending whose expiry has nonetheless passed on
	// A's clock (skew, or a listing that sat in a pager) is refused before
	// anything is written, and the request is left untouched.
	staleJoin := c.startJoin()
	plane.mu.Lock()
	var staleID string
	for _, p := range plane.pairings {
		if p.status == "pending" && !p.expired {
			staleID = p.id
			p.expiresAt = time.Now().UTC().Add(-time.Minute)
		}
	}
	plane.mu.Unlock()
	if out, errb, code := a.approve(staleJoin.code, false); code != ExitUsage || !strings.Contains(errb, "expired at") {
		t.Fatalf("approve past expiry: exit=%d out=%q err=%q", code, out, errb)
	}
	if n := a.keyringDevices(); n != 2 {
		t.Fatalf("refused approval changed the keyring: %d devices", n)
	}
	plane.mu.Lock()
	if p := plane.pairings[staleID]; p.status != "pending" || p.payload != "" {
		t.Fatalf("refused approval touched the request: status=%s payload=%q", p.status, p.payload)
	}
	plane.pairings[staleID].expired = true
	plane.mu.Unlock()
	if out, errb, code := staleJoin.finish(t); code != ExitAuthStorage || !strings.Contains(errb, "expired") {
		t.Fatalf("C stale join: exit=%d out=%q err=%q", code, out, errb)
	}

	// C retries: a fresh request with a fresh code, never "already
	// enrolled" from a wrap left behind by a failed approval.
	joinC := c.startJoin()
	if strings.Contains(joinC.stdout.String(), "already enrolled") || joinC.code == promptJoin.code {
		t.Fatalf("C retry after rollback: out=%q err=%q", joinC.stdout.String(), joinC.stderr.String())
	}

	// Two requests pending at once (C retries, D is new): approving both
	// back to back converges on one keyring holding every wrap, and each
	// joiner verifies its own wrap before trusting anything.
	d := newPairDevice(t, plane, "workstation")
	if _, errb, code := d.run("login"); code != ExitOK {
		t.Fatalf("D login: %d %q", code, errb)
	}
	if _, errb, code := d.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "workstation-target")); code != ExitOK {
		t.Fatalf("D init --hop: %d %q", code, errb)
	}
	joinD := d.startJoin()
	if joinC.code == joinD.code {
		t.Fatal("two requests drew the same code")
	}
	if out, errb, code := a.approve(joinC.code, false); code != ExitUsage || !strings.Contains(errb, "--request") || !strings.Contains(errb, "laptop-2") || !strings.Contains(errb, "workstation") {
		t.Fatalf("approve with two pending and no --request: exit=%d out=%q err=%q", code, out, errb)
	}
	var idC, idD string
	plane.mu.Lock()
	for _, p := range plane.pairings {
		if p.status != "pending" || p.expired {
			continue
		}
		switch plane.deviceByID(p.deviceID).Name {
		case "laptop-2":
			idC = p.id
		case "workstation":
			idD = p.id
		}
	}
	plane.mu.Unlock()
	// The code for one request does not approve the other.
	if out, errb, code := a.approve(joinD.code, false, "--request", idC); code != ExitSafety || !strings.Contains(errb, "does not match") {
		t.Fatalf("D's code on C's request: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := a.approve(joinD.code, false, "--request", idD); code != ExitOK {
		t.Fatalf("approve D: exit=%d out=%q err=%q", code, out, errb)
	}
	// B, not A, approves C: any enrolled device can.
	if out, errb, code := b.approve(joinC.code, false, "--request", idC); code != ExitOK || !strings.Contains(out, `approved device "laptop-2"`) {
		t.Fatalf("B approves C: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := joinC.finish(t); code != ExitOK || !strings.Contains(out, `approved by "desktop"`) {
		t.Fatalf("C join: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := joinD.finish(t); code != ExitOK || !strings.Contains(out, `approved by "macbook"`) {
		t.Fatalf("D join: exit=%d out=%q err=%q", code, out, errb)
	}
	for _, dev := range []*pairDevice{a, b, c, d} {
		if n := dev.keyringDevices(); n != 4 {
			t.Fatalf("%s sees %d enrolled devices, want 4", dev.name, n)
		}
	}
	if out, errb, code := d.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("D pull: exit=%d out=%q err=%q", code, out, errb)
	}
}

// assertPairingCodeNeverSent checks every pairing request the fake control
// plane holds for the code: the relay must only ever see key material the
// code cannot be recovered from.
func assertPairingCodeNeverSent(t *testing.T, plane *fakeControlPlane, code string) {
	t.Helper()
	plane.mu.Lock()
	defer plane.mu.Unlock()
	compact := strings.ReplaceAll(code, "-", "")
	for _, p := range plane.pairings {
		for _, field := range []string{p.publicKey, p.salt, p.binding, p.payload} {
			if strings.Contains(field, code) || strings.Contains(field, compact) {
				t.Fatalf("pairing code reached the control plane in %+v", p)
			}
		}
	}
}
