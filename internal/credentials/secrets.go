package credentials

import (
	"errors"
	"fmt"
	"sync"

	keyring "github.com/zalando/go-keyring"
)

// ErrSecretNotFound reports that the store has no entry for the ref.
var ErrSecretNotFound = errors.New("credentials: secret not found")

// SecretStore persists opaque device secrets (the hosted-tier device key)
// outside config files. Production uses the OS keyring; tests use memory.
type SecretStore interface {
	SetSecret(ref string, secret []byte) error
	GetSecret(ref string) ([]byte, error)
	DeleteSecret(ref string) error
}

// SetSecret stores an opaque secret in the OS keyring.
func (k *KeyringStore) SetSecret(ref string, secret []byte) error {
	if ref == "" {
		return fmt.Errorf("empty secret ref")
	}
	if len(secret) == 0 {
		return fmt.Errorf("empty secret")
	}
	if err := keyring.Set(keyringService, ref, string(secret)); err != nil {
		return fmt.Errorf("store secret in OS keyring: %w", err)
	}
	return nil
}

// GetSecret loads an opaque secret from the OS keyring.
func (k *KeyringStore) GetSecret(ref string) ([]byte, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty secret ref")
	}
	raw, err := keyring.Get(keyringService, ref)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load secret from OS keyring: %w", err)
	}
	return []byte(raw), nil
}

// DeleteSecret removes an opaque secret; a missing entry is not an error.
func (k *KeyringStore) DeleteSecret(ref string) error {
	err := keyring.Delete(keyringService, ref)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// MemorySecrets is an in-process SecretStore for tests.
type MemorySecrets struct {
	mu   sync.Mutex
	data map[string][]byte
}

// NewMemorySecrets returns an empty in-memory secret store.
func NewMemorySecrets() *MemorySecrets {
	return &MemorySecrets{data: map[string][]byte{}}
}

func (m *MemorySecrets) SetSecret(ref string, secret []byte) error {
	if ref == "" {
		return fmt.Errorf("empty secret ref")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[ref] = append([]byte(nil), secret...)
	return nil
}

func (m *MemorySecrets) GetSecret(ref string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	secret, ok := m.data[ref]
	if !ok {
		return nil, ErrSecretNotFound
	}
	return append([]byte(nil), secret...), nil
}

func (m *MemorySecrets) DeleteSecret(ref string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, ref)
	return nil
}
