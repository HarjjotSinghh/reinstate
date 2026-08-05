package workspace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func syntheticCredentialRemote(user, password, destination, query string) string {
	value := "https://" + user + ":" + password + "@" + destination
	if query != "" {
		value += "?" + query
	}
	return value
}

func TestRepositoryIDFromRemoteNormalizesTransportAndCredentials(t *testing.T) {
	t.Parallel()
	values := []string{
		syntheticCredentialRemote("token", "super-secret", "GitHub.com/Owner/repo.git", "access_token=hidden#fragment"),
		"ssh://git@github.com:22/Owner/repo.git",
		"git@github.com:Owner/repo.git",
	}
	var expected string
	for _, value := range values {
		identity, err := RepositoryIDFromRemote(value)
		if err != nil {
			t.Fatalf("RepositoryIDFromRemote(%q): %v", value, err)
		}
		if !strings.HasPrefix(identity, "remote-sha256:") || len(identity) != len("remote-sha256:")+64 {
			t.Fatalf("invalid opaque identity %q", identity)
		}
		if expected == "" {
			expected = identity
		} else if identity != expected {
			t.Fatalf("transport-specific identities: %q != %q", identity, expected)
		}
		encoded, _ := json.Marshal(identity)
		for _, secret := range []string{"token", "super-secret", "hidden", "Owner", "repo"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("identity leaked %q: %s", secret, encoded)
			}
		}
	}
}

func TestRepositoryIDFromRemoteUsesSchemeSpecificDefaultPorts(t *testing.T) {
	t.Parallel()
	identity := func(remote string) string {
		t.Helper()
		value, err := RepositoryIDFromRemote(remote)
		if err != nil {
			t.Fatalf("RepositoryIDFromRemote(%q): %v", remote, err)
		}
		return value
	}

	if got, want := identity("git://example.com:9418/team/repo.git"), identity("git://example.com/team/repo.git"); got != want {
		t.Fatalf("git default port changed identity: %q != %q", got, want)
	}
	if got, want := identity("git://example.com:22/team/repo.git"), identity("git://example.com/team/repo.git"); got == want {
		t.Fatalf("git non-default port 22 was stripped: %q", got)
	}
	if got, want := identity("ssh://git@example.com:22/team/repo.git"), identity("ssh://git@example.com/team/repo.git"); got != want {
		t.Fatalf("ssh default port changed identity: %q != %q", got, want)
	}
	if got, want := identity("ssh://git@example.com:9418/team/repo.git"), identity("ssh://git@example.com/team/repo.git"); got == want {
		t.Fatalf("ssh non-default port 9418 was stripped: %q", got)
	}
}

func TestRepositoryIDFromRemoteRejectsEncodedPathSeparators(t *testing.T) {
	t.Parallel()
	for _, remote := range []string{
		"https://example.com/team%2frepo.git",
		"https://example.com/team%2Frepo.git",
		"ssh://git@example.com/team%5crepo.git",
		"ssh://git@example.com/team%5Crepo.git",
	} {
		if _, err := RepositoryIDFromRemote(remote); !errors.Is(err, ErrRemoteIdentityUnavailable) {
			t.Fatalf("RepositoryIDFromRemote(%q) error = %v", remote, err)
		}
	}

	literal, err := RepositoryIDFromRemote("https://example.com/team/repo.git")
	if err != nil {
		t.Fatalf("literal path: %v", err)
	}
	encodedUnreserved, err := RepositoryIDFromRemote("https://example.com/%74eam/repo.git")
	if err != nil {
		t.Fatalf("encoded unreserved path: %v", err)
	}
	if encodedUnreserved != literal {
		t.Fatalf("encoded unreserved identity = %q, want %q", encodedUnreserved, literal)
	}
}

func TestRepositoryIDFromRemoteRejectsRawBackslashPath(t *testing.T) {
	t.Parallel()
	for _, remote := range []string{
		`git@example.com:team\repo.git`,
		`https://example.com/team\repo.git`,
	} {
		if _, err := RepositoryIDFromRemote(remote); !errors.Is(err, ErrRemoteIdentityUnavailable) {
			t.Fatalf("RepositoryIDFromRemote(%q) error = %v", remote, err)
		}
	}
	if _, err := RepositoryIDFromRemote("git@example.com:team/repo.git"); err != nil {
		t.Fatalf("SCP slash path: %v", err)
	}
}

func TestRepositoryIDFromRemoteRejectsNonPortableAndUnsafeValues(t *testing.T) {
	t.Parallel()
	tests := []string{
		"", "file:///tmp/repo", "/tmp/repo", `C:\work\repo`,
		"https://github.com/owner/../repo", "https://github.com/repo\nother",
		"https://github.com/repo\u202ename", strings.Repeat("x", maxRemoteBytes+1),
	}
	for _, value := range tests {
		if _, err := RepositoryIDFromRemote(value); !errors.Is(err, ErrRemoteIdentityUnavailable) {
			t.Fatalf("RepositoryIDFromRemote(%q) error = %v", value, err)
		}
	}
}

func TestRemoteSelectionPrefersOriginAndKeepsOpaqueCandidates(t *testing.T) {
	t.Parallel()
	output := []byte("remote.upstream.url\nhttps://example.com/team/upstream.git\x00" +
		"remote.origin.url\ngit@example.com:team/fork.git\x00")
	identities := parseRemoteIdentities(output)
	primary, candidates := selectRemoteIdentity(identities)
	want, err := RepositoryIDFromRemote("https://example.com/team/fork.git")
	if err != nil {
		t.Fatal(err)
	}
	if primary != want || len(candidates) != 2 {
		t.Fatalf("primary=%q candidates=%v", primary, candidates)
	}
}

func TestRepositoryIDFromRootsIsDeterministic(t *testing.T) {
	t.Parallel()
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	if left, right := repositoryIDFromRoots([]string{a, b}), repositoryIDFromRoots([]string{b, a}); left != right {
		t.Fatalf("root identity depends on ordering: %q != %q", left, right)
	}
}
