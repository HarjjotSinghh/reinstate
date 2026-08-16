package probe

import (
	"path"
	"regexp"
	"strings"
	"unicode"
)

var (
	reUUID     = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reUUIDSub  = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	reHex32    = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)
	reHex40    = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)
	reHex64    = regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)
	rePrefHex  = regexp.MustCompile(`(?i)^([a-z][a-z0-9_]*)_([0-9a-f]{32})$`)
	reTrailing = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9._-]*?)[-_]?([0-9]+)$`)
	reDigits   = regexp.MustCompile(`^[0-9]+$`)
	reSafeName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]*$`)
)

// normalizeComponent collapses identifying names to a token.
func normalizeComponent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return name
	}
	ext := path.Ext(name)
	stem := name
	if ext != "" && len(ext) <= 8 {
		stem = strings.TrimSuffix(name, ext)
	} else {
		ext = ""
	}
	token := normalizeStem(stem)
	if ext == "" {
		return token
	}
	if strings.HasPrefix(token, "<") {
		return token + ext
	}
	return token + ext
}

func normalizeStem(stem string) string {
	if reUUID.MatchString(stem) {
		return "<uuid-v4>"
	}
	if loc := reUUIDSub.FindStringIndex(stem); loc != nil && loc[0] > 0 {
		prefix := strings.TrimRight(stem[:loc[0]], "-_")
		if prefix == "" {
			return "<uuid-v4>"
		}
		return normalizeStem(prefix) + "-<uuid-v4>"
	}
	if reHex32.MatchString(stem) {
		return "<32-hex>"
	}
	if reHex40.MatchString(stem) {
		return "<40-hex>"
	}
	if reHex64.MatchString(stem) {
		return "<64-hex>"
	}
	if m := rePrefHex.FindStringSubmatch(stem); len(m) == 3 {
		return m[1] + "_<32-hex>"
	}
	if isSlug(stem) {
		return "<slug>"
	}
	if reDigits.MatchString(stem) {
		return "<n>"
	}
	if m := reTrailing.FindStringSubmatch(stem); len(m) == 3 && len(m[2]) >= 2 {
		return m[1] + "-<n>"
	}
	if reSafeName.MatchString(stem) && !looksIdentifying(stem) {
		return stem
	}
	return "<slug>"
}

func isSlug(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%3a") ||
		strings.Contains(name, "/") || strings.Contains(name, `\`) {
		return true
	}
	if strings.HasPrefix(name, "-") && strings.Count(name, "-") >= 2 {
		return true
	}
	if len(name) >= 4 && name[1] == '-' && name[2] == '-' &&
		unicode.IsLetter(rune(name[0])) {
		return true
	}
	return false
}

func looksIdentifying(name string) bool {
	if strings.Contains(name, "@") {
		return true
	}
	letters := 0
	uppers := 0
	for _, r := range name {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				uppers++
			}
		}
	}
	return letters >= 6 && uppers >= 2 && uppers*2 >= letters
}

func treePath(components []string) string {
	parts := make([]string, len(components))
	for i, c := range components {
		if strings.HasPrefix(c, "<") {
			parts[i] = "*"
		} else {
			parts[i] = c
		}
	}
	return strings.Join(parts, "/")
}

func shapeOf(component string) string {
	if strings.HasPrefix(component, "<") {
		return component
	}
	return component
}
