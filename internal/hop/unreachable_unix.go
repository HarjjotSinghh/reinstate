//go:build !windows

package hop

import (
	"errors"
	"syscall"
)

// isConnectionRefused reports whether err is a TCP connection actively
// refused (ECONNREFUSED) — no process listening at the dialled address, as
// opposed to a route that never answers at all.
func isConnectionRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
