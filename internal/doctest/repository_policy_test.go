package doctest

import (
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

func TestLocalMarkdownLinksResolve(t *testing.T) {
	root := repoRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "bin", "dist", "vendor":
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
