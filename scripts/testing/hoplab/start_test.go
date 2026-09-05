package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRefuseInsideCheckoutDetectsGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "worktrees", "lab")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := refuseInsideCheckout(nested); err == nil {
		t.Fatal("refuseInsideCheckout: want an error for a path under a .git ancestor")
	}
}

func TestRefuseInsideCheckoutAllowsOutside(t *testing.T) {
	root := t.TempDir() // t.TempDir() is never inside this repo's checkout
	if err := refuseInsideCheckout(root); err != nil {
		t.Fatalf("refuseInsideCheckout(%s): %v, want nil", root, err)
	}
}

func TestNonEmpty(t *testing.T) {
	if got := nonEmpty("", "", "third"); got != "third" {
		t.Fatalf("nonEmpty = %q, want %q", got, "third")
	}
	if got := nonEmpty("first", "second"); got != "first" {
		t.Fatalf("nonEmpty = %q, want %q", got, "first")
	}
	if got := nonEmpty("", ""); got != "" {
		t.Fatalf("nonEmpty = %q, want empty", got)
	}
}

func TestWaitHealthySucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := waitHealthy(srv.URL+"/healthz", 2*time.Second); err != nil {
		t.Fatalf("waitHealthy: %v", err)
	}
}

func TestWaitHealthyTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	if err := waitHealthy(srv.URL+"/healthz", 300*time.Millisecond); err == nil {
		t.Fatal("waitHealthy: want an error when the server never returns 200")
	}
}

func TestWaitHealthyRetriesUntilUp(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := waitHealthy(srv.URL+"/healthz", 3*time.Second); err != nil {
		t.Fatalf("waitHealthy: %v", err)
	}
	if calls < 3 {
		t.Fatalf("calls = %d, want at least 3 (it should retry)", calls)
	}
}
