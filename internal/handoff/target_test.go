package handoff

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

type stubTarget struct {
	name string
	caps TargetCapabilities
}

func (s *stubTarget) Name() string { return s.name }

func (s *stubTarget) Capabilities() TargetCapabilities {
	if s.caps.Agent == "" {
		s.caps.Agent = s.name
	}
	return s.caps
}

func (s *stubTarget) Compatible(context.Context) (adapter.Compatibility, error) {
	return adapter.CompatibilitySupported, nil
}

func (s *stubTarget) Plan(capsule.Capsule, Policy) (DestinationPlan, capsule.Fidelity, error) {
	return DestinationPlan{}, capsule.Fidelity{}, nil
}

func (s *stubTarget) Materialize(context.Context, DestinationPlan) error { return nil }

func (s *stubTarget) Launch(context.Context, DestinationPlan, sessionindex.LaunchRunner) error {
	return nil
}

func (s *stubTarget) Verify(context.Context, DestinationPlan, time.Time) (string, string, error) {
	return "", VerifyUnresolved, nil
}

func TestRegisterTargetRejectsDuplicateEmptyAndNil(t *testing.T) {
	name := "test-target-" + t.Name()
	tr := &stubTarget{name: name}

	if err := RegisterTarget(tr); err != nil {
		t.Fatalf("first RegisterTarget: %v", err)
	}
	if err := RegisterTarget(tr); err == nil {
		t.Fatal("second RegisterTarget succeeded, want duplicate error")
	}
	if err := RegisterTarget(&stubTarget{name: ""}); err == nil {
		t.Fatal("RegisterTarget empty name succeeded, want error")
	}
	if err := RegisterTarget(nil); err == nil {
		t.Fatal("RegisterTarget nil succeeded, want error")
	}

	got, ok := Target(name)
	if !ok || got.Name() != name {
		t.Fatalf("Target(%q) = (%v, %v)", name, got, ok)
	}
	if _, ok := Target("missing-" + t.Name()); ok {
		t.Fatal("Target missing name unexpectedly ok")
	}
}

func TestArgvBytesSumsExecutableAndArgs(t *testing.T) {
	t.Parallel()

	got := ArgvBytes("claude", []string{"--session-id", "abc", "hello"})
	want := len("claude") + len("--session-id") + len("abc") + len("hello")
	if got != want {
		t.Fatalf("ArgvBytes = %d, want %d", got, want)
	}
}

func TestWritePlannedFilesRejectsArgvBeforeAnyWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap.txt")
	plan := DestinationPlan{
		Executable: "agent",
		Args:       []string{strings.Repeat("x", 64)},
		Files: []PlannedFile{
			{Path: path, Mode: 0o600, SHA256: "unused"},
		},
		Bootstrap: []byte("bootstrap"),
	}
	contents := map[string][]byte{path: []byte("bootstrap")}

	err := WritePlannedFiles(plan, 32, contents)
	if !errors.Is(err, ErrArgvExceedsBudget) {
		t.Fatalf("WritePlannedFiles error = %v, want %v", err, ErrArgvExceedsBudget)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("planned file written despite argv budget failure: %v", statErr)
	}
}

func TestWritePlannedFilesWritesAfterArgvOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap.txt")
	body := []byte("bootstrap-ok")
	plan := DestinationPlan{
		Executable: "agent",
		Args:       []string{"short"},
		Files: []PlannedFile{
			{Path: path, Mode: 0o644, SHA256: ""},
		},
	}
	if err := WritePlannedFiles(plan, DefaultMaxArgvBytes, map[string][]byte{path: body}); err != nil {
		t.Fatalf("WritePlannedFiles: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("content = %q, want %q", got, body)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %04o, want 0600", info.Mode().Perm())
	}
}

func TestVerifyStateConstants(t *testing.T) {
	t.Parallel()

	if VerifyResolved != "resolved" || VerifyUnresolved != "unresolved" || VerifyAmbiguous != "ambiguous" {
		t.Fatalf("verify states = %q/%q/%q", VerifyResolved, VerifyUnresolved, VerifyAmbiguous)
	}
}
