package sessionindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/version"
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

// SourceFresh reports whether the named source participated in this refresh
// and completed successfully. An absent source is never considered fresh.
func (r RefreshResult) SourceFresh(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, source := range r.Sources {
		if source.Name == name {
			return source.Error == ""
		}
	}
	return false
}

// Index orchestrates sources and the private derived store.
type Index struct {
	store   *Store
	sources []Source
	// readerID identifies the build that reads the sources. It is held per
	// index rather than read from a global so that a test can stand in for a
	// different build without disturbing anything running alongside it.
	readerID string
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
	return &Index{store: store, sources: ordered, readerID: readerIdentity()}, nil
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
	return i.refresh(ctx, "")
}

// RefreshAgent scans one selected agent source. It avoids unrelated vendor
// scans while preserving freshness for the exact record being inspected or
// launched.
func (i *Index) RefreshAgent(ctx context.Context, agent string) (RefreshResult, error) {
	agent = strings.ToLower(SafeText(strings.TrimSpace(agent), 64))
	if agent == "" || agent == "all" {
		return RefreshResult{}, errors.New("target refresh requires one agent")
	}
	return i.refresh(ctx, agent)
}

// readerIdentity identifies the code that produced a stored row.
//
// A source fingerprint answers "have these files changed"; it cannot answer
// "would this build read them the same way". The same bytes parse into
// different records whenever a reader is fixed, so a stored fingerprint is
// only meaningful for the build that wrote it. Without this, upgrading
// Reinstate left every existing index frozen: a reader fix stayed invisible
// until the user's agent happened to write a new session.
//
// The release version distinguishes shipped builds and the executable's own
// size and modification time distinguish local ones, so any change to the
// binary invalidates what it previously stored.
var readerIdentity = sync.OnceValue(func() string {
	parts := []string{version.Version}
	if executable, err := os.Executable(); err == nil {
		if info, err := os.Stat(executable); err == nil {
			parts = append(parts,
				strconv.FormatInt(info.Size(), 10),
				strconv.FormatInt(info.ModTime().UnixNano(), 10))
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:16]
})

// qualifyFingerprint binds a source digest to the reader that produced it.
func (i *Index) qualifyFingerprint(digest string) string {
	return i.identity() + ":" + digest
}

func (i *Index) identity() string {
	if i.readerID != "" {
		return i.readerID
	}
	return readerIdentity()
}

// readerOf returns the reader identity a stored fingerprint was written by.
func readerOf(stored string) string {
	identity, _, found := strings.Cut(stored, ":")
	if !found {
		return ""
	}
	return identity
}

// Fingerprinter is an optional Source capability: a cheap summary of
// everything the source would read, computed without opening any file. A
// source that cannot produce one is always scanned.
type Fingerprinter interface {
	Fingerprint(context.Context) (digest string, usable bool, err error)
}

func (i *Index) refresh(ctx context.Context, selected string) (RefreshResult, error) {
	var result RefreshResult
	if i == nil || i.store == nil {
		return result, errors.New("session index is nil")
	}
	for _, source := range i.sources {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		name := strings.ToLower(SafeText(strings.TrimSpace(source.Name()), 64))
		if selected != "" && name != selected {
			continue
		}
		status := SourceRefresh{Name: name}

		// Everything below turns on one question: were these rows written by
		// this build? A fingerprint answers "have the files changed"; it never
		// answers "would this build read them the same way". So the reader
		// identity is recorded alongside it, and when it differs every row is
		// rewritten even though nothing on disk moved. Without that an upgrade
		// leaves the index frozen and a reader fix reaches nobody until their
		// agent happens to write a new session.
		storedFingerprint, storedErr := i.store.SourceFingerprint(ctx, name)
		readerChanged := storedErr != nil || readerOf(storedFingerprint) != i.identity()

		// A source that can summarise itself without opening any file lets an
		// unchanged refresh skip parsing entirely. The fingerprint covers every
		// path, modification time and size the scan would read, so an identical
		// one cannot hide a changed record.
		var pendingFingerprint string
		if fp, ok := source.(Fingerprinter); ok {
			digest, usable, fpErr := fp.Fingerprint(ctx)
			if fpErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return result, ctxErr
				}
			} else if usable && digest != "" {
				digest = i.qualifyFingerprint(digest)
				if storedErr == nil && storedFingerprint == digest {
					count, countErr := i.store.CountSource(ctx, name)
					if countErr == nil {
						status.Records = count
						status.Unchanged = count
						result.Sources = append(result.Sources, status)
						continue
					}
				}
				pendingFingerprint = digest
			}
		}
		if pendingFingerprint == "" {
			// No usable digest: record the reader alone, so the next run can
			// still tell whether this build wrote these rows.
			pendingFingerprint = i.qualifyFingerprint("")
		}

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
		var replaceOptions []ReplaceOption
		if readerChanged {
			replaceOptions = append(replaceOptions, RewriteEveryRow())
		}
		replaced, err := i.store.ReplaceSource(ctx, name, scan.Records, replaceOptions...)
		if err != nil {
			return result, fmt.Errorf("refresh %s session index: %w", name, err)
		}
		status.Upserted = replaced.Upserted
		status.Unchanged = replaced.Unchanged
		status.Deleted = replaced.Deleted
		// Recorded only after the rows are persisted, so a failed scan can
		// never mark a source as up to date.
		if pendingFingerprint != "" {
			if err := i.store.SetSourceFingerprint(ctx, name, pendingFingerprint, time.Now().UTC().UnixNano()); err != nil {
				return result, fmt.Errorf("record %s source fingerprint: %w", name, err)
			}
		}
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
	if selected != "" && len(result.Sources) == 0 {
		return result, fmt.Errorf("session source %q is unavailable", selected)
	}
	return result, nil
}

// RefreshAndResolve refreshes the narrowest knowable source, resolves the
// exact requested identity, and binds freshness to the resolved record.
func (i *Index) RefreshAndResolve(ctx context.Context, reference string) (Record, RefreshResult, bool, error) {
	agent, _, qualified := ParseCompositeReference(reference)
	var (
		refresh RefreshResult
		err     error
	)
	if qualified {
		refresh, err = i.RefreshAgent(ctx, agent)
	} else {
		refresh, err = i.Refresh(ctx)
	}
	if err != nil {
		return Record{}, refresh, false, err
	}
	record, err := i.Resolve(ctx, reference)
	if err != nil {
		return Record{}, refresh, false, err
	}
	return record, refresh, refresh.SourceFresh(record.Agent), nil
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
