package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "credentials")
	s := NewFileStore(dir)
	ref := "reinstate/prof/s3"
	if err := s.Set(ref, StorageCredentials{AccessKeyID: "AKIA", SecretAccessKey: "secret"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ref)
	if err != nil || got.AccessKeyID != "AKIA" || got.SecretAccessKey != "secret" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestResolvePrefersEnv(t *testing.T) {
	home := t.TempDir()
	fs := NewFileStore(filepath.Join(home, "credentials"))
	_ = fs.Set("ref", StorageCredentials{AccessKeyID: "file", SecretAccessKey: "file"})
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "env")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "envsec")
	got, err := Resolve(home, "ref")
	if err != nil || got.AccessKeyID != "env" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestResolveFileWhenNoEnv(t *testing.T) {
	home := t.TempDir()
	_ = os.Unsetenv("REINSTATE_S3_ACCESS_KEY_ID")
	_ = os.Unsetenv("REINSTATE_S3_SECRET_ACCESS_KEY")
	fs := NewFileStore(filepath.Join(home, "credentials"))
	if err := fs.Set("ref", StorageCredentials{AccessKeyID: "file", SecretAccessKey: "file"}); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(home, "ref")
	if err != nil || got.AccessKeyID != "file" {
		t.Fatalf("%+v %v", got, err)
	}
}
