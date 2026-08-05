//go:build !windows

package fileidentity

import (
	"errors"
	"os"
	"syscall"
)

func platformIdentity(_ *os.File, info os.FileInfo) (uint64, uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("filesystem identity is unavailable")
	}
	return uint64(stat.Dev), uint64(stat.Ino), nil
}
