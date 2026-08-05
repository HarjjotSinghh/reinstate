package workspace

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBoundedBufferDrainsWithoutRetainingExcess(t *testing.T) {
	t.Parallel()
	var buffer boundedBuffer
	input := []byte(strings.Repeat("x", maxGitOutputBytes+10))
	written, err := buffer.Write(input)
	if err != nil || written != len(input) || buffer.Len() != maxGitOutputBytes || !buffer.exceeded {
		t.Fatalf("written=%d len=%d exceeded=%t err=%v", written, buffer.Len(), buffer.exceeded, err)
	}
}

func TestExecGitRunnerNeverRunsWorkspaceOwnedGit(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("system git unavailable")
	}
	repository := t.TempDir()
	command := exec.Command(realGit, "init", repository)
	command.Env = isolatedTestGitEnvironment(os.Environ(), t.TempDir())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	nested := filepath.Join(repository, "nested", "workspace")
	unsafeDirectory := filepath.Join(repository, "tools")
	// An inner marker is intentionally attacker-controlled. Executable trust
	// must still use the outer repository boundary rather than narrowing to it.
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(unsafeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	fakeGit := filepath.Join(unsafeDirectory, filepath.Base(realGit))
	fakeContents := []byte("workspace-owned fake git")
	if runtime.GOOS != "windows" {
		fakeContents = []byte("#!/bin/sh\ntouch \"$0.executed\"\nexit 99\n")
	}
	if err := os.WriteFile(fakeGit, fakeContents, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", strings.Join([]string{unsafeDirectory, filepath.Dir(realGit)}, string(os.PathListSeparator)))

	output, err := (ExecGitRunner{}).Run(t.Context(), nested, "rev-parse", "--path-format=absolute", "--show-toplevel")
	if err != nil {
		t.Fatalf("Run selected workspace-owned git instead of system git: %v", err)
	}
	canonicalRepository, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(output)) != canonicalRepository {
		t.Fatalf("repository root = %q, want %q", strings.TrimSpace(string(output)), canonicalRepository)
	}
	if _, err := os.Stat(fakeGit + ".executed"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace-owned git was executed: %v", err)
	}
}

func TestRepositoryDiscoveryRejectsOversizedAndSymlinkGitMarkers(t *testing.T) {
	repository := initTestRepository(t)
	t.Run("oversized regular pointer", func(t *testing.T) {
		nested := filepath.Join(repository, "oversized", "workspace")
		if err := os.MkdirAll(nested, 0o700); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(nested, ".git")
		handle, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if err := handle.Truncate(1 << 30); err != nil {
			_ = handle.Close()
			t.Fatal(err)
		}
		if err := handle.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := parseGitDirectoryFile(nested, marker); !errors.Is(err, ErrNotRepository) {
			t.Fatalf("oversized gitfile error = %v", err)
		}
		output, err := (ExecGitRunner{}).Run(t.Context(), nested, gitRootArgs...)
		if err != nil {
			t.Fatal(err)
		}
		if root := strings.TrimSpace(string(output)); root != repository {
			t.Fatalf("oversized nested marker changed root to %q", root)
		}
	})
	t.Run("directory symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation is not reliably available on Windows CI")
		}
		nested := filepath.Join(repository, "symlink", "workspace")
		if err := os.MkdirAll(nested, 0o700); err != nil {
			t.Fatal(err)
		}
		outside := initTestRepository(t)
		if err := os.Symlink(filepath.Join(outside, ".git"), filepath.Join(nested, ".git")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		output, err := (ExecGitRunner{}).Run(t.Context(), nested, gitRootArgs...)
		if err != nil {
			t.Fatal(err)
		}
		if root := strings.TrimSpace(string(output)); root != repository {
			t.Fatalf("symlink nested marker changed root to %q", root)
		}
	})
}

func TestExecGitRunnerStatusMarksCommandBearingRepositoryConfigUncertainWithoutExecution(t *testing.T) {
	repository := initTestRepository(t)
	marker := filepath.Join(t.TempDir(), "configured-command-executed")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := strconv.Quote(executable) + " -test.run=^TestGitConfiguredCommandHelper$"
	t.Setenv("REINSTATE_GIT_CONFIG_HELPER_MARKER", marker)
	t.Setenv("GIT_EXTERNAL_DIFF", command)
	for _, setting := range [][2]string{
		{"core.fsmonitor", command},
		{"diff.external", command},
		{"diff.evil.command", command},
		{"diff.evil.textconv", command},
		{"filter.evil.clean", command},
		{"filter.evil.process", command},
		{"filter.evil.required", "true"},
	} {
		runTestGit(t, repository, "config", setting[0], setting[1])
	}
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("unsafe-config status/error = %+v / %v", status, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repository-configured command executed: %v", err)
	}
}

func TestExecGitRunnerStatusMarksResolvedFilterAttributesUncertain(t *testing.T) {
	for _, location := range []string{"worktree", "info"} {
		t.Run(location, func(t *testing.T) {
			repository := initTestRepository(t)
			attributePath := filepath.Join(repository, ".gitattributes")
			if location == "info" {
				attributePath = filepath.Join(repository, ".git", "info", "attributes")
			}
			if err := os.WriteFile(attributePath, []byte("tracked.txt filter=controlled\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
			if err != nil {
				t.Fatal(err)
			}
			status, err := parsePorcelainV2(output)
			if err != nil || !status.workingTree.Uncertain {
				t.Fatalf("filter-attribute status/error = %+v / %v", status, err)
			}
		})
	}
}

func TestExecGitRunnerStatusPreservesStateSemantics(t *testing.T) {
	repository := initTestRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "staged.txt"), []byte("staged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "untracked.txt"), []byte("untracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatal(err)
	}
	if status.workingTree.State != WorkingTreeModified || status.workingTree.Staged != 1 ||
		status.workingTree.Unstaged != 1 || status.workingTree.Untracked != 1 {
		t.Fatalf("working tree = %+v", status.workingTree)
	}
}

func TestExecGitRunnerStatusConservativelyReportsTouchedIdenticalFile(t *testing.T) {
	repository := initTestRepository(t)
	tracked := filepath.Join(repository, "tracked.txt")
	contents, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracked, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(tracked, future, future); err != nil {
		t.Fatal(err)
	}

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatal(err)
	}
	if status.workingTree.State != WorkingTreeModified || status.workingTree.Unstaged != 1 {
		t.Fatalf("touched-identical working tree = %+v", status.workingTree)
	}
}

func TestExecGitRunnerStatusHandlesUnbornDetachedAndUpstream(t *testing.T) {
	t.Run("unborn", func(t *testing.T) {
		repository := t.TempDir()
		runTestGit(t, repository, "init")
		if err := os.WriteFile(filepath.Join(repository, "staged.txt"), []byte("staged\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "staged.txt")
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || !status.unborn || status.detached || status.workingTree.Staged != 1 {
			t.Fatalf("unborn status/error = %+v / %v", status, err)
		}
	})
	t.Run("detached", func(t *testing.T) {
		repository := initTestRepository(t)
		runTestGit(t, repository, "checkout", "--detach")
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || !status.detached || status.unborn {
			t.Fatalf("detached status/error = %+v / %v", status, err)
		}
	})
	t.Run("upstream", func(t *testing.T) {
		repository := initTestRepository(t)
		branch := strings.TrimSpace(runTestGit(t, repository, "branch", "--show-current"))
		runTestGit(t, repository, "update-ref", "refs/remotes/origin/"+branch, "HEAD")
		runTestGit(t, repository, "branch", "--set-upstream-to=origin/"+branch)
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || !status.upstreamSet || status.relation.Relation != RelationEqual {
			t.Fatalf("upstream status/error = %+v / %v", status, err)
		}
		if err := os.WriteFile(filepath.Join(repository, "ahead.txt"), []byte("ahead\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "ahead.txt")
		runTestGit(t, repository, "commit", "-m", "ahead")
		output, err = (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err = parsePorcelainV2(output)
		if err != nil || status.relation.Relation != RelationAhead || status.relation.Ahead != 1 || status.relation.Behind != 0 {
			t.Fatalf("ahead status/error = %+v / %v", status, err)
		}
	})
	t.Run("behind", func(t *testing.T) {
		repository := initTestRepository(t)
		branch := strings.TrimSpace(runTestGit(t, repository, "branch", "--show-current"))
		runTestGit(t, repository, "update-ref", "refs/remotes/origin/"+branch, "HEAD")
		runTestGit(t, repository, "branch", "--set-upstream-to=origin/"+branch)
		runTestGit(t, repository, "checkout", "-b", "remote-ahead")
		if err := os.WriteFile(filepath.Join(repository, "remote.txt"), []byte("remote\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "remote.txt")
		runTestGit(t, repository, "commit", "-m", "remote ahead")
		runTestGit(t, repository, "update-ref", "refs/remotes/origin/"+branch, "HEAD")
		runTestGit(t, repository, "checkout", branch)
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || status.relation.Relation != RelationBehind || status.relation.Ahead != 0 || status.relation.Behind != 1 {
			t.Fatalf("behind status/error = %+v / %v", status, err)
		}
	})
}

func TestExecGitRunnerStatusHandlesConflictRenameAndAssumeUnchanged(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		repository := initTestRepository(t)
		baseBranch := strings.TrimSpace(runTestGit(t, repository, "branch", "--show-current"))
		runTestGit(t, repository, "checkout", "-b", "other")
		if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("other branch\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "tracked.txt")
		runTestGit(t, repository, "commit", "-m", "other")
		runTestGit(t, repository, "checkout", baseBranch)
		if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("base branch\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "tracked.txt")
		runTestGit(t, repository, "commit", "-m", "base")
		merge := exec.Command("git", "merge", "other")
		merge.Dir = repository
		merge.Env = isolatedTestGitEnvironment(os.Environ(), t.TempDir())
		if err := merge.Run(); err == nil {
			t.Fatal("merge unexpectedly succeeded without conflict")
		}
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || status.workingTree.Conflicted != 1 {
			t.Fatalf("conflict status/error = %+v / %v", status, err)
		}
		repeated, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		repeatedStatus, err := parsePorcelainV2(repeated)
		if err != nil || repeatedStatus.workingTree.Conflicted != 1 ||
			repeatedStatus.workingTree.Digest != status.workingTree.Digest {
			t.Fatalf("repeated conflict status/error = %+v / %v", repeatedStatus, err)
		}
	})
	t.Run("rename is conservative add delete", func(t *testing.T) {
		repository := initTestRepository(t)
		if err := os.Rename(filepath.Join(repository, "tracked.txt"), filepath.Join(repository, "renamed.txt")); err != nil {
			t.Fatal(err)
		}
		runTestGit(t, repository, "add", "-A")
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || status.workingTree.Staged != 2 {
			t.Fatalf("rename status/error = %+v / %v", status, err)
		}
	})
	t.Run("assume unchanged is not false clean", func(t *testing.T) {
		repository := initTestRepository(t)
		runTestGit(t, repository, "update-index", "--assume-unchanged", "tracked.txt")
		baselineOutput, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		baseline, err := parsePorcelainV2(baselineOutput)
		if err != nil || !baseline.workingTree.Uncertain {
			t.Fatalf("assume-unchanged baseline/error = %+v / %v", baseline, err)
		}
		if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("hidden change\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
		if err != nil {
			t.Fatal(err)
		}
		status, err := parsePorcelainV2(output)
		if err != nil || !status.workingTree.Uncertain {
			t.Fatalf("assume-unchanged status/error = %+v / %v", status, err)
		}
		comparison := compareWorkingTree(nil, &ExpectedString{
			Value: baseline.workingTree.Digest, Provenance: ProvenanceReinstatePrelaunchObserved,
		}, GitFingerprint{WorkingTree: status.workingTree})
		if comparison.Status == StatusMatch || comparison.Severity != SeverityWarning {
			t.Fatalf("hidden assume-unchanged comparison = %+v", comparison)
		}
	})
}

func TestExecGitRunnerStatusDoesNotInspectNestedSubmoduleCommands(t *testing.T) {
	child := initTestRepository(t)
	parent := initTestRepository(t)
	runTestGit(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", "--", child, "module")
	runTestGit(t, parent, "commit", "-m", "submodule")
	module := filepath.Join(parent, "module")
	baselineOutput, err := (ExecGitRunner{}).Run(t.Context(), parent, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := parsePorcelainV2(baselineOutput)
	if err != nil || !baseline.workingTree.Uncertain {
		t.Fatalf("submodule baseline/error = %+v / %v", baseline, err)
	}
	marker := filepath.Join(t.TempDir(), "nested-command-executed")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := strconv.Quote(executable) + " -test.run=^TestGitConfiguredCommandHelper$"
	t.Setenv("REINSTATE_GIT_CONFIG_HELPER_MARKER", marker)
	runTestGit(t, module, "config", "filter.evil.clean", command)
	runTestGit(t, module, "config", "filter.evil.required", "true")
	if err := os.WriteFile(filepath.Join(module, ".gitattributes"), []byte("tracked.txt filter=evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "tracked.txt"), []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := (ExecGitRunner{}).Run(t.Context(), parent, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("submodule status/error = %+v / %v", status, err)
	}
	comparison := compareWorkingTree(nil, &ExpectedString{
		Value: baseline.workingTree.Digest, Provenance: ProvenanceReinstatePrelaunchObserved,
	}, GitFingerprint{WorkingTree: status.workingTree})
	if comparison.Status == StatusMatch || comparison.Severity != SeverityWarning {
		t.Fatalf("hidden submodule comparison = %+v", comparison)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested submodule command executed: %v", err)
	}
}

func TestExecGitRunnerStatusDisablesGlobalConfigAndAttributes(t *testing.T) {
	repository := initTestRepository(t)
	marker := filepath.Join(t.TempDir(), "global-command-executed")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := strconv.Quote(executable) + " -test.run=^TestGitConfiguredCommandHelper$"
	attributes := filepath.Join(t.TempDir(), "attributes")
	if err := os.WriteFile(attributes, []byte("tracked.txt filter=evil diff=evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(t.TempDir(), "gitconfig")
	configContents := "[core]\n\tattributesFile = " + strconv.Quote(attributes) + "\n\tfsmonitor = " + strconv.Quote(command) +
		"\n[filter \"evil\"]\n\tclean = " + strconv.Quote(command) + "\n[diff]\n\texternal = " + strconv.Quote(command) + "\n"
	if err := os.WriteFile(config, []byte(configContents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_GIT_CONFIG_HELPER_MARKER", marker)
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_ATTR_GLOBAL", attributes)
	t.Setenv("GIT_ATTR_SYSTEM", attributes)

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || status.workingTree.State != WorkingTreeClean {
		t.Fatalf("global-config status/error = %+v / %v", status, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("global command executed: %v", err)
	}
}

func TestExecGitRunnerStatusMarksConcurrentIndexMutationUncertain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper fixture")
	}
	repository := initTestRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("concurrent change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	wrapperDirectory := t.TempDir()
	wrapper := filepath.Join(wrapperDirectory, "git")
	contents := `#!/bin/sh
case " $* " in
  *" diff --cached "*)
    "$REINSTATE_REAL_GIT" "$@"
    result=$?
    "$REINSTATE_REAL_GIT" add -- tracked.txt
    exit $result
    ;;
esac
exec "$REINSTATE_REAL_GIT" "$@"
`
	if err := os.WriteFile(wrapper, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_REAL_GIT", realGit)
	t.Setenv("PATH", strings.Join([]string{wrapperDirectory, filepath.Dir(realGit)}, string(os.PathListSeparator)))

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("concurrent-index status/error = %+v / %v", status, err)
	}
}

func TestExecGitRunnerPinsRepositoryWorktree(t *testing.T) {
	repository := initTestRepository(t)
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "tracked.txt"), []byte("external content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "external-only.txt"), []byte("must not be observed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "config", "core.worktree", external)

	rootOutput, err := (ExecGitRunner{}).Run(t.Context(), filepath.Join(repository, ".git", "objects"), gitRootArgs...)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRepository, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if root := strings.TrimSpace(string(rootOutput)); root != canonicalRepository {
		t.Fatalf("repository root = %q, want %q", root, canonicalRepository)
	}
	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatal(err)
	}
	if status.workingTree.State != WorkingTreeClean || status.workingTree.Unstaged != 0 ||
		status.workingTree.Untracked != 0 || !status.workingTree.Uncertain {
		t.Fatalf("core.worktree redirected observation: %+v", status.workingTree)
	}
}

func TestExecGitRunnerStatusDoesNotTrustCoreIgnoreStat(t *testing.T) {
	repository := initTestRepository(t)
	runTestGit(t, repository, "config", "core.ignoreStat", "true")
	runTestGit(t, repository, "update-index", "--refresh")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("hidden by ignoreStat\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("core.ignoreStat status/error = %+v / %v", status, err)
	}
	comparison := compareWorkingTree(nil, &ExpectedString{
		Value: status.workingTree.Digest, Provenance: ProvenanceReinstatePrelaunchObserved,
	}, GitFingerprint{WorkingTree: status.workingTree})
	if comparison.Status == StatusMatch || comparison.Severity != SeverityWarning {
		t.Fatalf("core.ignoreStat comparison = %+v", comparison)
	}
}

func TestExecGitRunnerStatusMarksConcurrentBranchRenameUncertain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper fixture")
	}
	repository := initTestRepository(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	wrapperDirectory := t.TempDir()
	wrapper := filepath.Join(wrapperDirectory, "git")
	marker := filepath.Join(wrapperDirectory, "renamed")
	contents := `#!/bin/sh
case " $* " in
  *" symbolic-ref --quiet --short HEAD "*)
    "$REINSTATE_REAL_GIT" "$@"
    result=$?
    if [ ! -e "$REINSTATE_RACE_MARKER" ]; then
      : > "$REINSTATE_RACE_MARKER"
      "$REINSTATE_REAL_GIT" branch -m independently-renamed
    fi
    exit $result
    ;;
esac
exec "$REINSTATE_REAL_GIT" "$@"
`
	if err := os.WriteFile(wrapper, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_REAL_GIT", realGit)
	t.Setenv("REINSTATE_RACE_MARKER", marker)
	t.Setenv("PATH", strings.Join([]string{wrapperDirectory, filepath.Dir(realGit)}, string(os.PathListSeparator)))

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("concurrent-branch status/error = %+v / %v", status, err)
	}
}

func TestExecGitRunnerStatusMarksLateTrackedMutationUncertain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper fixture")
	}
	repository := initTestRepository(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	wrapperDirectory := t.TempDir()
	wrapper := filepath.Join(wrapperDirectory, "git")
	counter := filepath.Join(wrapperDirectory, "symbolic-count")
	tracked := filepath.Join(repository, "tracked.txt")
	contents := `#!/bin/sh
case " $* " in
  *" symbolic-ref --quiet --short HEAD "*)
    "$REINSTATE_REAL_GIT" "$@"
    result=$?
    if [ -e "$REINSTATE_RACE_MARKER" ]; then
      printf 'late mutation\n' > "$REINSTATE_TRACKED_PATH"
    else
      : > "$REINSTATE_RACE_MARKER"
    fi
    exit $result
    ;;
esac
exec "$REINSTATE_REAL_GIT" "$@"
`
	if err := os.WriteFile(wrapper, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_REAL_GIT", realGit)
	t.Setenv("REINSTATE_RACE_MARKER", counter)
	t.Setenv("REINSTATE_TRACKED_PATH", tracked)
	t.Setenv("PATH", strings.Join([]string{wrapperDirectory, filepath.Dir(realGit)}, string(os.PathListSeparator)))

	output, err := (ExecGitRunner{}).Run(t.Context(), repository, gitStatusArgs...)
	if err != nil {
		t.Fatal(err)
	}
	status, err := parsePorcelainV2(output)
	if err != nil || !status.workingTree.Uncertain {
		t.Fatalf("late-mutation status/error = %+v / %v", status, err)
	}
}

func TestGitConfiguredCommandHelper(t *testing.T) {
	marker := os.Getenv("REINSTATE_GIT_CONFIG_HELPER_MARKER")
	if marker == "" {
		return
	}
	if err := os.WriteFile(marker, []byte("executed"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExecGitRunnerRejectsCommandsOutsideReadOnlyAllowlist(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"fetch"},
		{"pull", "origin"},
		{"clone", "https://example.invalid/repo"},
		{"-c"},
		{"config", "--global", "user.name", "attacker"},
		{"config", "--unset", "remote.origin.url"},
		{"status", "--porcelain=v1"},
		{"rev-list", "--objects", "HEAD"},
		{"config", "--null", "--get-regexp", `^remote\..*\.url$`},
	} {
		if allowedGitProbe(args) {
			t.Fatalf("unsafe command passed allowlist: %v", args)
		}
	}
	if !allowedGitProbe([]string{"config", "--local", "--no-includes", "--null", "--get-regexp", `^remote\..*\.url$`}) {
		t.Fatal("fixed local no-include repository identity command was rejected")
	}
	if _, err := (ExecGitRunner{}).Run(t.Context(), t.TempDir(), "--definitely-not-an-allowed-probe"); !errors.Is(err, ErrUnsafeGitCommand) {
		t.Fatalf("ExecGitRunner rejection error = %v", err)
	}
}

func TestSafeGitEnvironmentRemovesRepositoryAndConfigInjection(t *testing.T) {
	t.Parallel()
	environment := safeGitEnvironment([]string{
		"PATH=/bin", "GIT_DIR=/attacker", "GIT_WORK_TREE=/wrong",
		"GIT_CONFIG_COUNT=2", "GIT_CONFIG_KEY_1=core.fsmonitor",
		"GIT_CONFIG_VALUE_1=malicious", "GIT_CONFIG_PARAMETERS='alias.status=!steal'",
		"GIT_TRACE=/tmp/private-trace", "GIT_CEILING_DIRECTORIES=/wrong", "LC_ALL=host",
	})
	joined := strings.Join(environment, "\n")
	for _, forbidden := range []string{"/attacker", "/wrong", "malicious", "steal", "private-trace", "LC_ALL=host"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("unsafe environment retained %q: %s", forbidden, joined)
		}
	}
	for _, required := range []string{"PATH=/bin", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1", "LC_ALL=C"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("safe environment missing %q: %s", required, joined)
		}
	}
}

func TestSafeGitEnvironmentWithPathReplacesInheritedPath(t *testing.T) {
	t.Parallel()
	environment := safeGitEnvironmentWithPath([]string{"PATH=/unsafe", "Path=/also-unsafe", "HOME=/safe"}, "/trusted")
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "/unsafe") || strings.Count(strings.ToUpper(joined), "PATH=") != 1 || !strings.Contains(joined, "PATH=/trusted") {
		t.Fatalf("PATH was not replaced safely: %s", joined)
	}
}
