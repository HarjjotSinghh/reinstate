package sessionindex

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Source discovers and parses one agent's complete local read surface.
// A successful Scan is authoritative for deletion; an error leaves that
// source's previously indexed records untouched.
type Source interface {
	Name() string
	Scan(context.Context) (ScanResult, error)
}

// SourceRefresh is one source's refresh outcome.
type SourceRefresh struct {
	Name      string `json:"name"`
	Records   int    `json:"records"`
	Upserted  int    `json:"upserted"`
	Unchanged int    `json:"unchanged"`
	Deleted   int    `json:"deleted"`
	Error     string `json:"error,omitempty"`
}

// RefreshResult is a deterministic aggregate refresh report.
type RefreshResult struct {
	Sources  []SourceRefresh `json:"sources"`
	Warnings []Warning       `json:"warnings,omitempty"`
}

// Failed reports whether any source scan failed. Other sources may still have
// refreshed successfully.
func (r RefreshResult) Failed() bool {
	for _, source := range r.Sources {
		if source.Error != "" {
			return true
		}
	}
	return false
}

// Index orchestrates sources and the private derived store.
type Index struct {
	store   *Store
	sources []Source
}

// NewIndex constructs an index over an already-open store.
func NewIndex(store *Store, sources ...Source) (*Index, error) {
	if store == nil {
		return nil, errors.New("session-index store must not be nil")
	}
	ordered := append([]Source(nil), sources...)
	for _, source := range ordered {
		if source == nil {
			return nil, errors.New("session-index source must not be nil")
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return strings.ToLower(ordered[i].Name()) < strings.ToLower(ordered[j].Name())
	})
	seen := make(map[string]struct{}, len(ordered))
	for _, source := range ordered {
		name := strings.ToLower(SafeText(strings.TrimSpace(source.Name()), 64))
		if name == "" {
			return nil, errors.New("session-index source name must not be empty")
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("duplicate session-index source %q", name)
		}
		seen[name] = struct{}{}
	}
	return &Index{store: store, sources: ordered}, nil
}

// OpenIndex opens the default private store and constructs an index.
func OpenIndex(reinstateHome string, sources ...Source) (*Index, error) {
	store, err := OpenDefault(reinstateHome)
	if err != nil {
		return nil, err
	}
	index, err := NewIndex(store, sources...)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return index, nil
}

// Close closes the underlying store.
func (i *Index) Close() error {
	if i == nil || i.store == nil {
		return nil
	}
	return i.store.Close()
}

// Store returns the underlying derived store for advanced local callers.
func (i *Index) Store() *Store {
	if i == nil {
		return nil
	}
	return i.store
}

// Refresh scans every source. An individual source failure becomes a warning
// and leaves that source's old rows intact; context and SQLite failures remain
// fatal because freshness cannot be represented honestly.
func (i *Index) Refresh(ctx context.Context) (RefreshResult, error) {
	var result RefreshResult
	if i == nil || i.store == nil {
		return result, errors.New("session index is nil")
	}
	for _, source := range i.sources {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		name := strings.ToLower(SafeText(strings.TrimSpace(source.Name()), 64))
		status := SourceRefresh{Name: name}
		scan, err := source.Scan(ctx)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return result, ctxErr
			}
			status.Error = SafeText(err.Error(), 512)
			result.Sources = append(result.Sources, status)
			result.Warnings = append(result.Warnings, Warning{
				Agent:   name,
				Source:  name,
				Code:    "source_scan_failed",
				Message: status.Error,
			})
			continue
		}
		var coalesceWarnings []Warning
		scan.Records, coalesceWarnings = CoalesceRecords(scan.Records)
		scan.Warnings = append(scan.Warnings, coalesceWarnings...)
		status.Records = len(scan.Records)
		replaced, err := i.store.ReplaceSource(ctx, name, scan.Records)
		if err != nil {
			return result, fmt.Errorf("refresh %s session index: %w", name, err)
		}
		status.Upserted = replaced.Upserted
		status.Unchanged = replaced.Unchanged
		status.Deleted = replaced.Deleted
		result.Sources = append(result.Sources, status)
		for _, warning := range scan.Warnings {
			if warning.Agent == "" {
				warning.Agent = name
			}
			if warning.Source == "" {
				warning.Source = name
			}
			warning.Agent = SafeText(warning.Agent, 64)
			warning.SessionID = SafeText(warning.SessionID, maxMetadataRunes)
			warning.Source = SafeText(warning.Source, maxMetadataRunes)
			warning.Code = SafeText(warning.Code, 128)
			warning.Message = SafeText(warning.Message, 512)
			result.Warnings = append(result.Warnings, warning)
		}
	}
	return result, nil
}

// Search reads current derived state. Call Refresh first when freshness matters.
func (i *Index) Search(ctx context.Context, filter Filter) ([]Record, error) {
	if i == nil || i.store == nil {
		return nil, errors.New("session index is nil")
	}
	return i.store.Search(ctx, filter)
}

// Resolve resolves a qualified or unambiguous bare session reference.
func (i *Index) Resolve(ctx context.Context, reference string) (Record, error) {
	if i == nil || i.store == nil {
		return Record{}, errors.New("session index is nil")
	}
	return i.store.Resolve(ctx, reference)
}

// Last returns the newest matching record.
func (i *Index) Last(ctx context.Context, filter Filter) (Record, error) {
	if i == nil || i.store == nil {
		return Record{}, errors.New("session index is nil")
	}
	return i.store.Last(ctx, filter)
}
