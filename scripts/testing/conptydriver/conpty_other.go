//go:build !windows

package main

import (
	"fmt"
	"runtime"
)

// StartConsole exists on every platform so this package builds everywhere
// (Gate 1 cross-builds darwin and linux), but ConPTY is a Windows-only
// mechanism: there is nothing to stand in for it here, and conptydriver is
// a Windows acceptance tool, not a cross-platform one.
func StartConsole(args []string, cols, rows int, dir string) (Console, error) {
	return nil, fmt.Errorf("conptydriver: ConPTY is only available on Windows (GOOS=%s)", runtime.GOOS)
}
