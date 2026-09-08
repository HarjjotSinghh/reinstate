//go:build !windows

package main

// restrictSecretFileACL is a no-op on POSIX: os.WriteFile's 0o600 mode
// (state.go's saveRecoveryCode) already creates a real owner-only
// permission bitmask there -- the process umask can only narrow it
// further, never widen it -- so there is no separate ACL step needed to
// make good on the "owner only" claim in the hoplab README and
// docs/testing/windows-acceptance-host.md. This stub only keeps `go build
// ./...`/`go vet ./...` clean on every non-Windows GOOS, the same
// convention secretfd_other.go and registry_other.go already use.
func restrictSecretFileACL(string) error { return nil }
