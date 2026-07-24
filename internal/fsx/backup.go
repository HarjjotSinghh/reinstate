package fsx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupDir copies src tree into destRoot/<timestamp>-<name>.
func BackupDir(src, destRoot, name string) (string, error) {
	ts := time.Now().UTC().Format("20060102T150405Z")
	dest := filepath.Join(destRoot, fmt.Sprintf("%s-%s", ts, name))
	if err := copyDir(src, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// AtomicRestoreFile writes content via temp + rename into dest, leaving dest intact on failure.
func AtomicRestoreFile(dest string, content []byte, fail bool) error {
	if fail {
		return WriteFileAtomicFail(dest, content, 0o600)
	}
	return WriteFileAtomic(dest, content, 0o600)
}
