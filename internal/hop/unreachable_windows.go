//go:build windows

package hop

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isConnectionRefused reports whether err is a TCP connection actively
// refused (WSAECONNREFUSED) — no process listening at the dialled address,
// as opposed to a route that never answers at all.
//
// The standard library's syscall.ECONNREFUSED is not the Windows Sockets
// error code on this platform (it is a portable placeholder value that
// never matches what net/http actually wraps here), so this needs the
// platform constant from golang.org/x/sys/windows, already a direct
// dependency of this module.
func isConnectionRefused(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
