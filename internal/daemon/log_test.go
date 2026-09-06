package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingLogRotatesAndKeeps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.log")
	l, err := openLog(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("x", 39) + "\n" // 40 bytes
	for i := 0; i < 8; i++ {
		if _, err := l.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	// 8 lines of 40 bytes with a 100-byte bound: two lines per file, so
	// current + .1 + .2 exist and .3 does not.
	for _, name := range []string{"daemon.log", "daemon.log.1", "daemon.log.2"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() > 100 {
			t.Fatalf("%s is %d bytes", name, info.Size())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "daemon.log.3")); !os.IsNotExist(err) {
		t.Fatal("kept more rotated copies than asked")
	}
	// Reopening appends and honours the existing size.
	l, err = openLog(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Write([]byte("tail\n")); err != nil {
		t.Fatal(err)
	}
	_ = l.Close()
	raw, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(raw), "tail\n") || !strings.HasPrefix(string(raw), "xxx") {
		t.Fatalf("reopen did not append: %q", raw)
	}
}
