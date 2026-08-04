package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const maxGitOutputBytes = 4 << 20

var (
	ErrGitUnavailable   = errors.New("git executable is unavailable")
	ErrOutputTooLarge   = errors.New("git command output exceeds the safe limit")
	ErrNotRepository    = errors.New("workspace is not a Git repository")
	ErrUnsafeGitCommand = errors.New("Git command is outside the read-only probe allowlist")
)

type GitRunner interface {
	Run(ctx context.Context, dir string, args ...string) ([]byte, error)
}

type GitRunnerFunc func(context.Context, string, ...string) ([]byte, error)

func (runner GitRunnerFunc) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return runner(ctx, dir, args...)
}

type CommandError struct {
	ExitCode      int
	NotRepository bool
}

func (err *CommandError) Error() string {
	return fmt.Sprintf("git command exited with status %d", err.ExitCode)
}

func (err *CommandError) Unwrap() error {
	if err.NotRepository {
		return ErrNotRepository
	}
	return nil
}

type ExecGitRunner struct{}

func (ExecGitRunner) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if !allowedGitProbe(args) {
		return nil, ErrUnsafeGitCommand
	}
	executable, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrGitUnavailable
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = dir
	command.Env = safeGitEnvironment(os.Environ())
	var stdout boundedBuffer
	var stderr boundedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, ErrOutputTooLarge
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, &CommandError{
				ExitCode:      exitErr.ExitCode(),
				NotRepository: strings.Contains(strings.ToLower(stderr.String()), "not a git repository"),
			}
		}
		return nil, fmt.Errorf("execute git command: %w", err)
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func allowedGitProbe(args []string) bool {
	if stringSlicesEqual(args, gitStatusArgs) ||
		stringSlicesEqual(args, []string{"rev-parse", "--path-format=absolute", "--show-toplevel"}) ||
		stringSlicesEqual(args, []string{"rev-parse", "--is-shallow-repository"}) ||
		stringSlicesEqual(args, []string{"config", "--null", "--get-regexp", `^remote\..*\.url$`}) ||
		stringSlicesEqual(args, []string{"rev-list", "--max-parents=0", "HEAD"}) {
		return true
	}
	if len(args) != 4 || args[0] != "rev-list" || args[1] != "--left-right" || args[2] != "--count" {
		return false
	}
	left, right, ok := strings.Cut(args[3], "...")
	return ok && validObjectID(left) && validObjectID(right)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

type boundedBuffer struct {
	bytes.Buffer
	exceeded bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	length := len(value)
	remaining := maxGitOutputBytes - buffer.Len()
	if length > remaining {
		buffer.exceeded = true
		value = value[:max(remaining, 0)]
	}
	if len(value) > 0 {
		_, _ = buffer.Buffer.Write(value)
	}
	// Continue draining child output without retaining bytes beyond the cap.
	return length, nil
}

func safeGitEnvironment(environment []string) []string {
	blocked := map[string]struct{}{
		"GIT_DIR": {}, "GIT_WORK_TREE": {}, "GIT_INDEX_FILE": {},
		"GIT_COMMON_DIR": {}, "GIT_OBJECT_DIRECTORY": {}, "GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
		"GIT_SHALLOW_FILE": {}, "GIT_NAMESPACE": {}, "GIT_REPLACE_REF_BASE": {},
		"GIT_CEILING_DIRECTORIES": {}, "GIT_DISCOVERY_ACROSS_FILESYSTEM": {},
		"GIT_CONFIG": {}, "GIT_CONFIG_GLOBAL": {}, "GIT_CONFIG_SYSTEM": {},
		"GIT_CONFIG_NOSYSTEM": {}, "GIT_CONFIG_PARAMETERS": {}, "GIT_EXEC_PATH": {},
		"GIT_CONFIG_COUNT": {}, "GIT_CONFIG_KEY_0": {}, "GIT_CONFIG_VALUE_0": {},
	}
	result := make([]string, 0, len(environment)+4)
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		upperKey := strings.ToUpper(key)
		if _, skip := blocked[upperKey]; skip ||
			strings.HasPrefix(upperKey, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(upperKey, "GIT_CONFIG_VALUE_") ||
			strings.HasPrefix(upperKey, "GIT_TRACE") {
			continue
		}
		switch upperKey {
		case "GIT_OPTIONAL_LOCKS", "GIT_TERMINAL_PROMPT", "GIT_NO_REPLACE_OBJECTS", "LC_ALL":
			continue
		}
		result = append(result, entry)
	}
	return append(result,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"LC_ALL=C",
	)
}

func commandExitCode(err error) (int, bool) {
	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		return 0, false
	}
	return commandErr.ExitCode, true
}
