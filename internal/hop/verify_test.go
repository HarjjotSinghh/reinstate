package hop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyReferenceAndPostReport(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    any
		want    Reference
		wantErr error
	}{
		{"reference", 200, map[string]string{"endpoint": "https://s3.example", "bucket": "lk-ref", "region": "auto", "key": "reference/probe.txt"},
			Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Region: "auto", Key: "reference/probe.txt"}, nil},
		{"none", 404, map[string]string{"error": "no reference", "code": "no_reference"}, Reference{}, ErrNoReference},
		{"revoked", 401, map[string]string{"error": "unknown or revoked device token"}, Reference{}, ErrUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/verify/reference" || r.Header.Get("Authorization") != "Bearer hop_t" {
					t.Errorf("unexpected request %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(tc.body)
			}))
			defer srv.Close()
			got, err := New(srv.URL).VerifyReference(context.Background(), "hop_t")
			if !errors.Is(err, tc.wantErr) || got != tc.want {
				t.Fatalf("got %+v err=%v; want %+v err=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}

	t.Run("post report", func(t *testing.T) {
		var received []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/verify-reports" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			received, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "received_at": "2026-08-23T12:06:00Z"})
		}))
		defer srv.Close()
		receipt, err := New(srv.URL).PostVerifyReport(context.Background(), "hop_t", map[string]any{"version": 1, "outcome": "pass"})
		if err != nil || receipt.ID != 7 || receipt.ReceivedAt != "2026-08-23T12:06:00Z" {
			t.Fatalf("receipt %+v err=%v", receipt, err)
		}
		if string(received) != `{"outcome":"pass","version":1}` {
			t.Fatalf("body %s", received)
		}
	})
}
