package hop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocationHintForZone(t *testing.T) {
	tests := map[string]string{
		"Asia/Kolkata":        LocationAPAC,
		"Asia/Tokyo":          LocationAPAC,
		"Asia/Singapore":      LocationAPAC,
		"Europe/London":       LocationWestEU,
		"Europe/Berlin":       LocationWestEU,
		"Europe/Warsaw":       LocationEastEU,
		"Europe/Helsinki":     LocationEastEU,
		"America/New_York":    LocationEastNA,
		"America/Chicago":     LocationEastNA,
		"America/Los_Angeles": LocationWestNA,
		"America/Denver":      LocationWestNA,
		"Pacific/Honolulu":    LocationWestNA,
		"Australia/Sydney":    LocationOceania,
		"Pacific/Auckland":    LocationOceania,
		"Africa/Lagos":        LocationWestEU,
		"UTC":                 LocationAPAC,
		"":                    LocationAPAC,
		"Local":               LocationAPAC,
	}
	for zone, want := range tests {
		if got := LocationHintForZone(zone); got != want {
			t.Errorf("%q: got %s want %s", zone, got, want)
		}
	}
}

func TestLocationHintOverride(t *testing.T) {
	t.Setenv(LocationEnv, " WEUR ")
	if got := LocationHint(); got != LocationWestEU {
		t.Fatalf("override: %s", got)
	}
	t.Setenv(LocationEnv, "")
	t.Setenv("TZ", "America/New_York")
	if got := LocationHint(); got != LocationEastNA {
		t.Fatalf("from TZ: %s", got)
	}
}

func TestLockerErrorsAreTyped(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   map[string]string
		check  func(error) bool
	}{
		{"storage quota", 403, map[string]string{"error": "full", "code": "quota_storage"}, func(err error) bool {
			var qe *QuotaError
			return errors.As(err, &qe) && qe.Kind == QuotaStorage && qe.Message == "full"
		}},
		{"device quota", 403, map[string]string{"error": "too many", "code": "quota_devices"}, func(err error) bool {
			var qe *QuotaError
			return errors.As(err, &qe) && qe.Kind == QuotaDevices
		}},
		{"push rate", 429, map[string]string{"error": "slow down", "code": "quota_push_rate"}, func(err error) bool {
			var qe *QuotaError
			return errors.As(err, &qe) && qe.Kind == QuotaPushRate
		}},
		{"no locker", 404, map[string]string{"error": "none", "code": "no_locker"}, func(err error) bool { return errors.Is(err, ErrNoLocker) }},
		{"provider down", 502, map[string]string{"error": "down", "code": "storage_unavailable"}, func(err error) bool { return errors.Is(err, ErrStorageUnavailable) }},
		{"revoked token", 401, map[string]string{"error": "unknown or revoked device token"}, func(err error) bool { return errors.Is(err, ErrUnauthorized) }},
		{"other", 500, map[string]string{"error": "boom"}, func(err error) bool {
			var he *Error
			return errors.As(err, &he) && he.Status == 500
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer hop_x" {
					t.Errorf("missing bearer on %s", r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(tc.body)
			}))
			defer srv.Close()
			c := New(srv.URL)
			ctx := context.Background()
			if _, err := c.MintCredentials(ctx, "hop_x"); !tc.check(err) {
				t.Fatalf("mint: %v", err)
			}
			if _, err := c.ProvisionLocker(ctx, "hop_x"); !tc.check(err) {
				t.Fatalf("provision: %v", err)
			}
			if _, err := c.LockerStatus(ctx, "hop_x"); !tc.check(err) {
				t.Fatalf("status: %v", err)
			}
		})
	}
}

func TestSourceProvisionsOnceAndMintsPerCall(t *testing.T) {
	var provisions, mints int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/locker":
			provisions++
			_ = json.NewEncoder(w).Encode(map[string]any{"endpoint": "https://x", "bucket": "lk-1", "region": "auto"})
		case "/v1/locker/credentials":
			mints++
			_ = json.NewEncoder(w).Encode(map[string]any{"access_key_id": "A", "secret_access_key": "S", "session_token": "T", "expires_at": "2026-08-23T13:00:00Z"})
		case "/v1/locker/first-push":
			_ = json.NewEncoder(w).Encode(map[string]any{"first": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := NewSource(New(srv.URL), "hop_x")
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if l, err := s.Locker(ctx); err != nil || l.Bucket != "lk-1" {
			t.Fatalf("locker: %+v %v", l, err)
		}
	}
	if provisions != 1 {
		t.Fatalf("provisioned %d times", provisions)
	}
	c, err := s.Credentials(ctx)
	if err != nil || c.AccessKeyID != "A" || c.SessionToken != "T" || !c.Expires.Equal(time.Date(2026, 8, 23, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("credentials %+v %v", c, err)
	}
	if _, err := s.Credentials(ctx); err != nil {
		t.Fatal(err)
	}
	if mints != 2 || s.Mints() != 2 || s.LastError() != nil {
		t.Fatalf("mints=%d source=%d last=%v", mints, s.Mints(), s.LastError())
	}
	if err := s.ReportFirstPush(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ReportFirstPush(ctx); err != nil {
		t.Fatal(err)
	}
}
