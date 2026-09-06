package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func TestHomeDefaultAndOverride(t *testing.T) {
	t.Setenv("REINSTATE_HOME", "")
	// clear may not work for empty — set then unset
	_ = os.Unsetenv("REINSTATE_HOME")
	h, err := Home()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(h) != ".reinstate" {
		t.Fatalf("home=%q", h)
	}
	abs := filepath.Join(t.TempDir(), "custom")
	if err := os.MkdirAll(abs, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_HOME", abs)
	h2, err := Home()
	if err != nil {
		t.Fatal(err)
	}
	if h2 != abs {
		t.Fatalf("got %q want %q", h2, abs)
	}
	t.Setenv("REINSTATE_HOME", "relative/path")
	if _, err := Home(); err == nil {
		t.Fatal("expected relative REINSTATE_HOME error")
	}
}

func TestNativeWindowsHome(t *testing.T) {
	p := NativeWindowsHome(`C:\Users\alice`)
	if runtime.GOOS == "windows" {
		if filepath.Base(p) != ".reinstate" {
			t.Fatalf("%q", p)
		}
	} else {
		// On non-Windows, filepath.Join still appends .reinstate
		if filepath.Base(p) != ".reinstate" {
			t.Fatalf("%q", p)
		}
	}
}

func TestConfigRoundTripAndSecrets(t *testing.T) {
	home := t.TempDir()
	if err := EnsureLayout(home); err != nil {
		t.Fatal(err)
	}
	c := schema.DefaultConfig("prof1", "dev1")
	c.Storage.Endpoint = "https://example.r2.cloudflarestorage.com"
	c.Storage.Bucket = "reinstate"
	c.Storage.Prefix = "profiles/prof1"
	c.Storage.CredentialRef = "reinstate/prof1/s3"
	if err := SaveConfig(home, c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProfileID != "prof1" || got.Storage.Bucket != "reinstate" {
		t.Fatalf("%+v", got)
	}

	// secret field rejection
	bad := filepath.Join(home, "config.toml")
	if err := os.WriteFile(bad, []byte("schema_version = 1\npassphrase = \"secret\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(home); err == nil {
		t.Fatal("expected secret rejection")
	}

	// unknown version
	if err := os.WriteFile(bad, []byte("schema_version = 99\nprofile_id = \"p\"\ndevice_id = \"d\"\n[storage]\ntype = \"s3\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(home); err == nil {
		t.Fatal("expected version error")
	}
}

func TestRC4ConfigWithoutRemoteProfileRequiredDefaultsFalse(t *testing.T) {
	home := t.TempDir()
	raw := []byte(`
schema_version = 1
profile_id = "11111111-1111-4111-8111-111111111111"
device_id = "22222222-2222-4222-8222-222222222222"

[storage]
type = "s3"
endpoint = "https://example.r2.cloudflarestorage.com"
region = "auto"
bucket = "reinstate-test"
prefix = "profiles/11111111-1111-4111-8111-111111111111"
credential_ref = "reinstate/11111111-1111-4111-8111-111111111111/s3"

[encryption]
type = "age-scrypt"
`)
	if err := os.WriteFile(ConfigPath(home), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RemoteProfileRequired {
		t.Fatal("RC4 config without remote_profile_required loaded as true")
	}
}

func TestStateRoundTripAndMigrationError(t *testing.T) {
	home := t.TempDir()
	s := schema.NewState()
	s.LastRemoteETag = "etag-1"
	if err := SaveState(home, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(home)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastRemoteETag != "etag-1" {
		t.Fatalf("%+v", got)
	}
	// migration error
	if err := os.WriteFile(StatePath(home), []byte(`{"schema_version": 99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(home); err == nil {
		t.Fatal("expected migration error")
	}
}

func TestSaveConfigOmitsUnsetHopSection(t *testing.T) {
	home := t.TempDir()
	cfg := &schema.Config{SchemaVersion: 1, ProfileID: "p", DeviceID: "d"}
	cfg.Storage.Type = "s3"
	if err := SaveConfig(home, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "[hop]") {
		t.Fatalf("BYO config gained a [hop] section:\n%s", raw)
	}
	cfg.Hop.URL = "https://staging.example"
	if err := SaveConfig(home, cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(home, "config.toml"))
	if !strings.Contains(string(raw), "[hop]") || !strings.Contains(string(raw), "https://staging.example") {
		t.Fatalf("set hop url not written:\n%s", raw)
	}
}
