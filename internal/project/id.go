// Package project derives canonical project identity.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"strings"
)

// FromGitRemote normalizes a git remote into a canonical project id.
// Priority: normalized git remote > alias > opaque hash.
func FromGitRemote(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	// git@host:org/repo.git
	if strings.HasPrefix(remote, "git@") {
		remote = strings.TrimPrefix(remote, "git@")
		remote = strings.Replace(remote, ":", "/", 1)
		remote = "https://" + remote
	}
	remote = strings.TrimSuffix(remote, ".git")
	u, err := url.Parse(remote)
	if err != nil {
		return normalizePathID(remote)
	}
	host := strings.ToLower(u.Host)
	p := strings.Trim(u.Path, "/")
	return host + "/" + p
}

// OpaqueID creates a stable opaque id from a local root when no remote exists.
func OpaqueID(localRoot string) string {
	sum := sha256.Sum256([]byte(path.Clean(localRoot)))
	return "local/" + hex.EncodeToString(sum[:8])
}

func normalizePathID(s string) string {
	s = strings.TrimSuffix(s, ".git")
	s = strings.ReplaceAll(s, "\\", "/")
	return strings.Trim(s, "/")
}
