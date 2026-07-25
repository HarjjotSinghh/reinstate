package memory

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

func TestDiskStoreRoundTrip(t *testing.T) {
	s, err := NewDisk(filepath.Join(t.TempDir(), "backend"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	meta, err := s.Put(ctx, "snapshots/a.age", bytes.NewReader([]byte("cipher")), 6, backend.PutOptions{IfNoneMatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, "snapshots/a.age", bytes.NewReader([]byte("x")), 1, backend.PutOptions{IfNoneMatch: true}); err != backend.ErrAlreadyExists {
		t.Fatalf("got %v", err)
	}
	rc, m, err := s.Get(ctx, "snapshots/a.age")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	b, _ := io.ReadAll(rc)
	if string(b) != "cipher" || m.ETag != meta.ETag {
		t.Fatalf("%s %+v", b, m)
	}
}
