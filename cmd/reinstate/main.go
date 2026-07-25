// Command reinstate is the CLI entrypoint for Reinstate —
// encrypted multi-agent session sync for AI coding tools.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/cli"
)

func main() {
	name := "reinstate"
	if base := filepath.Base(os.Args[0]); base != "" {
		base = strings.TrimSuffix(base, ".exe")
		if base == "rein" || base == "reinstate" {
			name = base
		}
	}
	os.Exit(cli.Execute(cli.Options{Name: name}))
}
