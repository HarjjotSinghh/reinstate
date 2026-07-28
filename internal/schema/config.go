// Package schema defines versioned Reinstate data contracts.
package schema

import "fmt"

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
)

// DefaultActiveAgentPolicy scopes the check to the session being replaced.
const DefaultActiveAgentPolicy = ActiveAgentScoped

// RestoreConfig tunes restore safety behavior.
type RestoreConfig struct {
	ActiveAgentPolicy string `toml:"active_agent_policy"`
}

// StorageConfig describes remote storage (no secrets).
type StorageConfig struct {
	Type          string `toml:"type"`
	Endpoint      string `toml:"endpoint"`
	Region        string `toml:"region"`
	Bucket        string `toml:"bucket"`
	Prefix        string `toml:"prefix"`
	CredentialRef string `toml:"credential_ref"`
}

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
		c.Encryption.Type = "age-scrypt"
	}
	if c.Restore.ActiveAgentPolicy == "" {
		c.Restore.ActiveAgentPolicy = DefaultActiveAgentPolicy
	}
	switch c.Restore.ActiveAgentPolicy {
	case ActiveAgentStrict, ActiveAgentScoped, ActiveAgentOff:
	default:
		return fmt.Errorf(
			"unsupported restore.active_agent_policy %q (want %q, %q, or %q)",
			c.Restore.ActiveAgentPolicy, ActiveAgentStrict, ActiveAgentScoped, ActiveAgentOff)
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
		Agents: map[string]AgentConfig{
			"claude": {Enabled: true},
			"codex":  {Enabled: true},
		},
		Projects: nil,
	}
}
