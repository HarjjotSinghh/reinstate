// Package hometree is the shared F1 scanner: home-tree root resolution, a
// bounded glob walk, JSONL reading to the last complete line, and mod-time
// plus size change detection.
//
// It is read-only. It reuses sessionindex.MaxJSONLineBytes,
// MaxSearchTextBytes, and MaxFileReferences. It never writes, renames,
// truncates, or locks a vendor tree.
package hometree

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Existing ceilings, reused so a new F1 agent cannot introduce an unbounded read.
const (
	MaxJSONLineBytes   = sessionindex.MaxJSONLineBytes
	MaxSearchTextBytes = sessionindex.MaxSearchTextBytes
	MaxFileReferences  = sessionindex.MaxFileReferences
)

// Config is the shared discovery surface extracted from Claude, Codex, Gemini,
// and Grok: an explicit root, an environment override, ordered home-relative
// candidates, a marker that must exist, a session glob, and excluded subtrees.
type Config struct {
	Explicit    string
	RootEnv     string
	LookupEnv   func(string) string
	Candidates  []string
	Marker      string
	SessionGlob string
	Excluded    []string
}

// File is one path discovered under a home-tree root.
type File struct {
	Path    string
	ModTime time.Time
	Size    int64
}

// Stamp is the freshness tuple the index uses to skip unchanged sessions.
type Stamp struct {
	ModTime int64
	Size    int64
}

// ResolveRoot returns the storage root. Explicit wins, then the environment
// override, then the first existing candidate. A missing live tree is not an
// error: the caller treats an empty root as "not installed".
func ResolveRoot(cfg Config) (string, error) {
	if explicit := strings.TrimSpace(cfg.Explicit); explicit != "" {
		return cleanRoot(explicit), nil
	}
	lookup := cfg.LookupEnv
	if lookup == nil {
		lookup = os.Getenv
	}
	if envName := strings.TrimSpace(cfg.RootEnv); envName != "" {
		if configured := strings.TrimSpace(lookup(envName)); configured != "" {
			return cleanRoot(configured), nil
		}
	}
	for _, candidate := range cfg.Candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("inspect candidate %q: %w", candidate, err)
		}
		if info.IsDir() {
			return cleanRoot(candidate), nil
		}
	}
	return "", nil
}

// HasMarker reports whether the relative marker exists under root. An empty
// marker always matches. A missing marker is not an error.
func HasMarker(root, marker string) (bool, error) {
	root = strings.TrimSpace(root)
	marker = strings.TrimSpace(marker)
	if root == "" {
		return false, nil
	}
	if marker == "" {
		return true, nil
	}
	info, err := os.Stat(filepath.Join(root, marker))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("inspect marker %q: %w", marker, err)
	}
	return info != nil, nil
}

// Discover resolves the root, checks the marker, and walks matching session
// files. A missing root or marker yields an empty file list, matching the
// existing F1 sources.
func Discover(ctx context.Context, cfg Config) (string, []File, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	root, err := ResolveRoot(cfg)
	if err != nil {
		return "", nil, err
	}
	if root == "" {
		return "", nil, nil
	}
	ok, err := HasMarker(root, cfg.Marker)
	if err != nil {
		return root, nil, err
	}
	if !ok {
		return root, nil, nil
	}
	files, err := Walk(ctx, root, cfg.SessionGlob, cfg.Excluded)
	return root, files, err
}

// Walk lists regular files under root whose relative path matches glob,
// skipping excluded subtrees. The walk is read-only and context-cancellable.
// An empty glob matches nothing so a missing pattern cannot dump the tree.
func Walk(ctx context.Context, root, glob string, excluded []string) ([]File, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	glob = strings.TrimSpace(glob)
	if glob == "" {
		return nil, nil
	}
	var files []File
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && excludedPath(rel, entry.Name(), excluded) {
				return filepath.SkipDir
			}
			return nil
		}
		if excludedPath(rel, entry.Name(), excluded) {
			return nil
		}
		if !matchGlob(glob, rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, File{
			Path:    path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", root, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// ReadJSONL visits complete JSON values up to the last complete line, using
// the shared MaxJSONLineBytes ceiling. An incomplete trailing record is
// ignored. The file is opened read-only.
func ReadJSONL(path string, visit func(lineNumber int, line []byte) error) ([]sessionindex.Warning, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return sessionindex.ScanJSONLines(file, MaxJSONLineBytes, visit)
}

// Stat returns the freshness stamp for path.
func Stat(path string) (Stamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Stamp{}, err
	}
	return Stamp{ModTime: info.ModTime().UnixNano(), Size: info.Size()}, nil
}

// Changed reports whether current differs from previous in mod-time or size.
func Changed(current, previous Stamp) bool {
	return current.ModTime != previous.ModTime || current.Size != previous.Size
}

func cleanRoot(root string) string {
	root = filepath.Clean(root)
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func excludedPath(rel, name string, excluded []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range excluded {
		pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
		pattern = strings.TrimPrefix(pattern, "./")
		if pattern == "" {
			continue
		}
		base := path.Base(pattern)
		if strings.EqualFold(name, base) {
			return true
		}
		if matchGlob(pattern, rel) {
			return true
		}
		lowerRel := strings.ToLower(rel)
		lowerPat := strings.ToLower(strings.Trim(pattern, "/"))
		if lowerPat != "" && (lowerRel == lowerPat || strings.HasPrefix(lowerRel, lowerPat+"/")) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, name string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "./")
	name = strings.TrimPrefix(filepath.ToSlash(name), "./")
	return matchGlobParts(splitSlash(pattern), splitSlash(name))
}

func splitSlash(value string) []string {
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

func matchGlobParts(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			if len(pattern) == 1 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchGlobParts(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		ok, err := path.Match(pattern[0], name[0])
		if err != nil || !ok {
			return false
		}
		pattern = pattern[1:]
		name = name[1:]
	}
	return len(name) == 0
}

// Fingerprint summarises everything Discover would return without opening a
// single file: the resolved root plus each matched file's path, modification
// time and size. Two refreshes that produce the same fingerprint cannot have
// produced different records, so the caller can skip parsing entirely.
//
// The second return reports whether a fingerprint is meaningful at all. A
// source whose root does not resolve has nothing to compare, and must be
// scanned rather than assumed unchanged.
func Fingerprint(ctx context.Context, cfg Config) (string, bool, error) {
	root, files, err := Discover(ctx, cfg)
	if err != nil {
		return "", false, err
	}
	if root == "" {
		return "", false, nil
	}
	sorted := make([]File, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	sum := sha256.New()
	_, _ = sum.Write([]byte(root))
	_, _ = sum.Write([]byte{0})
	for _, file := range sorted {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		_, _ = sum.Write([]byte(file.Path))
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(file.ModTime.UnixNano(), 10)))
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(strconv.FormatInt(file.Size, 10)))
		_, _ = sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil)), true, nil
}
