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
	"crypto/ed25519"
	"crypto/rand"
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
// adopt it. Version 3 MACed each generation under the previous generation's
// root key, which failed in the case it was written for: the device revoked
// at the N -> N+1 rollover is exactly the device that held generation N's
// root key, so within its credential window it could write the generation
// created to remove it. A reader that held no key for generation N could
// not check that link either, and skipped it.
//
// Version 4 signs instead. Every generation, including the first, carries an
// ed25519 signature over its own header, under a keypair derived from the
// **recovery code** — the one secret no device ever holds (see signing.go).
// Verification needs only the public half, which the object publishes and
// every device pins locally, so every read path can check every generation
// without holding any key at all, and a generation that does not verify
// makes the whole keyring untrusted. Version 4 also keeps version 3's rule
// that every wrap is bound; unbound wraps no longer exist anywhere.
//
// Version 5 pulls the recovery wrap inside that signature. Version 4 left it
// out, and the cost was not confidentiality but diagnosis: a party with
// write access could flip one byte of the wrap's ciphertext, and every later
// `rein account recover` told the person their recovery code was wrong. At
// that moment the one thing a person can act on is getting the code right,
// and the product was telling them the one wrong thing. The recovery wrap is
// fixed for the life of a generation — it is written once, by a caller
// holding the code, and never appended to the way device wraps are — so
// covering it costs nothing and turns a damaged wrap into a refusal that
// names tampering. Device wraps are still deliberately outside the
// signature; see generationMessage for why, and what stops that mattering.
const SchemaVersion = 5

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
	// ErrRecoveryMismatch reports a recovery wrap that this code did not
	// open. Since version 5 the wrap's ciphertext and parameters are inside
	// the generation signature, so a keyring that parsed at all is holding
	// the recovery wrap the account wrote: this is the typed code being
	// wrong, or the keyring belonging to another account. It is not the
	// answer for a damaged wrap — that is ErrRecoveryWrapMalformed, or a
	// signature refusal before any code is asked for.
	ErrRecoveryMismatch = errors.New("keyring: recovery code does not match this keyring")
	// ErrRecoveryWrapMalformed reports a recovery wrap this build cannot
	// even attempt: an unknown KDF or wrap format, a salt or ciphertext
	// that is not base64, a ciphertext shorter than a nonce. None of these
	// is a statement about the code that was typed, and reporting them as
	// one sends a person to check the only thing that is not wrong.
	ErrRecoveryWrapMalformed = errors.New("keyring: the recovery wrap is not well-formed")
	ErrDeviceExists          = errors.New("keyring: device already enrolled")
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
	// must not adopt: its signature is missing, malformed, or does not
	// verify under the account's signing key. One such generation makes the
	// whole keyring untrusted — there is no partial acceptance.
	ErrUnauthenticatedGeneration = errors.New("keyring: key generation is not signed by this account's key")
)

// Keyring is the wire shape of the keyring object.
type Keyring struct {
	SchemaVersion     int    `json:"schema_version"`
	ProfileID         string `json:"profile_id"`
	CurrentGeneration int    `json:"current_generation"`
	// AccountKey is the public half of the account signing key, base64. It
	// is derived from the recovery code (see signing.go) and every
	// generation in this object is signed under it. Publishing it here is
	// what lets a device with no local record verify at all; pinning it in
	// account.json is what stops the whole object being replaced by one
	// signed under somebody else's key.
	AccountKey  string       `json:"account_key"`
	Generations []Generation `json:"generations"`
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
	// Signature authenticates this generation: ed25519 over its own header
	// — the profile id, the account key, this generation's number,
	// created_at and recipient, the number and recipient of the generation
	// it follows (0 and "" for the first), and every revocation record —
	// under the account signing key the recovery code derives. Every
	// generation carries one, the first included, and a keyring holding a
	// generation whose signature does not verify is refused whole.
	Signature string `json:"signature"`
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
	// The account signing key is derived here for the first and only time
	// this device will hold it without being asked for the code again: the
	// caller had the code in hand to write the recovery wrap above.
	account, err := DeriveAccountKey(profileID, recoveryCode)
	if err != nil {
		return nil, err
	}
	defer account.Zero()
	identity, _ := crypto.RootKeyIdentity(rootKey)
	k := &Keyring{
		SchemaVersion:     SchemaVersion,
		ProfileID:         profileID,
		CurrentGeneration: 1,
		AccountKey:        account.Public(),
		Generations: []Generation{{
			Number:    1,
			CreatedAt: now.UTC().Format(time.RFC3339),
			Recipient: identity.Recipient().String(),
			Recovery:  recovery,
			Devices:   []DeviceWrap{deviceWrap},
		}},
	}
	k.Generations[0].Signature = account.sign(k.generationMessage(&k.Generations[0], nil))
	return k, nil
}

// Parse decodes a keyring object, validates its structure, and verifies
// every generation's signature under the account key the object publishes.
// A keyring holding one generation that does not verify does not parse at
// all, so no read path can act on part of it.
//
// What Parse cannot answer is whether that account key is this account's.
// Nothing inside the object can say so: a keyring built from generation 1
// upward under a signing key of the attacker's choosing is self-consistent.
// That is the anchor's question — the key pinned in account.json, or the key
// the typed recovery code derives — and callers supply it through
// VerifyGenerations.
//
// # Which of these checks is load-bearing
//
// Several rules below are also enforced further along, and a mutation test
// that removes one at a time will find them individually survivable. That is
// defence in depth and worth keeping, but a later refactor tidying the
// duplication must delete the right copy, so:
//
//   - **Load-bearing, delete nothing here:** the schema_version gate (no
//     other reader checks it), the wrap-format rule (nothing else refuses an
//     unbound wrap before it is opened), the duplicate device id rule
//     (unwrapDevice takes the first match and would silently prefer it), and
//     the closing VerifyGenerations call, which is what makes a keyring
//     holding one unverifiable generation refuse to parse *at all* rather
//     than relying on every caller to ask.
//   - **Duplicated on purpose, and the copy to keep is the other one:**
//     checkSignatureShape (VerifyGenerations refuses the same shapes and is
//     run again by every caller against its anchor), the generation
//     numbering rules — positive, no duplicates, no gaps — which
//     VerifyGenerations re-derives when it looks up each generation's
//     predecessor, and the current_generation presence check, which every
//     caller reaches again through current().
//
// The rule of thumb: a check whose only other copy is inside
// VerifyGenerations may be simplified away here; a check that exists only
// here may not.
func Parse(raw []byte) (*Keyring, error) {
	var k Keyring
	if err := json.Unmarshal(raw, &k); err != nil {
		return nil, fmt.Errorf("keyring: invalid object: %w", err)
	}
	if k.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("keyring: unsupported schema_version %d (this version reads %d only; earlier formats did not authenticate their key generations against a key no device holds, and are not read)", k.SchemaVersion, SchemaVersion)
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
		if err := checkSignatureShape(g); err != nil {
			return nil, err
		}
	}
	// Generations run 1..n with no gaps. Rollover only ever appends
	// maxGeneration()+1, so a gap means generations were removed or a
	// number was invented; either way a generation past the gap names a
	// predecessor that is not here, and its signature cannot be checked.
	for n := 1; n <= len(k.Generations); n++ {
		if !seen[n] {
			return nil, fmt.Errorf("keyring: generation %d is missing from a keyring holding %d generations", n, len(k.Generations))
		}
	}
	if k.current() == nil {
		return nil, fmt.Errorf("keyring: current_generation %d is not present", k.CurrentGeneration)
	}
	// Last, and never skipped: every generation must be signed by the
	// account key this object publishes. Whether that key is the account's
	// own is VerifyGenerations' question, asked with an anchor.
	if err := k.VerifyGenerations(""); err != nil {
		return nil, err
	}
	return &k, nil
}

// checkSignatureShape refuses a generation whose signature field could never
// verify — the wrong length, or not base64. Whether it is *correct* is
// VerifyGenerations' question; this only rules out an object that is
// malformed before any key is involved.
//
// **Not load-bearing on its own, and deliberately kept.** VerifyGenerations,
// which Parse runs a few lines later and which every read path runs again
// against its anchor, refuses the same shapes and is the guard to keep.
// Deleting this one changes an error message and nothing else; deleting
// VerifyGenerations' equivalent opens the hole. See the note on Parse for
// the full list of which of its checks are duplicated and which are not.
func checkSignatureShape(g Generation) error {
	sig, err := base64.StdEncoding.DecodeString(g.Signature)
	if err != nil {
		return fmt.Errorf("%w: generation %d has a signature that is not valid base64", ErrUnauthenticatedGeneration, g.Number)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("%w: generation %d has a %d-byte signature, want %d", ErrUnauthenticatedGeneration, g.Number, len(sig), ed25519.SignatureSize)
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

// generationMessage is the byte string one generation's signature covers:
// the domain separator, the profile id, the account key, this generation's
// number, created_at and recipient, the number and recipient of the
// generation it follows (0 and "" for the first), every revocation record,
// and the whole recovery wrap — its derivation parameters, its salt, its
// format, and its ciphertext. Fields are length-prefixed, so no two
// different headers can produce the same message.
//
// The recovery wrap is covered because it is fixed for the life of a
// generation and because leaving it out had a cost in the one moment it
// mattered: a flipped byte in the ciphertext is indistinguishable, to the
// AEAD, from a wrong code, so `rein account recover` reported a mistyped
// recovery code to a person whose code was correct. Covered, the same
// flipped byte fails the signature and the object is refused as tampered
// with, which is what happened.
//
// The device wraps are deliberately not covered. They change after a
// generation is written — a device enrolled later is given a wrap in every
// generation it may read — and re-signing on every enrolment would need the
// recovery code at moments no caller has it. They do not need covering: a
// wrap is only ever accepted when the key inside it derives the generation's
// recorded recipient (see unwrapDevice), and the recipient is signed here.
// So appending a *working* wrap to a generation still requires that
// generation's root key; what a writer without it can do to the device list
// is remove or corrupt entries, which denies service rather than
// substituting a key.
func (k *Keyring) generationMessage(g, prev *Generation) []byte {
	var buf bytes.Buffer
	var size [8]byte
	write := func(fields ...string) {
		for _, f := range fields {
			binary.BigEndian.PutUint64(size[:], uint64(len(f)))
			buf.Write(size[:])
			buf.WriteString(f)
		}
	}
	prevNumber, prevRecipient := 0, ""
	if prev != nil {
		prevNumber, prevRecipient = prev.Number, prev.Recipient
	}
	write(generationSignatureInfo, k.ProfileID, k.AccountKey,
		strconv.Itoa(g.Number), g.CreatedAt, g.Recipient,
		strconv.Itoa(prevNumber), prevRecipient,
		strconv.Itoa(len(g.Revoked)))
	for _, r := range g.Revoked {
		write(r.DeviceID, r.RevokedAt, r.RevokedBy)
	}
	write(g.Recovery.KDF,
		strconv.FormatUint(uint64(g.Recovery.Time), 10),
		strconv.FormatUint(uint64(g.Recovery.MemoryKiB), 10),
		strconv.FormatUint(uint64(g.Recovery.Threads), 10),
		strconv.Itoa(g.Recovery.Format),
		g.Recovery.Salt, g.Recovery.Wrap)
	return buf.Bytes()
}

// VerifyGenerations authenticates every generation in this keyring under the
// account signing key, and fails closed on the first one that does not
// verify: there is no partial acceptance, and in particular no path that
// accepts a keyring because the *current* generation happens to check out.
// It needs no root key and no device key, so every command can afford it,
// including the ones that hold no keys at all.
//
// pinned is the account key this caller already trusts — the one recorded in
// account.json at enrolment, or the one the typed recovery code derives —
// and "" when the caller has none. With a pinned key, the object's own
// account_key must equal it; without one, the object is verified under the
// key it publishes, which authenticates the generations against each other
// but says nothing about whose account this is. A caller with no anchor is
// trusting whatever brought it here: the recovery code, or the root key an
// enrolled device relayed through a pairing approval.
func (k *Keyring) VerifyGenerations(pinned string) error {
	public, err := parseAccountPublicKey(k.AccountKey)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnauthenticatedGeneration, err)
	}
	if pinned != "" && pinned != k.AccountKey {
		return fmt.Errorf("%w: it is signed by account key %s, not the %s expected here", ErrAccountKeyMismatch, k.AccountKey, pinned)
	}
	numbers := k.GenerationNumbers()
	for _, n := range numbers {
		g := k.generation(n)
		prev := k.generation(n - 1)
		if n > 1 && prev == nil {
			return fmt.Errorf("%w: generation %d has no generation %d before it", ErrUnauthenticatedGeneration, n, n-1)
		}
		sig, err := base64.StdEncoding.DecodeString(g.Signature)
		if err != nil || len(sig) != ed25519.SignatureSize {
			return fmt.Errorf("%w: generation %d carries no usable signature", ErrUnauthenticatedGeneration, n)
		}
		if !ed25519.Verify(public, k.generationMessage(g, prev), sig) {
			return fmt.Errorf("%w: generation %d does not verify under account key %s", ErrUnauthenticatedGeneration, n, k.AccountKey)
		}
	}
	return nil
}

// signGeneration signs g in place, against the generation before it.
func (k *Keyring) signGeneration(account *AccountKey, g, prev *Generation) error {
	if account.Public() != k.AccountKey {
		return fmt.Errorf("%w: this recovery code derives account key %s, but the keyring is signed by %s", ErrAccountKeyMismatch, account.Public(), k.AccountKey)
	}
	g.Signature = account.sign(k.generationMessage(g, prev))
	return nil
}

// AccountPublicKey is the published half of this account's signing key. It
// is what a device pins locally at enrolment.
func (k *Keyring) AccountPublicKey() string { return k.AccountKey }

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
// generation and signed by the account key the code derives. Earlier
// generations are not touched. The caller must hold the current
// generation's root key (checked against the recorded recipient, so a
// caller whose view is behind a concurrent rollover gets ErrStaleRootKey and
// must reload) and the recovery code (checked against the current recovery
// wrap first, so a wrong code can never produce a generation the code does
// not open, and then against the keyring's account key, so it can never
// produce one no device will verify).
//
// The recovery code is what makes this a revocation rather than a formality:
// the device being revoked held the current root key — that is why it is
// being revoked — but it never held the recovery code, so it cannot sign a
// generation of its own.
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
	// The code opened the current generation, so it is this account's code.
	// If the key it derives is nevertheless not the one the keyring is
	// signed by, the object was tampered with rather than mistyped, and
	// signing a generation nothing would verify is worse than refusing.
	account, err := DeriveAccountKey(k.ProfileID, recoveryCode)
	if err != nil {
		return nil, err
	}
	defer account.Zero()
	if account.Public() != k.AccountKey {
		return nil, fmt.Errorf("%w: this recovery code derives account key %s, but the keyring is signed by %s", ErrAccountKeyMismatch, account.Public(), k.AccountKey)
	}
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
	// Last, once the header is final: sign the new generation, naming the
	// one it follows. Only a holder of the recovery code can produce this
	// signature, and the device being revoked is not one — the root key it
	// held until a moment ago buys it nothing here.
	if err := k.signGeneration(account, &next, g); err != nil {
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
	// Everything below the AEAD open is a statement about the wrap, not
	// about the code that was typed, and each one says so: a caller can
	// tell "your code did not open this" from "this wrap is damaged" and
	// tell the person the right thing to do.
	if wrap.KDF != recoveryKDFName {
		return nil, fmt.Errorf("%w: generation %d names key derivation %q, which this version does not read", ErrRecoveryWrapMalformed, bind.generation, wrap.KDF)
	}
	if wrap.Format != WrapFormatBound {
		return nil, fmt.Errorf("%w: generation %d holds a recovery wrap of unknown format %d", ErrRecoveryWrapMalformed, bind.generation, wrap.Format)
	}
	salt, err := base64.StdEncoding.DecodeString(wrap.Salt)
	if err != nil {
		return nil, fmt.Errorf("%w: generation %d has a recovery salt that is not valid base64", ErrRecoveryWrapMalformed, bind.generation)
	}
	cipher, err := base64.StdEncoding.DecodeString(wrap.Wrap)
	if err != nil {
		return nil, fmt.Errorf("%w: generation %d has a recovery wrap that is not valid base64", ErrRecoveryWrapMalformed, bind.generation)
	}
	key := deriveRecoveryKey(canonical, salt, wrap)
	defer crypto.Zero(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(cipher) < aead.NonceSize() {
		return nil, fmt.Errorf("%w: generation %d has a recovery wrap of %d bytes, shorter than the %d-byte nonce it must start with", ErrRecoveryWrapMalformed, bind.generation, len(cipher), aead.NonceSize())
	}
	rootKey, err := aead.Open(nil, cipher[:aead.NonceSize()], cipher[aead.NonceSize():], bind.recoveryAAD())
	if err != nil {
		return nil, ErrRecoveryMismatch
	}
	if len(rootKey) != crypto.RootKeySize {
		return nil, fmt.Errorf("%w: generation %d has a recovery wrap holding %d bytes, want %d", ErrRecoveryWrapMalformed, bind.generation, len(rootKey), crypto.RootKeySize)
	}
	return rootKey, nil
}

func deriveRecoveryKey(canonicalCode string, salt []byte, wrap RecoveryWrap) []byte {
	return argon2.IDKey([]byte(canonicalCode), salt, wrap.Time, wrap.MemoryKiB, wrap.Threads, chacha20poly1305.KeySize)
}
