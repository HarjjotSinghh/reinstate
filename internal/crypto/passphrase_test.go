package crypto

import (
	"bytes"
	"os"
	"strconv"
	"testing"
)

func TestReadPassphraseFromConfiguredFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "passphrase-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString("fd-secret\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.Itoa(int(file.Fd())))
	got, err := ReadPassphrase(bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fd-secret" {
		t.Fatalf("got %q", got)
	}
	Zero(got)
}

func TestReadPassphraseRejectsOrdinaryEnvironmentSecret(t *testing.T) {
	t.Setenv("REINSTATE_PASSPHRASE_FD", "")
	t.Setenv("REINSTATE_PASSPHRASE", "must-not-be-used")
	if _, err := ReadPassphrase(bytes.NewReader(nil), &bytes.Buffer{}); err == nil {
		t.Fatal("expected non-TTY refusal")
	}
}
