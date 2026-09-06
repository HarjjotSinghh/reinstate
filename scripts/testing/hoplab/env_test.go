package main

import (
	"strings"
	"testing"
)

func TestPrintEnvSh(t *testing.T) {
	var out strings.Builder
	printEnv(&out, "sh", []envPair{{"REINSTATE_HOME", `D:\lab\device-a\reinstate`}})
	got := out.String()
	if got != `export REINSTATE_HOME="D:\\lab\\device-a\\reinstate"`+"\n" {
		t.Fatalf("sh output = %q", got)
	}
}

func TestPrintEnvPowerShell(t *testing.T) {
	var out strings.Builder
	printEnv(&out, "powershell", []envPair{{"REINSTATE_HOME", `D:\lab\device-a\reinstate`}})
	got := out.String()
	want := "$env:REINSTATE_HOME = 'D:\\lab\\device-a\\reinstate'\n"
	if got != want {
		t.Fatalf("powershell output = %q, want %q", got, want)
	}
}

func TestPsQuoteEscapesSingleQuote(t *testing.T) {
	if got := psQuote("it's here"); got != "it''s here" {
		t.Fatalf("psQuote = %q", got)
	}
}

func TestHopLabEnvNamesEveryIsolationVariable(t *testing.T) {
	s := LabState{HopdBaseURL: "http://127.0.0.1:8082"}
	h := BuildDeviceHome(`D:\lab`, "device-a")
	pairs := hopLabEnv(s, h)
	want := []string{
		"REINSTATE_HOP_URL", "REINSTATE_HOME", "HOME", "USERPROFILE",
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "XDG_DATA_HOME",
		"GROK_HOME", "GEMINI_CLI_HOME", "KIMI_CODE_HOME", "QWEN_HOME",
		"CLINE_DATA_DIR", "COPILOT_HOME", "CURSOR_CONFIG_DIR", "PI_CODING_AGENT_DIR",
	}
	if len(pairs) != len(want) {
		t.Fatalf("got %d pairs, want %d: %+v", len(pairs), len(want), pairs)
	}
	for i, k := range want {
		if pairs[i].Key != k {
			t.Fatalf("pair %d key = %q, want %q", i, pairs[i].Key, k)
		}
		if pairs[i].Value == "" {
			t.Fatalf("pair %d (%s) has an empty value", i, k)
		}
	}
}

// TestHopLabEnvHomeAndUserprofileMatchTheDeviceHome pins the blocker fix
// itself: HOME and USERPROFILE must be the device's own isolated home, not
// left unset (which falls back to the real process home -- see homes.go's
// DeviceHome doc comment and TestDeviceHomesDoNotLeakTheHostAccount in
// homes_isolation_test.go for what that leak looked like end to end).
func TestHopLabEnvHomeAndUserprofileMatchTheDeviceHome(t *testing.T) {
	h := BuildDeviceHome(`D:\lab`, "device-a")
	if h.Home == "" {
		t.Fatal("BuildDeviceHome left Home empty")
	}
	pairs := hopLabEnv(LabState{}, h)
	got := map[string]string{}
	for _, p := range pairs {
		got[p.Key] = p.Value
	}
	if got["HOME"] != h.Home {
		t.Fatalf("HOME = %q, want %q", got["HOME"], h.Home)
	}
	if got["USERPROFILE"] != h.Home {
		t.Fatalf("USERPROFILE = %q, want %q", got["USERPROFILE"], h.Home)
	}
}

func TestPrintEnvClearSh(t *testing.T) {
	var out strings.Builder
	printEnvClear(&out, "sh", []string{"REINSTATE_BACKEND", "REINSTATE_MEMORY_BACKEND_DIR"})
	want := "unset REINSTATE_BACKEND\nunset REINSTATE_MEMORY_BACKEND_DIR\n"
	if got := out.String(); got != want {
		t.Fatalf("sh clear output = %q, want %q", got, want)
	}
}

func TestPrintEnvClearPowerShell(t *testing.T) {
	var out strings.Builder
	printEnvClear(&out, "powershell", []string{"REINSTATE_BACKEND"})
	want := "Remove-Item Env:REINSTATE_BACKEND -ErrorAction SilentlyContinue\n"
	if got := out.String(); got != want {
		t.Fatalf("powershell clear output = %q, want %q", got, want)
	}
}

// TestAmbientOverrideEnvNamesTheKnownEscapeHatches pins the exact set: a
// name silently added to internal/cli's REINSTATE_* env-var surface later
// (openBackend, credentials.Resolve) needs a matching addition here, not
// automatic coverage, so this test is meant to need updating when that
// happens rather than passing by accident.
func TestAmbientOverrideEnvNamesTheKnownEscapeHatches(t *testing.T) {
	want := map[string]bool{
		"REINSTATE_BACKEND":              true,
		"REINSTATE_MEMORY_BACKEND_DIR":   true,
		"REINSTATE_S3_ACCESS_KEY_ID":     true,
		"REINSTATE_S3_SECRET_ACCESS_KEY": true,
		"REINSTATE_S3_ENDPOINT":          true,
		"REINSTATE_S3_BUCKET":            true,
		"REINSTATE_S3_REGION":            true,
	}
	if len(ambientOverrideEnv) != len(want) {
		t.Fatalf("ambientOverrideEnv has %d entries, want %d: %v", len(ambientOverrideEnv), len(want), ambientOverrideEnv)
	}
	seen := map[string]bool{}
	for _, k := range ambientOverrideEnv {
		if !want[k] {
			t.Fatalf("unexpected ambientOverrideEnv entry %q", k)
		}
		if seen[k] {
			t.Fatalf("ambientOverrideEnv lists %q twice", k)
		}
		seen[k] = true
	}
}

func TestStripEnvDropsListedKeysRegardlessOfValue(t *testing.T) {
	base := []string{
		"REINSTATE_BACKEND=memory",
		"OTHER=kept",
		"REINSTATE_MEMORY_BACKEND_DIR=" + `D:\shared-from-earlier-testing`,
		"PATH=C:\\Windows",
	}
	got := stripEnv(base, ambientOverrideEnv)
	for _, kv := range got {
		if strings.HasPrefix(kv, "REINSTATE_BACKEND=") || strings.HasPrefix(kv, "REINSTATE_MEMORY_BACKEND_DIR=") {
			t.Fatalf("stripEnv left an ambient override in place: %v", got)
		}
	}
	values := map[string]bool{}
	for _, kv := range got {
		values[kv] = true
	}
	if !values["OTHER=kept"] || !values["PATH=C:\\Windows"] {
		t.Fatalf("stripEnv dropped an unrelated entry: %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("stripEnv = %v, want exactly the 2 unrelated entries", got)
	}
}
