package catalog

import (
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

// TestOpenCodeDeclaresTheRootVariableItsReaderHonours pins the catalog to the
// variable the OpenCode reader actually uses.
//
// The reader resolves $XDG_DATA_HOME/opencode, so the variable names the parent
// of the root. The probe's RootEnv treats a variable's value as the root
// itself, so before RootEnvSuffix existed OpenCode could declare nothing at
// all — and an operator who redirected it got their real tree probed anyway,
// silently. Drift here is not cosmetic: it is the difference between an audit
// reading a prepared root and reading the operator's own repositories.
func TestOpenCodeDeclaresTheRootVariableItsReaderHonours(t *testing.T) {
	t.Parallel()

	var descriptor agents.Descriptor
	for _, candidate := range agents.All() {
		if candidate.Key == "opencode" {
			descriptor = candidate
			break
		}
	}
	if descriptor.Key == "" {
		t.Fatal("opencode is not in the catalog")
	}
	if descriptor.Storage.RootEnv != "XDG_DATA_HOME" || descriptor.Storage.RootEnvSuffix != "opencode" {
		t.Fatalf("opencode declares RootEnv=%q suffix=%q, want XDG_DATA_HOME + opencode",
			descriptor.Storage.RootEnv, descriptor.Storage.RootEnvSuffix)
	}

	// The declaration is only worth anything if it agrees with the reader.
	parent := t.TempDir()
	got, err := transcript.ResolveOpenCodeDataRoot(
		func(key string) string {
			if key == descriptor.Storage.RootEnv {
				return parent
			}
			return ""
		},
		func() (string, error) { return filepath.Join(parent, "unused-home"), nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(parent, descriptor.Storage.RootEnvSuffix)
	if got != want {
		t.Fatalf("reader resolved %q from the declared variable, want %q", got, want)
	}
}
