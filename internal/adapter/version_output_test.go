package adapter_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
)

// TestStableVersionFromOutput covers the vendor `--version` shapes the adapters
// have to survive. Reading a fixed field index made a supported vendor report
// as untested whenever the wording or the platform package differed, which
// blocks sync writes and changes setup-check exit codes.
func TestStableVersionFromOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "claude trailing product name", output: "2.1.220 (Claude Code)\n", want: "2.1.220"},
		{name: "codex leading package name", output: "codex-cli 0.145.0\n", want: "0.145.0"},
		{name: "bare version only", output: "0.145.0\n", want: "0.145.0"},
		{name: "windows carriage return", output: "codex-cli 0.145.0\r\n", want: "0.145.0"},
		{name: "v prefix", output: "codex v0.145.0", want: "0.145.0"},
		{name: "parenthesised version", output: "codex (0.145.0)", want: "0.145.0"},
		{name: "extra leading words", output: "OpenAI Codex CLI version 0.146.0", want: "0.146.0"},
		{name: "no version present", output: "not logged in", want: ""},
		{name: "empty output", output: "", want: ""},
		{name: "two component version is not stable", output: "codex 0.145", want: ""},
		{name: "prerelease is not stable", output: "codex 0.145.0-beta.1", want: ""},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := adapter.StableVersionFromOutput(test.output); got != test.want {
				t.Fatalf("StableVersionFromOutput(%q) = %q, want %q", test.output, got, test.want)
			}
		})
	}
}

func TestRunVersionCommandUnblocksGrandchildPipes(t *testing.T) {
	dir := t.TempDir()
	writeHangingVersionShim(t, dir, "claude")
	t.Setenv("PATH", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	}

	start := time.Now()
	_, err := adapter.RunVersionCommand(context.Background(), "claude")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("hanging grandchild --version returned success")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("RunVersionCommand blocked %s on pipes held by a grandchild", elapsed)
	}
}

func writeHangingVersionShim(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = os.Getenv("SYSTEMROOT")
		}
		if root == "" {
			t.Skip("SystemRoot is unset")
		}
		ping := filepath.Join(root, "System32", "ping.exe")
		if _, err := os.Stat(ping); err != nil {
			t.Skip("ping.exe is unavailable to stall the version probe")
		}
		path := filepath.Join(dir, name+".cmd")
		body := "@echo off\r\n\"" + ping + "\" -n 30 127.0.0.1 >nul\r\n"
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		return
	}
	sleepBinary := ""
	for _, candidate := range []string{"/bin/sleep", "/usr/bin/sleep"} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			sleepBinary = candidate
			break
		}
	}
	if sleepBinary == "" {
		t.Skip("no absolute sleep binary is available to stall the version probe")
	}
	path := filepath.Join(dir, name)
	body := "#!/bin/sh\n" + sleepBinary + " 30\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}
