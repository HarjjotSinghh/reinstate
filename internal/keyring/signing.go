package keyring

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// The account signing key authenticates every key generation in the keyring.
//
// It is derived from the recovery code, which is the one secret no device
// ever holds: the code is shown once on the first device, typed in again to
// confirm, and after that only re-typed at the prompts that need it
// (`rein devices revoke`, `rein account recover`). Nothing writes it to
// disk, and neither half of this keypair is stored anywhere either — the
// private half exists only while a command that took the code is running,
// and the public half is published in the keyring and pinned in
// `account.json` so every device can verify without the code.
//
// Deriving from a typed code has a cost, and it is the same cost the
// recovery wrap in the same object already carries: a party holding the
// keyring can mount an offline guess at the code by testing candidate
// signatures, exactly as it can by testing candidate unwraps. Both cost one
// argon2id derivation per candidate at the parameters below (3 passes,
// 64 MiB, 4 lanes), and the recovery code carries 140 bits of entropy
// (recoveryDataChars Crockford base32 characters at 5 bits each), so the
// search is 2^140 memory-hard derivations wide. The salt is derived from the
// profile id, so the work done against one account is worth nothing against
// another. See docs/security-model.md, "Key generations".
const accountKeyInfo = "reinstate/keyring/account-signing-key/v1"

// generationSignatureInfo domain-separates the bytes a generation signature
// covers from anything else this account's key might ever sign. Changing it
// would invalidate every signature ever written, so it is versioned.
const generationSignatureInfo = "reinstate/keyring/generation/v1"

// ErrAccountKeyMismatch reports a keyring whose account signing key is not
// the one expected here: not the key this device pinned at enrolment, and
// not the key the typed recovery code derives.
var ErrAccountKeyMismatch = errors.New("keyring: the keyring is signed by a different account key")

// AccountKey is the account's signing keypair, re-derived from the recovery
// code whenever a generation has to be written. Zero it when done.
type AccountKey struct {
	public  string
	private ed25519.PrivateKey
}

// DeriveAccountKey re-derives the account signing keypair from the recovery
// code. recoveryCode is normalized here, so a code as a person typed it is
// accepted; the canonical form is the only string fed to the derivation.
//
// The derivation is memory-hard (argon2id, the same parameters as the
// pairing code) and salted with the profile id, so the keypair is
// account-specific and a table built against one account is worthless
// against another.
func DeriveAccountKey(profileID, recoveryCode string) (*AccountKey, error) {
	if profileID == "" {
		return nil, fmt.Errorf("keyring: a profile id is required to derive the account signing key")
	}
	canonical, err := NormalizeRecoveryCode(recoveryCode)
	if err != nil {
		return nil, err
	}
	salt := sha256.Sum256([]byte(accountKeyInfo + "\x00" + profileID))
	seed := argon2.IDKey([]byte(canonical), salt[:], argon2Time, argon2Memory, argon2Threads, ed25519.SeedSize)
	defer crypto.Zero(seed)
	private := ed25519.NewKeyFromSeed(seed)
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("keyring: derived account key is not an ed25519 key")
	}
	return &AccountKey{public: base64.StdEncoding.EncodeToString(public), private: private}, nil
}

// Public is the published half: base64, as it appears in the keyring's
// account_key and in account.json. It is not key material.
func (a *AccountKey) Public() string { return a.public }

// Zero wipes the private half.
func (a *AccountKey) Zero() {
	if a != nil {
		crypto.Zero(a.private)
	}
}

// sign produces one generation's signature.
func (a *AccountKey) sign(message []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(a.private, message))
}

// parseAccountPublicKey decodes a published account key.
func parseAccountPublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" {
		return nil, fmt.Errorf("keyring: the keyring carries no account_key, so nothing in it can be authenticated")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("keyring: account_key is not valid base64")
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("keyring: account_key is %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}
