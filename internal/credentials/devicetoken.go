package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	keyring "github.com/zalando/go-keyring"
)

// DeviceTokenRef is the OS keyring entry holding this device's Hop token.
const DeviceTokenRef = "hop/device-token"

// DeviceToken is the sign-in credential of this device on the hosted tier.
// It never appears in config files; only the OS keyring holds it.
type DeviceToken struct {
	Token           string `json:"token"`
	ControlPlaneURL string `json:"control_plane_url"`
	AccountID       string `json:"account_id"`
	DeviceID        string `json:"device_id"`
}

// ErrNoDeviceToken reports that this device has not signed in.
var ErrNoDeviceToken = errors.New("this device is not signed in to Reinstate Hop; run `rein login`")

// DeviceTokenStore persists the device token.
type DeviceTokenStore interface {
	SetDeviceToken(DeviceToken) error
	GetDeviceToken() (DeviceToken, error)
	DeleteDeviceToken() error
}

// SetDeviceToken stores the token in the OS keyring.
func (k *KeyringStore) SetDeviceToken(t DeviceToken) error {
	if t.Token == "" || t.ControlPlaneURL == "" {
		return fmt.Errorf("incomplete device token")
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, DeviceTokenRef, string(raw)); err != nil {
		return fmt.Errorf("store device token in OS keyring: %w", err)
	}
	return nil
}

// GetDeviceToken loads the token from the OS keyring.
func (k *KeyringStore) GetDeviceToken() (DeviceToken, error) {
	raw, err := keyring.Get(keyringService, DeviceTokenRef)
	if errors.Is(err, keyring.ErrNotFound) {
		return DeviceToken{}, ErrNoDeviceToken
	}
	if err != nil {
		return DeviceToken{}, fmt.Errorf("load device token from OS keyring: %w", err)
	}
	var t DeviceToken
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return DeviceToken{}, fmt.Errorf("decode OS keyring device token: %w", err)
	}
	if t.Token == "" {
		return DeviceToken{}, ErrNoDeviceToken
	}
	return t, nil
}

// DeleteDeviceToken removes the token; a missing entry is not an error.
func (k *KeyringStore) DeleteDeviceToken() error {
	err := keyring.Delete(keyringService, DeviceTokenRef)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// MemoryDeviceTokenStore is an in-process store for tests.
type MemoryDeviceTokenStore struct {
	mu    sync.Mutex
	token *DeviceToken
}

// SetDeviceToken stores t.
func (m *MemoryDeviceTokenStore) SetDeviceToken(t DeviceToken) error {
	if t.Token == "" || t.ControlPlaneURL == "" {
		return fmt.Errorf("incomplete device token")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = &t
	return nil
}

// GetDeviceToken returns the stored token or ErrNoDeviceToken.
func (m *MemoryDeviceTokenStore) GetDeviceToken() (DeviceToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token == nil {
		return DeviceToken{}, ErrNoDeviceToken
	}
	return *m.token, nil
}

// DeleteDeviceToken clears the store.
func (m *MemoryDeviceTokenStore) DeleteDeviceToken() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = nil
	return nil
}
