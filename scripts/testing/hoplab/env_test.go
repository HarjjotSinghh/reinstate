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
	want := []string{"REINSTATE_HOP_URL", "REINSTATE_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "XDG_DATA_HOME"}
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
