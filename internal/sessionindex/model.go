// Package sessionindex provides Reinstate's private, derived local session index.
package sessionindex

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

const (
	// SchemaVersion is the on-disk session-index schema version.
	SchemaVersion = 2

	// DefaultLimit is used when a query does not specify a result limit.
	DefaultLimit = 100
	// MaxLimit bounds one query so accidental broad searches remain cheap.
	MaxLimit = 1000
)

const (
	AgentClaude   = "claude"
	AgentCodex    = "codex"
	AgentGemini   = "gemini"
	AgentOpenCode = "opencode"
)

var (
	// ErrNotFound means a session reference did not resolve.
	ErrNotFound = errors.New("session not found")
	// ErrAmbiguous means a bare native session ID matched multiple agents.
	ErrAmbiguous = errors.New("ambiguous session reference")
)

// Record is the canonical metadata and bounded searchable content for one local
// vendor session. Fields tagged with json:"-" are private index bookkeeping and
// must not be emitted by ordinary CLI JSON output.
type Record struct {
	Key            string    `json:"key"`
	ID             string    `json:"id"`
	Agent          string    `json:"agent"`
	Title          string    `json:"title,omitempty"`
	Project        string    `json:"project,omitempty"`
	Workspace      string    `json:"workspace,omitempty"`
	Branch         string    `json:"branch,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
	SizeBytes      int64     `json:"size_bytes"`
	MessageCount   int       `json:"message_count"`
	PromptPreview  string    `json:"prompt_preview,omitempty"`
	Files          []string  `json:"files,omitempty"`
	CanResume      bool      `json:"can_resume"`
	CanFork        bool      `json:"can_fork"`
	ReadOnlyReason string    `json:"read_only_reason,omitempty"`
	// RecordedEnvironment contains only bounded facts explicitly present in
	// recognized vendor metadata. Live workspace observations are stored
	// separately and never inferred during indexing.
	RecordedEnvironment environment.RecordedEnvironment `json:"recorded_environment,omitempty"`

	SourcePath    string `json:"-"`
	SourceModTime int64  `json:"-"`
	SourceSize    int64  `json:"-"`
	SearchText    string `json:"-"`
}

// Capabilities describes the native actions Reinstate can safely offer.
type Capabilities struct {
	Resume bool `json:"resume"`
	Fork   bool `json:"fork"`
}

// Capabilities returns the record's native execution support.
func (r Record) Capabilities() Capabilities {
	return Capabilities{Resume: r.CanResume, Fork: r.CanFork}
}

// Reference returns the stable, agent-qualified reference for a record.
func (r Record) Reference() string {
	if r.Key != "" {
		return r.Key
	}
	return CompositeReference(r.Agent, r.ID)
}

// CompositeReference constructs an unambiguous session reference.
func CompositeReference(agent, nativeID string) string {
	return strings.TrimSpace(agent) + ":" + strings.TrimSpace(nativeID)
}

// ParseCompositeReference splits an agent-qualified reference. Native IDs may
// contain colons, so only the first colon is structural.
func ParseCompositeReference(ref string) (agent, nativeID string, ok bool) {
	ref = strings.TrimSpace(ref)
	index := strings.IndexByte(ref, ':')
	if index <= 0 || index == len(ref)-1 {
		return "", "", false
	}
	agent = strings.TrimSpace(ref[:index])
	nativeID = strings.TrimSpace(ref[index+1:])
	if agent == "" || nativeID == "" {
		return "", "", false
	}
	return agent, nativeID, true
}

// Filter describes literal, case-insensitive index filters.
type Filter struct {
	Query         string `json:"query,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Project       string `json:"project,omitempty"`
	Branch        string `json:"branch,omitempty"`
	File          string `json:"file,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	ResumableOnly bool   `json:"resumable_only,omitempty"`
}

// EffectiveLimit returns a safe query limit.
func (f Filter) EffectiveLimit() int {
	switch {
	case f.Limit <= 0:
		return DefaultLimit
	case f.Limit > MaxLimit:
		return MaxLimit
	default:
		return f.Limit
	}
}

// Warning is a sanitized, non-fatal source or extraction diagnostic.
type Warning struct {
	Agent     string `json:"agent,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Source    string `json:"source,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

// ScanResult is the complete result of one successful source scan. Records
// omitted from a successful scan are removed from that source's derived index.
// A failed Scan call never causes deletion.
type ScanResult struct {
	Records  []Record  `json:"records"`
	Warnings []Warning `json:"warnings,omitempty"`
}

// AmbiguousReferenceError reports every qualified match for a bare native ID.
type AmbiguousReferenceError struct {
	Reference string
	Matches   []string
}

func (e *AmbiguousReferenceError) Error() string {
	return fmt.Sprintf(
		"session reference %q is ambiguous; use one of: %s",
		e.Reference,
		strings.Join(e.Matches, ", "),
	)
}

// Is lets errors.Is(err, ErrAmbiguous) distinguish ambiguity from generic
// runtime failures while preserving a specific actionable error type.
func (e *AmbiguousReferenceError) Is(target error) bool {
	return target == ErrAmbiguous
}

// SortRecords applies the Phase 2 deterministic ordering contract.
func SortRecords(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		left, right := records[i], records[j]
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		if left.Agent != right.Agent {
			return left.Agent < right.Agent
		}
		return left.ID < right.ID
	})
}
