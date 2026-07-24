package memory

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

func TestMemoryCRUDAndPreconditions(t *testing.T) {
	s := New()
	ctx := context.Background()
	meta, err := s.Put(ctx, "a", bytes.NewReader([]byte("one")), 3, backend.PutOptions{IfNoneMatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(ctx, "a", bytes.NewReader([]byte("two")), 3, backend.PutOptions{IfNoneMatch: true}); err != backend.ErrAlreadyExists {
		t.Fatalf("got %v", err)
	}
	meta2, err := s.Put(ctx, "a", bytes.NewReader([]byte("two")), 3, backend.PutOptions{IfMatch: meta.ETag})
	if err != nil {
		t.Fatal(err)
	}
	if meta2.ETag == meta.ETag {
		t.Fatal("etag should change")
	}
	rc, m, err := s.Get(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if string(b) != "two" || m.ETag != meta2.ETag {
		t.Fatalf("%s %+v", b, m)
	}
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Head(ctx, "a"); err != backend.ErrNotFound {
		t.Fatal(err)
	}
}
