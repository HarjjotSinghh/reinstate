package credentials

import "testing"

func TestMemoryStore(t *testing.T) {
	s := NewMemory()
	if err := s.Set("reinstate/p/s3", StorageCredentials{AccessKeyID: "a", SecretAccessKey: "b"}); err != nil {
		t.Fatal(err)
	}
	c, err := s.Get("reinstate/p/s3")
	if err != nil || c.AccessKeyID != "a" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestEnvStore(t *testing.T) {
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "secret")
	c, err := (EnvStore{}).Get("any")
	if err != nil || c.AccessKeyID != "AKIA" {
		t.Fatalf("%+v %v", c, err)
	}
}
