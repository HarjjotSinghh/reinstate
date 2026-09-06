package doctor

import (
	"io"
	"sync/atomic"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/crypto/cryptotest"
)

// fastAgeEnvelopeCodec preserves the real age format and decrypt path while
// reducing only the deterministic test's scrypt work factor.
type fastAgeEnvelopeCodec struct {
	encryptions atomic.Int64
}

func (c *fastAgeEnvelopeCodec) Encrypt(source io.Reader, dest io.Writer, keys crypto.KeyProvider) error {
	c.encryptions.Add(1)
	return crypto.Seal(source, dest, cryptotest.FastScrypt(keys))
}

func (*fastAgeEnvelopeCodec) DecryptReader(source io.Reader, keys crypto.KeyProvider) (io.Reader, error) {
	return crypto.OpenReader(source, keys)
}
