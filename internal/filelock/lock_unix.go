//go:build !windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryExclusive(file *os.File) (bool, error) {
	return try(file, unix.LOCK_EX|unix.LOCK_NB)
}

func tryShared(file *os.File) (bool, error) {
	return try(file, unix.LOCK_SH|unix.LOCK_NB)
}

func try(file *os.File, operation int) (bool, error) {
	err := unix.Flock(int(file.Fd()), operation)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

func unlock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
