package keyring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// Second synthetic device for the version 2 fixture. Protects nothing.
const (
	goldenDeviceBID  = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	goldenDeviceBKey = "AGE-SECRET-KEY-1VS4RSCKH9SR69GZ3KNZEJFATML08MLNAY3TV79RAX2YAKVS87V6Q9AEM8G"
)

var goldenV2Path = filepath.Join("..", "..", "testdata", "keyring", "keyring.v2.json")

func goldenIdentities(t *testing.T) (a, b *age.X25519Identity) {
	t.Helper()
	a, err := age.ParseX25519Identity(goldenDeviceKey)
	if err != nil {
		t.Fatal(err)
	}
	b, err = age.ParseX25519Identity(goldenDeviceBKey)
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

// TestGoldenV2Fixture pins the version 2 object format: two generations,
// device B revoked by device A between them. Set KEYRING_WRITE_GOLDEN=1 to
// regenerate the fixture after a deliberate format change.
func TestGoldenV2Fixture(t *testing.T) {
	a, b := goldenIdentities(t)
	if os.Getenv("KEYRING_WRITE_GOLDEN") == "1" {
		t0 := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
		k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, t0)
		if err != nil {
			t.Fatal(err)
		}
		if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), t0.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, t0.Add(2*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		crypto.Zero(next)
		raw, err := k.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenV2Path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(goldenV2Path)
	if err != nil {
		t.Fatal(err)
	}
	k, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if k.SchemaVersion != 2 || k.CurrentGeneration != 2 || len(k.Generations) != 2 || k.DeviceCount() != 1 {
		t.Fatalf("unexpected fixture shape: version=%d current=%d gens=%d devices=%d", k.SchemaVersion, k.CurrentGeneration, len(k.Generations), k.DeviceCount())
	}
	for _, g := range k.Generations {
		if g.Recovery.Format != WrapFormatBound {
			t.Fatalf("generation %d recovery wrap is not bound", g.Number)
		}
		for _, d := range g.Devices {
			if d.Format != WrapFormatBound {
				t.Fatalf("generation %d device %s wrap is not bound", g.Number, d.DeviceID)
			}
		}
	}
	if k.Generations[0].Recipient != goldenRecipient {
		t.Fatalf("generation 1 recipient drifted: %s", k.Generations[0].Recipient)
	}
	revs := k.Revocations()
	if len(revs) != 1 || revs[0].DeviceID != goldenDeviceBID || revs[0].RevokedBy != goldenDeviceID || !k.RevokedDevice(goldenDeviceBID) || k.RevokedDevice(goldenDeviceID) {
		t.Fatalf("revocations: %+v", revs)
	}
	for _, secret := range []string{goldenRecoveryCode, goldenDeviceKey, goldenDeviceBKey, "0102030405060708"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("fixture leaks %q", secret)
		}
	}

	t.Run("A reads both generations", func(t *testing.T) {
		current, earlier, err := k.UnwrapForDevice(goldenDeviceID, a)
		if err != nil {
			t.Fatal(err)
		}
		if len(earlier) != 1 || !bytes.Equal(earlier[0], goldenRootKey()) {
			t.Fatalf("earlier generations: %d", len(earlier))
		}
		identity, _ := crypto.RootKeyIdentity(current)
		if identity.Recipient().String() != k.CurrentRecipient() {
			t.Fatal("current key does not match the recorded recipient")
		}
		if k.DeviceMembership(goldenDeviceID, a) != Enrolled {
			t.Fatal("A is not enrolled")
		}
	})
	t.Run("B reads only the old generation", func(t *testing.T) {
		_, _, err := k.UnwrapForDevice(goldenDeviceBID, b)
		if !errors.Is(err, ErrDeviceNotEnrolled) || errors.Is(err, ErrDeviceKeyMismatch) {
			t.Fatalf("revoked device: got %v", err)
		}
		if m := k.DeviceMembership(goldenDeviceBID, b); m != NotListed {
			t.Fatalf("membership %v", m)
		}
		old, err := unwrapDevice(k.generation(1), goldenDeviceBID, b, k.bindingFor(k.generation(1)))
		if err != nil || !bytes.Equal(old, goldenRootKey()) {
			t.Fatalf("B lost generation 1: %v", err)
		}
	})
	t.Run("recovery code opens the new generation", func(t *testing.T) {
		got, err := k.UnwrapWithRecoveryCode(goldenRecoveryCode)
		if err != nil {
			t.Fatal(err)
		}
		identity, _ := crypto.RootKeyIdentity(got)
		if identity.Recipient().String() != k.CurrentRecipient() {
			t.Fatal("recovery unwrapped a key that is not the current generation's")
		}
	})
}

// TestBoundWrapsRefuseTransplant: a version 2 wrap moved to another profile
// or generation does not open, which is the whole point of the binding.
func TestBoundWrapsRefuseTransplant(t *testing.T) {
	a, _ := goldenIdentities(t)
	raw, err := os.ReadFile(goldenV2Path)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("other profile", func(t *testing.T) {
		k, _ := Parse(raw)
		k.ProfileID = "99999999-2222-4333-8444-555555555555"
		if _, _, err := k.UnwrapForDevice(goldenDeviceID, a); err == nil || !strings.Contains(err.Error(), "bound to another profile or generation") {
			t.Fatalf("device wrap opened under another profile: %v", err)
		}
		if _, err := k.UnwrapWithRecoveryCode(goldenRecoveryCode); !errors.Is(err, ErrRecoveryMismatch) {
			t.Fatalf("recovery wrap opened under another profile: %v", err)
		}
	})
	t.Run("other generation", func(t *testing.T) {
		k, _ := Parse(raw)
		// Swap the generation numbers: each wrap is now claimed by the
		// other generation.
		k.Generations[0].Number, k.Generations[1].Number = 2, 1
		if _, _, err := k.UnwrapForDevice(goldenDeviceID, a); err == nil {
			t.Fatal("device wrap opened under another generation number")
		}
		if _, err := k.UnwrapWithRecoveryCode(goldenRecoveryCode); !errors.Is(err, ErrRecoveryMismatch) {
			t.Fatalf("recovery wrap opened under another generation: %v", err)
		}
	})
}

// TestLegacyKeyringMigratesOnWrite: a version 1 object is read as is, and
// the first write that holds the root key rebinds its device wraps and
// stamps version 2; the recovery wrap is rebound at the next rollover.
func TestLegacyKeyringMigratesOnWrite(t *testing.T) {
	k := loadGolden(t)
	if k.SchemaVersion != 1 || k.Generations[0].Devices[0].Format != WrapFormatLegacy || k.Generations[0].Recovery.Format != WrapFormatLegacy {
		t.Fatalf("v1 fixture is not legacy: %+v", k.Generations[0])
	}
	a, b := goldenIdentities(t)
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	raw, err := k.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if again.SchemaVersion != 2 {
		t.Fatalf("schema_version %d after write", again.SchemaVersion)
	}
	for _, d := range again.Generations[0].Devices {
		if d.Format != WrapFormatBound {
			t.Fatalf("device %s was not rebound", d.DeviceID)
		}
	}
	if again.Generations[0].Recovery.Format != WrapFormatLegacy {
		t.Fatal("recovery wrap cannot be rebound without the code")
	}
	for id, identity := range map[string]*age.X25519Identity{goldenDeviceID: a, goldenDeviceBID: b} {
		if current, _, err := again.UnwrapForDevice(id, identity); err != nil || !bytes.Equal(current, goldenRootKey()) {
			t.Fatalf("%s after migration: %v", id, err)
		}
	}
	if got, err := again.UnwrapWithRecoveryCode(goldenRecoveryCode); err != nil || !bytes.Equal(got, goldenRootKey()) {
		t.Fatalf("legacy recovery wrap after migration: %v", err)
	}
	next, err := again.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(next)
	if again.current().Recovery.Format != WrapFormatBound {
		t.Fatal("rollover did not bind the recovery wrap")
	}
	got, err := again.UnwrapWithRecoveryCode(goldenRecoveryCode)
	if err != nil || !bytes.Equal(got, next) {
		t.Fatalf("recovery after rollover: %v", err)
	}
	// A version 1 object claiming bound wraps is malformed.
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	obj["schema_version"] = 1
	bad, _ := json.Marshal(obj)
	if _, err := Parse(bad); err == nil {
		t.Fatal("accepted schema_version 1 with bound wraps")
	}
}

func TestParseRejectsMalformedGenerations(t *testing.T) {
	raw, err := os.ReadFile(goldenV2Path)
	if err != nil {
		t.Fatal(err)
	}
	mutate := func(f func(map[string]any)) []byte {
		var obj map[string]any
		_ = json.Unmarshal(raw, &obj)
		f(obj)
		out, _ := json.Marshal(obj)
		return out
	}
	gens := func(obj map[string]any) []any { return obj["generations"].([]any) }
	cases := map[string]struct {
		raw  []byte
		want string
	}{
		"duplicate generation": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["number"] = float64(1)
		}), "appears more than once"},
		"zero generation": {mutate(func(o map[string]any) {
			gens(o)[0].(map[string]any)["number"] = float64(0)
		}), "not positive"},
		"missing current": {mutate(func(o map[string]any) { o["current_generation"] = float64(7) }), "not present"},
		"duplicate device": {mutate(func(o map[string]any) {
			g := gens(o)[0].(map[string]any)
			devs := g["devices"].([]any)
			g["devices"] = append(devs, devs[0])
		}), "more than once"},
		"unknown wrap format": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["recovery"].(map[string]any)["format"] = float64(9)
		}), "unknown format"},
		"future schema": {mutate(func(o map[string]any) { o["schema_version"] = float64(3) }), "unsupported schema_version"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(tc.raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRolloverRefusals(t *testing.T) {
	a, b := goldenIdentities(t)
	fresh := func() *Keyring {
		k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
			t.Fatal(err)
		}
		return k
	}
	wrongCode, _ := GenerateRecoveryCode()
	stale, _ := crypto.NewRootKey()
	cases := map[string]struct {
		code    string
		key     []byte
		revoke  []string
		by      string
		wantErr error
	}{
		"wrong recovery code": {wrongCode, goldenRootKey(), []string{goldenDeviceBID}, goldenDeviceID, ErrRecoveryMismatch},
		"stale root key":      {goldenRecoveryCode, stale, []string{goldenDeviceBID}, goldenDeviceID, ErrStaleRootKey},
		"self":                {goldenRecoveryCode, goldenRootKey(), []string{goldenDeviceID}, goldenDeviceID, ErrSelfRevoke},
		"not listed":          {goldenRecoveryCode, goldenRootKey(), []string{"ghost"}, goldenDeviceID, ErrDeviceNotEnrolled},
		"revoker not listed":  {goldenRecoveryCode, goldenRootKey(), []string{goldenDeviceBID}, "ghost", ErrDeviceNotEnrolled},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			k := fresh()
			_, err := k.Rollover(tc.key, tc.code, tc.revoke, tc.by, time.Now())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
			if k.CurrentGeneration != 1 || len(k.Generations) != 1 {
				t.Fatal("a refused rollover changed the keyring")
			}
		})
	}
	// A successful rollover: the old key no longer enrols anyone, the new
	// one does, and the numbers never repeat even after two rollovers.
	k := fresh()
	next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	c, _ := age.GenerateX25519Identity()
	if err := k.Enrol(goldenRootKey(), "device-c", c.Recipient(), time.Now()); !errors.Is(err, ErrStaleRootKey) {
		t.Fatalf("old root key enrolled into the new generation: %v", err)
	}
	if err := k.Enrol(next, "device-c", c.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Rollover(next, goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now()); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("revoking twice: %v", err)
	}
	third, err := k.Rollover(next, goldenRecoveryCode, []string{"device-c"}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	crypto.Zero(third)
	if got := k.GenerationNumbers(); len(got) != 3 || got[2] != 3 || k.CurrentGeneration != 3 {
		t.Fatalf("generations %v current %d", got, k.CurrentGeneration)
	}
	current, earlier, err := k.UnwrapForDevice(goldenDeviceID, a)
	if err != nil || len(earlier) != 2 {
		t.Fatalf("A after two rollovers: %v earlier=%d", err, len(earlier))
	}
	crypto.Zero(current)
	if !bytes.Equal(earlier[1], goldenRootKey()) || !bytes.Equal(earlier[0], next) {
		t.Fatal("earlier generations are not newest first")
	}
}

// TestUnwrapToleratesReenrolledKeyInEarlierGeneration: a device revoked,
// then enrolled again under a fresh key, is listed in an old generation
// under the old key. That wrap is skipped, not an error.
func TestUnwrapToleratesReenrolledKeyInEarlierGeneration(t *testing.T) {
	a, bOld := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, bOld.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(next)
	bNew, _ := age.GenerateX25519Identity()
	if err := k.Enrol(next, goldenDeviceBID, bNew.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	current, earlier, err := k.UnwrapForDevice(goldenDeviceBID, bNew)
	if err != nil || !bytes.Equal(current, next) || len(earlier) != 0 {
		t.Fatalf("re-enrolled device: %v earlier=%d", err, len(earlier))
	}
	if _, _, err := k.UnwrapForDevice(goldenDeviceBID, bOld); !errors.Is(err, ErrDeviceKeyMismatch) || !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("old key against the re-enrolled id: %v", err)
	}
	if m := k.DeviceMembership(goldenDeviceBID, bOld); m != KeyMismatch {
		t.Fatalf("membership %v", m)
	}
	if m := k.DeviceMembership(goldenDeviceBID, nil); m != KeyGone {
		t.Fatalf("membership with no key %v", m)
	}
	if m := k.DeviceMembership("ghost", nil); m != NotListed {
		t.Fatalf("membership of unknown %v", m)
	}
}

// TestRolloverConvergesAgainstConcurrentEnrol: an approval lands between a
// revoking device's load and its conditional put. The CAS retry reloads,
// re-unwraps, and rolls over again, so the newly approved device gets a
// wrap in the new generation and nothing is lost.
func TestRolloverConvergesAgainstConcurrentEnrol(t *testing.T) {
	ctx := context.Background()
	a, b := goldenIdentities(t)
	disk, err := memory.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := Create(ctx, disk, ObjectName, k); err != nil {
		t.Fatal(err)
	}
	c, _ := age.GenerateX25519Identity()
	racing := &racingBackend{Backend: disk}
	racing.race = func() {
		if _, err := Update(ctx, disk, ObjectName, func(k *Keyring) error {
			current, _, err := k.UnwrapForDevice(goldenDeviceID, a)
			if err != nil {
				return err
			}
			defer crypto.Zero(current)
			return k.Enrol(current, "device-c", c.Recipient(), time.Now())
		}); err != nil {
			t.Errorf("competing approval: %v", err)
		}
	}
	attempts := 0
	updated, err := Update(ctx, racing, ObjectName, func(k *Keyring) error {
		attempts++
		current, _, err := k.UnwrapForDevice(goldenDeviceID, a)
		if err != nil {
			return err
		}
		defer crypto.Zero(current)
		next, err := k.Rollover(current, goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
		if err != nil {
			return err
		}
		crypto.Zero(next)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || updated.CurrentGeneration != 2 || len(updated.Generations) != 2 {
		t.Fatalf("attempts=%d current=%d gens=%d", attempts, updated.CurrentGeneration, len(updated.Generations))
	}
	stored, _, err := Load(ctx, disk, ObjectName)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.HasDevice("device-c") || stored.HasDevice(goldenDeviceBID) || stored.DeviceCount() != 2 {
		t.Fatalf("converged keyring lists %d devices, c=%v b=%v", stored.DeviceCount(), stored.HasDevice("device-c"), stored.HasDevice(goldenDeviceBID))
	}
	if _, _, err := stored.UnwrapForDevice("device-c", c); err != nil {
		t.Fatalf("device c lost its wrap across the rollover: %v", err)
	}
	// The same race the other way round: two devices revoke the same
	// device at once. The second sees it gone and reports that rather than
	// starting a third generation.
	racing2 := &racingBackend{Backend: disk}
	d, _ := age.GenerateX25519Identity()
	if _, err := Update(ctx, disk, ObjectName, func(k *Keyring) error {
		current, _, err := k.UnwrapForDevice(goldenDeviceID, a)
		if err != nil {
			return err
		}
		defer crypto.Zero(current)
		return k.Enrol(current, "device-d", d.Recipient(), time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	revokeD := func(k *Keyring) error {
		current, _, err := k.UnwrapForDevice(goldenDeviceID, a)
		if err != nil {
			return err
		}
		defer crypto.Zero(current)
		next, err := k.Rollover(current, goldenRecoveryCode, []string{"device-d"}, goldenDeviceID, time.Now())
		if err != nil {
			return err
		}
		crypto.Zero(next)
		return nil
	}
	racing2.race = func() {
		if _, err := Update(ctx, disk, ObjectName, revokeD); err != nil {
			t.Errorf("competing revocation: %v", err)
		}
	}
	if _, err := Update(ctx, racing2, ObjectName, revokeD); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("second revocation: %v", err)
	}
	stored, _, _ = Load(ctx, disk, ObjectName)
	if stored.CurrentGeneration != 3 || len(stored.Generations) != 3 {
		t.Fatalf("concurrent revocations produced %d generations (current %d)", len(stored.Generations), stored.CurrentGeneration)
	}
}

// TestLegacyWrapCannotBeReplayedIntoLaterGeneration mirrors the review probe
// for #11: a legacy (unbound) generation-1 wrap lifted into a later
// generation must not be accepted, or the holder would seal new pushes to a
// root key the revoked device still has. Three layers refuse it: Parse
// rejects unbound wraps outside generation 1, Rollover rebinds the outgoing
// generation so no legacy device wrap survives, and unwrapDevice checks the
// unwrapped key against the generation's recipient.
func TestLegacyWrapCannotBeReplayedIntoLaterGeneration(t *testing.T) {
	k := loadGolden(t)
	legacyA := k.Generations[0].Devices[0]
	if legacyA.Format != WrapFormatLegacy {
		t.Fatalf("fixture wrap for A is not legacy")
	}
	a, b := goldenIdentities(t)
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	// Undo Enrol's rebinding of A so the outgoing generation still holds
	// the legacy wrap when Rollover runs.
	k.Generations[0].Devices[0] = legacyA
	next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(next)
	for _, d := range k.Generations[0].Devices {
		if d.Format != WrapFormatBound {
			t.Fatalf("rollover left a legacy wrap for %s in the outgoing generation", d.DeviceID)
		}
	}

	// Transplant A's legacy generation-1 wrap into generation 2.
	cur := k.current()
	for i := range cur.Devices {
		if cur.Devices[i].DeviceID == goldenDeviceID {
			cur.Devices[i] = legacyA
		}
	}
	raw, err := k.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(raw); err == nil {
		t.Fatal("Parse accepted a legacy wrap in generation 2")
	}
	// Even bypassing Parse, the unwrapped key must match the recipient.
	current, _, err := k.UnwrapForDevice(goldenDeviceID, a)
	if err == nil {
		crypto.Zero(current)
		t.Fatal("UnwrapForDevice returned a key for a transplanted wrap")
	}
	if bytes.Equal(current, goldenRootKey()) {
		t.Fatal("generation-1 root key reported as current")
	}
}

// generationsListing lists the generation numbers whose device table lists
// id, in storage order.
func generationsListing(k *Keyring, id string) []int {
	var listed []int
	for _, g := range k.Generations {
		for _, d := range g.Devices {
			if d.DeviceID == id {
				listed = append(listed, g.Number)
				break
			}
		}
	}
	return listed
}

// TestUnenrolEverywhereSweepsAllGenerations: an approval enrols the joining
// device into every generation it can read (EnrolAll), so rolling back a
// refused approval must sweep every generation. Unenrol alone clears only
// the current one — the exact hole that would leave the refused device able
// to unwrap pre-revocation history.
func TestUnenrolEverywhereSweepsAllGenerations(t *testing.T) {
	a, b := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(next)

	joiner, _ := age.GenerateX25519Identity()
	if err := k.EnrolAll(map[int][]byte{1: goldenRootKey(), 2: next}, "joiner", joiner.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := generationsListing(k, "joiner"); len(got) != 2 {
		t.Fatalf("EnrolAll wrote into generations %v, want both", got)
	}

	// The key check still holds across generations: a different public
	// key, or none, removes nothing.
	other, _ := age.GenerateX25519Identity()
	if k.UnenrolEverywhere("joiner", other.Recipient().String()) || len(generationsListing(k, "joiner")) != 2 {
		t.Fatal("removed a wrap made for a different public key")
	}
	if k.UnenrolEverywhere("joiner", "") || len(generationsListing(k, "joiner")) != 2 {
		t.Fatal("removed a wrap for an empty key")
	}

	if !k.UnenrolEverywhere("joiner", joiner.Recipient().String()) {
		t.Fatal("did not remove the joiner's wraps")
	}
	if got := generationsListing(k, "joiner"); len(got) != 0 {
		t.Fatalf("generations %v still list the joiner after the sweep", got)
	}
	if _, err := k.UnwrapGenerations("joiner", joiner); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("swept device still opens the keyring: %v", err)
	}
	// The approving device's wraps are untouched: it still reads both
	// generations.
	keys, err := k.UnwrapGenerations(goldenDeviceID, a)
	if err != nil || len(keys) != 2 {
		t.Fatalf("approver disturbed: %v generations=%d", err, len(keys))
	}
	ZeroGenerations(keys)
	if k.UnenrolEverywhere("joiner", joiner.Recipient().String()) {
		t.Fatal("removed the same wraps twice")
	}
}
