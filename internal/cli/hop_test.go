package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// fakeControlPlane speaks the public Hop sign-in protocol in-process. It
// approves a session when the "browser" visits the verification URL or,
// for email, when the test approves the emailed link.
type fakeControlPlane struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	sessions map[string]*fakeSession
	tokens   map[string]hop.Identity // device token -> identity
	emails   []string                // addresses a link was "sent" to
	expireAt map[string]bool         // session ids reported as expired
	seq      int

	// Locker state (see hop_locker_test.go for the journeys).
	s3 *s3test.Fake // nil until a test attaches one
	// locker is the single account's bucket; provisioned on first POST.
	locker *fakeLocker
	// lockerPrefix is the key prefix the locker record advertises. Hop
	// provisions lockers without one, but the record carries the field and
	// every client path honours it, so a journey can set one and see what
	// a prefixed locker actually does.
	lockerPrefix string
	provisions   int      // POST /v1/locker calls; only the first should ever happen
	hints        []string // location hints received at sign-in
	mints        []string // access key ids minted, in order
	credTTL      time.Duration
	refuse       string // error code every mint answers with, when set
	usageBytes   int64
	firstPushes  int

	// Pairing relays (see pairing_fake_test.go).
	pairings   map[string]*fakePairing
	pairingSeq int

	// Verification (see verify_fake_test.go): the reference locker the
	// plane advertises (nil = none), an HTTP status the reference lookup
	// answers with instead (0 = answer normally), and every report posted,
	// in order.
	reference       *fakeReference
	referenceStatus int
	reports         []fakeReport
}

type fakeLocker struct {
	bucket      string
	firstPushAt string
}

type fakeSession struct {
	id, secret, method, email, link string
	device                          hop.DeviceInfo
	status                          string
	token                           string
	identity                        hop.Identity
}

func newFakeControlPlane(t *testing.T) *fakeControlPlane {
	t.Helper()
	f := &fakeControlPlane{t: t, sessions: map[string]*fakeSession{}, tokens: map[string]hop.Identity{}, expireAt: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/login/sessions", f.create)
	mux.HandleFunc("POST /v1/login/sessions/{id}/poll", f.poll)
	mux.HandleFunc("GET /v1/whoami", f.whoami)
	mux.HandleFunc("POST /v1/locker", f.provisionLocker)
	mux.HandleFunc("GET /v1/locker", f.lockerStatus)
	mux.HandleFunc("POST /v1/locker/credentials", f.mintCredentials)
	mux.HandleFunc("POST /v1/locker/first-push", f.firstPush)
	f.registerPairing(mux)
	f.registerVerify(mux)
	mux.HandleFunc("GET /login/github/{link}", func(w http.ResponseWriter, r *http.Request) {
		f.approveLink(w, r.PathValue("link"), "github")
	})
	// Like hopd, the emailed link renders a confirm form on GET and enrols
	// only on POST, so a mail scanner's prefetch cannot sign anyone in.
	mux.HandleFunc("GET /login/email/{link}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<h1>Approve this device?</h1><form method="post"><button>Approve device</button></form>`))
	})
	mux.HandleFunc("POST /login/email/{link}", func(w http.ResponseWriter, r *http.Request) {
		f.approveLink(w, r.PathValue("link"), "email")
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeControlPlane) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string         `json:"method"`
		Email  string         `json:"email"`
		Device hop.DeviceInfo `json:"device"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.Device.Name == "" || req.Device.Platform == "" {
		writeFakeError(w, 400, "device.name and device.platform are required")
		return
	}
	f.hints = append(f.hints, req.Device.LocationHint)
	f.seq++
	s := &fakeSession{id: "sess-" + strconv.Itoa(f.seq), secret: "secret-" + strconv.Itoa(f.seq), link: "link-" + strconv.Itoa(f.seq), method: req.Method, email: req.Email, device: req.Device, status: hop.StatusPending}
	resp := map[string]any{"session_id": s.id, "poll_secret": s.secret, "method": s.method, "expires_at": "2026-08-23T12:10:00Z", "interval_seconds": 0}
	switch req.Method {
	case "github":
		resp["verification_url"] = f.srv.URL + "/login/github/" + s.link
	case "email":
		if !strings.Contains(req.Email, "@") {
			writeFakeError(w, 400, "email must be a plain address such as you@example.com")
			return
		}
		f.emails = append(f.emails, req.Email)
	default:
		writeFakeError(w, 400, `method must be "github" or "email"`)
		return
	}
	f.sessions[s.id] = s
	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *fakeControlPlane) approveLink(w http.ResponseWriter, link, method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if s.link != link || s.method != method {
			continue
		}
		if s.status != hop.StatusPending {
			http.Error(w, "Link already used", 410)
			return
		}
		s.status = hop.StatusApproved
		s.token = "hop_" + s.id
		acct := hop.Account{ID: "acct-1", Plan: "hop", LocationHint: "apac", CreatedAt: "2026-08-23T12:00:00Z"}
		if method == "github" {
			acct.GitHubLogin, acct.Email = "octocat", "octo@example.com"
		} else {
			acct.Email = s.email
		}
		s.identity = hop.Identity{Account: acct, Device: hop.Device{ID: "dev-" + s.id, Name: s.device.Name, Platform: s.device.Platform, CreatedAt: "2026-08-23T12:01:00Z", LastSeenAt: "2026-08-23T12:01:00Z"}}
		f.tokens[s.token] = s.identity
		_, _ = w.Write([]byte("<h1>Signed in</h1>"))
		return
	}
	http.Error(w, "Unknown sign-in link", 404)
}

func (f *fakeControlPlane) poll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PollSecret string `json:"poll_secret"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[r.PathValue("id")]
	if !ok || s.secret != req.PollSecret {
		writeFakeError(w, 404, "unknown login session")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case s.status == hop.StatusConsumed:
		w.WriteHeader(410)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "consumed"})
	case s.status == hop.StatusPending && f.expireAt[s.id]:
		w.WriteHeader(410)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "expired"})
	case s.status == hop.StatusPending:
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
	default:
		s.status = hop.StatusConsumed
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "approved", "device_token": s.token, "account": s.identity.Account, "device": s.identity.Device})
	}
}

func (f *fakeControlPlane) whoami(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if !ok {
		writeFakeError(w, 401, "unknown or revoked device token")
		return
	}
	_ = json.NewEncoder(w).Encode(id)
}

// approveLatestEmail acts as the person clicking the emailed link.
func (f *fakeControlPlane) approveLatestEmail() {
	f.mu.Lock()
	var latest *fakeSession
	for _, s := range f.sessions {
		if s.method == "email" && (latest == nil || s.id > latest.id) {
			latest = s
		}
	}
	f.mu.Unlock()
	if latest == nil {
		f.t.Fatal("no email session to approve")
	}
	link := f.srv.URL + "/login/email/" + latest.link
	// A prefetch of the link must leave the session pending.
	get, err := http.Get(link)
	if err != nil {
		f.t.Fatal(err)
	}
	get.Body.Close()
	f.mu.Lock()
	if latest.status != hop.StatusPending {
		f.mu.Unlock()
		f.t.Fatal("GET on the emailed link approved the session")
	}
	f.mu.Unlock()
	resp, err := http.PostForm(link, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	resp.Body.Close()
}

func (f *fakeControlPlane) revoke(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tokens, token)
}

func writeFakeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// hopHarness runs the CLI against the fake control plane with an in-memory
// device-token store and a browser that approves the GitHub link.
type hopHarness struct {
	t       *testing.T
	plane   *fakeControlPlane
	tokens  *credentials.MemoryDeviceTokenStore
	browsed []string
	browser func(string) error
}

func newHopHarness(t *testing.T) *hopHarness {
	t.Helper()
	h := &hopHarness{t: t, plane: newFakeControlPlane(t), tokens: &credentials.MemoryDeviceTokenStore{}}
	h.browser = func(url string) error {
		h.browsed = append(h.browsed, url)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	t.Setenv("REINSTATE_HOME", t.TempDir())
	t.Setenv(hop.URLEnv, h.plane.srv.URL)
	return h
}

func (h *hopHarness) run(args ...string) (stdout, stderr string, code int) {
	h.t.Helper()
	var out, errb bytes.Buffer
	code = Execute(Options{
		Name:             "rein",
		Stdout:           &out,
		Stderr:           &errb,
		Args:             args,
		DeviceTokenStore: h.tokens,
		OpenBrowser:      func(u string) error { return h.browser(u) },
		LoginPollSleep:   func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
		DeviceName:       "laptop",
	})
	return out.String(), errb.String(), code
}

func TestLoginWithGitHubThenWhoami(t *testing.T) {
	h := newHopHarness(t)

	out, errb, code := h.run("login")
	if code != ExitOK {
		t.Fatalf("login exit=%d stdout=%q stderr=%q", code, out, errb)
	}
	if len(h.browsed) != 1 || !strings.Contains(h.browsed[0], "/login/github/") {
		t.Fatalf("browser opened %v", h.browsed)
	}
	if !strings.Contains(errb, "Sign in with GitHub at:") || !strings.Contains(errb, h.browsed[0]) {
		t.Fatalf("stderr %q", errb)
	}
	if !strings.Contains(out, "Signed in to Reinstate Hop as octo@example.com (GitHub @octocat)") || !strings.Contains(out, `enrolled as "laptop"`) {
		t.Fatalf("stdout %q", out)
	}
	tok, err := h.tokens.GetDeviceToken()
	if err != nil || !strings.HasPrefix(tok.Token, "hop_") || tok.ControlPlaneURL != h.plane.srv.URL || tok.DeviceID != "dev-sess-1" {
		t.Fatalf("stored token %+v err=%v", tok, err)
	}

	out, errb, code = h.run("whoami")
	if code != ExitOK {
		t.Fatalf("whoami exit=%d stderr=%q", code, errb)
	}
	for _, want := range []string{"Account: octo@example.com (GitHub @octocat)", "Device:  laptop (", "Hop:     " + h.plane.srv.URL} {
		if !strings.Contains(out, want) {
			t.Fatalf("whoami output %q missing %q", out, want)
		}
	}

	// A second login on a signed-in machine says so before enrolling again.
	if _, errb, code := h.run("login"); code != ExitOK || !strings.Contains(errb, "already signed in (device dev-sess-1") {
		t.Fatalf("re-login exit=%d stderr=%q", code, errb)
	}
	if tok, _ := h.tokens.GetDeviceToken(); tok.DeviceID != "dev-sess-2" {
		t.Fatalf("re-login did not replace the token: %+v", tok)
	}

	out, _, code = h.run("whoami", "--json")
	if code != ExitOK {
		t.Fatalf("whoami --json exit=%d", code)
	}
	var got struct {
		ControlPlane string      `json:"control_plane"`
		Account      hop.Account `json:"account"`
		Device       hop.Device  `json:"device"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v %q", err, out)
	}
	if got.ControlPlane != h.plane.srv.URL || got.Account.GitHubLogin != "octocat" || got.Device.Name != "laptop" || got.Device.Platform == "" {
		t.Fatalf("%+v", got)
	}
	if strings.Contains(out, "hop_") {
		t.Fatalf("whoami --json leaked the device token: %q", out)
	}
}

func TestLoginWithEmailLink(t *testing.T) {
	h := newHopHarness(t)
	// No browser is involved in email sign-in; the person clicks the link on
	// any device. Approve it after the CLI has started polling.
	h.browser = func(string) error { t.Fatal("email login must not open a browser"); return nil }
	approved := false
	var out, errb bytes.Buffer
	code := Execute(Options{
		Name: "rein", Stdout: &out, Stderr: &errb,
		Args:             []string{"login", "--email", "You@Example.com", "--json"},
		DeviceTokenStore: h.tokens,
		OpenBrowser:      h.browser,
		DeviceName:       "desktop",
		LoginPollSleep: func(context.Context, time.Duration) error {
			if !approved {
				approved = true
				h.plane.approveLatestEmail()
			}
			return nil
		},
	})
	if code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), errb.String())
	}
	if len(h.plane.emails) != 1 || h.plane.emails[0] != "You@Example.com" {
		t.Fatalf("emails sent %v", h.plane.emails)
	}
	if errb.Len() != 0 {
		t.Fatalf("--json must keep stderr quiet: %q", errb.String())
	}
	var got struct {
		Account hop.Account `json:"account"`
		Device  hop.Device  `json:"device"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Account.Email != "You@Example.com" || got.Device.Name != "desktop" {
		t.Fatalf("%+v err=%v out=%q", got, err, out.String())
	}
	if _, err := h.tokens.GetDeviceToken(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginFailures(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		prepare  func(h *hopHarness)
		wantCode int
		wantErr  string
	}{
		{
			name:     "expired session",
			args:     []string{"login"},
			prepare:  func(h *hopHarness) { h.browser = func(string) error { h.plane.expireAt["sess-1"] = true; return nil } },
			wantCode: ExitRuntime,
			wantErr:  "expired",
		},
		{
			name:     "bad email address",
			args:     []string{"login", "--email", "nope"},
			wantCode: ExitUsage,
			wantErr:  "plain address",
		},
		{
			name:     "unreachable control plane",
			args:     []string{"login"},
			prepare:  func(h *hopHarness) { h.t.Setenv(hop.URLEnv, "http://127.0.0.1:1") },
			wantCode: ExitRuntime,
			wantErr:  "reach control plane",
		},
		{
			name: "no-browser prints the url only",
			args: []string{"login", "--no-browser"},
			prepare: func(h *hopHarness) {
				h.browser = func(string) error { h.t.Fatal("browser opened"); return nil }
				h.plane.expireAt["sess-1"] = true
			},
			wantCode: ExitRuntime,
			wantErr:  "expired",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHopHarness(t)
			if tc.prepare != nil {
				tc.prepare(h)
			}
			_, errb, code := h.run(tc.args...)
			if code != tc.wantCode || !strings.Contains(errb, tc.wantErr) {
				t.Fatalf("exit=%d want %d stderr=%q", code, tc.wantCode, errb)
			}
			if _, err := h.tokens.GetDeviceToken(); err == nil {
				t.Fatal("a failed login must not store a token")
			}
		})
	}
}

func TestWhoamiWithoutOrWithRevokedToken(t *testing.T) {
	h := newHopHarness(t)
	_, errb, code := h.run("whoami")
	if code != ExitAuthStorage || !strings.Contains(errb, "not signed in") {
		t.Fatalf("exit=%d stderr=%q", code, errb)
	}
	if _, _, code := h.run("login"); code != ExitOK {
		t.Fatal("login failed")
	}
	tok, _ := h.tokens.GetDeviceToken()
	h.plane.revoke(tok.Token)
	_, errb, code = h.run("whoami", "--json")
	if code != ExitAuthStorage || !strings.Contains(errb, "rejected") {
		t.Fatalf("exit=%d stderr=%q", code, errb)
	}
	var e ErrorJSON
	if err := json.Unmarshal([]byte(errb), &e); err != nil || e.Code != "auth_storage" {
		t.Fatalf("json error %+v err=%v", e, err)
	}
}

func TestControlPlaneURLResolution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv(hop.URLEnv, "")
	_ = os.Unsetenv(hop.URLEnv)
	if got := controlPlaneURL(); got != hop.DefaultURL {
		t.Fatalf("default %q", got)
	}
	cfg := "schema_version = 1\nprofile_id = \"p\"\ndevice_id = \"d\"\n[storage]\ntype = \"s3\"\n[hop]\nurl = \"https://staging.example/\"\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := controlPlaneURL(); got != "https://staging.example" {
		t.Fatalf("config %q", got)
	}
	t.Setenv(hop.URLEnv, "http://127.0.0.1:9999/")
	if got := controlPlaneURL(); got != "http://127.0.0.1:9999" {
		t.Fatalf("env %q", got)
	}
}

func TestPlaintextRemoteWarning(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://hop.reinstate.dev", false},
		{"http://127.0.0.1:8080", false},
		{"http://localhost:8080", false},
		{"http://[::1]:8080", false},
		{"http://staging.example", true},
		{"http://10.0.0.5:8080", true},
	}
	for _, tc := range tests {
		if got := plaintextRemote(tc.url); got != tc.want {
			t.Errorf("plaintextRemote(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}
