package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Runner pumps a Console's raw output through a Screen and executes a step
// script against the result. It depends only on the Console and clock
// interfaces, so it is exercised in tests against a fake console and never
// touches a real pseudo console.
type Runner struct {
	console Console
	screen  *Screen
	clk     clock
	raw     io.Writer // optional: every output chunk, unmodified, for -raw

	mu   sync.Mutex
	done bool
}

// NewRunner builds a Runner. raw may be nil.
func NewRunner(c Console, screen *Screen, raw io.Writer) *Runner {
	return &Runner{console: c, screen: screen, clk: realClock{}, raw: raw}
}

// Pump reads the console's output until it closes, feeding the VT screen
// model and writing back any reply bytes a device query produced (OSC 11,
// CSI 6n). Call it in its own goroutine before running a script; it returns
// when the console's output channel closes.
func (r *Runner) Pump() {
	for chunk := range r.console.Output() {
		if r.raw != nil {
			_, _ = r.raw.Write(chunk)
		}
		r.mu.Lock()
		reply := r.screen.Feed(chunk)
		r.mu.Unlock()
		if len(reply) > 0 {
			_, _ = r.console.Write(reply)
		}
	}
	r.mu.Lock()
	r.done = true
	r.mu.Unlock()
}

// Render returns the current frame, safe to call while Pump is running
// concurrently in another goroutine.
func (r *Runner) Render() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.screen.Render()
}

// Run executes steps in order and returns the first error (a wait timeout,
// a malformed key name, or a write failure), naming the source line.
func (r *Runner) Run(steps []Step) error {
	for _, st := range steps {
		if err := r.runOne(st); err != nil {
			return fmt.Errorf("line %d: %w", st.Line, err)
		}
	}
	return nil
}

func (r *Runner) runOne(st Step) error {
	switch st.Kind {
	case StepWait:
		return r.wait(st)
	case StepSend:
		_, err := r.console.Write([]byte(st.Text))
		return err
	case StepKey:
		b, err := KeyBytes(st.Key)
		if err != nil {
			return err
		}
		_, err = r.console.Write(b)
		return err
	case StepSnapshot:
		return r.snapshot(st.Path)
	case StepSleep:
		r.clk.sleep(st.Timeout)
		return nil
	case StepKill:
		return r.console.Kill()
	default:
		return fmt.Errorf("unhandled step kind %d", st.Kind)
	}
}

func (r *Runner) wait(st Step) error {
	deadline := r.clk.now().Add(st.Timeout)
	for {
		if st.Pattern.MatchString(r.Render()) {
			return nil
		}
		r.mu.Lock()
		finished := r.done
		r.mu.Unlock()
		if r.clk.now().After(deadline) {
			return fmt.Errorf("wait %s: timed out after %s; last frame:\n%s", st.Pattern, st.Timeout, r.Render())
		}
		if finished {
			// One more check: the last chunk may have landed between the
			// match check above and reading r.done.
			if st.Pattern.MatchString(r.Render()) {
				return nil
			}
			return fmt.Errorf("wait %s: the child exited before this matched; last frame:\n%s", st.Pattern, r.Render())
		}
		<-r.clk.after(pollInterval)
	}
}

// pollInterval is the wait-loop poll granularity.
const pollInterval = 25 * time.Millisecond

func (r *Runner) snapshot(path string) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(r.Render()+"\n"), 0o644)
}
