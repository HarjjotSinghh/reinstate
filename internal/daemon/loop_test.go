package daemon_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/daemon"
	"github.com/HarjjotSinghh/reinstate/internal/daemon/daemontest"
)

// fakeSyncer counts pushes and pulls and answers with scripted errors.
type fakeSyncer struct {
	mu       sync.Mutex
	pushes   int
	pulls    int
	pushErrs []error // consumed in order; nil entries succeed
	pullErrs []error
	panics   bool
}

func (s *fakeSyncer) Push(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pushes++
	if s.panics {
		s.panics = false
		panic("vendor store changed under the adapter")
	}
	if len(s.pushErrs) > 0 {
		err := s.pushErrs[0]
		s.pushErrs = s.pushErrs[1:]
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("pushed (%d)", s.pushes), nil
}

func (s *fakeSyncer) Pull(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pulls++
	if len(s.pullErrs) > 0 {
		err := s.pullErrs[0]
		s.pullErrs = s.pullErrs[1:]
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("pulled (%d)", s.pulls), nil
}

func (s *fakeSyncer) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pushes, s.pulls
}

type fakeAccount struct {
	mu      sync.Mutex
	pending []daemon.PendingApproval
	devices []daemon.Device
	err     error
	polls   int
}

func (a *fakeAccount) Pending(context.Context) ([]daemon.PendingApproval, []daemon.Device, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.polls++
	if a.err != nil {
		return nil, nil, a.err
	}
	return append([]daemon.PendingApproval{}, a.pending...), append([]daemon.Device{}, a.devices...), nil
}

func (a *fakeAccount) set(p []daemon.PendingApproval) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pending = p
}

type fakeNotifier struct {
	mu    sync.Mutex
	shown []string
}

func (n *fakeNotifier) Notify(title, body string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.shown = append(n.shown, title+": "+body)
	return nil
}

func (n *fakeNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.shown)
}

// harness runs one loop under test and lets the test wait for events.
type harness struct {
	t       *testing.T
	clock   *daemontest.FakeClock
	syncer  *fakeSyncer
	account *fakeAccount
	notify  *fakeNotifier
	events  chan daemon.Change
	seen    chan daemon.Event
	home    string
	cancel  context.CancelFunc
	done    chan error
}

func newHarness(t *testing.T, account *fakeAccount, syncer *fakeSyncer) *harness {
	t.Helper()
	h := &harness{
		t: t, clock: daemontest.NewFakeClock(), syncer: syncer, account: account, notify: &fakeNotifier{},
		events: make(chan daemon.Change, 16), seen: make(chan daemon.Event, 1024), home: t.TempDir(), done: make(chan error, 1),
	}
	if h.syncer == nil {
		h.syncer = &fakeSyncer{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	opts := daemon.Options{
		Home: h.home, Syncer: h.syncer, Notifier: h.notify, Clock: h.clock, Events: h.events,
		Logger:   log.New(io.Discard, "", 0),
		Observe:  func(e daemon.Event) { h.seen <- e },
		Debounce: 3 * time.Second, MaxDebounce: 30 * time.Second,
		PullEvery: 5 * time.Minute, ApprovalsEvery: time.Minute,
		BackoffMin: 5 * time.Second, BackoffMax: 10 * time.Minute,
		Watch: "fake", Backend: "byo",
	}
	if account != nil {
		opts.Account = account
		opts.Backend = "hop"
	}
	go func() { h.done <- daemon.Run(ctx, opts) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-h.done:
		case <-time.After(5 * time.Second):
			t.Error("loop did not stop")
		}
	})
	// Keep the observer from blocking the loop once the test stops reading.
	_ = h.seen
	return h
}

// until collects events until n "idle" events have passed, so the loop
// has handled n triggers and re-armed its timers before the test goes on.
func (h *harness) until(n int) []daemon.Event {
	h.t.Helper()
	var events []daemon.Event
	deadline := time.After(5 * time.Second)
	for n > 0 {
		select {
		case e := <-h.seen:
			events = append(events, e)
			if e.Kind == "idle" {
				n--
			}
		case <-deadline:
			h.t.Fatalf("loop did not settle; events so far: %v", kinds(events))
		}
	}
	return events
}

// start waits for the start-up pass (pull, approvals) to finish.
func (h *harness) start() []daemon.Event { h.t.Helper(); return h.until(1) }

// advance moves the clock and waits for every timer it fired to be handled.
func (h *harness) advance(d time.Duration) []daemon.Event {
	h.t.Helper()
	return h.until(h.clock.Advance(d))
}

// change delivers one file event and waits for the loop to take it.
func (h *harness) change(path string) []daemon.Event {
	h.t.Helper()
	h.events <- daemon.Change{Path: path}
	return h.until(1)
}

// stop cancels the loop and waits for it to return.
func (h *harness) stop() {
	h.t.Helper()
	h.cancel()
	select {
	case err := <-h.done:
		if err != nil {
			h.t.Fatal(err)
		}
		h.done <- nil
	case <-time.After(5 * time.Second):
		h.t.Fatal("loop did not stop")
	}
}

func kinds(events []daemon.Event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}
	return out
}

// find returns the first event of kind, or fails.
func (h *harness) find(events []daemon.Event, kind string) daemon.Event {
	h.t.Helper()
	for _, e := range events {
		if e.Kind == kind {
			return e
		}
	}
	h.t.Fatalf("no %q among %v", kind, kinds(events))
	return daemon.Event{}
}

func has(events []daemon.Event, kind string) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func (h *harness) status() daemon.Status {
	h.t.Helper()
	s, err := daemon.ReadStatus(h.home)
	if err != nil {
		h.t.Fatal(err)
	}
	return s
}

func TestLoopStartsWithPullThenPushesAfterDebounce(t *testing.T) {
	h := newHarness(t, nil, nil)
	h.find(h.start(), "pull")
	if pushes, pulls := h.syncer.counts(); pushes != 0 || pulls != 1 {
		t.Fatalf("after start: pushes=%d pulls=%d, want 0/1", pushes, pulls)
	}
	// The start marks the store dirty; the push follows the debounce.
	if evs := h.advance(2 * time.Second); has(evs, "push") {
		t.Fatal("pushed before the debounce elapsed")
	}
	h.find(h.advance(time.Second), "push")
	if pushes, _ := h.syncer.counts(); pushes != 1 {
		t.Fatalf("pushes=%d, want 1", pushes)
	}
	s := h.status()
	if !s.Push.OK || !s.Pull.OK || s.Push.Summary != "pushed (1)" || s.Pull.Summary != "pulled (1)" || s.Backend != "byo" || s.Watch != "fake" {
		t.Fatalf("status: %+v", s)
	}
	if s.PID == 0 || !s.Alive(h.clock.Now()) {
		t.Fatalf("status should describe a live daemon: %+v", s)
	}
}

func TestLoopCoalescesChangesWithinDebounce(t *testing.T) {
	h := newHarness(t, nil, nil)
	h.start()
	h.find(h.advance(3*time.Second), "push") // the start-up push

	// Five changes one second apart: each one re-arms the debounce, so a
	// single push happens three seconds after the last.
	for i := 0; i < 5; i++ {
		h.change(fmt.Sprintf("/store/session-%d.jsonl", i))
		if evs := h.advance(time.Second); has(evs, "push") {
			t.Fatalf("pushed during the change burst at change %d", i)
		}
	}
	if pushes, _ := h.syncer.counts(); pushes != 1 {
		t.Fatalf("pushed during the change burst: pushes=%d", pushes)
	}
	h.find(h.advance(2*time.Second), "push")
	if pushes, _ := h.syncer.counts(); pushes != 2 {
		t.Fatalf("pushes=%d, want exactly one more", pushes)
	}
	// Quiet afterwards: no further push.
	if evs := h.advance(time.Minute); has(evs, "push") {
		t.Fatal("pushed with nothing dirty")
	}
}

func TestLoopPushesAStreamThatNeverGoesQuiet(t *testing.T) {
	h := newHarness(t, nil, nil)
	h.start()
	h.find(h.advance(3*time.Second), "push")
	// A change every two seconds: the debounce alone would never fire; the
	// 30-second cap does, on the first change at or past the cap.
	pushedAt := -1
	for i := 0; i < 20 && pushedAt < 0; i++ {
		h.change("/store/busy.jsonl")
		if has(h.advance(2*time.Second), "push") {
			pushedAt = i
		}
	}
	if pushedAt != 15 { // change 0 at t=0, change 15 at t=30s
		t.Fatalf("push happened at change %d, want 15", pushedAt)
	}
	if pushes, _ := h.syncer.counts(); pushes != 2 {
		t.Fatalf("pushes=%d, want 2", pushes)
	}
}

func TestLoopPullSchedule(t *testing.T) {
	h := newHarness(t, nil, nil)
	h.start()
	h.find(h.advance(3*time.Second), "push")
	if evs := h.advance(4*time.Minute + 56*time.Second); has(evs, "pull") {
		t.Fatal("pulled early")
	}
	h.find(h.advance(time.Second), "pull")
	if _, pulls := h.syncer.counts(); pulls != 2 {
		t.Fatalf("pulls=%d, want 2", pulls)
	}
	h.find(h.advance(5*time.Minute), "pull")
	if _, pulls := h.syncer.counts(); pulls != 3 {
		t.Fatalf("pulls=%d, want 3", pulls)
	}
}

func TestLoopBacksOffAfterFailuresAndRecovers(t *testing.T) {
	boom := errors.New("locker unreachable")
	syncer := &fakeSyncer{pushErrs: []error{boom, boom, nil}}
	h := newHarness(t, nil, syncer)
	h.start()
	if e := h.find(h.advance(3*time.Second), "push"); e.Err == nil {
		t.Fatal("first push should fail")
	}
	s := h.status()
	if s.Push.OK || s.Push.Error != boom.Error() || !s.Push.LastOK.IsZero() {
		t.Fatalf("status after failure: %+v", s.Push)
	}
	// Retry after BackoffMin (5s), then after 10s.
	if has(h.advance(4*time.Second), "push") {
		t.Fatal("retried too early")
	}
	if e := h.find(h.advance(time.Second), "push"); e.Err == nil {
		t.Fatal("second push should fail")
	}
	if has(h.advance(9*time.Second), "push") {
		t.Fatal("second retry too early")
	}
	if e := h.find(h.advance(time.Second), "push"); e.Err != nil {
		t.Fatalf("third push should succeed: %v", e.Err)
	}
	if s := h.status(); !s.Push.OK || s.Push.Error != "" {
		t.Fatalf("status after recovery: %+v", s.Push)
	}
}

func TestLoopBackoffHoldsWhileSessionStaysActiveDuringOutage(t *testing.T) {
	boom := errors.New("locker unreachable")
	errs := make([]error, 200)
	for i := range errs {
		errs[i] = boom
	}
	syncer := &fakeSyncer{pushErrs: errs}
	h := newHarness(t, nil, syncer)
	h.start()
	h.find(h.advance(3*time.Second), "push") // fails; retry in 5s
	// One write per second for 90 seconds while the locker is down. A
	// change arriving during the backoff waits for it, and every attempt
	// restarts the 30-second max-debounce window, so the attempts land at
	// t=3s, 33s, and 63s: three in total, not one per change.
	for i := 0; i < 90; i++ {
		h.change("/store/busy.jsonl")
		h.advance(time.Second)
	}
	if pushes, _ := h.syncer.counts(); pushes != 3 {
		t.Fatalf("pushes=%d in 90s of outage, want 3", pushes)
	}
	s := h.status()
	if s.Push.OK || s.Push.Error != boom.Error() {
		t.Fatalf("status during outage: %+v", s.Push)
	}
}

func TestLoopConflictStopsRetryUntilNextChange(t *testing.T) {
	syncer := &fakeSyncer{pushErrs: []error{fmt.Errorf("push: %w", daemon.ErrConflict), nil}}
	h := newHarness(t, nil, syncer)
	h.start()
	h.find(h.advance(3*time.Second), "push")
	if s := h.status(); !s.Push.Conflict || s.Push.OK {
		t.Fatalf("conflict not recorded: %+v", s.Push)
	}
	if has(h.advance(time.Hour), "push") {
		t.Fatal("retried a conflict")
	}
	h.change("/store/resolved.jsonl")
	h.find(h.advance(3*time.Second), "push")
	if s := h.status(); !s.Push.OK || s.Push.Conflict {
		t.Fatalf("push after the next change: %+v", s.Push)
	}
}

func TestLoopSurvivesAPanickingSyncer(t *testing.T) {
	syncer := &fakeSyncer{panics: true}
	h := newHarness(t, nil, syncer)
	h.start()
	if e := h.find(h.advance(3*time.Second), "push"); e.Err == nil || e.Err.Error() != "recovered: vendor store changed under the adapter" {
		t.Fatalf("panic should surface as an error: %v", e.Err)
	}
	if e := h.find(h.advance(5*time.Second), "push"); e.Err != nil {
		t.Fatalf("retry after the panic: %v", e.Err)
	}
}

func TestLoopSurfacesPendingApprovalsOnce(t *testing.T) {
	account := &fakeAccount{devices: []daemon.Device{{ID: "dev-a", Name: "macbook", This: true}}}
	h := newHarness(t, account, nil)
	h.find(h.start(), "approvals")
	if s := h.status(); s.Backend != "hop" || len(s.Pending) != 0 || len(s.Devices) != 1 {
		t.Fatalf("status after first poll: %+v", s)
	}
	expires := h.clock.Now().Add(10 * time.Minute)
	account.set([]daemon.PendingApproval{{RequestID: "pair-1", DeviceID: "dev-b", DeviceName: "desktop", Platform: "windows", ExpiresAt: expires}})
	h.find(h.advance(3*time.Second), "push") // start-up push; not the approvals poll
	evs := h.advance(57 * time.Second)
	h.find(evs, "notify")
	h.find(evs, "approvals")
	if n := h.notify.count(); n != 1 {
		t.Fatalf("notifications=%d, want 1", n)
	}
	if h.notify.shown[0] != "Reinstate: device wants to join: desktop wants to join your account. Run: rein devices approve" {
		t.Fatalf("notification text: %q", h.notify.shown[0])
	}
	s := h.status()
	if len(s.Pending) != 1 || s.Pending[0].DeviceName != "desktop" || s.Pending[0].RequestID != "pair-1" {
		t.Fatalf("pending in status: %+v", s.Pending)
	}
	// The same request on the next poll notifies nobody again.
	h.find(h.advance(time.Minute), "approvals")
	if n := h.notify.count(); n != 1 {
		t.Fatalf("re-notified the same request: %d", n)
	}
	// Approved elsewhere: the list empties; a later fresh request from the
	// same device notifies again.
	account.set(nil)
	h.find(h.advance(time.Minute), "approvals")
	if s := h.status(); len(s.Pending) != 0 {
		t.Fatalf("pending should be empty: %+v", s.Pending)
	}
	account.set([]daemon.PendingApproval{{RequestID: "pair-2", DeviceID: "dev-b", DeviceName: "desktop", ExpiresAt: expires.Add(time.Hour)}})
	h.find(h.advance(time.Minute), "notify")
	if n := h.notify.count(); n != 2 {
		t.Fatalf("notifications=%d, want 2", n)
	}
}

func TestLoopApprovalsErrorIsRecordedNotFatal(t *testing.T) {
	account := &fakeAccount{err: errors.New("control plane: 503")}
	h := newHarness(t, account, nil)
	if e := h.find(h.start(), "approvals"); e.Err == nil {
		t.Fatal("expected the control-plane error")
	}
	if s := h.status(); s.ApprovalsError == "" {
		t.Fatalf("approvals error missing from status: %+v", s)
	}
	h.find(h.advance(3*time.Second), "push") // the rest of the loop is unaffected
}

func TestLoopWritesStoppedStatusOnCancel(t *testing.T) {
	h := newHarness(t, nil, nil)
	h.start()
	h.stop()
	if s := h.status(); s.PID != 0 || s.Alive(h.clock.Now()) {
		t.Fatalf("stopped daemon still looks alive: %+v", s)
	}
}

func TestStatusAliveRequiresFreshHeartbeat(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		s    daemon.Status
		want bool
	}{
		{"fresh own pid", daemon.Status{PID: ownPID(), UpdatedAt: now.Add(-time.Minute)}, true},
		{"stale", daemon.Status{PID: ownPID(), UpdatedAt: now.Add(-daemon.StaleAfter - time.Second)}, false},
		{"no pid", daemon.Status{UpdatedAt: now}, false},
		{"dead pid", daemon.Status{PID: 2147483000, UpdatedAt: now}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.Alive(now); got != c.want {
				t.Fatalf("Alive=%v, want %v", got, c.want)
			}
		})
	}
}

func TestReadStatusMissing(t *testing.T) {
	if _, err := daemon.ReadStatus(t.TempDir()); !errors.Is(err, daemon.ErrNoStatus) {
		t.Fatalf("err=%v, want daemon.ErrNoStatus", err)
	}
}

func ownPID() int { return os.Getpid() }
