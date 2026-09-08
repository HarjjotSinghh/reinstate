package main

import (
	"os"
	"strings"
	"testing"
)

func TestLabStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := LabState{
		Root: dir, HopdPID: 1234, HopdAddr: "127.0.0.1:8082", HopdBaseURL: "http://127.0.0.1:8082",
		HopdLog: dir + "/hopd.log", HopdDB: dir + "/hopd.db", LockerPID: 5678, LockerAddr: "127.0.0.1:9002",
	}
	if err := s.save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadState(dir)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got != s {
		t.Fatalf("loadState = %+v, want %+v", got, s)
	}
}

func TestLoadStateMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadState(dir); !os.IsNotExist(err) {
		t.Fatalf("loadState on an empty dir: err=%v, want IsNotExist", err)
	}
}

func TestRemoveStateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := removeState(dir); err != nil {
		t.Fatalf("removeState on an already-clean dir: %v", err)
	}
	s := LabState{Root: dir}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := removeState(dir); err != nil {
		t.Fatalf("removeState: %v", err)
	}
	if _, err := loadState(dir); !os.IsNotExist(err) {
		t.Fatalf("state file survived removeState: err=%v", err)
	}
}

// TestPairingRecoveryCodeNeverEntersHoplabStateJSON is the regression for
// the plaintext-disclosure fix: `pair init` used to persist the recovery
// code straight into hoplab-state.json (shared, world-readable, meant to be
// copied around freely for pids/addresses/log paths) and print it to
// stdout. save() must instead write the code only to its own mode-0600
// sibling file, and hoplab-state.json's own bytes must never contain it,
// while loadState still hands the code back to callers exactly as before
// (pair.go's `recover` action reads it via s.PairingRecoveryCode).
func TestPairingRecoveryCodeNeverEntersHoplabStateJSON(t *testing.T) {
	dir := t.TempDir()
	const code = "GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ"
	s := LabState{Root: dir, HopdPID: 1234, PairingRecoveryCode: code}
	if err := s.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(statePath(dir))
	if err != nil {
		t.Fatalf("read hoplab-state.json: %v", err)
	}
	if strings.Contains(string(raw), code) {
		t.Fatalf("hoplab-state.json contains the plaintext recovery code: %s", raw)
	}

	sibling, err := os.ReadFile(recoveryCodePath(dir))
	if err != nil {
		t.Fatalf("read the recovery-code sibling file: %v", err)
	}
	if got := strings.TrimSpace(string(sibling)); got != code {
		t.Fatalf("recovery-code sibling file = %q, want %q", got, code)
	}

	got, err := loadState(dir)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got.PairingRecoveryCode != code {
		t.Fatalf("loadState PairingRecoveryCode = %q, want %q (pair.go's `recover` action depends on this)", got.PairingRecoveryCode, code)
	}
}

// TestSaveRecoveryCodeRemovesFileWhenCodeIsEmpty covers every action other
// than `pair init` (join, or a lab state that never paired at all): save()
// must not leave a stale, empty recovery-code file around, and a later
// loadState must report "no code" the same way it did before this file
// existed.
func TestSaveRecoveryCodeRemovesFileWhenCodeIsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := (LabState{Root: dir, PairingRecoveryCode: "GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ"}).save(); err != nil {
		t.Fatalf("save (with code): %v", err)
	}
	if err := (LabState{Root: dir}).save(); err != nil {
		t.Fatalf("save (without code): %v", err)
	}
	if _, err := os.Stat(recoveryCodePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("recovery-code file survived a save() with no code: err=%v", err)
	}
	got, err := loadState(dir)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got.PairingRecoveryCode != "" {
		t.Fatalf("loadState PairingRecoveryCode = %q, want empty", got.PairingRecoveryCode)
	}
}

// TestRemoveStateAlsoRemovesTheRecoveryCodeFile covers `hoplab stop`/`stop
// -all` (start.go's removeState calls): tearing down a lab must not leave
// its recovery code behind on disk after everything else is cleaned up.
func TestRemoveStateAlsoRemovesTheRecoveryCodeFile(t *testing.T) {
	dir := t.TempDir()
	s := LabState{Root: dir, PairingRecoveryCode: "GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ"}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := removeState(dir); err != nil {
		t.Fatalf("removeState: %v", err)
	}
	if _, err := os.Stat(recoveryCodePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("recovery-code file survived removeState: err=%v", err)
	}
}
