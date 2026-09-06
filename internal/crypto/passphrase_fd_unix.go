//go:build aix || android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package crypto

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func duplicatePassphraseFD(fd int) (*os.File, error) {
	duplicate, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(duplicate), "reinstate-passphrase-fd")
	if file == nil {
		_ = unix.Close(duplicate)
		return nil, errors.New("duplicate passphrase descriptor is unavailable")
	}
	return file, nil
}
