// Package keyring implements the hosted-tier keyring: the per-account object
// that holds the root key wrapped for each enrolled device and under the
// recovery code (ADR 0002, ADR 0004).
//
// The keyring is the only key-related artefact that reaches storage. It
// carries ciphertext and public keys only: the root key is unwrapped on a
// device that holds either an enrolled device identity or the recovery code,
// and never appears in it in plaintext.
package keyring

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"filippo.io/age"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// SchemaVersion is the keyring object format version.
const SchemaVersion = 1

// ObjectName is the keyring object's name inside the profile prefix.
const ObjectName = "keyring.v1.json"

// Argon2id parameters for the recovery-code wrap. Memory-hard so an offline
// guess against a leaked keyring costs real hardware per attempt; recorded in
// the object so the parameters can be raised later without breaking existing
// wraps.
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // KiB
	argon2Threads = 4
	argon2Salt    = 16
)

const recoveryKDFName = "argon2id"

// Errors callers branch on.
var (
	ErrDeviceNotEnrolled = errors.New("keyring: this device is not enrolled")
	ErrRecoveryMismatch  = errors.New("keyring: recovery code does not match this keyring")
	ErrDeviceExists      = errors.New("keyring: device already enrolled")
)

// Keyring is the wire shape of the keyring object.
type Keyring struct {
	SchemaVersion     int          `json:"schema_version"`
	ProfileID         string       `json:"profile_id"`
	CurrentGeneration int          `json:"current_generation"`
	Generations       []Generation `json:"generations"`
}

// Generation is one root key's lifetime. A new generation starts when a
// device is revoked; earlier generations stay so older objects remain readable.
type Generation struct {
	Number    int          `json:"number"`
	CreatedAt string       `json:"created_at"`
	Recipient string       `json:"recipient"`
	Recovery  RecoveryWrap `json:"recovery"`
	Devices   []DeviceWrap `json:"devices"`
}

// RecoveryWrap is the root key wrapped under a key derived from the recovery
// code. Only the derivation parameters and ciphertext are stored.
type RecoveryWrap struct {
	KDF       string `json:"kdf"`
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
	Salt      string `json:"salt"`
	Wrap      string `json:"wrap"`
}

// DeviceWrap is the root key wrapped to one device's public key.
type DeviceWrap struct {
	DeviceID   string `json:"device_id"`
	PublicKey  string `json:"public_key"`
	EnrolledAt string `json:"enrolled_at"`
	Wrap       string `json:"wrap"`
}

// New builds the first generation of a keyring: the root key wrapped for the
// first device and under the recovery code. recoveryCode must already be in
// canonical form (see NormalizeRecoveryCode).
func New(profileID string, rootKey []byte, recoveryCode string, deviceID string, device *age.X25519Identity, now time.Time) (*Keyring, error) {
	if profileID == "" || deviceID == "" || device == nil {
		return nil, fmt.Errorf("keyring: profile, device id, and device identity are required")
	}
	if _, err := crypto.RootKeyIdentity(rootKey); err != nil {
		return nil, err
	}
	recovery, err := wrapUnderRecoveryCode(rootKey, recoveryCode)
	if err != nil {
		return nil, err
	}
	deviceWrap, err := wrapForDevice(rootKey, deviceID, device.Recipient(), now)
	if err != nil {
		return nil, err
	}
	identity, _ := crypto.RootKeyIdentity(rootKey)
	return &Keyring{
		SchemaVersion:     SchemaVersion,
		ProfileID:         profileID,
		CurrentGeneration: 1,
		Generations: []Generation{{
			Number:    1,
			CreatedAt: now.UTC().Format(time.RFC3339),
			Recipient: identity.Recipient().String(),
			Recovery:  recovery,
			Devices:   []DeviceWrap{deviceWrap},
		}},
	}, nil
}

// Parse decodes and validates a keyring object.
func Parse(raw []byte) (*Keyring, error) {
	var k Keyring
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("keyring: invalid object: %w", err)
	}
	if k.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("keyring: unsupported schema_version %d (want %d)", k.SchemaVersion, SchemaVersion)
	}
	if k.ProfileID == "" || len(k.Generations) == 0 {
		return nil, fmt.Errorf("keyring: profile_id and at least one generation are required")
	}
	if k.current() == nil {
		return nil, fmt.Errorf("keyring: current_generation %d is not present", k.CurrentGeneration)
	}
	return &k, nil
}

// Marshal encodes the keyring for storage.
func (k *Keyring) Marshal() ([]byte, error) {
	raw, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func (k *Keyring) current() *Generation {
	for i := range k.Generations {
		if k.Generations[i].Number == k.CurrentGeneration {
			return &k.Generations[i]
		}
	}
	return nil
}

// CurrentRecipient is the current generation's root-key recipient, the
// public half a joining device checks a received root key against.
func (k *Keyring) CurrentRecipient() string {
	if g := k.current(); g != nil {
		return g.Recipient
	}
	return ""
}

// DevicePublicKey returns the public key the current generation lists for
// deviceID, or "" when the device is not enrolled.
func (k *Keyring) DevicePublicKey(deviceID string) string {
	if g := k.current(); g != nil {
		for _, d := range g.Devices {
			if d.DeviceID == deviceID {
				return d.PublicKey
			}
		}
	}
	return ""
}

// DeviceCount reports how many devices are enrolled in the current generation.
func (k *Keyring) DeviceCount() int {
	if g := k.current(); g != nil {
		return len(g.Devices)
	}
	return 0
}

// HasDevice reports whether deviceID holds a wrap in the current generation.
func (k *Keyring) HasDevice(deviceID string) bool {
	g := k.current()
	if g == nil {
		return false
	}
	for _, d := range g.Devices {
		if d.DeviceID == deviceID {
			return true
		}
	}
	return false
}

// UnwrapWithRecoveryCode recovers the current generation's root key from a
// recovery code. A wrong code fails closed with ErrRecoveryMismatch and
// nothing else is learned.
func (k *Keyring) UnwrapWithRecoveryCode(recoveryCode string) ([]byte, error) {
	g := k.current()
	if g == nil {
		return nil, fmt.Errorf("keyring: no current generation")
	}
	return unwrapWithRecoveryCode(g.Recovery, recoveryCode)
}

// UnwrapForDevice recovers the current generation's root key, plus any
// earlier generations the device can read, using the device's identity.
// Returns ErrDeviceNotEnrolled when the current generation has no wrap for
// deviceID.
func (k *Keyring) UnwrapForDevice(deviceID string, device *age.X25519Identity) (current []byte, earlier [][]byte, err error) {
	g := k.current()
	if g == nil {
		return nil, nil, fmt.Errorf("keyring: no current generation")
	}
	current, err = unwrapDevice(g, deviceID, device)
	if err != nil {
		return nil, nil, err
	}
	for i := range k.Generations {
		if k.Generations[i].Number == k.CurrentGeneration {
			continue
		}
		key, err := unwrapDevice(&k.Generations[i], deviceID, device)
		if errors.Is(err, ErrDeviceNotEnrolled) {
			continue
		}
		if err != nil {
			crypto.Zero(current)
			return nil, nil, err
		}
		earlier = append(earlier, key)
	}
	return current, earlier, nil
}

// Enrol appends a wrap of rootKey for a new device to the current generation.
// rootKey must be the current generation's key (it is checked against the
// recorded recipient so a stale key can never be wrapped as current).
func (k *Keyring) Enrol(rootKey []byte, deviceID string, recipient *age.X25519Recipient, now time.Time) error {
	g := k.current()
	if g == nil {
		return fmt.Errorf("keyring: no current generation")
	}
	if deviceID == "" || recipient == nil {
		return fmt.Errorf("keyring: device id and public key are required")
	}
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return err
	}
	if identity.Recipient().String() != g.Recipient {
		return fmt.Errorf("keyring: root key does not belong to generation %d", g.Number)
	}
	if k.HasDevice(deviceID) {
		return ErrDeviceExists
	}
	wrap, err := wrapForDevice(rootKey, deviceID, recipient, now)
	if err != nil {
		return err
	}
	g.Devices = append(g.Devices, wrap)
	return nil
}

// Unenrol removes the current generation's wrap for deviceID, but only when
// the wrap was made for publicKey: an approving device uses it to roll back
// a wrap it appended itself when the relay then refused (request expired or
// already decided), and the key check keeps it from ever touching a wrap a
// different approval wrote for the same device id. Reports whether a wrap
// was removed.
func (k *Keyring) Unenrol(deviceID, publicKey string) bool {
	g := k.current()
	if g == nil || publicKey == "" {
		return false
	}
	for i, d := range g.Devices {
		if d.DeviceID == deviceID && d.PublicKey == publicKey {
			g.Devices = append(g.Devices[:i:i], g.Devices[i+1:]...)
			return true
		}
	}
	return false
}

func unwrapDevice(g *Generation, deviceID string, device *age.X25519Identity) ([]byte, error) {
	for _, d := range g.Devices {
		if d.DeviceID != deviceID {
			continue
		}
		if d.PublicKey != device.Recipient().String() {
			return nil, fmt.Errorf("%w: the keyring lists a different public key for this device", ErrDeviceNotEnrolled)
		}
		cipher, err := base64.StdEncoding.DecodeString(d.Wrap)
		if err != nil {
			return nil, fmt.Errorf("keyring: device wrap is not valid base64")
		}
		r, err := age.Decrypt(bytes.NewReader(cipher), device)
		if err != nil {
			return nil, fmt.Errorf("keyring: unwrap root key for device: %w", err)
		}
		key, err := io.ReadAll(io.LimitReader(r, crypto.RootKeySize+1))
		if err != nil {
			return nil, err
		}
		if len(key) != crypto.RootKeySize {
			return nil, fmt.Errorf("keyring: device wrap holds %d bytes, want %d", len(key), crypto.RootKeySize)
		}
		return key, nil
	}
	return nil, ErrDeviceNotEnrolled
}

func wrapForDevice(rootKey []byte, deviceID string, recipient *age.X25519Recipient, now time.Time) (DeviceWrap, error) {
	var cipher bytes.Buffer
	w, err := age.Encrypt(&cipher, recipient)
	if err != nil {
		return DeviceWrap{}, err
	}
	if _, err := w.Write(rootKey); err != nil {
		return DeviceWrap{}, err
	}
	if err := w.Close(); err != nil {
		return DeviceWrap{}, err
	}
	return DeviceWrap{
		DeviceID:   deviceID,
		PublicKey:  recipient.String(),
		EnrolledAt: now.UTC().Format(time.RFC3339),
		Wrap:       base64.StdEncoding.EncodeToString(cipher.Bytes()),
	}, nil
}

func wrapUnderRecoveryCode(rootKey []byte, recoveryCode string) (RecoveryWrap, error) {
	if _, err := NormalizeRecoveryCode(recoveryCode); err != nil {
		return RecoveryWrap{}, err
	}
	salt := make([]byte, argon2Salt)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return RecoveryWrap{}, err
	}
	wrap := RecoveryWrap{
		KDF:       recoveryKDFName,
		Time:      argon2Time,
		MemoryKiB: argon2Memory,
		Threads:   argon2Threads,
		Salt:      base64.StdEncoding.EncodeToString(salt),
	}
	key := deriveRecoveryKey(recoveryCode, salt, wrap)
	defer crypto.Zero(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return RecoveryWrap{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return RecoveryWrap{}, err
	}
	sealed := aead.Seal(nil, nonce, rootKey, []byte(recoveryAAD))
	wrap.Wrap = base64.StdEncoding.EncodeToString(append(nonce, sealed...))
	return wrap, nil
}

const recoveryAAD = "reinstate/keyring/recovery-wrap/v1"

func unwrapWithRecoveryCode(wrap RecoveryWrap, recoveryCode string) ([]byte, error) {
	canonical, err := NormalizeRecoveryCode(recoveryCode)
	if err != nil {
		return nil, err
	}
	if wrap.KDF != recoveryKDFName {
		return nil, fmt.Errorf("keyring: unsupported recovery kdf %q", wrap.KDF)
	}
	salt, err := base64.StdEncoding.DecodeString(wrap.Salt)
	if err != nil {
		return nil, fmt.Errorf("keyring: recovery salt is not valid base64")
	}
	cipher, err := base64.StdEncoding.DecodeString(wrap.Wrap)
	if err != nil {
		return nil, fmt.Errorf("keyring: recovery wrap is not valid base64")
	}
	key := deriveRecoveryKey(canonical, salt, wrap)
	defer crypto.Zero(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(cipher) < aead.NonceSize() {
		return nil, fmt.Errorf("keyring: recovery wrap is truncated")
	}
	rootKey, err := aead.Open(nil, cipher[:aead.NonceSize()], cipher[aead.NonceSize():], []byte(recoveryAAD))
	if err != nil {
		return nil, ErrRecoveryMismatch
	}
	if len(rootKey) != crypto.RootKeySize {
		return nil, fmt.Errorf("keyring: recovery wrap holds %d bytes, want %d", len(rootKey), crypto.RootKeySize)
	}
	return rootKey, nil
}

func deriveRecoveryKey(canonicalCode string, salt []byte, wrap RecoveryWrap) []byte {
	return argon2.IDKey([]byte(canonicalCode), salt, wrap.Time, wrap.MemoryKiB, wrap.Threads, chacha20poly1305.KeySize)
}
