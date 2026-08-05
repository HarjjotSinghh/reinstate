package preflight

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// Verify performs one local-only, bounded preflight for a selected record.
func Verify(ctx context.Context, input Input, options Options) (Report, error) {
	input.SessionRef = strings.TrimSpace(input.SessionRef)
	input.Agent = strings.ToLower(strings.TrimSpace(input.Agent))
	input.Workspace = strings.TrimSpace(input.Workspace)
	if input.SessionRef == "" || input.Agent == "" {
		return Report{}, errors.New("preflight requires a session reference and agent")
	}
	recorded, err := environment.NormalizeRecordedEnvironment(input.Recorded)
	if err != nil {
		return Report{}, err
	}
	input.Recorded = recorded
	if input.Baseline != nil {
		normalized, normalizeErr := environment.NormalizePrelaunchBaseline(*input.Baseline)
		if normalizeErr != nil {
			return Report{}, normalizeErr
		}
		if normalized.SessionRef != input.SessionRef {
			return Report{}, errors.New("prelaunch baseline belongs to a different session")
		}
		input.Baseline = &normalized
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultVerifierTimeout
	}
	verifyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	expected := workspaceExpectation(input)
	workspaceOptions := options.Workspace
	workspaceOptions.Timeout = remainingTimeout(verifyCtx, workspaceOptions.Timeout)
	workspaceVerification, err := workspace.Verify(verifyCtx, input.Workspace, expected, workspaceOptions)
	if err != nil {
		return Report{}, err
	}

	agentOptions := options.Agent
	if agentOptions.Root == "" {
		agentOptions.Root = input.AgentRoot
	}
	agentOptions.Timeout = remainingTimeout(verifyCtx, agentOptions.Timeout)
	agentResult := agentcheck.Inspect(verifyCtx, input.Agent, agentOptions)

	capabilityOptions := options.Capability
	if input.Agent == "claude" && capabilityOptions.ClaudeHome == "" {
		capabilityOptions.ClaudeHome = input.AgentRoot
	}
	if input.Agent == "codex" && capabilityOptions.CodexHome == "" {
		capabilityOptions.CodexHome = input.AgentRoot
	}
	if capabilityOptions.ProjectRoot == "" {
		capabilityOptions.ProjectRoot = workspaceVerification.Fingerprint.Git.Root
		if capabilityOptions.ProjectRoot == "" {
			capabilityOptions.ProjectRoot = input.Workspace
		}
	}
	if capabilityOptions.WorkingDir == "" {
		capabilityOptions.WorkingDir = input.Workspace
	}
	capabilityInventory := capability.DiscoverContext(verifyCtx, capabilityOptions)
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	capabilityInventory = filterCapabilities(capabilityInventory, input.Agent)

	runtimeOptions := options.Runtime
	runtimeOptions.Timeout = remainingTimeout(verifyCtx, runtimeOptions.Timeout)
	runtimeResults := runtimecheck.Inspect(verifyCtx, input.Workspace, runtimeOptions)

	report := Report{
		SchemaVersion: SchemaVersion,
		SessionRef:    input.SessionRef,
		Workspace:     workspaceVerification.Fingerprint,
		Agent:         agentResult,
		Capabilities:  capabilityInventory,
		Runtimes:      runtimeResults,
		Recorded:      input.Recorded,
	}
	report.Checks = append(report.Checks, sourceFreshnessCheck(input.SourceFresh))
	if input.Baseline == nil {
		report.Checks = append(report.Checks, Check{
			ID: "baseline.unavailable", Status: StatusUnknown, Severity: SeverityWarning,
			Provenance: workspace.ProvenanceUnavailable,
			Message:    "no previous Reinstate prelaunch baseline exists for this session",
			Repair:     "review the current environment before authorizing this first verified launch",
			ExitCode:   exitcode.Safety,
		})
	} else {
		report.Checks = append(report.Checks, Check{
			ID: "baseline.available", Status: StatusPresent, Severity: SeverityInfo,
			Provenance: workspace.ProvenanceReinstatePrelaunchObserved,
			Message:    "a previous successful prelaunch observation is available",
		})
	}
	report.Checks = append(report.Checks, translateWorkspaceChecks(workspaceVerification)...)
	report.Checks = append(report.Checks, agentChecks(agentResult)...)
	report.Checks = append(report.Checks, capabilityChecks(input, capabilityInventory)...)
	report.Checks = append(report.Checks, runtimeChecks(input, runtimeResults)...)
	report.Checks = normalizeChecks(report.Checks)
	report.Decision, report.BlockExitCode = aggregate(report.Checks)
	return report, nil
}

func workspaceExpectation(input Input) workspace.Expectation {
	expected := workspace.Expectation{
		Workspace: &workspace.ExpectedString{Value: input.Workspace, Provenance: workspace.ProvenanceVendorRecorded},
	}
	if input.Baseline != nil {
		baseline := input.Baseline
		if baseline.RepositoryID != "" {
			expected.RepositoryID = &workspace.ExpectedString{Value: baseline.RepositoryID, Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
		}
		if baseline.Branch != "" {
			expected.Branch = &workspace.ExpectedString{Value: baseline.Branch, Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
		}
		if baseline.GitHead != "" {
			expected.Head = &workspace.ExpectedString{Value: baseline.GitHead, Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
		}
		if baseline.WorkingTreeState != environment.WorkingTreeUnavailable {
			dirty := baseline.WorkingTreeState == environment.WorkingTreeModified
			expected.Dirty = &workspace.ExpectedBool{Value: dirty, Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
		}
		if baseline.WorkingTreeDigest != "" {
			expected.WorkingTreeDigest = &workspace.ExpectedString{Value: baseline.WorkingTreeDigest, Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
		}
		return expected
	}
	if input.Recorded.RepositoryID.Value != "" {
		expected.RepositoryID = &workspace.ExpectedString{Value: input.Recorded.RepositoryID.Value, Provenance: workspace.ProvenanceVendorRecorded}
	}
	if input.Recorded.Branch.Value != "" {
		expected.Branch = &workspace.ExpectedString{Value: input.Recorded.Branch.Value, Provenance: workspace.ProvenanceVendorRecorded}
	}
	if input.Recorded.GitHead.Value != "" {
		expected.Head = &workspace.ExpectedString{Value: input.Recorded.GitHead.Value, Provenance: workspace.ProvenanceVendorRecorded}
	}
	return expected
}

func sourceFreshnessCheck(fresh bool) Check {
	if fresh {
		return Check{ID: "source.fresh", Status: StatusMatch, Severity: SeverityInfo,
			Actual: true, Provenance: workspace.ProvenanceCurrentObservation,
			Message: "the selected vendor source was refreshed successfully"}
	}
	return Check{ID: "source.fresh", Status: StatusChanged, Severity: SeverityBlock,
		Actual: false, Provenance: workspace.ProvenanceUnavailable,
		Message:  "the selected vendor source could not be refreshed safely",
		Repair:   "resolve the source scan failure and retry before launching",
		ExitCode: exitcode.Safety}
}

func translateWorkspaceChecks(value workspace.Verification) []Check {
	checks := make([]Check, 0, len(value.Comparison.Checks)+len(value.Diagnostics))
	for _, current := range value.Comparison.Checks {
		check := Check{ID: current.ID, Status: Status(current.Status), Severity: Severity(current.Severity),
			Expected: current.Expected, Actual: current.Actual, Provenance: current.Provenance,
			Message: current.Message, Repair: current.Repair}
		check.ExitCode = workspaceExitCode(check)
		checks = append(checks, check)
	}
	for _, diagnostic := range value.Diagnostics {
		check := Check{ID: diagnostic.ID, Status: Status(diagnostic.Status), Severity: Severity(diagnostic.Severity),
			Provenance: workspace.ProvenanceCurrentObservation, Message: diagnostic.Message}
		check.ExitCode = workspaceExitCode(check)
		checks = append(checks, check)
	}
	return checks
}

func workspaceExitCode(check Check) int {
	if check.Severity == SeverityWarning {
		return exitcode.Safety
	}
	if check.Severity != SeverityBlock {
		return 0
	}
	if check.Status == StatusError || check.ID == "git.probe" || check.ID == "git.status" || check.ID == "git.shallow" {
		return exitcode.Runtime
	}
	if check.ID == "workspace.available" || check.ID == "git.available" {
		return exitcode.Compatibility
	}
	return exitcode.Safety
}

func agentChecks(result agentcheck.Result) []Check {
	present := Check{ID: "agent.executable", Actual: result.ExecutablePresent,
		Provenance: workspace.ProvenanceCurrentObservation}
	if result.ExecutablePresent {
		present.Status, present.Severity, present.Message = StatusPresent, SeverityInfo, "the native agent executable is available"
	} else {
		present.Status, present.Severity, present.Message = StatusMissing, SeverityBlock, "the native agent executable is unavailable"
		present.Repair, present.ExitCode = "install the supported same-vendor executable before continuing", exitcode.Compatibility
	}
	layout := Check{ID: "agent.layout", Actual: result.Layout,
		Provenance: workspace.ProvenanceCurrentObservation}
	if result.LayoutRecognized {
		layout.Status, layout.Severity, layout.Message = StatusMatch, SeverityInfo, "the native agent session layout is recognized"
	} else {
		layout.Status, layout.Severity, layout.Message = StatusMissing, SeverityBlock, "the native agent session layout is unrecognized"
		layout.Repair, layout.ExitCode = "restore a supported same-vendor session layout before continuing", exitcode.Compatibility
	}
	version := Check{ID: "agent.version", Actual: result.Version,
		Provenance: workspace.ProvenanceCurrentObservation}
	if result.Status == agentcheck.StatusSupported {
		version.Status, version.Severity, version.Message = StatusMatch, SeverityInfo, "the native agent version is in the verified range"
	} else {
		version.Status, version.Severity, version.Message = StatusUnknown, SeverityBlock, result.Message
		version.Repair, version.ExitCode = "install a native agent version verified by this Reinstate release", exitcode.Compatibility
		if result.Status == agentcheck.StatusError {
			version.Status, version.ExitCode = StatusError, exitcode.Runtime
		}
	}
	return []Check{present, layout, version}
}

func filterCapabilities(inventory capability.Inventory, agent string) capability.Inventory {
	filtered := capability.Inventory{}
	for _, item := range inventory.Items {
		if string(item.Agent) == agent {
			filtered.Items = append(filtered.Items, item)
		}
	}
	for _, diagnostic := range inventory.Diagnostics {
		if diagnostic.Agent == "" || string(diagnostic.Agent) == agent {
			filtered.Diagnostics = append(filtered.Diagnostics, diagnostic)
		}
	}
	if filtered.Items == nil {
		filtered.Items = []capability.Item{}
	}
	return filtered
}

func capabilityChecks(input Input, inventory capability.Inventory) []Check {
	current := make(map[string]capability.Item)
	for _, item := range inventory.Items {
		if item.VerifiedPresence() {
			current[capabilityIdentity(string(item.Agent), string(item.Kind), item.Name, string(item.Scope))] = item
		}
	}
	var checks []Check
	for _, diagnostic := range inventory.Diagnostics {
		id := capabilityDiagnosticCheckID(diagnostic)
		check := Check{ID: id, Status: StatusUnknown, Severity: SeverityWarning,
			Actual: string(diagnostic.Code), Provenance: workspace.ProvenanceCurrentObservation,
			Message: capabilityDiagnosticMessage(diagnostic),
			Repair:  "review the named capability probe warning before continuing", ExitCode: exitcode.Safety}
		if diagnostic.Code == capability.DiagnosticCancelled {
			check.Status, check.Severity, check.ExitCode = StatusError, SeverityBlock, exitcode.Runtime
			check.Message = "capability discovery exceeded the bounded preflight deadline"
			check.Repair = "retry after reducing local capability inventory pressure"
		}
		checks = append(checks, check)
	}

	expected := make(map[string]environment.Capability)
	if input.Baseline != nil {
		for _, item := range input.Baseline.Capabilities {
			if item.Agent == input.Agent {
				expected[capabilityIdentity(item.Agent, item.Kind, item.Name, item.Scope)] = item
			}
		}
	}
	for _, requirement := range input.Recorded.Requirements {
		present := false
		for _, item := range current {
			if strings.EqualFold(string(item.Kind), requirement.Kind) && strings.EqualFold(item.Name, requirement.Name) {
				present = true
				break
			}
		}
		check := Check{ID: capabilityCheckID(input.Agent, requirement.Kind, requirement.Name, "required"),
			Expected: requirement.Name, Provenance: workspace.ProvenanceVendorRecorded}
		if present {
			check.Status, check.Severity, check.Actual = StatusMatch, SeverityInfo, requirement.Name
			check.Message = fmt.Sprintf("the recorded %s capability %q is present", requirement.Kind, requirement.Name)
		} else {
			check.Status, check.Severity = StatusMissing, SeverityWarning
			check.Message = fmt.Sprintf("the recorded %s capability %q is missing", requirement.Kind, requirement.Name)
			check.Repair, check.ExitCode = "restore the named capability or acknowledge this exact warning", exitcode.Safety
		}
		checks = append(checks, check)
	}

	keys := make([]string, 0, len(expected)+len(current))
	seen := make(map[string]struct{}, len(expected)+len(current))
	for key := range expected {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range current {
		if _, ok := seen[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		prior, hadPrior := expected[key]
		actual, hasActual := current[key]
		name, kind := actual.Name, string(actual.Kind)
		provenance := workspace.ProvenanceCurrentObservation
		if hadPrior {
			name, kind = prior.Name, prior.Kind
			provenance = baselineOrVendorProvenance(prior.Provenance)
		}
		scope := string(actual.Scope)
		if hadPrior {
			scope = prior.Scope
		}
		check := Check{ID: capabilityCheckID(input.Agent, kind, name, scope), Expected: nil,
			Actual: name, Provenance: provenance}
		switch {
		case hadPrior && hasActual:
			currentTransport := string(actual.Transport)
			if prior.Transport != currentTransport {
				check.Status, check.Severity = StatusChanged, SeverityWarning
				check.Expected, check.Actual = prior.Transport, currentTransport
				check.Message = fmt.Sprintf("the %s capability %q transport differs from the previous prelaunch observation", kind, name)
				check.Repair, check.ExitCode = "review the capability transport change or acknowledge this exact warning", exitcode.Safety
			} else {
				check.Status, check.Severity = StatusMatch, SeverityInfo
				check.Message = fmt.Sprintf("the %s capability %q is present", kind, name)
				check.Expected = prior.Name
			}
		case hadPrior:
			check.Status, check.Severity = StatusMissing, SeverityWarning
			check.Message = fmt.Sprintf("the previously observed %s capability %q is missing", kind, name)
			check.Expected = prior.Name
			check.Actual = nil
			check.Repair, check.ExitCode = "restore the named capability or acknowledge this exact warning", exitcode.Safety
		case input.Baseline != nil:
			check.Status, check.Severity = StatusChanged, SeverityWarning
			check.Message = fmt.Sprintf("the %s capability %q is new since the previous prelaunch observation", kind, name)
			check.Repair, check.ExitCode = "review the named capability or acknowledge this exact warning", exitcode.Safety
		default:
			check.Status, check.Severity = StatusPresent, SeverityInfo
			check.Message = fmt.Sprintf("the %s capability %q is currently present", kind, name)
		}
		checks = append(checks, check)
	}
	return checks
}

func capabilityDiagnosticMessage(diagnostic capability.Diagnostic) string {
	kind := safeIDPart(string(diagnostic.Kind))
	scope := safeIDPart(string(diagnostic.Scope))
	if diagnostic.Kind == "" {
		kind = "capability"
	}
	if diagnostic.Scope == "" {
		return fmt.Sprintf("%s discovery was incomplete", kind)
	}
	return fmt.Sprintf("%s %s discovery was incomplete", scope, kind)
}

func runtimeChecks(input Input, results []runtimecheck.Result) []Check {
	baseline := make(map[string]environment.Runtime)
	if input.Baseline != nil {
		for _, value := range input.Baseline.Runtimes {
			baseline[value.Name] = value
		}
	}
	var checks []Check
	seen := make(map[string]struct{})
	for _, result := range results {
		seen[result.Name] = struct{}{}
		check := Check{ID: "runtime." + safeIDPart(result.Name) + ".declaration",
			Expected: result.Declared, Actual: result.Actual,
			Provenance: workspace.ProvenanceCurrentObservation, Message: result.Message}
		switch result.Status {
		case runtimecheck.StatusMatch:
			check.Status, check.Severity = StatusMatch, SeverityInfo
		case runtimecheck.StatusError:
			check.Status, check.Severity, check.ExitCode = StatusError, SeverityBlock, exitcode.Runtime
			check.Repair = "resolve the bounded runtime probe failure before continuing"
		case runtimecheck.StatusChanged:
			check.Status, check.Severity, check.ExitCode = StatusChanged, SeverityWarning, exitcode.Safety
			check.Repair = "use a compatible runtime or acknowledge this exact warning"
		case runtimecheck.StatusMissing:
			check.Status, check.Severity, check.ExitCode = StatusMissing, SeverityWarning, exitcode.Safety
			check.Repair = "install a compatible runtime or acknowledge this exact warning"
		default:
			check.Status, check.Severity, check.ExitCode = StatusUnknown, SeverityWarning, exitcode.Safety
			check.Repair = "review the unrecognized runtime declaration before continuing"
		}
		checks = append(checks, check)
		if prior, ok := baseline[result.Name]; ok {
			baselineCheck := Check{ID: "runtime." + safeIDPart(result.Name) + ".baseline",
				Expected: runtimeIdentity(prior.SourceKind, prior.Version), Actual: runtimeIdentity(result.SourceKind, result.Actual),
				Provenance: workspace.ProvenanceReinstatePrelaunchObserved}
			if prior.Version != "" && prior.Version == result.Actual && prior.SourceKind == result.SourceKind {
				baselineCheck.Status, baselineCheck.Severity, baselineCheck.Message = StatusMatch, SeverityInfo, "the runtime matches the previous prelaunch observation"
			} else {
				baselineCheck.Status, baselineCheck.Severity, baselineCheck.Message = StatusChanged, SeverityWarning, "the runtime differs from the previous prelaunch observation"
				baselineCheck.Repair, baselineCheck.ExitCode = "review the runtime change or acknowledge this exact warning", exitcode.Safety
			}
			checks = append(checks, baselineCheck)
		} else if input.Baseline != nil {
			checks = append(checks, Check{ID: "runtime." + safeIDPart(result.Name) + ".baseline",
				Status: StatusChanged, Severity: SeverityWarning,
				Actual:     runtimeIdentity(result.SourceKind, result.Actual),
				Provenance: workspace.ProvenanceReinstatePrelaunchObserved,
				Message:    "a new runtime declaration is present since the previous prelaunch observation",
				Repair:     "review the runtime change or acknowledge this exact warning", ExitCode: exitcode.Safety})
		}
	}
	for name, prior := range baseline {
		if _, ok := seen[name]; ok {
			continue
		}
		checks = append(checks, Check{ID: "runtime." + safeIDPart(name) + ".baseline",
			Status: StatusMissing, Severity: SeverityWarning, Expected: runtimeIdentity(prior.SourceKind, prior.Version),
			Provenance: workspace.ProvenanceReinstatePrelaunchObserved,
			Message:    "a previously observed runtime declaration is missing",
			Repair:     "restore the runtime declaration or acknowledge this exact warning", ExitCode: exitcode.Safety})
	}
	return checks
}

func runtimeIdentity(sourceKind, version string) string {
	if version == "" {
		return sourceKind
	}
	return sourceKind + "@" + version
}

func normalizeChecks(checks []Check) []Check {
	priority := map[string]int{"source.fresh": 0, "baseline.unavailable": 1, "baseline.available": 1,
		"workspace.available": 2, "git.available": 3, "git.probe": 4, "git.status": 5,
		"git.repository": 6, "git.branch": 7, "git.head": 8, "git.working_tree": 9,
		"agent.executable": 10, "agent.layout": 11, "agent.version": 12}
	sort.SliceStable(checks, func(i, j int) bool {
		left, lok := priority[checks[i].ID]
		right, rok := priority[checks[j].ID]
		if lok || rok {
			if !lok {
				return false
			}
			if !rok {
				return true
			}
			if left != right {
				return left < right
			}
		}
		return checks[i].ID < checks[j].ID
	})
	result := checks[:0]
	for _, check := range checks {
		if len(result) != 0 && result[len(result)-1].ID == check.ID {
			if checkTrustRank(check) > checkTrustRank(result[len(result)-1]) {
				result[len(result)-1] = check
			}
			continue
		}
		result = append(result, check)
	}
	return result
}

func checkTrustRank(check Check) int {
	rank := severityRank(check.Severity) * 100
	if check.Severity == SeverityBlock {
		switch check.ExitCode {
		case exitcode.Runtime:
			rank += 30
		case exitcode.Safety:
			rank += 20
		case exitcode.Compatibility:
			rank += 10
		}
	}
	if check.Status == StatusError {
		rank++
	}
	return rank
}

func aggregate(checks []Check) (Decision, int) {
	decision := DecisionReady
	exit := 0
	for _, check := range checks {
		if check.Severity == SeverityBlock {
			decision = DecisionBlocked
			exit = preferredBlockExit(exit, check.ExitCode)
		} else if check.Severity == SeverityWarning && decision == DecisionReady {
			decision = DecisionConfirmationRequired
		}
	}
	return decision, exit
}

func preferredBlockExit(current, candidate int) int {
	if candidate == 0 {
		candidate = exitcode.Runtime
	}
	// Infrastructure failure takes precedence because the verifier could not
	// produce trustworthy evidence. A known safety mismatch then takes
	// precedence over an independently observed compatibility blocker.
	rank := map[int]int{exitcode.Compatibility: 1, exitcode.Safety: 2, exitcode.Runtime: 3}
	if current == 0 || rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func severityRank(value Severity) int {
	if value == SeverityBlock {
		return 3
	}
	if value == SeverityWarning {
		return 2
	}
	return 1
}

func capabilityIdentity(agent, kind, name, scope string) string {
	return strings.ToLower(strings.TrimSpace(agent)) + "\x00" + strings.ToLower(strings.TrimSpace(kind)) + "\x00" + strings.ToLower(strings.TrimSpace(name)) + "\x00" + strings.ToLower(strings.TrimSpace(scope))
}

func capabilityCheckID(agent, kind, name, scope string) string {
	digest := sha256.Sum256([]byte(capabilityIdentity(agent, kind, name, scope)))
	return "capability." + safeIDPart(kind) + "." + hex.EncodeToString(digest[:8])
}

func capabilityDiagnosticCheckID(diagnostic capability.Diagnostic) string {
	tuple := strings.Join([]string{
		string(diagnostic.Agent), string(diagnostic.Kind), string(diagnostic.Scope), string(diagnostic.Code),
	}, "\x00")
	digest := sha256.Sum256([]byte(tuple))
	return "capability.probe." + safeIDPart(string(diagnostic.Code)) + "." + hex.EncodeToString(digest[:8])
}

func safeIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var result strings.Builder
	for _, current := range value {
		if current >= 'a' && current <= 'z' || current >= '0' && current <= '9' || current == '_' || current == '-' {
			result.WriteRune(current)
		}
	}
	if result.Len() == 0 {
		return "unknown"
	}
	return result.String()
}

func baselineOrVendorProvenance(value string) workspace.Provenance {
	if value == environment.PrelaunchObservedProvenance {
		return workspace.ProvenanceReinstatePrelaunchObserved
	}
	return workspace.ProvenanceVendorRecorded
}

func remainingTimeout(ctx context.Context, requested time.Duration) time.Duration {
	remaining := DefaultVerifierTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining = time.Until(deadline)
	}
	if requested > 0 && requested < remaining {
		return requested
	}
	if remaining <= 0 {
		return time.Nanosecond
	}
	return remaining
}
