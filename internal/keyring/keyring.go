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
	"sort"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// SchemaVersion is the keyring object format version this package writes.
//
// Version 1 wrapped the root key with nothing but the key itself. Version 2
// binds every wrap to the profile id and the key generation it belongs to
// (device wraps carry the binding inside the age payload; the recovery wrap
// carries it as AEAD associated data), so a wrap lifted out of one keyring
// or generation cannot be replayed in another. Version 1 objects are still
// read; they are rewritten as version 2 on the first write that holds the
// root key (see Enrol and Rollover).
const SchemaVersion = 2

// ObjectName is the keyring object's name inside the profile prefix. The
// name is the object's identity in storage and does not change with the
// schema version; schema_version inside the object does.
const ObjectName = "keyring.v1.json"

// Wrap formats recorded per wrap. WrapFormatLegacy is the version 1 wrap
// (no binding) and is written as an absent field; WrapFormatBound is the
// version 2 wrap.
const (
	WrapFormatLegacy = 0
	WrapFormatBound  = 2
)

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
	// Format is WrapFormatBound for a wrap whose associated data names the
	// profile and generation; absent for a version 1 wrap.
	Format int `json:"format,omitempty"`
}

// DeviceWrap is the root key wrapped to one device's public key.
type DeviceWrap struct {
	DeviceID   string `json:"device_id"`
	PublicKey  string `json:"public_key"`
	EnrolledAt string `json:"enrolled_at"`
	Wrap       string `json:"wrap"`
	// Format is WrapFormatBound for a wrap whose payload names the profile
	// and generation; absent for a version 1 wrap.
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

// Parse decodes and validates a keyring object.
func Parse(raw []byte) (*Keyring, error) {
	var k Keyring
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("keyring: invalid object: %w", err)
	}
	if k.SchemaVersion < 1 || k.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("keyring: unsupported schema_version %d (this version reads 1 to %d)", k.SchemaVersion, SchemaVersion)
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
			if w != WrapFormatLegacy && w != WrapFormatBound {
				return nil, fmt.Errorf("keyring: generation %d holds a wrap of unknown format %d", g.Number, w)
			}
			if k.SchemaVersion == 1 && w != WrapFormatLegacy {
				return nil, fmt.Errorf("keyring: schema_version 1 cannot hold format %d wraps", w)
			}
		}
	}
	if k.current() == nil {
		return nil, fmt.Errorf("keyring: current_generation %d is not present", k.CurrentGeneration)
	}
	return &k, nil
}

func deviceFormats(devices []DeviceWrap) []int {
	out := make([]int, 0, len(devices))
	for _, d := range devices {
		out = append(out, d.Format)
	}
	return out
}

// Marshal encodes the keyring for storage. A keyring read as version 1 is
// written as version 2 (the object format is forwards-only); wraps keep the
// format they were made in until a holder of the root key rebinds them.

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
	// Holding the root key is the one moment legacy wraps can be rebound
	// without anyone else's help; the recovery wrap waits for a rollover.
	return k.rebindDevices(g, rootKey)
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

// EnrolAll enrols deviceID into every generation in keys (the current one
// with Enrol, earlier ones with EnrolInto). keys is the map shape returned
// by UnwrapGenerations; the current generation's key must be present.
func (k *Keyring) EnrolAll(keys map[int][]byte, deviceID string, recipient *age.X25519Recipient, now time.Time) error {
	current, ok := keys[k.CurrentGeneration]
	if !ok {
		return fmt.Errorf("%w: no key for generation %d", ErrStaleRootKey, k.CurrentGeneration)
	}
	if err := k.Enrol(current, deviceID, recipient, now); err != nil {
		return err
	}
	for n, key := range keys {
		if n == k.CurrentGeneration {
			continue
		}
		if err := k.EnrolInto(n, key, deviceID, recipient, now); err != nil {
			return err
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

// rebindDevices rewrites every legacy device wrap in g as a bound wrap.
// Public keys and enrolment times are kept; only the ciphertext changes.
func (k *Keyring) rebindDevices(g *Generation, rootKey []byte) error {
	bind := k.bindingFor(g)
	for i, d := range g.Devices {
		if d.Format == WrapFormatBound {
			continue
		}
		recipient, err := age.ParseX25519Recipient(d.PublicKey)
		if err != nil {
			return fmt.Errorf("keyring: device %s has a malformed public key: %w", d.DeviceID, err)
		}
		var cipher bytes.Buffer
		if err := sealForDevice(&cipher, rootKey, recipient, bind); err != nil {
			return err
		}
		g.Devices[i].Wrap = base64.StdEncoding.EncodeToString(cipher.Bytes())
		g.Devices[i].Format = WrapFormatBound
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

// Binding prefixes. age has no associated data, so a bound device wrap
// carries the binding inside its authenticated payload and the reader
// checks it; the recovery wrap uses real AEAD associated data.
const (
	deviceWrapInfo   = "reinstate/keyring/device-wrap/v2"
	recoveryAAD      = "reinstate/keyring/recovery-wrap/v1"
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
		cipher, err := base64.StdEncoding.DecodeString(d.Wrap)
		if err != nil {
			return nil, fmt.Errorf("keyring: device wrap is not valid base64")
		}
		r, err := age.Decrypt(bytes.NewReader(cipher), device)
		if err != nil {
			return nil, fmt.Errorf("keyring: unwrap root key for device: %w", err)
		}
		prefix := []byte(nil)
		if d.Format == WrapFormatBound {
			prefix = bind.devicePrefix()
		}
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
	aad := []byte(recoveryAAD)
	if wrap.Format == WrapFormatBound {
		aad = bind.recoveryAAD()
	}
	rootKey, err := aead.Open(nil, cipher[:aead.NonceSize()], cipher[aead.NonceSize():], aad)
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
