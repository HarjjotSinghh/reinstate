// Package adapter defines agent adapter contracts and the registry.
package adapter

import (
	"context"
	"io"
)

// Compatibility is the adapter install compatibility state.
type Compatibility string

const (
	CompatibilitySupported    Compatibility = "SUPPORTED"
	CompatibilityUntested     Compatibility = "UNTESTED"
	CompatibilityUnsupported  Compatibility = "UNSUPPORTED"
	CompatibilityNotInstalled Compatibility = "NOT_INSTALLED"
)

// Install describes a detected agent installation (no session contents).
type Install struct {
	Agent   string
	Root    string
	Version string
	Layout  string
}

// Session is a discovered local session (metadata only).
type Session struct {
	ID        string
	Agent     string
	ProjectID string
	Title     string
	UpdatedAt int64
	SizeBytes int64
	Path      string
	// RelativePath preserves the vendor-native path below the adapter root.
	// It always uses forward slashes so snapshots are portable across OSes.
	RelativePath string
}

// DiscoverOptions filters discovery.
type DiscoverOptions struct {
	ProjectID string
}

// ExportOptions controls export planning.
type ExportOptions struct {
	DryRun bool
}

// ExportPlan describes files to include in a snapshot.
type ExportPlan struct {
	Session Session
	Files   []string
}

// Snapshot is remote snapshot metadata for restore planning.
type Snapshot struct {
	ID        string
	Agent     string
	SessionID string
	ProjectID string
	// RelativePath is the validated vendor-native path stored in the envelope.
	RelativePath string
}

// RestoreOptions controls restore.
type RestoreOptions struct {
	DryRun          bool
	Force           bool
	CompatibilityOK bool // explicit override for UNTESTED
	BackupRoot      string
	// DestinationRelativePath overrides the vendor destination for keep-both.
	DestinationRelativePath string
	// ForkSessionID rewrites the structural session identity for keep-both.
	ForkSessionID string
}

// RestorePlan describes restore destinations.
type RestorePlan struct {
	Session         Session
	Files           []string
	Refuse          string
	BackupRoot      string
	ArchivePath     string
	SourceSessionID string
}

// Exclusion is a path that must never be synced.
type Exclusion struct {
	Pattern string
	Reason  string
}

// Adapter is the planning-oriented agent adapter contract.
type Adapter interface {
	Name() string
	Detect(context.Context) (Install, Compatibility, error)
	Discover(context.Context, DiscoverOptions) ([]Session, error)
	PlanExport(context.Context, Session, ExportOptions) (ExportPlan, error)
	Export(context.Context, ExportPlan, io.Writer) error
	PlanRestore(context.Context, Snapshot, RestoreOptions) (RestorePlan, error)
	Restore(context.Context, RestorePlan, io.Reader) error
	Exclusions() []Exclusion
}

// CanRestore reports whether restore is allowed for a compatibility state.
func CanRestore(c Compatibility, override bool) bool {
	switch c {
	case CompatibilitySupported:
		return true
	case CompatibilityUntested:
		return override
	default:
		return false
	}
}
