package probe

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
)

var (
	reUUID    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reUUIDSub = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	reHex32   = regexp.MustCompile(`(?i)^[0-9a-f]{32}$`)
	reHex40   = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)
	reHex64   = regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)
	rePrefHex = regexp.MustCompile(`(?i)^([a-z][a-z0-9_]*)_([0-9a-f]{32})$`)
	// A vendor prefix joined to a long content hash, e.g. Git's
	// pack-<40-hex>.idx under a marketplace checkout. Without this the
	// trailing-digits rule split the hash and left most of it verbatim.
	reLongHexTail = regexp.MustCompile(`(?i)^([a-z][a-z0-9_.]*)[-_]([0-9a-f]{32,})$`)
	reTrailing    = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9._-]*?)[-_]?([0-9]+)$`)
	reDigits      = regexp.MustCompile(`^[0-9]+$`)
	reSafeName    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]*$`)
	// reWorkspaceBucket matches a short prefix, a free-form stem, and a
	// content hash: Kimi Code's wd_<workspace>_<12-hex>. The stem is the
	// basename of the working directory, so it is a repository name.
	//
	// A native Windows probe produced wd_portfolio-25_6d65015f0cb0 and this
	// rule is why that no longer reaches an artifact. The earlier macOS probe
	// produced wd_<account>_<hex> and was redacted only by accident, because
	// that session happened to run in the home directory, whose basename is
	// the account name — which is also why the shape was first misread as
	// carrying a username.
	reWorkspaceBucket = regexp.MustCompile(`(?i)^([a-z][a-z0-9]{0,7})_(.+)_([0-9a-f]{8,64})$`)
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
	if loc := reUUIDSub.FindStringIndex(stem); loc != nil {
		prefix := strings.TrimRight(stem[:loc[0]], "-_")
		rest := strings.Trim(stem[loc[1]:], "-_.")
		switch {
		case prefix == "" && rest == "":
			return "<uuid-v4>"
		case prefix == "":
			return "<uuid-v4>-" + normalizeStem(rest)
		case rest == "":
			return normalizeStem(prefix) + "-<uuid-v4>"
		default:
			return normalizeStem(prefix) + "-<uuid-v4>-" + normalizeStem(rest)
		}
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
	if m := reLongHexTail.FindStringSubmatch(stem); len(m) == 3 {
		return fmt.Sprintf("%s-<%d-hex>", m[1], len(m[2]))
	}
	if m := reWorkspaceBucket.FindStringSubmatch(stem); len(m) == 4 {
		return fmt.Sprintf("%s_<project>_<%d-hex>", m[1], len(m[3]))
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
	if looksLikeEncodedPath(stem) {
		return "<path-slug>"
	}
	// Hyphenated leftovers are project or workspace names, not vendor
	// constants. reSafeName accepts [A-Za-z0-9._-], which would otherwise
	// keep those filenames verbatim in a committed artifact.
	if strings.Contains(stem, "-") {
		return "<slug>"
	}
	if reSafeName.MatchString(stem) && !looksIdentifying(stem) {
		return stem
	}
	return "<slug>"
}

// filesystemRoots are the first path component of an absolute path on a
// supported platform, plus single letters for Windows drives.
var filesystemRoots = map[string]bool{
	"users": true, "home": true, "var": true, "private": true, "tmp": true,
	"mnt": true, "opt": true, "srv": true, "media": true, "root": true,
	"volumes": true, "documents": true,
}

// looksLikeEncodedPath reports whether a component is an absolute path with
// its separators rewritten, the scheme Cursor uses for project buckets:
// /Users/<user>/Documents/Projects/demo becomes
// Users-<user>-Documents-Projects-demo.
//
// Such a component is an absolute path and a repository name wearing a
// disguise, and the token normalizer would otherwise pass it through intact
// because every character in it is unremarkable. Detection anchors on the
// first segment being a filesystem root rather than on counting segments,
// which keeps vendor bucket prefixes like Kimi's wd_<user>_<hash> readable.
func looksLikeEncodedPath(stem string) bool {
	fields := strings.FieldsFunc(stem, func(r rune) bool { return r == '-' || r == '_' })
	if len(fields) < 3 {
		return false
	}
	head := strings.ToLower(fields[0])
	if filesystemRoots[head] {
		return true
	}
	// A Windows drive letter, as in C-Users-<user>-src.
	return len(head) == 1 && head[0] >= 'a' && head[0] <= 'z' &&
		strings.EqualFold(fields[1], "users")
}

// redactUser strips the operating-system account name from an already
// normalized component. Vendors embed it in bucket directory names — Kimi Code
// uses wd_<user>_<hash> — and the token-based normalizer preserves such a stem
// verbatim because nothing about it looks like an identifier. Probe artifacts
// are committed, so the account name is removed after normalization, which
// keeps the informative structure around it.
func redactUser(shape, user string) string {
	if len(user) < 3 || shape == "" {
		return shape
	}
	lowerUser := strings.ToLower(user)
	var b strings.Builder
	for {
		idx := strings.Index(strings.ToLower(shape), lowerUser)
		if idx < 0 {
			b.WriteString(shape)
			return b.String()
		}
		b.WriteString(shape[:idx])
		b.WriteString("<user>")
		shape = shape[idx+len(user):]
	}
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
