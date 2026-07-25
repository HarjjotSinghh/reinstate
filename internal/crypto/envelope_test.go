package crypto

import (
	"bytes"
	"testing"
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
	var buf bytes.Buffer
	_ = Encrypt(bytes.NewReader([]byte("x")), &buf, "right")
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(buf.Bytes()), &out, "wrong"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestTamperFails(t *testing.T) {
	var buf bytes.Buffer
	_ = Encrypt(bytes.NewReader([]byte("payload-data-here")), &buf, "pass")
	b := buf.Bytes()
	if len(b) < 10 {
		t.Fatal("short")
	}
	b[len(b)/2] ^= 0xff
	var out bytes.Buffer
	if err := Decrypt(bytes.NewReader(b), &out, "pass"); err == nil {
		t.Fatal("expected tamper failure")
	}
}

func TestSHA256Hex(t *testing.T) {
	if SHA256Hex([]byte("abc")) == "" {
		t.Fatal("empty")
	}
}
