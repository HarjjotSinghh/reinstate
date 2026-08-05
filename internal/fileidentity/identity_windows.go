//go:build windows

package fileidentity

import (
	"os"

	"golang.org/x/sys/windows"
)

func platformIdentity(handle *os.File, _ os.FileInfo) (uint64, uint64, error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(handle.Fd()), &info); err != nil {
		return 0, 0, err
	}
	file := uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow)
	return uint64(info.VolumeSerialNumber), file, nil
}
