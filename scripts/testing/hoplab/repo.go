package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findRepoRoot walks upward from dir (the current working directory by
// default) looking for this module's go.mod, so hoplab can build fakelocker
// and locate testdata fixtures regardless of which subdirectory it is run
// from.
func findRepoRoot(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		if b, err := os.ReadFile(modPath); err == nil {
			if strings.Contains(string(b), "module github.com/HarjjotSinghh/reinstate") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find the reinstate module root above %s; run hoplab from inside the reinstate checkout", dir)
		}
		dir = parent
	}
}
