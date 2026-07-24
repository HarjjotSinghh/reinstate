package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// FileStore persists credentials under home/credentials/<safe-ref>.json
// with owner-only permissions. Secrets never enter config.toml.
type FileStore struct {
	Dir string
}

// NewFileStore returns a store rooted at dir (typically ~/.reinstate/credentials).
func NewFileStore(dir string) *FileStore {
	return &FileStore{Dir: dir}
}

func (f *FileStore) path(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty credential ref")
	}
	// map ref to a single path segment
	safe := strings.ReplaceAll(ref, "/", "__")
	safe = strings.ReplaceAll(safe, string(os.PathSeparator), "__")
	if safe == "." || safe == ".." || strings.Contains(safe, "..") {
		return "", fmt.Errorf("invalid credential ref")
	}
	return filepath.Join(f.Dir, safe+".json"), nil
}

func (f *FileStore) Set(ref string, c StorageCredentials) error {
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("incomplete credentials")
	}
	if err := fsx.EnsureOwnerOnlyDir(f.Dir); err != nil {
		return err
	}
	p, err := f.path(ref)
	if err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomic(p, append(b, '\n'), fsx.OwnerOnlyFilePerm)
}

func (f *FileStore) Get(ref string) (StorageCredentials, error) {
	p, err := f.path(ref)
	if err != nil {
		return StorageCredentials{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return StorageCredentials{}, fmt.Errorf("credentials not found for %s", ref)
	}
	var c StorageCredentials
	if err := json.Unmarshal(b, &c); err != nil {
		return StorageCredentials{}, err
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return StorageCredentials{}, fmt.Errorf("incomplete credentials for %s", ref)
	}
	return c, nil
}

func (f *FileStore) Delete(ref string) error {
	p, err := f.path(ref)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Resolve loads credentials: env overrides, then file store by ref.
func Resolve(home, ref string) (StorageCredentials, error) {
	if ak, sk := os.Getenv("REINSTATE_S3_ACCESS_KEY_ID"), os.Getenv("REINSTATE_S3_SECRET_ACCESS_KEY"); ak != "" && sk != "" {
		return StorageCredentials{AccessKeyID: ak, SecretAccessKey: sk}, nil
	}
	fs := NewFileStore(filepath.Join(home, "credentials"))
	return fs.Get(ref)
}
