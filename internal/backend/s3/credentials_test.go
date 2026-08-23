package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	reinsync "github.com/HarjjotSinghh/reinstate/internal/sync"
)

// fakeSource hands out a scripted sequence of credentials and records how
// often it was asked. The last entry repeats once the script runs out.
type fakeSource struct {
	mu     sync.Mutex
	script []Credentials
	calls  int
	err    error
}

func (f *fakeSource) Credentials(context.Context) (Credentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return Credentials{}, f.err
	}
	i := f.calls - 1
	if i >= len(f.script) {
		i = len(f.script) - 1
	}
	return f.script[i], nil
}

func (f *fakeSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func hourly(akid string) Credentials {
	return Credentials{AccessKeyID: akid, SecretAccessKey: "secret-" + akid, Expires: time.Now().Add(time.Hour)}
}

func signedBy(log []string, akid string) int {
	n := 0
	for _, l := range log {
		if strings.HasSuffix(l, " as "+akid) {
			n++
		}
	}
	return n
}

func TestStaticCredentialsBehaveAsBefore(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "access key fields", cfg: Config{AccessKey: "AKIASTATIC", SecretKey: "s"}},
		{name: "explicit static source", cfg: Config{Credentials: Static("AKIASTATIC", "s")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeS3(t, "reinstate")
			f.accept("AKIASTATIC")
			c := f.client(t, tc.cfg)
			if c.refreshable {
				t.Fatal("static keys must not be marked refreshable")
			}
			ctx := context.Background()
			meta, err := c.Put(ctx, "manifest.age", strings.NewReader("v1"), 2, backend.PutOptions{IfNoneMatch: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Put(ctx, "manifest.age", strings.NewReader("v2"), 2, backend.PutOptions{IfMatch: meta.ETag}); err != nil {
				t.Fatalf("compare-and-swap with fresh etag: %v", err)
			}
			if _, err := c.Put(ctx, "manifest.age", strings.NewReader("v3"), 2, backend.PutOptions{IfMatch: meta.ETag}); !errors.Is(err, backend.ErrPrecondition) {
				t.Fatalf("stale etag: want ErrPrecondition, got %v", err)
			}

			// A rejected static key is reported once; there is nothing to refresh.
			f.accept()
			before := len(f.requestLog())
			if _, err := c.Head(ctx, "manifest.age"); !errors.Is(err, backend.ErrUnauthorized) {
				t.Fatalf("rejected static key: want ErrUnauthorized, got %v", err)
			}
			if got := len(f.requestLog()) - before; got != 1 {
				t.Fatalf("static key was retried: %d requests, want 1\n%s", got, strings.Join(f.requestLog(), "\n"))
			}
		})
	}
}

func TestExpiredCredentialIsRefreshedBeforeUse(t *testing.T) {
	f := newFakeS3(t, "reinstate")
	f.accept("AKIA1", "AKIA2")
	expired := hourly("AKIA1")
	expired.Expires = time.Now().Add(-time.Second)
	src := &fakeSource{script: []Credentials{expired, hourly("AKIA2")}}
	c := f.client(t, Config{Credentials: src})
	ctx := context.Background()

	if _, err := c.Put(ctx, "a", strings.NewReader("x"), 1, backend.PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Head(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	log := f.requestLog()
	if signedBy(log, "AKIA2") == 0 {
		t.Fatalf("expired credential was never replaced:\n%s", strings.Join(log, "\n"))
	}
	if src.callCount() < 2 {
		t.Fatalf("source asked %d times, want at least 2", src.callCount())
	}
}

func TestSourceErrorSurfacesWithoutRetry(t *testing.T) {
	f := newFakeS3(t, "reinstate")
	src := &fakeSource{err: errors.New("control plane unreachable")}
	c := f.client(t, Config{Credentials: src})
	_, err := c.Head(context.Background(), "a")
	if err == nil || !strings.Contains(err.Error(), "control plane unreachable") {
		t.Fatalf("want source error, got %v", err)
	}
	if n := len(f.requestLog()); n != 0 {
		t.Fatalf("%d requests sent without a credential", n)
	}
}

func TestRejectedCredentialIsRefreshedAndRetried(t *testing.T) {
	for _, code := range []string{"ExpiredToken", "InvalidAccessKeyId", "AccessDenied"} {
		t.Run(code, func(t *testing.T) {
			f := newFakeS3(t, "reinstate")
			f.rejectAs = code
			f.accept("AKIA1")
			src := &fakeSource{script: []Credentials{hourly("AKIA1"), hourly("AKIA2")}}
			c := f.client(t, Config{Credentials: src})
			ctx := context.Background()

			meta, err := c.Put(ctx, "manifest.age", bytes.NewReader([]byte("v1")), 2, backend.PutOptions{IfNoneMatch: true})
			if err != nil {
				t.Fatal(err)
			}
			// Revoke the first key before its stated expiry, as a control plane
			// rotation would; the next call must refresh and retry transparently.
			f.accept("AKIA2")
			meta2, err := c.Put(ctx, "manifest.age", bytes.NewReader([]byte("v2")), 2, backend.PutOptions{IfMatch: meta.ETag})
			if err != nil {
				t.Fatalf("compare-and-swap across a refreshed credential: %v", err)
			}
			rc, got, err := c.Get(ctx, "manifest.age")
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(rc)
			_ = rc.Close()
			if string(body) != "v2" || got.ETag != meta2.ETag {
				t.Fatalf("after retry body=%q etag=%q want v2/%q", body, got.ETag, meta2.ETag)
			}
			// A genuinely stale etag is still a precondition failure, not a
			// credential problem: no extra refresh, no extra request.
			calls := src.callCount()
			before := len(f.requestLog())
			if _, err := c.Put(ctx, "manifest.age", bytes.NewReader([]byte("v3")), 2, backend.PutOptions{IfMatch: meta.ETag}); !errors.Is(err, backend.ErrPrecondition) {
				t.Fatalf("stale etag through refreshed credential: want ErrPrecondition, got %v", err)
			}
			if src.callCount() != calls || len(f.requestLog())-before != 1 {
				t.Fatalf("precondition failure triggered a credential refresh or retry:\n%s", strings.Join(f.requestLog(), "\n"))
			}
			if src.callCount() != 2 {
				t.Fatalf("source asked %d times, want exactly 2", src.callCount())
			}
		})
	}
}

func TestRejectedCredentialIsNotRetriedForever(t *testing.T) {
	f := newFakeS3(t, "reinstate")
	src := &fakeSource{script: []Credentials{hourly("AKIA1"), hourly("AKIA2"), hourly("AKIA3")}}
	c := f.client(t, Config{Credentials: src})
	_, err := c.Head(context.Background(), "a")
	if !errors.Is(err, backend.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	if n := len(f.requestLog()); n != 2 {
		t.Fatalf("%d requests, want exactly one retry (2 total)\n%s", n, strings.Join(f.requestLog(), "\n"))
	}
}

func TestNonSeekableBodyIsNotRetried(t *testing.T) {
	f := newFakeS3(t, "reinstate")
	src := &fakeSource{script: []Credentials{hourly("AKIA1"), hourly("AKIA2")}}
	c := f.client(t, Config{Credentials: src})
	body := io.MultiReader(strings.NewReader("not "), strings.NewReader("seekable"))
	_, err := c.Put(context.Background(), "a", body, 12, backend.PutOptions{})
	if !errors.Is(err, backend.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	if n := len(f.requestLog()); n != 1 {
		t.Fatalf("%d requests, want 1 (a body that cannot be rewound must not be resent)", n)
	}
}

// markerCodec is a cheap stand-in for the age envelope so the push journey
// below exercises the backend, not scrypt.
type markerCodec struct{}

const marker = "fake-age-envelope:"

func (markerCodec) Encrypt(src io.Reader, dst io.Writer, _ string) error {
	if _, err := io.WriteString(dst, marker); err != nil {
		return err
	}
	_, err := io.Copy(dst, src)
	return err
}

func (markerCodec) DecryptReader(src io.Reader, _ string) (io.Reader, error) {
	head := make([]byte, len(marker))
	if _, err := io.ReadFull(src, head); err != nil || string(head) != marker {
		return nil, errors.New("bad envelope")
	}
	return src, nil
}

// TestPushCompletesWhenCredentialExpiresMidPush is the ticket's journey: the
// locker credential stops being accepted between the snapshot upload and the
// manifest compare-and-swap, and the push still lands with a refreshed key.
func TestPushCompletesWhenCredentialExpiresMidPush(t *testing.T) {
	f := newFakeS3(t, "reinstate")
	f.accept("AKIA1")
	src := &fakeSource{script: []Credentials{hourly("AKIA1"), hourly("AKIA2")}}
	c := f.client(t, Config{Credentials: src, Prefix: "profiles/p1"})

	// Seed a manifest so the push performs a real If-Match compare-and-swap.
	ctx := context.Background()
	eng := &reinsync.Engine{Backend: c, Passphrase: "test-pass-phrase-32", Platform: "darwin-arm64", Codec: markerCodec{}}
	seed := filepath.Join(t.TempDir(), "seed.jsonl")
	if err := os.WriteFile(seed, []byte(`{"type":"user","text":"synthetic seed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.PushSession(ctx, reinsync.PushItem{Agent: "claude", SessionID: "seed", ProjectID: "github.com/example/demo", LocalPath: seed}, false); err != nil {
		t.Fatalf("seed push: %v", err)
	}
	seedRequests := len(f.requestLog())

	// Expire AKIA1 right after the snapshot PUT of the second push, so the
	// manifest GET and the conditional manifest PUT run on a refreshed key.
	var expiredAt int
	f.mu.Lock()
	f.hook = func(n int) {
		if expiredAt == 0 && n > seedRequests && strings.HasPrefix(f.requests[n-1], "PUT profiles/p1/snapshots/") {
			expiredAt = n
			f.valid = map[string]bool{"AKIA2": true}
		}
	}
	f.mu.Unlock()

	session := filepath.Join(t.TempDir(), "session-001.jsonl")
	plain := []byte(`{"type":"user","text":"synthetic fixture only"}`)
	if err := os.WriteFile(session, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := eng.PushSession(ctx, reinsync.PushItem{Agent: "claude", SessionID: "session-001", ProjectID: "github.com/example/demo", LocalPath: session}, false)
	if err != nil {
		t.Fatalf("push across credential expiry: %v\n%s", err, strings.Join(f.requestLog(), "\n"))
	}
	if expiredAt == 0 {
		t.Fatalf("test never expired the credential:\n%s", strings.Join(f.requestLog(), "\n"))
	}
	log := f.requestLog()
	if signedBy(log, "AKIA2") == 0 {
		t.Fatalf("no request was signed with the refreshed key:\n%s", strings.Join(log, "\n"))
	}
	if src.callCount() != 2 {
		t.Fatalf("source asked %d times, want 2", src.callCount())
	}

	// The manifest head advanced through the compare-and-swap on the new key,
	// and the snapshot is readable with it too.
	man, err := eng.FetchManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if man.Sessions[reinsync.SessionKey("claude", "session-001")].SnapshotID != id {
		t.Fatalf("manifest did not advance: %+v", man.Sessions)
	}
	if _, ok := man.Sessions[reinsync.SessionKey("claude", "seed")]; !ok {
		t.Fatal("seed entry lost during compare-and-swap")
	}
	_, payload, err := eng.PullSession(ctx, reinsync.PullItem{Agent: "claude", SessionID: "session-001", SnapshotID: id, ProjectID: "github.com/example/demo"}, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, plain) {
		t.Fatalf("payload %q", payload)
	}
}
