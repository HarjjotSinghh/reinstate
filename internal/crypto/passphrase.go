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

// PassphraseFDEnv names the descriptor that carries a BYO passphrase for
// automation.
const PassphraseFDEnv = "REINSTATE_PASSPHRASE_FD"

// RecoveryCodeFDEnv names the descriptor that carries a hosted-tier recovery
// code for automation. It follows the passphrase descriptor pattern exactly:
// never an ordinary environment value, never a flag.
const RecoveryCodeFDEnv = "REINSTATE_RECOVERY_CODE_FD"

// PairingCodeFDEnv names the descriptor that carries a pairing code for
// automation on the approving device. Same pattern: never an ordinary
// environment value, never a flag.
const PairingCodeFDEnv = "REINSTATE_PAIRING_CODE_FD"

// ReadPassphrase reads a passphrase from REINSTATE_PASSPHRASE_FD when
// deliberately configured, otherwise from an interactive terminal with echo
// disabled. Passphrases are never accepted as command arguments or ordinary
// environment values.
func ReadPassphrase(input io.Reader, promptOut io.Writer) ([]byte, error) {
	if secret, configured, err := ReadSecretFD(PassphraseFDEnv); configured {
		return secret, err
	}
	return ReadHiddenSecret(input, promptOut, "Encryption passphrase: ")
}

// ReadSecretFD reads a secret from the file descriptor named by envName.
// configured reports whether the variable was set at all, so callers can fall
// back to an interactive prompt only when automation did not opt in.
func ReadSecretFD(envName string) (secret []byte, configured bool, err error) {
	rawFD := strings.TrimSpace(os.Getenv(envName))
	if rawFD == "" {
		return nil, false, nil
	}
	fd, err := strconv.ParseUint(rawFD, 10, 64)
	if err != nil {
		return nil, true, fmt.Errorf("%s must be a valid file descriptor", envName)
	}
	file, err := duplicatePassphraseFD(uintptr(fd))
	if err != nil || file == nil {
		return nil, true, fmt.Errorf("%s is unavailable", envName)
	}
	// Read and close only a duplicate. A caller that owns the configured
	// descriptor may keep its wrapper alive, so closing the original handle
	// here can close an unrelated reused Windows handle later.
	defer func() { _ = file.Close() }()
	secret, err = readBoundedSecret(file)
	return secret, true, err
}

// ReadHiddenSecret reads from a real terminal with echo disabled.
func ReadHiddenSecret(input io.Reader, promptOut io.Writer, prompt string) ([]byte, error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return nil, fmt.Errorf("secret input requires an interactive terminal or a secret file descriptor")
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
