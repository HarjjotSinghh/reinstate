// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog" // agents.MustRegister side effects
	_ "github.com/HarjjotSinghh/reinstate/internal/cli"            // agentcheck.SetDefinitions + native launch lookup
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Why the bench has to seed prelaunch baselines at all:
//
// READY means zero warnings and zero blocks. But a session with no stored
// prelaunch baseline gets `baseline.unavailable` — an UNCONDITIONAL warning
// (internal/preflight/verify.go) that no fixture can avoid. So a perfectly
// healthy workspace still renders as NEEDS ACKNOWLEDGEMENT on its first visit.
// That is correct product behaviour: the first verified launch of a session is
// the one a human is supposed to look at.
//
// The bench needs to be green on FIRST launch, so it does for the READY rows
// exactly what `rein resume` does after a successful launch — snapshot the
// environment and store it. Everything below is the same call sequence the CLI
// uses (internal/cli/sessions.go verifyLocalRecord + the BaselineFromReport /
// PutPrelaunchBaseline pair).
//
// WHY THE DERIVED FORM AND NOT A HAND-BUILT STRUCT. A hand-seeded baseline with
// Capabilities: nil is only correct on a machine that has no managed Claude
// configuration. preflight.DefaultService sets ManagedRoot to "/" on darwin, so
// /Library/Application Support/ClaudeCode is scanned; on a host that has one,
// every hand-seeded READY row silently degrades to NEEDS ACKNOWLEDGEMENT with
// capability.* warnings, on that host only. BaselineFromReport copies whatever
// discovery actually found, so the environment is compared against itself and
// every check comes back info. That is the single strongest reason to prefer
// this path.
//
// RESIDUAL HAZARD even here: BaselineFromReport skips runtimes whose observed
// version is empty. If a READY workspace declares a runtime whose binary is
// absent, the derived baseline omits it and the very next Verify reports
// `runtime.<name>.baseline` "is new since the previous prelaunch observation".
// The no-runtime-declarations rule for READY workspaces (see workspace.go) is
// therefore still required.

// seedBaselines opens the sandbox index, refreshes it, and stores a derived
// prelaunch baseline for each requested session reference.
//
// Preconditions the caller must satisfy, in this order:
//  1. every fixture file is final (nothing may change after the snapshot);
//  2. the generator's OWN process environment already points at the sandbox
//     (HOME, USERPROFILE, REINSTATE_HOME, CLAUDE_CONFIG_DIR, CODEX_HOME, PATH).
//     preflight.DefaultService reads CLAUDE_CONFIG_DIR/CODEX_HOME and the user
//     home AT CONSTRUCTION TIME, and the version probe resolves the vendor
//     shims off PATH. Constructing it before the env is set would snapshot the
//     developer's real ~/.claude.
func seedBaselines(ctx context.Context, reinstateHome string, references []string) error {
	sources, err := localSources()
	if err != nil {
		return err
	}
	index, err := sessionindex.OpenIndex(reinstateHome, sources...)
	if err != nil {
		return err
	}
	defer func() { _ = index.Close() }()

	// The FULL refresh, not RefreshAgent. PutPrelaunchBaseline has a foreign key
	// to sessions.key and probes existence first; an agent-narrowed refresh
	// leaves the other vendor's rows missing and the insert fails with
	// ErrNotFound.
	refresh, err := index.Refresh(ctx)
	if err != nil {
		return err
	}

	service := preflight.DefaultService()
	for _, reference := range references {
		record, err := index.Store().Resolve(ctx, reference)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", reference, err)
		}
		// Mirrors internal/cli/sessions.go verifyLocalRecord, with Baseline nil
		// because this is by definition the first observation.
		report, err := service.Verify(ctx, preflight.Input{
			SessionRef:  record.Reference(),
			Agent:       record.Agent,
			Workspace:   record.Workspace,
			AgentRoot:   sessionindex.AgentRoot(record),
			Recorded:    record.RecordedEnvironment,
			Baseline:    nil,
			SourceFresh: refresh.SourceFresh(record.Agent),
		})
		if err != nil {
			return fmt.Errorf("verify %s: %w", reference, err)
		}
		// The report here is normally confirmation_required — baseline.unavailable
		// is present by construction — and BaselineFromReport refuses only a
		// BLOCKED report, so that is fine. A blocked one is reachable on purpose
		// under -stale-claude, where every Claude row fails agent.version; a
		// blocked row cannot have a baseline and does not want one. Say so
		// loudly rather than aborting, because on any OTHER run this is the
		// signal that a fixture is broken.
		if report.Decision == preflight.DecisionBlocked {
			var blocking []string
			for _, check := range report.Checks {
				if check.Severity == preflight.SeverityBlock {
					blocking = append(blocking, check.ID)
				}
			}
			fmt.Fprintf(os.Stderr,
				"tuisandbox: %s is blocked (%s); skipping its baseline — it cannot be READY\n",
				reference, strings.Join(blocking, ", "))
			continue
		}
		baseline, err := preflight.BaselineFromReport(report, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("derive baseline for %s: %w", reference, err)
		}
		if err := index.Store().PutPrelaunchBaseline(ctx, baseline); err != nil {
			return fmt.Errorf("store baseline for %s: %w", reference, err)
		}
	}
	return nil
}

// localSources rebuilds what internal/cli's unexported defaultLocalSources()
// does. There is no exported helper, and the blank import of the catalog is
// mandatory: without it agents.Capable returns an empty list and the sandbox
// indexes nothing at all.
func localSources() ([]sessionindex.Source, error) {
	var sources []sessionindex.Source
	for _, descriptor := range agents.Capable(agents.CapabilityIndex) {
		if descriptor.NewIndexSource == nil {
			continue
		}
		// agents.Env{} resolves the home through os.UserHomeDir(), which is the
		// sandbox home because HOME/USERPROFILE were already redirected.
		source, err := descriptor.NewIndexSource(agents.Env{})
		if err != nil || source == nil {
			continue
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no index-capable agents registered")
	}
	return sources, nil
}
