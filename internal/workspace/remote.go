package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxRemoteBytes = 4096

var ErrRemoteIdentityUnavailable = errors.New("portable repository identity is unavailable")

// RepositoryIDFromRemote converts a supported network Git remote into an
// opaque, credential-free identifier. Raw URLs are never returned.
func RepositoryIDFromRemote(raw string) (string, error) {
	normalized, err := normalizeRemote(raw)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte("reinstate.repository.remote.v1\x00" + normalized))
	return "remote-sha256:" + hex.EncodeToString(digest[:]), nil
}

func normalizeRemote(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxRemoteBytes || !utf8.ValidString(raw) || hasUnsafeRemoteRune(raw) {
		return "", ErrRemoteIdentityUnavailable
	}

	var host, port, repositoryPath, scheme string
	if !strings.Contains(raw, "://") {
		// SCP-like Git syntax: user@host:path. A drive path such as C:\repo is
		// deliberately not accepted as a portable repository identity.
		colon := strings.IndexByte(raw, ':')
		if colon <= 0 || strings.ContainsAny(raw[:colon], `/\`) {
			return "", ErrRemoteIdentityUnavailable
		}
		left := raw[:colon]
		repositoryPath = raw[colon+1:]
		if at := strings.LastIndexByte(left, '@'); at >= 0 {
			left = left[at+1:]
		}
		if left == "" || repositoryPath == "" || len(left) == 1 {
			return "", ErrRemoteIdentityUnavailable
		}
		host = left
		scheme = "ssh"
	} else if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" {
		scheme = strings.ToLower(parsed.Scheme)
		switch scheme {
		case "http", "https", "ssh", "git":
		default:
			return "", ErrRemoteIdentityUnavailable
		}
		if parsed.Hostname() == "" || parsed.Opaque != "" {
			return "", ErrRemoteIdentityUnavailable
		}
		host = parsed.Hostname()
		port = parsed.Port()
		escapedPath := parsed.EscapedPath()
		if hasEncodedPathSeparator(escapedPath) {
			return "", ErrRemoteIdentityUnavailable
		}
		repositoryPath, err = url.PathUnescape(escapedPath)
		if err != nil {
			return "", ErrRemoteIdentityUnavailable
		}
	} else {
		return "", ErrRemoteIdentityUnavailable
	}

	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" || strings.ContainsAny(host, `/\@`) {
		return "", ErrRemoteIdentityUnavailable
	}
	if parsedIP := net.ParseIP(host); parsedIP == nil {
		for _, label := range strings.Split(host, ".") {
			if label == "" {
				return "", ErrRemoteIdentityUnavailable
			}
		}
	}
	if defaultPort(scheme, port) {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	}

	if strings.Contains(repositoryPath, "\\") {
		return "", ErrRemoteIdentityUnavailable
	}
	repositoryPath = strings.Trim(repositoryPath, "/")
	if strings.HasSuffix(strings.ToLower(repositoryPath), ".git") {
		repositoryPath = repositoryPath[:len(repositoryPath)-4]
	}
	repositoryPath = strings.TrimSuffix(repositoryPath, "/")
	if repositoryPath == "" || repositoryPath == "." || repositoryPath == ".." {
		return "", ErrRemoteIdentityUnavailable
	}
	for _, segment := range strings.Split(repositoryPath, "/") {
		if segment == "." || segment == ".." {
			return "", ErrRemoteIdentityUnavailable
		}
	}
	cleaned := pathpkg.Clean("/" + repositoryPath)
	if cleaned == "/" || strings.Contains(cleaned, "/../") || strings.HasSuffix(cleaned, "/..") {
		return "", ErrRemoteIdentityUnavailable
	}
	repositoryPath = strings.TrimPrefix(cleaned, "/")
	if hasUnsafeRemoteRune(repositoryPath) {
		return "", ErrRemoteIdentityUnavailable
	}
	return host + "/" + repositoryPath, nil
}

func hasEncodedPathSeparator(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "%2f") || strings.Contains(value, "%5c")
}

func hasUnsafeRemoteRune(value string) bool {
	for _, current := range value {
		if unicode.IsControl(current) || unicode.In(current, unicode.Cf) {
			return true
		}
	}
	return false
}

func defaultPort(scheme, port string) bool {
	if port == "" {
		return true
	}
	switch scheme {
	case "http":
		return port == "80"
	case "https":
		return port == "443"
	case "ssh":
		return port == "22"
	case "git":
		return port == "9418"
	default:
		return false
	}
}

type remoteIdentity struct {
	name string
	id   string
}

func parseRemoteIdentities(output []byte) []remoteIdentity {
	var identities []remoteIdentity
	for _, record := range strings.Split(string(output), "\x00") {
		key, raw, ok := strings.Cut(record, "\n")
		if !ok || !strings.HasPrefix(key, "remote.") || !strings.HasSuffix(key, ".url") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(key, "remote."), ".url")
		if name == "" {
			continue
		}
		id, err := RepositoryIDFromRemote(raw)
		if err != nil {
			continue
		}
		identities = append(identities, remoteIdentity{name: name, id: id})
	}
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].name != identities[j].name {
			return identities[i].name < identities[j].name
		}
		return identities[i].id < identities[j].id
	})
	return identities
}

func selectRemoteIdentity(identities []remoteIdentity) (string, []string) {
	if len(identities) == 0 {
		return "", nil
	}
	unique := make(map[string]struct{}, len(identities))
	var all []string
	primary := ""
	for _, identity := range identities {
		if _, exists := unique[identity.id]; !exists {
			unique[identity.id] = struct{}{}
			all = append(all, identity.id)
		}
		if identity.name == "origin" && primary == "" {
			primary = identity.id
		}
	}
	if primary == "" {
		primary = identities[0].id
	}
	sort.Strings(all)
	return primary, all
}

func repositoryIDFromRoots(roots []string) string {
	sort.Strings(roots)
	digest := sha256.New()
	_, _ = digest.Write([]byte("reinstate.repository.roots.v1\x00"))
	for _, root := range roots {
		_, _ = digest.Write([]byte(root))
		_, _ = digest.Write([]byte{0})
	}
	return "roots-sha256:" + hex.EncodeToString(digest.Sum(nil))
}
