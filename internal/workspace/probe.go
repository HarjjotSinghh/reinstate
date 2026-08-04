package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var gitStatusArgs = []string{
	"-c", "core.fsmonitor=false",
	"-c", "core.untrackedCache=false",
	"-c", "submodule.recurse=false",
	"status", "--porcelain=v2", "--branch", "-z", "--untracked-files=normal",
}

// Probe observes the current workspace through a single bounded, local-only
// deadline. It never fetches or contacts a remote.
func Probe(ctx context.Context, workspace string, options ProbeOptions) (ProbeResult, error) {
	probeContext, cancel := sharedProbeContext(ctx, options.Timeout)
	defer cancel()
	return probe(probeContext, workspace, selectedRunner(options))
}

// Verify probes and compares within one shared deadline. Expected values must
// originate in vendor metadata or an explicit Reinstate checkpoint.
func Verify(
	ctx context.Context,
	workspace string,
	expected Expectation,
	options ProbeOptions,
) (Verification, error) {
	probeContext, cancel := sharedProbeContext(ctx, options.Timeout)
	defer cancel()
	runner := selectedRunner(options)
	observed, err := probe(probeContext, workspace, runner)
	if err != nil {
		return Verification{}, err
	}
	if expected.Head != nil && trustedProvenance(expected.Head.Provenance) {
		observed.Fingerprint.Git.ExpectedHeadRelation = resolveHeadRelation(
			probeContext,
			runner,
			observed.Fingerprint.Git,
			expected.Head.Value,
		)
	}
	comparison := Compare(expected, observed.Fingerprint)
	comparison.Decision = aggregateDecision(comparison.Checks, observed.Diagnostics)
	return Verification{
		Fingerprint: observed.Fingerprint,
		Comparison:  comparison,
		Diagnostics: observed.Diagnostics,
	}, nil
}

func sharedProbeContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultProbeTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func selectedRunner(options ProbeOptions) GitRunner {
	if options.Runner != nil {
		return options.Runner
	}
	return ExecGitRunner{}
}

func probe(ctx context.Context, workspace string, runner GitRunner) (ProbeResult, error) {
	result := ProbeResult{Fingerprint: Fingerprint{
		SchemaVersion: SchemaVersion,
		Provenance:    ProvenanceCurrentObservation,
		Workspace:     WorkspaceFingerprint{Path: safeMetadata(workspace, 4096)},
		Git: GitFingerprint{
			WorkingTree:          WorkingTreeFingerprint{State: WorkingTreeUnavailable},
			UpstreamRelation:     CommitRelation{Relation: RelationUnknown, LocalOnly: true},
			ExpectedHeadRelation: CommitRelation{Relation: RelationUnknown, LocalOnly: true},
		},
	}}

	info, err := os.Stat(workspace)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				ID: "workspace.available", Status: StatusMissing, Severity: SeverityBlock,
				Message: "the recorded workspace is missing",
			})
			return result, nil
		}
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "workspace.available", Status: StatusError, Severity: SeverityBlock,
			Message: "the recorded workspace could not be inspected",
		})
		return result, nil
	}
	result.Fingerprint.Workspace.Exists = true
	if !info.IsDir() {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "workspace.available", Status: StatusMissing, Severity: SeverityBlock,
			Message: "the recorded workspace is not a directory",
		})
		return result, nil
	}
	result.Fingerprint.Workspace.Directory = true
	if absolute, absErr := filepath.Abs(workspace); absErr == nil {
		workspace = absolute
	}
	if physical, evalErr := filepath.EvalSymlinks(workspace); evalErr == nil {
		workspace = physical
	}
	workspace = filepath.Clean(workspace)
	result.Fingerprint.Workspace.Path = safeMetadata(workspace, 4096)

	rootOutput, runErr := runner.Run(ctx, workspace,
		"rev-parse", "--path-format=absolute", "--show-toplevel",
	)
	if runErr != nil {
		return classifyInitialGitFailure(ctx, result, runErr)
	}
	root := strings.TrimSpace(string(rootOutput))
	if root == "" {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "git.probe", Status: StatusError, Severity: SeverityBlock,
			Message: "Git returned an empty repository root",
		})
		return result, nil
	}
	if absolute, absErr := filepath.Abs(root); absErr == nil {
		root = absolute
	}
	root = filepath.Clean(root)
	result.Fingerprint.Git.Available = true
	result.Fingerprint.Git.Repository = true
	result.Fingerprint.Git.Root = safeMetadata(root, 4096)
	result.Fingerprint.Git.rootPath = root

	statusOutput, runErr := runner.Run(ctx, root, gitStatusArgs...)
	if runErr != nil {
		appendGitDiagnostic(&result, ctx, "git.status", runErr)
		return result, nil
	}
	status, parseErr := parsePorcelainV2(statusOutput)
	if parseErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "git.status", Status: StatusError, Severity: SeverityBlock,
			Message: "Git returned malformed status metadata",
		})
		return result, nil
	}
	result.Fingerprint.Git.Branch = status.branch
	result.Fingerprint.Git.Detached = status.detached
	result.Fingerprint.Git.Unborn = status.unborn
	result.Fingerprint.Git.Head = status.head
	result.Fingerprint.Git.WorkingTree = status.workingTree
	result.Fingerprint.Git.Upstream = status.upstream
	result.Fingerprint.Git.UpstreamRelation = status.relation

	shallowOutput, runErr := runner.Run(ctx, root, "rev-parse", "--is-shallow-repository")
	if runErr != nil {
		appendGitDiagnostic(&result, ctx, "git.shallow", runErr)
		return result, nil
	}
	shallow, parseBoolErr := strconv.ParseBool(strings.TrimSpace(string(shallowOutput)))
	if parseBoolErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "git.shallow", Status: StatusError, Severity: SeverityBlock,
			Message: "Git returned malformed shallow-repository metadata",
		})
		return result, nil
	}
	result.Fingerprint.Git.Shallow = shallow

	remoteOutput, remoteErr := runner.Run(ctx, root,
		"config", "--null", "--get-regexp", `^remote\..*\.url$`,
	)
	if remoteErr == nil {
		primary, identities := selectRemoteIdentity(parseRemoteIdentities(remoteOutput))
		result.Fingerprint.Git.RepositoryID = primary
		result.Fingerprint.Git.repositoryIDs = identities
		if primary != "" {
			result.Fingerprint.Git.RepositoryIDSource = "remote"
		}
	} else if exit, ok := commandExitCode(remoteErr); !ok || exit != 1 {
		appendGitDiagnostic(&result, ctx, "git.repository_identity", remoteErr)
	}

	if result.Fingerprint.Git.RepositoryID == "" && !shallow && status.head != "" {
		rootsOutput, rootsErr := runner.Run(ctx, root, "rev-list", "--max-parents=0", "HEAD")
		if rootsErr == nil {
			roots := validObjectIDs(strings.Fields(string(rootsOutput)))
			if len(roots) > 0 {
				result.Fingerprint.Git.RepositoryID = repositoryIDFromRoots(roots)
				result.Fingerprint.Git.RepositoryIDSource = "root_commits"
				result.Fingerprint.Git.repositoryIDs = []string{result.Fingerprint.Git.RepositoryID}
			}
		} else {
			appendGitDiagnostic(&result, ctx, "git.repository_identity", rootsErr)
		}
	}
	return result, nil
}

func classifyInitialGitFailure(ctx context.Context, result ProbeResult, err error) (ProbeResult, error) {
	if errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ProbeResult{}, context.Canceled
	}
	if errors.Is(err, ErrGitUnavailable) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "git.available", Status: StatusMissing, Severity: SeverityBlock,
			Message: "Git is unavailable for workspace verification",
		})
		return result, nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			ID: "git.probe", Status: StatusError, Severity: SeverityBlock,
			Message: "the bounded Git probe timed out",
		})
		return result, nil
	}
	if errors.Is(err, ErrNotRepository) {
		result.Fingerprint.Git.Available = true
		result.Fingerprint.Git.Repository = false
		return result, nil
	}
	result.Diagnostics = append(result.Diagnostics, Diagnostic{
		ID: "git.probe", Status: StatusError, Severity: SeverityBlock,
		Message: "the bounded Git probe failed",
	})
	return result, nil
}

func appendGitDiagnostic(result *ProbeResult, ctx context.Context, id string, err error) {
	message := "a bounded Git probe failed"
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		message = "the bounded Git probe timed out"
	} else if errors.Is(err, ErrOutputTooLarge) {
		message = "Git probe output exceeded the safe limit"
	} else if errors.Is(err, ErrGitUnavailable) {
		message = "Git became unavailable during workspace verification"
	}
	result.Diagnostics = append(result.Diagnostics, Diagnostic{
		ID: id, Status: StatusError, Severity: SeverityBlock, Message: message,
	})
}

func resolveHeadRelation(ctx context.Context, runner GitRunner, git GitFingerprint, expected string) CommitRelation {
	unknown := CommitRelation{Relation: RelationUnknown, LocalOnly: true}
	expected = strings.ToLower(strings.TrimSpace(expected))
	if !validObjectID(expected) || !validObjectID(git.Head) || !git.Repository || git.Shallow {
		return unknown
	}
	if expected == git.Head {
		return relationFromCounts(0, 0)
	}
	root := git.rootPath
	if root == "" {
		root = git.Root
	}
	output, err := runner.Run(ctx, root, "rev-list", "--left-right", "--count", expected+"..."+git.Head)
	if err != nil {
		return unknown
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return unknown
	}
	expectedOnly, leftErr := strconv.Atoi(fields[0])
	currentOnly, rightErr := strconv.Atoi(fields[1])
	if leftErr != nil || rightErr != nil || expectedOnly < 0 || currentOnly < 0 {
		return unknown
	}
	return relationFromCounts(currentOnly, expectedOnly)
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, current := range value {
		if !(current >= '0' && current <= '9') && !(current >= 'a' && current <= 'f') && !(current >= 'A' && current <= 'F') {
			return false
		}
	}
	return true
}

func validObjectIDs(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(value)
		if validObjectID(value) {
			unique[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func safeMetadata(value string, limit int) string {
	var result strings.Builder
	runes := 0
	for _, current := range strings.ToValidUTF8(value, "") {
		if unicode.IsControl(current) || unicode.In(current, unicode.Cf) {
			continue
		}
		if limit > 0 && runes >= limit {
			break
		}
		result.WriteRune(current)
		runes++
	}
	return result.String()
}
