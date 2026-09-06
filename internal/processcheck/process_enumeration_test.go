package processcheck

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// TestEnumerationFailureIsReportedNotReadAsIdle pins the fail-safe the
// v0.6.0-rc.1 pre-tag Windows run found missing: on a host whose process
// enumeration fails (a broken WMI repository made both Get-CimInstance and
// tasklist exit with "Critical error"), SessionBusy answered "not busy" with
// no error, so preflight reported a confident "no running instance is using
// this session" for a session that was open. The error must reach the
// caller, which reports the check as one that could not run.
func TestEnumerationFailureIsReportedNotReadAsIdle(t *testing.T) {
	previous := enumerateProcesses
	t.Cleanup(func() { enumerateProcesses = previous })
	boom := errors.New("Critical error")
	enumerateProcesses = func(context.Context) ([]Process, error) { return nil, boom }

	busy, scoped, err := SessionBusy(context.Background(), "claude", Target{
		Path:      filepath.Join(t.TempDir(), "session.jsonl"),
		SessionID: "0e6586d4-3739-47cd-bad3-395939e10d53",
	})
	if !errors.Is(err, boom) {
		t.Fatalf("SessionBusy error = %v, want the enumeration failure", err)
	}
	if busy || scoped {
		t.Fatalf("SessionBusy = busy %v scoped %v alongside an error; want neither claimed", busy, scoped)
	}

	active, err := AgentActive(context.Background(), "claude")
	if !errors.Is(err, boom) {
		t.Fatalf("AgentActive error = %v, want the enumeration failure", err)
	}
	if active {
		t.Fatal("AgentActive claimed a running agent alongside an error")
	}
}
