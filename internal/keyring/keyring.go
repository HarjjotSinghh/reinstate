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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// SchemaVersion is the keyring object format version this package reads and
// writes. It is the only version accepted: there is no compatibility mode,
// because every earlier version is missing a property this one relies on.
//
// Version 1 wrapped the root key with nothing but the key itself. Version 2
// bound every wrap to the profile id and the key generation it belongs to
// (device wraps inside the age payload, the recovery wrap as AEAD associated
// data), so a wrap could not be replayed in another keyring or generation —
// but nothing tied one generation to the next, so a party that could write
// the object could append a generation of its own and every device would
// adopt it. Version 3 chains the generations: each one past the first
// carries a MAC over its own header, keyed by the previous generation's root
// key (see generationChain), so only a holder of generation N's root key can
// write generation N+1. Version 3 also requires every wrap to be bound;
// unbound wraps no longer exist anywhere.
const SchemaVersion = 3

// ObjectName is the keyring object's name inside the profile prefix. The
// name is the object's identity in storage and does not change with the
// schema version; schema_version inside the object does.
const ObjectName = "keyring.v1.json"

// WrapFormatBound is the format recorded on every wrap: one whose payload
// (device) or associated data (recovery) names the profile and generation
// it belongs to. It is the only format this version reads or writes.
const WrapFormatBound = 2

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
	// ErrDeviceKeyMismatch reports a device id that is listed, but with a
	// public key other than the one this machine holds. It is always
	// returned wrapped together with ErrDeviceNotEnrolled.
	ErrDeviceKeyMismatch = errors.New("keyring: the keyring lists this device with a different public key")
	ErrRecoveryMismatch  = errors.New("keyring: recovery code does not match this keyring")
	ErrDeviceExists      = errors.New("keyring: device already enrolled")
	// ErrStaleRootKey reports a root key that is not the current
	// generation's: the caller's view of the keyring is behind a rollover.
	ErrStaleRootKey = errors.New("keyring: root key does not belong to the current generation")
	// ErrAlreadyRevoked reports a revocation target with no wrap in the
	// current generation; it is returned wrapped together with
	// ErrDeviceNotEnrolled.
	ErrAlreadyRevoked = errors.New("keyring: device already has no wrap in the current generation")
	// ErrSelfRevoke reports an attempt to revoke the device doing the
	// revoking, which would leave it unable to read the new generation.
	ErrSelfRevoke = errors.New("keyring: a device cannot revoke itself")
	// ErrUnauthenticatedGeneration reports a key generation this reader
	// must not adopt: its chain does not check out against the previous
	// generation's root key, or the reader holds no key that could check it.
	ErrUnauthenticatedGeneration = errors.New("keyring: key generation is not authenticated by the generation before it")
)

// Keyring is the wire shape of the keyring object.
type Keyring struct {
	SchemaVersion     int          `json:"schema_version"`
	ProfileID         string       `json:"profile_id"`
	CurrentGeneration int          `json:"current_generation"`
	Generations       []Generation `json:"generations"`
}

// Generation is one root key's lifetime. A new generation starts when a
// device is revoked; earlier generations stay, untouched, so older objects
// remain readable by every device that held them.
type Generation struct {
	Number    int          `json:"number"`
	CreatedAt string       `json:"created_at"`
	Recipient string       `json:"recipient"`
	Recovery  RecoveryWrap `json:"recovery"`
	Devices   []DeviceWrap `json:"devices"`
	// Revoked lists the devices whose revocation started this generation.
	// It is a record for people and status output; the key model is the
	// absence of a wrap.
	Revoked []Revocation `json:"revoked,omitempty"`
	// Chain authenticates this generation against the one before it: a MAC
	// over this generation's header — number, created_at, recipient, the
	// revocations that started it, the profile id, and the number and
	// recipient of the generation it follows — keyed by the previous
	// generation's root key. Only a device that could open the previous
	// generation can produce it, which is precisely what a revoked device
	// can no longer do. Generation 1 has no predecessor and carries none;
	// what anchors generation 1 is outside the object (see VerifyChain).
	Chain string `json:"chain,omitempty"`
}

// Revocation records why a generation exists.
type Revocation struct {
	DeviceID  string `json:"device_id"`
	RevokedAt string `json:"revoked_at"`
	RevokedBy string `json:"revoked_by"`
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
	// Format is WrapFormatBound: the associated data names the profile and
	// generation. No other value is accepted.
	Format int `json:"format,omitempty"`
}

// DeviceWrap is the root key wrapped to one device's public key.
type DeviceWrap struct {
	DeviceID   string `json:"device_id"`
	PublicKey  string `json:"public_key"`
	EnrolledAt string `json:"enrolled_at"`
	Wrap       string `json:"wrap"`
	// Format is WrapFormatBound: the age payload names the profile and
	// generation. No other value is accepted.
	Format int `json:"format,omitempty"`
}

// binding is what a bound wrap is tied to.
type binding struct {
	profileID  string
	generation int
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
	bind := binding{profileID: profileID, generation: 1}
	recovery, err := wrapUnderRecoveryCode(rootKey, recoveryCode, bind)
	if err != nil {
		return nil, err
	}
	deviceWrap, err := wrapForDevice(rootKey, deviceID, device.Recipient(), now, bind)
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

// Parse decodes and structurally validates a keyring object. Structure is
// all it can check: whether the generations are the account's own is a
// cryptographic question, answered by VerifyChain against root keys the
// reader holds, and Parse deliberately does not pretend to answer it.
func Parse(raw []byte) (*Keyring, error) {
	var k Keyring
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("keyring: invalid object: %w", err)
	}
	if k.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("keyring: unsupported schema_version %d (this version reads %d only; earlier formats did not authenticate their key generations and are not read)", k.SchemaVersion, SchemaVersion)
	}
	if k.ProfileID == "" || len(k.Generations) == 0 {
		return nil, fmt.Errorf("keyring: profile_id and at least one generation are required")
	}
	seen := map[int]bool{}
	for _, g := range k.Generations {
		if g.Number <= 0 {
			return nil, fmt.Errorf("keyring: generation number %d is not positive", g.Number)
		}
		if seen[g.Number] {
			return nil, fmt.Errorf("keyring: generation %d appears more than once", g.Number)
		}
		seen[g.Number] = true
		devices := map[string]bool{}
		for _, d := range g.Devices {
			if d.DeviceID == "" {
				return nil, fmt.Errorf("keyring: generation %d lists a device without an id", g.Number)
			}
			if devices[d.DeviceID] {
				return nil, fmt.Errorf("keyring: generation %d lists device %s more than once", g.Number, d.DeviceID)
			}
			devices[d.DeviceID] = true
		}
		for _, w := range append([]int{g.Recovery.Format}, deviceFormats(g.Devices)...) {
			if w != WrapFormatBound {
				return nil, fmt.Errorf("keyring: generation %d holds a wrap of unknown format %d", g.Number, w)
			}
		}
		if err := checkChainShape(g); err != nil {
			return nil, err
		}
	}
	// Generations run 1..n with no gaps. Rollover only ever appends
	// maxGeneration()+1, so a gap means generations were removed or a
	// number was invented; either way the chain from 1 upward is broken
	// and no reader could authenticate what is left.
	for n := 1; n <= len(k.Generations); n++ {
		if !seen[n] {
			return nil, fmt.Errorf("keyring: generation %d is missing from a keyring holding %d generations", n, len(k.Generations))
		}
	}
	if k.current() == nil {
		return nil, fmt.Errorf("keyring: current_generation %d is not present", k.CurrentGeneration)
	}
	return &k, nil
}

// checkChainShape refuses a generation whose chain field is the wrong shape
// for its position: absent on generation 1, and a full-length MAC on every
// generation after it. Whether the MAC is *correct* is VerifyChain's
// question; this only rules out an object no reader could ever check.
func checkChainShape(g Generation) error {
	if g.Number == 1 {
		if g.Chain != "" {
			return fmt.Errorf("keyring: generation 1 carries a chain but has no generation before it")
		}
		return nil
	}
	mac, err := base64.StdEncoding.DecodeString(g.Chain)
	if err != nil {
		return fmt.Errorf("keyring: generation %d has a chain that is not valid base64", g.Number)
	}
	if len(mac) != chainMACSize {
		return fmt.Errorf("keyring: generation %d has a %d-byte chain, want %d", g.Number, len(mac), chainMACSize)
	}
	return nil
}

func deviceFormats(devices []DeviceWrap) []int {
	out := make([]int, 0, len(devices))
	for _, d := range devices {
		out = append(out, d.Format)
	}
	return out
}

// Marshal encodes the keyring for storage.

func (k *Keyring) Marshal() ([]byte, error) {
	k.SchemaVersion = SchemaVersion
	raw, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func (k *Keyring) current() *Generation {
	return k.generation(k.CurrentGeneration)
}

func (k *Keyring) generation(number int) *Generation {
	for i := range k.Generations {
		if k.Generations[i].Number == number {
			return &k.Generations[i]
		}
	}
	return nil
}

func (k *Keyring) bindingFor(g *Generation) binding {
	return binding{profileID: k.ProfileID, generation: g.Number}
}

// generationChainInfo is the HKDF info string that separates the chain MAC
// key from everything else a root key derives (it also derives the age
// identity that seals envelopes). Changing it would invalidate every chain
// ever written, so it is versioned and never reused.
const generationChainInfo = "reinstate/keyring/generation-chain/v1"

// chainMACSize is the chain MAC's length: HMAC-SHA256.
const chainMACSize = sha256.Size

// generationChainKey derives the MAC key for the link out of the generation
// whose root key is rootKey.
func generationChainKey(rootKey []byte) ([]byte, error) {
	if len(rootKey) != crypto.RootKeySize {
		return nil, fmt.Errorf("keyring: root key must be %d bytes, got %d", crypto.RootKeySize, len(rootKey))
	}
	key := make([]byte, chainMACSize)
	if _, err := io.ReadFull(hkdf.New(sha256.New, rootKey, nil, []byte(generationChainInfo)), key); err != nil {
		return nil, fmt.Errorf("keyring: derive generation chain key: %w", err)
	}
	return key, nil
}

// generationChain computes g's chain value: a MAC keyed by prevRootKey, the
// root key of the generation g follows, over every part of g's header a
// reader trusts. Fields are length-prefixed so no two different headers can
// produce the same input.
//
// The device wraps are deliberately not covered. They change after a
// generation is written — a device enrolled later is given a wrap in every
// generation it may read — and re-MACing on every enrolment would need the
// previous generation's key at moments a caller does not have it. They do
// not need covering: a wrap is only ever accepted when the key inside it
// derives the generation's recorded recipient (see unwrapDevice), and the
// recipient is covered here. So appending a working wrap to a generation
// still requires that generation's root key; what an attacker with bucket
// write access can do to the device list is remove or corrupt entries, which
// denies service rather than substituting a key.
func generationChain(profileID string, g, prev *Generation, prevRootKey []byte) (string, error) {
	key, err := generationChainKey(prevRootKey)
	if err != nil {
		return "", err
	}
	defer crypto.Zero(key)
	mac := hmac.New(sha256.New, key)
	var size [8]byte
	write := func(fields ...string) {
		for _, f := range fields {
			binary.BigEndian.PutUint64(size[:], uint64(len(f)))
			mac.Write(size[:])
			mac.Write([]byte(f))
		}
	}
	write(generationChainInfo, profileID,
		strconv.Itoa(g.Number), g.CreatedAt, g.Recipient,
		strconv.Itoa(prev.Number), prev.Recipient,
		strconv.Itoa(len(g.Revoked)))
	for _, r := range g.Revoked {
		write(r.DeviceID, r.RevokedAt, r.RevokedBy)
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// VerifyChain authenticates this keyring's generations against the root keys
// the reader was able to unwrap from it, keyed by generation number. Every
// link it can check, it checks; and the current generation must be one of
// them, so a reader never acts on a generation it cannot authenticate.
//
// What this proves: generation N+1 was written by someone holding generation
// N's root key. A device revoked at generation N never held N's root key —
// that is the whole point of the rollover — so it cannot append a generation
// the rest of the account will adopt, even with full write access to the
// bucket.
//
// What this does not prove: that generation 1 is the account's own. Nothing
// inside the object can say so; a chain forged from a root key of the
// attacker's choosing at generation 1 is self-consistent. Generation 1 is
// anchored from outside — by the recovery code (only its holder can write a
// recovery wrap that opens), by the root key relayed through a pairing
// approval, or by the generation and recipient a device recorded locally the
// last time it read the keyring. Callers must supply one of those anchors;
// see the anchor check in the CLI's account state.
func (k *Keyring) VerifyChain(keys map[int][]byte) error {
	for _, n := range k.GenerationNumbers() {
		prevKey, ok := keys[n-1]
		if n == 1 || !ok {
			continue
		}
		if err := k.verifyGenerationChain(n, prevKey); err != nil {
			return err
		}
	}
	if k.CurrentGeneration == 1 {
		return nil
	}
	if _, ok := keys[k.CurrentGeneration-1]; !ok {
		return fmt.Errorf("%w: generation %d cannot be checked here, because this device holds no root key for generation %d", ErrUnauthenticatedGeneration, k.CurrentGeneration, k.CurrentGeneration-1)
	}
	return nil
}

// verifyGenerationChain checks the single link from n-1 into n, given n-1's
// root key.
func (k *Keyring) verifyGenerationChain(n int, prevRootKey []byte) error {
	g, prev := k.generation(n), k.generation(n-1)
	if g == nil || prev == nil {
		return fmt.Errorf("%w: generation %d has no generation %d before it", ErrUnauthenticatedGeneration, n, n-1)
	}
	want, err := generationChain(k.ProfileID, g, prev, prevRootKey)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(want), []byte(g.Chain)) != 1 {
		return fmt.Errorf("%w: generation %d was not written by a holder of generation %d's root key", ErrUnauthenticatedGeneration, n, n-1)
	}
	return nil
}

// GenerationRecipient is the root-key recipient recorded for one generation,
// or "" when the keyring holds no such generation. A device records the
// current one locally so a later read can tell an appended keyring from a
// replaced one.
func (k *Keyring) GenerationRecipient(number int) string {
	if g := k.generation(number); g != nil {
		return g.Recipient
	}
	return ""
}

// GenerationNumbers lists every generation held, oldest first.
func (k *Keyring) GenerationNumbers() []int {
	out := make([]int, 0, len(k.Generations))
	for _, g := range k.Generations {
		out = append(out, g.Number)
	}
	sort.Ints(out)
	return out
}

// Revocations lists every revocation recorded across all generations,
// oldest generation first.
func (k *Keyring) Revocations() []Revocation {
	var out []Revocation
	for _, n := range k.GenerationNumbers() {
		out = append(out, k.generation(n).Revoked...)
	}
	return out
}

// RevokedDevice reports whether deviceID was revoked in some generation
// and has no wrap in the current one.
func (k *Keyring) RevokedDevice(deviceID string) bool {
	if k.HasDevice(deviceID) {
		return false
	}
	for _, r := range k.Revocations() {
		if r.DeviceID == deviceID {
			return true
		}
	}
	return false
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
	return unwrapWithRecoveryCode(g.Recovery, recoveryCode, k.bindingFor(g))
}

// UnwrapForDevice recovers the current generation's root key, plus the root
// keys of every earlier generation the device can read (newest first),
// using the device's identity. Returns ErrDeviceNotEnrolled when the
// current generation has no wrap for deviceID; when the id is listed under
// a different public key the error also matches ErrDeviceKeyMismatch.
//
// Earlier generations are best effort: one that does not list the device,
// or lists it under a key this machine no longer holds (the device was
// re-enrolled with a fresh key after a rollover), is skipped rather than
// treated as an error, since nothing written under the current generation
// depends on it.
func (k *Keyring) UnwrapForDevice(deviceID string, device *age.X25519Identity) (current []byte, earlier [][]byte, err error) {
	g := k.current()
	if g == nil {
		return nil, nil, fmt.Errorf("keyring: no current generation")
	}
	current, err = unwrapDevice(g, deviceID, device, k.bindingFor(g))
	if err != nil {
		return nil, nil, err
	}
	numbers := k.GenerationNumbers()
	for i := len(numbers) - 1; i >= 0; i-- {
		if numbers[i] == k.CurrentGeneration {
			continue
		}
		eg := k.generation(numbers[i])
		key, err := unwrapDevice(eg, deviceID, device, k.bindingFor(eg))
		if errors.Is(err, ErrDeviceNotEnrolled) {
			continue
		}
		if err != nil {
			crypto.Zero(current)
			for _, e := range earlier {
				crypto.Zero(e)
			}
			return nil, nil, fmt.Errorf("generation %d: %w", eg.Number, err)
		}
		earlier = append(earlier, key)
	}
	return current, earlier, nil
}

// UnwrapGenerations recovers every generation's root key the device can
// read, keyed by generation number. The current generation must open
// (ErrDeviceNotEnrolled otherwise); earlier ones are best effort exactly as
// in UnwrapForDevice. Every value must be zeroed by the caller.
func (k *Keyring) UnwrapGenerations(deviceID string, device *age.X25519Identity) (map[int][]byte, error) {
	current, earlier, err := k.UnwrapForDevice(deviceID, device)
	if err != nil {
		return nil, err
	}
	keys := map[int][]byte{k.CurrentGeneration: current}
	// earlier is newest first; match each key to its generation by the
	// recipient it derives, so the mapping cannot drift from the order.
	for _, key := range earlier {
		n := k.generationOf(key)
		if n == 0 {
			crypto.Zero(key)
			continue
		}
		keys[n] = key
	}
	return keys, nil
}

// UnwrapGenerationsWithRecoveryCode is UnwrapGenerations for the recovery
// code: every generation was wrapped under the same code when it started,
// so a holder of the code reads all of them. The current generation must
// open (ErrRecoveryMismatch otherwise).
func (k *Keyring) UnwrapGenerationsWithRecoveryCode(recoveryCode string) (map[int][]byte, error) {
	current, err := k.UnwrapWithRecoveryCode(recoveryCode)
	if err != nil {
		return nil, err
	}
	keys := map[int][]byte{k.CurrentGeneration: current}
	for _, n := range k.GenerationNumbers() {
		if n == k.CurrentGeneration {
			continue
		}
		g := k.generation(n)
		key, err := unwrapWithRecoveryCode(g.Recovery, recoveryCode, k.bindingFor(g))
		if err != nil {
			continue
		}
		keys[n] = key
	}
	return keys, nil
}

// generationOf finds the generation whose recipient rootKey derives, or 0.
func (k *Keyring) generationOf(rootKey []byte) int {
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return 0
	}
	recipient := identity.Recipient().String()
	for _, g := range k.Generations {
		if g.Recipient == recipient {
			return g.Number
		}
	}
	return 0
}

// ZeroGenerations wipes every key in a map returned by UnwrapGenerations.
func ZeroGenerations(keys map[int][]byte) {
	for _, key := range keys {
		crypto.Zero(key)
	}
}

// Membership is how the current generation relates to one device and the
// key (if any) that machine holds for it.
type Membership int

const (
	// NotListed: the current generation has no wrap for the device id.
	NotListed Membership = iota
	// Enrolled: the current generation wraps the root key to the key the
	// machine holds.
	Enrolled
	// KeyMismatch: the id is listed, but under a different public key than
	// the machine holds; the wrap is useless here and the id is taken.
	KeyMismatch
	// KeyGone: the id is listed, but the machine holds no key at all for it
	// (the OS keyring entry was lost or never written).
	KeyGone
)

func (m Membership) String() string {
	switch m {
	case NotListed:
		return "not listed"
	case Enrolled:
		return "enrolled"
	case KeyMismatch:
		return "listed under a different key"
	case KeyGone:
		return "listed but the device key is gone"
	}
	return fmt.Sprintf("membership(%d)", int(m))
}

// DeviceMembership classifies deviceID against the current generation given
// the identity this machine holds for it (nil when it holds none). It is
// the one place the "listed but the key is gone" and "listed under another
// key" cases are told apart, so every command words them the same way.
func (k *Keyring) DeviceMembership(deviceID string, device *age.X25519Identity) Membership {
	listed := k.DevicePublicKey(deviceID)
	switch {
	case listed == "":
		return NotListed
	case device == nil:
		return KeyGone
	case listed != device.Recipient().String():
		return KeyMismatch
	}
	return Enrolled
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
	if err := k.checkCurrentRootKey(rootKey); err != nil {
		return err
	}
	if k.HasDevice(deviceID) {
		return ErrDeviceExists
	}
	wrap, err := wrapForDevice(rootKey, deviceID, recipient, now, k.bindingFor(g))
	if err != nil {
		return err
	}
	g.Devices = append(g.Devices, wrap)
	return nil
}

// EnrolInto gives deviceID a wrap in an earlier generation so a device
// enrolled after a rollover can read what was written before it. rootKey
// must be that generation's key (checked against its recipient). A wrap
// already there for the same public key is left alone; one under another
// key (the device was revoked and enrolled again with a fresh key) is
// replaced, since the caller proved it holds that generation's root key.
// The current generation is enrolled with Enrol, which never replaces.
func (k *Keyring) EnrolInto(generation int, rootKey []byte, deviceID string, recipient *age.X25519Recipient, now time.Time) error {
	g := k.generation(generation)
	if g == nil {
		return fmt.Errorf("keyring: generation %d is not present", generation)
	}
	if generation == k.CurrentGeneration {
		return k.Enrol(rootKey, deviceID, recipient, now)
	}
	if deviceID == "" || recipient == nil {
		return fmt.Errorf("keyring: device id and public key are required")
	}
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return err
	}
	if identity.Recipient().String() != g.Recipient {
		return fmt.Errorf("keyring: root key does not belong to generation %d", generation)
	}
	wrap, err := wrapForDevice(rootKey, deviceID, recipient, now, k.bindingFor(g))
	if err != nil {
		return err
	}
	for i, d := range g.Devices {
		if d.DeviceID != deviceID {
			continue
		}
		if d.PublicKey == recipient.String() {
			return nil
		}
		g.Devices[i] = wrap
		return nil
	}
	g.Devices = append(g.Devices, wrap)
	return nil
}

// AppendedWrap names one wrap an enrolment wrote: the generation it went
// into and the exact ciphertext written there. A caller that must undo an
// enrolment matches on both, so it can never take back a wrap some other
// approval wrote for the same device under the same public key.
type AppendedWrap struct {
	Generation int
	Wrap       string
}

// EnrolAll enrols deviceID into every generation in keys (the current one
// with Enrol, earlier ones with EnrolInto). keys is the map shape returned
// by UnwrapGenerations; the current generation's key must be present.
//
// It reports the wraps it actually wrote, oldest generation first. That is
// not the same as "every generation in keys": EnrolInto leaves a generation
// alone when it already holds a wrap for this public key, and a caller
// rolling the enrolment back must leave those alone too.
func (k *Keyring) EnrolAll(keys map[int][]byte, deviceID string, recipient *age.X25519Recipient, now time.Time) ([]AppendedWrap, error) {
	current, ok := keys[k.CurrentGeneration]
	if !ok {
		return nil, fmt.Errorf("%w: no key for generation %d", ErrStaleRootKey, k.CurrentGeneration)
	}
	if err := k.Enrol(current, deviceID, recipient, now); err != nil {
		return nil, err
	}
	appended := []AppendedWrap{{Generation: k.CurrentGeneration, Wrap: k.deviceWrap(k.CurrentGeneration, deviceID).Wrap}}
	earlier := make([]int, 0, len(keys))
	for n := range keys {
		if n != k.CurrentGeneration {
			earlier = append(earlier, n)
		}
	}
	sort.Ints(earlier)
	for _, n := range earlier {
		before := ""
		if w := k.deviceWrap(n, deviceID); w != nil {
			before = w.Wrap
		}
		if err := k.EnrolInto(n, keys[n], deviceID, recipient, now); err != nil {
			return appended, err
		}
		if after := k.deviceWrap(n, deviceID); after != nil && after.Wrap != before {
			appended = append(appended, AppendedWrap{Generation: n, Wrap: after.Wrap})
		}
	}
	sort.Slice(appended, func(i, j int) bool { return appended[i].Generation < appended[j].Generation })
	return appended, nil
}

// deviceWrap is the wrap generation number holds for deviceID, or nil.
func (k *Keyring) deviceWrap(number int, deviceID string) *DeviceWrap {
	g := k.generation(number)
	if g == nil {
		return nil
	}
	for i := range g.Devices {
		if g.Devices[i].DeviceID == deviceID {
			return &g.Devices[i]
		}
	}
	return nil
}

func (k *Keyring) checkCurrentRootKey(rootKey []byte) error {
	g := k.current()
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		return err
	}
	if identity.Recipient().String() != g.Recipient {
		return fmt.Errorf("%w (generation %d)", ErrStaleRootKey, g.Number)
	}
	return nil
}

// Rollover starts a new key generation without the revoked devices: a fresh
// root key, wrapped (bound) for every device still listed in the current
// generation and under the recovery code, appended as the new current
// generation. Earlier generations are not touched. The caller must hold
// the current generation's root key (checked against the recorded
// recipient, so a caller whose view is behind a concurrent rollover gets
// ErrStaleRootKey and must reload) and the recovery code (checked against
// the current recovery wrap first, so a wrong code can never produce a
// generation the code does not open).
//
// revoke must name devices listed in the current generation; one that is
// not listed yields ErrAlreadyRevoked (which also matches
// ErrDeviceNotEnrolled) so a revocation that already happened is reported
// rather than repeated. revokedBy is recorded for
// people and must not be among revoke (ErrSelfRevoke).
//
// The new root key is returned so the caller can use it immediately; it
// must be zeroed when no longer needed.
func (k *Keyring) Rollover(currentRootKey []byte, recoveryCode string, revoke []string, revokedBy string, now time.Time) ([]byte, error) {
	g := k.current()
	if g == nil {
		return nil, fmt.Errorf("keyring: no current generation")
	}
	if len(revoke) == 0 {
		return nil, fmt.Errorf("keyring: nothing to revoke")
	}
	if err := k.checkCurrentRootKey(currentRootKey); err != nil {
		return nil, err
	}
	fromRecovery, err := unwrapWithRecoveryCode(g.Recovery, recoveryCode, k.bindingFor(g))
	if err != nil {
		return nil, err
	}
	crypto.Zero(fromRecovery)
	revoked := map[string]bool{}
	for _, id := range revoke {
		if id == revokedBy {
			return nil, ErrSelfRevoke
		}
		if !k.HasDevice(id) {
			return nil, fmt.Errorf("%w: %w (device %s, generation %d)", ErrDeviceNotEnrolled, ErrAlreadyRevoked, id, g.Number)
		}
		revoked[id] = true
	}
	if revokedBy != "" && !k.HasDevice(revokedBy) {
		return nil, fmt.Errorf("%w: revoking device %s has no wrap in generation %d", ErrDeviceNotEnrolled, revokedBy, g.Number)
	}

	next := Generation{Number: k.maxGeneration() + 1, CreatedAt: now.UTC().Format(time.RFC3339)}
	bind := binding{profileID: k.ProfileID, generation: next.Number}
	rootKey, err := crypto.NewRootKey()
	if err != nil {
		return nil, err
	}
	identity, err := crypto.RootKeyIdentity(rootKey)
	if err != nil {
		crypto.Zero(rootKey)
		return nil, err
	}
	next.Recipient = identity.Recipient().String()
	next.Recovery, err = wrapUnderRecoveryCode(rootKey, recoveryCode, bind)
	if err != nil {
		crypto.Zero(rootKey)
		return nil, err
	}
	stamp := now.UTC().Format(time.RFC3339)
	for _, d := range g.Devices {
		if revoked[d.DeviceID] {
			next.Revoked = append(next.Revoked, Revocation{DeviceID: d.DeviceID, RevokedAt: stamp, RevokedBy: revokedBy})
			continue
		}
		recipient, err := age.ParseX25519Recipient(d.PublicKey)
		if err != nil {
			crypto.Zero(rootKey)
			return nil, fmt.Errorf("keyring: device %s has a malformed public key: %w", d.DeviceID, err)
		}
		wrap, err := wrapForDevice(rootKey, d.DeviceID, recipient, now, bind)
		if err != nil {
			crypto.Zero(rootKey)
			return nil, err
		}
		wrap.EnrolledAt = d.EnrolledAt
		next.Devices = append(next.Devices, wrap)
	}
	// Last, once the header is final: chain the new generation to the one
	// it replaces, keyed by the root key this caller proved it holds. The
	// device being revoked does not hold that key from here on, so this is
	// the last generation it could ever have signed.
	next.Chain, err = generationChain(k.ProfileID, &next, g, currentRootKey)
	if err != nil {
		crypto.Zero(rootKey)
		return nil, err
	}
	k.Generations = append(k.Generations, next)
	k.CurrentGeneration = next.Number
	k.SchemaVersion = SchemaVersion
	return rootKey, nil
}

func (k *Keyring) maxGeneration() int {
	n := 0
	for _, g := range k.Generations {
		if g.Number > n {
			n = g.Number
		}
	}
	return n
}

// UnenrolAppended takes back exactly the wraps an EnrolAll wrote for
// deviceID, and nothing else: a wrap is removed only where the generation,
// the device id, and the ciphertext all still match what was written. An
// approving device uses it to undo an enrolment the relay then refused (the
// request expired, or another device had already decided it).
//
// Matching on the ciphertext rather than on the device's public key is what
// keeps it honest. Two approvals of the same joining device carry the same
// public key — the device generates its key once, before its first request —
// so a key match would let a refused approval strip wraps a successful one
// wrote, and would also strip wraps from generations this approval found
// already populated and deliberately left alone.
//
// One case it cannot separate: if a competing approval relayed the very wrap
// this one appended (it found the device listed and sealed for the same key
// without appending a second time), taking it back leaves that device
// without a wrap until it is approved again. That is visible and repairable;
// leaving a refused device enrolled would not be.
//
// Reports whether any wrap was removed. Removing nothing is not an error: a
// concurrent revocation or rollover may already have taken them.
func (k *Keyring) UnenrolAppended(deviceID string, appended []AppendedWrap) bool {
	removed := false
	for _, a := range appended {
		g := k.generation(a.Generation)
		if g == nil || a.Wrap == "" {
			continue
		}
		for i, d := range g.Devices {
			if d.DeviceID == deviceID && d.Wrap == a.Wrap {
				g.Devices = append(g.Devices[:i:i], g.Devices[i+1:]...)
				removed = true
				break
			}
		}
	}
	return removed
}

// Binding prefixes. age has no associated data, so a bound device wrap
// carries the binding inside its authenticated payload and the reader
// checks it; the recovery wrap uses real AEAD associated data.
const (
	deviceWrapInfo   = "reinstate/keyring/device-wrap/v2"
	recoveryAADBound = "reinstate/keyring/recovery-wrap/v2"
)

func (b binding) devicePrefix() []byte {
	return []byte(strings.Join([]string{deviceWrapInfo, b.profileID, strconv.Itoa(b.generation)}, "\x00") + "\x00")
}

func (b binding) recoveryAAD() []byte {
	return []byte(strings.Join([]string{recoveryAADBound, b.profileID, strconv.Itoa(b.generation)}, "\x00"))
}

func unwrapDevice(g *Generation, deviceID string, device *age.X25519Identity, bind binding) ([]byte, error) {
	for _, d := range g.Devices {
		if d.DeviceID != deviceID {
			continue
		}
		if d.PublicKey != device.Recipient().String() {
			return nil, fmt.Errorf("%w: %w", ErrDeviceNotEnrolled, ErrDeviceKeyMismatch)
		}
		if d.Format != WrapFormatBound {
			return nil, fmt.Errorf("keyring: generation %d holds a wrap of unknown format %d for device %s", g.Number, d.Format, deviceID)
		}
		cipher, err := base64.StdEncoding.DecodeString(d.Wrap)
		if err != nil {
			return nil, fmt.Errorf("keyring: device wrap is not valid base64")
		}
		r, err := age.Decrypt(bytes.NewReader(cipher), device)
		if err != nil {
			return nil, fmt.Errorf("keyring: unwrap root key for device: %w", err)
		}
		prefix := bind.devicePrefix()
		payload, err := io.ReadAll(io.LimitReader(r, int64(len(prefix)+crypto.RootKeySize+1)))
		if err != nil {
			return nil, err
		}
		if len(payload) != len(prefix)+crypto.RootKeySize {
			return nil, fmt.Errorf("keyring: device wrap holds %d bytes, want %d", len(payload), len(prefix)+crypto.RootKeySize)
		}
		if !bytes.Equal(payload[:len(prefix)], prefix) {
			crypto.Zero(payload)
			return nil, fmt.Errorf("keyring: device wrap is bound to another profile or generation")
		}
		key := append([]byte(nil), payload[len(prefix):]...)
		crypto.Zero(payload)
		identity, err := crypto.RootKeyIdentity(key)
		if err != nil {
			crypto.Zero(key)
			return nil, err
		}
		if identity.Recipient().String() != g.Recipient {
			crypto.Zero(key)
			return nil, fmt.Errorf("keyring: device wrap does not hold generation %d's root key", g.Number)
		}
		return key, nil
	}
	return nil, ErrDeviceNotEnrolled
}

func sealForDevice(cipher *bytes.Buffer, rootKey []byte, recipient *age.X25519Recipient, bind binding) error {
	w, err := age.Encrypt(cipher, recipient)
	if err != nil {
		return err
	}
	if _, err := w.Write(bind.devicePrefix()); err != nil {
		return err
	}
	if _, err := w.Write(rootKey); err != nil {
		return err
	}
	return w.Close()
}

func wrapForDevice(rootKey []byte, deviceID string, recipient *age.X25519Recipient, now time.Time, bind binding) (DeviceWrap, error) {
	var cipher bytes.Buffer
	if err := sealForDevice(&cipher, rootKey, recipient, bind); err != nil {
		return DeviceWrap{}, err
	}
	return DeviceWrap{
		DeviceID:   deviceID,
		PublicKey:  recipient.String(),
		EnrolledAt: now.UTC().Format(time.RFC3339),
		Wrap:       base64.StdEncoding.EncodeToString(cipher.Bytes()),
		Format:     WrapFormatBound,
	}, nil
}

func wrapUnderRecoveryCode(rootKey []byte, recoveryCode string, bind binding) (RecoveryWrap, error) {
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
		Format:    WrapFormatBound,
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
	sealed := aead.Seal(nil, nonce, rootKey, bind.recoveryAAD())
	wrap.Wrap = base64.StdEncoding.EncodeToString(append(nonce, sealed...))
	return wrap, nil
}

func unwrapWithRecoveryCode(wrap RecoveryWrap, recoveryCode string, bind binding) ([]byte, error) {
	canonical, err := NormalizeRecoveryCode(recoveryCode)
	if err != nil {
		return nil, err
	}
	if wrap.KDF != recoveryKDFName {
		return nil, fmt.Errorf("keyring: unsupported recovery kdf %q", wrap.KDF)
	}
	if wrap.Format != WrapFormatBound {
		return nil, fmt.Errorf("keyring: generation %d holds a recovery wrap of unknown format %d", bind.generation, wrap.Format)
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
	rootKey, err := aead.Open(nil, cipher[:aead.NonceSize()], cipher[aead.NonceSize():], bind.recoveryAAD())
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
