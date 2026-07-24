package main

import (
	"testing"

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
