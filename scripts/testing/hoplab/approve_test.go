package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const sampleLog = `2026/09/05 10:00:00 hopd listening on 127.0.0.1:8082
hopd: email to first-push-1@example.com — Sign in to Reinstate Hop
Open this link to sign in the device "acceptance-laptop" (windows-amd64):

http://127.0.0.1:8082/login/email/oezM6epQabc123

The link expires at 2026-09-05T10:10:00Z. If you did not run ` + "`rein login`" + `, ignore this email.
2026/09/05 10:00:05 200 POST /v1/login/sessions
hopd: email to first-push-1@example.com — Sign in to Reinstate Hop
Open this link to sign in the device "acceptance-laptop-wiped" (windows-amd64):

http://127.0.0.1:8082/login/email/KpJWcclpdef456

The link expires at 2026-09-05T10:10:05Z. If you did not run ` + "`rein login`" + `, ignore this email.
`

func TestParseSignInEmails(t *testing.T) {
	got := ParseSignInEmails(sampleLog)
	if len(got) != 2 {
		t.Fatalf("got %d emails, want 2: %+v", len(got), got)
	}
	if got[0].To != "first-push-1@example.com" || got[0].Device != "acceptance-laptop" || got[0].Link != "http://127.0.0.1:8082/login/email/oezM6epQabc123" {
		t.Fatalf("email 0 = %+v", got[0])
	}
	if got[1].Device != "acceptance-laptop-wiped" || got[1].Link != "http://127.0.0.1:8082/login/email/KpJWcclpdef456" {
		t.Fatalf("email 1 = %+v", got[1])
	}
}

func TestParseSignInEmailsEmpty(t *testing.T) {
	if got := ParseSignInEmails("nothing here\n"); len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

// fakeHopd is a minimal stand-in for hopd's two sign-in-link routes, so the
// approver is tested against a fake hopd handler rather than a real hopd
// binary.
type fakeHopd struct {
	mu       sync.Mutex
	gets     []string
	posts    []string
	postCode int
}

func newFakeHopd() *fakeHopd { return &fakeHopd{postCode: http.StatusOK} }

func (f *fakeHopd) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			f.gets = append(f.gets, r.URL.Path)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>confirm</html>"))
		case http.MethodPost:
			f.posts = append(f.posts, r.URL.Path)
			w.WriteHeader(f.postCode)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func TestApproverApprove(t *testing.T) {
	fake := newFakeHopd()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	a := &Approver{}
	result, err := a.Approve(context.Background(), srv.URL+"/login/email/tok1", false)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if result != ResultApproved {
		t.Fatalf("result = %q, want approved", result)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.gets) != 1 || len(fake.posts) != 1 {
		t.Fatalf("gets=%v posts=%v, want exactly one GET then one POST", fake.gets, fake.posts)
	}
}

func TestApproverRefuseNeverPosts(t *testing.T) {
	fake := newFakeHopd()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	a := &Approver{}
	result, err := a.Approve(context.Background(), srv.URL+"/login/email/tok2", true)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if result != ResultDeclined {
		t.Fatalf("result = %q, want declined", result)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.gets) != 1 {
		t.Fatalf("gets=%v, want exactly one", fake.gets)
	}
	if len(fake.posts) != 0 {
		t.Fatalf("posts=%v, want none: a refusal must never approve", fake.posts)
	}
}

func TestApproverPostFailurePropagates(t *testing.T) {
	fake := newFakeHopd()
	fake.postCode = http.StatusGone // e.g. the link already expired
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	a := &Approver{}
	if _, err := a.Approve(context.Background(), srv.URL+"/login/email/tok3", false); err == nil {
		t.Fatal("Approve: want an error when POST does not return 200")
	} else if !strings.Contains(err.Error(), "410") && !strings.Contains(err.Error(), "Gone") {
		t.Fatalf("error = %v, want it to name the status", err)
	}
}

func TestApproverGetFailureNeverPosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := &Approver{}
	if _, err := a.Approve(context.Background(), srv.URL+"/login/email/expired", false); err == nil {
		t.Fatal("Approve: want an error when the confirm page 404s")
	}
}

func TestRunApprove(t *testing.T) {
	fake := newFakeHopd()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	log := strings.ReplaceAll(sampleLog, "http://127.0.0.1:8082", srv.URL)
	dir := t.TempDir()
	logPath := dir + "/hopd.log"
	if err := os.WriteFile(logPath, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err := runApprove(context.Background(), approveOptions{LogPath: logPath, Count: 2, Timeout: 5 * time.Second}, &out, &Approver{})
	if err != nil {
		t.Fatalf("runApprove: %v", err)
	}
	if strings.Count(out.String(), "approved") != 2 {
		t.Fatalf("output = %q, want two approvals", out.String())
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.posts) != 2 {
		t.Fatalf("posts=%v, want 2", fake.posts)
	}
}

func TestRunApproveFiltersByEmail(t *testing.T) {
	fake := newFakeHopd()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	log := strings.ReplaceAll(sampleLog, "http://127.0.0.1:8082", srv.URL)
	log = strings.Replace(log, "first-push-1@example.com", "someone-else@example.com", 1) // only the FIRST occurrence (the first block's To)
	dir := t.TempDir()
	logPath := dir + "/hopd.log"
	if err := os.WriteFile(logPath, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err := runApprove(context.Background(), approveOptions{LogPath: logPath, Email: "first-push-1@example.com", Count: 1, Timeout: 5 * time.Second}, &out, &Approver{})
	if err != nil {
		t.Fatalf("runApprove: %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.posts) != 1 {
		t.Fatalf("posts=%v, want exactly 1 (the second email, addressed to first-push-1@example.com)", fake.posts)
	}
}
