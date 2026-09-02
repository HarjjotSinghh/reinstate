package keyring

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

var pairingCodePattern = regexp.MustCompile(`^(?:[0-9A-Z]{4}-){3}[0-9A-Z]{4}$`)

func TestPairingRoundTrip(t *testing.T) {
	joining, err := NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	if joining.Version != PairingVersion2 {
		t.Fatalf("new pairing version = %d, want 2", joining.Version)
	}
	if !pairingCodePattern.MatchString(joining.Code) {
		t.Fatalf("code %q", joining.Code)
	}
	deviceKey, _ := age.GenerateX25519Identity()
	pub := deviceKey.Recipient().String()
	binding := joining.Binding(pub)

	// The approving side types the code casually.
	typed := strings.ToLower(strings.ReplaceAll(joining.Code, "-", " "))
	approving, err := PairingFromCode(typed, joining.Salt, joining.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !approving.VerifyBinding(pub, binding) {
		t.Fatal("binding did not verify with the right code")
	}
	other, _ := age.GenerateX25519Identity()
	if approving.VerifyBinding(other.Recipient().String(), binding) {
		t.Fatal("binding verified for a substituted public key")
	}

	rootKey, _ := crypto.NewRootKey()
	payload, err := approving.SealRootKey(rootKey, "req-1", deviceKey.Recipient(), 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := joining.OpenRootKey(payload, "req-1", deviceKey, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, rootKey) {
		t.Fatal("root key did not survive the round trip")
	}

	cases := map[string]func() ([]byte, error){
		"wrong request id":  func() ([]byte, error) { return joining.OpenRootKey(payload, "req-2", deviceKey, 1) },
		"wrong generation":  func() ([]byte, error) { return joining.OpenRootKey(payload, "req-1", deviceKey, 2) },
		"wrong device key":  func() ([]byte, error) { return joining.OpenRootKey(payload, "req-1", other, 1) },
		"truncated payload": func() ([]byte, error) { return joining.OpenRootKey(payload[:len(payload)-8], "req-1", deviceKey, 1) },
		"wrong code": func() ([]byte, error) {
			wrong, _ := NewPairing()
			wrongSide, _ := PairingFromCode(wrong.Code, joining.Salt, joining.Version)
			return wrongSide.OpenRootKey(payload, "req-1", deviceKey, 1)
		},
	}
	for name, open := range cases {
		if key, err := open(); err == nil || key != nil {
			t.Fatalf("%s: opened (%v)", name, err)
		} else if name != "truncated payload" && !errors.Is(err, ErrPairingMismatch) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Only the code holder can produce a payload that opens.
	forged, _ := NewPairing()
	forgedSide, _ := PairingFromCode(forged.Code, joining.Salt, joining.Version)
	forgedPayload, _ := forgedSide.SealRootKey(rootKey, "req-1", deviceKey.Recipient(), 1)
	if _, err := joining.OpenRootKey(forgedPayload, "req-1", deviceKey, 1); !errors.Is(err, ErrPairingMismatch) {
		t.Fatalf("forged payload: %v", err)
	}
}

func TestNormalizePairingCode(t *testing.T) {
	p, _ := NewPairing()
	good := p.Code
	cases := map[string]string{
		"canonical":   good,
		"lower":       strings.ToLower(good),
		"spaces":      strings.ReplaceAll(good, "-", " "),
		"no dashes":   strings.ReplaceAll(good, "-", ""),
		"confusables": strings.NewReplacer("0", "O", "1", "l").Replace(good),
	}
	for name, typed := range cases {
		got, err := NormalizePairingCode(typed)
		if err != nil || got != good {
			t.Fatalf("%s: %q -> %q %v", name, typed, got, err)
		}
	}
	typo := "0000" + good[4:]
	if typo == good {
		typo = "1111" + good[4:]
	}
	for name, typed := range map[string]string{
		"typo":       typo,
		"short":      good[:14],
		"bad char":   good[:15] + "U",
		"empty":      "",
		"recovery":   "AAAA-AAAA-AAAA-AAAA-AAAA-AAAA-AAAA-AAAA",
		"wrong salt": good,
	} {
		salt := p.Salt
		if name == "wrong salt" {
			salt = salt[:3]
		}
		if _, err := PairingFromCode(typed, salt, PairingVersion2); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

const (
	goldenPairingCode = "QXEG-8K98-STMX-KHCK"
	goldenPairingID   = "123e4567-e89b-12d3-a456-426614174000"
)

var goldenPairingSalt = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

func goldenPairing(t *testing.T, version int) *Pairing {
	t.Helper()
	p, err := PairingFromCode(goldenPairingCode, goldenPairingSalt, version)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Zero)
	return p
}

func goldenPairingPayload(t *testing.T, p *Pairing) string {
	t.Helper()
	identity, err := age.ParseX25519Identity(goldenDeviceKey)
	if err != nil {
		t.Fatal(err)
	}
	rootKey := make([]byte, crypto.RootKeySize)
	for i := range rootKey {
		rootKey[i] = byte(i)
	}
	random := make([]byte, 4096)
	for i := range random {
		random[i] = byte(i)
	}
	oldReader := rand.Reader
	rand.Reader = bytes.NewReader(random)
	defer func() { rand.Reader = oldReader }()
	payload, err := p.SealRootKey(rootKey, goldenPairingID, identity.Recipient(), 7)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

// TestPairingProtocolGoldens pins both protocols. Version 1 is immutable: a
// request opened by the previous client must still verify and open. Version 2
// keeps the same expensive master derivation, then separates the binding and
// payload uses with HKDF-SHA256 and binds the server request id into the
// payload-key derivation as well as the AEAD associated data.
func TestPairingProtocolGoldens(t *testing.T) {
	identity, err := age.ParseX25519Identity(goldenDeviceKey)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, master, bindingKey, payloadKey, binding, payload string
		version                                                int
	}{
		{name: "v1", version: PairingVersion1, master: "ad711a3c71a643be681add3822ecb716ff3064a12b5f66e38e1e2b4da2cd528c", bindingKey: "ad711a3c71a643be681add3822ecb716ff3064a12b5f66e38e1e2b4da2cd528c", payloadKey: "ad711a3c71a643be681add3822ecb716ff3064a12b5f66e38e1e2b4da2cd528c", binding: "jK1+j97F5je+HhiJGFuTYwPaf+471FyPWsWi+i7EF3k=", payload: "QEFCQ0RFRkdISUpLTE1OT1BRUlNUVVZXehhK0W78n/owcAelNoDw1PmfcFT0smN7A4zMvUkp56BnIRcTtn2Eqh9WDT7Sz8mZKx6B7TIVUMhUBw5PYOGSud99Q3ZRjp4NRsqZ4wYoe74fwfYVnI9CqS4yhfs5EC0c8/0gkkLqu7Ga++LMcspG9aCgQzfKIkF/iU2qYzgX9V7L5pJSKJaZwmKEeEDn8PD3f807FLWSR6/u4UkQTSX51XIjZiqr2Bpmchvbt/Gn4ndLRab5D0QdDvuEsurazxSKzrtsA7X7mL0dxsExFIKKH9yiEuDACuScBEZR2D9dUuJftALzULdWvR+86/FQxh1udhR4HJglRmk="},
		{name: "v2", version: PairingVersion2, master: "ad711a3c71a643be681add3822ecb716ff3064a12b5f66e38e1e2b4da2cd528c", bindingKey: "ccfe47c51cd8d04a60bf04c40304868a8f85003e4e2a370cbd306c2bc7daed3a", payloadKey: "19471fce98d98c55c9ca380e8d1d0aa3104529582f393cdb4a8cc4a6894edff5", binding: "zv2gpgt5xG2L5CSScoBxJ25T8ka4qF0Q29MjN71g/LQ=", payload: "QEFCQ0RFRkdISUpLTE1OT1BRUlNUVVZXNNiU3ooiQrnPpOAiLGlpNv1YXweyB+0LhRpXbJbMwiII2sC1mn9fUk75ob3/WW61YpsWJ1BdXY6LSnspBjVE4ZV84DVXmTAk3W6Ou0D7XVnX7ArCwRvByALygwSJYpZmvckTVFI7BFEahdXXFHhGxPgqpV2Y/ZCqWRQfd1oSH6ypm8Qkn5kNDWVaEY79WTbgmCCsoY9kJZoQyXFWdHQDi0LJJEFHqVyF508jkun1M39ctwa540cNV94wd1DLGAlpHZONsKISc526c/bQVEb9u06Zmvf/tgRL8bWx4/Zlzo5+z6dHM71w1YGoHoEVXpH/fUAFoh4lUdQ="},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := goldenPairing(t, tc.version)
			if got := hex.EncodeToString(p.key); got != tc.master {
				t.Fatalf("master = %q", got)
			}
			bindingKey := p.bindingKey()
			defer crypto.Zero(bindingKey)
			if got := hex.EncodeToString(bindingKey); got != tc.bindingKey {
				t.Fatalf("binding key = %q", got)
			}
			payloadKey := p.payloadKey(goldenPairingID)
			defer crypto.Zero(payloadKey)
			if got := hex.EncodeToString(payloadKey); got != tc.payloadKey {
				t.Fatalf("payload key = %q", got)
			}
			if got := p.Binding(identity.Recipient().String()); got != tc.binding {
				t.Fatalf("binding = %q", got)
			}
			if got := goldenPairingPayload(t, p); got != tc.payload {
				t.Fatalf("payload = %q", got)
			}
		})
	}
}

func TestPairingV2UsesDistinctRequestBoundSubkeys(t *testing.T) {
	p := goldenPairing(t, PairingVersion2)
	bindingKey := p.bindingKey()
	payloadKey := p.payloadKey(goldenPairingID)
	otherPayloadKey := p.payloadKey("another-server-pairing-id")
	defer crypto.Zero(bindingKey)
	defer crypto.Zero(payloadKey)
	defer crypto.Zero(otherPayloadKey)
	if bytes.Equal(bindingKey, payloadKey) {
		t.Fatal("v2 reused one key for the binding HMAC and payload AEAD")
	}
	if bytes.Equal(payloadKey, otherPayloadKey) {
		t.Fatal("v2 payload key is not bound to the server pairing id")
	}
}

func TestPairingVersionCompatibilityAndWrongID(t *testing.T) {
	identity, err := age.ParseX25519Identity(goldenDeviceKey)
	if err != nil {
		t.Fatal(err)
	}
	rootKey := make([]byte, crypto.RootKeySize)
	for i := range rootKey {
		rootKey[i] = byte(i)
	}
	for _, version := range []int{PairingVersion1, PairingVersion2} {
		t.Run("v"+string(rune('0'+version)), func(t *testing.T) {
			p := goldenPairing(t, version)
			payload := goldenPairingPayload(t, p)
			got, err := p.OpenRootKey(payload, goldenPairingID, identity, 7)
			if err != nil || !bytes.Equal(got, rootKey) {
				t.Fatalf("open = %x, %v", got, err)
			}
			if got, err := p.OpenRootKey(payload, "wrong-pairing-id", identity, 7); !errors.Is(err, ErrPairingMismatch) || got != nil {
				t.Fatalf("wrong id opened = %x, %v", got, err)
			}
		})
	}

	legacy := goldenPairing(t, PairingVersion1)
	missing, err := PairingFromCode(goldenPairingCode, goldenPairingSalt, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Zero()
	legacyPayload := goldenPairingPayload(t, legacy)
	opened, openErr := missing.OpenRootKey(legacyPayload, goldenPairingID, identity, 7)
	if missing.Version != PairingVersion1 || missing.Binding(identity.Recipient().String()) != legacy.Binding(identity.Recipient().String()) || openErr != nil || !bytes.Equal(opened, rootKey) {
		t.Fatalf("a missing wire version no longer means pairing v1: open=%x, %v", opened, openErr)
	}
	current := goldenPairing(t, PairingVersion2)
	if opened, err := current.OpenRootKey(legacyPayload, goldenPairingID, identity, 7); !errors.Is(err, ErrPairingMismatch) || opened != nil {
		t.Fatalf("v2 accepted a v1 payload: %x, %v", opened, err)
	}
	if _, err := PairingFromCode(goldenPairingCode, goldenPairingSalt, 3); err == nil {
		t.Fatal("unsupported pairing version 3 was accepted")
	}
}
