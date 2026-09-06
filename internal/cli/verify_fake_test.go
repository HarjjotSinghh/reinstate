package cli

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/HarjjotSinghh/reinstate/internal/verify"
)

// fakeReference is the reference locker the fake control plane advertises.
// The fake S3 refuses every bucket but its own with AccessDenied, so a
// reference bucket that is simply "another name" behaves like R2.
type fakeReference struct {
	bucket, key string
	// endpoint overrides the storage endpoint advertised for the reference
	// locker; empty means the fake S3's own URL, which is what a correctly
	// configured control plane answers.
	endpoint string
}

// fakeReport is one posted verification report with the token that sent it.
type fakeReport struct {
	token  string
	raw    []byte
	report verify.Upload
}

func (f *fakeControlPlane) registerVerify(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/verify/reference", f.verifyReference)
	mux.HandleFunc("POST /v1/verify-reports", f.postVerifyReport)
}

func (f *fakeControlPlane) verifyReference(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	if f.referenceStatus != 0 {
		// An operator-side fault: the row is unreadable, the storage
		// provider is down, the handler panicked. The client half of the
		// service has to survive it without telling every account that its
		// locker failed a security check.
		writeFakeError(w, f.referenceStatus, "internal error")
		return
	}
	if f.reference == nil {
		writeFakeErrorCode(w, 404, "no_reference", "this control plane has no reference locker; the isolation check cannot run here")
		return
	}
	endpoint := f.reference.endpoint
	if endpoint == "" {
		endpoint = f.s3.URL()
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"endpoint": endpoint, "bucket": f.reference.bucket, "region": "auto", "key": f.reference.key,
	})
}

func (f *fakeControlPlane) postVerifyReport(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authed(w, r) {
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	var up verify.Upload
	if err := json.Unmarshal(raw, &up); err != nil || up.Version != 1 || len(up.Steps) == 0 {
		writeFakeError(w, 400, "malformed report")
		return
	}
	f.reports = append(f.reports, fakeReport{token: r.Header.Get("Authorization"), raw: raw, report: up})
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": len(f.reports), "received_at": "2026-08-23T12:06:00Z"})
}
