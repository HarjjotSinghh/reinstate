// Package pathmap rewrites absolute paths to portable tokens and back.
package pathmap

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Mapper holds project roots and home for normalize/denormalize.
type Mapper struct {
	Home string
	// Projects maps canonical project IDs to configured local roots and is used
	// when denormalizing portable paths on the destination device.
	Projects map[string]string
	// NormalizeProjects optionally maps the same IDs to resolved physical roots.
	// This lets paths emitted beneath a symlink target normalize to the canonical
	// project while preserving the configured root for destination rewrites.
	NormalizeProjects map[string]string
	Aliases           map[string]string // WORK alias -> local absolute root
	// GOOS overrides path style for tests ("windows" | "darwin" | "linux").
	GOOS string
}

func (m Mapper) goos() string {
	if m.GOOS != "" {
		return m.GOOS
	}
	return runtime.GOOS
}

var (
	reRepo = regexp.MustCompile(`\$\{REPO:([^}]+)\}`)
	reWork = regexp.MustCompile(`\$\{WORK:([^}]+)\}`)
)

// Normalize converts a platform path into portable form.
func (m Mapper) Normalize(platformPath string) string {
	p := filepath.Clean(platformPath)
	// longest project root first
	bestID, bestRoot := "", ""
	for _, projects := range []map[string]string{m.NormalizeProjects, m.Projects} {
		for id, root := range projects {
			r := filepath.Clean(root)
			if hasPrefixPath(p, r, m.goos()) && len(r) > len(bestRoot) {
				bestID, bestRoot = id, r
			}
		}
	}
	if bestID != "" {
		rel := trimPrefixPath(p, bestRoot, m.goos())
		rel = toSlash(rel)
		if rel == "" || rel == "." {
			return fmt.Sprintf("${REPO:%s}", bestID)
		}
		return fmt.Sprintf("${REPO:%s}/%s", bestID, strings.TrimPrefix(rel, "/"))
	}
	for alias, root := range m.Aliases {
		r := filepath.Clean(root)
		if hasPrefixPath(p, r, m.goos()) {
			rel := toSlash(trimPrefixPath(p, r, m.goos()))
			if rel == "" || rel == "." {
				return fmt.Sprintf("${WORK:%s}", alias)
			}
			return fmt.Sprintf("${WORK:%s}/%s", alias, strings.TrimPrefix(rel, "/"))
		}
	}
	if m.Home != "" && hasPrefixPath(p, filepath.Clean(m.Home), m.goos()) {
		rel := toSlash(trimPrefixPath(p, filepath.Clean(m.Home), m.goos()))
		if rel == "" || rel == "." {
			return "${HOME}"
		}
		return "${HOME}/" + strings.TrimPrefix(rel, "/")
	}
	// WSL /mnt/c/...
	if strings.HasPrefix(strings.ToLower(toSlash(p)), "/mnt/") {
		return toSlash(p) // leave; denormalize may map when on Windows
	}
	return toSlash(p)
}

// Denormalize converts a portable path to the current platform.
func (m Mapper) Denormalize(portable string) string {
	p := portable
	if strings.HasPrefix(p, "${HOME}") {
		rest := strings.TrimPrefix(p, "${HOME}")
		rest = strings.TrimPrefix(rest, "/")
		if rest == "" {
			return filepath.Clean(m.Home)
		}
		return filepath.Join(m.Home, filepath.FromSlash(rest))
	}
	if match := reRepo.FindStringSubmatch(p); len(match) == 2 {
		id := match[1]
		root, ok := m.Projects[id]
		if !ok {
			return p
		}
		rest := strings.TrimPrefix(p, match[0])
		rest = strings.TrimPrefix(rest, "/")
		if rest == "" {
			return filepath.Clean(root)
		}
		return filepath.Join(root, filepath.FromSlash(rest))
	}
	if match := reWork.FindStringSubmatch(p); len(match) == 2 {
		alias := match[1]
		root, ok := m.Aliases[alias]
		if !ok {
			return p
		}
		rest := strings.TrimPrefix(p, match[0])
		rest = strings.TrimPrefix(rest, "/")
		if rest == "" {
			return filepath.Clean(root)
		}
		return filepath.Join(root, filepath.FromSlash(rest))
	}
	if m.goos() == "windows" {
		return filepath.FromSlash(p)
	}
	return filepath.Clean(filepath.FromSlash(p))
}

func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func hasPrefixPath(path, prefix, goos string) bool {
	path = toSlash(filepath.Clean(path))
	prefix = toSlash(filepath.Clean(prefix))
	if goos == "windows" {
		path = strings.ToLower(path)
		prefix = strings.ToLower(prefix)
	}
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/")
}

func trimPrefixPath(path, prefix, goos string) string {
	path = toSlash(filepath.Clean(path))
	prefix = toSlash(filepath.Clean(prefix))
	orig := path
	if goos == "windows" {
		path = strings.ToLower(path)
		prefix = strings.ToLower(prefix)
	}
	if path == prefix {
		return ""
	}
	pre := strings.TrimSuffix(prefix, "/") + "/"
	if !strings.HasPrefix(path, pre) {
		return orig
	}
	// return from original with correct case
	return orig[len(pre):]
}
