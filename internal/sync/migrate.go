package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// Migration copies every snapshot and the manifest from one storage (the
// source, read with its own keys) to another (the destination, sealed with
// different keys). It is how an account leaves the hosted tier for BYO
// storage: the locker is only ever read, every envelope is opened on the
// device and re-sealed under the destination provider, and the destination
// never receives the source's key material. Snapshot ids are preserved so
// local state on other devices keeps meaning.
//
// A migration is resumable: the caller keeps the Done set across runs and a
// snapshot whose object already exists at the destination is verified in
// place rather than written again, so an interruption never leaves a
// duplicate. The manifest is written last, with the usual compare-and-swap,
// only once every snapshot has been re-read from the destination and its
// plaintext digest compared with the source's.
type Migration struct {
	// Source reads the envelopes to move. Only Get, Head, and List are
	// ever called on it, so a read-only credential suffices.
	Source *Engine
	// Destination writes the re-sealed envelopes. Its Keys must not share
	// material with Source.Keys; the migration does not check this (it
	// cannot see inside a provider) but the CLI never offers the same
	// provider on both sides.
	Destination *Engine
	// Done lists the snapshot ids an earlier run verified at the
	// destination, keyed to their plaintext digest. A snapshot in Done whose
	// object still exists is skipped without being re-read.
	Done map[string]string
	// Progress, when set, is called after every snapshot is settled and
	// once after the manifest.
	Progress func(MigrateProgress)
}

// MigrateProgress is one step of a migration as reported to Progress.
type MigrateProgress struct {
	// SnapshotID is empty for the manifest step.
	SnapshotID string
	// Digest is the hex SHA-256 of the snapshot's plaintext, the value to
	// remember in Done.
	Digest string
	// Skipped is true when the snapshot was already verified by an earlier
	// run; Existing is true when its object was found at the destination
	// and verified instead of written.
	Skipped, Existing bool
	// Completed and Total count snapshots; Bytes is the ciphertext written
	// or verified so far.
	Completed, Total int
	Bytes            int64
}

// MigrateReport summarises a finished migration.
type MigrateReport struct {
	Snapshots        int    `json:"snapshots"`
	Written          int    `json:"written"`
	Verified         int    `json:"verified"`
	Skipped          int    `json:"skipped"`
	Bytes            int64  `json:"bytes"`
	ManifestSessions int    `json:"manifest_sessions"`
	ManifestRevision string `json:"manifest_revision"`
}

// ErrMigrateVerify means a re-read of the destination did not match what
// was read from the source; the object is left for the person to inspect.
var ErrMigrateVerify = errors.New("sync: migration verification failed")

// Run performs the migration. It never writes to the source.
func (m *Migration) Run(ctx context.Context) (MigrateReport, error) {
	var report MigrateReport
	if m.Source == nil || m.Destination == nil || m.Source.Backend == nil || m.Destination.Backend == nil {
		return report, fmt.Errorf("source and destination engines are required")
	}
	if m.Source.Keys == nil || m.Destination.Keys == nil {
		return report, fmt.Errorf("source and destination key providers are required")
	}
	ids, err := m.Source.ListSnapshots(ctx)
	if err != nil {
		return report, fmt.Errorf("list source snapshots: %w", err)
	}
	// The manifest is read before any snapshot so a source that cannot be
	// opened (wrong generation, missing key) fails before anything is written.
	manifest, _, err := m.Source.loadManifest(ctx, true)
	if err != nil {
		return report, fmt.Errorf("read source manifest: %w", err)
	}
	report.Snapshots = len(ids)
	// On a resumed run the first remembered snapshot is re-read rather than
	// trusted, so a resume under a different destination passphrase is
	// caught before it writes a manifest the earlier objects cannot share.
	checkedResume := false
	for i, id := range ids {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		step := MigrateProgress{SnapshotID: id, Completed: i + 1, Total: len(ids)}
		destKey := m.Destination.key("snapshots/" + id + ".age")
		if digest, ok := m.Done[id]; ok {
			if meta, err := m.Destination.Backend.Head(ctx, destKey); err == nil {
				if !checkedResume {
					if err := m.Destination.verifySnapshot(ctx, id, digest); err != nil {
						return report, fmt.Errorf("snapshot %s (verified by an earlier run): %w", id, err)
					}
					checkedResume = true
				}
				step.Digest, step.Skipped = digest, true
				report.Skipped++
				report.Bytes += meta.Size
				step.Bytes = report.Bytes
				m.report(step)
				continue
			} else if !errors.Is(err, backend.ErrNotFound) {
				return report, err
			}
		}
		digest, size, existing, err := m.copySnapshot(ctx, id, destKey)
		if err != nil {
			return report, fmt.Errorf("snapshot %s: %w", id, err)
		}
		if existing {
			report.Verified++
		} else {
			report.Written++
		}
		report.Bytes += size
		step.Digest, step.Existing, step.Bytes = digest, existing, report.Bytes
		m.report(step)
	}
	if err := m.writeManifest(ctx, manifest); err != nil {
		return report, fmt.Errorf("write destination manifest: %w", err)
	}
	report.ManifestSessions = len(manifest.Sessions)
	report.ManifestRevision = manifest.Revision
	m.report(MigrateProgress{Completed: len(ids), Total: len(ids), Bytes: report.Bytes})
	return report, nil
}

func (m *Migration) report(p MigrateProgress) {
	if m.Progress != nil {
		m.Progress(p)
	}
}

// copySnapshot opens one source snapshot, re-seals it to a temporary file,
// writes it create-only, and verifies the destination by re-reading it. When
// the destination object already exists (an earlier run was interrupted
// after the put), it is verified instead of replaced.
func (m *Migration) copySnapshot(ctx context.Context, id, destKey string) (digest string, size int64, existing bool, err error) {
	if meta, err := m.Destination.Backend.Head(ctx, destKey); err == nil {
		want, err := m.Source.snapshotDigest(ctx, id)
		if err != nil {
			return "", 0, true, fmt.Errorf("read source: %w", err)
		}
		if err := m.Destination.verifySnapshot(ctx, id, want); err != nil {
			return "", 0, true, err
		}
		return want, meta.Size, true, nil
	} else if !errors.Is(err, backend.ErrNotFound) {
		return "", 0, false, err
	}

	// One pass over the source: the plaintext is hashed as it is re-sealed,
	// so the locker is read exactly once per snapshot.
	rc, _, err := m.Source.Backend.Get(ctx, m.Source.key("snapshots/"+id+".age"))
	if err != nil {
		return "", 0, false, fmt.Errorf("read source: %w", err)
	}
	defer func() { _ = rc.Close() }()
	plain, err := m.Source.envelopeCodec().DecryptReader(rc, m.Source.Keys)
	if err != nil {
		return "", 0, false, fmt.Errorf("open source envelope: %w", err)
	}
	cipher, err := os.CreateTemp("", ".reinstate-migrate-*.age")
	if err != nil {
		return "", 0, false, err
	}
	cipherPath := cipher.Name()
	defer func() {
		_ = cipher.Close()
		_ = os.Remove(cipherPath)
	}()
	if err := os.Chmod(cipherPath, 0o600); err != nil {
		return "", 0, false, err
	}
	hash := sha256.New()
	if err := m.Destination.envelopeCodec().Encrypt(io.TeeReader(plain, hash), cipher, m.Destination.Keys); err != nil {
		return "", 0, false, fmt.Errorf("seal for destination: %w", err)
	}
	digest = hex.EncodeToString(hash.Sum(nil))
	if err := cipher.Sync(); err != nil {
		return "", 0, false, err
	}
	info, err := cipher.Stat()
	if err != nil {
		return "", 0, false, err
	}
	if _, err := cipher.Seek(0, io.SeekStart); err != nil {
		return "", 0, false, err
	}
	_, err = m.Destination.Backend.Put(ctx, destKey, cipher, info.Size(), backend.PutOptions{
		IfNoneMatch: true,
		ContentType: "application/octet-stream",
	})
	if errors.Is(err, backend.ErrAlreadyExists) || errors.Is(err, backend.ErrPrecondition) {
		// Another run raced this one; what is there must still match.
		existing = true
	} else if err != nil {
		return "", 0, false, fmt.Errorf("write destination: %w", err)
	}
	if err := m.Destination.verifySnapshot(ctx, id, digest); err != nil {
		return "", 0, existing, err
	}
	return digest, info.Size(), existing, nil
}

// writeManifest installs the source manifest's sessions at the destination
// with compare-and-swap. A destination manifest written by an earlier run is
// replaced wholesale; one that belongs to a different profile is left alone.
func (m *Migration) writeManifest(ctx context.Context, source *schema.Manifest) error {
	for attempt := 0; attempt < maxManifestRetries; attempt++ {
		current, etag, err := m.Destination.loadManifest(ctx, true)
		if err != nil {
			return err
		}
		for key := range current.Sessions {
			if _, moved := source.Sessions[key]; !moved {
				return fmt.Errorf("destination manifest already lists %s, which the source does not; refusing to merge into a storage that is in use", key)
			}
		}
		out := *source
		out.Sessions = map[string]schema.ManifestSession{}
		for key, session := range source.Sessions {
			out.Sessions[key] = session
		}
		out.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		err = m.Destination.saveManifest(ctx, &out, etag)
		if errors.Is(err, backend.ErrPrecondition) || errors.Is(err, backend.ErrAlreadyExists) {
			continue
		}
		if err != nil {
			return err
		}
		verify, _, err := m.Destination.loadManifest(ctx, false)
		if err != nil {
			return fmt.Errorf("%w: re-read manifest: %v", ErrMigrateVerify, err)
		}
		if len(verify.Sessions) != len(source.Sessions) || verify.Revision != source.Revision {
			return fmt.Errorf("%w: destination manifest differs from the source", ErrMigrateVerify)
		}
		for key, want := range source.Sessions {
			if got, ok := verify.Sessions[key]; !ok || got.SnapshotID != want.SnapshotID {
				return fmt.Errorf("%w: destination manifest differs from the source at %s", ErrMigrateVerify, key)
			}
		}
		return nil
	}
	return fmt.Errorf("%w: manifest changed too many times", ErrConflict)
}

// ListSnapshots returns every snapshot id stored under the engine's prefix,
// sorted, whether or not the manifest still points at it.
func (e *Engine) ListSnapshots(ctx context.Context) ([]string, error) {
	objects, err := e.Backend.List(ctx, e.key("snapshots/"))
	if err != nil {
		return nil, err
	}
	prefix := strings.Trim(e.Prefix, "/")
	var ids []string
	for _, o := range objects {
		key := o.Key
		if prefix != "" {
			key = strings.TrimPrefix(key, prefix+"/")
		}
		name := strings.TrimPrefix(key, "snapshots/")
		if name == key || !strings.HasSuffix(name, ".age") || strings.Contains(name, "/") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".age"))
	}
	sort.Strings(ids)
	return ids, nil
}

// snapshotDigest opens a snapshot and returns the hex SHA-256 of its
// plaintext (metadata line and payload together).
func (e *Engine) snapshotDigest(ctx context.Context, id string) (string, error) {
	rc, _, err := e.Backend.Get(ctx, e.key("snapshots/"+id+".age"))
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()
	plain, err := e.envelopeCodec().DecryptReader(rc, e.Keys)
	if err != nil {
		return "", fmt.Errorf("open envelope: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, plain); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifySnapshot re-reads a destination snapshot and compares its plaintext
// digest with the one read from the source.
func (e *Engine) verifySnapshot(ctx context.Context, id, want string) error {
	got, err := e.snapshotDigest(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: re-read destination: %v", ErrMigrateVerify, err)
	}
	if got != want {
		return fmt.Errorf("%w: destination snapshot %s does not match the source", ErrMigrateVerify, id)
	}
	return nil
}

// ContainsKeyringObject reports whether any object under the engine's prefix
// is a keyring, which a migration to BYO storage must never carry over.
func (e *Engine) ContainsKeyringObject(ctx context.Context) (bool, error) {
	_, err := e.Backend.Head(ctx, keyring.ObjectKey(e.Prefix))
	if errors.Is(err, backend.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}
