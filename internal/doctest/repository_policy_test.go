package doctest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
