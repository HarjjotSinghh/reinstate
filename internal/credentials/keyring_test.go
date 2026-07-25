package credentials

import (
	"os"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestKeyringStoreRoundTrip(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore()
	ref := "reinstate/profile/s3"
	want := StorageCredentials{AccessKeyID: "AKIA_TEST", SecretAccessKey: "secret-test"}
	if err := store.Set(ref, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ref)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if err := store.Delete(ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); err == nil {
		t.Fatal("expected deleted credential to be absent")
	}
}

func TestResolvePrefersExplicitEnvironmentFallback(t *testing.T) {
	keyring.MockInit()
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "env-access")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "env-secret")
	got, err := Resolve(t.TempDir(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessKeyID != "env-access" || got.SecretAccessKey != "env-secret" {
		t.Fatalf("unexpected credentials: %#v", got)
	}
}

func TestResolveRejectsPartialEnvironmentCredentials(t *testing.T) {
	keyring.MockInit()
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "env-access")
	_ = os.Unsetenv("REINSTATE_S3_SECRET_ACCESS_KEY")
	if _, err := Resolve(t.TempDir(), "missing"); err == nil {
		t.Fatal("expected partial environment credentials rejection")
	}
}
