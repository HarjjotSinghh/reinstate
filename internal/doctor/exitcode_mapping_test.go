package doctor

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
)

// TestExitCodeMappingIsDeterministic pins the report-to-exit-code contract
// without probing the host. The CLI-level setup-check test can only assert that
// the process agrees with the report it printed, because which vendor CLIs are
// installed changes which checks fail; the exact mapping belongs here.
func TestExitCodeMappingIsDeterministic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		checks []Check
		want   int
	}{
		{
			name:   "no checks",
			checks: nil,
			want:   exitcode.OK,
		},
		{
			name: "everything passing or skipped",
			checks: []Check{
				{Name: "config", Status: "ok"},
				{Name: "agent.codex", Status: "skip", Message: "not installed"},
				{Name: "keyring", Status: "warn"},
			},
			want: exitcode.OK,
		},
		{
			name: "only the config check fails",
			checks: []Check{
				{Name: "config", Status: "fail", Code: exitcode.Config},
				{Name: "agent.claude", Status: "ok"},
			},
			want: exitcode.Config,
		},
		{
			name: "an untested agent outranks a failing config",
			checks: []Check{
				{Name: "config", Status: "fail", Code: exitcode.Config},
				{Name: "agent.codex", Status: "fail", Code: exitcode.Compatibility},
			},
			want: exitcode.Compatibility,
		},
		{
			name: "a failing check without a code is a runtime failure",
			checks: []Check{
				{Name: "config", Status: "fail", Code: exitcode.Config},
				{Name: "self_test", Status: "fail"},
			},
			want: exitcode.Runtime,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCode(&Report{Checks: test.checks}); got != test.want {
				t.Fatalf("ExitCode() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestExitCodeOfNilReportIsRuntime(t *testing.T) {
	t.Parallel()
	if got := ExitCode(nil); got != exitcode.Runtime {
		t.Fatalf("ExitCode(nil) = %d, want %d", got, exitcode.Runtime)
	}
}

// TestAdapterCheckStatusPerCompatibility pins the per-agent mapping that made a
// supported-but-undetected vendor look like a product failure on one host.
func TestAdapterCheckStatusPerCompatibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		compat     adapter.Compatibility
		wantStatus string
		wantCode   int
	}{
		{name: "supported", compat: adapter.CompatibilitySupported, wantStatus: "ok", wantCode: 0},
		{name: "not installed", compat: adapter.CompatibilityNotInstalled, wantStatus: "skip", wantCode: 0},
		{name: "untested", compat: adapter.CompatibilityUntested, wantStatus: "fail", wantCode: exitcode.Compatibility},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			install := adapter.Install{Agent: "codex", Version: "0.145.0"}
			got := adapterCheck("codex", install, test.compat, nil)
			if got.Status != test.wantStatus || got.Code != test.wantCode {
				t.Fatalf("status/code = %q/%d, want %q/%d",
					got.Status, got.Code, test.wantStatus, test.wantCode)
			}
			if got.Name != "agent.codex" {
				t.Fatalf("check name = %q, want agent.codex", got.Name)
			}
		})
	}
}
