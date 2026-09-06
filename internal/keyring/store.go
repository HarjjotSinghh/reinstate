package keyring

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

// maxObjectBytes bounds a keyring read; a real keyring is a few KiB.
const maxObjectBytes = 1 << 20

// maxWriteBytes bounds a keyring write, well below maxObjectBytes on
// purpose. The object only grows: every revocation appends a generation,
// each carrying one wrap per surviving device, and nothing is ever removed.
// An account that once wrote a keyring larger than a read accepts could
// never read it back — not to push, not to pull, not to revoke again, and
// not to recover from the recovery code. Refusing the write while there is
// still a third of the read budget spare means that state is unreachable.
const maxWriteBytes = maxObjectBytes * 3 / 4

// maxUpdateRetries bounds the compare-and-swap loop in Update.
const maxUpdateRetries = 8

// ErrNotFound reports that the profile has no keyring object yet.
var ErrNotFound = errors.New("keyring: not found")

// ErrTooLarge reports a keyring that has grown too close to the size a read
// accepts for this package to keep appending to it.
var ErrTooLarge = errors.New("keyring: object has grown too large to write")

// checkWriteSize refuses a keyring that is approaching the read cap, naming
// what actually causes the growth and what can be done about it. There is no
// compaction: dropping a superseded generation would drop the only copy of
// the root key that opens everything written under it, so history and size
// cannot both be kept.
func checkWriteSize(k *Keyring, raw []byte) error {
	if len(raw) <= maxWriteBytes {
		return nil
	}
	return fmt.Errorf("%w: %d bytes across %d key generations, and a read accepts at most %d. Every revocation appends a generation, each holding one wrap per remaining device, and no generation is ever removed — removing one would make everything written under it unreadable. Nothing was written: pull what each device still needs, then start the account again on a fresh locker",
		ErrTooLarge, len(raw), len(k.Generations), maxObjectBytes)
}

// ErrConflict reports that another device kept changing the keyring faster
// than Update could re-apply its change.
var ErrConflict = errors.New("keyring: concurrent update did not converge")

// ObjectKey is the keyring's object key under an engine prefix (empty when
// the backend already scopes keys to the profile).
func ObjectKey(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ObjectName
	}
	return prefix + "/" + ObjectName
}

// Load fetches and parses the keyring, returning the ETag for a later
// compare-and-swap write.
func Load(ctx context.Context, store backend.Backend, key string) (*Keyring, string, error) {
	rc, meta, err := store.Get(ctx, key)
	if errors.Is(err, backend.ErrNotFound) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("keyring: fetch: %w", err)
	}
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(io.LimitReader(rc, maxObjectBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxObjectBytes {
		return nil, "", fmt.Errorf("keyring: object exceeds maximum size")
	}
	k, err := Parse(raw)
	if err != nil {
		return nil, "", err
	}
	return k, meta.ETag, nil
}

// Create writes a brand-new keyring with a create-only put, so two first
// devices can never silently overwrite each other's root key.
func Create(ctx context.Context, store backend.Backend, key string, k *Keyring) error {
	raw, err := k.Marshal()
	if err != nil {
		return err
	}
	if err := checkWriteSize(k, raw); err != nil {
		return err
	}
	_, err = store.Put(ctx, key, bytes.NewReader(raw), int64(len(raw)), backend.PutOptions{
		IfNoneMatch: true,
		ContentType: "application/json",
	})
	if errors.Is(err, backend.ErrAlreadyExists) || errors.Is(err, backend.ErrPrecondition) {
		return fmt.Errorf("keyring: a keyring already exists for this profile")
	}
	if err != nil {
		return fmt.Errorf("keyring: create: %w", err)
	}
	return nil
}

// Update applies mutate to the latest keyring and writes it back with the
// same compare-and-swap discipline as the manifest. When another device wrote
// in between, the keyring is reloaded and mutate runs again on the fresh
// copy, so concurrent enrolments converge instead of one wrap being lost.
func Update(ctx context.Context, store backend.Backend, key string, mutate func(*Keyring) error) (*Keyring, error) {
	for range maxUpdateRetries {
		k, etag, err := Load(ctx, store, key)
		if err != nil {
			return nil, err
		}
		if err := mutate(k); err != nil {
			return nil, err
		}
		raw, err := k.Marshal()
		if err != nil {
			return nil, err
		}
		if err := checkWriteSize(k, raw); err != nil {
			return nil, err
		}
		_, err = store.Put(ctx, key, bytes.NewReader(raw), int64(len(raw)), backend.PutOptions{
			IfMatch:     etag,
			ContentType: "application/json",
		})
		if errors.Is(err, backend.ErrPrecondition) || errors.Is(err, backend.ErrAlreadyExists) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("keyring: update: %w", err)
		}
		return k, nil
	}
	return nil, ErrConflict
}
