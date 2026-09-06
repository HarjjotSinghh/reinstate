// Package daemontest holds deterministic stand-ins for the daemon loop's
// dependencies, shared by the daemon's own tests and the CLI journeys.
package daemontest

import (
	"sync"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/daemon"
)

// FakeClock hands out timers the test fires by advancing time.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*fakeTimer
}

type fakeTimer struct {
	clock   *FakeClock
	at      time.Time
	ch      chan time.Time
	stopped bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.ch }

func (t *fakeTimer) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.stopped = true
}

// NewFakeClock starts at a fixed instant.
func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)}
}

// Now implements daemon.Clock.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// NewTimer implements daemon.Clock.
func (c *FakeClock) NewTimer(d time.Duration) daemon.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{clock: c, at: c.now.Add(d), ch: make(chan time.Time, 1)}
	c.waiters = append(c.waiters, t)
	return t
}

// Advance moves time forward and fires every live timer that came due,
// returning how many fired. Each fired timer is one loop iteration.
func (c *FakeClock) Advance(d time.Duration) int {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due, rest []*fakeTimer
	for _, t := range c.waiters {
		switch {
		case t.stopped:
		case !t.at.After(c.now):
			due = append(due, t)
		default:
			rest = append(rest, t)
		}
	}
	c.waiters = rest
	now := c.now
	c.mu.Unlock()
	for _, t := range due {
		t.ch <- now
	}
	return len(due)
}
