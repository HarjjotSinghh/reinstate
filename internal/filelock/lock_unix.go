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
	return tryFlock(file, operation, unix.Flock)
}

func tryFlock(file *os.File, operation int, flock func(int, int) error) (bool, error) {
	err := flock(int(file.Fd()), operation)
	if err == nil {
		return true, nil
	}
	// Let the context-bounded outer acquisition loop retry interruptions. An
	// immediate inner retry loop could spin indefinitely under a signal storm.
	if errors.Is(err, unix.EINTR) || errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

func unlock(file *os.File) error {
	return unlockFlock(file, unix.Flock)
}

func unlockFlock(file *os.File, flock func(int, int) error) error {
	for {
		err := flock(int(file.Fd()), unix.LOCK_UN)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
