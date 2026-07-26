package cli

import (
	"io"
	"sync/atomic"

	"filippo.io/age"
)

// fastAgeEnvelopeCodec keeps the real age envelope format and decrypt path
// while reducing only the test scrypt cost.
type fastAgeEnvelopeCodec struct {
	encryptions atomic.Int64
}

func (c *fastAgeEnvelopeCodec) Encrypt(source io.Reader, dest io.Writer, passphrase string) error {
	c.encryptions.Add(1)
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	recipient.SetWorkFactor(1)
	writer, err := age.Encrypt(dest, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, source); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func (*fastAgeEnvelopeCodec) DecryptReader(source io.Reader, passphrase string) (io.Reader, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	return age.Decrypt(source, identity)
}
