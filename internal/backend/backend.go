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
	// ErrAccessDenied is the scope half of ErrUnauthorized: the endpoint
	// recognised the credential and refused the request anyway (AccessDenied,
	// or a bodiless 403). Errors that match it also match ErrUnauthorized.
	ErrAccessDenied = errors.New("backend: access denied")
	// ErrCredentialRejected is the other half: the credential itself was not
	// accepted (unknown key id, bad signature, expired or invalid token), so
	// the response says nothing about what the credential is scoped to.
	ErrCredentialRejected = errors.New("backend: credential rejected")
)

// Refusal is an unauthorized response with the endpoint's error code kept,
// so a caller can tell a scope refusal from a dead credential. It matches
// ErrUnauthorized and exactly one of ErrAccessDenied / ErrCredentialRejected.
type Refusal struct {
	// Code is the endpoint's error code, for example "AccessDenied".
	Code string
	// Credential is true when the credential itself was rejected.
	Credential bool
}

func (r *Refusal) Error() string {
	if r.Credential {
		return "backend: credential rejected (" + r.Code + ")"
	}
	return "backend: access denied (" + r.Code + ")"
}

// APIAnswer is a structured answer from the endpoint that no other error
// here names: an S3 API error with a code, arriving from the endpoint
// itself. It exists so a caller can tell "the endpoint said something this
// build has no case for" from "nothing answered at all", which is a
// distinction `rein sync verify` reports to a person. NoSuchBucket is the
// plainest example, and reporting it as an endpoint that said nothing is
// how a locker whose bucket is gone came to stop failing step 1.
type APIAnswer struct {
	// Code is the endpoint's error code, for example "NoSuchBucket".
	Code string
	// Err is the underlying error, kept for the exact text.
	Err error
}

func (a *APIAnswer) Error() string {
	if a.Err == nil {
		return "backend: endpoint answered " + a.Code
	}
	return "backend: endpoint answered " + a.Code + " (" + a.Err.Error() + ")"
}

func (a *APIAnswer) Unwrap() error { return a.Err }

// Is makes errors.Is match the sentinel errors above.
func (r *Refusal) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return true
	case ErrAccessDenied:
		return !r.Credential
	case ErrCredentialRejected:
		return r.Credential
	}
	return false
}

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
