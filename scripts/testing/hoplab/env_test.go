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
