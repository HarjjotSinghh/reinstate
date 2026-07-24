package config

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// SaveConfig writes config.toml atomically.
func SaveConfig(home string, c *schema.Config) error {
	if err := schema.ValidateConfig(c); err != nil {
		return err
	}
	b, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomic(ConfigPath(home), b, fsx.OwnerOnlyFilePerm)
}

// SaveState writes state.json atomically.
func SaveState(home string, s *schema.State) error {
	if s.Sessions == nil {
		s.Sessions = map[string]schema.SessionState{}
	}
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := schema.ValidateState(s); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return fsx.WriteFileAtomic(StatePath(home), b, fsx.OwnerOnlyFilePerm)
}

// EnsureLayout creates the Reinstate home directory tree.
func EnsureLayout(home string) error {
	subs := []string{"", "cache", "backups", "conflicts", "locks", "logs"}
	for _, s := range subs {
		p := home
		if s != "" {
			p = filepath.Join(home, s)
		}
		if err := fsx.EnsureOwnerOnlyDir(p); err != nil {
			return err
		}
	}
	return nil
}
