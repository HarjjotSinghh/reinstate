// Package conformance enforces a descriptor's declared tier with nine checks.
package conformance

import (
	"os"
	"path/filepath"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

// Fixtures names the synthetic tree Run scans. Root is repository-relative.
// OS lists platform subdirectories under Root when they exist (macos, windows).
type Fixtures struct {
	Root string
	OS   []string
}

// Check is one named conformance result.
type Check struct {
	Name string
	Err  error
}

// Tester is the *testing.T surface Run needs.
type Tester interface {
	Helper()
	Errorf(format string, args ...any)
}

// Run asserts every check for descriptor against fixtures.
func Run(t Tester, d agents.Descriptor, fixtures Fixtures) {
	t.Helper()
	for _, check := range Evaluate(d, fixtures) {
		if check.Err != nil {
			t.Errorf("%s: %v", check.Name, check.Err)
		}
	}
}

// Evaluate runs the nine SDK checks without failing a test.
func Evaluate(d agents.Descriptor, fixtures Fixtures) []Check {
	root, err := repoRoot()
	if err != nil {
		return []Check{{Name: "structure", Err: err}}
	}
	return []Check{
		{Name: "structure", Err: checkStructure(d)},
		{Name: "capability", Err: checkCapability(d)},
		{Name: "evidence", Err: checkEvidence(d, root)},
		{Name: "determinism", Err: checkDeterminism(d, root, fixtures)},
		{Name: "isolation", Err: checkIsolation(d, root, fixtures)},
		{Name: "corruption", Err: checkCorruption(d)},
		{Name: "privacy", Err: checkPrivacy(d, root, fixtures)},
		{Name: "version", Err: checkVersion(d)},
		{Name: "readonly", Err: checkReadOnly(d, root, fixtures)},
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func fixtureRoots(root string, fixtures Fixtures) []string {
	base := fixtures.Root
	if base == "" {
		return nil
	}
	if !filepath.IsAbs(base) {
		base = filepath.Join(root, base)
	}
	if len(fixtures.OS) == 0 {
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			return []string{base}
		}
		return nil
	}
	var out []string
	for _, osName := range fixtures.OS {
		candidate := filepath.Join(base, osName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			return []string{base}
		}
	}
	return out
}
