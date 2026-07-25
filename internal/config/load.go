package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// LoadConfig reads and validates config.toml from home.
func LoadConfig(home string) (*schema.Config, error) {
	path := ConfigPath(home)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Reject secret-looking keys in raw TOML before decode.
	lower := strings.ToLower(string(b))
	for _, k := range schema.ForbiddenConfigKeys {
		// Match key = value style assignments.
		if strings.Contains(lower, k+" ") || strings.Contains(lower, k+"=") || strings.Contains(lower, k+" =") {
			// Allow credential_ref which is a name, not a secret value.
			if k == "credential" && strings.Contains(lower, "credential_ref") {
				continue
			}
			if k == "secret" && strings.Contains(lower, "secret_key") {
				return nil, fmt.Errorf("config must not contain secret fields (%s)", k)
			}
			if k == "credential" {
				continue
			}
			// stricter: only flag if it's a left-hand key
			if containsForbiddenKey(string(b), k) {
				return nil, fmt.Errorf("config must not contain secret fields (%s)", k)
			}
		}
	}
	var c schema.Config
	if err := toml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := schema.ValidateConfig(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func containsForbiddenKey(raw, key string) bool {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// strip section headers
		if strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.ToLower(parts[0]))
		if k == key || strings.HasSuffix(k, "."+key) {
			if k == "credential_ref" {
				continue
			}
			return true
		}
	}
	return false
}

// LoadState reads and validates state.json.
func LoadState(home string) (*schema.State, error) {
	path := StatePath(home)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Check version for migration errors.
	var probe map[string]any
	if err := json.Unmarshal(b, &probe); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	ver := 0
	switch v := probe["schema_version"].(type) {
	case float64:
		ver = int(v)
	case int:
		ver = v
	}
	if ver != 0 && ver != schema.StateSchemaVersion {
		_, err := schema.MigrateState(ver, probe)
		return nil, err
	}
	var s schema.State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if err := schema.ValidateState(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
