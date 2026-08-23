package keyring

import (
	"bytes"
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
	if !pairingCodePattern.MatchString(joining.Code) {
		t.Fatalf("code %q", joining.Code)
	}
	deviceKey, _ := age.GenerateX25519Identity()
	pub := deviceKey.Recipient().String()
	binding := joining.Binding(pub)

	// The approving side types the code casually.
	typed := strings.ToLower(strings.ReplaceAll(joining.Code, "-", " "))
	approving, err := PairingFromCode(typed, joining.Salt)
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
			wrongSide, _ := PairingFromCode(wrong.Code, joining.Salt)
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
	forgedSide, _ := PairingFromCode(forged.Code, joining.Salt)
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
		if _, err := PairingFromCode(typed, salt); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}
