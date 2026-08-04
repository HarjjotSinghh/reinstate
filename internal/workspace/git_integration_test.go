package workspace

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestProbeRealRepositoryAndWorkingTreePrivacy(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	repository := initTestRepository(t)
	nested := filepath.Join(repository, "nested", "directory")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}

	clean, err := Probe(context.Background(), nested, ProbeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !clean.Fingerprint.Git.Repository || clean.Fingerprint.Git.Root != repository ||
		clean.Fingerprint.Git.WorkingTree.State != WorkingTreeClean ||
		clean.Fingerprint.Git.RepositoryIDSource != "remote" {
		t.Fatalf("clean fingerprint = %+v diagnostics=%+v", clean.Fingerprint, clean.Diagnostics)
	}
	cleanDigest := clean.Fingerprint.Git.WorkingTree.Digest

	privateName := "PRIVATE-CUSTOMER-secret-file.txt"
	if err := os.WriteFile(filepath.Join(repository, privateName), []byte("controlled"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirty, err := Probe(context.Background(), repository, ProbeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if dirty.Fingerprint.Git.WorkingTree.State != WorkingTreeModified ||
		dirty.Fingerprint.Git.WorkingTree.Untracked != 1 ||
		dirty.Fingerprint.Git.WorkingTree.Digest == cleanDigest {
		t.Fatalf("dirty fingerprint = %+v", dirty.Fingerprint.Git.WorkingTree)
	}
	encoded, err := json.Marshal(dirty)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), privateName) || strings.Contains(string(encoded), "example.com/team/private") {
		t.Fatalf("probe exposed private Git metadata: %s", encoded)
	}
}

func TestVerifyRealRepositoryComputesExpectedHeadRelation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	repository := initTestRepository(t)
	expected := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repository, "second.txt"), []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "add", "second.txt")
	runTestGit(t, repository, "commit", "-m", "second")

	verification, err := Verify(context.Background(), repository, Expectation{
		Head: trustedString(expected),
	}, ProbeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	relation := verification.Fingerprint.Git.ExpectedHeadRelation
	if relation.Relation != RelationAhead || relation.Ahead != 1 || relation.Behind != 0 || !relation.LocalOnly {
		t.Fatalf("relation = %+v", relation)
	}
}

func TestProbeRealRepositoryIsConcurrencySafe(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	repository := initTestRepository(t)
	const workers = 8
	var wait sync.WaitGroup
	errorsFound := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := Probe(context.Background(), repository, ProbeOptions{})
			if err != nil {
				errorsFound <- err
				return
			}
			if !result.Fingerprint.Git.Repository || len(result.Diagnostics) != 0 {
				errorsFound <- &unexpectedProbeResult{result: result}
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func TestProbeCanonicalizesSymlinkedWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available on Windows CI")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	repository := initTestRepository(t)
	link := filepath.Join(t.TempDir(), "repository-link")
	if err := os.Symlink(repository, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result, err := Probe(context.Background(), link, ProbeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Fingerprint.Workspace.Path != repository || result.Fingerprint.Git.Root != repository {
		t.Fatalf("symlink fingerprint = %+v", result.Fingerprint)
	}
}

type unexpectedProbeResult struct {
	result ProbeResult
}

func (err *unexpectedProbeResult) Error() string {
	return "unexpected concurrent probe result"
}

func initTestRepository(t *testing.T) string {
	t.Helper()
	repository := filepath.Clean(t.TempDir())
	if physical, err := filepath.EvalSymlinks(repository); err == nil {
		repository = physical
	}
	runTestGit(t, repository, "init")
	runTestGit(t, repository, "config", "user.name", "Reinstate Test")
	runTestGit(t, repository, "config", "user.email", "test@invalid.example")
	runTestGit(t, repository, "remote", "add", "origin", "https://example.com/team/private.git")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "add", "tracked.txt")
	runTestGit(t, repository, "commit", "-m", "first")
	return repository
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = safeGitEnvironment(os.Environ())
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
