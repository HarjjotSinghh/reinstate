package doctor

import "testing"

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
