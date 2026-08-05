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
	"regexp"
	"strings"
	"time"

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

// VersionOutput keeps the two process streams separate so a warning written to
// stderr cannot be mistaken for authoritative version output.
type VersionOutput struct {
	Stdout string
	Stderr string
}

// VersionRunner runs one fixed vendor --version probe.
type VersionRunner interface {
	Version(context.Context, string, ...string) (VersionOutput, error)
}

// Options makes filesystem and process boundaries injectable for tests.
type Options struct {
	Home       string
	Root       string
	LookPath   func(string) (string, error)
	Getenv     func(string) string
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
		getenv := opts.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		root = getenv(definition.rootEnvironment)
	}
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
			if layoutCandidateExists(candidate) {
				root = candidate
				break
			}
		}
	}
	result.Layout = definition.layout
	if root == "" || !recognizedLayout(root, definition.marker) {
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
		result.Status = StatusError
		result.Message = "native agent version probe failed"
		return result
	}
	version, ok := definition.parseVersion(output)
	if !ok {
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
	executable      string
	layout          string
	marker          string
	rootEnvironment string
	roots           func(string) []string
	parseVersion    func(VersionOutput) (string, bool)
	supported       func(string) bool
}

var definitions = map[string]definition{
	"claude": {
		executable:      "claude",
		layout:          "projects-jsonl",
		marker:          "projects",
		rootEnvironment: "CLAUDE_CONFIG_DIR",
		roots: func(home string) []string {
			return []string{filepath.Join(home, ".claude"), filepath.Join(home, ".config", "claude")}
		},
		parseVersion: parseClaudeVersion,
		supported:    claude.SupportedVersion,
	},
	"codex": {
		executable:      "codex",
		layout:          "sessions-rollout-jsonl",
		marker:          "sessions",
		rootEnvironment: "CODEX_HOME",
		roots: func(home string) []string {
			return []string{filepath.Join(home, ".codex"), filepath.Join(home, ".config", "codex")}
		},
		parseVersion: parseCodexVersion,
		supported:    codex.SupportedVersion,
	},
}

var (
	claudeVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)) \(Claude Code\)$`)
	codexVersionPattern  = regexp.MustCompile(`^codex-cli ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)
)

func parseClaudeVersion(output VersionOutput) (string, bool) {
	return parseVersionLine(output, claudeVersionPattern)
}

func parseCodexVersion(output VersionOutput) (string, bool) {
	return parseVersionLine(output, codexVersionPattern)
}

func parseVersionLine(output VersionOutput, pattern *regexp.Regexp) (string, bool) {
	if output.Stderr != "" {
		return "", false
	}
	line, ok := oneVersionLine(output.Stdout)
	if !ok {
		return "", false
	}
	matches := pattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

func oneVersionLine(output string) (string, bool) {
	if strings.HasSuffix(output, "\r\n") {
		output = strings.TrimSuffix(output, "\r\n")
	} else if strings.HasSuffix(output, "\n") {
		output = strings.TrimSuffix(output, "\n")
	}
	if output == "" || strings.ContainsAny(output, "\r\n") {
		return "", false
	}
	for _, character := range output {
		if character < 0x20 || character == 0x7f {
			return "", false
		}
	}
	return output, true
}

func layoutCandidateExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// recognizedLayout opens the marker relative to an os.Root and compares the
// object before, during, and after opening. The marker and the root itself must
// be real directories, never symlinks. This fails closed if either path is
// replaced while it is being inspected.
func recognizedLayout(root, marker string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rootBefore, err := os.Lstat(rootAbs)
	if err != nil || !rootBefore.IsDir() || rootBefore.Mode()&os.ModeSymlink != 0 {
		return false
	}
	rootHandle, err := os.OpenRoot(rootAbs)
	if err != nil {
		return false
	}
	defer rootHandle.Close()

	openedRoot, err := rootHandle.Open(".")
	if err != nil {
		return false
	}
	openedRootInfo, statErr := openedRoot.Stat()
	closeErr := openedRoot.Close()
	if statErr != nil || closeErr != nil || !openedRootInfo.IsDir() || !os.SameFile(rootBefore, openedRootInfo) {
		return false
	}

	markerBefore, err := rootHandle.Lstat(marker)
	if err != nil || !markerBefore.IsDir() || markerBefore.Mode()&os.ModeSymlink != 0 {
		return false
	}
	openedMarker, err := rootHandle.Open(marker)
	if err != nil {
		return false
	}
	openedMarkerInfo, statErr := openedMarker.Stat()
	closeErr = openedMarker.Close()
	if statErr != nil || closeErr != nil || !openedMarkerInfo.IsDir() || !os.SameFile(markerBefore, openedMarkerInfo) {
		return false
	}
	markerAfter, err := rootHandle.Lstat(marker)
	if err != nil || markerAfter.Mode()&os.ModeSymlink != 0 || !os.SameFile(markerBefore, markerAfter) {
		return false
	}
	rootAfter, err := os.Lstat(rootAbs)
	return err == nil && rootAfter.Mode()&os.ModeSymlink == 0 && os.SameFile(rootBefore, rootAfter)
}

// ExecRunner is a shell-free, output-bounded vendor version runner.
type ExecRunner struct {
	MaxOutput int64
}

// Version runs the exact executable and fixed arguments without stdin.
func (runner ExecRunner) Version(ctx context.Context, executable string, args ...string) (VersionOutput, error) {
	limit := runner.MaxOutput
	if limit <= 0 || limit > maxVersionOutput {
		limit = maxVersionOutput
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdin = nil
	stdout := boundedBuffer{limit: limit}
	stderr := boundedBuffer{limit: limit}
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return VersionOutput{}, err
	}
	if stdout.overflow || stderr.overflow {
		return VersionOutput{}, errors.New("native agent version output exceeded limit")
	}
	return VersionOutput{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	overflow bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := buffer.limit - int64(buffer.buffer.Len())
	if remaining <= 0 {
		buffer.overflow = true
		return original, nil
	}
	if int64(len(value)) > remaining {
		value = value[:remaining]
		buffer.overflow = true
	}
	_, _ = buffer.buffer.Write(value)
	return original, nil
}

func (buffer *boundedBuffer) String() string {
	return buffer.buffer.String()
}
