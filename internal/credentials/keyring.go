package credentials

import (
	"encoding/json"
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

const keyringService = "reinstate"

// KeyringStore persists storage credentials in the native OS credential store.
type KeyringStore struct{}

func NewKeyringStore() *KeyringStore {
	return &KeyringStore{}
}

func (k *KeyringStore) Set(ref string, c StorageCredentials) error {
	if ref == "" {
		return fmt.Errorf("empty credential ref")
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("incomplete credentials")
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, ref, string(raw)); err != nil {
		return fmt.Errorf("store credentials in OS keyring: %w", err)
	}
	return nil
}

func (k *KeyringStore) Get(ref string) (StorageCredentials, error) {
	if ref == "" {
		return StorageCredentials{}, fmt.Errorf("empty credential ref")
	}
	raw, err := keyring.Get(keyringService, ref)
	if err != nil {
		return StorageCredentials{}, fmt.Errorf("load credentials from OS keyring: %w", err)
	}
	var c StorageCredentials
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return StorageCredentials{}, fmt.Errorf("decode OS keyring credentials: %w", err)
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return StorageCredentials{}, fmt.Errorf("incomplete credentials in OS keyring")
	}
	return c, nil
}

func (k *KeyringStore) Delete(ref string) error {
	err := keyring.Delete(keyringService, ref)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// Probe checks that the OS keyring provider is callable without writing a
// credential. A missing random entry is the expected successful response.
func (k *KeyringStore) Probe() error {
	_, err := keyring.Get(keyringService, "__reinstate_probe_missing__")
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("OS keyring unavailable: %w", err)
	}
	return nil
}

// Resolve loads credentials from the explicit environment fallback first,
// then from the native OS keyring named by credential_ref.
func Resolve(_ string, ref string) (StorageCredentials, error) {
	if ak, sk := envCredentials(); ak != "" || sk != "" {
		if ak == "" || sk == "" {
			return StorageCredentials{}, fmt.Errorf("both REINSTATE_S3_ACCESS_KEY_ID and REINSTATE_S3_SECRET_ACCESS_KEY are required")
		}
		return StorageCredentials{AccessKeyID: ak, SecretAccessKey: sk}, nil
	}
	return NewKeyringStore().Get(ref)
}
