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
	if f.reference == nil {
		writeFakeErrorCode(w, 404, "no_reference", "this control plane has no reference locker; the isolation check cannot run here")
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"endpoint": f.s3.URL(), "bucket": f.reference.bucket, "region": "auto", "key": f.reference.key,
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
