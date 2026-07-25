// Package cli implements the Reinstate command-line interface.
package cli

import "github.com/HarjjotSinghh/reinstate/internal/exitcode"

// Stable process exit codes (public contract).
const (
	ExitOK            = exitcode.OK
	ExitRuntime       = exitcode.Runtime
	ExitUsage         = exitcode.Usage
	ExitConfig        = exitcode.Config
	ExitAuthStorage   = exitcode.AuthStorage
	ExitCompatibility = exitcode.Compatibility
	ExitConflict      = exitcode.Conflict
	ExitSafety        = exitcode.Safety
)

// ExitError carries a stable exit code and message.
type ExitError struct {
	Code    int
	Message string
	Details map[string]any
	Retry   bool
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// NewExitError constructs an ExitError.
func NewExitError(code int, message string) *ExitError {
	return &ExitError{Code: code, Message: message, Details: map[string]any{}}
}

// ExitCodeFrom returns a process exit code for err.
func ExitCodeFrom(err error) int {
	if err == nil {
		return ExitOK
	}
	if ee, ok := err.(*ExitError); ok {
		return ee.Code
	}
	return ExitRuntime
}
