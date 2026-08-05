package main

import (
	"bytes"
	"context"
	"crypto/sha1" // #nosec G505 -- Git SHA-1 object identity is the frozen fixture contract.
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	commandTimeout               = 30 * time.Second
	warmupCount                  = 1
	warmSampleCount              = 20
	coldSampleCount              = 3
	maxStdoutBytes               = 32 << 20
	maxStderrBytes               = 1 << 20
	maxSmallCommandStdoutBytes   = 4 << 20
	maxSmallCommandStderrBytes   = 64 << 10
	maxTaggedHarnessFileBytes    = 1 << 20
	maxImmutableFixtureFileBytes = 4 << 20
	maxInstalledBinaryBytes      = 256 << 20
)

var coldIndexFamily = []string{
	"session-index-v2.sqlite",
	"session-index-v2.sqlite.lock",
	"session-index-v2.sqlite.write.lock",
	"session-index-v2.sqlite-journal",
	"session-index-v2.sqlite-wal",
	"session-index-v2.sqlite-shm",
}

type runConfig struct {
	Root            string
	Rein            string
	Reinstate       string
	SourceRoot      string
	ExpectedCommit  string
	ExpectedVersion string
	CuratedPath     string
}

type commandDefinition struct {
	Label string
	Args  []string
}

type durationMetrics struct {
	Samples   int   `json:"samples"`
	MedianNS  int64 `json:"median_ns"`
	P95NS     int64 `json:"p95_ns"`
	MaximumNS int64 `json:"maximum_ns"`
	Timeouts  int   `json:"timeouts"`
	Warmups   int   `json:"warmups"`
	Validated bool  `json:"validated"`
}

type commandMetrics struct {
	Label   string          `json:"label"`
	Metrics durationMetrics `json:"metrics"`
}

type startupCommandMetrics struct {
	Label string          `json:"label"`
	Cold  durationMetrics `json:"cold"`
	Warm  durationMetrics `json:"warm"`
}

type corpusResult struct {
	Corpus                    string           `json:"corpus"`
	CanonicalDigest           string           `json:"canonical_digest"`
	MaterializedDigest        string           `json:"materialized_digest"`
	RecordCount               int              `json:"record_count"`
	CapabilityNameCount       int              `json:"capability_name_count"`
	Limit                     int              `json:"limit"`
	AliasParity               bool             `json:"alias_parity"`
	OpenCodeOmitted           bool             `json:"opencode_omitted"`
	AmbientCapabilitiesAbsent bool             `json:"ambient_capabilities_absent"`
	WorkspaceGitHead          string           `json:"workspace_git_head"`
	WorkspaceGitClean         bool             `json:"workspace_git_clean"`
	SourceFingerprintBefore   string           `json:"source_fingerprint_before"`
	SourceFingerprintAfter    string           `json:"source_fingerprint_after"`
	ColdSessions              durationMetrics  `json:"cold_sessions"`
	WarmCommands              []commandMetrics `json:"warm_commands"`
}

type harnessResult struct {
	SchemaVersion             int                     `json:"schema_version"`
	Generator                 string                  `json:"generator"`
	ExpectedVersion           string                  `json:"expected_version"`
	ExpectedCommit            string                  `json:"expected_commit"`
	InstalledBinarySHA256     string                  `json:"installed_binary_sha256"`
	AliasBinarySHA256         string                  `json:"alias_binary_sha256"`
	FixtureCanonicalDigest    string                  `json:"fixture_canonical_digest"`
	EnvironmentDigest         string                  `json:"environment_digest"`
	Clock                     string                  `json:"clock"`
	Capture                   string                  `json:"capture"`
	TimeoutMilliseconds       int64                   `json:"timeout_milliseconds"`
	WarmupCountPerCommand     int                     `json:"warmup_count_per_command"`
	WarmSampleCountPerCommand int                     `json:"warm_sample_count_per_command"`
	ColdSampleCount           int                     `json:"cold_sample_count"`
	P95Method                 string                  `json:"p95_method"`
	Startup                   []startupCommandMetrics `json:"startup"`
	Normal                    corpusResult            `json:"normal"`
	Large                     corpusResult            `json:"large"`
}

type commandCapture struct {
	Stdout   []byte
	Stderr   []byte
	Elapsed  time.Duration
	ExitCode int
	TimedOut bool
}

type strictWarning struct {
	Agent     string `json:"agent,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type strictSessionSummary struct {
	Key            string                    `json:"key"`
	ID             string                    `json:"id"`
	Agent          string                    `json:"agent"`
	Title          string                    `json:"title,omitempty"`
	Project        string                    `json:"project,omitempty"`
	Workspace      string                    `json:"workspace,omitempty"`
	Branch         string                    `json:"branch,omitempty"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	SizeBytes      int64                     `json:"size_bytes"`
	MessageCount   int                       `json:"message_count"`
	Files          []string                  `json:"files,omitempty"`
	Capabilities   sessionindex.Capabilities `json:"capabilities"`
	ReadOnlyReason string                    `json:"read_only_reason,omitempty"`
}

type strictSessionsOutput struct {
	Sessions []strictSessionSummary `json:"sessions"`
	Warnings []strictWarning        `json:"warnings,omitempty"`
}

type strictInspectOutput struct {
	Session     sessionindex.Record `json:"session"`
	Environment preflight.Report    `json:"environment"`
	Warnings    []strictWarning     `json:"warnings,omitempty"`
}

type strictLaunchOutput struct {
	sessionindex.LaunchPlan
	Environment preflight.Report `json:"environment"`
}

type strictVersionOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type cappedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *cappedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining < len(value) {
		buffer.exceeded = true
		value = value[:max(remaining, 0)]
	}
	if len(value) > 0 {
		_, _ = buffer.buffer.Write(value)
	}
	return original, nil
}

func runHarness(config runConfig, spec fixtureSpec) (harnessResult, error) {
	if err := validateRunConfig(config); err != nil {
		return harnessResult{}, err
	}
	if err := validateCuratedExecutionPaths(config); err != nil {
		return harnessResult{}, err
	}
	if err := os.Setenv("PATH", config.CuratedPath); err != nil {
		return harnessResult{}, err
	}
	if _, err := exec.LookPath("opencode"); err == nil {
		return harnessResult{}, errors.New("curated PATH resolves OpenCode; deterministic performance corpus requires omission")
	} else if !errors.Is(err, exec.ErrNotFound) {
		return harnessResult{}, errors.New("curated PATH cannot prove OpenCode omission")
	}
	if err := verifyTaggedHarnessSource(config); err != nil {
		return harnessResult{}, err
	}
	manifest, err := generateFixture(config.Root, spec)
	if err != nil {
		return harnessResult{}, err
	}
	if err := verifyGeneratedFixture(config.Root, spec, manifest); err != nil {
		return harnessResult{}, err
	}
	for _, corpus := range []corpusManifest{manifest.Normal, manifest.Large} {
		if _, err := initializeWorkspaceRepository(config, spec, corpus); err != nil {
			return harnessResult{}, fmt.Errorf("initialize %s workspace: %w", corpus.Corpus, err)
		}
	}

	normalEnv, err := corpusEnvironment(config.Root, manifest.Normal, config.CuratedPath)
	if err != nil {
		return harnessResult{}, err
	}
	reinHash, err := fileSHA256(config.Rein)
	if err != nil {
		return harnessResult{}, err
	}
	reinstateHash, err := fileSHA256(config.Reinstate)
	if err != nil {
		return harnessResult{}, err
	}
	if reinHash != reinstateHash {
		return harnessResult{}, errors.New("installed rein and reinstate bytes differ")
	}
	startup, err := measureStartupCommands(config)
	if err != nil {
		return harnessResult{}, err
	}
	if err := requireInstalledAliasHashes(config, reinHash); err != nil {
		return harnessResult{}, err
	}

	normal, err := measureCorpus(config, spec, manifest.Normal)
	if err != nil {
		return harnessResult{}, fmt.Errorf("measure normal corpus: %w", err)
	}
	if err := requireInstalledAliasHashes(config, reinHash); err != nil {
		return harnessResult{}, err
	}
	large, err := measureCorpus(config, spec, manifest.Large)
	if err != nil {
		return harnessResult{}, fmt.Errorf("measure large corpus: %w", err)
	}
	if err := requireInstalledAliasHashes(config, reinHash); err != nil {
		return harnessResult{}, err
	}
	return harnessResult{
		SchemaVersion:             1,
		Generator:                 spec.Generator,
		ExpectedVersion:           config.ExpectedVersion,
		ExpectedCommit:            strings.ToLower(config.ExpectedCommit),
		InstalledBinarySHA256:     reinHash,
		AliasBinarySHA256:         reinstateHash,
		FixtureCanonicalDigest:    manifest.CanonicalDigest,
		EnvironmentDigest:         digestEnvironment(normalEnv),
		Clock:                     "time.Now/time.Since monotonic process-start-to-exit",
		Capture:                   "bounded memory; raw stdout/stderr discarded after validation",
		TimeoutMilliseconds:       commandTimeout.Milliseconds(),
		WarmupCountPerCommand:     warmupCount,
		WarmSampleCountPerCommand: warmSampleCount,
		ColdSampleCount:           coldSampleCount,
		P95Method:                 "nearest-rank ceil(0.95*n), one-indexed",
		Startup:                   startup,
		Normal:                    normal,
		Large:                     large,
	}, nil
}

func validateRunConfig(config runConfig) error {
	for label, value := range map[string]string{
		"root":        config.Root,
		"rein":        config.Rein,
		"reinstate":   config.Reinstate,
		"source root": config.SourceRoot,
	} {
		if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be an absolute path", label)
		}
	}
	if len(config.ExpectedCommit) != 40 {
		return errors.New("expected commit must be a literal 40-character SHA-1")
	}
	if _, err := hex.DecodeString(config.ExpectedCommit); err != nil {
		return errors.New("expected commit must be hexadecimal")
	}
	if strings.TrimSpace(config.ExpectedVersion) == "" {
		return errors.New("expected version is required")
	}
	if strings.TrimSpace(config.CuratedPath) == "" {
		return errors.New("a curated PATH is required")
	}
	physicalSource, err := filepath.EvalSymlinks(config.SourceRoot)
	if err != nil {
		return errors.New("source root cannot be resolved physically")
	}
	physicalSource, err = filepath.Abs(physicalSource)
	if err != nil || !sameCanonicalPath(config.SourceRoot, physicalSource) {
		return errors.New("source root must be its canonical physical path")
	}
	if info, err := os.Stat(physicalSource); err != nil || !info.IsDir() {
		return errors.New("source root is not a directory")
	}
	if _, err := os.Lstat(config.Root); !errors.Is(err, os.ErrNotExist) {
		return errors.New("performance evidence root must not exist")
	}
	rootParent := filepath.Dir(config.Root)
	physicalParent, err := filepath.EvalSymlinks(rootParent)
	if err != nil {
		return errors.New("performance evidence parent cannot be resolved physically")
	}
	physicalRoot := filepath.Join(physicalParent, filepath.Base(config.Root))
	if !sameCanonicalPath(config.Root, physicalRoot) {
		return errors.New("performance evidence root must use a canonical physical parent")
	}
	if pathWithin(config.SourceRoot, config.Root) || pathWithin(config.Root, config.SourceRoot) {
		return errors.New("source and performance evidence roots must be separate")
	}
	for _, path := range []string{config.Rein, config.Reinstate} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("installed alias is not a regular file")
		}
	}
	return nil
}

func validateCuratedExecutionPaths(config runConfig) error {
	entries := filepath.SplitList(config.CuratedPath)
	if len(entries) == 0 {
		return errors.New("curated PATH has no entries")
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry == "" || !filepath.IsAbs(entry) || filepath.Clean(entry) != entry {
			return errors.New("every curated PATH entry must be non-empty, absolute, and clean")
		}
		physical, err := filepath.EvalSymlinks(entry)
		if err != nil {
			return errors.New("curated PATH entry cannot be resolved physically")
		}
		physical, err = filepath.Abs(physical)
		if err != nil || !sameCanonicalPath(entry, physical) {
			return errors.New("curated PATH entry must use its canonical physical path")
		}
		info, err := os.Stat(physical)
		if err != nil || !info.IsDir() {
			return errors.New("curated PATH entry is not a directory")
		}
		if pathWithin(config.SourceRoot, physical) || pathWithin(config.Root, physical) {
			return errors.New("curated PATH entry is inside source or performance evidence")
		}
		key := physical
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, duplicate := seen[key]; duplicate {
			return errors.New("curated PATH contains a duplicate physical directory")
		}
		seen[key] = struct{}{}
	}
	prior := os.Getenv("PATH")
	if err := os.Setenv("PATH", config.CuratedPath); err != nil {
		return err
	}
	defer func() { _ = os.Setenv("PATH", prior) }()
	for _, name := range []string{"git", "claude", "codex"} {
		resolved, err := exec.LookPath(name)
		if err != nil {
			return fmt.Errorf("curated PATH does not resolve required %s executable", name)
		}
		if err := validateTrustedExecutable(resolved, config); err != nil {
			return fmt.Errorf("untrusted %s executable: %w", name, err)
		}
	}
	for _, path := range []string{config.Rein, config.Reinstate} {
		if err := validateTrustedExecutable(path, config); err != nil {
			return fmt.Errorf("untrusted installed alias: %w", err)
		}
	}
	if _, err := exec.LookPath("opencode"); err == nil {
		return errors.New("curated PATH resolves OpenCode; deterministic omission is impossible")
	} else if !errors.Is(err, exec.ErrNotFound) {
		return errors.New("curated PATH cannot prove OpenCode omission")
	}
	return nil
}

func validateTrustedExecutable(path string, config runConfig) error {
	if !filepath.IsAbs(path) {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return errors.New("executable path is not absolute")
		}
		path = absolute
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return errors.New("executable cannot be resolved physically")
	}
	physical, err = filepath.Abs(physical)
	if err != nil {
		return errors.New("executable physical path is unavailable")
	}
	info, err := os.Stat(physical)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("executable is not a regular file")
	}
	if pathWithin(config.SourceRoot, physical) || pathWithin(config.Root, physical) {
		return errors.New("executable is owned by source or performance evidence")
	}
	return nil
}

func sameCanonicalPath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func pathWithin(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func verifyTaggedHarnessSource(config runConfig) error {
	environment := minimalGitEnvironment(config.CuratedPath)
	head, err := runSmallCommand(config.SourceRoot, environment, "git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("verify tagged source HEAD: %w", err)
	}
	if strings.TrimSpace(string(head)) != strings.ToLower(config.ExpectedCommit) {
		return errors.New("performance harness source checkout does not match expected commit")
	}
	tree, err := runSmallCommand(config.SourceRoot, environment, "git", "ls-tree", "-r", "-z", "--full-tree", config.ExpectedCommit, "--", "scripts/testing/phase3perf")
	if err != nil {
		return errors.New("read tagged performance harness tree")
	}
	expected := make(map[string]string)
	for _, encoded := range bytes.Split(tree, []byte{0}) {
		if len(encoded) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(encoded, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 3 || fields[1] != "blob" || fields[0] != "100644" && fields[0] != "100755" {
			return errors.New("tagged performance harness tree contains an unsupported entry")
		}
		relative := filepath.ToSlash(string(name))
		if !strings.HasPrefix(relative, "scripts/testing/phase3perf/") || len(fields[2]) != 40 {
			return errors.New("tagged performance harness tree contains an invalid path or blob")
		}
		expected[relative] = fields[2]
	}
	if len(expected) == 0 {
		return errors.New("tagged performance harness tree is empty")
	}

	actual := make(map[string]string)
	harnessRoot := filepath.Join(config.SourceRoot, "scripts", "testing", "phase3perf")
	if err := filepath.WalkDir(harnessRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("performance harness worktree contains a non-regular file")
		}
		relative, err := filepath.Rel(config.SourceRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		blob, ok := expected[relative]
		if !ok {
			return errors.New("performance harness contains an untracked or unexpected source file")
		}
		actual[relative] = blob
		return nil
	}); err != nil {
		return err
	}
	if len(actual) != len(expected) {
		return errors.New("performance harness is missing a tagged source file")
	}
	for relative, blob := range actual {
		sizeOutput, err := runSmallCommand(config.SourceRoot, environment, "git", "cat-file", "-s", blob)
		if err != nil {
			return errors.New("read tagged performance harness blob size")
		}
		blobSize, err := strconv.ParseInt(strings.TrimSpace(string(sizeOutput)), 10, 64)
		if err != nil || blobSize < 0 || blobSize > maxTaggedHarnessFileBytes {
			return errors.New("tagged performance harness blob exceeds the fixed size limit")
		}
		worktreeBytes, err := readBoundedRegular(filepath.Join(config.SourceRoot, filepath.FromSlash(relative)), maxTaggedHarnessFileBytes)
		if err != nil {
			return errors.New("read performance harness worktree bytes")
		}
		if int64(len(worktreeBytes)) != blobSize {
			return errors.New("performance harness worktree size differs from tagged blob")
		}
		blobBytes, err := runSmallCommand(config.SourceRoot, environment, "git", "cat-file", "blob", blob)
		if err != nil || !bytes.Equal(worktreeBytes, blobBytes) {
			return errors.New("performance harness bytes differ from tagged blob bytes")
		}
	}
	return nil
}

func measureStartupCommands(config runConfig) ([]startupCommandMetrics, error) {
	definitions := []commandDefinition{
		{Label: "version", Args: []string{"version", "--json"}},
		{Label: "help", Args: []string{"--help"}},
	}
	results := make([]startupCommandMetrics, 0, len(definitions))
	for _, definition := range definitions {
		cold := make([]time.Duration, 0, coldSampleCount)
		for sample := 1; sample <= coldSampleCount; sample++ {
			directory, environment, err := startupEnvironment(config.Root, fmt.Sprintf("%s-cold-%d", definition.Label, sample), config.CuratedPath)
			if err != nil {
				return nil, err
			}
			capture, err := executeProcess(config.Rein, definition.Args, directory, environment)
			if err != nil {
				return nil, fmt.Errorf("cold startup %s sample %d: %w", definition.Label, sample, err)
			}
			if err := validateStartupResult(definition, capture.Stdout, config); err != nil {
				return nil, fmt.Errorf("cold startup %s sample %d validation: %w", definition.Label, sample, err)
			}
			cold = append(cold, capture.Elapsed)
		}

		parityDirectory, parityEnvironment, err := startupEnvironment(config.Root, definition.Label+"-alias-parity", config.CuratedPath)
		if err != nil {
			return nil, err
		}
		left, err := executeProcess(config.Rein, definition.Args, parityDirectory, parityEnvironment)
		if err != nil {
			return nil, err
		}
		right, err := executeProcess(config.Reinstate, definition.Args, parityDirectory, parityEnvironment)
		if err != nil {
			return nil, err
		}
		if err := validateStartupResult(definition, left.Stdout, config); err != nil {
			return nil, err
		}
		if err := validateStartupResult(definition, right.Stdout, config); err != nil {
			return nil, err
		}
		if !startupOutputsEqual(definition.Label, left.Stdout, right.Stdout) {
			return nil, fmt.Errorf("startup alias mismatch for %s", definition.Label)
		}

		warmDirectory, warmEnvironment, err := startupEnvironment(config.Root, definition.Label+"-warm", config.CuratedPath)
		if err != nil {
			return nil, err
		}
		for sample := 0; sample < warmupCount; sample++ {
			capture, err := executeProcess(config.Rein, definition.Args, warmDirectory, warmEnvironment)
			if err != nil {
				return nil, fmt.Errorf("startup %s warmup: %w", definition.Label, err)
			}
			if err := validateStartupResult(definition, capture.Stdout, config); err != nil {
				return nil, err
			}
		}
		warm := make([]time.Duration, 0, warmSampleCount)
		for sample := 1; sample <= warmSampleCount; sample++ {
			capture, err := executeProcess(config.Rein, definition.Args, warmDirectory, warmEnvironment)
			if err != nil {
				return nil, fmt.Errorf("startup %s sample %d: %w", definition.Label, sample, err)
			}
			if err := validateStartupResult(definition, capture.Stdout, config); err != nil {
				return nil, err
			}
			warm = append(warm, capture.Elapsed)
		}
		warmMetrics := calculateMetrics(warm)
		warmMetrics.Warmups = warmupCount
		results = append(results, startupCommandMetrics{
			Label: definition.Label,
			Cold:  calculateMetrics(cold),
			Warm:  warmMetrics,
		})
	}
	return results, nil
}

func validateStartupResult(definition commandDefinition, output []byte, config runConfig) error {
	switch definition.Label {
	case "version":
		var value strictVersionOutput
		if err := decodeOneStrictJSON(output, &value); err != nil {
			return errors.New("installed version output is not valid JSON")
		}
		if value.Name != "reinstate" || value.Date == "" || value.Version != config.ExpectedVersion || !strings.EqualFold(value.Commit, config.ExpectedCommit) {
			return errors.New("installed version or full commit does not match expected tagged identity")
		}
		return nil
	case "help":
		text := string(output)
		for _, required := range []string{
			"Usage:", "Available Commands:", "sessions", "search", "inspect", "resume", "fork", "version",
		} {
			if !strings.Contains(text, required) {
				return errors.New("help output omits a required command surface")
			}
		}
		if strings.Contains(text, config.Root) || len(output) > maxStderrBytes {
			return errors.New("help output is unbounded or contains the private evidence root")
		}
		return nil
	default:
		return errors.New("unknown startup command")
	}
}

func startupOutputsEqual(label string, left, right []byte) bool {
	if label == "version" {
		leftJSON, leftErr := normalizeJSON(left)
		rightJSON, rightErr := normalizeJSON(right)
		return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
	}
	normalizeHelp := func(value []byte) []byte {
		return []byte(strings.ReplaceAll(string(value), "reinstate", "rein"))
	}
	return bytes.Equal(normalizeHelp(left), normalizeHelp(right))
}

func measureCorpus(config runConfig, spec fixtureSpec, manifest corpusManifest) (corpusResult, error) {
	environment, err := corpusEnvironment(config.Root, manifest, config.CuratedPath)
	if err != nil {
		return corpusResult{}, err
	}
	corpusRoot := filepath.Join(config.Root, manifest.Corpus)
	workingDirectory := filepath.Join(corpusRoot, manifest.RelativeWorkingDirectory)
	commands := performanceCommands(manifest)
	workspaceHeadBefore, err := validateWorkspaceRepository(config, spec, manifest)
	if err != nil {
		return corpusResult{}, err
	}
	if workspaceHeadBefore != manifest.WorkspaceGitExpectedHead {
		return corpusResult{}, errors.New("controlled Git HEAD differs from frozen manifest")
	}
	fingerprintBefore, err := fingerprintImmutableFixture(config.Root, manifest)
	if err != nil {
		return corpusResult{}, err
	}
	if err := validateAliasParity(config, environment, workingDirectory, manifest, commands); err != nil {
		return corpusResult{}, err
	}

	coldDurations := make([]time.Duration, 0, coldSampleCount)
	for sample := 1; sample <= coldSampleCount; sample++ {
		if err := moveColdIndexFamily(corpusRoot, sample); err != nil {
			return corpusResult{}, err
		}
		capture, err := executeProcess(config.Rein, commands[0].Args, workingDirectory, environment)
		if err != nil {
			return corpusResult{}, fmt.Errorf("cold sample %d: %w", sample, err)
		}
		if err := validateCommandResult(commands[0], capture.Stdout, manifest); err != nil {
			return corpusResult{}, fmt.Errorf("cold sample %d validation: %w", sample, err)
		}
		coldDurations = append(coldDurations, capture.Elapsed)
	}
	// Establish and validate the exact warm state outside timing.
	seed, err := executeProcess(config.Rein, commands[0].Args, workingDirectory, environment)
	if err != nil {
		return corpusResult{}, fmt.Errorf("warm-state refresh: %w", err)
	}
	if err := validateCommandResult(commands[0], seed.Stdout, manifest); err != nil {
		return corpusResult{}, err
	}

	warm := make([]commandMetrics, 0, len(commands))
	for _, definition := range commands {
		for index := 0; index < warmupCount; index++ {
			capture, runErr := executeProcess(config.Rein, definition.Args, workingDirectory, environment)
			if runErr != nil {
				return corpusResult{}, fmt.Errorf("%s warmup: %w", definition.Label, runErr)
			}
			if err := validateCommandResult(definition, capture.Stdout, manifest); err != nil {
				return corpusResult{}, fmt.Errorf("%s warmup validation: %w", definition.Label, err)
			}
		}
		durations := make([]time.Duration, 0, warmSampleCount)
		for sample := 0; sample < warmSampleCount; sample++ {
			capture, runErr := executeProcess(config.Rein, definition.Args, workingDirectory, environment)
			if runErr != nil {
				return corpusResult{}, fmt.Errorf("%s sample %d: %w", definition.Label, sample+1, runErr)
			}
			if err := validateCommandResult(definition, capture.Stdout, manifest); err != nil {
				return corpusResult{}, fmt.Errorf("%s sample %d validation: %w", definition.Label, sample+1, err)
			}
			durations = append(durations, capture.Elapsed)
		}
		metrics := calculateMetrics(durations)
		metrics.Warmups = warmupCount
		warm = append(warm, commandMetrics{Label: definition.Label, Metrics: metrics})
	}
	fingerprintAfter, err := fingerprintImmutableFixture(config.Root, manifest)
	if err != nil {
		return corpusResult{}, err
	}
	if fingerprintBefore != fingerprintAfter {
		return corpusResult{}, errors.New("controlled fixture source changed during measurement")
	}
	workspaceHeadAfter, err := validateWorkspaceRepository(config, spec, manifest)
	if err != nil {
		return corpusResult{}, err
	}
	if workspaceHeadBefore != workspaceHeadAfter {
		return corpusResult{}, errors.New("controlled Git workspace identity changed during measurement")
	}
	return corpusResult{
		Corpus:                    manifest.Corpus,
		CanonicalDigest:           manifest.CanonicalDigest,
		MaterializedDigest:        manifest.MaterializedDigest,
		RecordCount:               manifest.TotalRecords,
		CapabilityNameCount:       manifest.TotalCapabilityNames,
		Limit:                     manifest.Limit,
		AliasParity:               true,
		OpenCodeOmitted:           true,
		AmbientCapabilitiesAbsent: true,
		WorkspaceGitHead:          workspaceHeadAfter,
		WorkspaceGitClean:         true,
		SourceFingerprintBefore:   fingerprintBefore,
		SourceFingerprintAfter:    fingerprintAfter,
		ColdSessions:              calculateMetrics(coldDurations),
		WarmCommands:              warm,
	}, nil
}

func performanceCommands(manifest corpusManifest) []commandDefinition {
	limit := strconv.Itoa(manifest.Limit)
	return []commandDefinition{
		{Label: "sessions", Args: []string{"sessions", "--limit", limit, "--json"}},
		{Label: "search", Args: []string{"search", manifest.Query, "--limit", limit, "--json"}},
		{Label: "inspect_claude", Args: []string{"inspect", manifest.ClaudeReference, "--json"}},
		{Label: "resume_claude_dry_run", Args: []string{"resume", manifest.ClaudeReference, "--dry-run", "--json"}},
		{Label: "resume_codex_dry_run", Args: []string{"resume", manifest.CodexReference, "--dry-run", "--json"}},
		{Label: "fork_claude_dry_run", Args: []string{"fork", manifest.ClaudeReference, "--dry-run", "--json"}},
		{Label: "fork_codex_dry_run", Args: []string{"fork", manifest.CodexReference, "--dry-run", "--json"}},
	}
}

func validateAliasParity(config runConfig, environment []string, workingDirectory string, manifest corpusManifest, commands []commandDefinition) error {
	for _, definition := range commands {
		left, err := executeProcess(config.Rein, definition.Args, workingDirectory, environment)
		if err != nil {
			return fmt.Errorf("rein alias precondition %s: %w", definition.Label, err)
		}
		if err := validateCommandResult(definition, left.Stdout, manifest); err != nil {
			return err
		}
		right, err := executeProcess(config.Reinstate, definition.Args, workingDirectory, environment)
		if err != nil {
			return fmt.Errorf("reinstate alias precondition %s: %w", definition.Label, err)
		}
		if err := validateCommandResult(definition, right.Stdout, manifest); err != nil {
			return err
		}
		leftJSON, err := normalizeJSON(left.Stdout)
		if err != nil {
			return err
		}
		rightJSON, err := normalizeJSON(right.Stdout)
		if err != nil {
			return err
		}
		if !bytes.Equal(leftJSON, rightJSON) {
			return fmt.Errorf("alias JSON mismatch for %s", definition.Label)
		}
	}
	return nil
}

func validateCommandResult(definition commandDefinition, output []byte, manifest corpusManifest) error {
	switch definition.Label {
	case "sessions", "search":
		var value strictSessionsOutput
		if err := decodeOneStrictJSON(output, &value); err != nil {
			return err
		}
		if manifest.TotalRecords > manifest.Limit {
			return errors.New("corpus count exceeds fixed result limit")
		}
		if len(value.Sessions) != manifest.TotalRecords {
			return fmt.Errorf("session result count = %d want %d", len(value.Sessions), manifest.TotalRecords)
		}
		expected, err := expectedRecordValues(manifest)
		if err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(value.Sessions))
		for _, session := range value.Sessions {
			want, ok := expected[session.Key]
			if !ok {
				return errors.New("result contains an unexpected session key")
			}
			if _, duplicate := seen[session.Key]; duplicate {
				return errors.New("result contains a duplicate session key")
			}
			seen[session.Key] = struct{}{}
			if session.Agent != want.Agent || session.Title != want.Title || !session.UpdatedAt.Equal(want.UpdatedAt) ||
				session.MessageCount != 2 || len(session.Files) != 2 ||
				session.Files[0] != "src/perf-aa.go" || session.Files[1] != "src/perf-bb.go" {
				return errors.New("result violates frozen message/file fanout")
			}
		}
		if _, ok := seen[manifest.ClaudeReference]; !ok {
			return errors.New("claude anchor is not visible within fixed limit")
		}
		if _, ok := seen[manifest.CodexReference]; !ok {
			return errors.New("codex anchor is not visible within fixed limit")
		}
		if len(value.Warnings) != 1 || value.Warnings[0].Agent != "opencode" || value.Warnings[0].Code != "agent_not_installed" {
			return errors.New("source warning set does not prove deterministic OpenCode omission")
		}
		return nil
	case "inspect_claude":
		return validatePreflightOutput(output, manifest, manifest.ClaudeReference, "claude", "")
	case "resume_claude_dry_run":
		return validatePreflightOutput(output, manifest, manifest.ClaudeReference, "claude", "resume")
	case "resume_codex_dry_run":
		return validatePreflightOutput(output, manifest, manifest.CodexReference, "codex", "resume")
	case "fork_claude_dry_run":
		return validatePreflightOutput(output, manifest, manifest.ClaudeReference, "claude", "fork")
	case "fork_codex_dry_run":
		return validatePreflightOutput(output, manifest, manifest.CodexReference, "codex", "fork")
	default:
		return errors.New("unknown performance command")
	}
}

func validatePreflightOutput(output []byte, manifest corpusManifest, reference, agent, operation string) error {
	var sessionRef string
	var environment preflight.Report
	if operation == "" {
		var value strictInspectOutput
		if err := decodeOneStrictJSON(output, &value); err != nil {
			return err
		}
		sessionRef = value.Session.Key
		environment = value.Environment
	} else {
		var value strictLaunchOutput
		if err := decodeOneStrictJSON(output, &value); err != nil {
			return err
		}
		sessionRef = value.SessionRef
		environment = value.Environment
		if value.Operation != operation {
			return errors.New("dry-run output has unexpected operation")
		}
		if value.Executable != agent {
			return errors.New("dry-run output has unexpected executable")
		}
	}
	if sessionRef != reference {
		return errors.New("preflight output selected the wrong anchor")
	}
	if err := validatePreflightDynamicValues(environment); err != nil {
		return err
	}
	if environment.SchemaVersion != preflight.SchemaVersion || environment.SessionRef != reference {
		return errors.New("preflight output has an unexpected schema or session reference")
	}
	if environment.Decision == preflight.DecisionBlocked || environment.Decision == "" {
		return errors.New("preflight decision is not a valid warning/ready result")
	}
	wantCount := manifest.ClaudeCapabilityNames
	if agent == "codex" {
		wantCount = manifest.CodexCapabilityNames
	}
	if len(environment.Capabilities.Items) != wantCount {
		return fmt.Errorf("%s capability count = %d want %d; ambient managed capabilities contaminated corpus", agent, len(environment.Capabilities.Items), wantCount)
	}
	for index, item := range environment.Capabilities.Items {
		if string(item.Agent) != agent || len(item.Name) != manifest.CapabilityNameBytes {
			return errors.New("capability identity differs from frozen corpus")
		}
		expected := capabilityName(agent, index, manifest.CapabilityNameBytes)
		if item.Name != expected {
			return errors.New("ambient or missing capability name changed frozen inventory")
		}
	}
	return nil
}

func validatePreflightDynamicValues(report preflight.Report) error {
	for _, check := range report.Checks {
		if !safePreflightDynamicValue(check.Expected) || !safePreflightDynamicValue(check.Actual) {
			return errors.New("preflight check contains an unsupported dynamic JSON value")
		}
	}
	return nil
}

func safePreflightDynamicValue(value any) bool {
	switch typed := value.(type) {
	case nil, bool, string:
		return true
	case []any:
		for _, item := range typed {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

type expectedRecord struct {
	Agent     string
	Title     string
	UpdatedAt time.Time
}

func expectedRecordValues(manifest corpusManifest) (map[string]expectedRecord, error) {
	base, err := time.Parse(time.RFC3339, manifest.TimestampBase)
	if err != nil {
		return nil, errors.New("manifest timestamp base is invalid")
	}
	result := make(map[string]expectedRecord, manifest.TotalRecords)
	for index := 0; index < manifest.ClaudeRecords; index++ {
		result["claude:"+fixtureID(1, index)] = expectedRecord{
			Agent:     "claude",
			Title:     fixedASCII(fmt.Sprintf("phase3-%s-claude-%04d", manifest.Corpus, index), manifest.TitleBytes, 't'),
			UpdatedAt: base.Add(time.Duration(int64(index)*manifest.TimestampStepSeconds) * time.Second),
		}
	}
	for index := 0; index < manifest.CodexRecords; index++ {
		result["codex:"+fixtureID(2, index)] = expectedRecord{
			Agent:     "codex",
			Title:     fixedASCII(fmt.Sprintf("phase3-%s-codex-%04d", manifest.Corpus, index), manifest.TitleBytes, 't'),
			UpdatedAt: base.Add(time.Duration(int64(index)*manifest.TimestampStepSeconds) * time.Second),
		}
	}
	return result, nil
}

func executeProcess(binary string, args []string, directory string, environment []string) (commandCapture, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	stdout := cappedBuffer{limit: maxStdoutBytes}
	stderr := cappedBuffer{limit: maxStderrBytes}
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = directory
	command.Env = append([]string(nil), environment...)
	command.Stdin = bytes.NewReader(nil)
	command.Stdout = &stdout
	command.Stderr = &stderr
	start := time.Now()
	err := command.Run()
	elapsed := time.Since(start)
	capture := commandCapture{Stdout: stdout.buffer.Bytes(), Stderr: stderr.buffer.Bytes(), Elapsed: elapsed, ExitCode: 0}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		capture.TimedOut = true
		return capture, errors.New("command timed out after 30 seconds")
	}
	if stdout.exceeded || stderr.exceeded {
		return capture, errors.New("command output exceeded fixed capture limit")
	}
	if err != nil {
		capture.ExitCode = -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			capture.ExitCode = exitError.ExitCode()
		}
		return capture, fmt.Errorf("command exited %d", capture.ExitCode)
	}
	if len(bytes.TrimSpace(capture.Stderr)) != 0 {
		return capture, errors.New("command produced unexpected stderr")
	}
	return capture, nil
}

func corpusEnvironment(root string, manifest corpusManifest, curatedPath string) ([]string, error) {
	corpusRoot := filepath.Join(root, manifest.Corpus)
	processHome := filepath.Join(corpusRoot, manifest.RelativeProcessHome)
	temporary := filepath.Join(corpusRoot, "tmp")
	for _, directory := range []string{
		filepath.Join(corpusRoot, manifest.RelativeReinstateHome),
		filepath.Join(corpusRoot, manifest.RelativeClaudeConfigDir),
		filepath.Join(corpusRoot, manifest.RelativeCodexHome),
		filepath.Join(corpusRoot, manifest.RelativeGeminiCLIHome),
		processHome,
		temporary,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, err
		}
	}
	emptyGitConfig := filepath.Join(processHome, "phase3-perf-empty-gitconfig")
	if err := os.WriteFile(emptyGitConfig, nil, 0o600); err != nil {
		return nil, err
	}
	values := map[string]string{
		"PATH":                   curatedPath,
		"HOME":                   processHome,
		"USERPROFILE":            processHome,
		"REINSTATE_HOME":         filepath.Join(corpusRoot, manifest.RelativeReinstateHome),
		"CLAUDE_CONFIG_DIR":      filepath.Join(corpusRoot, manifest.RelativeClaudeConfigDir),
		"CODEX_HOME":             filepath.Join(corpusRoot, manifest.RelativeCodexHome),
		"GEMINI_CLI_HOME":        filepath.Join(corpusRoot, manifest.RelativeGeminiCLIHome),
		"TMPDIR":                 temporary,
		"TMP":                    temporary,
		"TEMP":                   temporary,
		"XDG_CONFIG_HOME":        filepath.Join(processHome, ".config"),
		"APPDATA":                filepath.Join(processHome, "AppData", "Roaming"),
		"LOCALAPPDATA":           filepath.Join(processHome, "AppData", "Local"),
		"GIT_CONFIG_GLOBAL":      emptyGitConfig,
		"GIT_CONFIG_NOSYSTEM":    "1",
		"GIT_ATTR_NOSYSTEM":      "1",
		"GIT_NO_REPLACE_OBJECTS": "1",
		"GIT_NO_LAZY_FETCH":      "1",
		"GIT_TERMINAL_PROMPT":    "0",
		"GIT_OPTIONAL_LOCKS":     "0",
		"LC_ALL":                 "C",
		"LANG":                   "C",
		"TZ":                     "UTC",
		"NO_COLOR":               "1",
	}
	for _, inherited := range []string{"SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
		if value := os.Getenv(inherited); value != "" {
			values[inherited] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment, nil
}

func startupEnvironment(root, label, curatedPath string) (string, []string, error) {
	base := filepath.Join(root, "startup", label)
	workingDirectory := filepath.Join(base, "workspace")
	processHome := filepath.Join(base, "process-home")
	temporary := filepath.Join(base, "tmp")
	for _, directory := range []string{workingDirectory, processHome, temporary, filepath.Join(base, "reinstate-home")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return "", nil, err
		}
	}
	values := map[string]string{
		"PATH":              curatedPath,
		"HOME":              processHome,
		"USERPROFILE":       processHome,
		"REINSTATE_HOME":    filepath.Join(base, "reinstate-home"),
		"CLAUDE_CONFIG_DIR": filepath.Join(base, "claude"),
		"CODEX_HOME":        filepath.Join(base, "codex"),
		"GEMINI_CLI_HOME":   filepath.Join(base, "gemini"),
		"TMPDIR":            temporary,
		"TMP":               temporary,
		"TEMP":              temporary,
		"LC_ALL":            "C",
		"LANG":              "C",
		"TZ":                "UTC",
		"NO_COLOR":          "1",
	}
	for _, inherited := range []string{"SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
		if value := os.Getenv(inherited); value != "" {
			values[inherited] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return workingDirectory, environment, nil
}

func initializeWorkspaceRepository(config runConfig, spec fixtureSpec, manifest corpusManifest) (string, error) {
	corpusRoot := filepath.Join(config.Root, manifest.Corpus)
	workspace := filepath.Join(corpusRoot, manifest.RelativeWorkingDirectory)
	if _, err := os.Lstat(filepath.Join(workspace, ".git")); !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("controlled workspace Git metadata already exists")
	}
	environment, err := corpusEnvironment(config.Root, manifest, config.CuratedPath)
	if err != nil {
		return "", err
	}
	environment = append(environment,
		"GIT_AUTHOR_NAME="+spec.GitAuthorName,
		"GIT_AUTHOR_EMAIL="+spec.GitAuthorEmail,
		"GIT_AUTHOR_DATE="+spec.GitCommitTimestamp,
		"GIT_COMMITTER_NAME="+spec.GitAuthorName,
		"GIT_COMMITTER_EMAIL="+spec.GitAuthorEmail,
		"GIT_COMMITTER_DATE="+spec.GitCommitTimestamp,
	)
	sort.Strings(environment)
	hooks := filepath.Join(corpusRoot, "tmp", "empty-hooks")
	if err := os.MkdirAll(hooks, 0o700); err != nil {
		return "", err
	}
	commands := [][]string{
		{"init", "--quiet", "--initial-branch", spec.GitBranch, "--object-format", "sha1"},
		{"add", "--", "phase3-performance.txt"},
		{"-c", "commit.gpgSign=false", "-c", "core.hooksPath=" + hooks, "commit", "--quiet", "-m", spec.GitCommitMessage},
	}
	for _, args := range commands {
		if _, err := executeProcess("git", args, workspace, environment); err != nil {
			return "", errors.New("create controlled Git workspace")
		}
	}
	head, err := validateWorkspaceRepository(config, spec, manifest)
	if err != nil {
		return "", err
	}
	if spec.GitExpectedHead != "PENDING" && head != spec.GitExpectedHead {
		return "", fmt.Errorf("controlled Git HEAD = %s want %s", head, spec.GitExpectedHead)
	}
	return head, nil
}

func validateWorkspaceRepository(config runConfig, spec fixtureSpec, manifest corpusManifest) (string, error) {
	corpusRoot := filepath.Join(config.Root, manifest.Corpus)
	workspace := filepath.Join(corpusRoot, manifest.RelativeWorkingDirectory)
	environment, err := corpusEnvironment(config.Root, manifest, config.CuratedPath)
	if err != nil {
		return "", err
	}
	if err := validateControlledGitLocalConfig(workspace, environment); err != nil {
		return "", err
	}
	safeGitPrefix := []string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false"}
	headOutput, err := runSmallCommand(workspace, environment, "git", append(safeGitPrefix, "rev-parse", "--verify", "HEAD^{commit}")...)
	if err != nil {
		return "", errors.New("read controlled Git HEAD")
	}
	head := strings.TrimSpace(string(headOutput))
	if len(head) != 40 {
		return "", errors.New("controlled Git repository did not use SHA-1 identity")
	}
	branchOutput, err := runSmallCommand(workspace, environment, "git", append(safeGitPrefix, "symbolic-ref", "--quiet", "HEAD")...)
	if err != nil || strings.TrimSpace(string(branchOutput)) != "refs/heads/"+spec.GitBranch {
		return "", errors.New("controlled Git branch changed")
	}
	treeOutput, err := runSmallCommand(workspace, environment, "git", append(safeGitPrefix, "ls-tree", "-r", "-z", "--full-tree", "HEAD")...)
	if err != nil {
		return "", errors.New("read controlled Git tree")
	}
	treeBlob, err := parseSingleGitEntry(treeOutput, "phase3-performance.txt", false)
	if err != nil {
		return "", errors.New("controlled Git HEAD tree changed")
	}
	indexOutput, err := runSmallCommand(workspace, environment, "git", append(safeGitPrefix, "ls-files", "--stage", "-z", "--", "phase3-performance.txt")...)
	if err != nil {
		return "", errors.New("read controlled Git index")
	}
	indexBlob, err := parseSingleGitEntry(indexOutput, "phase3-performance.txt", true)
	if err != nil || indexBlob != treeBlob {
		return "", errors.New("controlled Git index changed")
	}
	content, err := readBoundedRegular(filepath.Join(workspace, "phase3-performance.txt"), int64(spec.WorkspaceFileBytes))
	if err != nil || len(content) != spec.WorkspaceFileBytes || gitBlobSHA1(content) != treeBlob {
		return "", errors.New("controlled Git workspace content changed")
	}
	if err := validateControlledWorkspaceFiles(workspace); err != nil {
		return "", err
	}
	return head, nil
}

func validateControlledGitLocalConfig(workspace string, environment []string) error {
	output, err := runSmallCommand(workspace, environment, "git", "config", "--local", "--no-includes", "--null", "--name-only", "--list")
	if err != nil {
		return errors.New("read controlled Git local configuration")
	}
	allowed := map[string]struct{}{
		"core.repositoryformatversion": {},
		"core.filemode":                {},
		"core.bare":                    {},
		"core.logallrefupdates":        {},
		"core.ignorecase":              {},
		"core.precomposeunicode":       {},
		"core.symlinks":                {},
		"extensions.objectformat":      {},
	}
	for _, encoded := range bytes.Split(output, []byte{0}) {
		key := strings.ToLower(strings.TrimSpace(string(encoded)))
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return errors.New("controlled Git local configuration contains an unexpected key")
		}
	}
	return nil
}

func parseSingleGitEntry(output []byte, expectedPath string, index bool) (string, error) {
	entries := bytes.Split(output, []byte{0})
	if len(entries) != 2 || len(entries[0]) == 0 || len(entries[1]) != 0 {
		return "", errors.New("git entry count differs from the frozen workspace")
	}
	header, name, ok := bytes.Cut(entries[0], []byte{'\t'})
	if !ok || string(name) != expectedPath {
		return "", errors.New("git entry path differs from the frozen workspace")
	}
	fields := strings.Fields(string(header))
	if !index {
		if len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || len(fields[2]) != 40 {
			return "", errors.New("git tree entry differs from the frozen workspace")
		}
		return fields[2], nil
	}
	if len(fields) != 3 || fields[0] != "100644" || len(fields[1]) != 40 || fields[2] != "0" {
		return "", errors.New("git index entry differs from the frozen workspace")
	}
	return fields[1], nil
}

func gitBlobSHA1(content []byte) string {
	hash := sha1.New() // #nosec G401 -- compatibility with the controlled Git SHA-1 repository.
	_, _ = fmt.Fprintf(hash, "blob %d\x00", len(content))
	_, _ = hash.Write(content)
	return hex.EncodeToString(hash.Sum(nil))
}

func validateControlledWorkspaceFiles(workspace string) error {
	seen := 0
	err := filepath.WalkDir(workspace, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == workspace {
			return nil
		}
		if entry.IsDir() && entry.Name() == ".git" && filepath.Dir(path) == workspace {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return errors.New("controlled Git worktree contains an unexpected directory")
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || entry.Name() != "phase3-performance.txt" || filepath.Dir(path) != workspace {
			return errors.New("controlled Git worktree contains an unexpected file")
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != 1 {
		return errors.New("controlled Git worktree file count changed")
	}
	return nil
}

func minimalGitEnvironment(curatedPath string) []string {
	environment := []string{
		"PATH=" + curatedPath,
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
		"LC_ALL=C",
		"LANG=C",
	}
	for _, inherited := range []string{"SystemRoot", "WINDIR", "COMSPEC", "PATHEXT", "HOME", "USERPROFILE", "TMPDIR", "TMP", "TEMP"} {
		if value := os.Getenv(inherited); value != "" {
			environment = append(environment, inherited+"="+value)
		}
	}
	sort.Strings(environment)
	return environment
}

func moveColdIndexFamily(corpusRoot string, sample int) error {
	cache := filepath.Join(corpusRoot, "reinstate-home", "cache")
	destination := filepath.Join(corpusRoot, "cold-evidence", fmt.Sprintf("sample-%d", sample))
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	movedDatabase := false
	for _, name := range coldIndexFamily {
		source := filepath.Join(cache, name)
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("derived v2 family member is not a regular file")
		}
		if name == "session-index-v2.sqlite" {
			movedDatabase = true
		}
		if err := os.Rename(source, filepath.Join(destination, name)); err != nil {
			return fmt.Errorf("preserve derived v2 family: %w", err)
		}
	}
	if !movedDatabase {
		return errors.New("cold reset did not find the derived v2 database")
	}
	for _, name := range coldIndexFamily {
		if _, err := os.Lstat(filepath.Join(cache, name)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("cold reset left a derived v2 family member in cache")
		}
	}
	return nil
}

func verifyGeneratedFixture(root string, spec fixtureSpec, manifest aggregateManifest) error {
	for _, corpus := range []struct {
		spec     corpusSpec
		manifest corpusManifest
	}{{spec.Normal, manifest.Normal}, {spec.Large, manifest.Large}} {
		files, expected, err := buildCorpus(filepath.Join(root, corpus.spec.Name), spec, corpus.spec)
		if err != nil {
			return err
		}
		if expected.TotalRecords != corpus.manifest.TotalRecords || expected.TotalEvents != corpus.manifest.TotalEvents ||
			expected.TotalMessages != corpus.manifest.TotalMessages || expected.TotalCapabilityNames != corpus.manifest.TotalCapabilityNames {
			return errors.New("generated manifest count mismatch")
		}
		for _, file := range files {
			path := filepath.Join(root, filepath.FromSlash(file.RelativePath))
			info, err := os.Lstat(path)
			if err != nil {
				return errors.New("generated fixture file is missing")
			}
			if file.Mode.IsDir() {
				if !info.IsDir() {
					return errors.New("generated fixture directory has wrong type")
				}
				continue
			}
			if info.Size() > maxImmutableFixtureFileBytes {
				return errors.New("immutable fixture source exceeds the fixed file-size limit")
			}
			content, err := readBoundedRegular(path, maxImmutableFixtureFileBytes)
			if err != nil || !bytes.Equal(content, file.Materialized) {
				return errors.New("generated fixture bytes differ from frozen schema")
			}
		}
	}
	return nil
}

func fingerprintImmutableFixture(root string, manifest corpusManifest) (string, error) {
	hash := sha256.New()
	corpusRoot := filepath.Join(root, manifest.Corpus)
	controlledRoots := []string{
		filepath.Join(corpusRoot, manifest.RelativeClaudeConfigDir),
		filepath.Join(corpusRoot, manifest.RelativeCodexHome),
		filepath.Join(corpusRoot, manifest.RelativeGeminiCLIHome),
		filepath.Join(corpusRoot, manifest.RelativeProcessHome, ".agents", "skills"),
	}
	for _, controlledRoot := range controlledRoots {
		err := filepath.WalkDir(controlledRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.New("immutable fixture contains a symlink")
			}
			if entry.IsDir() {
				writeDigestEntry(hash, relative+"/", nil)
				return nil
			}
			if !info.Mode().IsRegular() {
				return errors.New("immutable fixture contains a non-regular file")
			}
			if info.Size() > maxImmutableFixtureFileBytes {
				return errors.New("immutable fixture source exceeds the fixed file-size limit")
			}
			content, err := readBoundedRegular(path, maxImmutableFixtureFileBytes)
			if err != nil {
				return err
			}
			writeDigestEntry(hash, relative, content)
			_, _ = io.WriteString(hash, fmt.Sprintf("%d\x00%d\x00", info.Size(), info.ModTime().UnixNano()))
			return nil
		})
		if err != nil {
			return "", errors.New("fingerprint immutable fixture source tree")
		}
	}
	workspaceFile := filepath.Join(corpusRoot, manifest.RelativeWorkingDirectory, "phase3-performance.txt")
	content, err := readBoundedRegular(workspaceFile, maxImmutableFixtureFileBytes)
	if err != nil {
		return "", errors.New("read immutable workspace fixture")
	}
	info, err := os.Stat(workspaceFile)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("stat immutable workspace fixture")
	}
	writeDigestEntry(hash, filepath.ToSlash(filepath.Join(manifest.Corpus, manifest.RelativeWorkingDirectory, "phase3-performance.txt")), content)
	_, _ = io.WriteString(hash, fmt.Sprintf("%d\x00%d\x00", info.Size(), info.ModTime().UnixNano()))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func calculateMetrics(values []time.Duration) durationMetrics {
	if len(values) == 0 {
		return durationMetrics{}
	}
	nanoseconds := make([]int64, len(values))
	for index, value := range values {
		nanoseconds[index] = value.Nanoseconds()
	}
	sort.Slice(nanoseconds, func(i, j int) bool { return nanoseconds[i] < nanoseconds[j] })
	median := nanoseconds[len(nanoseconds)/2]
	if len(nanoseconds)%2 == 0 {
		left, right := nanoseconds[len(nanoseconds)/2-1], nanoseconds[len(nanoseconds)/2]
		median = left + (right-left)/2
	}
	p95Index := int(math.Ceil(0.95*float64(len(nanoseconds)))) - 1
	return durationMetrics{
		Samples: len(values), MedianNS: median, P95NS: nanoseconds[p95Index],
		MaximumNS: nanoseconds[len(nanoseconds)-1], Validated: true,
	}
}

func normalizeJSON(value []byte) ([]byte, error) {
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, errors.New("command output is not valid JSON")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func decodeOneStrictJSON(value []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("command output violates the exact JSON schema")
	}
	return ensureJSONEOF(decoder)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("command output contains more than one JSON value")
	}
	return nil
}

func runSmallCommand(directory string, environment []string, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	stdout := cappedBuffer{limit: maxSmallCommandStdoutBytes}
	stderr := cappedBuffer{limit: maxSmallCommandStderrBytes}
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = directory
	command.Env = environment
	command.Stdin = bytes.NewReader(nil)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("bounded helper command timed out")
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, errors.New("bounded helper command output exceeded its limit")
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(stderr.buffer.Bytes())) != 0 {
		return nil, errors.New("bounded helper command produced stderr")
	}
	return append([]byte(nil), stdout.buffer.Bytes()...), nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxInstalledBinaryBytes {
		return "", errors.New("installed binary is not a bounded regular file")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maxInstalledBinaryBytes+1))
	if err != nil || written != info.Size() {
		return "", errors.New("hash installed binary")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func readBoundedRegular(path string, maximum int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maximum {
		return nil, errors.New("path is not a bounded regular file")
	}
	content, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) != info.Size() || int64(len(content)) > maximum {
		return nil, errors.New("bounded regular file changed while reading")
	}
	return content, nil
}

func requireInstalledAliasHashes(config runConfig, expected string) error {
	reinHash, err := fileSHA256(config.Rein)
	if err != nil {
		return errors.New("rehash installed rein alias")
	}
	reinstateHash, err := fileSHA256(config.Reinstate)
	if err != nil {
		return errors.New("rehash installed reinstate alias")
	}
	if reinHash != expected || reinstateHash != expected {
		return errors.New("installed alias bytes changed during performance measurement")
	}
	return nil
}

func digestEnvironment(environment []string) string {
	hash := sha256.New()
	for _, value := range environment {
		_, _ = io.WriteString(hash, value)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
