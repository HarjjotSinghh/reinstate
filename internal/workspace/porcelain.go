package workspace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const maxDirtyCount = 10_000

var ErrInvalidGitStatus = errors.New("git status returned invalid porcelain data")

type parsedStatus struct {
	branch      string
	head        string
	detached    bool
	unborn      bool
	upstream    string
	upstreamSet bool
	relation    CommitRelation
	workingTree WorkingTreeFingerprint
}

func parsePorcelainV2(output []byte) (parsedStatus, error) {
	treeDigest := sha256.New()
	_, _ = treeDigest.Write([]byte("reinstate.workspace.working-tree.v1\x00"))
	result := parsedStatus{
		relation:    CommitRelation{Relation: RelationUnknown, LocalOnly: true},
		workingTree: WorkingTreeFingerprint{State: WorkingTreeClean},
	}
	workingTreeUncertain := false
	tokens := bytes.Split(output, []byte{0})
	seen := make(map[string]string)
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if len(token) == 0 {
			continue
		}
		line := string(token)
		switch token[0] {
		case '#':
			key, value, ok := strings.Cut(strings.TrimPrefix(line, "# "), " ")
			if !ok {
				return parsedStatus{}, ErrInvalidGitStatus
			}
			if previous, exists := seen[key]; exists && previous != value {
				return parsedStatus{}, fmt.Errorf("%w: conflicting %s header", ErrInvalidGitStatus, key)
			}
			seen[key] = value
			switch key {
			case "branch.oid":
				if value == "(initial)" {
					result.unborn = true
					result.head = ""
				} else if validObjectID(value) {
					result.head = strings.ToLower(value)
				} else {
					return parsedStatus{}, fmt.Errorf("%w: invalid branch object", ErrInvalidGitStatus)
				}
			case "branch.head":
				if value == "(detached)" {
					result.detached = true
					result.branch = ""
				} else {
					result.branch = safeMetadata(value, 1024)
				}
			case "branch.upstream":
				result.upstreamSet = true
				result.upstream = safeMetadata(value, 1024)
			case "branch.ab":
				fields := strings.Fields(value)
				if len(fields) != 2 || !strings.HasPrefix(fields[0], "+") || !strings.HasPrefix(fields[1], "-") {
					return parsedStatus{}, fmt.Errorf("%w: invalid branch relation", ErrInvalidGitStatus)
				}
				ahead, aheadErr := strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
				behind, behindErr := strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
				if aheadErr != nil || behindErr != nil || ahead < 0 || behind < 0 {
					return parsedStatus{}, fmt.Errorf("%w: invalid branch relation counts", ErrInvalidGitStatus)
				}
				result.relation = relationFromCounts(ahead, behind)
			case "reinstate.working-tree":
				if value != "uncertain" {
					return parsedStatus{}, fmt.Errorf("%w: invalid Reinstate working-tree state", ErrInvalidGitStatus)
				}
				workingTreeUncertain = true
			}
		case '1', '2':
			_, _ = treeDigest.Write(token)
			_, _ = treeDigest.Write([]byte{0})
			fields := strings.Fields(line)
			if len(fields) < 3 || len(fields[1]) != 2 {
				return parsedStatus{}, ErrInvalidGitStatus
			}
			result.workingTree.State = WorkingTreeModified
			incrementChangeCounts(&result.workingTree, fields[1], fields[2])
			if token[0] == '2' {
				// Porcelain -z emits the rename source as a separate token. It is
				// a pathname, not another status/header record.
				index++
				if index >= len(tokens) || len(tokens[index]) == 0 {
					return parsedStatus{}, fmt.Errorf("%w: missing rename source", ErrInvalidGitStatus)
				}
				_, _ = treeDigest.Write(tokens[index])
				_, _ = treeDigest.Write([]byte{0})
			}
		case 'u':
			_, _ = treeDigest.Write(token)
			_, _ = treeDigest.Write([]byte{0})
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return parsedStatus{}, ErrInvalidGitStatus
			}
			result.workingTree.State = WorkingTreeModified
			boundedIncrement(&result.workingTree.Conflicted, &result.workingTree.CountsTruncated)
			if submoduleChanged(fields[2]) {
				boundedIncrement(&result.workingTree.Submodule, &result.workingTree.CountsTruncated)
			}
		case '?':
			_, _ = treeDigest.Write(token)
			_, _ = treeDigest.Write([]byte{0})
			result.workingTree.State = WorkingTreeModified
			boundedIncrement(&result.workingTree.Untracked, &result.workingTree.CountsTruncated)
		case '!':
			// Ignored paths are not requested, but accepting them keeps the
			// parser forward-compatible without treating them as modifications.
		default:
			return parsedStatus{}, fmt.Errorf("%w: unknown record type", ErrInvalidGitStatus)
		}
	}
	if !result.upstreamSet {
		result.relation = CommitRelation{Relation: RelationUnknown, LocalOnly: true}
	}
	if _, ok := seen["branch.oid"]; !ok {
		return parsedStatus{}, fmt.Errorf("%w: missing branch object", ErrInvalidGitStatus)
	}
	if _, ok := seen["branch.head"]; !ok {
		return parsedStatus{}, fmt.Errorf("%w: missing branch head", ErrInvalidGitStatus)
	}
	result.workingTree.Uncertain = workingTreeUncertain
	result.workingTree.Digest = "sha256:" + hex.EncodeToString(treeDigest.Sum(nil))
	return result, nil
}

func incrementChangeCounts(tree *WorkingTreeFingerprint, xy, submodule string) {
	if xy[0] != '.' {
		boundedIncrement(&tree.Staged, &tree.CountsTruncated)
	}
	if xy[1] != '.' {
		boundedIncrement(&tree.Unstaged, &tree.CountsTruncated)
	}
	if submoduleChanged(submodule) {
		boundedIncrement(&tree.Submodule, &tree.CountsTruncated)
	}
}

func submoduleChanged(value string) bool {
	return len(value) >= 4 && value[0] == 'S' && value[1:] != "..."
}

func boundedIncrement(value *int, truncated *bool) {
	if *value >= maxDirtyCount {
		*truncated = true
		return
	}
	*value++
}

func relationFromCounts(ahead, behind int) CommitRelation {
	relation := RelationEqual
	switch {
	case ahead > 0 && behind > 0:
		relation = RelationDiverged
	case ahead > 0:
		relation = RelationAhead
	case behind > 0:
		relation = RelationBehind
	}
	return CommitRelation{
		Relation:  relation,
		Ahead:     ahead,
		Behind:    behind,
		Knowable:  true,
		LocalOnly: true,
	}
}
