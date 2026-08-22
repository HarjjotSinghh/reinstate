package agentcheck

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// openCodeShapedDefinition mirrors the shipped OpenCode descriptor: one
// embedded SQLite store as the marker, and a root variable that names the
// parent of the root rather than the root itself.
func openCodeShapedDefinition() Definition {
	pattern := regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)
	return Definition{
		Executable:            "opencode",
		Layout:                "embedded-sqlite-session-store",
		Marker:                "opencode.db",
		RootEnvironment:       "XDG_DATA_HOME",
		RootEnvironmentSuffix: "opencode",
		Roots: func(home string) []string {
			return []string{filepath.Join(home, ".local", "share", "opencode")}
		},
		Parse: func(output VersionOutput) (string, bool) {
			return parseVersionLine(output, pattern)
		},
		Min: "1.18.21",
		Max: "1.18.21",
	}
}

func withOpenCodeDefinition(t *testing.T) {
	t.Helper()
	previous := definitions
	SetDefinitions(map[string]Definition{"opencode": openCodeShapedDefinition()})
	t.Cleanup(func() { definitions = previous })
}

func writeStore(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.db"), []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestInspectRecognizesAnEmbeddedStoreMarker covers an agent whose sessions are
// one file rather than a tree.
//
// Requiring the marker to be a directory reports the layout as unrecognized
// even when the store the index already reads is sitting right there, which
// refuses resume for every embedded-store agent.
func TestInspectRecognizesAnEmbeddedStoreMarker(t *testing.T) {
	withOpenCodeDefinition(t)
	home := t.TempDir()
	writeStore(t, filepath.Join(home, ".local", "share", "opencode"))

	result := Inspect(context.Background(), "opencode", Options{
		Home:            home,
		Getenv:          func(string) string { return "" },
		LookPath:        func(value string) (string, error) { return "/verified/" + value, nil },
		Runner:          &fakeRunner{output: VersionOutput{Stdout: "1.18.21\n"}},
		CaptureIdentity: testExecutableIdentity,
	})
	if !result.LayoutRecognized || result.Status != StatusSupported {
		t.Fatalf("embedded store was not recognized: %+v", result)
	}
	if result.Version != "1.18.21" {
		t.Fatalf("version = %q, want 1.18.21", result.Version)
	}
}

// TestInspectAppendsRootEnvironmentSuffix covers a vendor whose root variable
// names the parent directory.
//
// OpenCode reads $XDG_DATA_HOME/opencode. Treating the variable's value as the
// root looks for the store one directory too high, so an operator who points
// the variable at a sanitized root — the one thing the variable exists for —
// gets an unrecognized layout and no resume.
func TestInspectAppendsRootEnvironmentSuffix(t *testing.T) {
	withOpenCodeDefinition(t)
	parent := t.TempDir()
	writeStore(t, filepath.Join(parent, "opencode"))

	inspect := func() Result {
		return Inspect(context.Background(), "opencode", Options{
			Home: t.TempDir(),
			Getenv: func(name string) string {
				if name == "XDG_DATA_HOME" {
					return parent
				}
				return ""
			},
			LookPath:        func(value string) (string, error) { return "/verified/" + value, nil },
			Runner:          &fakeRunner{output: VersionOutput{Stdout: "1.18.21\n"}},
			CaptureIdentity: testExecutableIdentity,
		})
	}

	result := inspect()
	if !result.LayoutRecognized || result.Status != StatusSupported {
		t.Fatalf("suffixed root was not resolved: %+v", result)
	}

	// The parent alone must not satisfy the marker: a store sitting directly in
	// $XDG_DATA_HOME is a different layout, not this one.
	if err := os.WriteFile(filepath.Join(parent, "opencode.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(parent, "opencode")); err != nil {
		t.Fatal(err)
	}
	if bare := inspect(); bare.LayoutRecognized {
		t.Fatalf("a store beside the root was accepted as the root: %+v", bare)
	}
}

// TestRecognizedLayoutRejectsMarkersThatAreNeitherDirectoryNorFile keeps the
// widening to exactly one kind. A symlinked store is still refused, because a
// symlink is how a marker check is redirected out of the root it was scoped to.
func TestRecognizedLayoutRejectsMarkersThatAreNeitherDirectoryNorFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "opencode.db")
	if err := os.WriteFile(target, []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "opencode.db")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if recognizedLayout(root, "opencode.db") {
		t.Fatal("a symlinked store was accepted as a recognized layout")
	}
}
