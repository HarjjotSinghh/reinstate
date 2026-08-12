// Package handoff stores portable continuity capsules and append-only lineage
// under $REINSTATE_HOME/handoffs. The store is local-only and hard-excluded
// from sync in v0.4.0.
package handoff

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/filelock"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

const (
	handoffsDirName   = "handoffs"
	lineageFileName   = "lineage.jsonl"
	storeLockName     = ".store.lock"
	lineageLockName   = "lineage.jsonl.lock"
	writeLockTimeout  = 10 * time.Second
	defaultListLimit  = 100
	maxListLimit      = 1000
	capsuleFileName   = "capsule.json"
	projectionFile    = "projection.md"
	bootstrapFileName = "bootstrap.txt"
	fidelityFileName  = "fidelity.json"
	sidecarDirName    = "sidecar"
	eventsFileName    = "events.jsonl"
	blobsDirName      = "blobs"
)

// ErrNotFound means Get could not locate a handoff directory.
var ErrNotFound = errors.New("handoff not found")

// ErrInsideRepository means OpenStore refused a root under a git work tree.
var ErrInsideRepository = errors.New("handoff store must not live inside a repository")

// Store is the on-disk handoff + lineage root under $REINSTATE_HOME/handoffs.
type Store struct {
	root string
}

// Artifacts are the exact destination-facing files written beside capsule.json.
type Artifacts struct {
	ProjectionMD  []byte
	Bootstrap     []byte
	FidelityJSON  []byte
	SidecarEvents []byte
	SidecarBlobs  map[string][]byte
}

// LineageEndpoint identifies one side of a handoff in lineage.jsonl.
type LineageEndpoint struct {
	Agent          string `json:"agent"`
	SessionID      string `json:"session_id,omitempty"`
	ArtifactSHA256 string `json:"artifact_sha256,omitempty"`
	State          string `json:"state,omitempty"`
}

// LineageEntry is one append-only record in lineage.jsonl.
type LineageEntry struct {
	HandoffID        string          `json:"handoff_id"`
	LineageRoot      string          `json:"lineage_root"`
	CreatedAt        time.Time       `json:"created_at"`
	Source           LineageEndpoint `json:"source"`
	Destination      LineageEndpoint `json:"destination"`
	Policy           string          `json:"policy"`
	CapsuleSHA256    string          `json:"capsule_sha256"`
	ProjectionSHA256 string          `json:"projection_sha256"`
	FidelityOverall  string          `json:"fidelity_overall"`
	Launched         bool            `json:"launched"`
	Acknowledged     *bool           `json:"acknowledged"`
}

// OpenStore creates $REINSTATE_HOME/handoffs at 0700 (Windows: protected DACL).
func OpenStore(reinstateHome string) (*Store, error) {
	home := strings.TrimSpace(reinstateHome)
	if home == "" {
		return nil, errors.New("reinstate home must not be empty")
	}
	home = filepath.Clean(home)
	if !filepath.IsAbs(home) {
		return nil, errors.New("reinstate home must be an absolute path")
	}
	root := filepath.Join(home, handoffsDirName)
	if err := rejectInsideRepository(root); err != nil {
		return nil, err
	}
	if err := fsx.EnsureOwnerOnlyDir(home); err != nil {
		return nil, fmt.Errorf("ensure reinstate home: %w", err)
	}
	if err := fsx.EnsureOwnerOnlyDir(root); err != nil {
		return nil, fmt.Errorf("ensure handoffs root: %w", err)
	}
	return &Store{root: root}, nil
}

// Root returns the absolute handoffs directory.
func (s *Store) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

// Put writes capsule.json and artifacts under handoffs/<id>/. Returns the handoff id.
func (s *Store) Put(c capsule.Capsule, artifacts Artifacts) (string, error) {
	if s == nil || s.root == "" {
		return "", errors.New("handoff store is nil")
	}
	id := strings.TrimSpace(c.Identity.ID)
	if id == "" {
		computed, err := capsule.ComputeID(c)
		if err != nil {
			return "", err
		}
		id = computed
		c.Identity.ID = id
	}
	if err := validateHandoffID(id); err != nil {
		return "", err
	}

	lock, err := s.acquireLock(storeLockName)
	if err != nil {
		return "", err
	}
	defer func() { _ = lock.Close() }()

	dir := filepath.Join(s.root, id)
	if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
		return "", err
	}

	capsuleBytes, err := capsule.CanonicalBytes(c)
	if err != nil {
		return "", err
	}
	if err := writePrivateFile(filepath.Join(dir, capsuleFileName), capsuleBytes); err != nil {
		return "", err
	}
	if err := writePrivateFile(filepath.Join(dir, projectionFile), artifacts.ProjectionMD); err != nil {
		return "", err
	}
	if err := writePrivateFile(filepath.Join(dir, bootstrapFileName), artifacts.Bootstrap); err != nil {
		return "", err
	}
	fidelity := artifacts.FidelityJSON
	if len(fidelity) == 0 {
		fidelity, err = json.Marshal(c.Fidelity)
		if err != nil {
			return "", fmt.Errorf("marshal fidelity: %w", err)
		}
	}
	if err := writePrivateFile(filepath.Join(dir, fidelityFileName), fidelity); err != nil {
		return "", err
	}

	if len(artifacts.SidecarEvents) > 0 || len(artifacts.SidecarBlobs) > 0 {
		sidecar := filepath.Join(dir, sidecarDirName)
		if err := fsx.EnsureOwnerOnlyDir(sidecar); err != nil {
			return "", err
		}
		if len(artifacts.SidecarEvents) > 0 {
			if err := writePrivateFile(filepath.Join(sidecar, eventsFileName), artifacts.SidecarEvents); err != nil {
				return "", err
			}
		}
		if len(artifacts.SidecarBlobs) > 0 {
			blobs := filepath.Join(sidecar, blobsDirName)
			if err := fsx.EnsureOwnerOnlyDir(blobs); err != nil {
				return "", err
			}
			for name, data := range artifacts.SidecarBlobs {
				if err := validateBlobName(name); err != nil {
					return "", err
				}
				if err := writePrivateFile(filepath.Join(blobs, name), data); err != nil {
					return "", err
				}
			}
		}
	}
	return id, nil
}

// Get loads a stored capsule and its artifacts.
func (s *Store) Get(handoffID string) (capsule.Capsule, Artifacts, error) {
	var zero capsule.Capsule
	var arts Artifacts
	if s == nil || s.root == "" {
		return zero, arts, errors.New("handoff store is nil")
	}
	if err := validateHandoffID(handoffID); err != nil {
		return zero, arts, err
	}
	dir := filepath.Join(s.root, handoffID)
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, arts, fmt.Errorf("%w: %s", ErrNotFound, handoffID)
		}
		return zero, arts, err
	}
	if !info.IsDir() {
		return zero, arts, fmt.Errorf("%w: %s", ErrNotFound, handoffID)
	}

	capsuleRaw, err := os.ReadFile(filepath.Join(dir, capsuleFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, arts, fmt.Errorf("%w: %s", ErrNotFound, handoffID)
		}
		return zero, arts, err
	}
	var c capsule.Capsule
	if err := json.Unmarshal(capsuleRaw, &c); err != nil {
		return zero, arts, fmt.Errorf("decode capsule.json: %w", err)
	}

	arts.ProjectionMD, err = optionalRead(filepath.Join(dir, projectionFile))
	if err != nil {
		return zero, arts, err
	}
	arts.Bootstrap, err = optionalRead(filepath.Join(dir, bootstrapFileName))
	if err != nil {
		return zero, arts, err
	}
	arts.FidelityJSON, err = optionalRead(filepath.Join(dir, fidelityFileName))
	if err != nil {
		return zero, arts, err
	}
	arts.SidecarEvents, err = optionalRead(filepath.Join(dir, sidecarDirName, eventsFileName))
	if err != nil {
		return zero, arts, err
	}
	blobsDir := filepath.Join(dir, sidecarDirName, blobsDirName)
	entries, err := os.ReadDir(blobsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return zero, arts, err
	}
	if len(entries) > 0 {
		arts.SidecarBlobs = make(map[string][]byte, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if err := validateBlobName(name); err != nil {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(blobsDir, name))
			if readErr != nil {
				return zero, arts, readErr
			}
			arts.SidecarBlobs[name] = data
		}
	}
	return c, arts, nil
}

// List returns well-formed lineage entries, newest first. Partial trailing
// lines are skipped. limit <= 0 uses the default; values above max are capped.
func (s *Store) List(limit int) ([]LineageEntry, error) {
	if s == nil || s.root == "" {
		return nil, errors.New("handoff store is nil")
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	path := filepath.Join(s.root, lineageFileName)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var entries []LineageEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry LineageEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip corrupt or partially written lines; never rewrite the file.
			continue
		}
		if strings.TrimSpace(entry.HandoffID) == "" {
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// AppendLineage appends one JSON line to lineage.jsonl. The file is never rewritten.
func (s *Store) AppendLineage(entry LineageEntry) error {
	if s == nil || s.root == "" {
		return errors.New("handoff store is nil")
	}
	if strings.TrimSpace(entry.HandoffID) == "" {
		return errors.New("lineage entry requires handoff_id")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC().Truncate(time.Second)
	} else {
		entry.CreatedAt = entry.CreatedAt.UTC().Truncate(time.Second)
	}

	lock, err := s.acquireLock(lineageLockName)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()

	path := filepath.Join(s.root, lineageFileName)
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := fsx.ProtectOwnerOnly(path, false); err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		return err
	}
	return f.Sync()
}

func (s *Store) acquireLock(name string) (*filelock.Lock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), writeLockTimeout)
	defer cancel()
	lock, err := filelock.Acquire(ctx, filepath.Join(s.root, name))
	if err != nil {
		return nil, fmt.Errorf("lock handoff store: %w", err)
	}
	return lock, nil
}

func writePrivateFile(path string, data []byte) error {
	if err := fsx.WriteFileAtomic(path, data, fsx.OwnerOnlyFilePerm); err != nil {
		return err
	}
	return fsx.ProtectOwnerOnly(path, false)
}

func optionalRead(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func validateHandoffID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" || id == "." || id == ".." {
		return errors.New("invalid handoff id")
	}
	if filepath.Base(id) != id {
		return fmt.Errorf("invalid handoff id %q", id)
	}
	if strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("invalid handoff id %q", id)
	}
	return nil
}

func validateBlobName(name string) error {
	if err := validateHandoffID(name); err != nil {
		return fmt.Errorf("invalid sidecar blob name: %w", err)
	}
	return nil
}

func rejectInsideRepository(path string) error {
	current := filepath.Clean(path)
	for {
		marker := filepath.Join(current, ".git")
		if _, err := os.Lstat(marker); err == nil {
			return fmt.Errorf("%w: %s", ErrInsideRepository, path)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
