//go:build windows

package crypto

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func duplicatePassphraseFD(fd uintptr) (*os.File, error) {
	process := windows.CurrentProcess()
	var duplicate windows.Handle
	if err := windows.DuplicateHandle(
		process,
		windows.Handle(fd),
		process,
		&duplicate,
		0,
		false,
		windows.DUPLICATE_SAME_ACCESS,
	); err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(duplicate), "reinstate-passphrase-fd")
	if file == nil {
		_ = windows.CloseHandle(duplicate)
		return nil, errors.New("duplicate passphrase descriptor is unavailable")
	}
	return file, nil
}
