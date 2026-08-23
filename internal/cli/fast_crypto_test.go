package cli

import (
	"io"
	"sync/atomic"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// fastAgeEnvelopeCodec keeps the real age envelope format and decrypt path
// while reducing only the test scrypt cost.
type fastAgeEnvelopeCodec struct {
	encryptions atomic.Int64
}

func (c *fastAgeEnvelopeCodec) Encrypt(source io.Reader, dest io.Writer, keys crypto.KeyProvider) error {
	c.encryptions.Add(1)
	return crypto.Seal(source, dest, fastKeys(keys))
}

func (*fastAgeEnvelopeCodec) DecryptReader(source io.Reader, keys crypto.KeyProvider) (io.Reader, error) {
	return crypto.OpenReader(source, keys)
}

// fastKeys lowers only the scrypt cost of a passphrase provider; any other
// provider passes through untouched.
func fastKeys(keys crypto.KeyProvider) crypto.KeyProvider {
	if p, ok := keys.(*crypto.PassphraseProvider); ok {
		return p.WithWorkFactor(1)
	}
	return keys
}
