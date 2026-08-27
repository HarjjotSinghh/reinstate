package keyring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
)

// TestWriteRefusesAnOversizedKeyring: the object only ever grows — every
// revocation appends a generation holding one wrap per remaining device, and
// no generation is ever removed — while a read stops at maxObjectBytes. An
// account that once wrote past that could never read its keyring again: not
// to push, not to pull, not to revoke, not to recover. So the write is
// refused while a read still has headroom, and the object already in storage
// is left exactly as it was.
func TestWriteRefusesAnOversizedKeyring(t *testing.T) {
	ctx := context.Background()
	disk, err := memory.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := Create(ctx, disk, ObjectName, loadGolden(t)); err != nil {
		t.Fatal(err)
	}
	// Grown the cheap way: a device wrap costs the same bytes wherever it
	// sits, and enrolling avoids the argon2id pass a rollover would pay.
	grown := loadGolden(t)
	raw, err := grown.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; len(raw) <= maxWriteBytes; i++ {
		device, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatal(err)
		}
		if err := grown.Enrol(goldenRootKey(), fmt.Sprintf("device-%d", i), device.Recipient(), time.Now()); err != nil {
			t.Fatal(err)
		}
		if i%32 == 0 {
			if raw, err = grown.Marshal(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if raw, err = grown.Marshal(); err != nil {
		t.Fatal(err)
	}
	// The refusal must land while the object is still readable, or it
	// would be refusing something the account had already lost.
	if len(raw) > maxObjectBytes {
		t.Fatalf("the write cap left no headroom: %d bytes would not read back under %d", len(raw), maxObjectBytes)
	}
	fresh, err := memory.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := Create(ctx, fresh, ObjectName, grown); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Create accepted %d bytes: %v", len(raw), err)
	}
	if _, _, err := Load(ctx, fresh, ObjectName); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a refused Create left an object behind: %v", err)
	}
	_, err = Update(ctx, disk, ObjectName, func(k *Keyring) error {
		*k = *grown
		return nil
	})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Update accepted %d bytes: %v", len(raw), err)
	}
	if !strings.Contains(err.Error(), "start the account again on a fresh locker") {
		t.Fatalf("the refusal does not name a remedy: %v", err)
	}
	stored, _, err := Load(ctx, disk, ObjectName)
	if err != nil || stored.DeviceCount() != 1 {
		t.Fatalf("a refused Update changed the stored keyring: %d devices, %v", stored.DeviceCount(), err)
	}
}
