// Package adapter defines the interface every agent adapter implements.
// Concrete adapters (claude, codex, gemini, …) live in subpackages.
package adapter

import (
	"context"
	"io"
)

// SessionMeta is a portable description of one agent session.
type SessionMeta struct {
	ID        string
	Agent     string
	Project   string // canonical project key (not OS path)
	Title     string
	UpdatedAt int64 // unix seconds
	SizeBytes int64
}

// ImportOpts controls restore behavior.
type ImportOpts struct {
	DryRun       bool
	Force        bool
	BackupRoot   string
	PathMap      map[string]string // token → local absolute path
}

// Adapter translates one coding agent's on-disk layout into Reinstate's model.
type Adapter interface {
	// Name returns a stable adapter id (e.g. "claude", "codex").
	Name() string

	// Roots returns local directories this adapter may read.
	Roots() []string

	// Discover lists sessions available on this machine.
	Discover(ctx context.Context) ([]SessionMeta, error)

	// Export writes a normalized session payload for encryption/upload.
	Export(ctx context.Context, id string, w io.Writer) error

	// Import restores a payload into the local agent layout.
	Import(ctx context.Context, id string, r io.Reader, opts ImportOpts) error

	// ProjectKey maps a local absolute project path to a canonical key.
	ProjectKey(localPath string) string

	// Exclude returns globs that must never be synced (credentials, caches).
	Exclude() []string
}

// Registry holds enabled adapters by name.
type Registry map[string]Adapter

// Get returns an adapter by name.
func (r Registry) Get(name string) (Adapter, bool) {
	a, ok := r[name]
	return a, ok
}
