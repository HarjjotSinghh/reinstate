package adapter

import (
	"strconv"
	"strings"
)

// StableVersionInRange reports whether version is a stable three-component
// semantic version inside the inclusive tested range.
func StableVersionInRange(version, minimum, maximum string) bool {
	got, ok := parseStableVersion(version)
	if !ok {
		return false
	}
	minimumVersion, ok := parseStableVersion(minimum)
	if !ok {
		return false
	}
	maximumVersion, ok := parseStableVersion(maximum)
	if !ok {
		return false
	}
	return compareVersion(got, minimumVersion) >= 0 && compareVersion(got, maximumVersion) <= 0
}

// StableVersionFromOutput returns the first whitespace-separated field of a
// vendor `--version` output that parses as a stable three-component version.
//
// Vendors disagree about where the version sits — "2.1.220 (Claude Code)"
// versus "codex-cli 0.145.0" — and can change the wording between releases or
// between platform packages. Indexing a fixed field silently reports a
// supported vendor as untested, which blocks sync writes, so callers scan
// instead. An empty result means no field looked like a version.
func StableVersionFromOutput(output string) string {
	for _, field := range strings.Fields(output) {
		candidate := strings.TrimPrefix(strings.Trim(field, "()[],;"), "v")
		if _, ok := parseStableVersion(candidate); ok {
			return candidate
		}
	}
	return ""
}

func parseStableVersion(value string) ([3]uint64, bool) {
	var version [3]uint64
	parts := strings.Split(value, ".")
	if len(parts) != len(version) {
		return version, false
	}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return version, false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return version, false
			}
		}
		component, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return version, false
		}
		version[index] = component
	}
	return version, true
}

func compareVersion(left, right [3]uint64) int {
	for index := range left {
		switch {
		case left[index] < right[index]:
			return -1
		case left[index] > right[index]:
			return 1
		}
	}
	return 0
}
