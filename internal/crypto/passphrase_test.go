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
	t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.FormatUint(uint64(file.Fd()), 10))
	got, err := ReadPassphrase(bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fd-secret" {
		t.Fatalf("got %q", got)
	}
	Zero(got)
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("configured passphrase descriptor was closed: %v", err)
	}
}

func TestReadPassphraseRejectsOrdinaryEnvironmentSecret(t *testing.T) {
	t.Setenv("REINSTATE_PASSPHRASE_FD", "")
	t.Setenv("REINSTATE_PASSPHRASE", t.Name())
	if _, err := ReadPassphrase(bytes.NewReader(nil), &bytes.Buffer{}); err == nil {
		t.Fatal("expected non-TTY refusal")
	}
}
