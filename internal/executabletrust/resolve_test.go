package executabletrust

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveSkipsWorkspaceOwnedExecutable(t *testing.T) {
	workspace := t.TempDir()
	unsafeDirectory := filepath.Join(workspace, "tools")
	if err := os.Mkdir(unsafeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(realExecutable)
	if err := os.WriteFile(filepath.Join(unsafeDirectory, name), []byte("workspace-owned executable"), 0o700); err != nil {
		t.Fatal(err)
	}

	resolution, err := Resolve(name, workspace, []string{
		"PATH=" + strings.Join([]string{unsafeDirectory, filepath.Dir(realExecutable)}, string(os.PathListSeparator)),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
	if strings.Contains(resolution.SearchPath, unsafeDirectory) {
		t.Fatalf("SearchPath retained workspace directory: %q", resolution.SearchPath)
	}
}

func TestResolveUsesOutermostRepositoryBoundary(t *testing.T) {
	repository := t.TempDir()
	workspace := filepath.Join(repository, "nested", "workspace")
	unsafeDirectory := filepath.Join(repository, "tools")
	for _, directory := range []string{
		filepath.Join(repository, ".git"),
		filepath.Join(workspace, ".git"),
		unsafeDirectory,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(realExecutable)
	if err := os.WriteFile(filepath.Join(unsafeDirectory, name), []byte("repository-owned executable"), 0o700); err != nil {
		t.Fatal(err)
	}

	resolution, err := Resolve(name, workspace, []string{
		"PATH=" + strings.Join([]string{unsafeDirectory, filepath.Dir(realExecutable)}, string(os.PathListSeparator)),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
	if strings.Contains(resolution.SearchPath, unsafeDirectory) {
		t.Fatalf("SearchPath retained enclosing-repository directory: %q", resolution.SearchPath)
	}
}

func TestWithinUsesSupportedPlatformCaseRules(t *testing.T) {
	boundary := filepath.Join(string(os.PathSeparator), "Users", "Example", "Repository")
	candidate := filepath.Join(string(os.PathSeparator), "users", "example", "repository", "tools")
	want := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	if got := within(candidate, boundary); got != want {
		t.Fatalf("within(case variant) = %t, want %t on %s", got, want, runtime.GOOS)
	}
}

func TestWithinFilesystemUsesDirectoryIdentity(t *testing.T) {
	boundary := t.TempDir()
	candidate := filepath.Join(boundary, "nested", "tools")
	if err := os.MkdirAll(candidate, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "boundary-alias")
	if err := os.Symlink(boundary, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	aliasCandidate := filepath.Join(alias, "nested", "tools")
	if within(aliasCandidate, boundary) {
		t.Fatal("lexical containment unexpectedly recognized the alias")
	}
	if !withinFilesystem(aliasCandidate, boundary) {
		t.Fatal("filesystem identity did not recognize the aliased boundary ancestor")
	}
}

func TestResolveRejectsCaseVariantWorkspaceSearchDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("supported case-insensitive platform regression")
	}
	workspace := filepath.Join(t.TempDir(), "Workspace")
	unsafeDirectory := filepath.Join(workspace, "Tools")
	if err := os.MkdirAll(unsafeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	caseVariant := filepath.Join(workspace, "tools")
	if _, err := os.Stat(caseVariant); err != nil {
		t.Skipf("test filesystem is case-sensitive: %v", err)
	}

	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(realExecutable)
	if err := os.WriteFile(filepath.Join(unsafeDirectory, name), []byte("case-variant workspace executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	resolution, err := Resolve(name, workspace, []string{
		"PATH=" + strings.Join([]string{caseVariant, filepath.Dir(realExecutable)}, string(os.PathListSeparator)),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
	if strings.Contains(strings.ToLower(resolution.SearchPath), strings.ToLower(caseVariant)) {
		t.Fatalf("SearchPath retained case-variant workspace directory: %q", resolution.SearchPath)
	}
}

func TestResolveSkipsSearchDirectorySymlinkedIntoWorkspace(t *testing.T) {
	workspace := t.TempDir()
	unsafeDirectory := filepath.Join(workspace, "tools")
	if err := os.Mkdir(unsafeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedDirectory := filepath.Join(t.TempDir(), "workspace-tools")
	if err := os.Symlink(unsafeDirectory, linkedDirectory); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := Resolve(filepath.Base(realExecutable), workspace, []string{
		"PATH=" + strings.Join([]string{linkedDirectory, filepath.Dir(realExecutable)}, string(os.PathListSeparator)),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if strings.Contains(resolution.SearchPath, linkedDirectory) || strings.Contains(resolution.SearchPath, unsafeDirectory) {
		t.Fatalf("SearchPath retained workspace symlink: %q", resolution.SearchPath)
	}
}

func TestResolveSkipsExecutableSymlinkedIntoWorkspace(t *testing.T) {
	workspace := t.TempDir()
	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(realExecutable)
	workspaceExecutable := filepath.Join(workspace, name)
	if err := os.WriteFile(workspaceExecutable, []byte("workspace-owned executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	linkedDirectory := t.TempDir()
	if err := os.Symlink(workspaceExecutable, filepath.Join(linkedDirectory, name)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	resolution, err := Resolve(name, workspace, []string{
		"PATH=" + strings.Join([]string{linkedDirectory, filepath.Dir(realExecutable)}, string(os.PathListSeparator)),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
}

func TestResolveRejectsRelativeEmptyAndInvalidInputs(t *testing.T) {
	workspace := t.TempDir()
	for _, name := range []string{"", "../git", filepath.Join(workspace, "git")} {
		if _, err := Resolve(name, workspace, []string{"PATH=/usr/bin"}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Resolve(%q) error = %v", name, err)
		}
	}
	if _, err := Resolve("definitely-not-an-executable", workspace, []string{"PATH=relative" + string(os.PathListSeparator)}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("relative PATH error = %v", err)
	}
}

func TestWindowsPathExtensionsOrderAndDefaults(t *testing.T) {
	got := windowsPathExtensions([]string{"PATHEXT=.EXE;.CMD;.exe;.BAT"})
	want := []string{".EXE", ".CMD", ".BAT"}
	if len(got) != len(want) {
		t.Fatalf("extensions = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("extensions = %#v, want %#v", got, want)
		}
	}

	// Empty PATHEXT falls through to process env or the safe default list.
	defaults := windowsPathExtensions([]string{"PATHEXT="})
	if len(defaults) == 0 {
		t.Fatal("expected default Windows path extensions")
	}
	joined := strings.ToLower(strings.Join(defaults, ";"))
	for _, required := range []string{".exe", ".cmd"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("defaults %#v missing %q", defaults, required)
		}
	}
}

func TestExecutableCandidatesExpandPathExtOnlyOnWindows(t *testing.T) {
	directory := filepath.Join(string(os.PathSeparator), "trusted", "bin")
	environment := []string{"PATHEXT=.COM;.EXE;.CMD"}
	candidates := executableCandidates(directory, "codex", environment)
	if runtime.GOOS == "windows" {
		want := []string{
			filepath.Join(directory, "codex.COM"),
			filepath.Join(directory, "codex.EXE"),
			filepath.Join(directory, "codex.CMD"),
		}
		if len(candidates) != len(want) {
			t.Fatalf("candidates = %#v, want %#v", candidates, want)
		}
		for index := range want {
			if candidates[index] != want[index] {
				t.Fatalf("candidates = %#v, want %#v", candidates, want)
			}
		}
		// Names that already carry an extension stay exact.
		exact := executableCandidates(directory, "codex.exe", environment)
		if len(exact) != 1 || exact[0] != filepath.Join(directory, "codex.exe") {
			t.Fatalf("exact candidates = %#v", exact)
		}
		return
	}
	if len(candidates) != 1 || candidates[0] != filepath.Join(directory, "codex") {
		t.Fatalf("non-Windows candidates = %#v", candidates)
	}
}

func TestResolveExtensionlessWindowsVendorName(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PATHEXT regression")
	}
	workspace := t.TempDir()
	trusted := t.TempDir()
	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	// RC1 failure shape: installed tool is codex.exe while callers look up "codex".
	installed := filepath.Join(trusted, "codex.exe")
	if err := os.WriteFile(installed, payload, 0o700); err != nil {
		t.Fatal(err)
	}

	resolution, err := Resolve("codex", workspace, []string{
		"PATH=" + trusted,
		"PATHEXT=.COM;.EXE;.BAT;.CMD",
	})
	if err != nil {
		t.Fatalf("Resolve(codex): %v", err)
	}
	want, err := filepath.EvalSymlinks(installed)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
}

func TestResolveWindowsCmdShimAndSkipsWorkspaceOwnedExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PATHEXT regression")
	}
	workspace := t.TempDir()
	unsafe := filepath.Join(workspace, "tools")
	trusted := t.TempDir()
	if err := os.MkdirAll(unsafe, 0o700); err != nil {
		t.Fatal(err)
	}
	realExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(realExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unsafe, "claude.exe"), payload, 0o700); err != nil {
		t.Fatal(err)
	}
	// .cmd is a common npm shim form on native Windows.
	installed := filepath.Join(trusted, "claude.cmd")
	if err := os.WriteFile(installed, []byte("@echo off\r\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	resolution, err := Resolve("claude", workspace, []string{
		"PATH=" + strings.Join([]string{unsafe, trusted}, string(os.PathListSeparator)),
		"PATHEXT=.COM;.EXE;.BAT;.CMD",
	})
	if err != nil {
		t.Fatalf("Resolve(claude): %v", err)
	}
	want, err := filepath.EvalSymlinks(installed)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Executable != want {
		t.Fatalf("Executable = %q, want %q", resolution.Executable, want)
	}
	if strings.Contains(strings.ToLower(resolution.SearchPath), strings.ToLower(unsafe)) {
		t.Fatalf("SearchPath retained workspace directory: %q", resolution.SearchPath)
	}
}
