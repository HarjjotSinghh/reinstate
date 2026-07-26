package crypto

import (
	"bytes"
	"io"
	"testing"

	"filippo.io/age"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := []byte("session-payload-synthetic")
	pass := "correct horse battery staple"
	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(plain), &buf, pass); err != nil {
		t.Fatal(err)
	}
	cipher := buf.Bytes()
	if bytes.Contains(cipher, plain) {
		t.Fatal("plaintext leaked into ciphertext")
	}
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(cipher), &out, pass); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		t.Fatalf("got %q", out.Bytes())
	}
}

func TestWrongPassphraseFails(t *testing.T) {
	cipher := encryptFastForTest(t, []byte("x"), "right")
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(cipher), &out, "wrong"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestTamperFails(t *testing.T) {
	b := encryptFastForTest(t, []byte("payload-data-here"), "pass")
	if len(b) < 10 {
		t.Fatal("short")
	}
	b[len(b)/2] ^= 0xff
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(b), &out, "pass"); err == nil {
		t.Fatal("expected tamper failure")
	}
}

func encryptFastForTest(t *testing.T, plain []byte, passphrase string) []byte {
	t.Helper()
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		t.Fatal(err)
	}
	recipient.SetWorkFactor(1)
	var ciphertext bytes.Buffer
	writer, err := age.Encrypt(&ciphertext, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(writer, bytes.NewReader(plain)); err != nil {
		_ = writer.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return ciphertext.Bytes()
}

func TestSHA256Hex(t *testing.T) {
	if SHA256Hex([]byte("abc")) == "" {
		t.Fatal("empty")
	}
}
