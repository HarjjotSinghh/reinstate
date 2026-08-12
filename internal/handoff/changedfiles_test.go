package handoff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// A handoff over a dirty tree must name the dirty files. The whole pipeline is
// exercised against a real repository — probe, bind, checkpoint, capsule,
// bootstrap — because the defect this covers lived in the wiring between those
// steps, not inside any one of them: every stage was individually correct while
// the destination was still told "Changed files: (none)".
func TestPlanListsLiveChangedFilesFromRealRepository(t *testing.T) {
	repository := initChangedFilesRepository(t)
	runChangedFilesGit(t, repository, "checkout", "-b", "work")

	// One of each kind porcelain reports separately.
	writeChangedFilesFile(t, filepath.Join(repository, "tracked.txt"), "edited")
	writeChangedFilesFile(t, filepath.Join(repository, "staged.go"), "package demo\n")
	runChangedFilesGit(t, repository, "add", "staged.go")
	if err := os.MkdirAll(filepath.Join(repository, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeChangedFilesFile(t, filepath.Join(repository, "nested", "untracked.md"), "notes")

	plan := planAgainstRepository(t, repository, nil)

	want := []string{
		"${REPO:github.com/example/repo}/nested/untracked.md",
		"${REPO:github.com/example/repo}/staged.go",
		"${REPO:github.com/example/repo}/tracked.txt",
	}
	if !equalStrings(plan.Capsule.Workspace.ChangedFiles, want) {
		t.Fatalf("capsule workspace changed_files = %#v, want %#v",
			plan.Capsule.Workspace.ChangedFiles, want)
	}
	if !plan.Capsule.Workspace.Dirty {
		t.Fatal("dirty tree bound as clean")
	}
	if !equalStrings(plan.Capsule.Task.ChangedFiles.Items, want) {
		t.Fatalf("task changed_files = %#v, want %#v", plan.Capsule.Task.ChangedFiles.Items, want)
	}
	if plan.Capsule.Task.ChangedFiles.Portability != capsule.PortabilityExact {
		t.Fatalf("changed_files portability = %q, want exact",
			plan.Capsule.Task.ChangedFiles.Portability)
	}

	bootstrap := string(plan.Destination.Bootstrap)
	section := changedFilesSection(t, bootstrap)
	for _, item := range want {
		if !strings.Contains(section, "- "+item) {
			t.Fatalf("bootstrap changed-files section missing %q:\n%s", item, section)
		}
	}
	if strings.Contains(section, "(none)") {
		t.Fatalf("bootstrap claimed a clean tree over a dirty one:\n%s", section)
	}
	if !strings.Contains(string(plan.Artifacts.ProjectionMD), "- "+want[0]) {
		t.Fatalf("projection.md missing changed files:\n%s", plan.Artifacts.ProjectionMD)
	}

	// Tokenization is the contract, not decoration: the capsule may hold no
	// absolute path, and the operator's temporary directory is one.
	raw, err := capsule.CanonicalBytes(plan.Capsule)
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	if strings.Contains(string(raw), repository) {
		t.Fatalf("absolute repository path reached the capsule: %s", raw)
	}
	for _, item := range plan.Capsule.Workspace.ChangedFiles {
		if capsule.AbsolutePathForbidden(item) {
			t.Fatalf("absolute path in changed_files: %q", item)
		}
	}
}

// The mirror case. Silence is only honest when the tree is actually clean, and
// a clean tree must not acquire files from the transcript.
func TestPlanCleanRepositoryClaimsNoChangedFiles(t *testing.T) {
	repository := initChangedFilesRepository(t)

	plan := planAgainstRepository(t, repository, nil)

	if len(plan.Capsule.Workspace.ChangedFiles) != 0 ||
		plan.Capsule.Workspace.ChangedFilesOmitted != 0 ||
		len(plan.Capsule.Task.ChangedFiles.Items) != 0 {
		t.Fatalf("clean repository produced changed files: %+v", plan.Capsule.Workspace)
	}
	if plan.Capsule.Workspace.Dirty {
		t.Fatal("clean tree bound as dirty")
	}
	if section := changedFilesSection(t, string(plan.Destination.Bootstrap)); !strings.Contains(section, "(none)") {
		t.Fatalf("clean tree did not render (none):\n%s", section)
	}
}

// Past the cap the list stops growing, and the destination is told by how much.
func TestPlanReportsChangedFilesBeyondTheCap(t *testing.T) {
	repository := initChangedFilesRepository(t)
	const extra = 12
	for index := 0; index < workspace.MaxChangedPaths+extra; index++ {
		writeChangedFilesFile(t,
			filepath.Join(repository, "gen-"+strconv.Itoa(1000+index)+".txt"), "x")
	}

	plan := planAgainstRepository(t, repository, nil)

	if len(plan.Capsule.Workspace.ChangedFiles) != workspace.MaxChangedPaths {
		t.Fatalf("changed_files = %d entries, want the %d cap",
			len(plan.Capsule.Workspace.ChangedFiles), workspace.MaxChangedPaths)
	}
	if plan.Capsule.Workspace.ChangedFilesOmitted != extra {
		t.Fatalf("changed_files_omitted = %d, want %d",
			plan.Capsule.Workspace.ChangedFilesOmitted, extra)
	}
	section := changedFilesSection(t, string(plan.Destination.Bootstrap))
	if !strings.Contains(section, "not listed)") {
		t.Fatalf("truncation was silent in the bootstrap:\n%s", section)
	}
	if len(plan.Destination.Bootstrap) > BootstrapMaxBytes {
		t.Fatalf("bootstrap len = %d exceeds BootstrapMaxBytes=%d",
			len(plan.Destination.Bootstrap), BootstrapMaxBytes)
	}
	// A capped list is incomplete evidence, so it may not be used to call a
	// transcript claim contradicted.
	if plan.Capsule.Task.FilesTouchedPerTranscript.Reason == reasonEvidenceConflictsWithWorkspace {
		t.Fatal("a truncated changed-file list was used as counter-evidence")
	}
}

// planAgainstRepository runs Plan with the preflight workspace report produced
// by a real probe of the given repository.
func planAgainstRepository(t *testing.T, repository string, changedOverride []string) PlanResult {
	t.Helper()

	rec, _, verifier, _, opts := pipelineFixture(t)
	rec.Workspace = repository
	rec.Project = "github.com/example/repo"
	opts.ChangedFiles = changedOverride

	probe, err := workspace.Probe(context.Background(), repository,
		workspace.ProbeOptions{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("workspace.Probe: %v", err)
	}
	if len(probe.Diagnostics) != 0 {
		t.Fatalf("probe diagnostics: %+v", probe.Diagnostics)
	}
	verifier.report.Workspace = probe.Fingerprint

	plan, err := Plan(context.Background(), rec, opts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(plan.TempDir) })
	return plan
}

func changedFilesSection(t *testing.T, rendered string) string {
	t.Helper()
	const heading = "## Changed files\n"
	start := strings.Index(rendered, heading)
	if start < 0 {
		t.Fatalf("no changed-files section in:\n%s", rendered)
	}
	rest := rendered[start+len(heading):]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		return rest[:end]
	}
	return rest
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func initChangedFilesRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	repository := filepath.Clean(t.TempDir())
	if physical, err := filepath.EvalSymlinks(repository); err == nil {
		repository = physical
	}
	runChangedFilesGit(t, repository, "init")
	runChangedFilesGit(t, repository, "config", "user.name", "Reinstate Test")
	runChangedFilesGit(t, repository, "config", "user.email", "test@invalid.example")
	writeChangedFilesFile(t, filepath.Join(repository, "tracked.txt"), "first")
	runChangedFilesGit(t, repository, "add", "tracked.txt")
	runChangedFilesGit(t, repository, "commit", "-m", "first")
	return repository
}

func writeChangedFilesFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// runChangedFilesGit runs Git against a throwaway configuration so the
// developer's own Git identity, hooks, and templates never affect the fixture.
func runChangedFilesGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	configRoot := t.TempDir()
	command.Env = append(os.Environ(),
		"HOME="+configRoot,
		"XDG_CONFIG_HOME="+configRoot,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
