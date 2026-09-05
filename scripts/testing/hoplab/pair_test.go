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
