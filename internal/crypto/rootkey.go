package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"golang.org/x/crypto/hkdf"
)

// RootKeySize is the byte length of a hosted-tier root key: 256 bits drawn
// from crypto/rand on the first device.
const RootKeySize = 32

// rootKeyIdentityInfo is the HKDF info string that binds the derived age
// identity to this purpose. Changing it would change every recipient derived
// from every existing root key, so it is versioned and never reused.
const rootKeyIdentityInfo = "reinstate/root-key/age-x25519-identity/v1"

// NewRootKey draws a fresh random root key.
func NewRootKey() ([]byte, error) {
	key := make([]byte, RootKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate root key: %w", err)
	}
	return key, nil
}

// RootKeyIdentity derives the age X25519 identity that seals and opens
// envelopes for one root key. The derivation is deterministic (HKDF-SHA256
// over the root key, then the 32-byte output used as the X25519 scalar), so
// every device holding the same root key computes the same identity and the
// same recipient without any key material crossing the wire.
func RootKeyIdentity(rootKey []byte) (*age.X25519Identity, error) {
	if len(rootKey) != RootKeySize {
		return nil, fmt.Errorf("root key must be %d bytes, got %d", RootKeySize, len(rootKey))
	}
	scalar := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, rootKey, nil, []byte(rootKeyIdentityInfo)), scalar); err != nil {
		return nil, fmt.Errorf("derive root key identity: %w", err)
	}
	defer Zero(scalar)
	encoded, err := bech32Encode("AGE-SECRET-KEY-", scalar)
	if err != nil {
		return nil, err
	}
	identity, err := age.ParseX25519Identity(strings.ToUpper(encoded))
	if err != nil {
		return nil, fmt.Errorf("derive root key identity: %w", err)
	}
	return identity, nil
}

// RootKeyProvider is the hosted-tier key model: envelopes are sealed to the
// age identity derived from the current key generation's root key and opened
// with the identities of every key generation this device can read. The root
// key itself is handed over by the keyring, never typed and never stored in
// config.
type RootKeyProvider struct {
	current *age.X25519Identity
	all     []age.Identity
}

// NewRootKeyProvider builds a provider from the current generation's root key
// plus any earlier generations still needed to read older objects. Earlier
// keys only ever open; new envelopes are sealed to the current one.
func NewRootKeyProvider(current []byte, earlier ...[]byte) (*RootKeyProvider, error) {
	identity, err := RootKeyIdentity(current)
	if err != nil {
		return nil, err
	}
	provider := &RootKeyProvider{current: identity, all: []age.Identity{identity}}
	for _, key := range earlier {
		previous, err := RootKeyIdentity(key)
		if err != nil {
			return nil, err
		}
		provider.all = append(provider.all, previous)
	}
	return provider, nil
}

// Recipients implements KeyProvider.
func (p *RootKeyProvider) Recipients() ([]age.Recipient, error) {
	if p == nil || p.current == nil {
		return nil, fmt.Errorf("root key provider has no current key generation")
	}
	return []age.Recipient{p.current.Recipient()}, nil
}

// Identities implements KeyProvider.
func (p *RootKeyProvider) Identities() ([]age.Identity, error) {
	if p == nil || len(p.all) == 0 {
		return nil, fmt.Errorf("root key provider has no key generations")
	}
	return append([]age.Identity(nil), p.all...), nil
}

// Recipient returns the current generation's public recipient string, the
// only root-key derivative that is safe to display.
func (p *RootKeyProvider) Recipient() string {
	if p == nil || p.current == nil {
		return ""
	}
	return p.current.Recipient().String()
}

// bech32Encode is the standard BIP-173 encoding age uses for its key strings.
// age does not export a constructor from raw scalar bytes, so the derived
// scalar is wrapped in the same encoding ParseX25519Identity expects.
func bech32Encode(hrp string, data []byte) (string, error) {
	values, err := bech32ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	hrp = strings.ToLower(hrp)
	var out strings.Builder
	out.WriteString(hrp)
	out.WriteString("1")
	for _, v := range values {
		out.WriteByte(bech32Charset[v])
	}
	for _, v := range bech32Checksum(hrp, values) {
		out.WriteByte(bech32Charset[v])
	}
	return out.String(), nil
}

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var bech32Generator = [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

func bech32Polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(v)
		for i := range 5 {
			if (top>>i)&1 == 1 {
				chk ^= bech32Generator[i]
			}
		}
	}
	return chk
}

func bech32HRPExpand(hrp string) []byte {
	out := make([]byte, 0, len(hrp)*2+1)
	for i := range len(hrp) {
		out = append(out, hrp[i]>>5)
	}
	out = append(out, 0)
	for i := range len(hrp) {
		out = append(out, hrp[i]&31)
	}
	return out
}

func bech32Checksum(hrp string, data []byte) []byte {
	values := append(bech32HRPExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	mod := bech32Polymod(values) ^ 1
	out := make([]byte, 6)
	for i := range out {
		out[i] = byte(mod>>(5*(5-i))) & 31
	}
	return out
}

func bech32ConvertBits(data []byte, from, to byte, pad bool) ([]byte, error) {
	var out []byte
	acc := uint32(0)
	bits := byte(0)
	maxv := byte(1<<to - 1)
	for _, value := range data {
		if value>>from != 0 {
			return nil, fmt.Errorf("bech32: value out of range")
		}
		acc = acc<<from | uint32(value)
		bits += from
		for bits >= to {
			bits -= to
			out = append(out, byte(acc>>bits)&maxv)
		}
	}
	if pad && bits > 0 {
		out = append(out, byte(acc<<(to-bits))&maxv)
	}
	return out, nil
}
