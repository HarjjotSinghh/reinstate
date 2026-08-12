package workspace

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
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

func TestParsePorcelainV2CarriesBoundedUncertainty(t *testing.T) {
	t.Parallel()
	parsed, err := parsePorcelainV2([]byte(
		"# branch.oid " + strings.Repeat("a", 40) +
			"\x00# branch.head main\x00# reinstate.working-tree uncertain\x00",
	))
	if err != nil || !parsed.workingTree.Uncertain {
		t.Fatalf("uncertain status/error = %+v / %v", parsed, err)
	}
	encoded, err := json.Marshal(parsed.workingTree)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"uncertain":true`)) {
		t.Fatalf("uncertainty omitted from JSON: %s", encoded)
	}
	clean, err := json.Marshal(WorkingTreeFingerprint{State: WorkingTreeClean})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(clean, []byte("uncertain")) {
		t.Fatalf("false uncertainty was serialized: %s", clean)
	}
	if _, err := parsePorcelainV2([]byte(
		"# branch.oid " + strings.Repeat("a", 40) +
			"\x00# branch.head main\x00# reinstate.working-tree guessed\x00",
	)); err == nil {
		t.Fatal("invalid private uncertainty header was accepted")
	}
}

func TestParsePorcelainV2TracksStateAndChangedPaths(t *testing.T) {
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
	// The control character in the untracked fixture is stripped: a raw newline
	// inside a path would break every line-oriented rendering downstream.
	want := []string{"conflict.go", "staged.go", "unicode-βname.go", "unstaged-submodule"}
	if !reflect.DeepEqual(tree.Changed, want) || tree.ChangedOmitted != 0 {
		t.Fatalf("changed = %#v omitted=%d, want %#v", tree.Changed, tree.ChangedOmitted, want)
	}
}

// The synthesized subset safeStatus builds from plumbing output carries no mode
// or object-id columns. Both shapes must yield the same pathname, including
// when the pathname itself contains spaces, which -z never quotes.
func TestParsePorcelainV2ReadsPathsFromBothRecordShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		records []string
		want    []string
	}{
		{
			name: "synthesized subset",
			records: []string{
				"1 MM N... calc.go",
				"1 .M N... dir/with space.go",
				"u UU N... merge me.go",
				"? untracked file.txt",
			},
			want: []string{"calc.go", "dir/with space.go", "merge me.go", "untracked file.txt"},
		},
		{
			name: "vendor porcelain with spaces",
			records: []string{
				"1 M. N... 100644 100644 100644 aaaaaaa bbbbbbb dir/with space.go",
				"u UU N... 100644 100644 100644 100644 aaaaaaa bbbbbbb ccccccc merge me.go",
			},
			want: []string{"dir/with space.go", "merge me.go"},
		},
		{
			name: "rename records both names",
			records: []string{
				"2 R. N... 100644 100644 100644 aaaaaaa bbbbbbb R100 new/name.go\x00old/name.go",
			},
			want: []string{"new/name.go", "old/name.go"},
		},
		{
			name:    "one path reported staged and unstaged is listed once",
			records: []string{"1 MM N... calc.go"},
			want:    []string{"calc.go"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parsePorcelainV2(statusFixture(test.records...))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(parsed.workingTree.Changed, test.want) {
				t.Fatalf("changed = %#v, want %#v", parsed.workingTree.Changed, test.want)
			}
		})
	}
}

// A large dirty tree must not silently shrink: the list is capped and the
// remainder is counted so every renderer can say how much it is not showing.
func TestParsePorcelainV2CapsChangedPathsAndCountsTheRemainder(t *testing.T) {
	t.Parallel()
	const total = MaxChangedPaths + 25
	records := make([]string, 0, total)
	for index := 0; index < total; index++ {
		records = append(records, "? file-"+strconv.Itoa(1000+index)+".txt")
	}
	parsed, err := parsePorcelainV2(statusFixture(records...))
	if err != nil {
		t.Fatal(err)
	}
	tree := parsed.workingTree
	if len(tree.Changed) != MaxChangedPaths {
		t.Fatalf("changed list = %d entries, want the %d cap", len(tree.Changed), MaxChangedPaths)
	}
	if tree.ChangedOmitted != total-MaxChangedPaths {
		t.Fatalf("omitted = %d, want %d", tree.ChangedOmitted, total-MaxChangedPaths)
	}
	if !sort.StringsAreSorted(tree.Changed) {
		t.Fatalf("changed list is not deterministic: %#v", tree.Changed)
	}
	if tree.Untracked != total {
		t.Fatalf("counts must stay complete even when the list is capped: %d", tree.Untracked)
	}
}

// A clean tree must never produce a path, in any form.
func TestParsePorcelainV2CleanTreeReportsNoChangedPaths(t *testing.T) {
	t.Parallel()
	parsed, err := parsePorcelainV2(statusFixture())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.workingTree.State != WorkingTreeClean ||
		len(parsed.workingTree.Changed) != 0 || parsed.workingTree.ChangedOmitted != 0 {
		t.Fatalf("clean tree = %+v", parsed.workingTree)
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

func TestParsePorcelainV2DoesNotTrustRelationWithoutUpstream(t *testing.T) {
	t.Parallel()
	parsed, err := parsePorcelainV2([]byte(
		"# branch.oid " + strings.Repeat("a", 40) +
			"\x00# branch.head main\x00# branch.ab +2 -1\x00",
	))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.upstream != "" || parsed.upstreamSet || parsed.relation.Knowable ||
		parsed.relation.Relation != RelationUnknown || !parsed.relation.LocalOnly {
		t.Fatalf("relation without upstream = %+v", parsed)
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
