package doctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

func TestRedactSecretsAndPaths(t *testing.T) {
	home, _ := os.UserHomeDir()
	samples := []string{
		home + "/.claude/projects/foo",
		"user alice has key sk-abc1234567890xyz",
		"https://example.com/x?token=supersecret&x=1",
		"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc",
		"found auth.json and .credentials.json",
		"session transcript: " + strings.Repeat("secret-session-text ", 40),
	}
	for _, s := range samples {
		out := Redact(s)
		if home != "" && strings.Contains(out, home) {
			t.Fatalf("home leaked: %q", out)
		}
		if strings.Contains(out, "sk-abc") || strings.Contains(out, "supersecret") {
			t.Fatalf("secret leaked: %q", out)
		}
		if strings.Contains(out, "eyJhbGci") {
			t.Fatalf("bearer leaked: %q", out)
		}
		if strings.Contains(out, "auth.json") || strings.Contains(out, ".credentials.json") {
			t.Fatalf("auth filename leaked: %q", out)
		}
	}
}

func TestDoctorMissingConfigExit(t *testing.T) {
	home := t.TempDir()
	rep, err := Run(context.Background(), Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	code := ExitCode(rep)
	if code != exitcode.Config {
		if code == exitcode.OK {
			t.Fatalf("expected non-ok for missing config, got %+v", rep)
		}
	}
	human := FormatHuman(rep)
	js, _ := json.Marshal(rep)
	homePath, _ := os.UserHomeDir()
	if homePath != "" && (strings.Contains(human, homePath) || strings.Contains(string(js), homePath)) {
		t.Fatalf("home path in report")
	}
}

func TestDoctorHealthyWithConfig(t *testing.T) {
	home := t.TempDir()
	if err := config.EnsureLayout(home); err != nil {
		t.Fatal(err)
	}
	c := schema.DefaultConfig("p", "d")
	c.Storage.Endpoint = "https://example.r2.cloudflarestorage.com"
	c.Storage.Bucket = "b"
	c.Storage.CredentialRef = "ref"
	if err := config.SaveConfig(home, c); err != nil {
		t.Fatal(err)
	}
	testCodec := &fastAgeEnvelopeCodec{}
	rep, err := Run(context.Background(), Options{
		Home: home, SelfTest: true, EnvelopeCodec: testCodec,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.SelfTest != "ok" {
		t.Fatalf("selftest=%q checks=%+v", rep.SelfTest, rep.Checks)
	}
	if testCodec.encryptions.Load() == 0 {
		t.Fatal("doctor self-test did not exercise the injected age envelope codec")
	}
	_ = filepath.Join(home, "cache")
}

func TestUntestedAdapterFailsReadiness(t *testing.T) {
	check := adapterCheck(
		"claude",
		adapter.Install{Agent: "claude", Version: "2.1.221"},
		adapter.CompatibilityUntested,
		nil,
	)
	if check.Status != "fail" {
		t.Fatalf("status = %q, want fail", check.Status)
	}
	if check.Code != exitcode.Compatibility {
		t.Fatalf("code = %d, want %d", check.Code, exitcode.Compatibility)
	}
	rep := &Report{Checks: []Check{check}}
	if got := summarize(rep); got == "all checks passed" {
		t.Fatalf("untested adapter produced misleading summary %q", got)
	}
	if got := ExitCode(rep); got != exitcode.Compatibility {
		t.Fatalf("exit = %d, want %d", got, exitcode.Compatibility)
	}
}
