package verify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// s3Error is a signed S3 refusal, the shape R2 answers with.
func s3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, code)
}

// openThrough builds the reference-locker probe the CLI builds: a real S3
// client over the redirect-refusing, exchange-recording client. It is the
// production path, so these tests exercise the pin end to end rather than
// a hand-written record of what the wire supposedly looked like.
func openThrough(t *testing.T) func(context.Context, hop.Reference) (Probe, error) {
	t.Helper()
	return func(ctx context.Context, ref hop.Reference) (Probe, error) {
		httpClient, exchanges := ProbeClient(nil)
		b, err := s3.New(ctx, s3.Config{
			Endpoint: ref.Endpoint, Region: "auto", Bucket: ref.Bucket,
			Credentials: s3.Static("AKIAHOP1", "secret"), HTTPClient: httpClient,
			MaxAttempts: 1,
		})
		if err != nil {
			return Probe{}, err
		}
		return Probe{Backend: b, AccessKeyID: "AKIAHOP1", Exchanges: exchanges}, nil
	}
}

// TestIsolationIsPinnedToTheResponse: the isolation step's verdict must
// come from the answer this account's credential actually received at the
// endpoint step 1 listed, not from two strings the control plane supplied.
// Each case is a whole reference host behaving one way, driven through the
// real S3 client.
func TestIsolationIsPinnedToTheResponse(t *testing.T) {
	keys := rootKeys(t)
	tests := []struct {
		name string
		// handler serves the reference host; nil means the host is the
		// locker's own endpoint (the honest R2 case).
		handler http.HandlerFunc
		// closed shuts the reference host down before the run, which is
		// what a dead endpoint looks like from here.
		closed bool
		// refEndpoint rewrites the endpoint the control plane advertises,
		// given the reference host's own URL.
		refEndpoint func(string) string
		status      Status
		want        string
	}{
		{
			name:    "signed access denied at the pinned host",
			handler: func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusForbidden, "AccessDenied") },
			status:  Pass,
			want:    "Listing the reference locker was refused as access denied.",
		},
		{
			// Any web server answers 403. Only an S3 refusal naming its
			// code shows that a bucket refused this credential.
			name:    "bodiless 403",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
			status:  NotApplicable,
			want:    "answered 403 with no S3 error body",
		},
		{
			name:    "404",
			handler: func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusNotFound, "NoSuchBucket") },
			status:  Fail,
			want:    "neither succeeded nor was refused as access denied",
		},
		{
			name: "500",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				s3Error(w, http.StatusInternalServerError, "InternalError")
			},
			status: Fail,
			want:   "neither succeeded nor was refused as access denied",
		},
		{
			name:   "connection refused",
			closed: true,
			status: Fail,
			want:   "neither succeeded nor was refused as access denied",
		},
		{
			name:        "same host, different port",
			handler:     func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusForbidden, "AccessDenied") },
			refEndpoint: func(u string) string { return withPort(u, "9") },
			status:      Fail,
			want:        "A refusal from a different endpoint proves nothing",
		},
		{
			name:        "scheme case and trailing slash only",
			handler:     func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusForbidden, "AccessDenied") },
			refEndpoint: func(u string) string { return strings.Replace(u, "http://", "HTTP://", 1) + "/" },
			status:      Pass,
			want:        "Listing the reference locker was refused as access denied.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := tc.handler
			if handler == nil {
				handler = func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusForbidden, "AccessDenied") }
			}
			srv := httptest.NewServer(handler)
			defer srv.Close()
			if tc.closed {
				srv.Close()
			}
			refEndpoint := srv.URL
			if tc.refEndpoint != nil {
				refEndpoint = tc.refEndpoint(srv.URL)
			}
			store := lockerWith(t, keys, "")
			r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
				// Step 1 listed this account's locker at the same host the
				// reference is pinned to, which is the honest arrangement.
				Locker:        LockerInfo{Endpoint: srv.URL, Bucket: "lk-1"},
				Reference:     &hop.Reference{Endpoint: refEndpoint, Bucket: "lk-ref", Key: "reference/probe.txt"},
				OpenReference: openThrough(t), CredentialID: credential("AKIAHOP1")})
			step := r.Steps[3]
			if step.Status != tc.status || !strings.Contains(step.Observed, tc.want) {
				t.Fatalf("isolation %s: %q\nwant %s containing %q", step.Status, step.Observed, tc.status, tc.want)
			}
			// Only a pass may claim isolation, and only a fail may fail the
			// run: an inconclusive probe leaves the outcome alone and says
			// nothing.
			if r.IsolationChecked() != (tc.status == Pass) {
				t.Fatalf("IsolationChecked()=%v for a %s step", r.IsolationChecked(), tc.status)
			}
			if want := Pass; tc.status == Fail {
				want = Fail
				if r.Outcome != want {
					t.Fatalf("outcome %s, want %s", r.Outcome, want)
				}
			} else if r.Outcome != want {
				t.Fatalf("outcome %s, want %s", r.Outcome, want)
			}
			if tc.status == NotApplicable && strings.Contains(r.Summary(), "refused by a bucket") {
				t.Fatalf("an inconclusive probe still claimed isolation: %q", r.Summary())
			}
		})
	}
}

// TestIsolationRefusesARedirectedProbe is the hole the response pin closes.
// The control plane names its own storage endpoint, exactly as step 1
// listed it, so every string compare agrees — and that endpoint answers the
// probe with a redirect to a host that refuses everything with a
// well-formed AccessDenied. Following it would produce a textbook passing
// isolation step in which this account's credential was never offered to a
// bucket at all. The probe refuses the hop, the step fails, and the
// redirect target is never contacted.
func TestIsolationRefusesARedirectedProbe(t *testing.T) {
	keys := rootKeys(t)
	var elsewhere int
	always403 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		elsewhere++
		s3Error(w, http.StatusForbidden, "AccessDenied")
	}))
	defer always403.Close()
	pinned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", always403.URL+r.URL.Path)
		w.WriteHeader(http.StatusFound)
	}))
	defer pinned.Close()

	store := lockerWith(t, keys, "")
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: pinned.URL, Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: pinned.URL, Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openThrough(t), CredentialID: credential("AKIAHOP1")})
	step := r.Steps[3]
	if step.Status != Fail || r.Outcome != Fail || r.IsolationChecked() {
		t.Fatalf("a redirected probe passed isolation: %+v", step)
	}
	if !strings.Contains(step.Observed, "answered with a redirect to "+always403.URL) {
		t.Fatalf("the report does not say the probe was redirected: %q", step.Observed)
	}
	if elsewhere != 0 {
		t.Fatalf("the locker credential was sent to the redirect target %d time(s)", elsewhere)
	}
	if strings.Contains(r.Summary(), "refused by a bucket that is not its own") {
		t.Fatalf("summary claims isolation: %q", r.Summary())
	}
}

// TestProbeClientRefusesRedirects: the credential must never be sent to a
// host the control plane did not pin, so the redirect is refused at the
// client and the hop is recorded rather than followed.
func TestProbeClientRefusesRedirects(t *testing.T) {
	var elsewhere []string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhere = append(elsewhere, r.Header.Get("Authorization"))
		s3Error(w, http.StatusForbidden, "AccessDenied")
	}))
	defer target.Close()
	pinned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL+r.URL.Path)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer pinned.Close()

	client, exchanges := ProbeClient(nil)
	resp, err := client.Get(pinned.URL + "/lk-ref/reference/probe.txt")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("the probe followed a redirect")
	}
	if !strings.Contains(err.Error(), "refused to follow a redirect") {
		t.Fatalf("error %v", err)
	}
	if len(elsewhere) != 0 {
		t.Fatalf("the redirect target was contacted: %v", elsewhere)
	}
	seen := exchanges()
	if len(seen) != 1 || seen[0].RedirectedTo == "" || seen[0].Host != hostOf(pinned.URL) {
		t.Fatalf("exchanges %+v", seen)
	}
}

// TestProbeClientRecordsTheErrorCodeAndLeavesTheBody: the transport reads
// the refusal body to learn whether it is a signed S3 error, and must hand
// the same bytes on to the S3 client that asked for them.
func TestProbeClientRecordsTheErrorCodeAndLeavesTheBody(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		code    string
	}{
		{"signed refusal", func(w http.ResponseWriter, _ *http.Request) { s3Error(w, http.StatusForbidden, "AccessDenied") }, "AccessDenied"},
		{"bodiless 403", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }, ""},
		{"403 that is not S3 at all", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("<html><body>Forbidden</body></html>"))
		}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			client, exchanges := ProbeClient(nil)
			resp, err := client.Get(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			seen := exchanges()
			if len(seen) != 1 || seen[0].Status != http.StatusForbidden || seen[0].ErrorCode != tc.code {
				t.Fatalf("exchanges %+v", seen)
			}
			var want strings.Builder
			tc.handler(recorder{&want}, nil)
			if string(body) != want.String() {
				t.Fatalf("body handed on as %q, want %q", body, want.String())
			}
		})
	}
}

// recorder captures what a handler writes so a test can compare the bytes
// the probe handed on against the bytes the endpoint sent.
type recorder struct{ out *strings.Builder }

func (recorder) Header() http.Header           { return http.Header{} }
func (recorder) WriteHeader(int)               {}
func (r recorder) Write(p []byte) (int, error) { return r.out.Write(p) }

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// withPort replaces an endpoint's port, so a test can point the reference
// at a host that differs from the locker's endpoint by port alone.
func withPort(rawURL, port string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Host = u.Hostname() + ":" + port
	return u.String()
}
