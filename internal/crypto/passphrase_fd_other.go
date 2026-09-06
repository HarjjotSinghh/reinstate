//go:build !windows && !aix && !android && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris

package crypto

import (
	"errors"
	"os"
)

func duplicatePassphraseFD(_ int) (*os.File, error) {
	return nil, errors.New("passphrase descriptor duplication is unavailable")
}
