package pathmap

import (
	"path/filepath"
	"testing"
)

func TestRoundTripPOSIX(t *testing.T) {
	m := Mapper{
		Home: "/Users/fixture-user",
		Projects: map[string]string{
			"github.com/example/demo": "/Users/fixture-user/code/demo",
		},
		GOOS: "darwin",
	}
	in := "/Users/fixture-user/code/demo/src/main.go"
	port := m.Normalize(in)
	if port != "${REPO:github.com/example/demo}/src/main.go" {
		t.Fatalf("normalize: %q", port)
	}
	out := m.Denormalize(port)
	if filepath.Clean(out) != filepath.Clean(in) {
		t.Fatalf("denorm %q want %q", out, in)
	}
	homePort := m.Normalize("/Users/fixture-user/.config/x")
	if homePort != "${HOME}/.config/x" {
		t.Fatalf("%q", homePort)
	}
}

func TestWindowsDrive(t *testing.T) {
	m := Mapper{
		Home: `C:\Users\fixture-user`,
		Projects: map[string]string{
			"github.com/example/demo": `C:\Users\fixture-user\code\demo`,
		},
		GOOS: "windows",
	}
	in := `C:\Users\fixture-user\code\demo\src\main.go`
	port := m.Normalize(in)
	if port != "${REPO:github.com/example/demo}/src/main.go" {
		t.Fatalf("normalize: %q", port)
	}
	out := m.Denormalize(port)
	// Compare slash-normalized
	if filepath.ToSlash(out) != filepath.ToSlash(in) && filepath.Clean(out) != filepath.Clean(in) {
		// On non-windows host, FromSlash may differ; ensure ends with demo/src/main.go
		if !hasSuffixPath(out, "demo/src/main.go") && !hasSuffixPath(out, `demo\src\main.go`) {
			t.Fatalf("denorm %q", out)
		}
	}
}

func hasSuffixPath(p, suf string) bool {
	p = filepath.ToSlash(p)
	suf = filepath.ToSlash(suf)
	return len(p) >= len(suf) && p[len(p)-len(suf):] == suf
}

func TestSpacesAndUnicode(t *testing.T) {
	m := Mapper{
		Home: "/home/fixture-user",
		Projects: map[string]string{
			"local/proj": "/home/fixture-user/My Projects/项目",
		},
		GOOS: "linux",
	}
	in := "/home/fixture-user/My Projects/项目/a b.go"
	port := m.Normalize(in)
	want := "${REPO:local/proj}/a b.go"
	if port != want {
		t.Fatalf("%q", port)
	}
	if m.Denormalize(port) == "" {
		t.Fatal("empty")
	}
}
