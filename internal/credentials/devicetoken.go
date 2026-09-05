package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	keyring "github.com/zalando/go-keyring"
)

// DeviceTokenRef is the OS keyring entry holding the device token of the
// default Reinstate home. A home selected with REINSTATE_HOME gets its own
// entry (see DeviceTokenEntry), so two homes on one machine — two accounts,
// or the two device identities of an acceptance lab — never share a slot.
const DeviceTokenRef = "hop/device-token"

// deviceTokenHomeEnv is the same override internal/config honours for the
// home directory. It is read here, rather than through config, because a
// token must land in the entry of the home that signed in even when no
// config has been written yet (rein login runs before rein init).
const deviceTokenHomeEnv = "REINSTATE_HOME"

// DeviceTokenEntry returns the OS keyring entry for the home this process
// uses: DeviceTokenRef for the default home, and a per-home entry derived
// from REINSTATE_HOME otherwise.
func DeviceTokenEntry() string {
	return deviceTokenEntry(os.Getenv(deviceTokenHomeEnv))
}

func deviceTokenEntry(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return DeviceTokenRef
	}
	home = filepath.Clean(home)
	if runtime.GOOS == "windows" {
		home = strings.ToLower(home)
	}
	sum := sha256.Sum256([]byte(home))
	return DeviceTokenRef + "@" + hex.EncodeToString(sum[:8])
}

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
	if err := keyring.Set(keyringService, DeviceTokenEntry(), string(raw)); err != nil {
		return fmt.Errorf("store device token in OS keyring: %w", err)
	}
	return nil
}

// GetDeviceToken loads the token from the OS keyring.
func (k *KeyringStore) GetDeviceToken() (DeviceToken, error) {
	raw, err := keyring.Get(keyringService, DeviceTokenEntry())
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
	err := keyring.Delete(keyringService, DeviceTokenEntry())
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
