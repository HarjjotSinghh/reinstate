package schema

import "fmt"

// AccountSchemaVersion is the local account.json format version.
const AccountSchemaVersion = 1

// Account is the local, per-device record of hosted-tier enrolment. It holds
// no key material: the device key lives in the OS keyring and the root key
// only ever exists unwrapped in memory. Everything here is safe to print.
type Account struct {
	SchemaVersion int    `json:"schema_version"`
	ProfileID     string `json:"profile_id"`
	DeviceID      string `json:"device_id"`
	// KeyGeneration is the key generation this device last unwrapped.
	KeyGeneration int `json:"key_generation"`
	// KeyRecipient is the public half of that generation's root key: the age
	// recipient the keyring records for it. It is the local anchor the
	// keyring's generation chain hangs from — the chain proves that each
	// generation was written by a holder of the one before it, but only a
	// value kept outside the object can tell the account's own keyring from
	// a self-consistent replacement of the whole thing. It is not key
	// material: the same recipient is published in the keyring.
	KeyRecipient string `json:"key_recipient,omitempty"`
	// RecoveryCodeConfirmed records that the recovery code was re-entered on
	// this device. It is a local boolean only and is never sent anywhere.
	RecoveryCodeConfirmed bool `json:"recovery_code_confirmed"`
	// EnrolledVia is "init" for the first device or "recover" for a device
	// enrolled from the recovery code.
	EnrolledVia string `json:"enrolled_via"`
	EnrolledAt  string `json:"enrolled_at"`
}

// ValidateAccount checks the schema version and required fields.
func ValidateAccount(a *Account) error {
	if a == nil {
		return fmt.Errorf("nil account")
	}
	if a.SchemaVersion != AccountSchemaVersion {
		return fmt.Errorf("unsupported account schema_version %d (want %d)", a.SchemaVersion, AccountSchemaVersion)
	}
	if a.ProfileID == "" || a.DeviceID == "" {
		return fmt.Errorf("account profile_id and device_id are required")
	}
	if a.KeyGeneration < 1 {
		return fmt.Errorf("account key_generation must be at least 1")
	}
	return nil
}
