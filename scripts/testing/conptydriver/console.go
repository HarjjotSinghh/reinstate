package main

import "time"

// Console is a running child process attached to a pseudo console (or, in
// tests, a fake standing in for one). It is the whole seam run.go depends
// on, so the step-script interpreter is testable without ever allocating a
// real ConPTY.
type Console interface {
	// Write sends bytes to the child's input.
	Write(p []byte) (int, error)
	// Output returns a channel of raw output chunks read from the child.
	// The channel is closed when the child's output side is done (the
	// process exited and its output pipe drained).
	Output() <-chan []byte
	// Resize changes the pseudo console's buffer size.
	Resize(cols, rows int) error
	// Kill terminates the child immediately.
	Kill() error
	// Wait blocks until the child exits and returns its exit code.
	Wait() (int, error)
	// Close releases the console's own resources (safe to call after Wait).
	Close() error
}

// clock is the seam run.go uses for time, so tests never actually sleep.
type clock interface {
	now() time.Time
	sleep(time.Duration)
	after(time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) now() time.Time                         { return time.Now() }
func (realClock) sleep(d time.Duration)                  { time.Sleep(d) }
func (realClock) after(d time.Duration) <-chan time.Time { return time.After(d) }
