// Package readiness computes, caches, and presents how resumable a session is.
//
// The engine has always known this: preflight builds a full environment report
// before any native launch. Until now that answer only appeared after the user
// had already chosen a session and asked to resume it. Here it is computed for
// the rows on screen, so the answer arrives before the choice rather than after.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package readiness

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// VerifyFunc produces an environment report for one record. It is the only
// engine dependency, injected so the prober is testable without a workspace,
// a vendor binary, or a clock.
type VerifyFunc func(ctx context.Context, record sessionindex.Record) (preflight.Report, error)

// Result is one completed probe.
type Result struct {
	Key       string
	Readiness ui.Readiness
	Report    preflight.Report
	Err       error
}

// ProbedMsg tells a surface that one or more probes finished and the cache is
// worth reading again. It carries no payload because the cache is the state.
type ProbedMsg struct{ Keys []string }

// Prober computes readiness lazily and remembers the answer.
//
// Probing is deliberately not eager over the whole index. A single report runs
// workspace and vendor-version checks, so probing hundreds of rows would cost
// far more than it tells anyone. Only rows that reach the screen are probed.
type Prober struct {
	verify VerifyFunc

	mu      sync.Mutex
	cache   map[string]Result
	pending map[string]struct{}
}

// New builds a prober. A nil verify makes every lookup report unknown, which is
// how the surface degrades when no verifier is configured.
func New(verify VerifyFunc) *Prober {
	return &Prober{
		verify:  verify,
		cache:   make(map[string]Result),
		pending: make(map[string]struct{}),
	}
}

// Enabled reports whether readiness is being computed at all. A surface hides
// the status column entirely when it is not, rather than showing a column of
// placeholders that will never resolve.
func (p *Prober) Enabled() bool { return p != nil && p.verify != nil }

// Lookup returns the cached readiness for a record without blocking.
func (p *Prober) Lookup(record sessionindex.Record) ui.Readiness {
	if p == nil {
		return ui.ReadinessUnknown
	}
	// A record the index already knows is read-only can never resume, and that
	// is knowable without probing anything.
	if record.ReadOnlyReason != "" || !record.CanResume {
		return ui.ReadinessBlocked
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if result, ok := p.cache[record.Key]; ok {
		return result.Readiness
	}
	return ui.ReadinessUnknown
}

// Report returns the cached report for a record, if one has been computed.
func (p *Prober) Report(key string) (preflight.Report, bool) {
	if p == nil {
		return preflight.Report{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	result, ok := p.cache[key]
	if !ok || result.Err != nil {
		return preflight.Report{}, false
	}
	return result.Report, true
}

// Probe returns a command that computes readiness for every record that has no
// answer yet. Records already cached or already in flight are skipped, so a
// surface may call this on every keystroke without doing duplicate work.
//
// It returns nil when there is nothing to do, which Bubble Tea treats as a
// no-op.
func (p *Prober) Probe(ctx context.Context, records []sessionindex.Record) tea.Cmd {
	if !p.Enabled() || len(records) == 0 {
		return nil
	}
	if ctx == nil {
		// A nil context reaches the index as a nil argument and panics inside
		// database/sql rather than failing. Probing is best-effort background
		// work, so it degrades to an unbounded context instead of taking the
		// whole surface down.
		ctx = context.Background()
	}
	p.mu.Lock()
	todo := make([]sessionindex.Record, 0, len(records))
	for _, record := range records {
		if record.ReadOnlyReason != "" || !record.CanResume {
			continue
		}
		if _, done := p.cache[record.Key]; done {
			continue
		}
		if _, inFlight := p.pending[record.Key]; inFlight {
			continue
		}
		p.pending[record.Key] = struct{}{}
		todo = append(todo, record)
	}
	p.mu.Unlock()
	if len(todo) == 0 {
		return nil
	}

	commands := make([]tea.Cmd, 0, len(todo))
	for _, record := range todo {
		commands = append(commands, p.probeOne(ctx, record))
	}
	return tea.Batch(commands...)
}

func (p *Prober) probeOne(ctx context.Context, record sessionindex.Record) tea.Cmd {
	return func() tea.Msg {
		report, err := p.verify(ctx, record)
		result := Result{Key: record.Key, Report: report, Err: err}
		result.Readiness = FromReport(report, err)

		p.mu.Lock()
		p.cache[record.Key] = result
		delete(p.pending, record.Key)
		p.mu.Unlock()

		return ProbedMsg{Keys: []string{record.Key}}
	}
}

// FromReport maps a preflight decision onto a readiness glyph.
//
// A failed probe is reported as unknown rather than blocked. The environment
// might be perfectly fine; all that is known is that it could not be inspected,
// and claiming a session cannot resume on that basis would be a lie the user
// would have to disprove by hand.
func FromReport(report preflight.Report, err error) ui.Readiness {
	if err != nil {
		return ui.ReadinessUnknown
	}
	switch report.Decision {
	case preflight.DecisionReady:
		return ui.ReadinessReady
	case preflight.DecisionConfirmationRequired:
		return ui.ReadinessWarn
	case preflight.DecisionBlocked:
		return ui.ReadinessBlocked
	default:
		return ui.ReadinessUnknown
	}
}
