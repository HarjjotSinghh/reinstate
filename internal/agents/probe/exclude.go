package probe

import (
	"path"
	"path/filepath"
	"strings"
)

func isExcluded(rel, name string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
		pattern = strings.TrimPrefix(pattern, "./")
		if pattern == "" {
			continue
		}
		base := path.Base(pattern)
		if strings.EqualFold(name, base) {
			return true
		}
		if matchGlob(pattern, rel) {
			return true
		}
		lowerRel := strings.ToLower(rel)
		lowerPat := strings.ToLower(strings.Trim(pattern, "/"))
		if lowerPat != "" && (lowerRel == lowerPat || strings.HasPrefix(lowerRel, lowerPat+"/")) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, name string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "./")
	name = strings.TrimPrefix(filepath.ToSlash(name), "./")
	return matchGlobParts(splitSlash(pattern), splitSlash(name))
}

func splitSlash(value string) []string {
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

func matchGlobParts(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			if len(pattern) == 1 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchGlobParts(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		ok, err := path.Match(pattern[0], name[0])
		if err != nil || !ok {
			return false
		}
		pattern = pattern[1:]
		name = name[1:]
	}
	return len(name) == 0
}
