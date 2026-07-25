// Package backend defines remote object storage for Reinstate.
package backend

import (
	"context"
	"errors"
	"io"
)

// Common errors.
var (
	ErrNotFound      = errors.New("backend: not found")
	ErrPrecondition  = errors.New("backend: precondition failed")
	ErrAlreadyExists = errors.New("backend: already exists")
	ErrUnauthorized  = errors.New("backend: unauthorized")
)

// ObjectMeta is remote object metadata.
type ObjectMeta struct {
	Key  string
	ETag string
	Size int64
}

// PutOptions controls conditional puts.
type PutOptions struct {
	IfMatch     string // conditional update when ETag matches
	IfNoneMatch bool   // create-only when true (If-None-Match: *)
	ContentType string
}

// Backend is the storage interface.
type Backend interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) (ObjectMeta, error)
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectMeta, error)
	Head(ctx context.Context, key string) (ObjectMeta, error)
	Delete(ctx context.Context, key string) error
	// List returns keys with the given prefix (non-recursive best-effort).
	List(ctx context.Context, prefix string) ([]ObjectMeta, error)
}
