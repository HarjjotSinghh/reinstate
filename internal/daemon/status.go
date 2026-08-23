// Package daemon is the resident per-device process behind `rein daemon`:
// it watches every detected agent's session store, pushes after a session
// changes, pulls on a schedule, and surfaces pending device approvals. The
// loop is pure (clock, file events, sync, and notifications are injected);
// the CLI wires the production pieces and the service managers register it
// with launchd, Task Scheduler, or systemd --user.
//
// The daemon sends nothing anywhere that push and pull do not already send
// (ADR 0008: no telemetry). Its only outputs are the locker, the local
// status file, the log, and OS notifications on this machine.
package daemon

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// Dir is the daemon's private directory under the Reinstate home.
func Dir(home string) string { return filepath.Join(home, "daemon") }

// StatusPath is the status file the TUI and the next `rein` invocation read.
func StatusPath(home string) string { return filepath.Join(Dir(home), "status.json") }

// LockPath is the single-instance lock.
func LockPath(home string) string { return filepath.Join(Dir(home), "daemon.lock") }

// LogPath is the current log file; rotated copies carry a numeric suffix.
func LogPath(home string) string { return filepath.Join(Dir(home), "daemon.log") }

// Outcome is the last result of one kind of action (push or pull).
type Outcome struct {
	// At is when the action last completed (success or failure).
	At time.Time `json:"at"`
	// OK reports whether the last attempt succeeded.
	OK bool `json:"ok"`
	// Summary is a one-line human account ("pushed 2 snapshots").
	Summary string `json:"summary,omitempty"`
	// Error is the last failure, kept until the next success.
	Error string `json:"error,omitempty"`
	// Conflict reports that the last attempt stopped on a recorded
	// conflict; rein conflicts resolves it and the daemon never overwrites.
	Conflict bool `json:"conflict,omitempty"`
	// LastOK is when the action last succeeded, surviving later failures.
	LastOK time.Time `json:"last_ok,omitempty"`
}

// PendingApproval is one device waiting for rein devices approve.
type PendingApproval struct {
	RequestID  string    `json:"request_id"`
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Platform   string    `json:"platform,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Device is an enrolled device as the control plane lists it.
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform,omitempty"`
	// This marks the device the daemon runs on.
	This bool `json:"this,omitempty"`
}

// Status is what the daemon writes after every action. Readers treat a
// stale UpdatedAt (older than StaleAfter) as "daemon not running".
type Status struct {
	Version   int       `json:"version"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Watch is "fsnotify" or "poll".
	Watch string `json:"watch"`
	// Roots are the session directories being watched.
	Roots []string `json:"roots,omitempty"`
	// Backend is "hop" or "byo"; approvals and devices exist for hop only.
	Backend string `json:"backend"`

	Push Outcome `json:"push"`
	Pull Outcome `json:"pull"`

	// Pending lists devices waiting for approval right now.
	Pending []PendingApproval `json:"pending_approvals"`
	// Devices lists the account's enrolled devices (hop only).
	Devices []Device `json:"devices,omitempty"`
	// ApprovalsError is the last failure to reach the control plane.
	ApprovalsError string `json:"approvals_error,omitempty"`
	// ApprovalsAt is when the pending list was last refreshed.
	ApprovalsAt time.Time `json:"approvals_at,omitempty"`
}

// StatusVersion is the status file format.
const StatusVersion = 1

// StaleAfter is how old a status file may be before readers treat the
// daemon as stopped. The loop refreshes the file at least every
// HeartbeatEvery, so a live daemon never looks stale.
const StaleAfter = 3 * time.Minute

// HeartbeatEvery bounds the gap between status writes while idle.
const HeartbeatEvery = time.Minute

// ErrNoStatus reports that no daemon has ever written a status file here.
var ErrNoStatus = errors.New("no daemon status")

// ReadStatus loads the status file. A missing file is ErrNoStatus.
func ReadStatus(home string) (Status, error) {
	raw, err := os.ReadFile(StatusPath(home))
	if os.IsNotExist(err) {
		return Status{}, ErrNoStatus
	}
	if err != nil {
		return Status{}, err
	}
	var s Status
	if err := json.Unmarshal(raw, &s); err != nil {
		return Status{}, err
	}
	return s, nil
}

// Write stores the status atomically (temp file, rename) so a reader never
// sees a half-written file.
func (s Status) Write(home string) error {
	if err := os.MkdirAll(Dir(home), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := StatusPath(home)
	tmp, err := os.CreateTemp(Dir(home), ".status-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = fsx.ProtectOwnerOnly(tmpPath, false)
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// Alive reports whether the status describes a daemon that is still
// running as of now: the heartbeat is recent and the process exists.
func (s Status) Alive(now time.Time) bool {
	if s.PID == 0 || s.UpdatedAt.IsZero() {
		return false
	}
	if now.Sub(s.UpdatedAt) > StaleAfter {
		return false
	}
	return processExists(s.PID)
}
