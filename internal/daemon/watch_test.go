package daemon

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func collect(t *testing.T, events <-chan Change, wait time.Duration) []string {
	t.Helper()
	var paths []string
	deadline := time.After(wait)
	for {
		select {
		case c, ok := <-events:
			if !ok {
				return paths
			}
			paths = append(paths, c.Path)
		case <-deadline:
			return paths
		}
	}
}

func TestPollWatcherReportsChangedCreatedAndRemovedFiles(t *testing.T) {
	root := t.TempDir()
	session := filepath.Join(root, "projects", "p1", "s1.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tick := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := watchPoll(ctx, []string{root}, time.Hour, tick, log.New(io.Discard, "", 0))
	if w.Mode != "poll" {
		t.Fatal(w.Mode)
	}
	// No change: a tick reports nothing.
	tick <- time.Now()
	if got := collect(t, w.Events, 100*time.Millisecond); len(got) != 0 {
		t.Fatalf("spurious events: %v", got)
	}
	// A write (size changes) and a new file (a lock file is ignored).
	if err := os.WriteFile(session, []byte("a\nb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	created := filepath.Join(root, "projects", "p1", "s2.jsonl")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "p1", "s2.jsonl.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tick <- time.Now()
	got := collect(t, w.Events, 200*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("events=%v, want the write and the creation", got)
	}
	if err := os.Remove(created); err != nil {
		t.Fatal(err)
	}
	tick <- time.Now()
	if got := collect(t, w.Events, 200*time.Millisecond); len(got) != 1 || got[0] != created {
		t.Fatalf("removal events=%v", got)
	}
	w.Stop()
	if _, ok := <-w.Events; ok {
		t.Fatal("events should close after Stop")
	}
}

func TestNotifyWatcherSeesWritesInNewDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := Watch(ctx, []string{root, filepath.Join(root, "missing-root")}, log.New(io.Discard, "", 0))
	if w.Mode != "fsnotify" {
		t.Skipf("fsnotify unavailable here (mode %s)", w.Mode)
	}
	defer w.Stop()
	dir := filepath.Join(root, "projects", "new-project")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Give the watcher a moment to add the new directory before writing
	// into it.
	time.Sleep(200 * time.Millisecond)
	session := filepath.Join(dir, "s1.jsonl")
	if err := os.WriteFile(session, []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := collect(t, w.Events, time.Second)
	found := false
	for _, p := range got {
		if p == session {
			found = true
		}
	}
	if !found {
		t.Fatalf("write in a new directory not seen: %v", got)
	}
}

func TestIgnored(t *testing.T) {
	cases := map[string]bool{
		"/a/s1.jsonl":        false,
		"/a/opencode.db":     false,
		"/a/opencode.db-wal": false,
		"/a/opencode.db-shm": true,
		"/a/.s1.jsonl":       false,
		"/a/.DS_Store":       true,
		"/a/x.lock":          true,
		"/a/x.tmp":           true,
		"/a/x~":              true,
	}
	for path, want := range cases {
		if got := ignored(path); got != want {
			t.Errorf("ignored(%q)=%v, want %v", path, got, want)
		}
	}
}
