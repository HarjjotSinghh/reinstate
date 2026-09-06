// Package schema defines versioned Reinstate data contracts.
package schema

import (
	"fmt"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
)

// ConfigSchemaVersion is the current config schema.
const ConfigSchemaVersion = 1

// Config is the v1 configuration document (TOML).
type Config struct {
	SchemaVersion         int                    `toml:"schema_version"`
	ProfileID             string                 `toml:"profile_id"`
	DeviceID              string                 `toml:"device_id"`
	RemoteProfileRequired bool                   `toml:"remote_profile_required"`
	Storage               StorageConfig          `toml:"storage"`
	Encryption            EncryptionConfig       `toml:"encryption"`
	Agents                map[string]AgentConfig `toml:"agents"`
	Projects              []ProjectConfig        `toml:"projects"`
	Restore               RestoreConfig          `toml:"restore"`
	Hop                   HopConfig              `toml:"hop,omitempty"`
}

// HopConfig points at the Reinstate Hop control plane. It holds no secret:
// the device token lives in the OS keyring.
type HopConfig struct {
	// URL overrides the production control plane (for staging or a local
	// hopd). REINSTATE_HOP_URL takes precedence over this value.
	URL string `toml:"url"`
}

// Active-agent policies for restore.
const (
	// ActiveAgentStrict refuses a restore whenever the agent runs anywhere on
	// the host. This was the only behavior before scoped detection existed.
	ActiveAgentStrict = "strict"
	// ActiveAgentScoped refuses only when the exact target session file is held
	// open by that agent. Unrelated agents in other projects are ignored.
	ActiveAgentScoped = "scoped"
	// ActiveAgentOff performs no liveness check. Restores stay atomic and still
	// back up the previous file first.
	ActiveAgentOff = "off"
	// ActiveAgentFork never blocks. When the target session is in use the
	// remote copy is restored alongside it as a distinct vendor-safe session,
	// leaving the live file untouched.
	ActiveAgentFork = "fork"
)

// DefaultActiveAgentPolicy restores a busy session alongside the live one
// instead of refusing, so a restore never waits on a human closing an agent.
const DefaultActiveAgentPolicy = ActiveAgentFork

// RestoreConfig tunes restore safety behavior.
type RestoreConfig struct {
	ActiveAgentPolicy string `toml:"active_agent_policy"`
}

// Storage backend types.
const (
	// StorageS3 is BYO storage: an S3-compatible bucket the user owns,
	// reached with keys from the OS keyring or environment.
	StorageS3 = "s3"
	// StorageHop is the hosted tier: the account's locker, reached with
	// hourly credentials minted by the control plane for the signed-in
	// device. Endpoint, bucket, and region come from the control plane, so
	// none of them is stored here.
	StorageHop = "hop"
)

// StorageConfig describes remote storage (no secrets).
type StorageConfig struct {
	Type          string `toml:"type"`
	Endpoint      string `toml:"endpoint"`
	Region        string `toml:"region"`
	Bucket        string `toml:"bucket"`
	Prefix        string `toml:"prefix"`
	CredentialRef string `toml:"credential_ref"`
}

// Encryption key models. The selection decides which crypto.KeyProvider
// every push and pull uses; nothing else in sync changes between them.
const (
	// EncryptionPassphrase is BYO storage: an age scrypt passphrase typed on
	// every device.
	EncryptionPassphrase = "age-scrypt"
	// EncryptionRootKey is the hosted-tier model: a root key generated on the
	// first device and carried by the keyring, never typed and never stored
	// in config.
	EncryptionRootKey = "root-key"
)

// EncryptionConfig selects client-side encryption.
type EncryptionConfig struct {
	Type string `toml:"type"`
}

// AgentConfig toggles an adapter.
type AgentConfig struct {
	Enabled bool `toml:"enabled"`
}

// ProjectConfig maps a canonical project id to a local root.
type ProjectConfig struct {
	ID        string `toml:"id"`
	LocalRoot string `toml:"local_root"`
}

// ForbiddenConfigKeys must never appear in config files.
var ForbiddenConfigKeys = []string{
	"password", "passphrase", "secret", "api_key", "apikey", "token",
	"access_key", "secret_key", "private_key", "auth", "credential",
}

// ValidateConfig checks schema version and required fields.
func ValidateConfig(c *Config) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if c.SchemaVersion != ConfigSchemaVersion {
		return fmt.Errorf("unsupported config schema_version %d (want %d)", c.SchemaVersion, ConfigSchemaVersion)
	}
	if c.ProfileID == "" || c.DeviceID == "" {
		return fmt.Errorf("profile_id and device_id are required")
	}
	if c.Storage.Type == "" {
		return fmt.Errorf("storage.type is required")
	}
	if c.Encryption.Type == "" {
		c.Encryption.Type = EncryptionPassphrase
	}
	switch c.Encryption.Type {
	case EncryptionPassphrase, EncryptionRootKey:
	default:
		return fmt.Errorf("unsupported encryption.type %q (want %q or %q)", c.Encryption.Type, EncryptionPassphrase, EncryptionRootKey)
	}
	if c.Restore.ActiveAgentPolicy == "" {
		c.Restore.ActiveAgentPolicy = DefaultActiveAgentPolicy
	}
	switch c.Restore.ActiveAgentPolicy {
	case ActiveAgentStrict, ActiveAgentScoped, ActiveAgentOff, ActiveAgentFork:
	default:
		return fmt.Errorf(
			"unsupported restore.active_agent_policy %q (want %q, %q, %q, or %q)",
			c.Restore.ActiveAgentPolicy,
			ActiveAgentFork, ActiveAgentScoped, ActiveAgentStrict, ActiveAgentOff)
	}
	return nil
}

// DefaultConfig returns a minimal valid config skeleton.
func DefaultConfig(profileID, deviceID string) *Config {
	return &Config{
		SchemaVersion: ConfigSchemaVersion,
		ProfileID:     profileID,
		DeviceID:      deviceID,
		Storage: StorageConfig{
			Type:   "s3",
			Region: "auto",
			Bucket: "reinstate",
		},
		Encryption: EncryptionConfig{Type: "age-scrypt"},
		Restore:    RestoreConfig{ActiveAgentPolicy: DefaultActiveAgentPolicy},
		Agents:     defaultEnabledAgents(),
		Projects:   nil,
	}
}

func defaultEnabledAgents() map[string]AgentConfig {
	enabled := map[string]AgentConfig{}
	for _, descriptor := range agents.Capable(agents.CapabilitySync) {
		enabled[descriptor.Key] = AgentConfig{Enabled: true}
	}
	return enabled
}
