package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// DiskStore is a process-safe Backend that persists objects under a directory.
// Used for CLI e2e with REINSTATE_BACKEND=memory so push/pull share state.
type DiskStore struct {
	root string
	mu   sync.Mutex
}

type diskMeta struct {
	ETag string `json:"etag"`
	Size int64  `json:"size"`
}

// NewDisk returns a disk-backed store at root.
func NewDisk(root string) (*DiskStore, error) {
	if err := fsx.EnsureOwnerOnlyDir(root); err != nil {
		return nil, err
	}
	if err := fsx.EnsureOwnerOnlyDir(filepath.Join(root, "objects")); err != nil {
		return nil, err
	}
	return &DiskStore{root: root}, nil
}

func (d *DiskStore) objPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(d.root, "objects", hex.EncodeToString(sum[:]))
}

func (d *DiskStore) metaPath(key string) string {
	return d.objPath(key) + ".meta.json"
}

func (d *DiskStore) Put(ctx context.Context, key string, r io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	b, err := io.ReadAll(r)
	if err != nil {
		return backend.ObjectMeta{}, err
	}
	mp := d.metaPath(key)
	op := d.objPath(key)
	var cur diskMeta
	if mb, err := os.ReadFile(mp); err == nil {
		_ = json.Unmarshal(mb, &cur)
		if opts.IfNoneMatch {
			return backend.ObjectMeta{}, backend.ErrAlreadyExists
		}
		if opts.IfMatch != "" && cur.ETag != opts.IfMatch {
			return backend.ObjectMeta{}, backend.ErrPrecondition
		}
	} else if opts.IfMatch != "" {
		return backend.ObjectMeta{}, backend.ErrPrecondition
	}
	sum := sha256.Sum256(b)
	et := hex.EncodeToString(sum[:8])
	if err := fsx.WriteFileAtomic(op, b, 0o600); err != nil {
		return backend.ObjectMeta{}, err
	}
	meta := diskMeta{ETag: et, Size: int64(len(b))}
	mb, _ := json.Marshal(meta)
	if err := fsx.WriteFileAtomic(mp, mb, 0o600); err != nil {
		return backend.ObjectMeta{}, err
	}
	// index key for list
	idx := filepath.Join(d.root, "keys", filepath.FromSlash(key)+".key")
	if err := fsx.EnsureOwnerOnlyDir(filepath.Dir(idx)); err != nil {
		return backend.ObjectMeta{}, err
	}
	if err := fsx.WriteFileAtomic(idx, []byte(key), 0o600); err != nil {
		return backend.ObjectMeta{}, err
	}
	return backend.ObjectMeta{Key: key, ETag: et, Size: int64(len(b))}, nil
}

func (d *DiskStore) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	op := d.objPath(key)
	mp := d.metaPath(key)
	mb, err := os.ReadFile(mp)
	if err != nil {
		return nil, backend.ObjectMeta{}, backend.ErrNotFound
	}
	var meta diskMeta
	_ = json.Unmarshal(mb, &meta)
	f, err := os.Open(op)
	if err != nil {
		return nil, backend.ObjectMeta{}, backend.ErrNotFound
	}
	return f, backend.ObjectMeta{Key: key, ETag: meta.ETag, Size: meta.Size}, nil
}

func (d *DiskStore) Head(ctx context.Context, key string) (backend.ObjectMeta, error) {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	mb, err := os.ReadFile(d.metaPath(key))
	if err != nil {
		return backend.ObjectMeta{}, backend.ErrNotFound
	}
	var meta diskMeta
	_ = json.Unmarshal(mb, &meta)
	return backend.ObjectMeta{Key: key, ETag: meta.ETag, Size: meta.Size}, nil
}

func (d *DiskStore) Delete(ctx context.Context, key string) error {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	_ = os.Remove(d.objPath(key))
	_ = os.Remove(d.metaPath(key))
	_ = os.Remove(filepath.Join(d.root, "keys", filepath.FromSlash(key)+".key"))
	return nil
}

func (d *DiskStore) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	root := filepath.Join(d.root, "keys")
	var out []backend.ObjectMeta
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".key") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		key := string(b)
		if !strings.HasPrefix(key, prefix) {
			return nil
		}
		if m, err := d.headUnlocked(key); err == nil {
			out = append(out, m)
		}
		return nil
	})
	return out, nil
}

func (d *DiskStore) headUnlocked(key string) (backend.ObjectMeta, error) {
	mb, err := os.ReadFile(d.metaPath(key))
	if err != nil {
		return backend.ObjectMeta{}, backend.ErrNotFound
	}
	var meta diskMeta
	_ = json.Unmarshal(mb, &meta)
	return backend.ObjectMeta{Key: key, ETag: meta.ETag, Size: meta.Size}, nil
}
