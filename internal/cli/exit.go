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
	// cause is the error this one was built from, when there was one. The
	// message a person reads is Message and nothing else; cause is kept so
	// a caller can still ask what kind of failure it was — whether a
	// storage endpoint was unreachable, say — which a message flattened to
	// a string cannot answer.
	cause error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap exposes the error an ExitError was built from, so errors.Is and
// errors.As still see it.
func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// NewExitError constructs an ExitError.
func NewExitError(code int, message string) *ExitError {
	return &ExitError{Code: code, Message: message, Details: map[string]any{}}
}

// ExitErrorFrom is NewExitError for a failure that came from another
// error, keeping that error reachable through errors.Is and errors.As.
// The printed message is unchanged: err.Error() is what NewExitError
// would have been given.
func ExitErrorFrom(code int, err error) *ExitError {
	return &ExitError{Code: code, Message: err.Error(), Details: map[string]any{}, cause: err}
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
