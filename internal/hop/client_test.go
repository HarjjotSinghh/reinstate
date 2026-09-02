package hop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name string
		env  string
		cfg  *schema.Config
		want string
	}{
		{"default", "", nil, DefaultURL},
		{"config", "", &schema.Config{Hop: schema.HopConfig{URL: "https://staging.example/"}}, "https://staging.example"},
		{"env wins", "http://127.0.0.1:8080/", &schema.Config{Hop: schema.HopConfig{URL: "https://staging.example"}}, "http://127.0.0.1:8080"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(URLEnv, tc.env)
			if got := ResolveURL(tc.cfg); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestWaitForApprovalFollowsStatuses(t *testing.T) {
	tests := []struct {
		name    string
		answers []func(w http.ResponseWriter)
		wantErr string
	}{
		{
			name: "pending then approved",
			answers: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) { _ = json.NewEncoder(w).Encode(map[string]string{"status": "pending"}) },
				func(w http.ResponseWriter) {
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "approved", "device_token": "hop_t",
						"account": Account{ID: "a"}, "device": Device{ID: "d"}})
				},
			},
		},
		{
			name: "expired",
			answers: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					w.WriteHeader(http.StatusGone)
					_ = json.NewEncoder(w).Encode(map[string]string{"status": "expired"})
				},
			},
			wantErr: "expired",
		},
		{
			name: "approved without token",
			answers: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) { _ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"}) },
			},
			wantErr: "without a device token",
		},
		{
			name: "wrong secret",
			answers: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "unknown login session"})
				},
			},
			wantErr: "unknown login session",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/login/sessions/s1/poll" {
					t.Errorf("path %s", r.URL.Path)
				}
				tc.answers[i](w)
				i++
			}))
			defer srv.Close()
			slept := 0
			approval, err := New(srv.URL).WaitForApproval(context.Background(), LoginSession{ID: "s1", PollSecret: "p"},
				func(context.Context, time.Duration) error { slept++; return nil })
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || approval == nil || approval.DeviceToken != "hop_t" || slept != 1 {
				t.Fatalf("approval=%+v err=%v slept=%d", approval, err, slept)
			}
		})
	}
}

func TestWhoamiUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hop_ok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(Identity{Account: Account{ID: "a"}, Device: Device{ID: "d"}})
	}))
	defer srv.Close()
	c := New(srv.URL)
	if _, err := c.Whoami(context.Background(), "hop_bad"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err=%v", err)
	}
	id, err := c.Whoami(context.Background(), "hop_ok")
	if err != nil || id.Account.ID != "a" || id.Device.ID != "d" {
		t.Fatalf("%+v err=%v", id, err)
	}
}

func TestCreatePairingWritesIntegerVersion2(t *testing.T) {
	var received struct {
		Version   json.RawMessage `json:"version"`
		PublicKey string          `json:"public_key"`
		Salt      string          `json:"salt"`
		Binding   string          `json:"binding"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/pairing" || r.Header.Get("Authorization") != "Bearer hop_pair" {
			t.Errorf("request %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(PairingRequest{ID: "pair-1", Status: PairingPending, Version: PairingVersion2})
	}))
	defer srv.Close()

	req, err := New(srv.URL).CreatePairing(context.Background(), "hop_pair", "age1public", "c2FsdA==", "YmluZGluZw==")
	if err != nil || req.ID != "pair-1" || req.Version != PairingVersion2 {
		t.Fatalf("request = %+v, %v", req, err)
	}
	if string(received.Version) != "2" {
		t.Fatalf("version on wire = %s, want JSON integer 2", received.Version)
	}
	if received.PublicKey != "age1public" || received.Salt != "c2FsdA==" || received.Binding != "YmluZGluZw==" {
		t.Fatalf("body = %+v", received)
	}
}

func TestPairingWireVersionCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want int
		err  bool
	}{
		{name: "missing means v1", raw: `{"id":"legacy"}`, want: PairingVersion1},
		{name: "zero means v1", raw: `{"id":"legacy","version":0}`, want: PairingVersion1},
		{name: "explicit v1", raw: `{"id":"legacy","version":1}`, want: PairingVersion1},
		{name: "v2", raw: `{"id":"current","version":2}`, want: PairingVersion2},
		{name: "unsupported", raw: `{"id":"future","version":3}`, err: true},
		{name: "string is not an integer", raw: `{"id":"wrong","version":"2"}`, err: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var req PairingRequest
			decodeErr := json.Unmarshal([]byte(tc.raw), &req)
			if tc.name == "string is not an integer" {
				if decodeErr == nil {
					t.Fatal("string version decoded as an integer")
				}
				return
			}
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			got, err := req.ProtocolVersion()
			if tc.err {
				if err == nil {
					t.Fatalf("version %d accepted", req.Version)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ProtocolVersion() = %d, %v; want %d", got, err, tc.want)
			}
		})
	}
}

func TestDeviceRevocationRequestWire(t *testing.T) {
	var confirmed int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hop_device" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/device-revocation-requests":
			_ = json.NewEncoder(w).Encode(map[string]any{"requests": []DeviceRevocationRequest{{
				ID: "revoke-1", Status: RevocationRequestPending,
				Target: Device{ID: "dev-lost", Name: "desktop"}, RequestedGeneration: 2,
			}}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/device-revocation-requests/revoke-1/confirm":
			var body struct {
				Generation int `json:"generation"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			confirmed = body.Generation
			_ = json.NewEncoder(w).Encode(DeviceRevocationRequest{
				ID: "revoke-1", Status: RevocationRequestConfirmed,
				Target:              Device{ID: "dev-lost", Name: "desktop", RevokedAt: "2026-09-02T00:00:00Z"},
				RequestedGeneration: 2, ConfirmedGeneration: body.Generation,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := New(srv.URL)
	requests, err := client.PendingDeviceRevocations(context.Background(), "hop_device")
	if err != nil || len(requests) != 1 || requests[0].Target.ID != "dev-lost" || requests[0].RequestedGeneration != 2 {
		t.Fatalf("requests = %+v, %v", requests, err)
	}
	result, err := client.ConfirmDeviceRevocation(context.Background(), "hop_device", "revoke-1", 3)
	if err != nil || confirmed != 3 || result.Status != RevocationRequestConfirmed || !result.Target.Revoked() {
		t.Fatalf("confirm = %+v, generation=%d, err=%v", result, confirmed, err)
	}
}

func TestPendingDeviceRevocationsTreatsAnOldControlPlaneAsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	requests, err := New(srv.URL).PendingDeviceRevocations(context.Background(), "hop_device")
	if err != nil || len(requests) != 0 {
		t.Fatalf("requests = %+v, %v", requests, err)
	}
}

func TestPairingVersionIsConfirmedAcrossTheRelay(t *testing.T) {
	t.Run("create downgrade", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PairingRequest{ID: "pair-1"})
		}))
		defer srv.Close()
		if _, err := New(srv.URL).CreatePairing(context.Background(), "hop_pair", "age1public", "c2FsdA==", "YmluZGluZw=="); err == nil || !strings.Contains(err.Error(), "version 1") {
			t.Fatalf("v2 create accepted a v1 response: %v", err)
		}
	})

	t.Run("claim downgrade", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": PairingApproved, "version": PairingVersion1, "payload": "ciphertext", "key_generation": 1})
		}))
		defer srv.Close()
		req := PairingRequest{ID: "pair-2", Version: PairingVersion2, IntervalSeconds: 1}
		if _, err := New(srv.URL).WaitForPairing(context.Background(), "hop_pair", req, func(context.Context, time.Duration) error { return nil }); err == nil || !strings.Contains(err.Error(), "version 1, want version 2") {
			t.Fatalf("v2 claim accepted a v1 response: %v", err)
		}
	})

	for _, version := range []int{PairingVersion1, PairingVersion2} {
		t.Run("approve v"+string(rune('0'+version)), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				response := map[string]any{"status": PairingApproved}
				if version == PairingVersion2 {
					response["version"] = PairingVersion2
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer srv.Close()
			if err := New(srv.URL).ApprovePairing(context.Background(), "hop_pair", "pair-3", "ciphertext", 1, version); err != nil {
				t.Fatalf("approve v%d: %v", version, err)
			}
		})
	}
}
