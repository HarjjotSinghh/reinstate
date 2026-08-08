package doctor

import (
	"path/filepath"
	"testing"
)

func TestRedactPathHidesAbsolutePathsOutsideHome(t *testing.T) {
	for _, privatePath := range []string{
		"/private/reinstate/secret-project",
		`D:\ReinstateAcceptanceProjects\secret-project`,
	} {
		t.Run(privatePath, func(t *testing.T) {
			if got := RedactPath(privatePath); got != "[REDACTED_PATH]" {
				t.Fatalf("RedactPath(%q) = %q, want [REDACTED_PATH]", privatePath, got)
			}
		})
	}
}

func TestRedactPathDoesNotTreatHomePrefixSiblingAsHome(t *testing.T) {
	root := string(filepath.Separator)
	if volume := filepath.VolumeName(t.TempDir()); volume != "" {
		root = volume + root
	}
	home := filepath.Join(root, "reinstate-redact-test", "alice")
	t.Setenv("REINSTATE_HOME", home)
	sibling := home + "2" + string(filepath.Separator) + "secret-project"

	if got := RedactPath(sibling); got != "[REDACTED_PATH]" {
		t.Fatalf("RedactPath(%q) = %q, want [REDACTED_PATH]", sibling, got)
	}
}
