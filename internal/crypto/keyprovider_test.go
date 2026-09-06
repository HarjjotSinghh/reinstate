package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
)

// x25519TestProvider is a test-only second key model: a fixed age X25519
// identity. It proves the seam does not assume a passphrase.
type x25519TestProvider struct{ identity *age.X25519Identity }

func newX25519TestProvider(t *testing.T) *x25519TestProvider {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	return &x25519TestProvider{identity: identity}
}

func (p *x25519TestProvider) Recipients() ([]age.Recipient, error) {
	return []age.Recipient{p.identity.Recipient()}, nil
}

func (p *x25519TestProvider) Identities() ([]age.Identity, error) {
	return []age.Identity{p.identity}, nil
}

// fastPassphrase lowers only the scrypt cost of the production provider. It
// mirrors cryptotest.FastScrypt, which cannot be imported here without a cycle.
func fastPassphrase(passphrase string) KeyProvider {
	return fastScrypt{keys: NewPassphraseProvider(passphrase)}
}

type fastScrypt struct{ keys KeyProvider }

func (f fastScrypt) Recipients() ([]age.Recipient, error) {
	recipients, err := f.keys.Recipients()
	if err != nil {
		return nil, err
	}
	for _, r := range recipients {
		if scrypt, ok := r.(*age.ScryptRecipient); ok {
			scrypt.SetWorkFactor(1)
		}
	}
	return recipients, nil
}

func (f fastScrypt) Identities() ([]age.Identity, error) { return f.keys.Identities() }

func TestSealOpenRoundTripByProvider(t *testing.T) {
	plain := []byte("session-payload-synthetic")
	tests := []struct {
		name string
		keys KeyProvider
	}{
		{name: "passphrase", keys: fastPassphrase("correct horse battery staple")},
		{name: "x25519", keys: newX25519TestProvider(t)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var cipher bytes.Buffer
			if err := Seal(bytes.NewReader(plain), &cipher, tc.keys); err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(cipher.Bytes(), plain) {
				t.Fatal("plaintext leaked into ciphertext")
			}
			var out bytes.Buffer
			if err := Open(bytes.NewReader(cipher.Bytes()), &out, tc.keys); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), plain) {
				t.Fatalf("got %q", out.Bytes())
			}
		})
	}
}

func TestProvidersDoNotOpenEachOthersEnvelopes(t *testing.T) {
	passphrase := fastPassphrase("pass")
	x25519 := newX25519TestProvider(t)
	other := newX25519TestProvider(t)
	var cipher bytes.Buffer
	if err := Seal(bytes.NewReader([]byte("x")), &cipher, passphrase); err != nil {
		t.Fatal(err)
	}
	for name, keys := range map[string]KeyProvider{"x25519": x25519, "wrong passphrase": fastPassphrase("wrong")} {
		if _, err := OpenReader(bytes.NewReader(cipher.Bytes()), keys); err == nil {
			t.Fatalf("%s opened a passphrase envelope", name)
		}
	}
	cipher.Reset()
	if err := Seal(bytes.NewReader([]byte("x")), &cipher, x25519); err != nil {
		t.Fatal(err)
	}
	for name, keys := range map[string]KeyProvider{"passphrase": passphrase, "other x25519": other} {
		if _, err := OpenReader(bytes.NewReader(cipher.Bytes()), keys); err == nil {
			t.Fatalf("%s opened an x25519 envelope", name)
		}
	}
}

func TestPassphraseProviderMatchesLegacyHelpers(t *testing.T) {
	plain := []byte("legacy-compatible payload")
	const pass = "shared-passphrase"
	var viaProvider bytes.Buffer
	if err := Seal(bytes.NewReader(plain), &viaProvider, fastPassphrase(pass)); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(viaProvider.Bytes()), &out, pass); err != nil {
		t.Fatalf("legacy Decrypt rejected a provider-sealed envelope: %v", err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("got %q", out.Bytes())
	}
	legacy := encryptFastForTest(t, plain, pass)
	out.Reset()
	if err := Open(bytes.NewReader(legacy), &out, NewPassphraseProvider(pass)); err != nil {
		t.Fatalf("provider rejected a legacy envelope: %v", err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("got %q", out.Bytes())
	}
}

func TestPassphraseProviderRejectsEmpty(t *testing.T) {
	for name, fn := range map[string]func() error{
		"seal": func() error { return Seal(bytes.NewReader(nil), &bytes.Buffer{}, NewPassphraseProvider("")) },
		"open": func() error { _, err := OpenReader(bytes.NewReader(nil), NewPassphraseProvider("")); return err },
		"nil":  func() error { return Seal(bytes.NewReader(nil), &bytes.Buffer{}, nil) },
	} {
		if err := fn(); err == nil {
			t.Fatalf("%s accepted a missing passphrase", name)
		}
	}
}

// TestGoldenPreSeamEnvelopeDecrypts opens an envelope written by the code that
// predates the key-provider seam (testdata/crypto/pre-seam, generated from
// main with the default scrypt work factor) and requires identical plaintext.
func TestGoldenPreSeamEnvelopeDecrypts(t *testing.T) {
	if testing.Short() {
		t.Skip("decrypts at age's default scrypt cost")
	}
	cipher, err := os.ReadFile(filepath.Join("..", "..", "testdata", "crypto", "pre-seam", "envelope.age"))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("reinstate golden envelope plaintext v1\n")
	var out bytes.Buffer
	if err := Open(bytes.NewReader(cipher), &out, NewPassphraseProvider("golden-passphrase-not-real")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("golden plaintext mismatch: got %q", out.Bytes())
	}
}
