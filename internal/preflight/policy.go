package preflight

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/safetext"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// Authorize validates exact, invocation-scoped warning acknowledgements.
// Duplicate, unknown, wildcard, informational, blocker, and stale IDs fail.
func Authorize(report Report, allowed []string) (Authorization, error) {
	if err := validateReport(report); err != nil {
		return Authorization{ExitCode: exitcode.Runtime}, err
	}
	if report.Decision == DecisionBlocked {
		code := report.BlockExitCode
		if code == 0 {
			code = exitcode.Runtime
		}
		return Authorization{ExitCode: code}, errors.New("environment preflight is blocked")
	}
	warnings := make(map[string]struct{})
	for _, check := range report.Checks {
		if check.Severity == SeverityWarning {
			warnings[check.ID] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(allowed))
	for _, raw := range allowed {
		id := strings.TrimSpace(raw)
		if id == "" || strings.ContainsAny(id, "*?[]") {
			return Authorization{ExitCode: exitcode.Usage}, fmt.Errorf("invalid environment warning ID %q", id)
		}
		if _, duplicate := seen[id]; duplicate {
			return Authorization{ExitCode: exitcode.Usage}, fmt.Errorf("duplicate environment warning ID %q", id)
		}
		seen[id] = struct{}{}
		if _, exists := warnings[id]; !exists {
			return Authorization{ExitCode: exitcode.Usage}, fmt.Errorf("environment warning ID %q is not a current warning", id)
		}
	}
	missing := make([]string, 0, len(warnings))
	for id := range warnings {
		if _, ok := seen[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		return Authorization{ExitCode: exitcode.Safety, Warnings: missing}, fmt.Errorf("environment warnings require confirmation: %s", strings.Join(missing, ", "))
	}
	return Authorization{Allowed: true}, nil
}

// BaselineFromReport builds the private snapshot that may be persisted only
// after the corresponding native child exits successfully.
func BaselineFromReport(report Report, observedAt time.Time) (environment.PrelaunchBaseline, error) {
	if err := validateReport(report); err != nil {
		return environment.PrelaunchBaseline{}, err
	}
	if report.Decision == DecisionBlocked {
		return environment.PrelaunchBaseline{}, errors.New("blocked report cannot become a baseline")
	}
	baseline := environment.PrelaunchBaseline{
		SessionRef:        report.SessionRef,
		RepositoryID:      report.Workspace.Git.RepositoryID,
		Branch:            report.Workspace.Git.Branch,
		GitHead:           report.Workspace.Git.Head,
		WorkingTreeDigest: report.Workspace.Git.WorkingTree.Digest,
		WorkingTreeState:  environment.WorkingTreeState(report.Workspace.Git.WorkingTree.State),
		ObservedAt:        observedAt,
		Provenance:        environment.PrelaunchObservedProvenance,
		SourceSessionRef:  report.SessionRef,
	}
	for _, item := range report.Capabilities.Items {
		if !item.VerifiedPresence() {
			continue
		}
		baseline.Capabilities = append(baseline.Capabilities, environment.Capability{
			Agent: string(item.Agent), Kind: string(item.Kind), Name: item.Name,
			Scope: string(item.Scope), State: string(item.State), Transport: string(item.Transport),
			Provenance: environment.PrelaunchObservedProvenance,
		})
	}
	for _, result := range report.Runtimes {
		if result.Actual == "" {
			continue
		}
		baseline.Runtimes = append(baseline.Runtimes, environment.Runtime{
			Name: result.Name, Declared: result.Declared, Version: result.Actual, SourceKind: result.SourceKind,
			Provenance: environment.PrelaunchObservedProvenance,
		})
	}
	return environment.NormalizePrelaunchBaseline(baseline)
}

func validateReport(report Report) error {
	if report.SchemaVersion != SchemaVersion || strings.TrimSpace(report.SessionRef) != report.SessionRef ||
		!validReportText(report.SessionRef, environment.MaxSessionReferenceRunes) || report.SessionRef == "" ||
		len(report.Checks) == 0 || len(report.Checks) > 2048 {
		return errors.New("environment report contract is invalid")
	}
	seen := make(map[string]struct{}, len(report.Checks))
	for _, check := range report.Checks {
		if !validCheckID(check.ID) {
			return errors.New("environment report contains an invalid check ID")
		}
		if _, duplicate := seen[check.ID]; duplicate {
			return errors.New("environment report contains duplicate check IDs")
		}
		seen[check.ID] = struct{}{}
		if !validStatus(check.Status) || !validSeverity(check.Severity) || !validProvenance(check.Provenance) {
			return errors.New("environment report contains an invalid check")
		}
		if !validCheckExit(check) {
			return errors.New("environment report contains an inconsistent check exit")
		}
		if !validReportText(check.Message, 1024) || !validReportText(check.Repair, 1024) ||
			!validReportValue(check.Expected) || !validReportValue(check.Actual) {
			return errors.New("environment report contains unsafe check data")
		}
	}
	normalized := normalizeChecks(append([]Check(nil), report.Checks...))
	if !reflect.DeepEqual(normalized, report.Checks) {
		return errors.New("environment report check order is invalid")
	}
	decision, code := aggregate(report.Checks)
	if decision != report.Decision || code != report.BlockExitCode {
		return errors.New("environment report decision is inconsistent with its checks")
	}
	return nil
}

func validCheckExit(check Check) bool {
	switch check.Severity {
	case SeverityInfo:
		return check.ExitCode == 0
	case SeverityWarning:
		return check.ExitCode == exitcode.Safety
	case SeverityBlock:
		if check.ExitCode != exitcode.Runtime && check.ExitCode != exitcode.Compatibility && check.ExitCode != exitcode.Safety {
			return false
		}
		return check.Status != StatusError || check.ExitCode == exitcode.Runtime
	default:
		return false
	}
}

// ValidateReport verifies that a report is canonical, internally consistent,
// bounded, and safe to render before any policy or CLI surface trusts it.
func ValidateReport(report Report) error {
	return validateReport(report)
}

func validReportText(value string, maximum int) bool {
	return utf8.RuneCountInString(value) <= maximum && safetext.Text(value, 0) == value
}

func validReportValue(value any) bool {
	switch current := value.(type) {
	case nil, bool:
		return true
	case string:
		return validReportText(current, 4096)
	case workspace.WorkingTreeState:
		return validReportText(string(current), 64)
	case []string:
		if len(current) > 512 {
			return false
		}
		for _, item := range current {
			if !validReportText(item, 4096) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validCheckID(value string) bool {
	if value == "" || len(value) > 256 || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return false
	}
	for _, current := range value {
		if (current < 'a' || current > 'z') && (current < '0' || current > '9') && current != '.' && current != '_' && current != '-' {
			return false
		}
	}
	return true
}

func validStatus(value Status) bool {
	switch value {
	case StatusMatch, StatusPresent, StatusChanged, StatusMissing, StatusUnknown, StatusError:
		return true
	default:
		return false
	}
}

func validSeverity(value Severity) bool {
	return value == SeverityInfo || value == SeverityWarning || value == SeverityBlock
}

func validProvenance(value workspace.Provenance) bool {
	switch value {
	case workspace.ProvenanceVendorRecorded, workspace.ProvenanceReinstateCheckpoint,
		workspace.ProvenanceReinstatePrelaunchObserved, workspace.ProvenanceCurrentObservation,
		workspace.ProvenanceUnavailable:
		return true
	default:
		return false
	}
}

// WarningIDs returns the exact sorted IDs acceptable to Authorize.
func WarningIDs(report Report) []string {
	var result []string
	for _, check := range report.Checks {
		if check.Severity == SeverityWarning {
			result = append(result, check.ID)
		}
	}
	sort.Strings(result)
	return result
}
