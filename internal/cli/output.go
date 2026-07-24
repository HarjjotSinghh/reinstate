package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Streams for injection in tests.
type Streams struct {
	Out io.Writer
	Err io.Writer
}

func defaultStreams() Streams {
	return Streams{Out: os.Stdout, Err: os.Stderr}
}

// WriteJSON encodes v as JSON without ANSI and appends a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintHuman writes a plain human line to w.
func PrintHuman(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format+"\n", args...)
}

// ErrorJSON is the stable JSON error envelope.
type ErrorJSON struct {
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Details     map[string]any `json:"details,omitempty"`
	SafeToRetry bool           `json:"safe_to_retry"`
}

func codeName(code int) string {
	switch code {
	case ExitUsage:
		return "usage"
	case ExitConfig:
		return "config"
	case ExitAuthStorage:
		return "auth_storage"
	case ExitCompatibility:
		return "compatibility"
	case ExitConflict:
		return "conflict"
	case ExitSafety:
		return "safety"
	case ExitRuntime:
		return "runtime"
	default:
		return "ok"
	}
}

// WriteError emits human or JSON error output.
func WriteError(w io.Writer, jsonMode bool, err error) {
	if err == nil {
		return
	}
	ee, ok := err.(*ExitError)
	if !ok {
		ee = NewExitError(ExitRuntime, err.Error())
	}
	if jsonMode {
		_ = WriteJSON(w, ErrorJSON{
			Code:        codeName(ee.Code),
			Message:     ee.Message,
			Details:     ee.Details,
			SafeToRetry: ee.Retry,
		})
		return
	}
	fmt.Fprintln(w, ee.Message)
}
