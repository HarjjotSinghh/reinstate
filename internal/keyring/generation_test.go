package keyring

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// Second synthetic device for the rolled-over fixture. Protects nothing.
const (
	goldenDeviceBID  = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	goldenDeviceBKey = "AGE-SECRET-KEY-1VS4RSCKH9SR69GZ3KNZEJFATML08MLNAY3TV79RAX2YAKVS87V6Q9AEM8G"
)

var goldenV2Path = filepath.Join("..", "..", "testdata", "keyring", "keyring.two-generations.json")

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

// TestGoldenRolledOverFixture pins the object format across a rollover: two
// generations, device B revoked by device A between them, both signed by the
// account key the recovery code derives. Set KEYRING_WRITE_GOLDEN=1 to
// regenerate the fixture after a deliberate format change.
func TestGoldenRolledOverFixture(t *testing.T) {
	a, b := goldenIdentities(t)
	if os.Getenv("KEYRING_WRITE_GOLDEN") == "1" {
		t0 := goldenStamp
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
	if k.SchemaVersion != SchemaVersion || k.CurrentGeneration != 2 || len(k.Generations) != 2 || k.DeviceCount() != 1 {
		t.Fatalf("unexpected fixture shape: version=%d current=%d gens=%d devices=%d", k.SchemaVersion, k.CurrentGeneration, len(k.Generations), k.DeviceCount())
	}
	// Both generations are signed, and both verify with no key of any kind
	// in the reader's hands.
	if k.Generations[0].Signature == "" || k.Generations[1].Signature == "" {
		t.Fatal("a generation in the fixture carries no signature")
	}
	if err := k.VerifyGenerations(""); err != nil {
		t.Fatalf("golden signatures do not verify: %v", err)
	}
	account, err := DeriveAccountKey(goldenProfileID, goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Zero()
	if account.Public() != k.AccountKey {
		t.Fatalf("the fixture's account key is not the one the recovery code derives: %s", k.AccountKey)
	}
	if err := k.VerifyGenerations(account.Public()); err != nil {
		t.Fatalf("golden signatures do not verify against the pinned account key: %v", err)
	}
	otherCode, _ := GenerateRecoveryCode()
	other, err := DeriveAccountKey(goldenProfileID, otherCode)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Zero()
	if err := k.VerifyGenerations(other.Public()); !errors.Is(err, ErrAccountKeyMismatch) {
		t.Fatalf("the fixture verified against a foreign account key: %v", err)
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

// TestEarlierSchemaVersionsAreNotRead is the cutover. Versions 1 and 2 had
// nothing tying one generation to the next, and version 3 tied them with a
// MAC under the previous generation's root key — which the device being
// revoked was, by definition, holding. Reading any of them would keep the
// hole open, so all three are refused outright rather than upgraded in
// place.
func TestEarlierSchemaVersionsAreNotRead(t *testing.T) {
	raw, err := os.ReadFile(goldenV2Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []float64{1, 2, 3} {
		var obj map[string]any
		_ = json.Unmarshal(raw, &obj)
		obj["schema_version"] = version
		bad, _ := json.Marshal(obj)
		_, err := Parse(bad)
		if err == nil || !strings.Contains(err.Error(), "unsupported schema_version") {
			t.Fatalf("schema_version %v: got %v", version, err)
		}
		if !strings.Contains(err.Error(), "did not authenticate their key generations") {
			t.Fatalf("the refusal does not say why version %v is refused: %v", version, err)
		}
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
		"unbound device wrap": {mutate(func(o map[string]any) {
			delete(gens(o)[0].(map[string]any)["devices"].([]any)[0].(map[string]any), "format")
		}), "unknown format"},
		"future schema": {mutate(func(o map[string]any) { o["schema_version"] = float64(SchemaVersion + 1) }), "unsupported schema_version"},
		// A missing, short, or unreadable signature, an account key that
		// cannot hold one, or a gap in the numbering: each leaves a
		// generation no reader could ever authenticate.
		"missing signature": {mutate(func(o map[string]any) {
			delete(gens(o)[1].(map[string]any), "signature")
		}), "0-byte signature"},
		"short signature": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["signature"] = "AAAA"
		}), "3-byte signature"},
		"signature that is not base64": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["signature"] = "not base64!"
		}), "not valid base64"},
		"missing signature on the first generation": {mutate(func(o map[string]any) {
			delete(gens(o)[0].(map[string]any), "signature")
		}), "0-byte signature"},
		"no account key": {mutate(func(o map[string]any) {
			delete(o, "account_key")
		}), "carries no account_key"},
		"account key of the wrong length": {mutate(func(o map[string]any) {
			o["account_key"] = base64.StdEncoding.EncodeToString(make([]byte, 16))
		}), "account_key is 16 bytes"},
		"account key that is not base64": {mutate(func(o map[string]any) {
			o["account_key"] = "not base64!"
		}), "account_key is not valid base64"},
		"gap in the numbering": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["number"] = float64(3)
			o["current_generation"] = float64(3)
		}), "generation 2 is missing"},
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

// forgeGeneration appends the generation a party with write access to the
// bucket, but no recovery code, can build from the keyring alone: a fresh
// root key of its own, one age wrap per listed device sealed to that
// device's published public key, and a recovery wrap it cannot make open (it
// does not have the code). It is the exact object the 2026-08-27
// verification round used to take over an account; only the signature is
// beyond it.
func forgeGeneration(t *testing.T, k *Keyring, signature string) *Keyring {
	t.Helper()
	g := k.current()
	attackerKey, err := crypto.NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := crypto.RootKeyIdentity(attackerKey)
	if err != nil {
		t.Fatal(err)
	}
	next := Generation{
		Number:    k.maxGeneration() + 1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Recipient: identity.Recipient().String(),
		Recovery:  g.Recovery,
		Signature: signature,
	}
	bind := binding{profileID: k.ProfileID, generation: next.Number}
	for _, d := range g.Devices {
		recipient, err := age.ParseX25519Recipient(d.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		wrap, err := wrapForDevice(attackerKey, d.DeviceID, recipient, time.Now(), bind)
		if err != nil {
			t.Fatal(err)
		}
		next.Devices = append(next.Devices, wrap)
	}
	forged := *k
	forged.Generations = append(append([]Generation(nil), k.Generations...), next)
	forged.CurrentGeneration = next.Number
	return &forged
}

// rolledOverPair is a keyring at generation 2 with device B revoked, plus
// generation 2's root key. Every forgery probe starts from that state.
func rolledOverPair(t *testing.T) (*Keyring, []byte) {
	t.Helper()
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
	return k, next
}

// TestForgedGenerationIsRefused is the blocker probe from the 2026-08-27
// verification round. Everything the forgery needs is public — the profile
// id, the generation numbers, the account key, and every device's public
// key — so the object is well formed and the wraps open. What it cannot
// produce is the signature, which is under a key only the recovery code
// derives.
//
// Every case here is refused by Parse itself, before any caller has a chance
// to forget to check: a keyring holding one generation that does not verify
// does not parse at all.
func TestForgedGenerationIsRefused(t *testing.T) {
	a, _ := goldenIdentities(t)
	k, next := rolledOverPair(t)
	defer crypto.Zero(next)

	// A signature made under a key the attacker chose itself, over the
	// header it wants: what anyone would try after reading this package.
	otherCode, err := GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	attacker, err := DeriveAccountKey(goldenProfileID, otherCode)
	if err != nil {
		t.Fatal(err)
	}
	defer attacker.Zero()
	self := forgeGeneration(t, k, "")
	selfSigned := attacker.sign(self.generationMessage(self.current(), self.generation(2)))

	signatures := map[string]string{
		"no signature at all":                    "",
		"a signature of the right length":        base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
		"the previous generation's signature":    k.current().Signature,
		"a signature under the attacker's key":   selfSigned,
		"the first generation's signature moved": k.generation(1).Signature,
	}
	for name, signature := range signatures {
		t.Run(name, func(t *testing.T) {
			forged := forgeGeneration(t, k, signature)
			raw, err := json.Marshal(forged)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(raw); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("Parse accepted a forged generation: %v", err)
			}
			if err := forged.VerifyGenerations(""); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("VerifyGenerations accepted a forged generation: %v", err)
			}
			// And nothing about the wrap gave it away: a reader that
			// skipped the signature would have opened it happily.
			if _, err := forged.UnwrapGenerations(goldenDeviceID, a); err != nil {
				t.Fatalf("the forged wrap did not even open, so the probe proves nothing: %v", err)
			}
			// The recovery code cannot be talked into it either.
			if _, err := forged.UnwrapWithRecoveryCode(goldenRecoveryCode); !errors.Is(err, ErrRecoveryMismatch) {
				t.Fatalf("the recovery code opened a forged generation: %v", err)
			}
		})
	}

	// The genuine keyring still verifies, with and without a pinned key.
	if err := k.VerifyGenerations(""); err != nil {
		t.Fatalf("the genuine keyring does not verify: %v", err)
	}
	account, err := DeriveAccountKey(goldenProfileID, goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Zero()
	if err := k.VerifyGenerations(account.Public()); err != nil {
		t.Fatalf("the genuine keyring does not verify against its own account key: %v", err)
	}
}

// TestARevokedDevicesRootKeyCannotSignAGeneration is the second finding from
// the re-attack, and the reason the primitive is a signature rather than a
// MAC keyed by the previous generation's root key.
//
// The device revoked at the N -> N+1 rollover is exactly the device that
// held generation N's root key — that is why it is being revoked — and for
// the rest of its credential TTL it can still write the keyring object. So
// it holds every input a root-key-keyed construction would have used. Here
// it holds generation 1's root key and the published object, and none of it
// produces a generation the account will adopt.
func TestARevokedDevicesRootKeyCannotSignAGeneration(t *testing.T) {
	k, next := rolledOverPair(t)
	defer crypto.Zero(next)
	held := goldenRootKey()
	forged := forgeGeneration(t, k, "")
	message := forged.generationMessage(forged.current(), forged.generation(2))
	attempts := map[string]string{
		"the held root key as an ed25519 seed": base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(held), message)),
		"the held root key repeated":           base64.StdEncoding.EncodeToString(append(append([]byte(nil), held...), held...)),
	}
	for name, signature := range attempts {
		t.Run(name, func(t *testing.T) {
			attempt := forgeGeneration(t, k, signature)
			raw, err := json.Marshal(attempt)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(raw); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("a generation signed with a root key was accepted: %v", err)
			}
		})
	}
	// It cannot roll the keyring over through the package either: Rollover
	// needs the recovery code, which no device ever holds.
	wrongCode, err := GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Rollover(next, wrongCode, []string{goldenDeviceID}, goldenDeviceBID, time.Now()); !errors.Is(err, ErrRecoveryMismatch) {
		t.Fatalf("Rollover without the recovery code: %v", err)
	}

	// The exact move that worked against version 3: rather than append,
	// the revoked device writes its *own* generation 2 over the real one —
	// its own root key, wrapped to the remaining device's published public
	// key — and authenticates it with generation 1's root key, which it
	// held right up to the rollover.
	t.Run("generation 2 replaced by the revoked device", func(t *testing.T) {
		a, _ := goldenIdentities(t)
		attackerKey, err := crypto.NewRootKey()
		if err != nil {
			t.Fatal(err)
		}
		defer crypto.Zero(attackerKey)
		identity, err := crypto.RootKeyIdentity(attackerKey)
		if err != nil {
			t.Fatal(err)
		}
		replaced := *k
		replaced.Generations = append([]Generation(nil), k.Generations[:1]...)
		own := Generation{
			Number:    2,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Recipient: identity.Recipient().String(),
			Recovery:  k.generation(1).Recovery,
		}
		wrap, err := wrapForDevice(attackerKey, goldenDeviceID, a.Recipient(), time.Now(), binding{profileID: k.ProfileID, generation: 2})
		if err != nil {
			t.Fatal(err)
		}
		own.Devices = []DeviceWrap{wrap}
		own.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(held), replaced.generationMessage(&own, replaced.generation(1))))
		replaced.Generations = append(replaced.Generations, own)
		replaced.CurrentGeneration = 2
		raw, err := replaced.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(raw); !errors.Is(err, ErrUnauthenticatedGeneration) {
			t.Fatalf("a generation the revoked device wrote under the root key it held was accepted: %v", err)
		}
	})
}

// TestVerificationNeedsNoReaderKeys is the first finding from the re-attack.
// The version 3 chain could be made *uncheckable* by deleting the reader's
// wrap in the previous generation, and an unchecked link was skipped, so a
// keyring was accepted whenever its current generation happened to check
// out. A signature has no such degree of freedom: verification needs only
// the account's public key, which the object publishes and every enrolled
// device pins. A reader holding nothing at all still refuses, and refuses
// the whole object rather than the one generation it cannot trace.
func TestVerificationNeedsNoReaderKeys(t *testing.T) {
	k, next := rolledOverPair(t)
	defer crypto.Zero(next)
	if err := k.VerifyGenerations(""); err != nil {
		t.Fatalf("a keyless reader cannot verify a genuine keyring: %v", err)
	}
	// Strip every device wrap from generation 1 — the exact move that made
	// the version 3 link uncheckable — and the verdict does not move.
	stripped := *k
	stripped.Generations = append([]Generation(nil), k.Generations...)
	stripped.Generations[0].Devices = nil
	if err := stripped.VerifyGenerations(""); err != nil {
		t.Fatalf("removing generation 1's wraps changed the verdict: %v", err)
	}
	// Now break a generation that is not the current one: a reader that
	// accepted a keyring because the current generation checked out would
	// pass this, and nothing may.
	for _, n := range []int{1, 2} {
		t.Run(fmt.Sprintf("generation %d unsigned", n), func(t *testing.T) {
			broken := *k
			broken.Generations = append([]Generation(nil), k.Generations...)
			broken.Generations[n-1].Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
			if err := broken.VerifyGenerations(""); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("generation %d was accepted with a zero signature: %v", n, err)
			}
			raw, err := json.Marshal(&broken)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(raw); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("Parse accepted a keyring whose generation %d does not verify: %v", n, err)
			}
		})
	}

	// The version 3 repro in full. Generation 3 is forged with no usable
	// signature; generation 4 is appended on top of it so that generation 3
	// is no longer current; and the reader's wrap in generation 2 is
	// deleted, so the reader holds no key for the generation before the
	// forged one. Under version 3 that combination returned nil. Here it
	// makes no difference: the reader's keys were never an input.
	t.Run("the version 3 repro", func(t *testing.T) {
		a, _ := goldenIdentities(t)
		forged := *k
		forged.Generations = append([]Generation(nil), k.Generations...)
		for _, number := range []int{3, 4} {
			attackerKey, err := crypto.NewRootKey()
			if err != nil {
				t.Fatal(err)
			}
			identity, err := crypto.RootKeyIdentity(attackerKey)
			if err != nil {
				t.Fatal(err)
			}
			g := Generation{
				Number:    number,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				Recipient: identity.Recipient().String(),
				Recovery:  forged.generation(number - 1).Recovery,
				Signature: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
			}
			wrap, err := wrapForDevice(attackerKey, goldenDeviceID, a.Recipient(), time.Now(), binding{profileID: forged.ProfileID, generation: number})
			if err != nil {
				t.Fatal(err)
			}
			crypto.Zero(attackerKey)
			g.Devices = []DeviceWrap{wrap}
			forged.Generations = append(forged.Generations, g)
			forged.CurrentGeneration = number
		}
		forged.Generations[1].Devices = nil
		raw, err := forged.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(raw); !errors.Is(err, ErrUnauthenticatedGeneration) {
			t.Fatalf("the version 3 repro was accepted: %v", err)
		}
	})
}

// TestPinnedAccountKeyRefusesAReplacedKeyring: a party with write access can
// build a keyring that verifies perfectly — under its own account key. Only
// a value kept outside the object separates that from the account's own, and
// that value is the pinned account key.
func TestPinnedAccountKeyRefusesAReplacedKeyring(t *testing.T) {
	a, _ := goldenIdentities(t)
	genuine, next := rolledOverPair(t)
	defer crypto.Zero(next)
	account, err := DeriveAccountKey(goldenProfileID, goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Zero()

	attackerKey, err := crypto.NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(attackerKey)
	attackerCode, err := GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	attackerDevice, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := New(goldenProfileID, attackerKey, attackerCode, "attacker", attackerDevice, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := age.ParseX25519Recipient(genuine.DevicePublicKey(goldenDeviceID))
	if err != nil {
		t.Fatal(err)
	}
	if err := replacement.Enrol(attackerKey, goldenDeviceID, recipient, time.Now()); err != nil {
		t.Fatal(err)
	}
	raw, err := replacement.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	// It parses: it is internally consistent, and its wrap opens for the
	// device the genuine keyring listed.
	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("the replacement did not even parse, so the probe proves nothing: %v", err)
	}
	if _, err := parsed.UnwrapGenerations(goldenDeviceID, a); err != nil {
		t.Fatalf("the replacement's wrap did not open: %v", err)
	}
	// And it is refused the moment an anchor is supplied.
	if err := parsed.VerifyGenerations(account.Public()); !errors.Is(err, ErrAccountKeyMismatch) {
		t.Fatalf("a keyring signed by another account key was accepted: %v", err)
	}
}

// TestSignaturesSurviveManyRolloversAndRecovery: every generation must
// still verify at depth, and the recovery code must still enrol a device
// after several rollovers. A scheme that only verifies at depth 1, or that
// the recovery code cannot re-sign under, would brick the account rather
// than protect it.
func TestSignaturesSurviveManyRolloversAndRecovery(t *testing.T) {
	a, _ := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	current := goldenRootKey()
	for i := range 8 {
		victim, _ := age.GenerateX25519Identity()
		id := fmt.Sprintf("victim-%d", i)
		if err := k.Enrol(current, id, victim.Recipient(), time.Now()); err != nil {
			t.Fatal(err)
		}
		next, err := k.Rollover(current, goldenRecoveryCode, []string{id}, goldenDeviceID, time.Now())
		if err != nil {
			t.Fatalf("rollover %d: %v", i, err)
		}
		crypto.Zero(current)
		current = next
	}
	crypto.Zero(current)
	if k.CurrentGeneration != 9 {
		t.Fatalf("current generation %d after eight rollovers", k.CurrentGeneration)
	}
	raw, err := k.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	stored, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Nine generations, every one of them signed and verifying — with no
	// key in the reader's hands, and against the pinned account key.
	account, err := DeriveAccountKey(goldenProfileID, goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer account.Zero()
	if err := stored.VerifyGenerations(account.Public()); err != nil {
		t.Fatalf("signatures at depth 9: %v", err)
	}
	keys, err := stored.UnwrapGenerations(goldenDeviceID, a)
	if err != nil || len(keys) != 9 {
		t.Fatalf("A holds %d generations: %v", len(keys), err)
	}
	defer ZeroGenerations(keys)
	// A device enrolling from the recovery code after all of it lands in
	// every generation and still reads the whole locker.
	fromCode, err := stored.UnwrapGenerationsWithRecoveryCode(goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroGenerations(fromCode)
	fresh, _ := age.GenerateX25519Identity()
	appended, err := stored.EnrolAll(fromCode, "recovered", fresh.Recipient(), time.Now())
	if err != nil {
		t.Fatalf("recovery after eight rollovers: %v", err)
	}
	if len(appended) != 9 {
		t.Fatalf("recovery enrolled into %d generations, want 9", len(appended))
	}
	recovered, err := stored.UnwrapGenerations("recovered", fresh)
	if err != nil || len(recovered) != 9 {
		t.Fatalf("the recovered device holds %d generations: %v", len(recovered), err)
	}
	defer ZeroGenerations(recovered)
	// Enrolling appended wraps, which no signature covers; every
	// generation must still verify afterwards.
	if err := stored.VerifyGenerations(account.Public()); err != nil {
		t.Fatalf("enrolling a recovered device broke the signatures: %v", err)
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

// TestRollbackRemovesOnlyWhatTheApprovalWrote is the interleaving the
// 2026-08-27 verification round found: two approvals of the same joining
// device carry the same public key, because the device generates its key
// once, before its first request. Removing every (device, public key) match
// in every generation therefore takes back wraps this approval never wrote.
//
// The shape here is the one that actually occurs. A device joins and is
// enrolled into both generations; it is revoked, so generation 3 starts
// without it while its pre-revocation wraps stay where they are; it asks to
// join again with the key it still holds. The second approval appends into
// generation 3 only — generations 1 and 2 already list that key, and
// EnrolInto leaves them alone. When the relay then refuses, only generation
// 3's wrap may be taken back.
func TestRollbackRemovesOnlyWhatTheApprovalWrote(t *testing.T) {
	a, _ := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	joiner, _ := age.GenerateX25519Identity()
	if err := k.Enrol(goldenRootKey(), "joiner", joiner.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	spare, _ := age.GenerateX25519Identity()
	if err := k.Enrol(goldenRootKey(), "spare", spare.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	gen2, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{"spare"}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(gen2)
	gen3, err := k.Rollover(gen2, goldenRecoveryCode, []string{"joiner"}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(gen3)
	if got := generationsListing(k, "joiner"); len(got) != 2 {
		t.Fatalf("the revoked joiner is listed in generations %v, want 1 and 2", got)
	}

	// The re-join. Its wrap goes into generation 3 and nowhere else.
	keys := map[int][]byte{1: goldenRootKey(), 2: gen2, 3: gen3}
	appended, err := k.EnrolAll(keys, "joiner", joiner.Recipient(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(appended) != 1 || appended[0].Generation != 3 {
		t.Fatalf("EnrolAll reported %+v, want generation 3 alone", appended)
	}
	if got := generationsListing(k, "joiner"); len(got) != 3 {
		t.Fatalf("after the re-join the joiner is listed in %v", got)
	}

	// The relay refuses. Generation 3's wrap goes; 1 and 2 stay, because
	// this approval did not write them.
	if !k.UnenrolAppended("joiner", appended) {
		t.Fatal("the rollback removed nothing")
	}
	if got := generationsListing(k, "joiner"); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("the rollback left the joiner in %v, want exactly 1 and 2", got)
	}
	if _, err := k.UnwrapGenerations("joiner", joiner); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("the refused device still opens the current generation: %v", err)
	}
	// The approving device is untouched and still reads everything.
	approver, err := k.UnwrapGenerations(goldenDeviceID, a)
	if err != nil || len(approver) != 3 {
		t.Fatalf("approver disturbed: %v generations=%d", err, len(approver))
	}
	ZeroGenerations(approver)
	if k.UnenrolAppended("joiner", appended) {
		t.Fatal("removed the same wrap twice")
	}
}

// TestRollbackSweepsEveryGenerationItWrote: the other half of the same
// contract. A first-time joiner is enrolled into every generation the
// approver can read, so a refused approval must take all of them back —
// clearing only the current one would leave the refused device able to
// unwrap pre-revocation history.
func TestRollbackSweepsEveryGenerationItWrote(t *testing.T) {
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
	appended, err := k.EnrolAll(map[int][]byte{1: goldenRootKey(), 2: next}, "joiner", joiner.Recipient(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(appended) != 2 || appended[0].Generation != 1 || appended[1].Generation != 2 {
		t.Fatalf("EnrolAll reported %+v, want both generations", appended)
	}
	if !k.UnenrolAppended("joiner", appended) {
		t.Fatal("did not remove the joiner's wraps")
	}
	if got := generationsListing(k, "joiner"); len(got) != 0 {
		t.Fatalf("generations %v still list the joiner after the rollback", got)
	}
	if _, err := k.UnwrapGenerations("joiner", joiner); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("swept device still opens the keyring: %v", err)
	}
}
