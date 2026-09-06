package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// The login and whoami command tests themselves live in login_test.go and
// whoami_test.go (see file-ownership.md's internal/cli grant); this file
// stays behind because fakeControlPlane and hopHarness below are shared
// fixtures used well beyond those two commands — by, among others,
// hop_locker_test.go, hop_refusal_test.go, pairing_test.go,
// revocation_test.go, keyring_*_test.go, and daemon_test.go.

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
	polls    int // POST .../poll calls, so a test can see a loop stop

	// refuseSignIn, when set, is how the browser approval ends instead of
	// enrolling: the page renders the sentence and the refusal is recorded
	// on the session, so the poll answers `refused` (see hop_refusal_test.go).
	refuseSignIn *fakeSignInRefusal

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
	// Revoked devices keep their record (with revoked_at) but no token.
	revoked            map[string]hop.Device
	revocationRequests map[string]*fakeRevocationRequest
	revocationSeq      int
	events             []string // "<type>:<device id>" in order

	// The account key-generation floor (see pairing_fake_test.go).
	keyGeneration      int
	keyGenerationAt    string
	keyGenerationBy    string
	keyGenerationReads int
	// noKeyGenerationFloor makes both floor routes answer 404, as a
	// control plane that predates them does.
	noKeyGenerationFloor bool

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
	refusal                         *fakeSignInRefusal
}

// fakeSignInRefusal is one way the browser half of a sign-in can end
// without enrolling a device. pageStatus is the browser page's own status,
// which differs per refusal; the poll always answers 403, so a client that
// switched on the poll's status would learn nothing from it.
type fakeSignInRefusal struct {
	code       string
	reason     string
	pageStatus int
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
			http.Error(w, "Link already used", http.StatusGone)
			return
		}
		if r := f.refuseSignIn; r != nil {
			// Recorded before the page is rendered, which is the whole
			// point: the person reads the sentence in the browser and the
			// CLI polling this session reads the same one.
			s.status = hop.StatusRefused
			s.refusal = r
			w.WriteHeader(r.pageStatus)
			_, _ = w.Write([]byte("<h1>Sign-in refused</h1><p>" + r.reason + "</p>"))
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
	f.polls++
	s, ok := f.sessions[r.PathValue("id")]
	if !ok || s.secret != req.PollSecret {
		writeFakeError(w, 404, "unknown login session")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case s.status == hop.StatusRefused:
		// 403 for every refusal whatever the browser page's status was,
		// with the meaning in `code` and the browser's own sentence in
		// `reason`. No token, account or device.
		w.WriteHeader(403)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "refused", "code": s.refusal.code, "reason": s.refusal.reason})
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
	_ = get.Body.Close()
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
	_ = resp.Body.Close()
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
		_ = resp.Body.Close()
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
