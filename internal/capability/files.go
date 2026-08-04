package capability

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type pathStatus uint8

const (
	pathMissing pathStatus = iota
	pathRegular
	pathDirectory
	pathSymlink
	pathOversized
	pathUnsafe
	pathFailed
)

func inspectPath(root, path string, sizeLimit int64) (pathStatus, os.FileInfo) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return pathUnsafe, nil
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil || !withinRoot(rootAbs, pathAbs) {
		return pathUnsafe, nil
	}

	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return pathUnsafe, nil
	}
	current := rootAbs
	if rel != "." {
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			info, statErr := os.Lstat(current)
			if errors.Is(statErr, os.ErrNotExist) {
				return pathMissing, nil
			}
			if statErr != nil {
				return pathFailed, nil
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return pathSymlink, info
			}
		}
	}

	info, err := os.Lstat(pathAbs)
	if errors.Is(err, os.ErrNotExist) {
		return pathMissing, nil
	}
	if err != nil {
		return pathFailed, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return pathSymlink, info
	}
	if info.IsDir() {
		return pathDirectory, info
	}
	if !info.Mode().IsRegular() {
		return pathFailed, info
	}
	if sizeLimit >= 0 && info.Size() > sizeLimit {
		return pathOversized, info
	}
	return pathRegular, info
}

func readBounded(root, path string) ([]byte, pathStatus) {
	status, before := inspectPath(root, path, maxConfigBytes)
	if status != pathRegular {
		return nil, status
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, pathUnsafe
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil || !withinRoot(rootAbs, pathAbs) {
		return nil, pathUnsafe
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return nil, pathUnsafe
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return nil, pathFailed
	}
	defer rootHandle.Close()
	f, err := rootHandle.Open(rel)
	if err != nil {
		return nil, pathFailed
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || after.Size() > maxConfigBytes || !os.SameFile(before, after) {
		return nil, pathFailed
	}
	b, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil {
		return nil, pathFailed
	}
	if int64(len(b)) > maxConfigBytes {
		return nil, pathOversized
	}
	return b, pathRegular
}

func readDirBounded(root, path string) ([]os.DirEntry, bool, pathStatus) {
	status, before := inspectPath(root, path, -1)
	if status != pathDirectory {
		return nil, false, status
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, false, pathUnsafe
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil || !withinRoot(rootAbs, pathAbs) {
		return nil, false, pathUnsafe
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return nil, false, pathUnsafe
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return nil, false, pathFailed
	}
	defer rootHandle.Close()
	dir, err := rootHandle.Open(rel)
	if err != nil {
		return nil, false, pathFailed
	}
	defer dir.Close()
	after, err := dir.Stat()
	if err != nil || !after.IsDir() || !os.SameFile(before, after) {
		return nil, false, pathFailed
	}
	entries := make([]os.DirEntry, 0, maxEntries)
	truncated := false
	for {
		batch, readErr := dir.ReadDir(64)
		for _, entry := range batch {
			if len(entries) < maxEntries {
				entries = append(entries, entry)
				continue
			}
			truncated = true
			maxIndex := 0
			for i := 1; i < len(entries); i++ {
				if entries[i].Name() > entries[maxIndex].Name() {
					maxIndex = i
				}
			}
			if entry.Name() < entries[maxIndex].Name() {
				entries[maxIndex] = entry
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, false, pathFailed
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, truncated, pathDirectory
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func diagnosticForStatus(status pathStatus) DiagnosticCode {
	switch status {
	case pathSymlink:
		return DiagnosticSymlink
	case pathOversized:
		return DiagnosticOversized
	case pathUnsafe:
		return DiagnosticUnsafePath
	default:
		return DiagnosticReadFailed
	}
}
