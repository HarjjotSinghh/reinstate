// Package memory is an in-process Backend for tests.
package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

type object struct {
	data []byte
	etag string
}

// Store is a thread-safe memory backend.
type Store struct {
	mu   sync.Mutex
	data map[string]object
}

// New returns an empty memory store.
func New() *Store {
	return &Store{data: map[string]object{}}
}

func etagOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	_ = ctx
	b, err := io.ReadAll(r)
	if err != nil {
		return backend.ObjectMeta{}, err
	}
	if size >= 0 && int64(len(b)) != size && size != 0 {
		// allow size 0 meaning unknown
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, exists := s.data[key]
	if opts.IfNoneMatch && exists {
		return backend.ObjectMeta{}, backend.ErrAlreadyExists
	}
	if opts.IfMatch != "" {
		if !exists || cur.etag != opts.IfMatch {
			return backend.ObjectMeta{}, backend.ErrPrecondition
		}
	}
	et := etagOf(b)
	s.data[key] = object{data: append([]byte(nil), b...), etag: et}
	return backend.ObjectMeta{Key: key, ETag: et, Size: int64(len(b))}, nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data[key]
	if !ok {
		return nil, backend.ObjectMeta{}, backend.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(o.data)), backend.ObjectMeta{Key: key, ETag: o.etag, Size: int64(len(o.data))}, nil
}

func (s *Store) Head(ctx context.Context, key string) (backend.ObjectMeta, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data[key]
	if !ok {
		return backend.ObjectMeta{}, backend.ErrNotFound
	}
	return backend.ObjectMeta{Key: key, ETag: o.etag, Size: int64(len(o.data))}, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return backend.ErrNotFound
	}
	delete(s.data, key)
	return nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []backend.ObjectMeta
	for k, o := range s.data {
		if strings.HasPrefix(k, prefix) {
			out = append(out, backend.ObjectMeta{Key: k, ETag: o.etag, Size: int64(len(o.data))})
		}
	}
	return out, nil
}
