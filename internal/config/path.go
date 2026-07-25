// Package config loads and saves Reinstate local configuration.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Home returns the Reinstate home directory.
// Override with REINSTATE_HOME (absolute path).
func Home() (string, error) {
	if v := strings.TrimSpace(os.Getenv("REINSTATE_HOME")); v != "" {
		if !filepath.IsAbs(v) {
			return "", errNotAbs(v)
		}
		return filepath.Clean(v), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Windows: %USERPROFILE%\.reinstate — UserHomeDir covers this.
	// macOS/Linux/WSL: ~/.reinstate
	return filepath.Join(userHome, ".reinstate"), nil
}

type pathError string

func (e pathError) Error() string { return string(e) }

func errNotAbs(v string) error {
	return pathError("REINSTATE_HOME must be an absolute path: " + v)
}

// ConfigPath is home/config.toml.
func ConfigPath(home string) string { return filepath.Join(home, "config.toml") }

// StatePath is home/state.json.
func StatePath(home string) string { return filepath.Join(home, "state.json") }

// DevicePath is home/device.json.
func DevicePath(home string) string { return filepath.Join(home, "device.json") }

// NativeWindowsPath reports whether we should use Windows path conventions
// for documentation/tests of home layout (not the host OS).
func NativeWindowsHome(userProfile string) string {
	return filepath.Join(userProfile, ".reinstate")
}

// IsWindows reports host OS windows.
func IsWindows() bool { return runtime.GOOS == "windows" }
