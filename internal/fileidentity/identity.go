// Package fileidentity captures private, platform-native filesystem identities
// for launch-bound safety checks. Identities are never serialized.
package fileidentity

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"runtime"
)

const maxExecutableBytes = 512 << 20

// Identity combines a stable filesystem object key with metadata that changes
// on an in-place executable rewrite.
type Identity struct {
	volume  uint64
	file    uint64
	size    int64
	modTime int64
	mode    os.FileMode
	digest  [sha256.Size]byte
	hashed  bool
}

// CaptureExecutable captures both native object identity and a complete,
// bounded SHA-256 digest. The context is checked throughout the read.
func CaptureExecutable(ctx context.Context, path string) (Identity, error) {
	handle, err := os.Open(path)
	if err != nil {
		return Identity{}, err
	}
	defer func() { _ = handle.Close() }()
	before, err := handle.Stat()
	if err != nil {
		return Identity{}, err
	}
	if !isBoundedLaunchable(before) {
		return Identity{}, errors.New("executable is not a bounded launchable file")
	}
	volume, file, err := platformIdentity(handle, before)
	if err != nil {
		return Identity{}, err
	}
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return Identity{}, err
		}
		count, readErr := handle.Read(buffer)
		if count > 0 {
			total += int64(count)
			if total > before.Size() {
				return Identity{}, errors.New("executable grew while its identity was captured")
			}
			_, _ = hash.Write(buffer[:count])
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Identity{}, readErr
		}
	}
	if total != before.Size() {
		return Identity{}, errors.New("executable was truncated while its identity was captured")
	}
	after, err := handle.Stat()
	if err != nil {
		return Identity{}, err
	}
	afterVolume, afterFile, err := platformIdentity(handle, after)
	if err != nil {
		return Identity{}, err
	}
	if volume != afterVolume || file != afterFile || before.Size() != after.Size() ||
		before.ModTime() != after.ModTime() || before.Mode() != after.Mode() {
		return Identity{}, errors.New("executable changed while its identity was captured")
	}
	if err := ctx.Err(); err != nil {
		return Identity{}, err
	}
	identity := Identity{
		volume: volume, file: file, size: after.Size(),
		modTime: after.ModTime().UnixNano(), mode: after.Mode(), hashed: true,
	}
	copy(identity.digest[:], hash.Sum(nil))
	return identity, nil
}

// Capture opens path and captures the identity of the opened object, avoiding
// a separate path-stat race while the identity itself is collected.
func Capture(path string) (Identity, error) {
	handle, err := os.Open(path)
	if err != nil {
		return Identity{}, err
	}
	defer func() { _ = handle.Close() }()
	info, err := handle.Stat()
	if err != nil {
		return Identity{}, err
	}
	volume, file, err := platformIdentity(handle, info)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		volume:  volume,
		file:    file,
		size:    info.Size(),
		modTime: info.ModTime().UnixNano(),
		mode:    info.Mode(),
	}, nil
}

// SameObject reports whether two captures identify the same filesystem object.
func SameObject(left, right Identity) bool {
	return left.volume == right.volume && left.file == right.file &&
		left.mode.Type() == right.mode.Type()
}

// SameExecutable reports whether the same object also retained executable
// metadata. It detects both replacement and ordinary in-place rewrites.
func SameExecutable(left, right Identity) bool {
	return SameObject(left, right) && left.size == right.size &&
		left.modTime == right.modTime && left.mode == right.mode && left.hashed &&
		right.hashed && left.digest == right.digest
}

// IsZero reports whether no identity was captured.
func (identity Identity) IsZero() bool { return identity == (Identity{}) }

// IsRegular reports whether the captured object is a regular file.
func (identity Identity) IsRegular() bool { return identity.mode.IsRegular() }

// IsLaunchable reports whether the captured object may be launched as a host
// executable. On Windows this includes irregular reparse shims that still
// launch under CreateProcess.
func (identity Identity) IsLaunchable() bool {
	if identity.mode.IsRegular() {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}
	modeType := identity.mode & os.ModeType
	return modeType == 0 || modeType == os.ModeIrregular || modeType == os.ModeSymlink
}

func isBoundedLaunchable(info os.FileInfo) bool {
	if info == nil || info.IsDir() || info.Size() < 0 || info.Size() > maxExecutableBytes {
		return false
	}
	if info.Mode().IsRegular() {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}
	modeType := info.Mode() & os.ModeType
	return modeType == 0 || modeType == os.ModeIrregular || modeType == os.ModeSymlink
}

// IsDir reports whether the captured object is a directory.
func (identity Identity) IsDir() bool { return identity.mode.IsDir() }
