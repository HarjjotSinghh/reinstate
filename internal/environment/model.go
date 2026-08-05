// Package environment defines bounded, persistence-safe environment metadata.
//
// The types in this package contain observations and vendor-recorded facts, not
// configuration contents. They must never carry credentials, command lines,
// environment-variable values, instruction contents, or raw repository URLs.
package environment

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/safetext"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

const (
	MaxRepositoryIDRunes       = 78
	MaxBranchRunes             = 1024
	MaxGitHeadRunes            = 64
	MaxRequirementNameRunes    = 512
	MaxRequirementVersionRunes = 128
	MaxProvenanceRunes         = 128
	MaxRequirements            = 256
	MaxCapabilities            = 768
	MaxCapabilitiesPerKind     = 256
	MaxRuntimes                = 256
	MaxSessionReferenceRunes   = 4160
)

const (
	// PrelaunchObservedProvenance identifies a baseline observed by Reinstate
	// immediately before a native launch. It is deliberately distinct from
	// vendor-recorded session metadata.
	PrelaunchObservedProvenance = "reinstate_prelaunch_observed"
)

// RecordedField is one bounded vendor-recorded value and its explicit source.
// Empty values must not claim provenance.
type RecordedField struct {
	Value      string `json:"value"`
	Provenance string `json:"provenance"`
}

// Requirement is a non-secret capability identity recorded by a vendor.
// Names and versions are metadata only; command lines and configuration values
// are outside this contract.
type Requirement struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	Provenance string `json:"provenance"`
}

// Capability is one privacy-safe capability observation. It records identity
// and state, never configuration contents, paths, commands, or values.
type Capability struct {
	Agent      string `json:"agent"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	State      string `json:"state"`
	Transport  string `json:"transport,omitempty"`
	Provenance string `json:"provenance"`
}

// Runtime is one recognized runtime observation. SourceKind identifies the
// kind of declaration/probe (for example executable or version_file), not a
// filesystem path.
type Runtime struct {
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	SourceKind string `json:"source_kind"`
	Provenance string `json:"provenance"`
}

// RecordedEnvironment contains only facts explicitly present in recognized
// vendor metadata. Missing facts remain empty instead of being inferred from a
// later inspection of the live workspace.
type RecordedEnvironment struct {
	RepositoryID RecordedField `json:"repository_id,omitzero"`
	Branch       RecordedField `json:"branch,omitzero"`
	GitHead      RecordedField `json:"git_head,omitzero"`
	Requirements []Requirement `json:"requirements,omitempty"`
}

// Empty reports whether no trustworthy vendor-recorded facts are present.
func (r RecordedEnvironment) Empty() bool {
	return r.RepositoryID.Value == "" && r.Branch.Value == "" &&
		r.GitHead.Value == "" && len(r.Requirements) == 0
}

// WorkingTreeState is a privacy-safe summary. File names never enter the
// prelaunch baseline.
type WorkingTreeState string

const (
	WorkingTreeClean       WorkingTreeState = "clean"
	WorkingTreeModified    WorkingTreeState = "modified"
	WorkingTreeUnavailable WorkingTreeState = "unavailable"
)

// PrelaunchBaseline is a private, derived observation recorded immediately
// before native execution. It is independent of the vendor source fingerprint,
// so a successful native append does not invalidate it.
type PrelaunchBaseline struct {
	SessionRef        string           `json:"session_ref"`
	RepositoryID      string           `json:"repository_id,omitempty"`
	Branch            string           `json:"branch,omitempty"`
	GitHead           string           `json:"git_head,omitempty"`
	WorkingTreeDigest string           `json:"working_tree_digest,omitempty"`
	WorkingTreeState  WorkingTreeState `json:"working_tree_state"`
	ObservedAt        time.Time        `json:"observed_at"`
	Provenance        string           `json:"provenance"`
	SourceSessionRef  string           `json:"source_session_ref,omitempty"`
	Capabilities      []Capability     `json:"capabilities,omitempty"`
	Runtimes          []Runtime        `json:"runtimes,omitempty"`
}

// NormalizeRecordedEnvironment bounds and validates persistence-safe recorded
// facts. Raw repository URLs are normalized to a credential-free host/path ID.
func NormalizeRecordedEnvironment(value RecordedEnvironment) (RecordedEnvironment, error) {
	var result RecordedEnvironment
	var err error

	result.RepositoryID, err = normalizeRecordedField(value.RepositoryID, MaxRepositoryIDRunes, normalizeRepositoryField)
	if err != nil {
		return RecordedEnvironment{}, fmt.Errorf("repository identity: %w", err)
	}
	result.Branch, err = normalizeRecordedField(value.Branch, MaxBranchRunes, normalizeMetadata)
	if err != nil {
		return RecordedEnvironment{}, fmt.Errorf("branch: %w", err)
	}
	result.GitHead, err = normalizeRecordedField(value.GitHead, MaxGitHeadRunes, normalizeGitHead)
	if err != nil {
		return RecordedEnvironment{}, fmt.Errorf("git head: %w", err)
	}

	if len(value.Requirements) > MaxRequirements {
		return RecordedEnvironment{}, fmt.Errorf("requirements exceed maximum of %d", MaxRequirements)
	}
	seen := make(map[string]struct{}, len(value.Requirements))
	for _, requirement := range value.Requirements {
		normalized, normalizeErr := normalizeRequirement(requirement)
		if normalizeErr != nil {
			return RecordedEnvironment{}, normalizeErr
		}
		key := normalized.Kind + "\x00" + normalized.Name + "\x00" + normalized.Version + "\x00" + normalized.Provenance
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result.Requirements = append(result.Requirements, normalized)
	}
	sort.Slice(result.Requirements, func(i, j int) bool {
		left, right := result.Requirements[i], result.Requirements[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		return left.Provenance < right.Provenance
	})
	if len(result.Requirements) == 0 {
		result.Requirements = nil
	}
	return result, nil
}

// NormalizePrelaunchBaseline validates one private prelaunch observation.
func NormalizePrelaunchBaseline(value PrelaunchBaseline) (PrelaunchBaseline, error) {
	value.SessionRef = normalizeMetadata(value.SessionRef, MaxSessionReferenceRunes)
	if value.SessionRef == "" {
		return PrelaunchBaseline{}, errors.New("session reference must not be empty")
	}
	value.SourceSessionRef = normalizeMetadata(value.SourceSessionRef, MaxSessionReferenceRunes)
	rawRepositoryID := value.RepositoryID
	value.RepositoryID = NormalizeRepositoryID(value.RepositoryID)
	if strings.TrimSpace(rawRepositoryID) != "" && value.RepositoryID == "" {
		return PrelaunchBaseline{}, errors.New("repository identity is not a safe remote ID")
	}
	value.Branch = normalizeMetadata(value.Branch, MaxBranchRunes)
	rawGitHead := value.GitHead
	value.GitHead = normalizeGitHead(value.GitHead, MaxGitHeadRunes)
	if strings.TrimSpace(rawGitHead) != "" && value.GitHead == "" {
		return PrelaunchBaseline{}, errors.New("git head is not a hexadecimal object ID")
	}
	rawWorkingTreeDigest := value.WorkingTreeDigest
	value.WorkingTreeDigest = normalizeDigest(value.WorkingTreeDigest)
	if strings.TrimSpace(rawWorkingTreeDigest) != "" && value.WorkingTreeDigest == "" {
		return PrelaunchBaseline{}, errors.New("working tree digest is not a SHA-256 value")
	}
	if value.ObservedAt.IsZero() {
		return PrelaunchBaseline{}, errors.New("observed_at must not be zero")
	}
	value.ObservedAt = value.ObservedAt.UTC()
	value.Provenance = normalizeMetadata(value.Provenance, MaxProvenanceRunes)
	if value.Provenance != PrelaunchObservedProvenance {
		return PrelaunchBaseline{}, errors.New("unsupported prelaunch provenance")
	}
	switch value.WorkingTreeState {
	case WorkingTreeClean, WorkingTreeModified, WorkingTreeUnavailable:
	default:
		return PrelaunchBaseline{}, errors.New("unsupported working tree state")
	}
	if value.WorkingTreeState != WorkingTreeUnavailable && value.WorkingTreeDigest == "" {
		return PrelaunchBaseline{}, errors.New("available working tree state requires a digest")
	}
	var err error
	value.Capabilities, err = normalizeCapabilities(value.Capabilities)
	if err != nil {
		return PrelaunchBaseline{}, err
	}
	value.Runtimes, err = normalizeRuntimes(value.Runtimes)
	if err != nil {
		return PrelaunchBaseline{}, err
	}
	return value, nil
}

// NormalizeRepositoryID converts recognized network remotes into the same
// opaque credential-free identity used by live workspace fingerprints. Local
// filesystem remotes and malformed values return an empty identity.
func NormalizeRepositoryID(raw string) string {
	raw = normalizeMetadata(raw, 4096)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "remote-sha256:") || strings.HasPrefix(raw, "roots-sha256:") {
		_, digest, _ := strings.Cut(raw, ":")
		if len(digest) == 64 && isLowerHex(digest) {
			return raw
		}
		return ""
	}
	if !strings.Contains(raw, "://") {
		colon, at := strings.IndexByte(raw, ':'), strings.IndexByte(raw, '@')
		if colon < 1 || at > colon || strings.ContainsAny(raw, "?#") {
			return ""
		}
	}
	identity, err := workspace.RepositoryIDFromRemote(raw)
	if err != nil {
		return ""
	}
	return identity
}

// NormalizeGitHead returns a lower-case hexadecimal Git object identity, or an
// empty string when the vendor value is not a safe abbreviated/full object ID.
func NormalizeGitHead(value string) string {
	return normalizeGitHead(value, MaxGitHeadRunes)
}

func normalizeRecordedField(
	value RecordedField,
	maxRunes int,
	normalize func(string, int) string,
) (RecordedField, error) {
	result := RecordedField{
		Value:      normalize(value.Value, maxRunes),
		Provenance: normalizeMetadata(value.Provenance, MaxProvenanceRunes),
	}
	if result.Value == "" {
		if result.Provenance != "" {
			return RecordedField{}, errors.New("empty value must not claim provenance")
		}
		return RecordedField{}, nil
	}
	if result.Provenance == "" {
		return RecordedField{}, errors.New("recorded value requires provenance")
	}
	return result, nil
}

func normalizeRepositoryField(value string, _ int) string {
	return NormalizeRepositoryID(value)
}

func normalizeRequirement(value Requirement) (Requirement, error) {
	result := Requirement{
		Kind:       strings.ToLower(normalizeMetadata(value.Kind, 64)),
		Name:       normalizeMetadata(value.Name, MaxRequirementNameRunes),
		Version:    normalizeMetadata(value.Version, MaxRequirementVersionRunes),
		Provenance: normalizeMetadata(value.Provenance, MaxProvenanceRunes),
	}
	if result.Kind == "" || result.Name == "" || result.Provenance == "" {
		return Requirement{}, errors.New("environment requirement needs kind, name, and provenance")
	}
	for _, current := range result.Kind {
		if !(current >= 'a' && current <= 'z' || current >= '0' && current <= '9' || current == '_' || current == '-') {
			return Requirement{}, errors.New("invalid environment requirement kind")
		}
	}
	return result, nil
}

func normalizeCapabilities(values []Capability) ([]Capability, error) {
	if len(values) > MaxCapabilities {
		return nil, fmt.Errorf("capabilities exceed maximum of %d", MaxCapabilities)
	}
	result := make([]Capability, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	perKind := make(map[string]int)
	for _, value := range values {
		value.Agent = strings.ToLower(normalizeMetadata(value.Agent, 64))
		value.Kind = strings.ToLower(normalizeMetadata(value.Kind, 64))
		value.Name = normalizeMetadata(value.Name, MaxRequirementNameRunes)
		value.Scope = strings.ToLower(normalizeMetadata(value.Scope, 128))
		value.State = strings.ToLower(normalizeMetadata(value.State, 64))
		value.Transport = strings.ToLower(normalizeMetadata(value.Transport, 16))
		value.Provenance = normalizeMetadata(value.Provenance, MaxProvenanceRunes)
		if value.Agent == "" || value.Kind == "" || value.Name == "" ||
			value.Scope == "" || value.State == "" || value.Provenance == "" {
			return nil, errors.New("capability needs agent, kind, name, scope, state, and provenance")
		}
		for _, token := range []string{value.Agent, value.Kind, value.Scope, value.State} {
			if !safeToken(token) {
				return nil, errors.New("invalid capability identity")
			}
		}
		if value.Transport != "" && value.Transport != "unknown" && value.Transport != "stdio" &&
			value.Transport != "http" && value.Transport != "sse" {
			return nil, errors.New("invalid capability transport")
		}
		if value.Provenance != PrelaunchObservedProvenance {
			return nil, errors.New("unsupported capability provenance")
		}
		key := value.Agent + "\x00" + value.Kind + "\x00" + value.Name + "\x00" +
			value.Scope + "\x00" + value.State + "\x00" + value.Transport + "\x00" + value.Provenance
		if _, exists := seen[key]; exists {
			continue
		}
		group := value.Agent + "\x00" + value.Kind
		perKind[group]++
		if perKind[group] > MaxCapabilitiesPerKind {
			return nil, fmt.Errorf("capability group exceeds maximum of %d", MaxCapabilitiesPerKind)
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		leftKey := left.Agent + "\x00" + left.Kind + "\x00" + left.Name + "\x00" +
			left.Scope + "\x00" + left.State + "\x00" + left.Transport + "\x00" + left.Provenance
		rightKey := right.Agent + "\x00" + right.Kind + "\x00" + right.Name + "\x00" +
			right.Scope + "\x00" + right.State + "\x00" + right.Transport + "\x00" + right.Provenance
		return leftKey < rightKey
	})
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func normalizeRuntimes(values []Runtime) ([]Runtime, error) {
	if len(values) > MaxRuntimes {
		return nil, fmt.Errorf("runtimes exceed maximum of %d", MaxRuntimes)
	}
	result := make([]Runtime, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value.Name = strings.ToLower(normalizeMetadata(value.Name, 128))
		value.Version = normalizeMetadata(value.Version, MaxRequirementVersionRunes)
		value.SourceKind = strings.ToLower(normalizeMetadata(value.SourceKind, 128))
		value.Provenance = normalizeMetadata(value.Provenance, MaxProvenanceRunes)
		if value.Name == "" || value.SourceKind == "" || value.Provenance == "" {
			return nil, errors.New("runtime needs name, source kind, and provenance")
		}
		if !safeToken(value.Name) || !safeToken(value.SourceKind) {
			return nil, errors.New("invalid runtime identity")
		}
		if value.Provenance != PrelaunchObservedProvenance {
			return nil, errors.New("unsupported runtime provenance")
		}
		key := value.Name + "\x00" + value.Version + "\x00" + value.SourceKind + "\x00" + value.Provenance
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		leftKey := left.Name + "\x00" + left.Version + "\x00" + left.SourceKind + "\x00" + left.Provenance
		rightKey := right.Name + "\x00" + right.Version + "\x00" + right.SourceKind + "\x00" + right.Provenance
		return leftKey < rightKey
	})
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func normalizeGitHead(value string, maxRunes int) string {
	value = strings.ToLower(normalizeMetadata(value, maxRunes))
	if value == "" {
		return ""
	}
	if len(value) < 7 || len(value) > MaxGitHeadRunes {
		return ""
	}
	for _, current := range value {
		if !(current >= '0' && current <= '9' || current >= 'a' && current <= 'f') {
			return ""
		}
	}
	return value
}

func normalizeDigest(value string) string {
	value = strings.ToLower(normalizeMetadata(value, 71))
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 64 {
		return ""
	}
	for _, current := range value {
		if !(current >= '0' && current <= '9' || current >= 'a' && current <= 'f') {
			return ""
		}
	}
	return "sha256:" + value
}

func safeToken(value string) bool {
	for _, current := range value {
		if !(current >= 'a' && current <= 'z' || current >= '0' && current <= '9' ||
			current == '_' || current == '-' || current == '.') {
			return false
		}
	}
	return value != ""
}

func isLowerHex(value string) bool {
	for _, current := range value {
		if !(current >= '0' && current <= '9' || current >= 'a' && current <= 'f') {
			return false
		}
	}
	return value != ""
}

func normalizeMetadata(value string, maxRunes int) string {
	return safetext.Text(value, maxRunes)
}
