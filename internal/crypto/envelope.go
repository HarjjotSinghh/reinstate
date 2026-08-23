// Package crypto implements client-side age encryption for Reinstate.
package crypto

import "io"

// Encrypt streams plaintext through age scrypt passphrase encryption. It is
// the passphrase convenience form of Seal.
func Encrypt(r io.Reader, w io.Writer, passphrase string) error {
	return Seal(r, w, NewPassphraseProvider(passphrase))
}

// Decrypt streams age ciphertext with a passphrase. It is the passphrase
// convenience form of Open.
func Decrypt(r io.Reader, w io.Writer, passphrase string) error {
	return Open(r, w, NewPassphraseProvider(passphrase))
}

// DecryptReader authenticates an age stream and returns its plaintext reader.
// It is the passphrase convenience form of OpenReader.
func DecryptReader(r io.Reader, passphrase string) (io.Reader, error) {
	return OpenReader(r, NewPassphraseProvider(passphrase))
}
