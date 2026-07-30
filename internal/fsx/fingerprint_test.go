package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFingerprintDetectsConcurrentWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	before, err := FingerprintFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.Exists {
		t.Fatal("existing file reported as missing")
	}
	if err := VerifyUnchanged(path, before); err != nil {
		t.Fatalf("untouched file reported as changed: %v", err)
	}

	// An agent appends while the restore is being prepared.
	if err := os.WriteFile(path, []byte("original\nappended by a live agent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Guarantee a distinguishable modification time on coarse-grained clocks.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	err = VerifyUnchanged(path, before)
	if err == nil {
		t.Fatal("concurrent write was not detected")
	}
	if !strings.Contains(err.Error(), "changed on disk during restore") {
		t.Fatalf("unhelpful error: %v", err)
	}
}

func TestFingerprintTableCases(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		setup   func(path string)
		mutate  func(path string)
		wantErr bool
	}{
		{
			name:    "missing file stays missing",
			setup:   func(string) {},
			mutate:  func(string) {},
			wantErr: false,
		},
		{
			name:    "missing file is created underneath us",
			setup:   func(string) {},
			mutate:  func(p string) { _ = os.WriteFile(p, []byte("new\n"), 0o600) },
			wantErr: true,
		},
		{
			name:    "existing file is removed underneath us",
			setup:   func(p string) { _ = os.WriteFile(p, []byte("data\n"), 0o600) },
			mutate:  func(p string) { _ = os.Remove(p) },
			wantErr: true,
		},
		{
			name:    "size changes",
			setup:   func(p string) { _ = os.WriteFile(p, []byte("data\n"), 0o600) },
			mutate:  func(p string) { _ = os.WriteFile(p, []byte("data data\n"), 0o600) },
			wantErr: true,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, string(rune('a'+i))+".jsonl")
			tc.setup(path)
			before, err := FingerprintFile(path)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(path)
			err = VerifyUnchanged(path, before)
			if tc.wantErr && err == nil {
				t.Fatal("expected a change to be detected")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected change reported: %v", err)
			}
		})
	}
}
