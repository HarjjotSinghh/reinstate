package pathmap

import (
	"strings"
	"testing"
)

func portableMapper() Mapper {
	return Mapper{
		Home:     "/Users/fixture-user",
		Projects: map[string]string{"github.com/example/demo": "/Users/fixture-user/code/demo"},
		GOOS:     "darwin",
	}
}

func TestNormalizePortableAlwaysReturnsAToken(t *testing.T) {
	m := portableMapper()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "inside project", in: "/Users/fixture-user/code/demo/calc.go", want: "${REPO:github.com/example/demo}/calc.go"},
		{name: "inside home", in: "/Users/fixture-user/.config/demo/settings.json", want: "${HOME}/.config/demo/settings.json"},
		{name: "already a token", in: "${REPO:github.com/example/demo}/calc.go", want: "${REPO:github.com/example/demo}/calc.go"},
		{name: "relative stays relative", in: "calc.go", want: "calc.go"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := m.NormalizePortable(test.in); got != test.want {
				t.Fatalf("NormalizePortable(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestNormalizePortableReplacesOutsideRootPaths(t *testing.T) {
	m := portableMapper()
	got := m.NormalizePortable("/etc/hosts")
	if !strings.HasPrefix(got, ExternalPrefix) {
		t.Fatalf("NormalizePortable(/etc/hosts) = %q, want an %s token", got, ExternalPrefix)
	}
	if strings.Contains(got, "/etc/") {
		t.Fatalf("external token disclosed its location: %q", got)
	}
	if !strings.HasSuffix(got, "/hosts") {
		t.Fatalf("external token dropped the base name: %q", got)
	}
	if again := m.NormalizePortable("/etc/hosts"); again != got {
		t.Fatalf("external token is not stable: %q vs %q", got, again)
	}
	if other := m.NormalizePortable("/etc/other"); other == got {
		t.Fatalf("distinct paths collapsed to one token: %q", other)
	}
}

func TestNormalizeKeepsUnmatchedPathsForVendorRewriting(t *testing.T) {
	// Normalize must stay fail-open: vendor session rewriting leaves unknown
	// paths alone. Only NormalizePortable substitutes a token.
	m := portableMapper()
	if got := m.Normalize("/etc/hosts"); got != "/etc/hosts" {
		t.Fatalf("Normalize(/etc/hosts) = %q, want it unchanged", got)
	}
}

func TestIsToken(t *testing.T) {
	for _, value := range []string{
		"${REPO:x}/a", "${HOME}/a", "${WORK:alias}/a", ExternalPrefix + "abcd}/a",
	} {
		if !IsToken(value) {
			t.Fatalf("IsToken(%q) = false", value)
		}
	}
	for _, value := range []string{"", "/etc/hosts", "calc.go", `C:\Users\x`} {
		if IsToken(value) {
			t.Fatalf("IsToken(%q) = true", value)
		}
	}
}
