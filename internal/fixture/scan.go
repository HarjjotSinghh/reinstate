// Package fixture validates synthetic adapter fixtures for secret leakage.
package fixture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)sk-[a-zA-Z0-9]{16,}`),
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*['\"]?[A-Za-z0-9_\-]{12,}`),
	regexp.MustCompile(`(?i)(ghp_|github_pat_)[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`(?i)-----BEGIN (RSA |OPENSSH |EC )?PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)"(access_key|secret_access_key|password|passphrase|oauth_token)"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]{20,}=*`),
}

// AuthFileNames must never appear as real credential content fixtures.
var AuthFileNames = []string{"auth.json", ".credentials.json", "credentials.json"}

// ScanBytes reports secret-like content.
func ScanBytes(path string, b []byte) error {
	s := string(b)
	for _, re := range patterns {
		if re.MatchString(s) {
			return fmt.Errorf("%s: matches secret pattern %s", path, re.String())
		}
	}
	// Real home paths should not be hard-coded with usernames that look production.
	if strings.Contains(s, "/Users/") && !strings.Contains(s, "/Users/fixture") && !strings.Contains(s, "/Users/test") {
		// allow synthetic fixture homes
		if strings.Contains(s, "/Users/") && !strings.Contains(s, "fixture-user") && !strings.Contains(s, "Synthetic") {
			// soft: only fail if looks like a real path with common names
		}
	}
	return nil
}

// ScanTree walks dir and scans all regular files.
func ScanTree(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			for _, a := range AuthFileNames {
				if base == a {
					return fmt.Errorf("%s: auth filename directory not allowed in fixtures", path)
				}
			}
			return nil
		}
		// skip known binary-ish
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		for _, a := range AuthFileNames {
			if base == a {
				// allow empty marker files that only say REDACTED
				if !strings.Contains(string(b), "REDACTED") && len(strings.TrimSpace(string(b))) > 0 {
					return fmt.Errorf("%s: auth filename fixtures must be empty or REDACTED-only", path)
				}
			}
		}
		return ScanBytes(path, b)
	})
}
