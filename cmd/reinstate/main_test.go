package main

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/cli"
	"github.com/HarjjotSinghh/reinstate/internal/version"
)

func TestVersionString(t *testing.T) {
	s := version.String()
	if s == "" {
		t.Fatal("version string empty")
	}
	if len(s) < len("reinstate ") {
		t.Fatalf("unexpected version string: %q", s)
	}
}

func TestExecuteVersion(t *testing.T) {
	code := cli.Execute(cli.Options{
		Name: "reinstate",
		Args: []string{"version"},
	})
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
}
