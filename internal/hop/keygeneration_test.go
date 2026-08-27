package hop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestKeyGenerationFloorAnswers is the table of answers the client has to
// tell apart, because each one leads somewhere different: a floor it must
// enforce, a control plane that carries none (fall back to the last one
// this device confirmed), a token the control plane refuses (the device is
// revoked or stale), and an answer that is not usable as a floor at all.
func TestKeyGenerationFloorAnswers(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    any
		want    int
		wantErr error
		// anyErr marks a case where the error is not one of the sentinels
		// but must still not be reported as a successful read.
		anyErr bool
	}{
		{name: "a floor", status: 200, body: map[string]any{"key_generation": 4, "updated_at": "2026-08-27T00:00:00Z"}, want: 4},
		{name: "never raised", status: 200, body: map[string]any{"key_generation": 0}, want: 0},
		{name: "route not served", status: 404, body: map[string]any{"error": "not found"}, wantErr: ErrNoKeyGenerationFloor},
		{name: "route not served, by code", status: 400, body: map[string]any{"code": CodeNoKeyGeneration, "error": "no floor"}, wantErr: ErrNoKeyGenerationFloor},
		{name: "token refused", status: 401, body: map[string]any{"error": "revoked"}, wantErr: ErrUnauthorized},
		{name: "negative generation", status: 200, body: map[string]any{"key_generation": -1}, anyErr: true},
		{name: "control plane error", status: 503, body: map[string]any{"error": "down"}, anyErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != KeyGenerationPath || r.Method != http.MethodGet {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(tc.body)
			}))
			defer srv.Close()
			got, err := New(srv.URL).KeyGenerationFloor(context.Background(), "hop_token")
			switch {
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
			case tc.anyErr:
				if err == nil {
					t.Fatalf("accepted %v", tc.body)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Generation != tc.want {
					t.Fatalf("generation = %d, want %d", got.Generation, tc.want)
				}
			}
		})
	}
}

// TestRaiseKeyGenerationFloorRefusesAnAnswerBelowWhatItSent: the floor is
// monotonic on the control plane, so an answer below the number just
// reported means the party answering is not keeping it monotonic. Taking
// that at face value would let a control plane that also writes the bucket
// quietly turn the floor off; refusing keeps the caller's report honest.
func TestRaiseKeyGenerationFloorRefusesAnAnswerBelowWhatItSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"key_generation": 1})
	}))
	defer srv.Close()
	if _, err := New(srv.URL).RaiseKeyGenerationFloor(context.Background(), "hop_token", 3); err == nil {
		t.Fatal("accepted a floor below the generation reported")
	}
}

// TestRaiseKeyGenerationFloorSendsTheGeneration pins the request body and
// the two guards around it: a generation below 1 is never sent, and the
// answer is the floor as it stands after the call.
func TestRaiseKeyGenerationFloorSendsTheGeneration(t *testing.T) {
	var seen map[string]int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != KeyGenerationPath {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&seen)
		_ = json.NewEncoder(w).Encode(map[string]any{"key_generation": 7, "updated_at": "2026-08-27T00:00:00Z"})
	}))
	defer srv.Close()
	client := New(srv.URL)
	if _, err := client.RaiseKeyGenerationFloor(context.Background(), "hop_token", 0); err == nil {
		t.Fatal("reported generation 0 to the control plane")
	}
	if seen != nil {
		t.Fatalf("generation 0 reached the control plane: %v", seen)
	}
	got, err := client.RaiseKeyGenerationFloor(context.Background(), "hop_token", 5)
	if err != nil {
		t.Fatal(err)
	}
	if seen["key_generation"] != 5 {
		t.Fatalf("request body %v", seen)
	}
	if got.Generation != 7 {
		t.Fatalf("generation = %d, want the 7 the control plane holds", got.Generation)
	}
}
