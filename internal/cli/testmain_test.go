package cli

import (
	"os"
	"testing"
)

// TestMain neutralizes two environment variables that resolve straight from
// os.Getenv in product code rather than through a journey's isolated home,
// so a developer machine that happens to export either one for its own
// local tooling does not leak real, non-fixture state into every journey
// in this package that pushes or pulls "--all":
//
//   - XDG_DATA_HOME names OpenCode's data root
//     (internal/adapter/opencode.Adapter.resolveRoot). A real OpenCode
//     install on the host, or an unrelated worktree exporting this for its
//     own purposes, hands push/pull a real session none of these journeys
//     planted, which then diverges from whatever the fixture side expects.
//   - REINSTATE_MEMORY_BACKEND_DIR names where the disk-backed "memory"
//     backend keeps objects (memoryBackendRoot in commands_impl.go). It
//     exists so two simulated devices in one journey can share a store on
//     purpose; left ambient, it instead points every "memory" journey that
//     forgets to pin it at the same real, cross-run directory, so a
//     supposedly first-ever push collides with whatever an earlier run (or
//     another tool) already left there. root_test.go already carries this
//     exact warning at one call site — this generalizes it to the package.
//
// Individual tests that want either variable still set it with t.Setenv,
// which scopes the value to that test and restores the prior value (i.e.
// unset, once this has run) afterward. This only removes what every test
// would otherwise inherit unasked from whatever shell invoked `go test`.
func TestMain(m *testing.M) {
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("REINSTATE_MEMORY_BACKEND_DIR")
	os.Exit(m.Run())
}
