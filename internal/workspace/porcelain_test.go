package workspace

import (
	"bytes"
	"strings"
	"testing"
)

func statusFixture(records ...string) []byte {
	headers := []string{
		"# branch.oid " + strings.Repeat("a", 40),
		"# branch.head feature/test",
		"# branch.upstream origin/feature/test",
		"# branch.ab +2 -1",
	}
	return []byte(strings.Join(append(headers, records...), "\x00") + "\x00")
}

func TestParsePorcelainV2TracksStateWithoutPaths(t *testing.T) {
	t.Parallel()
	output := statusFixture(
		"1 M. N... 100644 100644 100644 aaaaaaa bbbbbbb staged.go",
		"1 .M S.M. 100644 100644 100644 aaaaaaa bbbbbbb unstaged-submodule",
		"? unicode-β\nname.go",
		"u UU N... 100644 100644 100644 100644 aaaaaaa bbbbbbb ccccccc conflict.go",
	)
	parsed, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.branch != "feature/test" || parsed.head != strings.Repeat("a", 40) {
		t.Fatalf("identity = %+v", parsed)
	}
	if parsed.relation.Relation != RelationDiverged || parsed.relation.Ahead != 2 || parsed.relation.Behind != 1 {
		t.Fatalf("upstream relation = %+v", parsed.relation)
	}
	tree := parsed.workingTree
	if tree.State != WorkingTreeModified || tree.Staged != 1 || tree.Unstaged != 1 ||
		tree.Untracked != 1 || tree.Conflicted != 1 || tree.Submodule != 1 ||
		!strings.HasPrefix(tree.Digest, "sha256:") {
		t.Fatalf("working tree = %+v", tree)
	}
}

func TestWorkingTreeDigestExcludesBranchHeadersAndHidesPaths(t *testing.T) {
	t.Parallel()
	entry := "? private/customer-name.txt\x00"
	left, err := parsePorcelainV2([]byte("# branch.oid " + strings.Repeat("a", 40) + "\x00# branch.head main\x00" + entry))
	if err != nil {
		t.Fatal(err)
	}
	right, err := parsePorcelainV2([]byte("# branch.oid " + strings.Repeat("b", 40) + "\x00# branch.head other\x00" + entry))
	if err != nil {
		t.Fatal(err)
	}
	if left.workingTree.Digest != right.workingTree.Digest {
		t.Fatalf("tree digest included HEAD/branch: %q != %q", left.workingTree.Digest, right.workingTree.Digest)
	}
	if strings.Contains(left.workingTree.Digest, "private") || strings.Contains(left.workingTree.Digest, "customer") {
		t.Fatalf("tree digest leaked a path: %q", left.workingTree.Digest)
	}
}

func TestParsePorcelainV2SkipsRenameSourceHeaderInjection(t *testing.T) {
	t.Parallel()
	output := statusFixture(
		"2 R. N... 100644 100644 100644 aaaaaaa bbbbbbb R100 destination.go",
		"# branch.oid "+strings.Repeat("f", 40),
	)
	parsed, err := parsePorcelainV2(output)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.head != strings.Repeat("a", 40) || parsed.workingTree.Staged != 1 {
		t.Fatalf("rename source was parsed as metadata: %+v", parsed)
	}
}

func TestParsePorcelainV2DetachedUnbornAndClean(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     string
		detached bool
		unborn   bool
	}{
		{"detached", "# branch.oid " + strings.Repeat("b", 40) + "\x00# branch.head (detached)\x00", true, false},
		{"unborn", "# branch.oid (initial)\x00# branch.head main\x00", false, true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := parsePorcelainV2([]byte(test.data))
			if err != nil {
				t.Fatal(err)
			}
			if parsed.detached != test.detached || parsed.unborn != test.unborn || parsed.workingTree.State != WorkingTreeClean {
				t.Fatalf("parsed = %+v", parsed)
			}
		})
	}
}

func TestParsePorcelainV2RejectsMalformedAndConflictingMetadata(t *testing.T) {
	t.Parallel()
	for _, output := range [][]byte{
		[]byte("# branch.oid nope\x00"),
		[]byte("# branch.head main\x00# branch.head other\x00"),
		[]byte("2 R. N... incomplete\x00"),
		[]byte("x unknown\x00"),
	} {
		if _, err := parsePorcelainV2(output); err == nil {
			t.Fatalf("malformed status accepted: %q", output)
		}
	}
}

func TestDirtyCountsAreBounded(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	output.WriteString("# branch.oid " + strings.Repeat("a", 40) + "\x00# branch.head main\x00")
	for index := 0; index < maxDirtyCount+2; index++ {
		output.WriteString("? path\x00")
	}
	parsed, err := parsePorcelainV2(output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.workingTree.Untracked != maxDirtyCount || !parsed.workingTree.CountsTruncated {
		t.Fatalf("untracked bound = %+v", parsed.workingTree)
	}
}
