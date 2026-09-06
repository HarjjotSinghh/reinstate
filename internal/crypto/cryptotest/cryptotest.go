// Package cryptotest provides deterministic, low-cost key providers for
// tests. It keeps the real age envelope format and decrypt path and lowers
// only the scrypt work factor, so production code never carries a knob that
// could weaken key derivation.
package cryptotest

import (
	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// fastWorkFactor is the lowest scrypt cost age accepts; envelopes sealed with
// it still open with the production PassphraseProvider.
const fastWorkFactor = 1

// Passphrase returns the BYO passphrase key model with only the scrypt cost
// lowered for tests.
func Passphrase(passphrase string) crypto.KeyProvider {
	return FastScrypt(crypto.NewPassphraseProvider(passphrase))
}

// FastScrypt wraps keys so that any scrypt recipient it returns seals at the
// lowest work factor. Identities and non-scrypt recipients pass through
// untouched, so the wrapper is transparent for other key models.
func FastScrypt(keys crypto.KeyProvider) crypto.KeyProvider {
	return fastScrypt{keys: keys}
}

type fastScrypt struct{ keys crypto.KeyProvider }

func (f fastScrypt) Recipients() ([]age.Recipient, error) {
	recipients, err := f.keys.Recipients()
	if err != nil {
		return nil, err
	}
	for _, r := range recipients {
		if scrypt, ok := r.(*age.ScryptRecipient); ok {
			scrypt.SetWorkFactor(fastWorkFactor)
		}
	}
	return recipients, nil
}

func (f fastScrypt) Identities() ([]age.Identity, error) {
	return f.keys.Identities()
}
