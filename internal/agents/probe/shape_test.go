package probe

import (
	"strings"
	"testing"
)

// hexOfLength builds a hex stem of an exact length without writing a long
// literal into the source: a run of literal hex reads as a credential to a
// secret scanner, and this file is about lengths, not particular values.
func hexOfLength(n int) string {
	const alphabet = "0123456789abcdef"
	return strings.Repeat(alphabet, n/len(alphabet)+1)[:n]
}

// A probe artifact must not carry a content hash, whatever length it happens to
// be. Git stores an object as a two-character directory plus a thirty-eight
// character file, and OpenCode keeps a Git object store under each snapshot, so
// a real agent root produced 38-character stems. Those matched none of the
// fixed-length rules and reached the artifact verbatim — they are content
// hashes of the operator's own repository.
func TestNormalizeCollapsesHexRunOfAnyLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
		want   string
	}{
		{"git object file", 38, "<38-hex>"},
		{"between the known lengths", 36, "<36-hex>"},
		{"shortest collapsed", 12, "<12-hex>"},

		// The established tokens must not churn: an artifact that already
		// reads <32-hex> should keep reading that.
		{"md5 length keeps its token", 32, "<32-hex>"},
		{"sha1 length keeps its token", 40, "<40-hex>"},
		{"sha256 length keeps its token", 64, "<64-hex>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stem := hexOfLength(tt.length)
			if got := normalizeStem(stem); got != tt.want {
				t.Fatalf("normalizeStem(%d hex characters) = %q, want %q", tt.length, got, tt.want)
			}
		})
	}
}

// A name too short to identify content stays readable, so shapes keep their
// meaning.
func TestNormalizeLeavesShortHexAlone(t *testing.T) {
	t.Parallel()

	for _, stem := range []string{"abc", "cafe", "deadbeef", hexOfLength(11)} {
		if got := normalizeStem(stem); strings.HasSuffix(got, "-hex>") {
			t.Fatalf("normalizeStem(%q) = %q; a stem this short is not a content hash", stem, got)
		}
	}
}
