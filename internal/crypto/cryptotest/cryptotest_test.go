package cryptotest

import (
	"bytes"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// TestFastScryptEnvelopesOpenWithProductionProvider proves the lowered cost
// changes only sealing speed: the production provider still opens the result.
func TestFastScryptEnvelopesOpenWithProductionProvider(t *testing.T) {
	plain := []byte("synthetic payload")
	var cipher bytes.Buffer
	if err := crypto.Seal(bytes.NewReader(plain), &cipher, Passphrase("not-a-real-passphrase")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := crypto.Open(bytes.NewReader(cipher.Bytes()), &out, crypto.NewPassphraseProvider("not-a-real-passphrase")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("got %q", out.Bytes())
	}
	if err := crypto.Open(bytes.NewReader(cipher.Bytes()), &out, crypto.NewPassphraseProvider("wrong")); err == nil {
		t.Fatal("wrong passphrase opened a fast envelope")
	}
}
