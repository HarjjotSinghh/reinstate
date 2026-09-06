package doctor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reAPIKey    = regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{10,}|api[_-]?key[=:]\s*\S+|ghp_[A-Za-z0-9]{20,}|xox[baprs]-[A-Za-z0-9-]{10,})`)
	reQueryCred = regexp.MustCompile(`(?i)([?&](token|key|password|secret|access_key|secret_key)=)[^&\s]+`)
	reBearer    = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`)
)

// Redact removes usernames, home paths, credentials, and session-like secrets.
func Redact(s string) string {
	if s == "" {
		return s
	}
	out := s
	if rh := strings.TrimSpace(os.Getenv("REINSTATE_HOME")); rh != "" {
		out = redactPathRoot(out, rh, "${REINSTATE_HOME}")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		out = redactPathRoot(out, home, "${HOME}")
		// username segment
		base := filepath.Base(home)
		if base != "" && base != "/" && base != "\\" {
			out = strings.ReplaceAll(out, "/"+base+"/", "/${USER}/")
			out = strings.ReplaceAll(out, "\\"+base+"\\", "\\${USER}\\")
		}
	}
	out = reAPIKey.ReplaceAllString(out, "[REDACTED]")
	out = reQueryCred.ReplaceAllString(out, "${1}[REDACTED]")
	out = reBearer.ReplaceAllString(out, "Bearer [REDACTED]")
	// auth filenames
	for _, name := range []string{"auth.json", ".credentials.json", "credentials.json", ".env"} {
		out = strings.ReplaceAll(out, name, "[AUTH_FILE]")
	}
	// session-ish long blobs
	if len(out) > 400 && strings.Contains(strings.ToLower(out), "session") {
		out = out[:200] + "…[REDACTED_SESSION]"
	}
	return out
}

// redactPathRoot replaces a configured home path only when it is a complete
// path component. A raw prefix replacement would turn /home/alice2 into
// ${HOME}2 and prevent RedactPath from recognizing and removing that absolute
// sibling path.
func redactPathRoot(value, root, replacement string) string {
	root = strings.TrimRight(root, "/\\")
	if root == "" || root == "." || root == "/" || root == `\` ||
		len(root) == 2 && root[1] == ':' {
		return value
	}

	var out strings.Builder
	for start := 0; start < len(value); {
		match := strings.Index(value[start:], root)
		if match < 0 {
			out.WriteString(value[start:])
			break
		}
		match += start
		end := match + len(root)
		if end == len(value) || isPathSeparator(value[end]) {
			out.WriteString(value[start:match])
			out.WriteString(replacement)
			start = end
			continue
		}
		out.WriteString(value[start:end])
		start = end
	}
	return out.String()
}

func isPathSeparator(value byte) bool {
	return value == '/' || value == '\\'
}

// RedactedPathToken is what a removed path is replaced with. It is
// exported so a caller that recognises a form RedactPath cannot — an
// absolute path a harness has flattened into one directory name, say —
// can remove that too and still read as one report.
const RedactedPathToken = "[REDACTED_PATH]"

// RedactPath removes absolute paths from human-facing output while preserving
// already-redacted home tokens.
func RedactPath(p string) string {
	redacted := Redact(p)
	if isAbsolutePath(redacted) {
		return RedactedPathToken
	}
	return redacted
}

func isAbsolutePath(p string) bool {
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return true
	}
	return len(p) >= 3 &&
		((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) &&
		p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}
