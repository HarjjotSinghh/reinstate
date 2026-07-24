// Package version holds build-time version metadata.
package version

// Version is set via -ldflags at release build time.
var Version = "0.0.0-dev"

// Commit is the git SHA when built from CI (optional).
var Commit = "unknown"

// Date is the build date (optional).
var Date = "unknown"

// String returns a human-readable version line.
func String() string {
	return "reinstate " + Version + " (" + Commit + " " + Date + ")"
}
