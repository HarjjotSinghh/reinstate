// Package credentials stores remote access secrets outside config files.
package credentials

import (
	"fmt"
	"os"
	"sync"
)

// StorageCredentials are S3/R2 access keys (never written to config.toml).
type StorageCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// Store abstracts credential persistence.
type Store interface {
	Set(ref string, c StorageCredentials) error
	Get(ref string) (StorageCredentials, error)
	Delete(ref string) error
}

// MemoryStore is an in-process store for tests.
type MemoryStore struct {
	mu   sync.Mutex
	data map[string]StorageCredentials
}

// NewMemory returns an empty memory store.
func NewMemory() *MemoryStore {
	return &MemoryStore{data: map[string]StorageCredentials{}}
}

func (m *MemoryStore) Set(ref string, c StorageCredentials) error {
	if ref == "" {
		return fmt.Errorf("empty credential ref")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[ref] = c
	return nil
}

func (m *MemoryStore) Get(ref string) (StorageCredentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.data[ref]
	if !ok {
		return StorageCredentials{}, fmt.Errorf("credentials not found for %s", ref)
	}
	return c, nil
}

func (m *MemoryStore) Delete(ref string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, ref)
	return nil
}

// EnvStore reads REINSTATE_S3_ACCESS_KEY_ID / REINSTATE_S3_SECRET_ACCESS_KEY.
// Explicit fallback when keyring is unavailable.
type EnvStore struct{}

func (EnvStore) Set(ref string, c StorageCredentials) error {
	return fmt.Errorf("env store is read-only; set REINSTATE_S3_ACCESS_KEY_ID and REINSTATE_S3_SECRET_ACCESS_KEY")
}

func (EnvStore) Get(ref string) (StorageCredentials, error) {
	ak := os.Getenv("REINSTATE_S3_ACCESS_KEY_ID")
	sk := os.Getenv("REINSTATE_S3_SECRET_ACCESS_KEY")
	if ak == "" || sk == "" {
		return StorageCredentials{}, fmt.Errorf("env credentials not set")
	}
	return StorageCredentials{AccessKeyID: ak, SecretAccessKey: sk}, nil
}

func (EnvStore) Delete(ref string) error { return nil }
