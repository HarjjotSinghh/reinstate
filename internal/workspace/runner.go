package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/executabletrust"
	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

const maxGitOutputBytes = 4 << 20

var (
	ErrGitUnavailable   = errors.New("git executable is unavailable")
	ErrOutputTooLarge   = errors.New("git command output exceeds the safe limit")
	ErrNotRepository    = errors.New("workspace is not a Git repository")
	ErrUnsafeGitCommand = errors.New("git command is outside the read-only probe allowlist")
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
	location, err := discoverRepositoryLocation(dir)
	if err != nil {
		return nil, err
	}
	resolution, err := executabletrust.Resolve("git", dir, os.Environ())
	if err != nil {
		return nil, ErrGitUnavailable
	}
	runner := resolvedGitRunner{
		executable: resolution.Executable,
		environment: bindGitRepository(
			safeGitEnvironmentWithPath(os.Environ(), resolution.SearchPath),
			location,
		),
	}
	if stringSlicesEqual(args, gitRootArgs) {
		inside, runErr := runner.run(ctx, location.root, "rev-parse", "--is-inside-work-tree")
		if runErr != nil || strings.TrimSpace(string(inside)) != "true" {
			if runErr != nil {
				return nil, runErr
			}
			return nil, ErrNotRepository
		}
		return []byte(location.root + "\n"), nil
	}
	if stringSlicesEqual(args, gitStatusArgs) {
		return runner.safeStatus(ctx, location.root)
	}
	return runner.run(ctx, location.root, args...)
}

type repositoryLocation struct {
	root   string
	gitDir string
}

func discoverRepositoryLocation(dir string) (repositoryLocation, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return repositoryLocation{}, err
	}
	physical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return repositoryLocation{}, ErrNotRepository
		}
		return repositoryLocation{}, err
	}
	current := filepath.Clean(physical)
	info, err := os.Stat(current)
	if err != nil || !info.IsDir() {
		return repositoryLocation{}, ErrNotRepository
	}
	for {
		marker := filepath.Join(current, ".git")
		markerInfo, markerErr := os.Lstat(marker)
		switch {
		case markerErr == nil && markerInfo.IsDir():
			if gitDirectoryLooksValid(marker) {
				return repositoryLocation{root: current, gitDir: marker}, nil
			}
		case markerErr == nil && markerInfo.Mode().IsRegular():
			gitDir, parseErr := parseGitDirectoryFile(current, marker)
			if parseErr == nil && gitDirectoryLooksValid(gitDir) {
				return repositoryLocation{root: current, gitDir: gitDir}, nil
			}
		case markerErr == nil:
			// An invalid nested marker must not narrow repository discovery.
			// This matches executable-trust boundary discovery and prevents an
			// attacker-created empty .git entry from masking the outer repo.
		case !errors.Is(markerErr, os.ErrNotExist):
			return repositoryLocation{}, markerErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return repositoryLocation{}, ErrNotRepository
		}
		current = parent
	}
}

func gitDirectoryLooksValid(gitDir string) bool {
	directory, err := os.Lstat(gitDir)
	if err != nil || !directory.IsDir() || directory.Mode()&os.ModeSymlink != 0 {
		return false
	}
	info, err := os.Lstat(filepath.Join(gitDir, "HEAD"))
	return err == nil && info.Mode().IsRegular()
}

func parseGitDirectoryFile(root, marker string) (string, error) {
	before, err := os.Lstat(marker)
	if err != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > 4096 {
		return "", ErrNotRepository
	}
	handle, err := os.Open(marker)
	if err != nil {
		return "", ErrNotRepository
	}
	defer func() { _ = handle.Close() }()
	opened, err := handle.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() > 4096 || !os.SameFile(before, opened) {
		return "", ErrNotRepository
	}
	contents, err := io.ReadAll(io.LimitReader(handle, 4097))
	if err != nil || len(contents) > 4096 || bytes.IndexByte(contents, 0) >= 0 {
		return "", ErrNotRepository
	}
	final, err := handle.Stat()
	if err != nil || final.Size() != opened.Size() || !os.SameFile(opened, final) {
		return "", ErrNotRepository
	}
	line := strings.TrimSpace(string(contents))
	if !strings.HasPrefix(line, "gitdir:") || strings.ContainsAny(line, "\r\n") {
		return "", ErrNotRepository
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if gitDir == "" {
		return "", ErrNotRepository
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	gitDir, err = filepath.Abs(gitDir)
	if err != nil {
		return "", err
	}
	gitDir = filepath.Clean(gitDir)
	info, err := os.Lstat(gitDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrNotRepository
	}
	return gitDir, nil
}

func bindGitRepository(environment []string, location repositoryLocation) []string {
	return append(environment,
		"GIT_DIR="+location.gitDir,
		"GIT_WORK_TREE="+location.root,
		"GIT_CEILING_DIRECTORIES="+filepath.Dir(location.root),
		"GIT_DISCOVERY_ACROSS_FILESYSTEM=0",
	)
}

type resolvedGitRunner struct {
	executable  string
	environment []string
}

func (runner resolvedGitRunner) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return runner.runInput(ctx, dir, nil, args...)
}

func (runner resolvedGitRunner) runInput(ctx context.Context, dir string, input []byte, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, runner.executable, args...)
	command.Dir = dir
	command.Env = runner.environment
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	var stdout boundedBuffer
	var stderr boundedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
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

var (
	gitRootArgs         = []string{"rev-parse", "--path-format=absolute", "--show-toplevel"}
	gitSymbolicHeadArgs = []string{"symbolic-ref", "--quiet", "--short", "HEAD"}
	gitHeadArgs         = []string{"rev-parse", "--verify", "HEAD"}
	gitIndexPathArgs    = []string{"rev-parse", "--path-format=absolute", "--git-path", "index"}
	gitUpstreamArgs     = []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"}
	gitUpstreamDiffArgs = []string{"rev-list", "--left-right", "--count", "HEAD...@{upstream}"}
	gitStagedArgs       = []string{
		"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "submodule.recurse=false",
		"diff", "--cached", "--raw", "-z", "--no-ext-diff", "--no-textconv", "--no-renames", "--ignore-submodules=all", "--",
	}
	gitUnstagedArgs = []string{
		"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "submodule.recurse=false",
		"diff-files", "--raw", "-z", "--no-ext-diff", "--no-textconv", "--no-renames", "--ignore-submodules=all", "--",
	}
	gitUntrackedArgs = []string{
		"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "submodule.recurse=false",
		"ls-files", "--others", "--exclude-standard", "-z", "--",
	}
	gitGitlinksArgs = []string{
		"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "submodule.recurse=false",
		"ls-files", "--stage", "-v", "-z", "--",
	}
	gitLocalConfigNamesArgs    = []string{"config", "--no-includes", "--local", "--null", "--name-only", "--list"}
	gitWorktreeConfigNamesArgs = []string{"config", "--no-includes", "--worktree", "--null", "--name-only", "--list"}
	gitFilterAttributesArgs    = []string{
		"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "submodule.recurse=false",
		"check-attr", "-z", "--stdin", "filter",
	}
)

// safeStatus builds the stable porcelain-v2 subset consumed by Reinstate from
// plumbing commands that never request working-tree content conversion. In
// particular, it does not execute Git's status machinery, clean/process
// filters, external diffs, textconv drivers, fsmonitor hooks, or nested
// submodule status. A stat-dirty but content-identical path is intentionally
// reported modified: conservative confirmation is safer than executing
// repository-configured code to prove equivalence.
func (runner resolvedGitRunner) safeStatus(ctx context.Context, dir string) ([]byte, error) {
	var synthesized boundedBuffer
	indexPathOutput, err := runner.run(ctx, dir, gitIndexPathArgs...)
	if err != nil {
		return nil, err
	}
	indexPath := strings.TrimSpace(string(indexPathOutput))
	if indexPath == "" || !filepath.IsAbs(indexPath) || strings.ContainsAny(indexPath, "\x00\r\n") {
		return nil, ErrInvalidGitStatus
	}
	beforeIndex, beforeIndexExists, err := captureIndexSnapshot(ctx, indexPath)
	if err != nil {
		return nil, err
	}
	type commandResult struct {
		args   []string
		output []byte
		err    error
	}
	branchResult := &commandResult{args: gitSymbolicHeadArgs}
	headResult := &commandResult{args: gitHeadArgs}
	localConfigResult := &commandResult{args: gitLocalConfigNamesArgs}
	worktreeConfigResult := &commandResult{args: gitWorktreeConfigNamesArgs}
	results := []*commandResult{
		branchResult, headResult, localConfigResult, worktreeConfigResult,
	}
	var wait sync.WaitGroup
	wait.Add(len(results))
	for _, current := range results {
		go func(result *commandResult) {
			defer wait.Done()
			result.output, result.err = runner.run(ctx, dir, result.args...)
		}(current)
	}
	wait.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if localConfigResult.err != nil {
		return nil, localConfigResult.err
	}
	workingTreeUncertain := unsafeConfigNames(localConfigResult.output)
	if worktreeConfigResult.err == nil {
		if unsafeConfigNames(worktreeConfigResult.output) {
			workingTreeUncertain = true
		}
	} else if exitCode, ok := commandExitCode(worktreeConfigResult.err); !ok || exitCode != 128 {
		return nil, worktreeConfigResult.err
	}

	branchOutput, branchErr := branchResult.output, branchResult.err
	branch, branchSet, err := optionalGitOutput(branchOutput, branchErr, 1)
	if err != nil {
		return nil, err
	}
	headOutput, headErr := headResult.output, headResult.err
	head, headSet, err := optionalGitOutput(headOutput, headErr, 128)
	if err != nil {
		return nil, err
	}
	if headSet && !validObjectID(head) {
		return nil, ErrInvalidGitStatus
	}
	if !headSet && !branchSet {
		return nil, ErrInvalidGitStatus
	}
	state, err := runner.captureStatusState(ctx, dir)
	if err != nil {
		return nil, err
	}
	if headSet {
		writeSynthesizedToken(&synthesized, "# branch.oid "+strings.ToLower(head))
	} else {
		writeSynthesizedToken(&synthesized, "# branch.oid (initial)")
	}
	if branchSet {
		writeSynthesizedToken(&synthesized, "# branch.head "+branch)
	} else {
		writeSynthesizedToken(&synthesized, "# branch.head (detached)")
	}

	upstream, err := runner.captureUpstreamState(ctx, dir, branchSet && headSet)
	if err != nil {
		return nil, err
	}
	if upstream.set {
		writeSynthesizedToken(&synthesized, "# branch.upstream "+upstream.name)
		writeSynthesizedToken(&synthesized, "# branch.ab +"+upstream.ahead+" -"+upstream.behind)
	}

	changes := make(map[string]*rawChange)
	if err := collectRawChanges(changes, state.staged, true); err != nil {
		return nil, err
	}
	if err := collectRawChanges(changes, state.unstaged, false); err != nil {
		return nil, err
	}
	appendRawChanges(&synthesized, changes)
	if err := appendUntracked(&synthesized, state.untracked); err != nil {
		return nil, err
	}
	uncertainIndex, err := conservativeIndexUncertainty(state.index)
	if err != nil {
		return nil, err
	}
	workingTreeUncertain = workingTreeUncertain || uncertainIndex
	trackedPaths, err := trackedPathsFromIndex(state.index)
	if err != nil {
		return nil, err
	}
	filterAttributes, err := runner.runInput(ctx, dir, trackedPaths, gitFilterAttributesArgs...)
	if err != nil {
		return nil, err
	}
	if filterAttributesConfigured(filterAttributes) {
		workingTreeUncertain = true
	}
	afterIndex, afterIndexExists, err := captureIndexSnapshot(ctx, indexPath)
	if err != nil {
		return nil, err
	}
	if beforeIndexExists != afterIndexExists || beforeIndexExists &&
		!fileidentity.SameExecutable(beforeIndex, afterIndex) {
		workingTreeUncertain = true
	}
	finalHeadOutput, finalHeadErr := runner.run(ctx, dir, gitHeadArgs...)
	finalHead, finalHeadSet, err := optionalGitOutput(finalHeadOutput, finalHeadErr, 128)
	if err != nil {
		return nil, err
	}
	if headSet != finalHeadSet || headSet && head != finalHead {
		workingTreeUncertain = true
	}
	finalBranchOutput, finalBranchErr := runner.run(ctx, dir, gitSymbolicHeadArgs...)
	finalBranch, finalBranchSet, err := optionalGitOutput(finalBranchOutput, finalBranchErr, 1)
	if err != nil {
		return nil, err
	}
	if branchSet != finalBranchSet || branchSet && branch != finalBranch {
		workingTreeUncertain = true
	}
	finalUpstream, err := runner.captureUpstreamState(ctx, dir, finalBranchSet && finalHeadSet)
	if err != nil {
		return nil, err
	}
	if !upstream.equal(finalUpstream) {
		workingTreeUncertain = true
	}
	finalState, err := runner.captureStatusState(ctx, dir)
	if err != nil {
		return nil, err
	}
	if !state.equal(finalState) {
		workingTreeUncertain = true
	}
	if workingTreeUncertain {
		writeSynthesizedToken(&synthesized, "# reinstate.working-tree uncertain")
	}
	if synthesized.exceeded {
		return nil, ErrOutputTooLarge
	}
	return append([]byte(nil), synthesized.Bytes()...), nil
}

type capturedUpstreamState struct {
	name   string
	ahead  string
	behind string
	set    bool
}

func (runner resolvedGitRunner) captureUpstreamState(
	ctx context.Context,
	dir string,
	available bool,
) (capturedUpstreamState, error) {
	if !available {
		return capturedUpstreamState{}, nil
	}
	output, runErr := runner.run(ctx, dir, gitUpstreamArgs...)
	upstream, set, err := optionalGitOutput(output, runErr, 1, 128)
	if err != nil || !set {
		return capturedUpstreamState{}, err
	}
	relationOutput, err := runner.run(ctx, dir, gitUpstreamDiffArgs...)
	if err != nil {
		return capturedUpstreamState{}, err
	}
	fields := strings.Fields(string(relationOutput))
	if len(fields) != 2 {
		return capturedUpstreamState{}, ErrInvalidGitStatus
	}
	return capturedUpstreamState{name: upstream, ahead: fields[0], behind: fields[1], set: true}, nil
}

func (state capturedUpstreamState) equal(other capturedUpstreamState) bool {
	return state == other
}

type capturedStatusState struct {
	staged    []byte
	unstaged  []byte
	untracked []byte
	index     []byte
}

func (runner resolvedGitRunner) captureStatusState(ctx context.Context, dir string) (capturedStatusState, error) {
	var result capturedStatusState
	commands := []struct {
		args   []string
		output *[]byte
	}{
		{gitStagedArgs, &result.staged},
		{gitUnstagedArgs, &result.unstaged},
		{gitUntrackedArgs, &result.untracked},
		{gitGitlinksArgs, &result.index},
	}
	for _, command := range commands {
		output, err := runner.run(ctx, dir, command.args...)
		if err != nil {
			return capturedStatusState{}, err
		}
		*command.output = output
	}
	return result, nil
}

func (state capturedStatusState) equal(other capturedStatusState) bool {
	return bytes.Equal(state.staged, other.staged) &&
		bytes.Equal(state.unstaged, other.unstaged) &&
		bytes.Equal(state.untracked, other.untracked) &&
		bytes.Equal(state.index, other.index)
}

func captureIndexSnapshot(ctx context.Context, path string) (fileidentity.Identity, bool, error) {
	identity, err := fileidentity.CaptureExecutable(ctx, path)
	if err == nil {
		return identity, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return fileidentity.Identity{}, false, nil
	}
	return fileidentity.Identity{}, false, err
}

func unsafeConfigNames(raw []byte) bool {
	for _, entry := range bytes.Split(raw, []byte{0}) {
		key := strings.ToLower(strings.TrimSpace(string(entry)))
		switch {
		case key == "include.path", strings.HasPrefix(key, "includeif."),
			key == "core.worktree", key == "core.ignorestat",
			key == "core.attributesfile", strings.HasPrefix(key, "filter."):
			return true
		}
	}
	return false
}

func trackedPathsFromIndex(raw []byte) ([]byte, error) {
	seen := make(map[string]struct{})
	var paths boundedBuffer
	for _, entry := range bytes.Split(raw, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		_, _, _, path, ok := parseIndexEntry(entry)
		if !ok {
			return nil, ErrInvalidGitStatus
		}
		key := string(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		_, _ = paths.Write(path)
		_, _ = paths.Write([]byte{0})
	}
	if paths.exceeded {
		return nil, ErrOutputTooLarge
	}
	return append([]byte(nil), paths.Bytes()...), nil
}

func filterAttributesConfigured(raw []byte) bool {
	tokens := bytes.Split(raw, []byte{0})
	for index := 0; index < len(tokens); {
		if len(tokens[index]) == 0 {
			index++
			continue
		}
		if index+2 >= len(tokens) || string(tokens[index+1]) != "filter" {
			return true
		}
		value := string(tokens[index+2])
		if value != "unspecified" && value != "unset" {
			return true
		}
		index += 3
	}
	return false
}

func optionalGitOutput(output []byte, err error, absentExitCodes ...int) (string, bool, error) {
	if err == nil {
		value := strings.TrimSpace(string(output))
		if value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return "", false, ErrInvalidGitStatus
		}
		return value, true, nil
	}
	exitCode, ok := commandExitCode(err)
	if !ok {
		return "", false, err
	}
	for _, expected := range absentExitCodes {
		if exitCode == expected {
			return "", false, nil
		}
	}
	return "", false, err
}

type rawChange struct {
	path       []byte
	staged     bool
	unstaged   bool
	conflicted bool
	submodule  bool
}

func collectRawChanges(changes map[string]*rawChange, raw []byte, staged bool) error {
	tokens := bytes.Split(raw, []byte{0})
	for index := 0; index < len(tokens); {
		if len(tokens[index]) == 0 {
			index++
			continue
		}
		if index+1 >= len(tokens) || len(tokens[index+1]) == 0 || tokens[index][0] != ':' {
			return ErrInvalidGitStatus
		}
		fields := strings.Fields(string(tokens[index][1:]))
		if len(fields) != 5 || len(fields[4]) != 1 {
			return ErrInvalidGitStatus
		}
		key := string(tokens[index+1])
		change := changes[key]
		if change == nil {
			change = &rawChange{path: append([]byte(nil), tokens[index+1]...)}
			changes[key] = change
		}
		change.conflicted = change.conflicted || fields[4][0] == 'U'
		change.submodule = change.submodule || fields[0] == "160000" || fields[1] == "160000"
		if staged {
			change.staged = true
		} else {
			change.unstaged = true
		}
		index += 2
	}
	return nil
}

func appendRawChanges(output *boundedBuffer, changes map[string]*rawChange) {
	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		change := changes[key]
		if change.conflicted {
			writeSynthesizedTokenBytes(output, []byte("u UU N... "), change.path)
			continue
		}
		xy := []byte{'.', '.'}
		if change.staged {
			xy[0] = 'M'
		}
		if change.unstaged {
			xy[1] = 'M'
		}
		submodule := "N..."
		if change.submodule {
			submodule = "S.M."
		}
		writeSynthesizedTokenBytes(output, []byte("1 "+string(xy)+" "+submodule+" "), change.path)
	}
}

func appendUntracked(output *boundedBuffer, raw []byte) error {
	for _, path := range bytes.Split(raw, []byte{0}) {
		if len(path) == 0 {
			continue
		}
		writeSynthesizedTokenBytes(output, []byte("? "), path)
	}
	return nil
}

func conservativeIndexUncertainty(raw []byte) (bool, error) {
	for _, entry := range bytes.Split(raw, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		tag, mode, stage, path, ok := parseIndexEntry(entry)
		if !ok {
			return false, ErrInvalidGitStatus
		}
		if stage != "0" {
			continue
		}
		_ = path
		if mode == "160000" || tag == 'S' || tag >= 'a' && tag <= 'z' {
			// Never recurse into a repository-controlled submodule process,
			// and never treat hidden index flags as a verifiable clean tree.
			return true, nil
		}
	}
	return false, nil
}

func parseIndexEntry(entry []byte) (tag byte, mode, stage string, path []byte, ok bool) {
	if len(entry) < 3 || entry[1] != ' ' {
		return 0, "", "", nil, false
	}
	tag = entry[0]
	header, path, ok := bytes.Cut(entry[2:], []byte{'\t'})
	fields := strings.Fields(string(header))
	if !ok || len(path) == 0 || len(fields) != 3 || !validObjectID(fields[1]) {
		return 0, "", "", nil, false
	}
	return tag, fields[0], fields[2], path, true
}

func writeSynthesizedToken(output *boundedBuffer, value string) {
	writeSynthesizedTokenBytes(output, []byte(value), nil)
}

func writeSynthesizedTokenBytes(output *boundedBuffer, prefix, value []byte) {
	_, _ = output.Write(prefix)
	_, _ = output.Write(value)
	_, _ = output.Write([]byte{0})
}

func allowedGitProbe(args []string) bool {
	if stringSlicesEqual(args, gitStatusArgs) ||
		stringSlicesEqual(args, gitRootArgs) ||
		stringSlicesEqual(args, []string{"rev-parse", "--is-shallow-repository"}) ||
		stringSlicesEqual(args, []string{"config", "--local", "--no-includes", "--null", "--get-regexp", `^remote\..*\.url$`}) ||
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
		"GIT_EXTERNAL_DIFF": {}, "GIT_DIFF_OPTS": {}, "GIT_ATTR_SOURCE": {},
		"GIT_ATTR_GLOBAL": {}, "GIT_ATTR_SYSTEM": {},
		"GIT_PAGER": {}, "PAGER": {}, "GIT_NO_LAZY_FETCH": {},
	}
	result := make([]string, 0, len(environment)+12)
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
		case "GIT_OPTIONAL_LOCKS", "GIT_TERMINAL_PROMPT", "GIT_NO_REPLACE_OBJECTS",
			"GIT_ATTR_NOSYSTEM", "LC_ALL":
			continue
		}
		result = append(result, entry)
	}
	return append(result,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_PAGER=cat",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_ATTR_GLOBAL="+os.DevNull,
		"GIT_ATTR_SYSTEM="+os.DevNull,
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_KEY_0=core.attributesFile",
		"GIT_CONFIG_VALUE_0="+os.DevNull,
		"GIT_CONFIG_KEY_1=core.fsmonitor",
		"GIT_CONFIG_VALUE_1=false",
		"GIT_CONFIG_KEY_2=diff.external",
		"GIT_CONFIG_VALUE_2=",
		"LC_ALL=C",
	)
}

func safeGitEnvironmentWithPath(environment []string, searchPath string) []string {
	withoutPath := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "PATH") {
			continue
		}
		withoutPath = append(withoutPath, entry)
	}
	return append(safeGitEnvironment(withoutPath), "PATH="+searchPath)
}

func commandExitCode(err error) (int, bool) {
	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		return 0, false
	}
	return commandErr.ExitCode, true
}
