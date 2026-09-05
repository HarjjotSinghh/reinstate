package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// The fixtures used below (hopHarness, newHopHarness, fakeControlPlane, ...)
// live in hop_test.go because they are shared across the wider hop CLI test
// surface, not exclusive to login and whoami.

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
			// http://127.0.0.1:1 refuses the connection outright (nothing
			// ever listens on port 1), which is one of the three transport
			// failures T-202 classifies.
			// TestLoginAndWhoamiControlPlaneUnreachable below pins the full
			// message and the --json form; this row only holds the
			// family's exit code and message start steady alongside the
			// rest of the failure table.
			name:     "unreachable control plane",
			args:     []string{"login"},
			prepare:  func(h *hopHarness) { h.t.Setenv(hop.URLEnv, "http://127.0.0.1:1") },
			wantCode: ExitRuntime,
			wantErr:  "could not reach the Reinstate Hop control plane at",
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

// TestLoginAndWhoamiControlPlaneUnreachable is T-202: a control plane this
// client cannot reach at all — no DNS answer, no route, no TLS handshake —
// must read as one timeless sentence naming the URL and the cause, not
// whatever text net/http happened to produce, and the exit code must not
// move. http://127.0.0.1:1 refuses the connection outright everywhere this
// suite runs (nothing listens on port 1), which is the deterministic half
// of the two repro cases the task card names; the DNS-error dialer is
// exercised at the internal/hop package level
// (TestClassifyUnreachableRecognizesEachTransportFailure), where a custom
// Transport can be injected without going through the CLI's own URL
// resolution.
func TestLoginAndWhoamiControlPlaneUnreachable(t *testing.T) {
	const unreachableURL = "http://127.0.0.1:1"
	wantFirstLine := "could not reach the Reinstate Hop control plane at " + unreachableURL + ": connection refused"
	wantSecondLine := "If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. " +
		"To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml."

	assertUnreachableJSON := func(t *testing.T, errb string, code int) {
		t.Helper()
		if code != ExitRuntime {
			t.Fatalf("exit=%d, want ExitRuntime; stderr=%q", code, errb)
		}
		var e ErrorJSON
		if err := json.Unmarshal([]byte(errb), &e); err != nil {
			t.Fatalf("decode %v: %s", err, errb)
		}
		if !strings.Contains(e.Message, wantFirstLine) {
			t.Fatalf("message = %q, want it to contain %q", e.Message, wantFirstLine)
		}
		if !strings.Contains(e.Message, wantSecondLine) {
			t.Fatalf("message = %q, want it to contain %q", e.Message, wantSecondLine)
		}
		if e.Details["kind"] != hop.KindControlPlaneUnreachable {
			t.Fatalf("details.kind = %v, want %q", e.Details["kind"], hop.KindControlPlaneUnreachable)
		}
		if e.Details["url"] != unreachableURL {
			t.Fatalf("details.url = %v, want %q", e.Details["url"], unreachableURL)
		}
	}

	t.Run("login", func(t *testing.T) {
		h := newHopHarness(t)
		h.t.Setenv(hop.URLEnv, unreachableURL)
		_, errb, code := h.run("login", "--json")
		assertUnreachableJSON(t, errb, code)
		if _, err := h.tokens.GetDeviceToken(); err == nil {
			t.Fatal("an unreachable control plane must not store a token")
		}
	})

	t.Run("whoami", func(t *testing.T) {
		h := newHopHarness(t)
		if err := h.tokens.SetDeviceToken(credentials.DeviceToken{
			Token: "hop_t", ControlPlaneURL: unreachableURL, AccountID: "a", DeviceID: "d",
		}); err != nil {
			t.Fatal(err)
		}
		_, errb, code := h.run("whoami", "--json")
		assertUnreachableJSON(t, errb, code)
	})
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
