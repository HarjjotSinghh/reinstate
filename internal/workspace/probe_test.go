package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type scriptedRunner struct {
	mu    sync.Mutex
	calls [][]string
	run   func(context.Context, string, []string) ([]byte, error)
}

func (runner *scriptedRunner) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	runner.mu.Lock()
	runner.calls = append(runner.calls, append([]string(nil), args...))
	runner.mu.Unlock()
	for _, arg := range args {
		switch arg {
		case "fetch", "pull", "push", "clone":
			return nil, errors.New("network-capable Git command attempted")
		}
	}
	return runner.run(ctx, dir, args)
}

func TestVerifyUsesOneSharedLocalProbeAndResolvesHeadRelation(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	expectedHead := strings.Repeat("a", 40)
	currentHead := strings.Repeat("b", 40)
	runner := &scriptedRunner{run: func(_ context.Context, dir string, args []string) ([]byte, error) {
		command := strings.Join(args, " ")
		switch {
		case command == "rev-parse --path-format=absolute --show-toplevel":
			return []byte(workspace + "\n"), nil
		case strings.Contains(command, "status --porcelain=v2"):
			return []byte("# branch.oid " + currentHead + "\x00# branch.head main\x00# branch.upstream origin/main\x00# branch.ab +1 -0\x00"), nil
		case command == "rev-parse --is-shallow-repository":
			return []byte("false\n"), nil
		case command == `config --null --get-regexp ^remote\..*\.url$`:
			return []byte("remote.origin.url\nhttps://token:secret@example.com/team/repo.git?key=hidden\x00"), nil
		case command == "rev-list --left-right --count "+expectedHead+"..."+currentHead:
			return []byte("0\t2\n"), nil
		default:
			return nil, errors.New("unexpected command: " + command + " in " + dir)
		}
	}}
	repositoryID, err := RepositoryIDFromRemote("git@example.com:team/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	verification, err := Verify(context.Background(), workspace, Expectation{
		Workspace: trustedString(workspace), RepositoryID: trustedString(repositoryID),
		Branch: trustedString("main"), Head: trustedString(expectedHead),
	}, ProbeOptions{Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Fingerprint.Git.ExpectedHeadRelation.Relation != RelationAhead ||
		verification.Fingerprint.Git.ExpectedHeadRelation.Ahead != 2 {
		t.Fatalf("head relation = %+v", verification.Fingerprint.Git.ExpectedHeadRelation)
	}
	if verification.Comparison.Decision != DecisionConfirmationRequired {
		t.Fatalf("decision = %s checks=%+v", verification.Comparison.Decision, verification.Comparison.Checks)
	}
	encoded, err := json.Marshal(verification)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"token", "secret", "hidden", "example.com", "team/repo"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("verification leaked %q: %s", forbidden, encoded)
		}
	}
	if len(runner.calls) != 5 {
		t.Fatalf("Git calls = %d, want 5: %v", len(runner.calls), runner.calls)
	}
}

func TestProbeMissingWorkspaceDoesNotRunGit(t *testing.T) {
	t.Parallel()
	runner := &scriptedRunner{run: func(context.Context, string, []string) ([]byte, error) {
		t.Fatal("Git ran for a missing workspace")
		return nil, nil
	}}
	result, err := Probe(context.Background(), filepath.Join(t.TempDir(), "missing"), ProbeOptions{Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	if result.Fingerprint.Workspace.Exists || len(result.Diagnostics) != 1 || result.Diagnostics[0].Status != StatusMissing {
		t.Fatalf("result = %+v", result)
	}
}

func TestProbeTimeoutIsBoundedAndBlocking(t *testing.T) {
	t.Parallel()
	runner := &scriptedRunner{run: func(ctx context.Context, _ string, _ []string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	started := time.Now()
	result, err := Probe(context.Background(), t.TempDir(), ProbeOptions{
		Runner: runner, Timeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("probe did not honor timeout: %v", elapsed)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Status != StatusError || result.Diagnostics[0].Severity != SeverityBlock {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestProbePropagatesCallerCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &scriptedRunner{run: func(ctx context.Context, _ string, _ []string) ([]byte, error) {
		return nil, ctx.Err()
	}}
	_, err := Probe(ctx, t.TempDir(), ProbeOptions{Runner: runner})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestProbeNonRepositoryIsAnHonestObservation(t *testing.T) {
	t.Parallel()
	runner := &scriptedRunner{run: func(context.Context, string, []string) ([]byte, error) {
		return nil, &CommandError{ExitCode: 128, NotRepository: true}
	}}
	result, err := Probe(context.Background(), t.TempDir(), ProbeOptions{Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Fingerprint.Git.Available || result.Fingerprint.Git.Repository || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestProbeNotDirectoryBlocksBeforeGit(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Probe(context.Background(), path, ProbeOptions{Runner: GitRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("Git ran for a non-directory")
		return nil, nil
	})})
	if err != nil || len(result.Diagnostics) != 1 || result.Diagnostics[0].Status != StatusMissing {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
