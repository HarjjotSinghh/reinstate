package keyring

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// Golden inputs for testdata/keyring/keyring.one-generation.json. Synthetic
// values generated for the fixture only; they protect nothing.
const (
	goldenProfileID    = "11111111-2222-4333-8444-555555555555"
	goldenDeviceID     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	goldenRecoveryCode = "S68E-X6C6-9J6V-P4QB-B98F-PT00-NW6K-W7C6"
	goldenDeviceKey    = "AGE-SECRET-KEY-1J7YHF3Y899CUH9V2WWWPDAR8DNUEASJDLAP9KY3KJ6YW0DZKL5XQQJJ87L"
	goldenRecipient    = "age12zhh9qvam60p0vhsv0x423pn2y6u423lwa5f83s3hf6nju2gfcssjvyj2f"
)

// goldenPath is the one-generation fixture; goldenV2Path (generation_test.go)
// is the rolled-over one. Both are written by this package's current format,
// so both are regenerated together when the format changes deliberately.
var goldenPath = filepath.Join("..", "..", "testdata", "keyring", "keyring.one-generation.json")

// goldenStamp is the fixed clock both fixtures are written against.
var goldenStamp = time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)

func goldenRootKey() []byte {
	key := make([]byte, crypto.RootKeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func loadGolden(t *testing.T) *Keyring {
	t.Helper()
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	k, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// TestGoldenKeyringUnwraps pins the keyring object format: a fixture written
// by this version must keep unwrapping under both paths in every later
// version, or existing lockers become unreadable. Set KEYRING_WRITE_GOLDEN=1
// to regenerate it after a deliberate format change.
func TestGoldenKeyringUnwraps(t *testing.T) {
	if os.Getenv("KEYRING_WRITE_GOLDEN") == "1" {
		identity, err := age.ParseX25519Identity(goldenDeviceKey)
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, identity, goldenStamp)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := fresh.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	k := loadGolden(t)
	if k.SchemaVersion != SchemaVersion || k.ProfileID != goldenProfileID || k.CurrentGeneration != 1 || k.DeviceCount() != 1 {
		t.Fatalf("unexpected golden shape: %+v", k)
	}
	if k.Generations[0].Chain != "" {
		t.Fatal("generation 1 carries a chain, but it has no generation before it")
	}
	if k.Generations[0].Recipient != goldenRecipient {
		t.Fatalf("recipient drifted: %s", k.Generations[0].Recipient)
	}
	t.Run("recovery code", func(t *testing.T) {
		got, err := k.UnwrapWithRecoveryCode(goldenRecoveryCode)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, goldenRootKey()) {
			t.Fatalf("root key mismatch: %x", got)
		}
	})
	t.Run("typed loosely", func(t *testing.T) {
		loose := strings.ToLower(strings.ReplaceAll(goldenRecoveryCode, "-", " "))
		if _, err := k.UnwrapWithRecoveryCode(loose); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("device key", func(t *testing.T) {
		identity, err := age.ParseX25519Identity(goldenDeviceKey)
		if err != nil {
			t.Fatal(err)
		}
		current, earlier, err := k.UnwrapForDevice(goldenDeviceID, identity)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(current, goldenRootKey()) || len(earlier) != 0 {
			t.Fatalf("root key mismatch: %x earlier=%d", current, len(earlier))
		}
	})
}

func TestGoldenKeyringFailsClosed(t *testing.T) {
	k := loadGolden(t)
	wrong, err := GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.UnwrapWithRecoveryCode(wrong); !errors.Is(err, ErrRecoveryMismatch) {
		t.Fatalf("wrong code: got %v", err)
	}
	other, _ := age.GenerateX25519Identity()
	if _, _, err := k.UnwrapForDevice("not-enrolled", other); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("unknown device: got %v", err)
	}
	if _, _, err := k.UnwrapForDevice(goldenDeviceID, other); !errors.Is(err, ErrDeviceNotEnrolled) {
		t.Fatalf("enrolled id with a different key: got %v", err)
	}
	// The fixture must contain no root key, recovery code, or device secret.
	raw, _ := os.ReadFile(goldenPath)
	for _, secret := range []string{goldenRecoveryCode, goldenDeviceKey, "0102030405060708"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("keyring fixture leaks %q", secret)
		}
	}
}

func TestRecoveryCodeNormalization(t *testing.T) {
	valid := "S68E-X6C6-9J6V-P4QB-B98F-PT00-NW6K-W7C6"
	cases := map[string]struct {
		typed string
		want  string
		ok    bool
	}{
		"canonical":          {valid, valid, true},
		"lower no dashes":    {strings.ToLower(strings.ReplaceAll(valid, "-", "")), valid, true},
		"spaces and newline": {strings.ReplaceAll(valid, "-", " ") + "\n", valid, true},
		"O for zero":         {strings.Replace(valid, "0", "O", 1), valid, true},
		"one char typo":      {"S68E-X6C6-9J6V-P4QB-B98F-PT00-NW6K-W7C7", "", false},
		"swapped chars":      {"S68E-X6C6-9J6V-P4QB-B98F-PT00-NW6K-W7C6"[:4] + "-6X6C" + valid[9:], "", false},
		"too short":          {valid[:20], "", false},
		"invalid character":  {strings.Replace(valid, "S", "U", 1), "", false},
		"empty":              {"", "", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := NormalizeRecoveryCode(tc.typed)
			if tc.ok && (err != nil || got != tc.want) {
				t.Fatalf("got %q, %v", got, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("accepted %q as %q", tc.typed, got)
			}
		})
	}
}

func TestGenerateRecoveryCodeShapeAndEntropy(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		code, err := GenerateRecoveryCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != len(RecoveryCodeFormat) || strings.Count(code, "-") != 7 {
			t.Fatalf("bad shape %q", code)
		}
		if got, err := NormalizeRecoveryCode(code); err != nil || got != code {
			t.Fatalf("generated code does not normalize to itself: %q %v", got, err)
		}
		if seen[code] {
			t.Fatalf("duplicate code %q", code)
		}
		seen[code] = true
	}
	if recoveryDataBits < 128 {
		t.Fatalf("recovery code carries %d bits, want at least 128", recoveryDataBits)
	}
}

func TestEnrolRejectsForeignRootKeyAndDuplicates(t *testing.T) {
	k := loadGolden(t)
	device, _ := age.GenerateX25519Identity()
	now := time.Now()
	foreign, _ := crypto.NewRootKey()
	if err := k.Enrol(foreign, "new-device", device.Recipient(), now); err == nil {
		t.Fatal("wrapped a root key that is not this generation's")
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceID, device.Recipient(), now); !errors.Is(err, ErrDeviceExists) {
		t.Fatalf("duplicate device: got %v", err)
	}
	if err := k.Enrol(goldenRootKey(), "new-device", device.Recipient(), now); err != nil {
		t.Fatal(err)
	}
	current, _, err := k.UnwrapForDevice("new-device", device)
	if err != nil || !bytes.Equal(current, goldenRootKey()) {
		t.Fatalf("new device cannot read: %v", err)
	}
}

func TestUnenrolAppendedRemovesOnlyTheMatchingWrap(t *testing.T) {
	k := loadGolden(t)
	device, _ := age.GenerateX25519Identity()
	appended, err := k.EnrolAll(map[int][]byte{1: goldenRootKey()}, "new-device", device.Recipient(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(appended) != 1 || appended[0].Generation != 1 || appended[0].Wrap == "" {
		t.Fatalf("EnrolAll reported %+v", appended)
	}
	before := k.DeviceCount()
	if k.UnenrolAppended("new-device", []AppendedWrap{{Generation: 1, Wrap: "some other ciphertext"}}) || k.DeviceCount() != before {
		t.Fatal("removed a wrap whose ciphertext does not match")
	}
	if k.UnenrolAppended("missing", appended) || k.UnenrolAppended("new-device", nil) || k.DeviceCount() != before {
		t.Fatal("removed a wrap for an unknown device, or with nothing to remove")
	}
	if k.UnenrolAppended("new-device", []AppendedWrap{{Generation: 9, Wrap: appended[0].Wrap}}) || k.DeviceCount() != before {
		t.Fatal("removed a wrap from a generation the keyring does not hold")
	}
	if !k.UnenrolAppended("new-device", appended) || k.DeviceCount() != before-1 || k.HasDevice("new-device") {
		t.Fatal("did not remove the matching wrap")
	}
	golden, err := age.ParseX25519Identity(goldenDeviceKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := k.UnwrapForDevice(goldenDeviceID, golden); err != nil {
		t.Fatalf("the original device's wrap was disturbed: %v", err)
	}
	if k.UnenrolAppended("new-device", appended) {
		t.Fatal("removed the same wrap twice")
	}
}

// racingBackend makes another device's enrolment land between this
// device's load and conditional put, exactly once, so Update must observe
// the precondition failure and re-apply on the fresh keyring.
type racingBackend struct {
	backend.Backend
	once sync.Once
	race func()
}

func (r *racingBackend) Put(ctx context.Context, key string, body io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	if opts.IfMatch != "" {
		r.once.Do(r.race)
	}
	return r.Backend.Put(ctx, key, body, size, opts)
}

func TestUpdateConvergesWhenTwoDevicesAppend(t *testing.T) {
	ctx := context.Background()
	disk, err := memory.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := ObjectKey("profiles/p")
	if key != "profiles/p/"+ObjectName || ObjectKey("") != ObjectName || ObjectKey("/x/") != "x/"+ObjectName {
		t.Fatalf("object key shape: %q", key)
	}
	if err := Create(ctx, disk, key, loadGolden(t)); err != nil {
		t.Fatal(err)
	}
	if err := Create(ctx, disk, key, loadGolden(t)); err == nil {
		t.Fatal("second create overwrote the keyring")
	}
	deviceB, _ := age.GenerateX25519Identity()
	deviceC, _ := age.GenerateX25519Identity()
	racing := &racingBackend{Backend: disk}
	racing.race = func() {
		if _, err := Update(ctx, disk, key, func(k *Keyring) error {
			return k.Enrol(goldenRootKey(), "device-c", deviceC.Recipient(), time.Now())
		}); err != nil {
			t.Errorf("competing update: %v", err)
		}
	}
	updated, err := Update(ctx, racing, key, func(k *Keyring) error {
		return k.Enrol(goldenRootKey(), "device-b", deviceB.Recipient(), time.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.DeviceCount() != 3 {
		t.Fatalf("device count %d, want 3 after converging", updated.DeviceCount())
	}
	stored, _, err := Load(ctx, disk, key)
	if err != nil {
		t.Fatal(err)
	}
	for id, identity := range map[string]*age.X25519Identity{"device-b": deviceB, "device-c": deviceC} {
		if _, _, err := stored.UnwrapForDevice(id, identity); err != nil {
			t.Fatalf("%s lost its wrap: %v", id, err)
		}
	}
}

func TestLoadReportsMissing(t *testing.T) {
	disk, _ := memory.NewDisk(t.TempDir())
	if _, _, err := Load(context.Background(), disk, ObjectName); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}
