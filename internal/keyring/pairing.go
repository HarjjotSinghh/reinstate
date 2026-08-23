package keyring

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"filippo.io/age"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// Pairing (device approval) moves the root key from an enrolled device to a
// joining device through the control plane, which relays ciphertext only.
// The joining device shows a short code; the person types it on the
// enrolled device. Both sides derive the same wrapping key from the code
// with argon2id and a per-request salt; the code itself is never sent
// anywhere. See docs/hop.md "Pairing" for the full argument.
//
// Code shape: three groups of four Crockford base32 characters carrying 60
// random bits, plus a fourth group that is a 20-bit checksum, so a typo is
// caught before a key derivation is attempted.
const (
	pairingDataGroups  = 3
	pairingTotalGroups = 4
	pairingDataChars   = recoveryGroupLen * pairingDataGroups
	pairingTotalChars  = recoveryGroupLen * pairingTotalGroups
	pairingDataBits    = pairingDataChars * 5
	pairingDataBytes   = (pairingDataBits + 7) / 8
	pairingSaltSize    = 16
)

// PairingCodeFormat is the human description printed in help text.
const PairingCodeFormat = "XXXX-XXXX-XXXX-XXXX"

const (
	pairingBindInfo    = "reinstate/pairing/v1/bind"
	pairingPayloadInfo = "reinstate/pairing/v1/payload"
)

// ErrPairingMismatch reports a payload or binding that was not produced
// with this code (a wrong code, or a control plane that altered the
// request).
var ErrPairingMismatch = errors.New("pairing: code does not match this request")

// Pairing is one side's view of a pairing: the canonical code, the salt
// published with the request, and the wrapping key derived from both.
type Pairing struct {
	Code string
	Salt []byte
	key  []byte
}

// NewPairing draws a fresh code and salt for a joining device.
func NewPairing() (*Pairing, error) {
	raw := make([]byte, pairingDataBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return nil, fmt.Errorf("generate pairing code: %w", err)
	}
	data := encodeCrockford(raw, pairingDataChars)
	code := formatRecoveryCode(data + pairingChecksum(data))
	salt := make([]byte, pairingSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate pairing salt: %w", err)
	}
	return &Pairing{Code: code, Salt: salt, key: derivePairingKey(code, salt)}, nil
}

// PairingFromCode is the approving side: the code as typed and the salt
// the joining device published.
func PairingFromCode(typed string, salt []byte) (*Pairing, error) {
	code, err := NormalizePairingCode(typed)
	if err != nil {
		return nil, err
	}
	if len(salt) != pairingSaltSize {
		return nil, fmt.Errorf("pairing: salt must be %d bytes, got %d", pairingSaltSize, len(salt))
	}
	return &Pairing{Code: code, Salt: salt, key: derivePairingKey(code, salt)}, nil
}

// Zero wipes the derived key.
func (p *Pairing) Zero() {
	crypto.Zero(p.key)
}

// NormalizePairingCode accepts a code as a person would type it (any case,
// with or without separators, O/I/L folded to 0/1) and returns the
// canonical grouped form, or an error when the length or checksum is wrong.
func NormalizePairingCode(typed string) (string, error) {
	code, err := compactCrockford(typed)
	if err != nil {
		return "", fmt.Errorf("pairing code %w", err)
	}
	if len(code) != pairingTotalChars {
		return "", fmt.Errorf("pairing code must have %d groups of %d characters (%s)", pairingTotalGroups, recoveryGroupLen, PairingCodeFormat)
	}
	data, check := code[:pairingDataChars], code[pairingDataChars:]
	if check != pairingChecksum(data) {
		return "", fmt.Errorf("pairing code checksum does not match; check it for typos")
	}
	return formatRecoveryCode(code), nil
}

// Binding ties the joining device's public key to the code: the approving
// device recomputes it from the typed code and refuses a request whose key
// was replaced in transit. The control plane, not knowing the code, can
// neither forge nor verify it.
func (p *Pairing) Binding(publicKey string) string {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(pairingBindInfo))
	mac.Write([]byte{0})
	mac.Write([]byte(publicKey))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyBinding checks a published binding in constant time.
func (p *Pairing) VerifyBinding(publicKey, binding string) bool {
	want, err := base64.StdEncoding.DecodeString(binding)
	if err != nil {
		return false
	}
	got, _ := base64.StdEncoding.DecodeString(p.Binding(publicKey))
	return subtle.ConstantTimeCompare(want, got) == 1
}

// SealRootKey wraps rootKey for the joining device: first to its public
// key (age X25519), then under the code-derived key (XChaCha20-Poly1305)
// with the request id, public key, and key generation as associated data.
// The result is what the approving device posts to the relay.
func (p *Pairing) SealRootKey(rootKey []byte, pairingID string, recipient *age.X25519Recipient, generation int) (string, error) {
	if len(rootKey) != crypto.RootKeySize {
		return "", fmt.Errorf("pairing: root key must be %d bytes", crypto.RootKeySize)
	}
	var inner bytes.Buffer
	w, err := age.Encrypt(&inner, recipient)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(rootKey); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	aead, err := chacha20poly1305.NewX(p.key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, inner.Bytes(), pairingAAD(pairingID, recipient.String(), generation))
	return base64.StdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

// OpenRootKey is the joining device's half of SealRootKey. It returns
// ErrPairingMismatch when the payload was not sealed under this code for
// this request, device key, and generation.
func (p *Pairing) OpenRootKey(payload, pairingID string, identity *age.X25519Identity, generation int) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("pairing: payload is not valid base64")
	}
	aead, err := chacha20poly1305.NewX(p.key)
	if err != nil {
		return nil, err
	}
	if len(raw) < aead.NonceSize() {
		return nil, fmt.Errorf("pairing: payload is truncated")
	}
	inner, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], pairingAAD(pairingID, identity.Recipient().String(), generation))
	if err != nil {
		return nil, ErrPairingMismatch
	}
	r, err := age.Decrypt(bytes.NewReader(inner), identity)
	if err != nil {
		return nil, fmt.Errorf("pairing: payload is not wrapped to this device's key: %w", err)
	}
	key, err := io.ReadAll(io.LimitReader(r, crypto.RootKeySize+1))
	if err != nil {
		return nil, err
	}
	if len(key) != crypto.RootKeySize {
		return nil, fmt.Errorf("pairing: payload holds %d bytes, want %d", len(key), crypto.RootKeySize)
	}
	return key, nil
}

func pairingAAD(pairingID, publicKey string, generation int) []byte {
	return []byte(strings.Join([]string{pairingPayloadInfo, pairingID, publicKey, strconv.Itoa(generation)}, "\x00"))
}

func derivePairingKey(canonicalCode string, salt []byte) []byte {
	return argon2.IDKey([]byte(canonicalCode), salt, argon2Time, argon2Memory, argon2Threads, chacha20poly1305.KeySize)
}

func pairingChecksum(data string) string {
	sum := sha256.Sum256([]byte("reinstate/pairing-code/v1:" + data))
	return encodeCrockford(sum[:], recoveryGroupLen)
}
