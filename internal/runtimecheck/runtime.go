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
	"math"
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

var (
	// ErrExecutableNotFound identifies a runtime that is not available on PATH.
	ErrExecutableNotFound = errors.New("runtime executable is unavailable")
	// ErrProbeFailed identifies a version probe infrastructure failure. It is
	// deliberately distinct from an absent runtime so policy cannot silently
	// turn a timeout, cancellation, or malformed execution into a compatibility
	// warning.
	ErrProbeFailed = errors.New("runtime version probe failed")
	// ErrOutputLimit identifies a version command that exceeded the bounded
	// output budget.
	ErrOutputLimit = errors.New("runtime version output exceeded limit")
)

// Status is a privacy-safe runtime comparison result.
type Status string

const (
	StatusMatch   Status = "match"
	StatusChanged Status = "changed"
	StatusMissing Status = "missing"
	StatusUnknown Status = "unknown"
	StatusError   Status = "error"
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
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return []Result{}
	}
	workspaceInfo, err := os.Stat(workspace)
	if err != nil || !workspaceInfo.IsDir() {
		return []Result{}
	}

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
		if errors.Is(err, ErrExecutableNotFound) {
			result.Status = StatusMissing
			result.Message = "declared runtime is unavailable"
			return result
		}
		result.Status = StatusError
		result.Message = "runtime version probe failed"
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
	var sawGo, sawToolchain bool
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := rawLine
		if comment := strings.Index(line, "//"); comment >= 0 {
			line = line[:comment]
		}
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "go":
			if sawGo || len(fields) != 2 {
				return versionConstraint{}, "go_mod_go", declarationUnknown
			}
			sawGo = true
			goValue = fields[1]
		case "toolchain":
			if sawToolchain || len(fields) != 2 {
				return versionConstraint{}, "go_mod_toolchain", declarationUnknown
			}
			sawToolchain = true
			toolchainValue = fields[1]
		}
	}
	value := ""
	source := "go_mod_toolchain"
	switch {
	case toolchainValue == "default":
		value = goValue
		source = "go_mod_go"
	case toolchainValue != "":
		if !strings.HasPrefix(toolchainValue, "go") || toolchainValue == "go" {
			return versionConstraint{}, source, declarationUnknown
		}
		value = strings.TrimPrefix(toolchainValue, "go")
	default:
		value = goValue
		source = "go_mod_go"
	}
	if value == "" {
		return versionConstraint{}, "", declarationAbsent
	}
	parsed, components, ok := parseVersion(value)
	if !ok || components < 2 || strings.HasPrefix(value, "v") {
		return versionConstraint{}, source, declarationUnknown
	}
	// A go directive is a minimum language/toolchain requirement. An explicit
	// toolchain directive is also treated as a minimum so compatible patch
	// releases do not become false mismatches.
	return versionConstraint{
		kind: constraintMinimum, version: parsed, display: safeVersionText(value),
	}, source, declarationValid
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
	return readBoundedRegularFileWithOpener(path, os.Open)
}

func readBoundedRegularFileWithOpener(path string, opener func(string) (*os.File, error)) ([]byte, declarationState) {
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
	file, err := opener(path)
	if err != nil {
		return nil, declarationUnknown
	}
	defer func() { _ = file.Close() }()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || openedInfo.Size() > maxDeclarationBytes || !os.SameFile(info, openedInfo) {
		return nil, declarationUnknown
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDeclarationBytes+1))
	if err != nil || len(data) > maxDeclarationBytes {
		return nil, declarationUnknown
	}
	finalInfo, err := file.Stat()
	if err != nil || !os.SameFile(openedInfo, finalInfo) || finalInfo.Size() != openedInfo.Size() {
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
	probe, ok := knownProbe(name, args)
	if !ok {
		return "", fmt.Errorf("%w: unsupported runtime probe", ErrProbeFailed)
	}
	resolved, err := exec.LookPath(name)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrExecutableNotFound, probe)
		}
		return "", fmt.Errorf("%w: executable lookup", ErrProbeFailed)
	}
	command := exec.CommandContext(ctx, resolved, args...)
	command.Stdin = nil
	command.Dir = neutralWorkingDirectory(resolved)
	command.Env = sanitizedEnvironment(probe, os.Environ())
	var output limitedBuffer
	output.limit = limit
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("%w: %w", ErrProbeFailed, ctxErr)
		}
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrExecutableNotFound, probe)
		}
		return "", fmt.Errorf("%w: command execution", ErrProbeFailed)
	}
	if output.overflow {
		return "", fmt.Errorf("%w: %w", ErrProbeFailed, ErrOutputLimit)
	}
	return output.String(), nil
}

func neutralWorkingDirectory(executable string) string {
	volume := filepath.VolumeName(executable)
	if volume != "" {
		return volume + string(os.PathSeparator)
	}
	return string(os.PathSeparator)
}

func knownProbe(name string, args []string) (string, bool) {
	base := strings.ToLower(filepath.Base(name))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	switch {
	case base == "node" && len(args) == 1 && args[0] == "--version":
		return "node", true
	case base == "go" && len(args) == 1 && args[0] == "version":
		return "go", true
	default:
		return "", false
	}
}

func sanitizedEnvironment(probe string, inherited []string) []string {
	blocked := map[string]struct{}{}
	forced := []string(nil)
	switch probe {
	case "node":
		blocked["NODE_OPTIONS"] = struct{}{}
	case "go":
		for _, key := range []string{"GOENV", "GOFLAGS", "GONOSUMDB", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN", "GOWORK"} {
			blocked[key] = struct{}{}
		}
		forced = []string{
			"GOENV=off",
			"GOFLAGS=",
			"GOPROXY=off",
			"GOSUMDB=off",
			"GOTOOLCHAIN=local",
			"GOWORK=off",
		}
	}
	result := make([]string, 0, len(inherited)+len(forced))
	for _, entry := range inherited {
		separator := strings.IndexByte(entry, '=')
		if separator > 0 {
			if _, remove := blocked[strings.ToUpper(entry[:separator])]; remove {
				continue
			}
		}
		result = append(result, entry)
	}
	return append(result, forced...)
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - int64(b.buffer.Len())
	if remaining <= 0 {
		b.overflow = true
		return original, nil
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	_, _ = b.buffer.Write(p)
	return original, nil
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
}

type constraintKind int

const (
	constraintExact constraintKind = iota
	constraintMinimum
	constraintMajor
	constraintCaret
	constraintTilde
	constraintMajorRange
	constraintMinor
)

type versionConstraint struct {
	kind    constraintKind
	version version
	upper   version
	// lowerExclusive is used only by explicit two-comparator ranges.
	lowerExclusive bool
	display        string
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
		return comparison >= 0 && actual.compare(c.upper) < 0
	case constraintTilde:
		return comparison >= 0 && actual.compare(c.upper) < 0
	case constraintMajorRange:
		if c.lowerExclusive && comparison <= 0 {
			return false
		}
		if !c.lowerExclusive && comparison < 0 {
			return false
		}
		return actual.compare(c.upper) < 0
	case constraintMinor:
		return actual.major == c.version.major && actual.minor == c.version.minor
	default:
		return false
	}
}

var majorRangePattern = regexp.MustCompile(`^(>=|>)\s*v?([0-9]+)(?:\.0(?:\.0)?)?\s+<\s*v?([0-9]+)(?:\.0(?:\.0)?)?$`)

func parseConstraint(raw string) (versionConstraint, bool) {
	value := strings.TrimSpace(raw)
	display := safeVersionText(value)
	if match := majorRangePattern.FindStringSubmatch(value); len(match) == 4 {
		lowerVersion, lowerComponents, lowerOK := parseVersion(match[2])
		upperVersion, upperComponents, upperOK := parseVersion(match[3])
		lower, upper := lowerVersion.major, upperVersion.major
		if lowerOK && upperOK && lowerComponents == 1 && upperComponents == 1 && lower < math.MaxInt && upper == lower+1 {
			return versionConstraint{
				kind: constraintMajorRange, version: version{major: lower}, upper: version{major: upper},
				lowerExclusive: match[1] == ">", display: display,
			}, true
		}
	}
	if strings.HasSuffix(value, ".x") || strings.HasSuffix(value, ".*") {
		majorText := strings.TrimSuffix(strings.TrimSuffix(value, ".x"), ".*")
		parsed, components, ok := parseVersion(majorText)
		if ok && components == 1 {
			return versionConstraint{kind: constraintMajor, version: parsed, display: display}, true
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
	constraint := versionConstraint{kind: kind, version: parsed, display: display}
	switch kind {
	case constraintExact:
		switch components {
		case 1:
			constraint.kind = constraintMajor
		case 2:
			constraint.kind = constraintMinor
		}
	case constraintCaret:
		upper, ok := caretUpperBound(parsed, components)
		if !ok {
			return versionConstraint{}, false
		}
		constraint.upper = upper
	case constraintTilde:
		upper, ok := tildeUpperBound(parsed, components)
		if !ok {
			return versionConstraint{}, false
		}
		constraint.upper = upper
	}
	return constraint, true
}

func caretUpperBound(value version, components int) (version, bool) {
	switch {
	case value.major > 0:
		next, ok := increment(value.major)
		return version{major: next}, ok
	case components == 1:
		return version{major: 1}, true
	case value.minor > 0 || components == 2:
		next, ok := increment(value.minor)
		return version{minor: next}, ok
	default:
		next, ok := increment(value.patch)
		return version{patch: next}, ok
	}
}

func tildeUpperBound(value version, components int) (version, bool) {
	if components == 1 {
		next, ok := increment(value.major)
		return version{major: next}, ok
	}
	next, ok := increment(value.minor)
	return version{major: value.major, minor: next}, ok
}

func increment(value int) (int, bool) {
	if value == math.MaxInt {
		return 0, false
	}
	return value + 1, true
}

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(raw string) (version, int, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if strings.ContainsAny(value, "+-") {
		return version{}, 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return version{}, 0, false
	}
	numbers := [3]int{}
	for index, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
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
	left := [...]int{v.major, v.minor, v.patch}
	right := [...]int{other.major, other.minor, other.patch}
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
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
	fields := strings.Fields(raw)
	if len(fields) != 4 || fields[0] != "go" || fields[1] != "version" || !strings.HasPrefix(fields[2], "go1.") || !strings.Contains(fields[3], "/") {
		return version{}, false
	}
	parsed, _, ok := parseVersion(strings.TrimPrefix(fields[2], "go"))
	return parsed, ok
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
