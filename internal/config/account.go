package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// AccountPath is home/account.json, the local hosted-tier enrolment record.
func AccountPath(home string) string { return filepath.Join(home, "account.json") }

// LoadAccount reads and validates account.json. A missing file returns an
// error satisfying os.IsNotExist so callers can distinguish "not enrolled".
func LoadAccount(home string) (*schema.Account, error) {
	b, err := os.ReadFile(AccountPath(home))
	if err != nil {
		return nil, err
	}
	var a schema.Account
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	if err := schema.ValidateAccount(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// SaveAccount writes account.json atomically with owner-only permissions.
func SaveAccount(home string, a *schema.Account) error {
	if err := schema.ValidateAccount(a); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return fsx.WriteFileAtomic(AccountPath(home), b, fsx.OwnerOnlyFilePerm)
}
