package handoff

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// DefaultMaxArgvBytes is the conservative Windows-safe argv ceiling (24 KiB).
const DefaultMaxArgvBytes = 24 << 10

// Destination Verify() states. Verify never guesses a session ID.
const (
	VerifyResolved   = "resolved"
	VerifyUnresolved = "unresolved"
	VerifyAmbiguous  = "ambiguous"
)

// ErrArgvExceedsBudget means destination argv exceeds TargetCapabilities.MaxArgvBytes.
var ErrArgvExceedsBudget = errors.New("handoff: destination argv exceeds MaxArgvBytes")

// PlannedFile is one destination file declared by Plan and written only during
// materialization (never during dry-run planning).
type PlannedFile struct {
	Path   string
	Mode   os.FileMode
	SHA256 string
}

// TargetCapabilities describes what a destination agent can accept.
type TargetCapabilities struct {
	Agent                 string
	SupportsPinnedID      bool
	SupportsInitialPrompt bool
	MaxArgvBytes          int
	ContextCeiling        int
	AttachmentSupport     bool
}

// DestinationPlan is the exact argv, cwd, and files for a handoff launch.
type DestinationPlan struct {
	Agent      string
	Executable string
	Args       []string
	Dir        string
	Files      []PlannedFile // path + mode + sha256, written only by Execute
	SessionID  string        // empty when the vendor assigns it
	Bootstrap  []byte
}

// HandoffTarget plans, materializes, launches, and verifies a destination session.
type HandoffTarget interface {
	Name() string
	Capabilities() TargetCapabilities
	Compatible(context.Context) (adapter.Compatibility, error)
	Plan(capsule.Capsule, Policy) (DestinationPlan, capsule.Fidelity, error)
	Materialize(context.Context, DestinationPlan) error // writes 0600 files
	Launch(context.Context, DestinationPlan, sessionindex.LaunchRunner) error
	Verify(context.Context, DestinationPlan, time.Time) (string, string, error) // id, state
}

var (
	targetsMu sync.RWMutex
	targets   = map[string]HandoffTarget{}
)

// RegisterTarget adds a destination handoff target. Empty and duplicate names
// are rejected.
func RegisterTarget(t HandoffTarget) error {
	if t == nil {
		return fmt.Errorf("handoff: nil target")
	}
	name := t.Name()
	if name == "" {
		return fmt.Errorf("handoff: empty target name")
	}
	targetsMu.Lock()
	defer targetsMu.Unlock()
	if _, exists := targets[name]; exists {
		return fmt.Errorf("handoff: target %q already registered", name)
	}
	targets[name] = t
	return nil
}

// Target returns a registered handoff target by agent name.
func Target(agent string) (HandoffTarget, bool) {
	targetsMu.RLock()
	defer targetsMu.RUnlock()
	t, ok := targets[agent]
	return t, ok
}

// ArgvBytes returns the summed UTF-8 byte length of executable plus args.
func ArgvBytes(executable string, args []string) int {
	n := len(executable)
	for _, a := range args {
		n += len(a)
	}
	return n
}

// ValidateDestinationArgv fails closed when plan argv exceeds maxArgvBytes.
// A non-positive maxArgvBytes means no limit is enforced by this helper.
func ValidateDestinationArgv(plan DestinationPlan, maxArgvBytes int) error {
	if maxArgvBytes <= 0 {
		return nil
	}
	got := ArgvBytes(plan.Executable, plan.Args)
	if got > maxArgvBytes {
		return fmt.Errorf("%w: %d > %d", ErrArgvExceedsBudget, got, maxArgvBytes)
	}
	return nil
}

// WritePlannedFiles validates argv against maxArgvBytes, then writes each
// planned path from contents as an owner-only file (0600 on Unix, protected
// DACL on Windows). On argv failure no files are written.
func WritePlannedFiles(plan DestinationPlan, maxArgvBytes int, contents map[string][]byte) error {
	if err := ValidateDestinationArgv(plan, maxArgvBytes); err != nil {
		return err
	}
	for _, f := range plan.Files {
		body, ok := contents[f.Path]
		if !ok {
			return fmt.Errorf("handoff: missing content for planned file %q", f.Path)
		}
		if err := fsx.WritePrivateFile(f.Path, body); err != nil {
			return fmt.Errorf("handoff: write planned file %q: %w", f.Path, err)
		}
	}
	return nil
}
