package crypto

import (
	"bytes"
	"testing"
)

func sequentialRootKey() []byte {
	key := make([]byte, RootKeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

// TestRootKeyIdentityIsDeterministic pins the root key → recipient
// derivation. Changing it would orphan every envelope already sealed under a
// root key, so the expected recipient is a golden value, not recomputed.
func TestRootKeyIdentityIsDeterministic(t *testing.T) {
	const want = "age12zhh9qvam60p0vhsv0x423pn2y6u423lwa5f83s3hf6nju2gfcssjvyj2f"
	for range 3 {
		identity, err := RootKeyIdentity(sequentialRootKey())
		if err != nil {
			t.Fatal(err)
		}
		if got := identity.Recipient().String(); got != want {
			t.Fatalf("recipient drifted: got %s want %s", got, want)
		}
	}
	other := sequentialRootKey()
	other[0] ^= 1
	identity, err := RootKeyIdentity(other)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Recipient().String() == want {
		t.Fatal("a different root key derived the same recipient")
	}
}

func TestRootKeyIdentityRejectsWrongLength(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33} {
		if _, err := RootKeyIdentity(make([]byte, n)); err == nil {
			t.Fatalf("%d-byte root key accepted", n)
		}
	}
}

// TestRootKeyProviderRoundTrip seals with one provider instance and opens
// with another built from the same bytes, as two devices would.
func TestRootKeyProviderRoundTrip(t *testing.T) {
	root, err := NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(root) != RootKeySize || bytes.Equal(root, make([]byte, RootKeySize)) {
		t.Fatalf("root key not random: %x", root)
	}
	deviceA, err := NewRootKeyProvider(root)
	if err != nil {
		t.Fatal(err)
	}
	deviceB, err := NewRootKeyProvider(append([]byte(nil), root...))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("synthetic hosted payload")
	var cipher bytes.Buffer
	if err := Seal(bytes.NewReader(plain), &cipher, deviceA); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Open(bytes.NewReader(cipher.Bytes()), &out, deviceB); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("got %q", out.Bytes())
	}
	wrongRoot, _ := NewRootKey()
	wrong, _ := NewRootKeyProvider(wrongRoot)
	if err := Open(bytes.NewReader(cipher.Bytes()), &out, wrong); err == nil {
		t.Fatal("a different root key opened the envelope")
	}
	if err := Open(bytes.NewReader(cipher.Bytes()), &out, NewPassphraseProvider("not-the-key")); err == nil {
		t.Fatal("a passphrase opened a root-key envelope")
	}
}

// TestRootKeyProviderReadsEarlierGenerations seals under an earlier
// generation and opens with a provider whose current key is newer; the
// current key alone must not open it, and new envelopes seal only to the
// current key.
func TestRootKeyProviderReadsEarlierGenerations(t *testing.T) {
	gen1, _ := NewRootKey()
	gen2, _ := NewRootKey()
	old, _ := NewRootKeyProvider(gen1)
	var cipher bytes.Buffer
	if err := Seal(bytes.NewReader([]byte("gen1 payload")), &cipher, old); err != nil {
		t.Fatal(err)
	}
	currentOnly, _ := NewRootKeyProvider(gen2)
	var out bytes.Buffer
	if err := Open(bytes.NewReader(cipher.Bytes()), &out, currentOnly); err == nil {
		t.Fatal("current-only provider opened an earlier generation")
	}
	both, _ := NewRootKeyProvider(gen2, gen1)
	if err := Open(bytes.NewReader(cipher.Bytes()), &out, both); err != nil {
		t.Fatal(err)
	}
	var fresh bytes.Buffer
	if err := Seal(bytes.NewReader([]byte("gen2 payload")), &fresh, both); err != nil {
		t.Fatal(err)
	}
	if err := Open(bytes.NewReader(fresh.Bytes()), &out, old); err == nil {
		t.Fatal("a revoked generation opened a new envelope")
	}
}
