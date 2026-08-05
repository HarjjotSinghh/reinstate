package fileidentity

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentityDistinguishesReplacementAndInPlaceRewrite(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "agent")
	if err := os.WriteFile(path, []byte("first"), 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := CaptureExecutable(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsRegular() || !SameExecutable(first, first) {
		t.Fatalf("invalid first identity: %+v", first)
	}
	originalInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Restore the original size, mode, and timestamp. The content digest must
	// still distinguish the in-place rewrite that metadata-only checks miss.
	if err := os.Chtimes(path, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	rewritten, err := CaptureExecutable(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !SameObject(first, rewritten) || SameExecutable(first, rewritten) {
		t.Fatalf("in-place rewrite identities = %+v / %+v", first, rewritten)
	}

	replacement := filepath.Join(directory, "replacement")
	if err := os.WriteFile(replacement, []byte("other"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	replaced, err := CaptureExecutable(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if SameObject(first, replaced) || SameExecutable(first, replaced) {
		t.Fatalf("replacement identities = %+v / %+v", first, replaced)
	}
}

func TestIdentityCapturesDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Clean(t.TempDir())
	identity, err := Capture(path)
	if err != nil || !identity.IsDir() {
		t.Fatalf("directory identity/error = %+v / %v", identity, err)
	}
}
