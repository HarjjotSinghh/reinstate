// Package agentcheck performs bounded, read-only native-agent compatibility
// probes for verified resume.
package agentcheck

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/codex"
)

const (
	defaultTimeout   = 2 * time.Second
	maxVersionOutput = 4 << 10
)

// Status is the current native-agent compatibility state.
type Status string

const (
	StatusSupported    Status = "supported"
	StatusUntested     Status = "untested"
	StatusNotInstalled Status = "not_installed"
	StatusError        Status = "error"
)

// Result contains only privacy-safe native-agent facts.
type Result struct {
	Agent             string `json:"agent"`
	ExecutablePresent bool   `json:"executable_present"`
	Layout            string `json:"layout,omitempty"`
	LayoutRecognized  bool   `json:"layout_recognized"`
	Version           string `json:"version,omitempty"`
	Status            Status `json:"status"`
	Message           string `json:"message"`
}

// VersionRunner runs one fixed vendor --version probe.
type VersionRunner interface {
	Version(context.Context, string, ...string) (string, error)
}

// Options makes filesystem and process boundaries injectable for tests.
type Options struct {
	Home       string
	Root       string
	LookPath   func(string) (string, error)
	Runner     VersionRunner
	Timeout    time.Duration
	MaxOutput  int64
	Executable string
}

// Inspect verifies executable, local session layout, and current version.
func Inspect(ctx context.Context, agentName string, opts Options) Result {
	agentName = strings.ToLower(strings.TrimSpace(agentName))
	result := Result{Agent: agentName, Status: StatusUntested}
	definition, ok := definitions[agentName]
	if !ok {
		result.Message = "agent does not support native verified resume"
		return result
	}

	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	executable := opts.Executable
	if executable == "" {
		executable = definition.executable
	}
	resolved, err := lookPath(executable)
	if err != nil || strings.TrimSpace(resolved) == "" {
		result.Status = StatusNotInstalled
		result.Message = "native agent executable is unavailable"
		return result
	}
	result.ExecutablePresent = true

	root := opts.Root
	if root == "" {
		home := opts.Home
		if home == "" {
			home, err = os.UserHomeDir()
			if err != nil {
				result.Status = StatusError
				result.Message = "user home is unavailable for agent layout inspection"
				return result
			}
		}
		for _, candidate := range definition.roots(home) {
			if isDirectory(candidate) {
				root = candidate
				break
			}
		}
	}
	result.Layout = definition.layout
	if root == "" || !isDirectory(filepath.Join(root, definition.marker)) {
		result.Message = "native agent session layout is unrecognized"
		return result
	}
	result.LayoutRecognized = true

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{MaxOutput: opts.MaxOutput}
	}
	output, err := runner.Version(probeCtx, resolved, "--version")
	if err != nil {
		result.Message = "native agent version probe failed"
		return result
	}
	version := adapter.StableVersionFromOutput(output)
	if version == "" {
		result.Message = "native agent version is unrecognized"
		return result
	}
	result.Version = version
	if !definition.supported(version) {
		result.Message = "native agent version is outside the verified range"
		return result
	}
	result.Status = StatusSupported
	result.Message = "native agent executable, version, and session layout are supported"
	return result
}

type definition struct {
	executable string
	layout     string
	marker     string
	roots      func(string) []string
	supported  func(string) bool
}

var definitions = map[string]definition{
	"claude": {
		executable: "claude",
		layout:     "projects-jsonl",
		marker:     "projects",
		roots: func(home string) []string {
			return []string{filepath.Join(home, ".claude"), filepath.Join(home, ".config", "claude")}
		},
		supported: claude.SupportedVersion,
	},
	"codex": {
		executable: "codex",
		layout:     "sessions-rollout-jsonl",
		marker:     "sessions",
		roots: func(home string) []string {
			return []string{filepath.Join(home, ".codex"), filepath.Join(home, ".config", "codex")}
		},
		supported: codex.SupportedVersion,
	},
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ExecRunner is a shell-free, output-bounded vendor version runner.
type ExecRunner struct {
	MaxOutput int64
}

// Version runs the exact executable and fixed arguments without stdin.
func (runner ExecRunner) Version(ctx context.Context, executable string, args ...string) (string, error) {
	limit := runner.MaxOutput
	if limit <= 0 || limit > maxVersionOutput {
		limit = maxVersionOutput
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdin = nil
	var output boundedBuffer
	output.limit = limit
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return "", err
	}
	if output.overflow {
		return "", errors.New("native agent version output exceeded limit")
	}
	return output.String(), nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int64
	overflow bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := buffer.limit - int64(buffer.Len())
	if remaining <= 0 {
		buffer.overflow = true
		return original, nil
	}
	if int64(len(value)) > remaining {
		value = value[:remaining]
		buffer.overflow = true
	}
	_, _ = buffer.Buffer.Write(value)
	return original, nil
}
