// Command reinstate is the CLI entrypoint for Reinstate —
// encrypted multi-agent session sync for AI coding tools.
//
// Copyright (c) 2026 Harjot Singh Rana. MIT License.
package main

import (
	"fmt"
	"os"

	"github.com/HarjjotSinghh/reinstate/internal/version"
)

const usage = `reinstate — sync AI coding agent sessions across devices

Usage:
  reinstate <command> [flags]

Commands:
  version     Print version information
  help        Show this help

Coming soon (see ROADMAP.md):
  init        Interactive setup (backend, encryption, path map)
  push        Encrypt and upload local sessions/config
  pull        Download, decrypt, and restore (with path remap)
  status      Compare local vs remote
  diff        Show pending changes
  conflicts   List and resolve sync conflicts

Docs: https://github.com/HarjjotSinghh/reinstate#readme
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(version.String())
	case "help", "--help", "-h":
		fmt.Print(usage)
	case "init", "push", "pull", "status", "diff", "conflicts":
		fmt.Fprintf(os.Stderr,
			"command %q is not implemented yet (pre-v0.1 scaffold).\nSee https://github.com/HarjjotSinghh/reinstate/blob/main/ROADMAP.md\n",
			os.Args[1],
		)
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}
