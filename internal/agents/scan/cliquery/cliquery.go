// Package cliquery is the shared F2 scanner: a bounded subprocess with a
// timeout and an output ceiling, JSON decode, and non-zero-exit handling.
//
// It is read-only. It reuses sessionindex.MaxJSONLineBytes as the output
// ceiling and does not introduce a new one.
package cliquery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Existing ceilings, reused so a new F2 agent cannot introduce an unbounded read.
const (
	MaxJSONLineBytes   = sessionindex.MaxJSONLineBytes
	MaxSearchTextBytes = sessionindex.MaxSearchTextBytes
	MaxFileReferences  = sessionindex.MaxFileReferences
)

// DefaultTimeout is the OpenCode-derived bound for one vendor CLI query.
const DefaultTimeout = 5 * time.Second

// ErrOutputTooLarge means the child wrote more than MaxJSONLineBytes.
var ErrOutputTooLarge = errors.New("cliquery: command output exceeds MaxJSONLineBytes")

// Runner is the injectable command surface used by F2 sources.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RunnerFunc adapts a function to Runner.
type RunnerFunc func(context.Context, string, ...string) ([]byte, error)

// Run calls the function.
func (fn RunnerFunc) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return fn(ctx, name, args...)
}

// Config bounds one vendor CLI query.
type Config struct {
	Timeout   time.Duration
	MaxOutput int
	Runner    Runner
}

// Run executes name with args under a timeout and an output ceiling. A
// non-zero exit is returned to the caller; the scanner does not interpret it.
func Run(ctx context.Context, name string, args []string, cfg Config) ([]byte, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxOutput := cfg.MaxOutput
	if maxOutput <= 0 || maxOutput > MaxJSONLineBytes {
		maxOutput = MaxJSONLineBytes
	}
	runner := cfg.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	output, err := runner.Run(runCtx, name, args...)
	if err != nil {
		return nil, err
	}
	if len(output) > maxOutput {
		return nil, fmt.Errorf("%w: maximum is %d bytes", ErrOutputTooLarge, maxOutput)
	}
	return output, nil
}

// ExecRunner runs a local executable without a shell and drains stdout up to
// the shared ceiling.
type ExecRunner struct {
	MaxOutput int
}

// Run executes the child. Output above the ceiling is discarded after the
// cap and reported as ErrOutputTooLarge.
func (r ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	maxOutput := r.MaxOutput
	if maxOutput <= 0 || maxOutput > MaxJSONLineBytes {
		maxOutput = MaxJSONLineBytes
	}
	command := exec.CommandContext(ctx, name, args...)
	output := boundedOutput{remaining: maxOutput}
	command.Stdout = &output
	err := command.Run()
	if output.exceeded {
		return nil, fmt.Errorf("%w: maximum is %d bytes", ErrOutputTooLarge, maxOutput)
	}
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// DecodeJSON decodes a bounded vendor JSON payload into dest.
func DecodeJSON(data []byte, dest any) error {
	if dest == nil {
		return errors.New("cliquery: decode destination must not be nil")
	}
	if len(data) > MaxJSONLineBytes {
		return fmt.Errorf("%w: maximum is %d bytes", ErrOutputTooLarge, MaxJSONLineBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	return nil
}

type boundedOutput struct {
	bytes.Buffer
	remaining int
	exceeded  bool
}

func (o *boundedOutput) Write(value []byte) (int, error) {
	length := len(value)
	if length > o.remaining {
		o.exceeded = true
		value = value[:max(o.remaining, 0)]
	}
	if len(value) > 0 {
		_, _ = o.Buffer.Write(value)
		o.remaining -= len(value)
	}
	// Report the original length so os/exec continues draining the child's
	// stdout without retaining bytes beyond the cap.
	return length, nil
}
