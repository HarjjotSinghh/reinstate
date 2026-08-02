package doctest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	markdownLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\n]+)\)`)
	workflowUsePattern  = regexp.MustCompile(`(?m)^\s*uses:\s*(\S+)`)
	immutableActionRef  = regexp.MustCompile(`^[^@\s]+@[0-9a-f]{40}$`)
)

func TestWorkflowActionsArePinnedAndPermissionsAreExplicit(t *testing.T) {
	workflowDir := filepath.Join(repoRoot(t), ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		path := filepath.Join(workflowDir, entry.Name())
		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		body := string(bodyBytes)
		if !regexp.MustCompile(`(?m)^permissions:\s*(?:$|[^{])`).MatchString(body) {
			t.Errorf("%s must declare workflow-level permissions", entry.Name())
		}
		if regexp.MustCompile(`(?m)^permissions:\s*write-all\s*$`).MatchString(body) {
			t.Errorf("%s grants write-all permissions", entry.Name())
		}
		for _, match := range workflowUsePattern.FindAllStringSubmatch(body, -1) {
			ref := strings.TrimSpace(match[1])
			if strings.HasPrefix(ref, "./") {
				continue
			}
			if !immutableActionRef.MatchString(ref) {
				t.Errorf("%s uses mutable action reference %q", entry.Name(), ref)
			}
		}
	}
}

func TestWebsitePathToRegexpIsPatched(t *testing.T) {
	type lockfile struct {
		Packages map[string]struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}
	var lock lockfile
	if err := json.Unmarshal([]byte(read(t, "website/package-lock.json")), &lock); err != nil {
		t.Fatal(err)
	}
	found := false
	for path, pkg := range lock.Packages {
		if path != "node_modules/path-to-regexp" && pkg.Name != "path-to-regexp" {
			continue
		}
		found = true
		if versionLessThan(pkg.Version, "6.3.0") {
			t.Errorf("%s pins vulnerable path-to-regexp %s; require >= 6.3.0", path, pkg.Version)
		}
	}
	if !found {
		t.Fatal("package-lock does not contain path-to-regexp; review this policy test")
	}
}

func versionLessThan(version, floor string) bool {
	var got [3]int
	var want [3]int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &got[0], &got[1], &got[2]); err != nil {
		return true
	}
	if _, err := fmt.Sscanf(floor, "%d.%d.%d", &want[0], &want[1], &want[2]); err != nil {
		return true
	}
	for i := range got {
		if got[i] != want[i] {
			return got[i] < want[i]
		}
	}
	return false
}

func TestReleaseWorkflowRestoresAnnotatedTagBeforeVerification(t *testing.T) {
	workflow := read(t, ".github/workflows/release.yml")
	restore := `git fetch --force origin "refs/tags/$TAG:refs/tags/$TAG"`
	verify := `git verify-tag "$TAG"`

	restoreAt := strings.Index(workflow, restore)
	if restoreAt < 0 {
		t.Fatalf("release workflow must restore the annotated tag after checkout; missing %q", restore)
	}
	verifyAt := strings.Index(workflow, verify)
	if verifyAt < 0 {
		t.Fatalf("release workflow must verify the release tag; missing %q", verify)
	}
	if restoreAt > verifyAt {
		t.Fatal("release workflow must restore the annotated tag before verifying its signature")
	}
}

func TestWebsiteDeploymentWorkflowValidatesSignedTagWithoutDeploying(t *testing.T) {
	workflow := read(t, ".github/workflows/website-deployment-tag.yml")
	for _, required := range []string{
		`tags:`,
		`- "website-*"`,
		`contents: read`,
		`^website-v([0-9]{4})\.([0-9]{2})\.([0-9]{2})\.([1-9][0-9]*)$`,
		`git verify-tag "$WEBSITE_TAG"`,
		`git verify-tag "$CLI_TAG"`,
		`git rev-parse origin/main`,
		`git merge-base --is-ancestor "$CLI_COMMIT" "$WEBSITE_COMMIT"`,
		`website/public/install.sh website/public/install.ps1`,
		`website/src/data/product.ts`,
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("website deployment tag workflow is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`vercel deploy`,
		`vercel promote`,
		`contents: write`,
	} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("website deployment tag workflow contains forbidden mutation %q", forbidden)
		}
	}

	releaseWorkflow := read(t, ".github/workflows/release.yml")
	if strings.Contains(releaseWorkflow, `"website-*"`) {
		t.Error("website deployment tags must not trigger the CLI release workflow")
	}
}

func TestDevelopmentVersionIgnoresWebsiteDeploymentTags(t *testing.T) {
	makefile := read(t, "Makefile")
	if !strings.Contains(
		makefile,
		`git describe --tags --match 'v[0-9]*' --always --dirty`,
	) {
		t.Fatal("development version must describe only v-prefixed CLI release tags")
	}
}

func TestGoReleaserSnapshotVersionIgnoresWebsiteDeploymentTags(t *testing.T) {
	config := read(t, ".goreleaser.yml")
	if !strings.Contains(config, `version_template: "0.0.0-{{ .ShortCommit }}"`) {
		t.Fatal("GoReleaser snapshots must use a commit-derived SemVer independent of the nearest tag")
	}
	if strings.Contains(config, `incpatch .Version`) {
		t.Fatal("GoReleaser snapshots must not parse a website deployment tag as a CLI SemVer")
	}
}

func TestGoReleaserEmbedsFullCommitIdentity(t *testing.T) {
	config := read(t, ".goreleaser.yml")
	if !strings.Contains(config, "release:\n  draft: true") {
		t.Fatal("GoReleaser must upload a draft so artifact validation finishes before publication")
	}
	fullCommitFlag := `internal/version.Commit={{.FullCommit}}`
	if !strings.Contains(config, fullCommitFlag) {
		t.Fatalf("GoReleaser must embed the full release commit; missing %q", fullCommitFlag)
	}
	if strings.Contains(config, `internal/version.Commit={{.ShortCommit}}`) {
		t.Fatal("GoReleaser still embeds a short commit in release binaries")
	}

	workflow := read(t, ".github/workflows/release.yml")
	if !strings.Contains(workflow, `./scripts/check-release-binary-identity.sh dist`) {
		t.Fatal("release workflow must execute an artifact and verify its full commit identity")
	}
}

func TestMakefileLimitsCGOToRaceTests(t *testing.T) {
	makefile := read(t, "Makefile")
	if !strings.Contains(makefile, "git rev-parse HEAD") || strings.Contains(makefile, "git rev-parse --short HEAD") {
		t.Fatal("development builds must embed the full commit identity")
	}
	if !strings.Contains(makefile, "test: ## Run unit tests\n\tCGO_ENABLED=0 ") {
		t.Fatal("ordinary tests must disable CGO for a deterministic cross-platform gate")
	}
	if !strings.Contains(makefile, "test-race: ## Run tests with race detector\n\tCGO_ENABLED=1 ") {
		t.Fatal("race tests must enable CGO explicitly instead of inheriting process state")
	}
}

func TestVerifyAvoidsRedundantDoctestRuns(t *testing.T) {
	command := exec.Command("make", "-n", "verify")
	command.Dir = repoRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n verify failed: %v\n%s", err, output)
	}
	text := string(output)
	if strings.Contains(text, "./scripts/check-docs.sh") {
		t.Fatal("verify repeats internal/doctest through docs-check after go test ./...")
	}
	if strings.Contains(text, "go test ./internal/fixture") {
		t.Fatal("verify repeats fixture scanning after go test ./... already covered it")
	}
	var raceCommand string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "go test") && strings.Contains(line, "-race") {
			raceCommand = line
			break
		}
	}
	if raceCommand == "" {
		t.Fatal("verify dry-run does not contain a race test command")
	}
	if strings.Contains(raceCommand, "./...") {
		t.Fatalf("race gate uses the unfiltered all-package pattern: %s", raceCommand)
	}
	if strings.Contains(raceCommand, "/internal/doctest") {
		t.Fatalf("race gate repeats subprocess/document contracts: %s", raceCommand)
	}
	if strings.Contains(raceCommand, "/internal/crypto") {
		t.Fatalf("race gate repeats production-strength scrypt: %s", raceCommand)
	}
}

func TestCIVerificationUsesOptimizedRaceGate(t *testing.T) {
	workflow := read(t, ".github/workflows/ci.yml")
	if !strings.Contains(workflow, "run: make test-race") {
		t.Fatal("Linux CI must reuse the filtered make test-race gate")
	}
	if strings.Contains(workflow, "go test ./... -race") {
		t.Fatal("CI still runs the redundant all-package race command")
	}
	if strings.Contains(workflow, "scripts/check-docs") {
		t.Fatal("CI repeats internal/doctest after go test ./... already covered it")
	}
	if strings.Contains(workflow, "go test ./internal/fixture") {
		t.Fatal("CI repeats fixture scanning after go test ./... already covered it")
	}
}

func TestQuickGateStaysFocusedAndNonRelease(t *testing.T) {
	command := exec.Command("make", "-n", "quick")
	command.Dir = repoRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n quick failed: %v\n%s", err, output)
	}
	text := string(output)
	if !strings.Contains(text, "go vet ./...") {
		t.Fatal("quick gate must retain go vet")
	}
	if strings.Contains(text, "/internal/doctest") || strings.Contains(text, "/internal/crypto") {
		t.Fatalf("quick gate includes a deliberately slow package:\n%s", text)
	}
	if strings.Contains(text, "-count=1") {
		t.Fatal("quick gate disables Go's test cache")
	}
	for _, releaseOnly := range []string{"-race", "golangci-lint", "govulncheck"} {
		if strings.Contains(text, releaseOnly) {
			t.Fatalf("quick gate unexpectedly includes release-only work %q", releaseOnly)
		}
	}
}

func TestLocalMarkdownLinksResolve(t *testing.T) {
	root := repoRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "bin", "dist", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := markdownOutsideFences(string(bodyBytes))
		for _, match := range markdownLinkPattern.FindAllStringSubmatch(body, -1) {
			target := markdownLinkTarget(match[1])
			if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") {
				continue
			}
			parsed, err := url.Parse(target)
			if err != nil {
				t.Errorf("%s has invalid link %q: %v", relativePath(root, path), target, err)
				continue
			}
			if parsed.IsAbs() || parsed.Host != "" {
				continue
			}
			decoded, err := url.PathUnescape(parsed.Path)
			if err != nil {
				t.Errorf("%s has invalid escaped link %q: %v", relativePath(root, path), target, err)
				continue
			}
			if decoded == "" {
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(decoded)))
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s links to missing local path %q", relativePath(root, path), target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func markdownOutsideFences(body string) string {
	var result strings.Builder
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if !inFence {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func markdownLinkTarget(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "<") {
		if end := strings.IndexByte(raw, '>'); end > 0 {
			return raw[1:end]
		}
	}
	if space := strings.IndexAny(raw, " \t"); space >= 0 {
		raw = raw[:space]
	}
	return strings.Trim(raw, "<>")
}

func relativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}
