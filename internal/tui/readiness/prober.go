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
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
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

// MaxConcurrentProbes bounds how many verifications run at once.
//
// A single probe is not free: preflight.Verify spawns several sequential Git
// processes plus a vendor version probe, all sharing one wall clock (see
// ProbeTimeout below). Probe hands back one tea.Cmd per record and Bubble Tea
// runs every command in a batch on its own goroutine simultaneously, so
// fanning out one goroutine per visible row starts that many independent
// report pipelines — and that many times as many OS processes — in the same
// instant. A page with a handful of rows absorbs this fine; a page with a
// couple dozen (the switcher's default all-projects scope routinely has more
// rows than any single project does, and on a host with other agent sources
// configured can easily have several dozen) does not: the resulting
// process-creation stampede can blow through a report's own timeout budget
// before its checks finish. FromReport already answers "could not be
// evaluated inside the budget" with Unknown rather than Blocked (see below),
// so a stampede no longer *lies* about a session's readiness — but it can
// still leave rows stuck reading "checking" far longer than a background
// computation should. Capping how many verifications run at once, alongside
// ProbeTimeout's larger budget, is what keeps that window small on a
// contended host: this value was picked empirically (see
// docs/testing/results for the readiness-glyph acceptance method) as the
// largest that still lets every row in a several-dozen-row, real-world-noisy
// ScopeAll listing settle within ProbeTimeout on ordinary Windows hardware.
const MaxConcurrentProbes = 6

// ProbeTimeout bounds each background verification the prober runs.
//
// It is deliberately more generous than preflight.DefaultVerifierTimeout,
// which is the *launch* path's budget: a user who just pressed enter to
// resume is waiting on that call and a strict two-second bound is the right
// tradeoff there. A background readiness probe has no one waiting on it in
// the same way — the row already shows "checking" — so it can afford a wider
// window to actually finish, which is what keeps FromReport's
// could-not-evaluate-in-time answer (Unknown, not Blocked) from being the
// common case on a host with more than a handful of rows to probe.
//
// newReadinessProber (internal/cli/switcher.go) is what actually applies
// this: it wraps the launch path's own preflight.Verifier with
// preflight.WithTimeout(verifier, readiness.ProbeTimeout) for the copy handed
// to this package, leaving the launch path's verifier — used for the actual
// resume attempt and the warning checklist — on its own unchanged budget.
const ProbeTimeout = 12 * time.Second

// maxProbeRetries bounds how many times Probe will re-verify a record whose
// most recent answer was Unknown because its report could not be evaluated in
// time (see FromReport). Retrying at all is the point — a report that only
// means "the host was briefly too busy to finish" deserves another try rather
// than a permanent, unearned verdict — but a host that is durably too slow
// (or a workspace whose probe genuinely cannot complete, ever) must not turn
// that into an unbounded loop competing for MaxConcurrentProbes' slots
// forever. Once a record has been retried this many times and is still
// Unknown, Probe stops scheduling it: the row keeps showing "checking"
// (ui.ReadinessUnknown, the same glyph it already had) rather than flickering
// to something else, and a future call — say, once the user has restarted the
// process on a quieter host — starts the count over.
const maxProbeRetries = 3

// Prober computes readiness lazily and remembers the answer.
//
// Probing is deliberately not eager over the whole index. A single report runs
// workspace and vendor-version checks, so probing hundreds of rows would cost
// far more than it tells anyone. Only rows that reach the screen are probed,
// and even those share a bounded pool of concurrent verifications rather than
// all starting at once — see MaxConcurrentProbes.
type Prober struct {
	verify VerifyFunc
	// slots bounds concurrent verifications. A buffered channel is used as a
	// semaphore: probeOne sends before verifying and receives after, so at
	// most cap(slots) verifications run at any moment regardless of how many
	// records Probe was asked to cover in one call.
	slots chan struct{}

	mu      sync.Mutex
	cache   map[string]Result
	pending map[string]struct{}
	// retries counts, per record key, how many times a probe has come back
	// Unknown because its report could not be evaluated in time. It exists
	// only for keys currently sitting on an Unknown answer; a terminal
	// (Ready/Warn/Blocked) result clears its entry, so a record that later
	// needs re-probing for an unrelated reason starts this count fresh. See
	// maxProbeRetries.
	retries map[string]int
}

// New builds a prober. A nil verify makes every lookup report unknown, which is
// how the surface degrades when no verifier is configured.
func New(verify VerifyFunc) *Prober {
	return &Prober{
		verify:  verify,
		slots:   make(chan struct{}, MaxConcurrentProbes),
		cache:   make(map[string]Result),
		pending: make(map[string]struct{}),
		retries: make(map[string]int),
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
		if cached, done := p.cache[record.Key]; done {
			// A terminal answer (Ready, Warn, or a genuinely Blocked report)
			// is never re-probed. An Unknown one is retried — up to
			// maxProbeRetries — because it does not mean "this session
			// cannot resume", it means "the last attempt could not tell
			// either way", and a background prober can afford another try.
			if cached.Readiness != ui.ReadinessUnknown || p.retries[record.Key] >= maxProbeRetries {
				continue
			}
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
		// Wait for a free slot before verifying at all, so at most
		// MaxConcurrentProbes reports run concurrently no matter how many
		// commands Bubble Tea started in this batch. A context that is
		// cancelled while still queued (the surface quit, or the caller's
		// deadline passed) is left pending rather than answered: nothing was
		// learned about the record's own environment, so it deserves another
		// try later, not a cached verdict manufactured from a queueing delay.
		select {
		case p.slots <- struct{}{}:
		case <-ctx.Done():
			p.mu.Lock()
			delete(p.pending, record.Key)
			p.mu.Unlock()
			return ProbedMsg{}
		}
		defer func() { <-p.slots }()

		report, err := p.verify(ctx, record)
		result := Result{Key: record.Key, Report: report, Err: err}
		result.Readiness = FromReport(report, err)

		p.mu.Lock()
		p.cache[record.Key] = result
		if result.Readiness == ui.ReadinessUnknown {
			p.retries[record.Key]++
		} else {
			// A terminal answer. Whatever count was accumulating no longer
			// applies: a later reason to re-probe this key (a fresh Probe
			// call sees the cache miss because Ready/Warn/Blocked stay
			// cached, so today this fires only when a caller retires an
			// entry some other way) starts the retry budget over rather than
			// inheriting an unrelated attempt count.
			delete(p.retries, record.Key)
		}
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
//
// A *successful* probe that concluded Blocked gets the same treatment when
// every one of its blocking checks is itself a could-not-evaluate result
// rather than a finding — see uninspectable. preflight.Verify shares one wall
// clock across several independent observers (internal/preflight/verify.go),
// and a budget that expires mid-observation is recorded as a normal, no-error
// report: the check in question is marked Severity block with ExitCode
// exitcode.Runtime — verify.go's own preferredBlockExit names that exit code
// as meaning specifically "the verifier could not produce trustworthy
// evidence," as opposed to exitcode.Compatibility or exitcode.Safety, both of
// which name an independently observed problem — and Severity block alone
// makes the whole report Blocked. Nothing about the environment was actually
// established in the exitcode.Runtime case, so a row rendering it must not
// read any more confidently than a report that errored outright — this is
// row 13 of the CLI experience matrix, "readiness glyphs resolve for visible
// rows; a read-only agent shows blocked without a probe," and its
// v0.6.0-rc.7 regression: under load (a ScopeAll listing with many rows to
// probe at once) enough checks timed out that most launches showed at least
// one row as Blocked whose ground truth was Ready or Warn, and — because
// FromReport could not yet tell a timeout-shaped Blocked report from a
// genuine one, and Probe cached whatever it returned as final — the wrong
// glyph never corrected itself. Its own v0.6.0-rc.8 follow-on regression
// showed that Status alone (specifically, StatusUnknown) is not a safe stand-in
// for that exit code: see uninspectable's comment.
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
		if uninspectable(report) {
			return ui.ReadinessUnknown
		}
		return ui.ReadinessBlocked
	default:
		return ui.ReadinessUnknown
	}
}

// uninspectable reports whether a Blocked report's decision rests entirely on
// checks that could not be evaluated, rather than on any actual finding.
//
// The one and only signal for "could not evaluate" is ExitCode
// exitcode.Runtime, the code preferredBlockExit (verify.go) documents as
// meaning specifically "the verifier could not produce trustworthy evidence"
// — as opposed to exitcode.Compatibility or exitcode.Safety, both
// independently observed problems. Status is deliberately *not* part of this
// test, even though every check this function currently sees with a
// could-not-evaluate meaning happens to carry Status unknown: verify.go's
// agentChecks proved that Status alone is not a reliable signal. A version
// probe that ran out of its own time budget and a version probe that
// deterministically failed against a corrupted, tampered, or simply
// non-launchable executable both surface as agentcheck.StatusError, and nothing
// forced the resulting preflight Check to pick a different Status for the
// two — before the exit-code split introduced alongside this comment,
// agentChecks gave both the identical shape (Status unknown, Severity block,
// ExitCode exitcode.Compatibility), which is indistinguishable from a
// once-in-a-while contention timeout and made this function treat a
// permanently, deterministically broken agent install as "still checking"
// forever (CLI experience row 13's v0.6.0-rc.8 regression: the row never
// settled, and the actionable repair message the check exists to carry —
// "install a native agent version verified by this Reinstate release", or
// "retry ... or pass --allow-untested" — never reached the screen). Keying
// only off ExitCode, which agentChecks now sets to exitcode.Runtime
// specifically and only for the genuinely time-starved case, resolves the
// ambiguity at its source instead of guessing from Status. A report with no
// blocking check at all is not "uninspectable" by this definition — Decision
// is only Blocked when at least one exists — so this only returns true when
// every blocking check present carries ExitCode exitcode.Runtime.
func uninspectable(report preflight.Report) bool {
	sawBlock := false
	for _, check := range report.Checks {
		if check.Severity != preflight.SeverityBlock {
			continue
		}
		sawBlock = true
		if check.ExitCode == exitcode.Runtime {
			continue
		}
		return false
	}
	return sawBlock
}
