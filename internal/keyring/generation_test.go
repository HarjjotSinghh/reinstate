package keyring

import (
	"bytes"
	"context"
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

// Second synthetic device for the version 2 fixture. Protects nothing.
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
// generations, device B revoked by device A between them, generation 2
// chained to generation 1. Set KEYRING_WRITE_GOLDEN=1 to regenerate the
// fixture after a deliberate format change.
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
	if k.Generations[0].Chain != "" || k.Generations[1].Chain == "" {
		t.Fatalf("chain shape: gen1=%q gen2=%q", k.Generations[0].Chain, k.Generations[1].Chain)
	}
	// The chain checks out against generation 1's root key, and only that.
	if err := k.VerifyChain(map[int][]byte{1: goldenRootKey()}); err != nil {
		t.Fatalf("golden chain does not verify: %v", err)
	}
	wrongKey, _ := crypto.NewRootKey()
	if err := k.VerifyChain(map[int][]byte{1: wrongKey}); !errors.Is(err, ErrUnauthenticatedGeneration) {
		t.Fatalf("chain verified under a foreign key: %v", err)
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

// TestEarlierSchemaVersionsAreNotRead is the cutover: version 1 and version
// 2 objects had no chain between their generations, so a party with write
// access to the bucket could append a generation of its own and every device
// would adopt it. Reading them at all would keep that hole open, so they are
// refused outright rather than upgraded in place.
func TestEarlierSchemaVersionsAreNotRead(t *testing.T) {
	raw, err := os.ReadFile(goldenV2Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []float64{1, 2} {
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
		// A generation past the first with no chain, a chain of the wrong
		// length, a chain on generation 1, or a gap in the numbering: each
		// leaves a generation no reader could ever authenticate.
		"missing chain": {mutate(func(o map[string]any) {
			delete(gens(o)[1].(map[string]any), "chain")
		}), "0-byte chain"},
		"short chain": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["chain"] = "AAAA"
		}), "3-byte chain"},
		"chain that is not base64": {mutate(func(o map[string]any) {
			gens(o)[1].(map[string]any)["chain"] = "not base64!"
		}), "not valid base64"},
		"chain on the first generation": {mutate(func(o map[string]any) {
			gens(o)[0].(map[string]any)["chain"] = gens(o)[1].(map[string]any)["chain"]
		}), "no generation before it"},
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
// bucket, but no root key, can build from the keyring alone: a fresh root
// key of its own, one age wrap per listed device sealed to that device's
// published public key, and a recovery wrap it cannot make open (it does not
// have the code). It is the exact object the 2026-08-27 verification round
// used to take over an account.
func forgeGeneration(t *testing.T, k *Keyring, chain string) *Keyring {
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
		Chain:     chain,
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

// TestForgedGenerationIsRefused is the blocker probe from the 2026-08-27
// verification round. Everything the forgery needs is public — the profile
// id, the generation numbers, and every device's public key — so the object
// parses and the wraps open. What it cannot produce is the chain, which is
// keyed by the root key of the generation it claims to follow, and a party
// that never held that key cannot compute it.
func TestForgedGenerationIsRefused(t *testing.T) {
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

	chains := map[string]string{
		"no chain at all":              "",
		"a chain of the right length":  base64.StdEncoding.EncodeToString(make([]byte, chainMACSize)),
		"the previous chain replayed":  k.current().Chain,
		"a chain keyed by its own key": "",
	}
	// The last case: the attacker MACs its own header with a key it chose
	// itself, which is what it would try after reading this package.
	attackerKey := bytes.Repeat([]byte{0xAA}, crypto.RootKeySize)
	forgedSelf := forgeGeneration(t, k, "")
	selfChain, err := generationChain(forgedSelf.ProfileID, forgedSelf.current(), forgedSelf.generation(2), attackerKey)
	if err != nil {
		t.Fatal(err)
	}
	chains["a chain keyed by its own key"] = selfChain

	for name, chain := range chains {
		t.Run(name, func(t *testing.T) {
			forged := forgeGeneration(t, k, chain)
			raw, err := json.Marshal(forged)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(raw)
			if err != nil {
				// Refused before any key is involved: also correct.
				return
			}
			if parsed.CurrentGeneration != 3 {
				t.Fatalf("forged keyring parsed at generation %d", parsed.CurrentGeneration)
			}
			// A remaining device unwraps its wraps — the forged one opens,
			// which is the point: nothing about the wrap gives it away.
			keys, err := parsed.UnwrapGenerations(goldenDeviceID, a)
			if err != nil {
				t.Fatalf("the forged wrap did not even open, so the probe proves nothing: %v", err)
			}
			defer ZeroGenerations(keys)
			if err := parsed.VerifyChain(keys); !errors.Is(err, ErrUnauthenticatedGeneration) {
				t.Fatalf("VerifyChain accepted a forged generation: %v", err)
			}
			// And the recovery code cannot be talked into it either.
			if _, err := parsed.UnwrapWithRecoveryCode(goldenRecoveryCode); !errors.Is(err, ErrRecoveryMismatch) {
				t.Fatalf("the recovery code opened a forged generation: %v", err)
			}
		})
	}

	// The genuine keyring still verifies, from every reader's point of view.
	genuine, err := k.UnwrapGenerations(goldenDeviceID, a)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroGenerations(genuine)
	if err := k.VerifyChain(genuine); err != nil {
		t.Fatalf("the genuine keyring does not verify: %v", err)
	}
	fromCode, err := k.UnwrapGenerationsWithRecoveryCode(goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroGenerations(fromCode)
	if err := k.VerifyChain(fromCode); err != nil {
		t.Fatalf("the recovery code cannot verify the genuine keyring: %v", err)
	}
}

// TestVerifyChainNeedsThePredecessorsKey: a reader that cannot reach the
// generation before the current one cannot authenticate the current one
// either, and must say so rather than accept it.
func TestVerifyChainNeedsThePredecessorsKey(t *testing.T) {
	a, b := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	// Generation 1 alone: nothing to chain, and nothing to refuse.
	if err := k.VerifyChain(map[int][]byte{}); err != nil {
		t.Fatalf("generation 1 needs no chain: %v", err)
	}
	next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(next)
	err = k.VerifyChain(map[int][]byte{2: next})
	if !errors.Is(err, ErrUnauthenticatedGeneration) || !strings.Contains(err.Error(), "no root key for generation 1") {
		t.Fatalf("holding only the current key: %v", err)
	}
	if err := k.VerifyChain(map[int][]byte{1: goldenRootKey(), 2: next}); err != nil {
		t.Fatalf("holding both keys: %v", err)
	}
}

// TestChainSurvivesManyRolloversAndRecovery: the chain must hold at depth,
// and the recovery code must still enrol a device after several rollovers —
// a chain that only verifies at depth 1, or that a recovery code cannot
// check, would brick the account instead of protecting it.
func TestChainSurvivesManyRolloversAndRecovery(t *testing.T) {
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
	// The device that made every generation verifies all of them.
	keys, err := stored.UnwrapGenerations(goldenDeviceID, a)
	if err != nil || len(keys) != 9 {
		t.Fatalf("A holds %d generations: %v", len(keys), err)
	}
	defer ZeroGenerations(keys)
	if err := stored.VerifyChain(keys); err != nil {
		t.Fatalf("chain at depth 9: %v", err)
	}
	// A device enrolling from the recovery code after all of it verifies
	// the whole chain from the code alone, and lands in every generation.
	fromCode, err := stored.UnwrapGenerationsWithRecoveryCode(goldenRecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroGenerations(fromCode)
	if err := stored.VerifyChain(fromCode); err != nil {
		t.Fatalf("the recovery code cannot verify a nine-generation chain: %v", err)
	}
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
	if err := stored.VerifyChain(recovered); err != nil {
		t.Fatalf("the recovered device cannot verify the chain it just joined: %v", err)
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
