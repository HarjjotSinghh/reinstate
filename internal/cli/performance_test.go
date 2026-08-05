package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// BenchmarkLocalMetadataCommands separates command construction/rendering,
// derived-index refresh/query work, and preflight orchestration from physical
// vendor-file scanning. The session-index benchmarks cover the scanner layer.
// Run this benchmark unchanged on macOS, Linux, and native Windows.
func BenchmarkLocalMetadataCommands(b *testing.B) {
	workspacePath := b.TempDir()
	sourceRoot := filepath.Join(b.TempDir(), ".claude")
	records := make([]sessionindex.Record, 1_000)
	for index := range records {
		id := fmt.Sprintf("controlled-%04d", index)
		records[index] = sessionindex.Record{
			Key:           "claude:" + id,
			ID:            id,
			Agent:         sessionindex.AgentClaude,
			Title:         "controlled synthetic session " + id,
			Project:       "controlled-performance",
			Workspace:     workspacePath,
			Branch:        "main",
			UpdatedAt:     time.Unix(1_800_000_000-int64(index), 0).UTC(),
			SizeBytes:     256,
			MessageCount:  2,
			PromptPreview: "controlled synthetic preview",
			CanResume:     true,
			CanFork:       true,
			SourcePath:    filepath.Join(sourceRoot, "projects", "controlled", id+".jsonl"),
			SourceModTime: int64(index + 1),
			SourceSize:    256,
			SearchText:    "controlled synthetic searchable marker",
		}
	}
	source := benchmarkSessionSource{records: records}
	verifier := benchmarkPreflightVerifier{}

	benchmarks := []struct {
		name string
		args []string
	}{
		{name: "version", args: []string{"version", "--json"}},
		{name: "help", args: []string{"--help"}},
		{name: "sessions_1000", args: []string{"sessions", "--limit", "1000", "--json"}},
		{name: "search_1000", args: []string{"search", "searchable", "marker", "--limit", "1000", "--json"}},
		{name: "inspect_1000", args: []string{"inspect", "claude:controlled-0500", "--json"}},
		{name: "resume_dry_run_1000", args: []string{"resume", "claude:controlled-0500", "--dry-run", "--json"}},
		{name: "fork_dry_run_1000", args: []string{"fork", "claude:controlled-0500", "--dry-run", "--json"}},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			home := b.TempDir()
			b.Setenv("REINSTATE_HOME", home)
			opts := Options{
				Name:              "rein",
				Stdout:            io.Discard,
				Stderr:            io.Discard,
				SessionSources:    []sessionindex.Source{source},
				PreflightVerifier: verifier,
			}
			// Populate the private index outside the timer for commands whose
			// production contract starts from a warm derived index.
			if benchmark.name != "version" && benchmark.name != "help" {
				opts.Args = []string{"sessions", "--limit", "1", "--json"}
				if code := Execute(opts); code != ExitOK {
					b.Fatalf("seed command exit = %d", code)
				}
			}
			opts.Args = benchmark.args
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if code := Execute(opts); code != ExitOK {
					b.Fatalf("command exit = %d", code)
				}
			}
		})
	}
}

type benchmarkSessionSource struct {
	records []sessionindex.Record
}

func (source benchmarkSessionSource) Name() string { return sessionindex.AgentClaude }

func (source benchmarkSessionSource) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return sessionindex.ScanResult{}, err
	}
	return sessionindex.ScanResult{Records: append([]sessionindex.Record(nil), source.records...)}, nil
}

type benchmarkPreflightVerifier struct{}

func (benchmarkPreflightVerifier) Verify(ctx context.Context, input preflight.Input) (preflight.Report, error) {
	if err := ctx.Err(); err != nil {
		return preflight.Report{}, err
	}
	return preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		SessionRef:    input.SessionRef,
		Decision:      preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID:         "benchmark.ready",
			Status:     preflight.StatusPresent,
			Severity:   preflight.SeverityInfo,
			Provenance: workspace.ProvenanceUnavailable,
			Message:    "controlled benchmark verifier completed",
		}},
	}, nil
}
