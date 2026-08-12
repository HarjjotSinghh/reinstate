package transcript

import (
	"context"
	"fmt"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Compatibility reports whether a reader supports a session's layout/version.
// Values reuse the docs/compatibility.md vocabulary via adapter.Compatibility.
type Compatibility = adapter.Compatibility

const (
	// CompatibilitySupported means the layout/version has required evidence.
	CompatibilitySupported = adapter.CompatibilitySupported
	// CompatibilityUntested means a recognizable but unverified layout/version.
	CompatibilityUntested = adapter.CompatibilityUntested
	// CompatibilityUnsupported means a known-incompatible layout/version.
	CompatibilityUnsupported = adapter.CompatibilityUnsupported
	// CompatibilityNotInstalled means the agent install was not found.
	CompatibilityNotInstalled = adapter.CompatibilityNotInstalled
)

// Warning is a sanitized, non-fatal parse or probe diagnostic.
// It reuses the sessionindex warning shape so CLI and index paths stay aligned.
type Warning = sessionindex.Warning

// ParseReport summarizes a Parse call without exposing vendor payload bodies.
type ParseReport struct {
	Events          int
	ByKind          map[capsule.Kind]int
	UnknownRecords  int
	MalformedLines  int
	TruncatedBlocks int
	Warnings        []Warning
}

// Reader converts one agent's native session artifact into canonical capsule
// events. Implementations must not use a model, the network, or write the
// source transcript.
type Reader interface {
	// Name returns the stable agent key used for registry lookup (e.g. "claude").
	Name() string
	// Probe reports whether this reader supports the record's layout/version.
	Probe(context.Context, sessionindex.Record) (Compatibility, error)
	// Snapshot freezes an immutable, complete-record boundary. Read-only.
	Snapshot(context.Context, sessionindex.Record) (Boundary, error)
	// Parse converts a boundary into canonical events. No model, no network.
	Parse(context.Context, Boundary) ([]capsule.Event, ParseReport, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Reader{}
)

// Register adds a transcript reader. Empty and duplicate names are rejected.
func Register(r Reader) error {
	if r == nil {
		return fmt.Errorf("transcript: nil reader")
	}
	name := r.Name()
	if name == "" {
		return fmt.Errorf("transcript: empty reader name")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		return fmt.Errorf("transcript: reader %q already registered", name)
	}
	registry[name] = r
	return nil
}

// Get returns a registered reader by agent name.
func Get(agent string) (Reader, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	r, ok := registry[agent]
	return r, ok
}
