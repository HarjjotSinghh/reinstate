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
