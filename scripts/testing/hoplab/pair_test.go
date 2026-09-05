package main

import (
	"strings"
	"testing"
)

func TestRecoveryCodePatternMatchesTheRealFormat(t *testing.T) {
	stderr := "\nYour recovery code (shown once, never stored anywhere):\n\n    GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ\n\nWrite it down"
	got := recoveryCodePattern.FindString(stderr)
	want := "GZ63-Z90G-XPS0-Y54Z-7YAT-RXDP-28H1-YPQZ"
	if got != want {
		t.Fatalf("recoveryCodePattern.FindString = %q, want %q", got, want)
	}
}

func TestRecoveryCodePatternIgnoresThePromptLine(t *testing.T) {
	// The confirmation prompt itself must never look like a code.
	prompt := "Re-enter the recovery code to confirm you saved it: "
	if m := recoveryCodePattern.FindString(prompt); m != "" {
		t.Fatalf("recoveryCodePattern matched the prompt line: %q", m)
	}
}

func TestProjectMappingIsIDEqualsAbsolutePath(t *testing.T) {
	h := BuildDeviceHome(`D:\lab`, "device-a")
	got := projectMapping(h)
	want := "hoplab-device-a=" + h.Project
	if got != want {
		t.Fatalf("projectMapping = %q, want %q", got, want)
	}
	if !strings.Contains(got, "=") {
		t.Fatalf("projectMapping %q has no '=' separator (internal/cli's parseProjectMapping requires ID=path)", got)
	}
}

func TestMergeEnvOverridesExistingKeys(t *testing.T) {
	base := []string{"PATH=C:\\Windows", "REINSTATE_HOME=old", "OTHER=kept"}
	overrides := []envPair{
		{"REINSTATE_HOME", "new"},
		{"HOME", "new-home"},
		{"REINSTATE_HOP_URL", ""}, // empty overrides are skipped, not exported empty
	}
	got := mergeEnv(base, overrides)

	values := map[string]string{}
	for _, kv := range got {
		i := strings.IndexByte(kv, '=')
		values[kv[:i]] = kv[i+1:]
	}
	if values["REINSTATE_HOME"] != "new" {
		t.Fatalf("REINSTATE_HOME = %q, want %q", values["REINSTATE_HOME"], "new")
	}
	if values["HOME"] != "new-home" {
		t.Fatalf("HOME = %q, want %q", values["HOME"], "new-home")
	}
	if values["OTHER"] != "kept" {
		t.Fatalf("OTHER = %q, want %q (an unrelated key must survive the merge)", values["OTHER"], "kept")
	}
	if values["PATH"] != "C:\\Windows" {
		t.Fatalf("PATH = %q, want %q (base entries the overrides don't touch must survive)", values["PATH"], "C:\\Windows")
	}
	if _, ok := values["REINSTATE_HOP_URL"]; ok {
		t.Fatalf("REINSTATE_HOP_URL was exported empty; an empty override must be skipped, not exported")
	}
	// No duplicate REINSTATE_HOME entries (Windows environment blocks
	// tolerate duplicates, but which one a lookup returns is not something
	// to rely on).
	count := 0
	for _, kv := range got {
		if strings.HasPrefix(kv, "REINSTATE_HOME=") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("REINSTATE_HOME appears %d times in the merged environment, want 1: %v", count, got)
	}
}

func TestMergeEnvOnEmptyBaseStillExportsOverrides(t *testing.T) {
	got := mergeEnv(nil, []envPair{{"REINSTATE_HOME", "x"}})
	if len(got) != 1 || got[0] != "REINSTATE_HOME=x" {
		t.Fatalf("mergeEnv(nil, ...) = %v", got)
	}
}

func TestPairingCodePatternMatchesTheRealFormat(t *testing.T) {
	stderr := "\nPairing code for this device (never sent to the control plane):\n\n    K3P9-7XQZ-M2VD-9RT4\n\nOn an already-enrolled device"
	got := pairingCodePattern.FindString(stderr)
	want := "K3P9-7XQZ-M2VD-9RT4"
	if got != want {
		t.Fatalf("pairingCodePattern.FindString = %q, want %q", got, want)
	}
}

func TestIsHelpFlagRecognizesEveryForm(t *testing.T) {
	for _, s := range []string{"-h", "--help", "-help", "help"} {
		if !isHelpFlag(s) {
			t.Errorf("isHelpFlag(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"init", "join", "recover", "-root", ""} {
		if isHelpFlag(s) {
			t.Errorf("isHelpFlag(%q) = true, want false", s)
		}
	}
}

// TestReinEnvironClearsAmbientOverridesFromTheOperatorShell is the
// regression for the round-2 blocker repro: an operator's shell (or a
// verifier's, or an earlier lab's leftover terminal) carrying
// REINSTATE_BACKEND=memory and REINSTATE_MEMORY_BACKEND_DIR from earlier,
// unrelated local testing must not reach a `rein` subprocess pair.go
// launches -- see env.go's ambientOverrideEnv doc comment for the exact,
// dated repro this produced (a brand-new hop-mode account whose keyring
// already "had" two devices, because every hop-mode config's storage
// prefix is empty and the leaked REINSTATE_MEMORY_BACKEND_DIR pointed
// every account at one shared on-disk object left over from unrelated
// earlier work).
func TestReinEnvironClearsAmbientOverridesFromTheOperatorShell(t *testing.T) {
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", `D:\shared-from-earlier-testing`)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "leaked-key")

	h := BuildDeviceHome(`D:\lab`, "device-a")
	env := reinEnviron(LabState{}, h)

	for _, kv := range env {
		for _, bad := range []string{"REINSTATE_BACKEND=", "REINSTATE_MEMORY_BACKEND_DIR=", "REINSTATE_S3_ACCESS_KEY_ID="} {
			if strings.HasPrefix(kv, bad) {
				t.Fatalf("reinEnviron carried an ambient override into the subprocess environment: %v", env)
			}
		}
	}
	// REINSTATE_HOME must still be h's own isolated home -- stripping the
	// ambient overrides must not also strip hopLabEnv's own overlay.
	found := false
	for _, kv := range env {
		if kv == "REINSTATE_HOME="+h.ReinstateHome {
			found = true
		}
	}
	if !found {
		t.Fatalf("reinEnviron dropped REINSTATE_HOME=%s: %v", h.ReinstateHome, env)
	}
}
