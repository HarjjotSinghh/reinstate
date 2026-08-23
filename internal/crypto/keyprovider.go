package crypto

import (
	"fmt"
	"io"

	"filippo.io/age"
)

// KeyProvider supplies the age recipients that seal an envelope and the age
// identities that open one. Every envelope writer and reader in Reinstate
// goes through this seam so the key model (a typed passphrase for BYO
// storage, a device-held root key for hosted storage) stays separate from
// sync, manifest, and conflict logic.
//
// Implementations must be safe for concurrent use.
type KeyProvider interface {
	// Recipients returns the recipients every new envelope is sealed to.
	Recipients() ([]age.Recipient, error)
	// Identities returns the identities tried, in order, when opening an
	// envelope.
	Identities() ([]age.Identity, error)
}

// PassphraseProvider is the BYO storage key model: a single age scrypt
// recipient and identity derived from a user-typed passphrase. It produces
// the same envelope bytes and accepts the same envelopes as Reinstate did
// before the provider seam existed.
type PassphraseProvider struct {
	passphrase string
	workFactor int
}

// NewPassphraseProvider wraps a passphrase. An empty passphrase is rejected
// lazily, when the provider is first asked for a recipient or identity.
func NewPassphraseProvider(passphrase string) *PassphraseProvider {
	return &PassphraseProvider{passphrase: passphrase}
}

// WithWorkFactor returns a copy whose scrypt recipient uses 2^workFactor
// iterations instead of age's default. Only deterministic tests should lower
// it; production callers keep the default.
func (p *PassphraseProvider) WithWorkFactor(workFactor int) *PassphraseProvider {
	return &PassphraseProvider{passphrase: p.passphrase, workFactor: workFactor}
}

// Recipients implements KeyProvider.
func (p *PassphraseProvider) Recipients() ([]age.Recipient, error) {
	if p == nil || p.passphrase == "" {
		return nil, fmt.Errorf("empty passphrase")
	}
	recipient, err := age.NewScryptRecipient(p.passphrase)
	if err != nil {
		return nil, err
	}
	if p.workFactor > 0 {
		recipient.SetWorkFactor(p.workFactor)
	}
	return []age.Recipient{recipient}, nil
}

// Identities implements KeyProvider.
func (p *PassphraseProvider) Identities() ([]age.Identity, error) {
	if p == nil || p.passphrase == "" {
		return nil, fmt.Errorf("empty passphrase")
	}
	identity, err := age.NewScryptIdentity(p.passphrase)
	if err != nil {
		return nil, err
	}
	return []age.Identity{identity}, nil
}

// Seal streams plaintext into an age envelope sealed to the provider's
// recipients.
func Seal(r io.Reader, w io.Writer, keys KeyProvider) error {
	if keys == nil {
		return fmt.Errorf("key provider required")
	}
	recipients, err := keys.Recipients()
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return fmt.Errorf("key provider returned no recipients")
	}
	wc, err := age.Encrypt(w, recipients...)
	if err != nil {
		return err
	}
	if _, err := io.Copy(wc, r); err != nil {
		_ = wc.Close()
		return err
	}
	return wc.Close()
}

// Open streams an age envelope's plaintext into w using the provider's
// identities.
func Open(r io.Reader, w io.Writer, keys KeyProvider) error {
	rc, err := OpenReader(r, keys)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rc)
	return err
}

// OpenReader authenticates an age envelope with the provider's identities and
// returns its plaintext reader. Callers can consume large payloads without
// buffering the entire plaintext.
func OpenReader(r io.Reader, keys KeyProvider) (io.Reader, error) {
	if keys == nil {
		return nil, fmt.Errorf("key provider required")
	}
	identities, err := keys.Identities()
	if err != nil {
		return nil, err
	}
	if len(identities) == 0 {
		return nil, fmt.Errorf("key provider returned no identities")
	}
	rc, err := age.Decrypt(r, identities...)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return rc, nil
}
