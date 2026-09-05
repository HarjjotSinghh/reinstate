package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeConsole is an in-memory Console for testing Runner without a real
// pseudo console. Feed pushes bytes as if the child produced them.
type fakeConsole struct {
	mu       sync.Mutex
	written  bytes.Buffer
	out      chan []byte
	killed   bool
	exitCode int
}

func newFakeConsole() *fakeConsole {
	return &fakeConsole{out: make(chan []byte, 64)}
}

func (f *fakeConsole) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.written.Write(p)
}
func (f *fakeConsole) Output() <-chan []byte { return f.out }
func (f *fakeConsole) Resize(int, int) error { return nil }
func (f *fakeConsole) Kill() error {
	f.mu.Lock()
	f.killed = true
	f.mu.Unlock()
	return nil
}
func (f *fakeConsole) Wait() (int, error) { return f.exitCode, nil }
func (f *fakeConsole) Close() error       { return nil }

// Feed pushes a chunk as if the child emitted it, and blocks until Pump has
// rendered it, so tests can assert on the resulting frame deterministically
// without a sleep. It waits for the chunk's own text to appear, so it is
// only right for a chunk with no escape sequences the VT model would strip;
// FeedAndWaitFor covers the rest.
func (f *fakeConsole) Feed(r *Runner, chunk []byte) {
	f.FeedAndWaitFor(r, chunk, string(chunk))
}

func (f *fakeConsole) FeedAndWaitFor(r *Runner, chunk []byte, renderSubstring string) {
	f.out <- chunk
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if renderSubstring == "" || strings.Contains(r.Render(), renderSubstring) {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *fakeConsole) closeOutput() { close(f.out) }

func (f *fakeConsole) writtenString() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.written.String()
}

// fakeClockT is a clock a test drives explicitly: sleep and after both
// return immediately, so a wait loop's timeout is governed only by how many
// times the test lets it poll, never by wall time.
type fakeClockT struct {
	mu  sync.Mutex
	t   time.Time
	adv time.Duration // how far now() advances every call, simulating time passing across polls
}

func (c *fakeClockT) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur := c.t
	c.t = c.t.Add(c.adv)
	return cur
}
func (c *fakeClockT) sleep(time.Duration) {}
func (c *fakeClockT) after(time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func TestRunnerSendAndKey(t *testing.T) {
	fc := newFakeConsole()
	r := NewRunner(fc, NewScreen(80, 25), nil)
	go r.Pump()
	steps := []Step{
		{Kind: StepSend, Text: "hello"},
		{Kind: StepKey, Key: "enter"},
	}
	if err := r.Run(steps); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := fc.writtenString(); got != "hello\r" {
		t.Fatalf("written = %q, want %q", got, "hello\r")
	}
	fc.closeOutput()
}

func TestRunnerWaitMatches(t *testing.T) {
	fc := newFakeConsole()
	screen := NewScreen(80, 25)
	r := NewRunner(fc, screen, nil)
	go r.Pump()
	defer fc.closeOutput()

	go func() {
		time.Sleep(20 * time.Millisecond)
		fc.out <- []byte("almost")
		time.Sleep(20 * time.Millisecond)
		fc.out <- []byte(" ready")
	}()

	err := r.Run([]Step{{Kind: StepWait, Line: 1, Pattern: mustRegex("almost ready"), Timeout: 2 * time.Second}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunnerWaitTimesOut(t *testing.T) {
	fc := newFakeConsole()
	r := NewRunner(fc, NewScreen(80, 25), nil)
	r.clk = &fakeClockT{t: time.Unix(0, 0), adv: time.Second} // each now() call jumps a second, so the deadline passes almost immediately
	go r.Pump()
	defer fc.closeOutput()

	err := r.Run([]Step{{Kind: StepWait, Line: 5, Pattern: mustRegex("never happens"), Timeout: 1 * time.Millisecond}})
	if err == nil {
		t.Fatal("Run: want a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "line 5") {
		t.Fatalf("error %q does not name the source line", err)
	}
}

func TestRunnerSnapshot(t *testing.T) {
	fc := newFakeConsole()
	r := NewRunner(fc, NewScreen(80, 25), nil)
	go r.Pump()
	fc.Feed(r, []byte("frame content"))

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "frame.txt")
	if err := r.Run([]Step{{Kind: StepSnapshot, Path: path}}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	fc.closeOutput()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(got), "frame content") {
		t.Fatalf("snapshot file = %q, want it to contain %q", got, "frame content")
	}
}

func TestRunnerKill(t *testing.T) {
	fc := newFakeConsole()
	r := NewRunner(fc, NewScreen(80, 25), nil)
	go r.Pump()
	if err := r.Run([]Step{{Kind: StepKill}}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	fc.mu.Lock()
	killed := fc.killed
	fc.mu.Unlock()
	if !killed {
		t.Fatal("Kill was not called")
	}
	fc.closeOutput()
}

func TestRunnerRawWriter(t *testing.T) {
	fc := newFakeConsole()
	var raw bytes.Buffer
	r := NewRunner(fc, NewScreen(80, 25), &raw)
	go r.Pump()
	fc.FeedAndWaitFor(r, []byte("\x1b[31mcolored\x1b[0m"), "colored")
	fc.closeOutput()
	if raw.Len() == 0 || !strings.Contains(raw.String(), "\x1b[31m") {
		t.Fatalf("raw writer did not receive the unmodified bytes, got %q", raw.String())
	}
}

func mustRegex(pattern string) *regexp.Regexp { return regexp.MustCompile(pattern) }
