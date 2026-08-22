// Package agentcheck performs bounded, read-only native-agent compatibility
// probes for verified resume.
package agentcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/executabletrust"
	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

const (
	defaultTimeout   = 2 * time.Second
	maxVersionOutput = 4 << 10
	// retryTimeoutFactor widens the budget for a retried version probe.
	//
	// The first attempt stays deliberately short because the answer normally
	// arrives in milliseconds and a hung agent should not stall a command. But
	// agent CLIs are language runtimes, and on a saturated machine even a
	// trivial one can miss a two-second budget twice in a row — measured, not
	// assumed. Retrying with the same short budget would just reproduce the
	// first failure, so the retry is patient instead. Worst case for a
	// genuinely hung agent is one short wait plus one long one, still bounded,
	// and an outer deadline (preflight passes its remaining budget) still caps
	// both.
	retryTimeoutFactor = 4
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
	// ExecutablePath binds the private executable selected by LookPath to the
	// later native launch. It is deliberately absent from public JSON.
	ExecutablePath string `json:"-"`
	// ExecutableIdentity binds the private filesystem object and its metadata
	// to the final native launch boundary. It is deliberately absent from JSON.
	ExecutableIdentity fileidentity.Identity `json:"-"`
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
	Home string
	Root string
	// Workspace is the executable trust boundary used by production lookup.
	// Repository-owned PATH entries and executable candidates are excluded.
	Workspace  string
	LookPath   func(string) (string, error)
	Getenv     func(string) string
	Runner     VersionRunner
	Timeout    time.Duration
	MaxOutput  int64
	Executable string
	// CaptureIdentity overrides private executable identity capture in tests.
	CaptureIdentity func(context.Context, string) (fileidentity.Identity, error)
}

// Inspect verifies executable, local session layout, and current version.
func Inspect(ctx context.Context, agentName string, opts Options) Result {
	agentName = strings.ToLower(strings.TrimSpace(agentName))
	result := Result{Agent: agentName, Status: StatusUntested}
	definition, ok := lookupDefinition(agentName)
	if !ok {
		result.Message = "agent does not support native verified resume"
		return result
	}

	executable := opts.Executable
	if executable == "" {
		executable = definition.executable
	}
	resolved := ""
	trustedSearchPath := ""
	var err error
	if opts.LookPath != nil {
		resolved, err = opts.LookPath(executable)
	} else {
		trustWorkspace := opts.Workspace
		if trustWorkspace == "" {
			trustWorkspace, err = os.Getwd()
		}
		lookupName := executable
		environment := os.Environ()
		if filepath.IsAbs(executable) {
			lookupName = filepath.Base(executable)
			environment = replaceEnvironmentValue(environment, "PATH", filepath.Dir(executable))
		}
		resolution, resolveErr := executabletrust.Resolve(lookupName, trustWorkspace, environment)
		if resolveErr == nil {
			resolved = resolution.Executable
			trustedSearchPath = resolution.SearchPath
		}
		if err == nil {
			err = resolveErr
		}
	}
	executableMissing := err != nil || strings.TrimSpace(resolved) == ""
	if executableMissing {
		result.Status = StatusNotInstalled
		result.Message = "native agent executable is unavailable"
	} else {
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			result.Status = StatusError
			result.Message = "native agent executable path is unavailable"
			return result
		}
		result.ExecutablePresent = true
		result.ExecutablePath = resolved
	}

	root := opts.Root
	if root == "" {
		getenv := opts.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		root = definition.rootFromEnvironment(getenv)
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
		if !executableMissing {
			result.Message = "native agent session layout is unrecognized"
		}
		return result
	}
	result.LayoutRecognized = true
	if executableMissing {
		result.Status = StatusSupported
		result.Message = "native agent version is not determinable; the session layout is still readable"
		return result
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{MaxOutput: opts.MaxOutput, SearchPath: trustedSearchPath}
	}
	captureIdentity := opts.CaptureIdentity
	if captureIdentity == nil {
		captureIdentity = fileidentity.CaptureExecutable
	}

	attempt := versionProbe{
		executable:      resolved,
		runner:          runner,
		captureIdentity: captureIdentity,
	}
	probe := attempt.run(ctx, timeout)
	if probe.timedOut && ctx.Err() == nil {
		// A probe that only ran out of time has measured nothing at all, and a
		// caller gating on version must not be handed "no version" — which is
		// indistinguishable from an uninstalled agent — because the machine was
		// briefly busy. Measure once more, with room to actually finish.
		//
		// Only a timeout is retried. A non-zero exit is deterministic and would
		// simply repeat, and an executable that changed underneath the probe
		// must be reported rather than re-rolled until it looks stable.
		probe = attempt.run(ctx, timeout*retryTimeoutFactor)
	}
	if probe.err != nil {
		result.Status = StatusError
		result.Message = probe.message
		return result
	}
	result.ExecutableIdentity = probe.identity
	version, ok := definition.parseVersion(probe.output)
	if !ok {
		result.Message = "native agent version is unrecognized"
		return result
	}
	result.Version = version
	if !definition.supported(version) {
		// Naming the range is the difference between a refusal a user can act
		// on and one they cannot: without it there is nothing to tell them
		// which version to install.
		result.Message = fmt.Sprintf(
			"native agent version %s is outside the verified range %s",
			version, definition.versionRange,
		)
		return result
	}
	result.Status = StatusSupported
	result.Message = "native agent executable, version, and session layout are supported"
	return result
}

// renderVersionRange renders the inclusive verified range for a message. An
// open end is stated as such rather than omitted, so a bound is never silently
// absent.
func renderVersionRange(min, max string) string {
	minimum := strings.TrimSpace(min)
	maximum := strings.TrimSpace(max)
	switch {
	case minimum == "" && maximum == "":
		return "(no verified range is declared)"
	case minimum == "":
		return "up to and including " + maximum
	case maximum == "":
		return minimum + " and newer"
	default:
		return minimum + " to " + maximum + " inclusive"
	}
}

// versionProbe is one bounded `--version` measurement: an executable-identity
// capture on each side of the call, so an executable swapped underneath the
// probe is detected rather than trusted. It carries its own deadline so it can
// be repeated with a fresh budget.
type versionProbe struct {
	executable      string
	runner          VersionRunner
	captureIdentity func(context.Context, string) (fileidentity.Identity, error)
}

type versionProbeResult struct {
	output   VersionOutput
	identity fileidentity.Identity
	timedOut bool
	err      error
	message  string
}

func (probe versionProbe) run(ctx context.Context, timeout time.Duration) versionProbeResult {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fail := func(err error, message string) versionProbeResult {
		return versionProbeResult{
			// The deadline belongs to this probe, so probeCtx — not the
			// caller's ctx — is what says the measurement ran out of time.
			timedOut: errors.Is(probeCtx.Err(), context.DeadlineExceeded),
			err:      err,
			message:  message,
		}
	}

	beforeIdentity, err := probe.captureIdentity(probeCtx, probe.executable)
	if err != nil || !beforeIdentity.IsLaunchable() {
		if err == nil {
			err = errNotLaunchable
		}
		return fail(err, "native agent executable identity is unavailable")
	}
	output, err := probe.runner.Version(probeCtx, probe.executable, "--version")
	if err != nil {
		return fail(err, "native agent version probe failed")
	}
	afterIdentity, err := probe.captureIdentity(probeCtx, probe.executable)
	if err != nil || !afterIdentity.IsLaunchable() ||
		!fileidentity.SameExecutable(beforeIdentity, afterIdentity) {
		if err == nil {
			err = errExecutableChanged
		}
		return fail(err, "native agent executable changed during version verification")
	}
	return versionProbeResult{output: output, identity: afterIdentity}
}

var (
	errNotLaunchable     = errors.New("agentcheck: native agent executable is not launchable")
	errExecutableChanged = errors.New("agentcheck: native agent executable changed during version verification")
)

// VersionEvidence classifies how much is known about an installed agent's
// version. The distinction matters to any caller that gates on version: two of
// these three answers carry no version, but they do not mean the same thing.
type VersionEvidence string

const (
	// VersionUnavailable means there is no version information to read: the
	// agent is not installed, Reinstate has no definition for it, its session
	// layout is unrecognized, or it answered `--version` with something that is
	// not a version. This is absence of evidence, not evidence of
	// incompatibility, and a read-only caller must not block on it.
	VersionUnavailable VersionEvidence = "unavailable"
	// VersionDetermined means the installed executable reported a version.
	VersionDetermined VersionEvidence = "determined"
	// VersionProbeFailed means the agent is installed and a version exists to
	// read, but the bounded probe could not read it — it timed out even after a
	// retry, failed to execute, or the executable changed underneath it. This
	// is a failed measurement rather than an absent one: repeating it could
	// well return a version outside the verified range. A caller gating on
	// version must treat it as uncertain, never as a clean bill of health.
	VersionProbeFailed VersionEvidence = "probe_failed"
)

// InstalledVersion returns the agent's self-reported version using exactly the
// mechanism Inspect uses: trusted executable resolution, an identity check
// around a bounded `--version` probe, and the vendor-specific parser.
//
// The returned evidence separates "there is nothing to read" from "reading
// failed". Collapsing those two into one "unknown" answer is what lets an
// installed, out-of-range agent pass as unrecognized when its version probe
// merely times out, so callers are given the distinction and must handle it.
func InstalledVersion(ctx context.Context, agentName string, opts Options) (string, VersionEvidence) {
	result := Inspect(ctx, agentName, opts)
	if version := strings.TrimSpace(result.Version); version != "" {
		return version, VersionDetermined
	}
	if result.Status == StatusError {
		return "", VersionProbeFailed
	}
	return "", VersionUnavailable
}

// SupportedVersion reports whether version is inside the agent's verified range
// as published in docs/compatibility.md. Unknown agents are never supported.
func SupportedVersion(agentName, version string) bool {
	definition, ok := lookupDefinition(strings.ToLower(strings.TrimSpace(agentName)))
	if !ok {
		return false
	}
	return definition.supported(strings.TrimSpace(version))
}

// Definition is the catalog-derived version/layout probe for one agent.
type Definition struct {
	Executable string
	Layout     string
	Marker     string
	// RootEnvironment names the environment variable that overrides the
	// agent's storage root.
	RootEnvironment string
	// RootEnvironmentSuffix is appended to RootEnvironment's value to reach the
	// root, for a vendor whose variable names a parent directory rather than
	// the root itself. OpenCode reads $XDG_DATA_HOME/opencode, so the variable
	// alone is one directory short and the marker would be looked for beside
	// the store instead of inside it.
	RootEnvironmentSuffix string
	Roots                 func(string) []string
	Parse                 func(VersionOutput) (string, bool)
	Min, Max              string
}

type definition struct {
	executable            string
	layout                string
	marker                string
	rootEnvironment       string
	rootEnvironmentSuffix string
	roots                 func(string) []string
	parseVersion          func(VersionOutput) (string, bool)
	supported             func(string) bool
	// versionRange is the inclusive verified range, rendered for a refusal
	// message. Without it a refusal could not tell the user which version to
	// install.
	versionRange string
}

// definitions is installed by CLI init from the catalog. Nil means tests
// that do not import the CLI should use testFallbackDefinitions.
var definitions map[string]definition

// SetDefinitions replaces the probe table. Production CLI installs catalog
// descriptors that have Native and Version specs.
func SetDefinitions(defs map[string]Definition) {
	converted := make(map[string]definition, len(defs))
	for name, spec := range defs {
		converted[strings.ToLower(strings.TrimSpace(name))] = definitionFrom(spec)
	}
	definitions = converted
}

// rootFromEnvironment resolves the storage root from the agent's declared root
// variable, appending the declared suffix when the variable names the parent
// rather than the root itself. An unset variable yields "" so the caller falls
// back to the per-OS root candidates.
func (d definition) rootFromEnvironment(getenv func(string) string) string {
	if strings.TrimSpace(d.rootEnvironment) == "" {
		return ""
	}
	value := strings.TrimSpace(getenv(d.rootEnvironment))
	if value == "" {
		return ""
	}
	suffix := strings.TrimSpace(d.rootEnvironmentSuffix)
	if suffix == "" {
		return value
	}
	return filepath.Join(value, filepath.FromSlash(suffix))
}

func lookupDefinition(name string) (definition, bool) {
	table := definitions
	if table == nil {
		table = testFallbackDefinitions()
	}
	d, ok := table[name]
	return d, ok
}

func definitionFrom(spec Definition) definition {
	min, max := spec.Min, spec.Max
	parse := spec.Parse
	return definition{
		executable:            spec.Executable,
		layout:                spec.Layout,
		marker:                spec.Marker,
		rootEnvironment:       spec.RootEnvironment,
		rootEnvironmentSuffix: spec.RootEnvironmentSuffix,
		roots:                 spec.Roots,
		parseVersion:          parse,
		supported: func(version string) bool {
			return adapter.StableVersionInRange(version, min, max)
		},
		versionRange: renderVersionRange(min, max),
	}
}

func testFallbackDefinitions() map[string]definition {
	return map[string]definition{
		"claude": definitionFrom(Definition{
			Executable:      "claude",
			Layout:          "projects-jsonl",
			Marker:          "projects",
			RootEnvironment: "CLAUDE_CONFIG_DIR",
			Roots: func(home string) []string {
				return []string{filepath.Join(home, ".claude"), filepath.Join(home, ".config", "claude")}
			},
			Parse: parseClaudeVersion,
			Min:   "2.1.219",
			Max:   "2.1.238",
		}),
		"codex": definitionFrom(Definition{
			Executable:      "codex",
			Layout:          "sessions-rollout-jsonl",
			Marker:          "sessions",
			RootEnvironment: "CODEX_HOME",
			Roots: func(home string) []string {
				return []string{filepath.Join(home, ".codex"), filepath.Join(home, ".config", "codex")}
			},
			Parse: parseCodexVersion,
			Min:   "0.133.0",
			Max:   "0.149.0",
		}),
	}
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

// markerKindStable reports whether two stats describe the same kind of marker,
// where "kind" is only ever a real directory or a real regular file.
//
// An agent whose sessions live in a tree declares a directory marker; an agent
// whose sessions live in one embedded database declares that file. Requiring a
// directory would report every embedded-store agent's layout as unrecognized
// even when the store is sitting right there, which is a refusal to resume a
// session the index can already read. Anything that is neither — a symlink, a
// device, a socket — is still rejected, so the widening is exactly one kind.
func markerKindStable(before, after os.FileInfo) bool {
	switch {
	case before.IsDir():
		return after.IsDir()
	case before.Mode().IsRegular():
		return after.Mode().IsRegular()
	default:
		return false
	}
}

// recognizedLayout opens the marker relative to an os.Root and compares the
// object before, during, and after opening. The root must be a real directory
// and the marker a real directory or a real regular file, never a symlink.
// This fails closed if either path is replaced while it is being inspected.
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
	defer func() { _ = rootHandle.Close() }()

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
	if err != nil || markerBefore.Mode()&os.ModeSymlink != 0 {
		return false
	}
	if !markerBefore.IsDir() && !markerBefore.Mode().IsRegular() {
		return false
	}
	openedMarker, err := rootHandle.Open(marker)
	if err != nil {
		return false
	}
	openedMarkerInfo, statErr := openedMarker.Stat()
	closeErr = openedMarker.Close()
	if statErr != nil || closeErr != nil ||
		!markerKindStable(markerBefore, openedMarkerInfo) ||
		!os.SameFile(markerBefore, openedMarkerInfo) {
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
	// SearchPath is the already-validated executable search path. It prevents
	// script launchers from delegating to workspace-owned runtimes.
	SearchPath      string
	testEnvironment []string
}

// Version runs the exact executable and fixed arguments without stdin.
func (runner ExecRunner) Version(ctx context.Context, executable string, args ...string) (VersionOutput, error) {
	limit := runner.MaxOutput
	if limit <= 0 || limit > maxVersionOutput {
		limit = maxVersionOutput
	}
	command := exec.CommandContext(ctx, executable, args...)
	configureVersionCommand(command)
	command.Stdin = nil
	command.Dir = neutralWorkingDirectory(executable)
	environment := sanitizedVersionEnvironment(os.Environ())
	if runner.SearchPath != "" {
		environment = replaceEnvironmentValue(environment, "PATH", runner.SearchPath)
	}
	command.Env = append(environment, runner.testEnvironment...)
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

func neutralWorkingDirectory(executable string) string {
	volume := filepath.VolumeName(executable)
	if volume != "" {
		return volume + string(os.PathSeparator)
	}
	return string(os.PathSeparator)
}

// sanitizedVersionEnvironment retains only the operating-system values needed
// to start a resolved executable (including script launchers) and locale/temp
// behavior. In particular, it excludes NODE_OPTIONS and vendor/project config
// selectors that could execute project-controlled code during a safe probe.
func sanitizedVersionEnvironment(inherited []string) []string {
	allowed := map[string]struct{}{
		"COMSPEC": {}, "LANG": {}, "LC_ALL": {}, "PATH": {}, "PATHEXT": {},
		"SYSTEMROOT": {}, "TEMP": {}, "TMP": {}, "TMPDIR": {}, "WINDIR": {},
	}
	result := make([]string, 0, len(allowed))
	for _, entry := range inherited {
		separator := strings.IndexByte(entry, '=')
		if separator <= 0 {
			continue
		}
		if _, ok := allowed[strings.ToUpper(entry[:separator])]; ok {
			result = append(result, entry)
		}
	}
	return result
}

func replaceEnvironmentValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		currentKey, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(currentKey, key) {
			continue
		}
		result = append(result, entry)
	}
	return append(result, key+"="+value)
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
