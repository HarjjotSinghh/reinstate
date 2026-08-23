package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

// Syncer runs one push or one pull of every session. The production
// implementation is the CLI's own push --all and pull --all; tests use an
// in-memory stand-in.
type Syncer interface {
	// Push pushes every changed local session. ErrConflict (possibly
	// wrapped) reports that a conflict was recorded; the daemon then waits
	// for the next change rather than retrying on a backoff.
	Push(ctx context.Context) (summary string, err error)
	// Pull restores every remote session that is newer than the local copy.
	Pull(ctx context.Context) (summary string, err error)
}

// Account reaches the control plane for the account's devices and the
// pairing requests waiting for approval. It is nil on BYO storage.
type Account interface {
	Pending(ctx context.Context) (pending []PendingApproval, devices []Device, err error)
}

// Notifier shows an OS notification. Failures are logged and ignored.
type Notifier interface {
	Notify(title, body string) error
}

// Clock is the loop's view of time, so tests can drive it.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) Timer
}

// Timer is one armed timer. Stop prevents a timer that has not fired yet
// from firing.
type Timer interface {
	C() <-chan time.Time
	Stop()
}

// Change is one filesystem event from the watcher. Path is informational.
type Change struct {
	Path string
}

// Event is one thing the loop did, for observers (tests, verbose logs).
type Event struct {
	// Kind is "start", "push", "pull", "approvals", "notify", "heartbeat",
	// or "idle" (the loop is blocked waiting for the next trigger).
	Kind string
	Err  error
}

// ErrConflict is what a Syncer returns when push or pull recorded a
// conflict instead of writing. The daemon never resolves conflicts.
var ErrConflict = errors.New("conflict recorded; rein conflicts shows it")

// ErrLocked is what a Syncer returns when another rein process holds the
// mutation lock; the daemon retries after a short backoff.
var ErrLocked = errors.New("another rein process holds the mutation lock")

// Options configure one daemon loop.
type Options struct {
	Home     string
	Syncer   Syncer
	Account  Account // nil on BYO storage
	Notifier Notifier
	Clock    Clock
	// Events delivers session-store changes. A nil channel disables
	// change-driven pushes (the schedule still runs).
	Events <-chan Change
	Logger *log.Logger
	// Observe, when set, is called after every action.
	Observe func(Event)

	// Debounce is the quiet period after the last change before a push;
	// MaxDebounce bounds how long a stream of changes can delay one.
	Debounce    time.Duration
	MaxDebounce time.Duration
	// PullEvery is the pull schedule; ApprovalsEvery the control-plane poll.
	PullEvery      time.Duration
	ApprovalsEvery time.Duration
	// BackoffMin and BackoffMax bound the retry delay after a failure.
	BackoffMin time.Duration
	BackoffMax time.Duration

	// Descriptive fields for the status file.
	Watch   string
	Roots   []string
	Backend string
}

// Defaults are the production intervals.
const (
	DefaultDebounce    = 3 * time.Second
	DefaultMaxDebounce = 30 * time.Second
	// DefaultPullEvery is short enough that a session edited on another
	// device appears here within a minute without any command; pull --all
	// skips snapshots this device already synced, so an idle pull is one
	// manifest read.
	DefaultPullEvery      = 30 * time.Second
	DefaultApprovalsEvery = time.Minute
	DefaultBackoffMin     = 5 * time.Second
	DefaultBackoffMax     = 10 * time.Minute
)

func (o *Options) fill() {
	if o.Clock == nil {
		o.Clock = realClock{}
	}
	if o.Logger == nil {
		o.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if o.Debounce <= 0 {
		o.Debounce = DefaultDebounce
	}
	if o.MaxDebounce <= 0 {
		o.MaxDebounce = DefaultMaxDebounce
	}
	if o.PullEvery <= 0 {
		o.PullEvery = DefaultPullEvery
	}
	if o.ApprovalsEvery <= 0 {
		o.ApprovalsEvery = DefaultApprovalsEvery
	}
	if o.BackoffMin <= 0 {
		o.BackoffMin = DefaultBackoffMin
	}
	if o.BackoffMax <= 0 {
		o.BackoffMax = DefaultBackoffMax
	}
	if o.Backend == "" {
		o.Backend = "byo"
	}
	if o.Watch == "" {
		o.Watch = "poll"
	}
}

type realClock struct{}

func (realClock) Now() time.Time                 { return time.Now() }
func (realClock) NewTimer(d time.Duration) Timer { return realTimer{time.NewTimer(d)} }

type realTimer struct{ t *time.Timer }

func (t realTimer) C() <-chan time.Time { return t.t.C }
func (t realTimer) Stop()               { t.t.Stop() }

// timerC is the channel of a possibly nil timer; a nil timer never fires.
func timerC(t Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C()
}

func stopTimer(t Timer) {
	if t != nil {
		t.Stop()
	}
}

// loop is the running state of one daemon.
type loop struct {
	opts   Options
	status Status

	dirty      bool
	firstDirty time.Time
	// notBefore is the earliest the next push attempt may run after a
	// failure; changes arriving during the backoff wait for it instead of
	// pushing immediately.
	notBefore time.Time
	pushFails int
	pullFails int
	seen      map[string]bool // pairing request ids already notified
}

// Run drives the loop until ctx ends. It pulls once and polls approvals
// once at start, then treats any session change since the last run as
// pending (a push follows after the debounce), so nothing written while
// the daemon was stopped is missed.
func Run(ctx context.Context, opts Options) error {
	opts.fill()
	if opts.Syncer == nil {
		return errors.New("daemon: Syncer is required")
	}
	now := opts.Clock.Now()
	l := &loop{
		opts: opts,
		status: Status{
			Version: StatusVersion, PID: os.Getpid(), StartedAt: now, UpdatedAt: now,
			Watch: opts.Watch, Roots: opts.Roots, Backend: opts.Backend,
			Pending: []PendingApproval{},
		},
		seen: map[string]bool{},
	}
	l.writeStatus()
	l.observe(Event{Kind: "start"})
	l.opts.Logger.Printf("daemon started pid=%d backend=%s watch=%s roots=%d", os.Getpid(), opts.Backend, opts.Watch, len(opts.Roots))

	// First pass: pull, then poll approvals, then schedule a push for
	// whatever changed while the daemon was away.
	pullT := l.pull(ctx)
	approvalsT := l.approvals(ctx)
	l.markDirty(now)
	debounceT := opts.Clock.NewTimer(opts.Debounce)
	heartbeatT := opts.Clock.NewTimer(HeartbeatEvery)
	defer func() {
		for _, t := range []Timer{pullT, approvalsT, debounceT, heartbeatT} {
			stopTimer(t)
		}
	}()

	for {
		l.observe(Event{Kind: "idle"})
		select {
		case <-ctx.Done():
			l.opts.Logger.Printf("daemon stopping: %v", ctx.Err())
			l.status.UpdatedAt = l.opts.Clock.Now()
			l.status.PID = 0
			l.writeStatus()
			return nil
		case change, ok := <-opts.Events:
			if !ok {
				// The watcher closed; keep the schedule alive without events.
				opts.Events = nil
				continue
			}
			now := opts.Clock.Now()
			l.markDirty(now)
			stopTimer(debounceT)
			debounceT = opts.Clock.NewTimer(l.pushDelay(now))
			l.opts.Logger.Printf("change: %s", change.Path)
		case <-timerC(debounceT):
			debounceT = nil
			if !l.dirty {
				continue
			}
			if retry := l.push(ctx); retry > 0 {
				debounceT = opts.Clock.NewTimer(retry)
			}
		case <-timerC(pullT):
			pullT = l.pull(ctx)
		case <-timerC(approvalsT):
			approvalsT = l.approvals(ctx)
		case <-timerC(heartbeatT):
			heartbeatT = opts.Clock.NewTimer(HeartbeatEvery)
			l.writeStatus()
			l.observe(Event{Kind: "heartbeat"})
		}
	}
}

func (l *loop) markDirty(now time.Time) {
	if !l.dirty {
		l.firstDirty = now
	}
	l.dirty = true
}

// pushDelay is how long a change that just arrived waits before the push:
// the debounce, or nothing once changes have kept the store dirty for
// MaxDebounce (a session that never stops changing still gets pushed),
// but never earlier than the backoff after a failed push allows.
func (l *loop) pushDelay(now time.Time) time.Duration {
	delay := l.opts.Debounce
	if now.Sub(l.firstDirty) >= l.opts.MaxDebounce {
		delay = 0
	}
	if wait := l.notBefore.Sub(now); wait > delay {
		delay = wait
	}
	return delay
}

// push runs one push. It returns the delay before the next attempt, or 0
// when nothing should be retried until the next change.
func (l *loop) push(ctx context.Context) time.Duration {
	// Every attempt restarts the max-debounce window, so changes that
	// arrive while a push is failing do not bypass the backoff.
	l.firstDirty = l.opts.Clock.Now()
	summary, err := l.guard(ctx, l.opts.Syncer.Push)
	now := l.opts.Clock.Now()
	l.status.Push.At = now
	switch {
	case err == nil:
		l.dirty = false
		l.pushFails = 0
		l.notBefore = time.Time{}
		l.status.Push = Outcome{At: now, OK: true, Summary: summary, LastOK: now}
		l.opts.Logger.Printf("push: %s", summary)
		l.writeStatus()
		l.observe(Event{Kind: "push"})
		return 0
	case errors.Is(err, ErrConflict):
		// The conflict is recorded locally; a retry would hit it again.
		l.dirty = false
		l.pushFails = 0
		l.notBefore = time.Time{}
		l.status.Push = Outcome{At: now, OK: false, Error: err.Error(), Conflict: true, LastOK: l.status.Push.LastOK}
		l.opts.Logger.Printf("push: %v", err)
		l.writeStatus()
		l.observe(Event{Kind: "push", Err: err})
		return 0
	default:
		l.pushFails++
		delay := l.backoff(l.pushFails)
		if errors.Is(err, ErrLocked) {
			delay = l.opts.BackoffMin
		}
		l.notBefore = now.Add(delay)
		l.status.Push = Outcome{At: now, OK: false, Error: err.Error(), LastOK: l.status.Push.LastOK}
		l.opts.Logger.Printf("push failed (retry in %s): %v", delay, err)
		l.writeStatus()
		l.observe(Event{Kind: "push", Err: err})
		return delay
	}
}

// pull runs one pull and returns the timer for the next one.
func (l *loop) pull(ctx context.Context) Timer {
	summary, err := l.guard(ctx, l.opts.Syncer.Pull)
	now := l.opts.Clock.Now()
	next := l.opts.PullEvery
	switch {
	case err == nil:
		l.pullFails = 0
		l.status.Pull = Outcome{At: now, OK: true, Summary: summary, LastOK: now}
		l.opts.Logger.Printf("pull: %s", summary)
	case errors.Is(err, ErrConflict):
		l.pullFails = 0
		l.status.Pull = Outcome{At: now, OK: false, Error: err.Error(), Conflict: true, LastOK: l.status.Pull.LastOK}
		l.opts.Logger.Printf("pull: %v", err)
	default:
		l.pullFails++
		next = l.backoff(l.pullFails)
		if errors.Is(err, ErrLocked) {
			next = l.opts.BackoffMin
		}
		l.status.Pull = Outcome{At: now, OK: false, Error: err.Error(), LastOK: l.status.Pull.LastOK}
		l.opts.Logger.Printf("pull failed (retry in %s): %v", next, err)
	}
	l.writeStatus()
	l.observe(Event{Kind: "pull", Err: err})
	return l.opts.Clock.NewTimer(next)
}

// approvals refreshes the pending pairing requests and notifies once per
// new request. It returns the timer for the next poll, or nil on BYO.
func (l *loop) approvals(ctx context.Context) Timer {
	if l.opts.Account == nil {
		return nil
	}
	pending, devices, err := l.guardAccount(ctx)
	now := l.opts.Clock.Now()
	l.status.ApprovalsAt = now
	if err != nil {
		l.status.ApprovalsError = err.Error()
		l.opts.Logger.Printf("approvals: %v", err)
		l.writeStatus()
		l.observe(Event{Kind: "approvals", Err: err})
		return l.opts.Clock.NewTimer(l.opts.ApprovalsEvery)
	}
	l.status.ApprovalsError = ""
	if pending == nil {
		pending = []PendingApproval{}
	}
	l.status.Pending = pending
	l.status.Devices = devices
	current := map[string]bool{}
	for _, p := range pending {
		current[p.RequestID] = true
		if l.seen[p.RequestID] {
			continue
		}
		l.seen[p.RequestID] = true
		name := p.DeviceName
		if name == "" {
			name = p.DeviceID
		}
		l.opts.Logger.Printf("approval pending: device %q (%s) wants to join; run rein devices approve", name, p.RequestID)
		if l.opts.Notifier != nil {
			if err := l.opts.Notifier.Notify("Reinstate: device wants to join",
				fmt.Sprintf("%s wants to join your account. Run: rein devices approve", name)); err != nil {
				l.opts.Logger.Printf("notify: %v", err)
			}
			l.observe(Event{Kind: "notify"})
		}
	}
	// Forget requests that are gone so a fresh request from the same
	// device notifies again.
	for id := range l.seen {
		if !current[id] {
			delete(l.seen, id)
		}
	}
	l.writeStatus()
	l.observe(Event{Kind: "approvals"})
	return l.opts.Clock.NewTimer(l.opts.ApprovalsEvery)
}

func (l *loop) backoff(fails int) time.Duration {
	delay := l.opts.BackoffMin
	for i := 1; i < fails && delay < l.opts.BackoffMax; i++ {
		delay *= 2
	}
	if delay > l.opts.BackoffMax {
		delay = l.opts.BackoffMax
	}
	return delay
}

// guard runs a sync step and turns a panic (a vendor store changing under
// an adapter mid-write, for instance) into an error so the daemon lives on.
func (l *loop) guard(ctx context.Context, step func(context.Context) (string, error)) (summary string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return step(ctx)
}

func (l *loop) guardAccount(ctx context.Context) (pending []PendingApproval, devices []Device, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return l.opts.Account.Pending(ctx)
}

func (l *loop) writeStatus() {
	l.status.UpdatedAt = l.opts.Clock.Now()
	if l.opts.Home == "" {
		return
	}
	if err := l.status.Write(l.opts.Home); err != nil {
		l.opts.Logger.Printf("status: %v", err)
	}
}

func (l *loop) observe(e Event) {
	if l.opts.Observe != nil {
		l.opts.Observe(e)
	}
}
