// Package executabletrust resolves host tools without trusting executable code
// supplied by the workspace being inspected.
package executabletrust

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrUnavailable reports that no executable can be selected without crossing
// the workspace trust boundary.
var ErrUnavailable = errors.New("trusted executable is unavailable")

// Resolution contains the canonical executable and a sanitized PATH suitable
// for its child process.
type Resolution struct {
	Executable string
	SearchPath string
}

// Resolve selects name from absolute PATH entries outside workspace. When the
// workspace is below one or more Git markers, the outermost marked ancestor is
// the trust boundary. This deliberately over-filters nested repositories so a
// workspace-controlled inner .git marker cannot re-admit executable siblings
// from the enclosing repository. Both PATH directories and executable
// candidates are checked after symlink evaluation, and the returned PATH
// contains only the retained trusted directories.
//
// On Windows, extensionless names such as "codex" are resolved through PATHEXT
// (for example codex.exe / codex.cmd). PATHEXT is read from the supplied
// environment first, then the process environment, then a safe default list.
// Unix-like platforms resolve the exact basename only.
func Resolve(name, workspace string, environment []string) (Resolution, error) {
	if !validName(name) {
		return Resolution{}, ErrUnavailable
	}
	boundary, err := trustBoundary(workspace)
	if err != nil {
		return Resolution{}, ErrUnavailable
	}

	pathValue, ok := environmentValue(environment, "PATH")
	if !ok {
		return Resolution{}, ErrUnavailable
	}
	searchDirectories := trustedSearchDirectories(pathValue, boundary)
	searchPath := strings.Join(searchDirectories, string(os.PathListSeparator))
	for _, directory := range searchDirectories {
		for _, candidate := range executableCandidates(directory, name, environment) {
			resolved, candidateErr := canonicalExecutable(candidate, boundary)
			if candidateErr != nil {
				continue
			}
			return Resolution{Executable: resolved, SearchPath: searchPath}, nil
		}
	}
	return Resolution{}, ErrUnavailable
}

func trustBoundary(workspace string) (string, error) {
	canonical, err := canonicalDirectory(workspace)
	if err != nil {
		return "", err
	}

	boundary := canonical
	current := canonical
	for {
		_, markerErr := os.Lstat(filepath.Join(current, ".git"))
		switch {
		case markerErr == nil:
			// Keep walking: the highest marked ancestor is the safe boundary.
			boundary = current
		case errors.Is(markerErr, os.ErrNotExist):
			// No marker at this level.
		default:
			// If marker inspection itself is unreliable, exclude the entire
			// filesystem tree rather than treating an uncertain ancestor as
			// trusted executable search space.
			return filesystemRoot(current), nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return boundary, nil
		}
		current = parent
	}
}

func filesystemRoot(value string) string {
	current := filepath.Clean(value)
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return current
		}
		current = parent
	}
}

func validName(name string) bool {
	return name != "" && name == filepath.Base(name) && !filepath.IsAbs(name) &&
		!strings.ContainsAny(name, `/\\`)
}

func canonicalDirectory(value string) (string, error) {
	if value == "" {
		return "", ErrUnavailable
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(absolute))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return "", ErrUnavailable
	}
	return filepath.Clean(canonical), nil
}

func trustedSearchDirectories(pathValue, boundary string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == "" || !filepath.IsAbs(entry) {
			continue
		}
		absolute := filepath.Clean(entry)
		if withinFilesystem(absolute, boundary) {
			continue
		}
		canonical, err := canonicalDirectory(absolute)
		if err != nil || withinFilesystem(canonical, boundary) {
			continue
		}
		key := pathKey(canonical)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, canonical)
	}
	return result
}

func canonicalExecutable(value, boundary string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if withinFilesystem(absolute, boundary) {
		return "", ErrUnavailable
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil || withinFilesystem(canonical, boundary) {
		return "", ErrUnavailable
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.Mode().IsRegular() {
		return "", ErrUnavailable
	}
	return filepath.Clean(canonical), nil
}

// executableCandidates returns the filesystem paths to try for name inside
// directory. On Windows, extensionless vendor names such as "codex" expand
// through PATHEXT so installed codex.exe / codex.cmd shims resolve. Names that
// already include a final extension are tried exactly once.
func executableCandidates(directory, name string, environment []string) []string {
	if runtime.GOOS != "windows" {
		return []string{filepath.Join(directory, name)}
	}
	if windowsNameHasExtension(name) {
		return []string{filepath.Join(directory, name)}
	}
	extensions := windowsPathExtensions(environment)
	candidates := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		candidates = append(candidates, filepath.Join(directory, name+extension))
	}
	return candidates
}

func windowsNameHasExtension(name string) bool {
	// Match Go's Windows LookPath hasExt: a final "." after the last separator.
	// validName already rejects separators, so filepath.Ext is sufficient.
	return filepath.Ext(name) != ""
}

func windowsPathExtensions(environment []string) []string {
	value, ok := environmentValue(environment, "PATHEXT")
	if !ok || strings.TrimSpace(value) == "" {
		value = os.Getenv("PATHEXT")
	}
	if strings.TrimSpace(value) == "" {
		return []string{".com", ".exe", ".bat", ".cmd"}
	}
	extensions := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range strings.Split(value, ";") {
		extension := strings.TrimSpace(part)
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		// Preserve PATHEXT order; compare case-insensitively for dedupe only.
		key := strings.ToLower(extension)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		extensions = append(extensions, extension)
	}
	if len(extensions) == 0 {
		return []string{".com", ".exe", ".bat", ".cmd"}
	}
	return extensions
}

func within(candidate, boundary string) bool {
	relative, err := filepath.Rel(pathKey(boundary), pathKey(candidate))
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func withinFilesystem(candidate, boundary string) bool {
	if within(candidate, boundary) {
		return true
	}
	boundaryInfo, err := os.Stat(boundary)
	if err != nil {
		// Existing callers canonicalize the boundary first. Preserve the
		// lexical fail-closed result if the filesystem changes underneath us.
		return true
	}
	current := filepath.Clean(candidate)
	for {
		if info, statErr := os.Stat(current); statErr == nil && os.SameFile(info, boundaryInfo) {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}

func environmentValue(environment []string, key string) (string, bool) {
	for index := len(environment) - 1; index >= 0; index-- {
		currentKey, value, ok := strings.Cut(environment[index], "=")
		if ok && strings.EqualFold(currentKey, key) {
			return value, true
		}
	}
	return "", false
}

func pathKey(value string) string {
	value = filepath.Clean(value)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.ToLower(value)
	}
	return value
}
