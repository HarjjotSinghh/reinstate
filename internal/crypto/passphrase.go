package crypto

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const maxSecretInputBytes = 4096

// ReadPassphrase reads a passphrase from REINSTATE_PASSPHRASE_FD when
// deliberately configured, otherwise from an interactive terminal with echo
// disabled. Passphrases are never accepted as command arguments or ordinary
// environment values.
func ReadPassphrase(input io.Reader, promptOut io.Writer) ([]byte, error) {
	if rawFD := strings.TrimSpace(os.Getenv("REINSTATE_PASSPHRASE_FD")); rawFD != "" {
		fd, err := strconv.Atoi(rawFD)
		if err != nil || fd < 0 {
			return nil, fmt.Errorf("REINSTATE_PASSPHRASE_FD must be a valid file descriptor")
		}
		file, err := duplicatePassphraseFD(uintptr(fd))
		if err != nil || file == nil {
			return nil, fmt.Errorf("REINSTATE_PASSPHRASE_FD is unavailable")
		}
		// Read and close only a duplicate. A caller that owns the configured
		// descriptor may keep its wrapper alive, so closing the original handle
		// here can close an unrelated reused Windows handle later.
		defer func() { _ = file.Close() }()
		return readBoundedSecret(file)
	}
	return ReadHiddenSecret(input, promptOut, "Encryption passphrase: ")
}

// ReadHiddenSecret reads from a real terminal with echo disabled.
func ReadHiddenSecret(input io.Reader, promptOut io.Writer, prompt string) ([]byte, error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return nil, fmt.Errorf("secret input requires an interactive terminal or REINSTATE_PASSPHRASE_FD")
	}
	if _, err := fmt.Fprint(promptOut, prompt); err != nil {
		return nil, err
	}
	secret, err := term.ReadPassword(int(file.Fd()))
	_, _ = fmt.Fprintln(promptOut)
	if err != nil {
		return nil, fmt.Errorf("read hidden secret: %w", err)
	}
	secret = bytes.TrimRight(secret, "\r\n")
	if len(secret) == 0 {
		return nil, fmt.Errorf("secret must not be empty")
	}
	if len(secret) > maxSecretInputBytes {
		Zero(secret)
		return nil, fmt.Errorf("secret exceeds %d bytes", maxSecretInputBytes)
	}
	return secret, nil
}

func readBoundedSecret(reader io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxSecretInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxSecretInputBytes {
		Zero(raw)
		return nil, fmt.Errorf("secret exceeds %d bytes", maxSecretInputBytes)
	}
	raw = bytes.TrimRight(raw, "\r\n")
	if len(raw) == 0 {
		return nil, fmt.Errorf("secret must not be empty")
	}
	return raw, nil
}

// Zero overwrites a secret byte slice after use where practical.
func Zero(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}
