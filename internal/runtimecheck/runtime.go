// Package runtimecheck inspects declared project runtime versions without
// executing project code or package-manager scripts.
package runtimecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	maxDeclarationBytes = 1 << 20
	maxVersionOutput    = 4 << 10
	defaultTimeout      = 2 * time.Second
)

// Status is a privacy-safe runtime comparison result.
type Status string

const (
	StatusMatch   Status = "match"
	StatusChanged Status = "changed"
	StatusMissing Status = "missing"
	StatusUnknown Status = "unknown"
)

// Result describes one deterministic project runtime declaration.
type Result struct {
	Name       string `json:"name"`
	Declared   string `json:"declared,omitempty"`
	Actual     string `json:"actual,omitempty"`
	SourceKind string `json:"source_kind"`
	Status     Status `json:"status"`
	Message    string `json:"message"`
}

// Runner executes a fixed version probe. Implementations must not invoke a
// shell or run project code.
type Runner interface {
	Version(context.Context, string, ...string) (string, error)
}

// Options configures inspection. Its zero value uses local fixed-path reads
// and bounded executable --version probes.
type Options struct {
	Runner  Runner
	Timeout time.Duration
}

// Inspect reports recognized Node and Go declarations below workspace.
func Inspect(ctx context.Context, workspace string, opts Options) []Result {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{MaxOutput: maxVersionOutput}
	}

	var results []Result
	if declaration, source, state := nodeDeclaration(workspace); state != declarationAbsent {
		results = append(results, inspectRuntime(
			probeCtx,
			runner,
			"node",
			[]string{"--version"},
			declaration,
			source,
			state,
			parseNodeOutput,
		))
	}
	if declaration, source, state := goDeclaration(workspace); state != declarationAbsent {
		results = append(results, inspectRuntime(
			probeCtx,
			runner,
			"go",
			[]string{"version"},
			declaration,
			source,
			state,
			parseGoOutput,
		))
	}
	return results
}

type declarationState int

const (
	declarationAbsent declarationState = iota
	declarationValid
	declarationUnknown
)

func inspectRuntime(
	ctx context.Context,
	runner Runner,
	name string,
	args []string,
	declaration versionConstraint,
	source string,
	state declarationState,
	parseOutput func(string) (version, bool),
) Result {
	result := Result{
		Name:       name,
		Declared:   declaration.display,
		SourceKind: source,
		Status:     StatusUnknown,
		Message:    "runtime declaration could not be compared safely",
	}
	if state == declarationUnknown {
		return result
	}
	raw, err := runner.Version(ctx, name, args...)
	if err != nil {
		result.Status = StatusMissing
		result.Message = "declared runtime is unavailable"
		return result
	}
	actual, ok := parseOutput(raw)
	if !ok {
		result.Message = "installed runtime version is unrecognized"
		return result
	}
	result.Actual = actual.String()
	if declaration.match(actual) {
		result.Status = StatusMatch
		result.Message = "installed runtime matches the project declaration"
		return result
	}
	result.Status = StatusChanged
	result.Message = "installed runtime differs from the project declaration"
	return result
}

func nodeDeclaration(workspace string) (versionConstraint, string, declarationState) {
	for _, candidate := range []struct {
		name   string
		source string
	}{
		{name: ".nvmrc", source: "nvmrc"},
		{name: ".node-version", source: "node_version_file"},
	} {
		value, state := readDeclarationFile(filepath.Join(workspace, candidate.name))
		if state == declarationAbsent {
			continue
		}
		constraint, ok := parseConstraint(value)
		if !ok {
			return versionConstraint{}, candidate.source, declarationUnknown
		}
		return constraint, candidate.source, declarationValid
	}

	data, state := readBoundedRegularFile(filepath.Join(workspace, "package.json"))
	if state == declarationAbsent {
		return versionConstraint{}, "", declarationAbsent
	}
	if state == declarationUnknown {
		return versionConstraint{}, "package_json_engines", declarationUnknown
	}
	var shape struct {
		Engines map[string]json.RawMessage `json:"engines"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return versionConstraint{}, "package_json_engines", declarationUnknown
	}
	raw, exists := shape.Engines["node"]
	if !exists {
		return versionConstraint{}, "", declarationAbsent
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return versionConstraint{}, "package_json_engines", declarationUnknown
	}
	constraint, ok := parseConstraint(value)
	if !ok {
		return versionConstraint{}, "package_json_engines", declarationUnknown
	}
	return constraint, "package_json_engines", declarationValid
}

func goDeclaration(workspace string) (versionConstraint, string, declarationState) {
	data, state := readBoundedRegularFile(filepath.Join(workspace, "go.mod"))
	if state != declarationValid {
		return versionConstraint{}, "go_mod", state
	}
	var goValue, toolchainValue string
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "go":
			goValue = fields[1]
		case "toolchain":
			toolchainValue = strings.TrimPrefix(fields[1], "go")
		}
	}
	value := toolchainValue
	source := "go_mod_toolchain"
	if value == "" {
		value = goValue
		source = "go_mod_go"
	}
	if value == "" {
		return versionConstraint{}, "", declarationAbsent
	}
	constraint, ok := parseConstraint(value)
	if !ok {
		return versionConstraint{}, source, declarationUnknown
	}
	// A go directive is a minimum language/toolchain requirement. An explicit
	// toolchain directive is also treated as a minimum so compatible patch
	// releases do not become false mismatches.
	constraint.kind = constraintMinimum
	return constraint, source, declarationValid
}

func readDeclarationFile(path string) (string, declarationState) {
	data, state := readBoundedRegularFile(path)
	if state != declarationValid {
		return "", state
	}
	value := strings.TrimSpace(string(data))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return value, declarationUnknown
	}
	return value, declarationValid
}

func readBoundedRegularFile(path string) ([]byte, declarationState) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, declarationAbsent
		}
		return nil, declarationUnknown
	}
	if !info.Mode().IsRegular() || info.Size() > maxDeclarationBytes {
		return nil, declarationUnknown
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, declarationUnknown
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxDeclarationBytes+1))
	if err != nil || len(data) > maxDeclarationBytes {
		return nil, declarationUnknown
	}
	return data, declarationValid
}

// ExecRunner is the production shell-free, output-bounded version runner.
type ExecRunner struct {
	MaxOutput int64
}

// Version runs name with fixed arguments and returns bounded combined output.
func (r ExecRunner) Version(ctx context.Context, name string, args ...string) (string, error) {
	limit := r.MaxOutput
	if limit <= 0 || limit > maxVersionOutput {
		limit = maxVersionOutput
	}
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = nil
	var output limitedBuffer
	output.limit = limit
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return "", err
	}
	if output.overflow {
		return "", errors.New("runtime version output exceeded limit")
	}
	return output.String(), nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int64
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - int64(b.Len())
	if remaining <= 0 {
		b.overflow = true
		return original, nil
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	_, _ = b.Buffer.Write(p)
	return original, nil
}

type constraintKind int

const (
	constraintExact constraintKind = iota
	constraintMinimum
	constraintMajor
	constraintCaret
	constraintTilde
	constraintMajorRange
)

type versionConstraint struct {
	kind    constraintKind
	version version
	display string
}

func (c versionConstraint) match(actual version) bool {
	comparison := actual.compare(c.version)
	switch c.kind {
	case constraintExact:
		return comparison == 0
	case constraintMinimum:
		return comparison >= 0
	case constraintMajor:
		return actual.major == c.version.major
	case constraintCaret:
		return actual.major == c.version.major && comparison >= 0
	case constraintTilde:
		return actual.major == c.version.major && actual.minor == c.version.minor && comparison >= 0
	case constraintMajorRange:
		return actual.major == c.version.major
	default:
		return false
	}
}

var majorRangePattern = regexp.MustCompile(`^>=?\s*v?([0-9]+)(?:\.0(?:\.0)?)?\s+<\s*v?([0-9]+)(?:\.0(?:\.0)?)?$`)

func parseConstraint(raw string) (versionConstraint, bool) {
	value := strings.TrimSpace(raw)
	display := safeVersionText(value)
	if match := majorRangePattern.FindStringSubmatch(value); len(match) == 3 {
		lower, _ := strconv.Atoi(match[1])
		upper, _ := strconv.Atoi(match[2])
		if upper == lower+1 {
			return versionConstraint{
				kind: constraintMajorRange, version: version{major: lower}, display: display,
			}, true
		}
	}
	if strings.HasSuffix(value, ".x") || strings.HasSuffix(value, ".*") {
		majorText := strings.TrimSuffix(strings.TrimSuffix(value, ".x"), ".*")
		major, err := strconv.Atoi(strings.TrimPrefix(majorText, "v"))
		if err == nil && major >= 0 {
			return versionConstraint{kind: constraintMajor, version: version{major: major}, display: display}, true
		}
	}

	kind := constraintExact
	switch {
	case strings.HasPrefix(value, ">="):
		kind = constraintMinimum
		value = strings.TrimSpace(strings.TrimPrefix(value, ">="))
	case strings.HasPrefix(value, "^"):
		kind = constraintCaret
		value = strings.TrimSpace(strings.TrimPrefix(value, "^"))
	case strings.HasPrefix(value, "~"):
		kind = constraintTilde
		value = strings.TrimSpace(strings.TrimPrefix(value, "~"))
	}
	parsed, components, ok := parseVersion(value)
	if !ok {
		return versionConstraint{}, false
	}
	if kind == constraintExact && components == 1 {
		kind = constraintMajor
	}
	return versionConstraint{kind: kind, version: parsed, display: display}, true
}

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(raw string) (version, int, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if index := strings.IndexAny(value, "+-"); index >= 0 {
		value = value[:index]
	}
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return version{}, 0, false
	}
	numbers := [3]int{}
	for index, part := range parts {
		if part == "" {
			return version{}, 0, false
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return version{}, 0, false
		}
		numbers[index] = number
	}
	return version{major: numbers[0], minor: numbers[1], patch: numbers[2]}, len(parts), true
}

func (v version) compare(other version) int {
	if v.major != other.major {
		return v.major - other.major
	}
	if v.minor != other.minor {
		return v.minor - other.minor
	}
	return v.patch - other.patch
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func parseNodeOutput(raw string) (version, bool) {
	fields := strings.Fields(raw)
	if len(fields) != 1 {
		return version{}, false
	}
	parsed, _, ok := parseVersion(fields[0])
	return parsed, ok
}

func parseGoOutput(raw string) (version, bool) {
	for _, field := range strings.Fields(raw) {
		if strings.HasPrefix(field, "go1.") {
			parsed, _, ok := parseVersion(strings.TrimPrefix(field, "go"))
			return parsed, ok
		}
	}
	return version{}, false
}

func safeVersionText(value string) string {
	value = strings.Map(func(r rune) rune {
		switch {
		case r == '\u001b', unicode.IsControl(r), unicode.In(r, unicode.Cf):
			return -1
		default:
			return r
		}
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 128 {
		value = string([]rune(value)[:128])
	}
	return value
}
