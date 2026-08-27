package schema

import "fmt"

// AccountSchemaVersion is the local account.json format version.
//
// Version 2 added the two fields that make this record an anchor rather than
// a note: key_recipient and account_key. Version 1 carried neither, and a
// record without them silently skipped the anchor check — the one check that
// tells this account's keyring from a replacement. There is no upgrade in
// place: neither value can be recovered from the record itself, and taking
// them from the keyring being checked would anchor that keyring to itself.
const AccountSchemaVersion = 2

// AccountRemedy is what to do with an enrolment record this build cannot
// use. It is named in the validation error, because the way out is a real
// command and not an obvious one.
const AccountRemedy = "back it up and clear it with rein init --hop --force (rein init --force for BYO storage), then enrol this device again with rein account join or rein account recover"

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
	// recipient the keyring records for it. A keyring where that generation
	// is gone, or now names a different root key, was replaced rather than
	// appended to. It is not key material: the same recipient is published
	// in the keyring.
	KeyRecipient string `json:"key_recipient"`
	// AccountKey is the public half of the account signing key, which every
	// key generation in the keyring is signed under. Pinning it here is what
	// separates this account's keyring from one an attacker signed with a
	// key of its own: verification alone only proves a keyring is internally
	// consistent. It is not key material — the private half is never stored
	// anywhere and is re-derived from the recovery code when a generation
	// has to be written.
	AccountKey string `json:"account_key"`
	// ControlPlaneKeyGeneration is the highest key generation the control
	// plane has confirmed to this device, and ControlPlaneConfirmedAt is
	// when it last did. Both are absent until a control plane that carries
	// the floor has answered once.
	//
	// The live answer is what a command enforces; this copy is the fallback
	// for the one case where a command runs without one — a control plane
	// that does not serve the floor at all — so a deployment that stops
	// serving it cannot quietly drop the account back to generation 0. It
	// moves up only. Neither field is key material, and neither is
	// required: a record without them has never had a floor confirmed,
	// which is where every device starts.
	ControlPlaneKeyGeneration int    `json:"control_plane_key_generation,omitempty"`
	ControlPlaneConfirmedAt   string `json:"control_plane_confirmed_at,omitempty"`
	// RecoveryCodeConfirmed records that the recovery code was re-entered on
	// this device. It is a local boolean only and is never sent anywhere.
	RecoveryCodeConfirmed bool `json:"recovery_code_confirmed"`
	// EnrolledVia is "init" for the first device, "join" for one approved by
	// another device, or "recover" for one enrolled from the recovery code.
	EnrolledVia string `json:"enrolled_via"`
	EnrolledAt  string `json:"enrolled_at"`
}

// ValidateAccount checks the schema version and required fields. Every
// anchor field is required: a record missing one cannot check a keyring, and
// accepting it would hand an attacker the unanchored path for the price of
// deleting a field.
func ValidateAccount(a *Account) error {
	if a == nil {
		return fmt.Errorf("nil account")
	}
	if a.SchemaVersion != AccountSchemaVersion {
		return fmt.Errorf("unsupported account schema_version %d (want %d); this enrolment record cannot anchor the account's keyring: %s", a.SchemaVersion, AccountSchemaVersion, AccountRemedy)
	}
	if a.ProfileID == "" || a.DeviceID == "" {
		return fmt.Errorf("account profile_id and device_id are required")
	}
	if a.KeyGeneration < 1 {
		return fmt.Errorf("account key_generation must be at least 1")
	}
	if a.KeyRecipient == "" || a.AccountKey == "" {
		return fmt.Errorf("account key_recipient and account_key are required: without them this record cannot tell the account's keyring from a replacement; %s", AccountRemedy)
	}
	return nil
}
