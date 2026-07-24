package doctor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// SelfTest runs a synthetic encryption + atomic write check using only temp data.
// It never reads real vendor sessions or credentials.
func SelfTest(home string) error {
	dir := filepath.Join(home, "cache", "selftest")
	if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
		// fall back to system temp if home not writable
		dir = filepath.Join(os.TempDir(), "reinstate-selftest")
		if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
			return err
		}
	}
	plain := []byte("reinstate-synthetic-self-test-payload-v1")
	pass := "test-passphrase-not-real"
	var buf bytes.Buffer
	if err := crypto.Encrypt(bytes.NewReader(plain), &buf, pass); err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	cipher := buf.Bytes()
	if bytes.Contains(cipher, plain) {
		return fmt.Errorf("ciphertext contains plaintext")
	}
	var out bytes.Buffer
	if err := crypto.Decrypt(bytes.NewReader(cipher), &out, pass); err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		return fmt.Errorf("round-trip mismatch")
	}
	path := filepath.Join(dir, "probe.bin")
	if err := fsx.WriteFileAtomic(path, cipher, 0o600); err != nil {
		return err
	}
	_ = os.Remove(path)
	return nil
}
