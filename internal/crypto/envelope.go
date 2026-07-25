// Package crypto implements client-side age encryption for Reinstate.
package crypto

import (
	"fmt"
	"io"

	"filippo.io/age"
)

// Encrypt streams plaintext through age scrypt passphrase encryption.
func Encrypt(r io.Reader, w io.Writer, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("empty passphrase")
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	wc, err := age.Encrypt(w, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(wc, r); err != nil {
		_ = wc.Close()
		return err
	}
	return wc.Close()
}

// Decrypt streams age ciphertext with a passphrase.
func Decrypt(r io.Reader, w io.Writer, passphrase string) error {
	rc, err := DecryptReader(r, passphrase)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rc)
	return err
}

// DecryptReader authenticates an age stream and returns its plaintext reader.
// Callers can consume large payloads without buffering the entire plaintext.
func DecryptReader(r io.Reader, passphrase string) (io.Reader, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("empty passphrase")
	}
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	rc, err := age.Decrypt(r, identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return rc, nil
}
