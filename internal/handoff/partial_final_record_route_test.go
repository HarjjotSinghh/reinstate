package handoff

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// This file closes the claude:D4/codex:D4/grok:D4 gap a verified rejection
// found in the first cut of these fixtures: adding a Windows-shaped
// testdata/handoff/<agent>/partial-final-record-windows/ tree, and asserting
// its boundary math at the transcript.Reader layer (internal/transcript's
// *_test.go), never actually drove the code path the tagged finding is
// about. That finding is specifically that `rein handoff --no-launch --json`
// — internal/handoff's Plan(), the exact function
// internal/cli/handoff.go calls — could not reach a byte-exact
// raw_source.byte_offset/artifact_sha256 cross-check against these fixtures
// on native Windows. Plan() is what has to run, end to end, against the
// committed fixture files through the real, registered transcript.Reader
// (opts.Reader is left nil so Plan resolves it exactly like production
// does), for that gap to be demonstrated closed.
//
// It also settles what actually gates the workspace-remap/refusal Plan()
// applies before it will trust a recorded workspace at all
// (shouldRemapWorkspace/sameProjectLeaf in workspace.go): the literal
// "/fixture-user/" (or "/synthetic-user/") substring, checked identically on
// every OS, and then whether the operator's git-root directory is literally
// named the same as the recorded workspace's last path segment ("demo" for
// every fixture in this tree). isForeignOSPath never enters into it for any
// fixture built to this repository's own fixture-user convention — a
// Windows-shaped recorded workspace and a macOS-shaped one behave
// identically here, on every OS, which TestPlanPartialFinalRecordRefusalIsOSShapeIndependent
// below asserts directly rather than merely stating in a README.
func expectedJSONLBoundary(t *testing.T, raw []byte) (offset int64, digestHex string) {
	t.Helper()
	var pos, lastComplete int64
	rest := raw
	for {
		nl := bytes.IndexByte(rest, '\n')
		if nl < 0 {
			break // trailing bytes with no terminating newline: never complete
		}
		line := bytes.TrimRight(rest[:nl], " \t\r")
		consumed := int64(nl + 1)
		if len(line) > 0 && json.Valid(line) {
			lastComplete = pos + consumed
		}
		pos += consumed
		rest = rest[nl+1:]
	}
	if lastComplete == 0 {
		t.Fatalf("fixture has no complete, valid JSONL record")
	}
	if lastComplete == int64(len(raw)) {
		t.Fatalf("fixture's last valid record reaches EOF; want a torn trailing record after it")
	}
	sum := sha256.Sum256(raw[:lastComplete])
	return lastComplete, hex.EncodeToString(sum[:])
}

// partialFinalRecordRouteCase is one committed partial-final-record fixture
// driven through the real Plan() pipeline.
type partialFinalRecordRouteCase struct {
	name string
	// buildRecord returns the source record and the path to the file whose
	// bytes carry the truncation boundary (independently re-hashed here,
	// never trusted from the reader under test).
	buildRecord func(t *testing.T) (sessionindex.Record, string)
	toAgent     string
}

func partialFinalRecordRouteCases(t *testing.T) []partialFinalRecordRouteCase {
	t.Helper()
	root := repoRoot(t)
	claudeCase := func(name, dir, projectDir, workspacePath string) partialFinalRecordRouteCase {
		return partialFinalRecordRouteCase{
			name: "claude-" + name, toAgent: sessionindex.AgentCodex,
			buildRecord: func(t *testing.T) (sessionindex.Record, string) {
				path := filepath.Join(root, "testdata", "handoff", "claude", dir, "projects", projectDir, "session-syn-001.jsonl")
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				return sessionindex.Record{
					Key: "claude:00000000-0000-4000-8000-000000000001", ID: "00000000-0000-4000-8000-000000000001",
					Agent: sessionindex.AgentClaude, Project: "github.com/example/demo", Workspace: workspacePath,
					SourcePath: path, SourceModTime: info.ModTime().UnixNano(), SourceSize: info.Size(),
				}, path
			},
		}
	}
	codexCase := func(name, dir, file, workspacePath string) partialFinalRecordRouteCase {
		return partialFinalRecordRouteCase{
			name: "codex-" + name, toAgent: sessionindex.AgentClaude,
			buildRecord: func(t *testing.T) (sessionindex.Record, string) {
				path := filepath.Join(root, "testdata", "handoff", "codex", dir, file)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				return sessionindex.Record{
					Key: "codex:00000000-0000-4000-8000-00000000ff01", ID: "00000000-0000-4000-8000-00000000ff01",
					Agent: sessionindex.AgentCodex, Project: "github.com/example/demo", Workspace: workspacePath,
					SourcePath: path, SourceModTime: info.ModTime().UnixNano(), SourceSize: info.Size(),
				}, path
			},
		}
	}
	grokCase := func(name, dir string) partialFinalRecordRouteCase {
		return partialFinalRecordRouteCase{
			name: "grok-" + name, toAgent: sessionindex.AgentClaude,
			buildRecord: func(t *testing.T) (sessionindex.Record, string) {
				scanRoot := filepath.Join(root, "testdata", "handoff", "grok", dir)
				result, err := sessionindex.NewGrokSource(scanRoot).Scan(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Records) != 1 {
					t.Fatalf("grok fixture %s: records = %d, want 1", dir, len(result.Records))
				}
				rec := result.Records[0]
				return rec, filepath.Join(rec.SourcePath, "updates.jsonl")
			},
		}
	}
	return []partialFinalRecordRouteCase{
		claudeCase("macos", "partial-final-record", "-Users-fixture-user-code-demo", `/Users/fixture-user/code/demo`),
		claudeCase("windows", "partial-final-record-windows", "C--Users-fixture-user-code-demo", `C:\Users\fixture-user\code\demo`),
		codexCase("macos", "partial-final-record", "rollout-2026-08-01T14-00-00-00000000-0000-4000-8000-00000000ff01.jsonl", `/Users/fixture-user/code/demo`),
		codexCase("windows", "partial-final-record-windows", "rollout-2026-08-01T14-00-00-00000000-0000-4000-8000-00000000ff02.jsonl", `C:\Users\fixture-user\code\demo`),
		grokCase("macos", "partial-final-record"),
		grokCase("windows", "partial-final-record-windows"),
	}
}

// TestPlanPartialFinalRecordByteExactCrossCheck is the D4 acceptance bar
// itself, reproduced in-process: it runs the real Plan() pipeline (the same
// function internal/cli/handoff.go calls for `rein handoff --no-launch
// --json`) against each committed partial-final-record fixture from a
// throwaway git repository whose root is named "demo" — matching the last
// path segment every fixture in this tree records, on every OS — and checks
// the resulting capsule's raw_source.byte_offset/artifact_sha256 against a
// boundary independently recomputed from the fixture's own bytes in this
// test, not borrowed from the reader under test.
func TestPlanPartialFinalRecordByteExactCrossCheck(t *testing.T) {
	for _, tc := range partialFinalRecordRouteCases(t) {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec, boundaryFile := tc.buildRecord(t)
			raw, err := os.ReadFile(boundaryFile)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			wantOffset, wantDigest := expectedJSONLBoundary(t, raw)

			local := filepath.Join(t.TempDir(), "demo")
			if err := os.MkdirAll(local, 0o700); err != nil {
				t.Fatal(err)
			}
			initTestGitRepo(t, local)

			capture := &capturingVerifier{report: preflight.Report{
				SchemaVersion: preflight.SchemaVersion,
				Decision:      preflight.DecisionReady,
				Checks: []preflight.Check{{
					ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
					Provenance: workspace.ProvenanceCurrentObservation, Message: "source is fresh",
				}},
			}}

			opts := Options{
				ToAgent: tc.toAgent, Policy: PolicyBalanced,
				ReinstateHome: filepath.Join(t.TempDir(), "reinstate-home"),
				Verifier:      capture,
				Target:        &pipelineTarget{},
				WorkingDir:    local,
				ResolveSource: func(_ context.Context, in sessionindex.Record) (sessionindex.Record, bool, error) {
					return in, true, nil
				},
				SessionBusy: func(context.Context, string, processcheck.Target) (bool, bool, error) {
					return false, true, nil
				},
			}

			plan, err := Plan(context.Background(), rec, opts)
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })

			if !plan.Capsule.RawSource.Partial {
				t.Fatal("capsule.RawSource.Partial = false, want true")
			}
			if plan.Capsule.RawSource.ByteOffset != wantOffset {
				t.Fatalf("capsule.RawSource.ByteOffset = %d, want %d (independently computed)", plan.Capsule.RawSource.ByteOffset, wantOffset)
			}
			if plan.Capsule.RawSource.ArtifactSHA256 != wantDigest {
				t.Fatalf("capsule.RawSource.ArtifactSHA256 = %q, want %q (independently computed)", plan.Capsule.RawSource.ArtifactSHA256, wantDigest)
			}
			// Proof the remap onto the local git root actually ran, not just
			// that the reader's own numbers happened to match: the verifier
			// must have been asked about the local temp repo, never the
			// recorded fixture-user path.
			if strings.Contains(strings.ToLower(capture.workspace), "fixture-user") {
				t.Fatalf("Verify was not remapped off the recorded workspace: %q", capture.workspace)
			}
			if strings.Contains(strings.ToLower(plan.Capsule.Workspace.Root), "fixture-user") {
				t.Fatalf("capsule leaked the recorded fixture-user workspace: %q", plan.Capsule.Workspace.Root)
			}
		})
	}
}

// TestPlanPartialFinalRecordRefusalIsOSShapeIndependent is the direct
// counter-evidence to the claim these fixtures' first cut shipped (in
// testdata/handoff/claude/partial-final-record-windows/README.md and this
// workstream's changelog bullet): that a macOS-shaped recorded workspace
// blocks `rein handoff --no-launch --json` on native Windows in a way a
// Windows-shaped one does not. It refuses identically for a matched OS
// shape too, gated purely on whether the operator's git-root directory
// shares the recorded workspace's last path segment
// (workspace.go's sameProjectLeaf) — never on isForeignOSPath.
func TestPlanPartialFinalRecordRefusalIsOSShapeIndependent(t *testing.T) {
	for _, tc := range partialFinalRecordRouteCases(t) {
		if strings.HasPrefix(tc.name, "grok-") {
			// Grok's Plan() route is exercised by the byte-exact cross-check
			// test above; the refusal mechanism under test here is agent-
			// agnostic (it runs before any reader-specific behaviour), so
			// claude and codex already cover both recorded-workspace shapes.
			continue
		}
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec, _ := tc.buildRecord(t)

			other := filepath.Join(t.TempDir(), "not-demo")
			if err := os.MkdirAll(other, 0o700); err != nil {
				t.Fatal(err)
			}
			initTestGitRepo(t, other)

			opts := Options{
				ToAgent: tc.toAgent, Policy: PolicyBalanced,
				ReinstateHome: filepath.Join(t.TempDir(), "reinstate-home"),
				Verifier:      &capturingVerifier{report: preflight.Report{Decision: preflight.DecisionReady}},
				Target:        &pipelineTarget{},
				WorkingDir:    other,
				ResolveSource: func(_ context.Context, in sessionindex.Record) (sessionindex.Record, bool, error) {
					return in, true, nil
				},
				SessionBusy: func(context.Context, string, processcheck.Target) (bool, bool, error) {
					return false, true, nil
				},
			}

			_, err := Plan(context.Background(), rec, opts)
			assertPipelineCode(t, err, exitcode.Compatibility)
			if !strings.Contains(err.Error(), "different repository") {
				t.Fatalf("error = %v, want a different-repository refusal", err)
			}
		})
	}
}
