package workspace

import (
	"strings"
	"testing"
)

func FuzzParsePorcelainV2(f *testing.F) {
	f.Add([]byte("# branch.oid 1111111111111111111111111111111111111111\x00# branch.head main\x00"))
	f.Add([]byte("1 .M N... 100644 100644 100644 a b controlled-name\x00"))
	f.Add([]byte("2 R. N... 100644 100644 100644 a b R100 destination\x00source\x00"))
	f.Fuzz(func(t *testing.T, input []byte) {
		result, err := parsePorcelainV2(input)
		if err != nil {
			return
		}
		counts := []int{result.workingTree.Staged, result.workingTree.Unstaged,
			result.workingTree.Untracked, result.workingTree.Conflicted, result.workingTree.Submodule}
		for _, count := range counts {
			if count < 0 || count > maxDirtyCount {
				t.Fatalf("unbounded dirty count %d: %+v", count, result)
			}
		}
	})
}

func FuzzRepositoryIDFromRemote(f *testing.F) {
	for _, seed := range []string{
		syntheticCredentialRemote("user", "secret", "example.com/org/repo.git", "token=private#fragment"),
		"ssh://git@example.com/org/repo.git",
		"git@example.com:org/repo.git",
		"file:///private/repo",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		first, err := RepositoryIDFromRemote(input)
		if err != nil {
			return
		}
		second, err := RepositoryIDFromRemote(input)
		if err != nil || first != second || !strings.HasPrefix(first, "remote-sha256:") || len(first) != len("remote-sha256:")+64 {
			t.Fatalf("unstable remote identity %q / %q / %v", first, second, err)
		}
		if first == input || strings.Contains(first, "secret") || strings.Contains(first, "token") {
			t.Fatalf("remote identity leaked input: %q", first)
		}
	})
}
