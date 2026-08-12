package pathmap

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"
)

// ExternalPrefix marks a path that belongs to no configured root.
const ExternalPrefix = "${EXTERNAL:"

// IsToken reports whether a value already is a portable pathmap token.
func IsToken(value string) bool {
	return strings.HasPrefix(value, "${REPO:") ||
		strings.HasPrefix(value, "${HOME}") ||
		strings.HasPrefix(value, "${WORK:") ||
		strings.HasPrefix(value, ExternalPrefix)
}

// IsAbsolutePlatform reports whether value is an absolute path on any supported
// platform: a POSIX path, a Windows drive path, or a UNC path. It is
// deliberately host-independent, because a capsule written on one platform is
// read on another. This is the single definition of "absolute path" in the
// codebase; internal/capsule rejects exactly these values.
func IsAbsolutePlatform(value string) bool {
	if value == "" {
		return false
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\`) {
		return true
	}
	return len(value) >= 3 &&
		((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) &&
		value[1] == ':' && (value[2] == '\\' || value[2] == '/')
}

// ExternalToken returns the portable stand-in for a path outside every
// configured root, for example /etc/hosts or another user's home.
//
// Such a path cannot be rewritten for the destination device and must not be
// carried verbatim: it is not portable, and it usually embeds the operator's
// account name. The token keeps the two properties that remain useful — a
// stable, non-reversible identity so repeated references to the same path stay
// comparable inside one capsule, and the file's base name, which is the only
// part that carries meaning without disclosing a location.
func ExternalToken(platformPath string) string {
	cleaned := toSlash(filepath.Clean(platformPath))
	sum := sha256.Sum256([]byte(cleaned))
	token := ExternalPrefix + hex.EncodeToString(sum[:8]) + "}"
	base := path.Base(cleaned)
	if base == "" || base == "." || base == "/" {
		return token
	}
	return token + "/" + base
}

// NormalizePortable converts a platform path into a value that is always
// portable: a ${REPO:…}, ${HOME}…, or ${WORK:…} token when the path is inside a
// configured root, and an ExternalToken otherwise.
//
// Normalize is deliberately different: it returns unmatched paths unchanged for
// vendor session rewriting, where leaving an unknown path alone is correct.
// Capsule content has no such option, because an absolute path there is both
// unusable on the destination device and rejected by capsule validation.
func (m Mapper) NormalizePortable(platformPath string) string {
	if platformPath == "" || IsToken(platformPath) {
		return platformPath
	}
	if !IsAbsolutePlatform(platformPath) {
		// A relative path is already portable; substituting a token would
		// destroy meaning the destination can still use.
		return platformPath
	}
	normalized := m.Normalize(platformPath)
	if IsToken(normalized) {
		return normalized
	}
	return ExternalToken(platformPath)
}
