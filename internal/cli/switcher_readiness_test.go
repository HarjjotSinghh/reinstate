// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// freshnessReport mirrors what the real verifier emits for source.fresh, which
// is the only check this test needs. It must satisfy preflight.ValidateReport,
// because verifyLocalRecord rejects a malformed report before the prober ever
// sees it — so a lazily-built stub would make this test pass for the wrong
// reason.
func freshnessReport(input preflight.Input) preflight.Report {
	report := preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		SessionRef:    input.SessionRef,
	}
	if input.SourceFresh {
		report.Decision = preflight.DecisionReady
		report.Checks = []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Actual: true, Provenance: workspace.ProvenanceCurrentObservation,
			Message: "the selected vendor source was refreshed successfully",
		}}
		return report
	}
	report.Decision = preflight.DecisionBlocked
	report.BlockExitCode = exitcode.Safety
	report.Checks = []preflight.Check{{
		ID: "source.fresh", Status: preflight.StatusChanged, Severity: preflight.SeverityBlock,
		Actual: false, Provenance: workspace.ProvenanceUnavailable,
		Message:  "the selected vendor source could not be refreshed safely",
		Repair:   "resolve the source scan failure and retry before launching",
		ExitCode: exitcode.Safety,
	}}
	return report
}

// TestReadinessProberThreadsRealSourceFreshness guards a defect that made the
// switcher's headline feature report the exact opposite of the truth.
//
// preflight's `source.fresh` check is SeverityBlock: a report built with
// SourceFresh false is DecisionBlocked regardless of how healthy the
// environment is. The prober originally passed a hardcoded false, so every row
// in the list read "cannot resume" — including sessions that `rein inspect`
// simultaneously reported as merely needing acknowledgement.
//
// The test asserts the freshness actually observed for each agent reaches
// preflight, rather than asserting a particular decision, so it keeps holding
// if the checks around it change.
func TestReadinessProberThreadsRealSourceFreshness(t *testing.T) {
	const (
		freshAgent = sessionindex.AgentClaude
		staleAgent = sessionindex.AgentCodex
	)

	// A refresh in which one source scanned cleanly and one failed. SourceFresh
	// is derived from the per-source error, so this is the shape the real
	// refresh produces when one vendor tree is unreadable.
	refresh := sessionindex.RefreshResult{
		Sources: []sessionindex.SourceRefresh{
			{Name: freshAgent},
			{Name: staleAgent, Error: "scan failed"},
		},
	}
	if !refresh.SourceFresh(freshAgent) {
		t.Fatalf("fixture is wrong: %s should be fresh", freshAgent)
	}
	if refresh.SourceFresh(staleAgent) {
		t.Fatalf("fixture is wrong: %s should be stale", staleAgent)
	}

	for _, test := range []struct {
		name      string
		agent     string
		wantFresh bool
	}{
		{name: "a source that refreshed cleanly is reported fresh", agent: freshAgent, wantFresh: true},
		{name: "a source that failed to refresh is reported stale", agent: staleAgent, wantFresh: false},
		{name: "an agent absent from the refresh is reported stale", agent: "gemini", wantFresh: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []preflight.Input
			options := localCommandOptions{
				verifier: preflightVerifierFunc(func(_ context.Context, input preflight.Input) (preflight.Report, error) {
					observed = append(observed, input)
					return freshnessReport(input), nil
				}),
			}

			record := sessionindex.Record{
				Key:       sessionindex.CompositeReference(test.agent, "demo"),
				ID:        "demo",
				Agent:     test.agent,
				CanResume: true,
				CanFork:   true,
			}

			// A nil index is enough here: verifyLocalRecord reads the baseline
			// through it and tolerates its absence, which is the same path a
			// session with no prior launch takes.
			index, err := openLocalIndexForTest(t)
			if err != nil {
				t.Fatalf("open index: %v", err)
			}
			t.Cleanup(func() { _ = index.Close() })

			prober := newReadinessProber(options, index, refresh)
			cmd := prober.Probe(context.Background(), []sessionindex.Record{record})
			if cmd == nil {
				t.Fatal("Probe returned no command for an unprobed record")
			}
			cmd()

			if len(observed) != 1 {
				t.Fatalf("verifier called %d times, want 1", len(observed))
			}
			if observed[0].SourceFresh != test.wantFresh {
				t.Fatalf("preflight received SourceFresh=%v, want %v", observed[0].SourceFresh, test.wantFresh)
			}

			// The consequence, stated explicitly: a stale source blocks, and a
			// fresh one does not. This is what the reader sees in the list.
			gotReadiness := prober.Lookup(record)
			wantReadiness := ui.ReadinessBlocked
			if test.wantFresh {
				wantReadiness = ui.ReadinessReady
			}
			if gotReadiness != wantReadiness {
				t.Fatalf("readiness = %v, want %v", gotReadiness, wantReadiness)
			}
		})
	}
}

// TestReadinessProberSkipsRecordsThatCannotResume asserts that a read-only
// agent is reported blocked without any probe at all. Probing it would run
// workspace and vendor checks to answer a question the index has already
// answered.
func TestReadinessProberSkipsRecordsThatCannotResume(t *testing.T) {
	calls := 0
	options := localCommandOptions{
		verifier: preflightVerifierFunc(func(context.Context, preflight.Input) (preflight.Report, error) {
			calls++
			return preflight.Report{Decision: preflight.DecisionReady}, nil
		}),
	}
	index, err := openLocalIndexForTest(t)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	t.Cleanup(func() { _ = index.Close() })

	refresh := sessionindex.RefreshResult{Sources: []sessionindex.SourceRefresh{{Name: "gemini"}}}
	prober := newReadinessProber(options, index, refresh)

	record := sessionindex.Record{
		Key:            sessionindex.CompositeReference("gemini", "demo"),
		ID:             "demo",
		Agent:          "gemini",
		CanResume:      false,
		ReadOnlyReason: "gemini sessions are read-only",
	}
	if got := prober.Lookup(record); got != ui.ReadinessBlocked {
		t.Fatalf("readiness = %v, want blocked", got)
	}
	if cmd := prober.Probe(context.Background(), []sessionindex.Record{record}); cmd != nil {
		t.Fatal("Probe scheduled work for a record that cannot resume")
	}
	if calls != 0 {
		t.Fatalf("verifier ran %d times for a read-only record, want 0", calls)
	}
}

// openLocalIndexForTest opens a throwaway index with no sources.
func openLocalIndexForTest(t *testing.T) (*sessionindex.Index, error) {
	t.Helper()
	t.Setenv("REINSTATE_HOME", t.TempDir())
	return openLocalIndex(localCommandOptions{sources: []sessionindex.Source{}})
}

// TestIndexLoaderNarrowsByWorkspaceAcrossSeparators guards a defect that made
// bare `rein` show an empty list inside every project on Windows.
//
// Vendors record the working directory in whatever form their runtime produced
// and the index stores it verbatim, so a session file can hold a forward-slash
// path while the process standing in that directory reports a backslash one.
// The scope used to be a literal path match against the index, which found
// nothing.
func TestIndexLoaderNarrowsByWorkspaceAcrossSeparators(t *testing.T) {
	for _, test := range []struct {
		name      string
		stored    string
		standing  string
		wantMatch bool
	}{
		{name: "identical", stored: "/home/u/Projects/app", standing: "/home/u/Projects/app", wantMatch: true},
		{name: "stored forward, standing native windows", stored: "D:/Projects/app", standing: `D:\Projects\app`, wantMatch: true},
		{name: "stored native windows, standing forward", stored: `D:\Projects\app`, standing: "D:/Projects/app", wantMatch: true},
		{name: "case differs", stored: "D:/Projects/App", standing: `d:\projects\app`, wantMatch: true},
		{name: "trailing separator", stored: "/home/u/Projects/app", standing: "/home/u/Projects/app/", wantMatch: true},
		{name: "a subdirectory of the workspace still belongs to it", stored: "/home/u/Projects/app/sub", standing: "/home/u/Projects/app", wantMatch: true},
		{name: "a different project does not match", stored: "/home/u/Projects/other", standing: "/home/u/Projects/app", wantMatch: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := strings.HasPrefix(normalizedWorkspace(test.stored), normalizedWorkspace(test.standing))
			if got != test.wantMatch {
				t.Fatalf("stored %q standing %q: match = %v, want %v (normalized %q vs %q)",
					test.stored, test.standing, got, test.wantMatch,
					normalizedWorkspace(test.stored), normalizedWorkspace(test.standing))
			}
		})
	}
}

// TestCurrentProjectScopeUsesAPortableFilter asserts the value handed to the
// index carries no path separator. Filter.Project is a literal substring match,
// and a path is not portable enough to match literally.
func TestCurrentProjectScopeUsesAPortableFilter(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "my-project", ".git"), 0o700); err != nil {
		t.Fatalf("create fixture repository: %v", err)
	}
	working := filepath.Join(directory, "my-project")
	t.Chdir(working)

	scope := currentProjectScope()
	if scope.filter != "my-project" {
		t.Fatalf("filter = %q, want the repository name", scope.filter)
	}
	if strings.ContainsAny(scope.filter, `/\`) {
		t.Fatalf("filter %q contains a path separator; it is matched literally by the index", scope.filter)
	}
	if scope.label != "my-project" {
		t.Fatalf("label = %q, want the repository name", scope.label)
	}
	if normalizedWorkspace(scope.root) != normalizedWorkspace(working) {
		t.Fatalf("root = %q, want %q", scope.root, working)
	}
}
