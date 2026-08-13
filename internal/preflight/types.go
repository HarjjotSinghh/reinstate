// Package preflight builds and authorizes deterministic verified-resume
// environment reports.
package preflight

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/environment"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

const (
	SchemaVersion          = 1
	DefaultVerifierTimeout = 2 * time.Second
)

// Decision is the launch policy derived from all checks.
type Decision string

const (
	DecisionReady                Decision = "ready"
	DecisionConfirmationRequired Decision = "confirmation_required"
	DecisionBlocked              Decision = "blocked"
)

// Status describes one current observation or comparison.
type Status string

const (
	StatusMatch   Status = "match"
	StatusPresent Status = "present"
	StatusChanged Status = "changed"
	StatusMissing Status = "missing"
	StatusUnknown Status = "unknown"
	StatusError   Status = "error"
)

// Severity controls launch authorization.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityBlock   Severity = "block"
)

// Check is one privacy-safe, stable preflight result. Expected and Actual are
// limited by the verifier to booleans, bounded strings, and string slices.
type Check struct {
	ID         string               `json:"id"`
	Status     Status               `json:"status"`
	Severity   Severity             `json:"severity"`
	Expected   any                  `json:"expected,omitempty"`
	Actual     any                  `json:"actual,omitempty"`
	Provenance workspace.Provenance `json:"provenance"`
	Message    string               `json:"message"`
	Repair     string               `json:"repair,omitempty"`
	ExitCode   int                  `json:"-"`
}

// Report is the exact immutable input to launch policy. It intentionally has
// no generation timestamp so equal observations produce equal JSON.
type Report struct {
	SchemaVersion int                             `json:"schema_version"`
	SessionRef    string                          `json:"session_ref"`
	Decision      Decision                        `json:"decision"`
	BlockExitCode int                             `json:"block_exit_code,omitempty"`
	Checks        []Check                         `json:"checks"`
	Workspace     workspace.Fingerprint           `json:"workspace"`
	Agent         agentcheck.Result               `json:"agent"`
	Capabilities  capability.Inventory            `json:"capabilities"`
	Runtimes      []runtimecheck.Result           `json:"runtimes"`
	Recorded      environment.RecordedEnvironment `json:"recorded,omitzero"`
}

// Input identifies the selected native session and the only historical facts
// the verifier may trust. SourceFresh must describe the selected source scan
// used to resolve this exact record.
type Input struct {
	SessionRef  string
	Agent       string
	Workspace   string
	AgentRoot   string
	Recorded    environment.RecordedEnvironment
	Baseline    *environment.PrelaunchBaseline
	SourceFresh bool
	// ReadOnly is true when the caller will only read the workspace and source
	// transcript (structured handoff). Native resume/fork leave it false so a
	// missing executable still blocks launch.
	ReadOnly bool
}

// Verifier is the single high-level boundary injected into CLI flows.
type Verifier interface {
	Verify(context.Context, Input) (Report, error)
}

// Service is the production verifier configured with bounded probe options.
type Service struct {
	Options Options
}

// DefaultService returns the production local-only verifier configuration.
// Private filesystem roots are inputs only and are excluded from reports.
//
// When CLAUDE_CONFIG_DIR or CODEX_HOME are set, capability discovery uses those
// throwaway roots so isolated acceptance homes do not silently mix with the
// operator's ambient agent trees for session-layout probes. UserHome remains
// the real home for path mapping; ClaudeHome/CodexHome override only those roots.
func DefaultService() Service {
	userHome, _ := os.UserHomeDir()
	managedRoot := ""
	switch runtime.GOOS {
	case "darwin":
		managedRoot = string(filepath.Separator)
	case "windows":
		if volume := filepath.VolumeName(userHome); volume != "" {
			managedRoot = volume + string(filepath.Separator)
		}
	}
	opts := capability.Options{
		GOOS: runtime.GOOS, UserHome: userHome, ManagedRoot: managedRoot,
	}
	if value := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); value != "" {
		opts.ClaudeHome = value
	}
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		opts.CodexHome = value
	}
	return Service{Options: Options{Capability: opts}}
}

// Verify implements Verifier.
func (service Service) Verify(ctx context.Context, input Input) (Report, error) {
	return Verify(ctx, input, service.Options)
}

// Options makes every external boundary injectable. Capability options are
// explicit paths and never serialize through Report.
type Options struct {
	Timeout    time.Duration
	Workspace  workspace.ProbeOptions
	Agent      agentcheck.Options
	Capability capability.Options
	Runtime    runtimecheck.Options
}

// Authorization is the deterministic result of applying invocation-scoped
// acknowledgements to one freshly computed report.
type Authorization struct {
	Allowed  bool
	ExitCode int
	Warnings []string
}
