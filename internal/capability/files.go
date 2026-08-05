package capability

import (
	"context"
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
	pathCancelled
)

func inspectPath(ctx context.Context, root, path string, sizeLimit int64) (pathStatus, os.FileInfo) {
	if contextCancelled(ctx) {
		return pathCancelled, nil
	}
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
			if contextCancelled(ctx) {
				return pathCancelled, nil
			}
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

	if contextCancelled(ctx) {
		return pathCancelled, nil
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

func readBounded(ctx context.Context, root, path string) ([]byte, pathStatus) {
	status, before := inspectPath(ctx, root, path, maxConfigBytes)
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
	if contextCancelled(ctx) {
		return nil, pathCancelled
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return nil, pathFailed
	}
	defer rootHandle.Close()
	if contextCancelled(ctx) {
		return nil, pathCancelled
	}
	f, err := rootHandle.Open(rel)
	if err != nil {
		return nil, pathFailed
	}
	defer f.Close()
	if contextCancelled(ctx) {
		return nil, pathCancelled
	}
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || after.Size() > maxConfigBytes || !os.SameFile(before, after) {
		return nil, pathFailed
	}
	b := make([]byte, 0, min(int(before.Size()), int(maxConfigBytes)))
	chunk := make([]byte, 32<<10)
	for {
		if contextCancelled(ctx) {
			return nil, pathCancelled
		}
		n, readErr := f.Read(chunk)
		if n > 0 {
			if int64(len(b)+n) > maxConfigBytes {
				return nil, pathOversized
			}
			b = append(b, chunk[:n]...)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, pathFailed
		}
	}
	if contextCancelled(ctx) {
		return nil, pathCancelled
	}
	return b, pathRegular
}

func readDirBounded(ctx context.Context, root, path string) ([]os.DirEntry, bool, pathStatus) {
	status, before := inspectPath(ctx, root, path, -1)
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
	if contextCancelled(ctx) {
		return nil, false, pathCancelled
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return nil, false, pathFailed
	}
	defer rootHandle.Close()
	if contextCancelled(ctx) {
		return nil, false, pathCancelled
	}
	dir, err := rootHandle.Open(rel)
	if err != nil {
		return nil, false, pathFailed
	}
	defer dir.Close()
	if contextCancelled(ctx) {
		return nil, false, pathCancelled
	}
	after, err := dir.Stat()
	if err != nil || !after.IsDir() || !os.SameFile(before, after) {
		return nil, false, pathFailed
	}
	entries := make([]os.DirEntry, 0, maxEntries)
	truncated := false
	for {
		if contextCancelled(ctx) {
			return nil, false, pathCancelled
		}
		batch, readErr := dir.ReadDir(64)
		for _, entry := range batch {
			if contextCancelled(ctx) {
				return nil, false, pathCancelled
			}
			if len(entries) < maxEntries {
				entries = append(entries, entry)
				continue
			}
			truncated = true
			maxIndex := 0
			for i := 1; i < len(entries); i++ {
				if contextCancelled(ctx) {
					return nil, false, pathCancelled
				}
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
	if contextCancelled(ctx) {
		return nil, false, pathCancelled
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	if contextCancelled(ctx) {
		return nil, false, pathCancelled
	}
	return entries, truncated, pathDirectory
}

func contextCancelled(ctx context.Context) bool {
	return ctx != nil && ctx.Err() != nil
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
	case pathCancelled:
		return DiagnosticCancelled
	default:
		return DiagnosticReadFailed
	}
}
