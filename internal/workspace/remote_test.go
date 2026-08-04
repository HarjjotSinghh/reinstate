package workspace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRepositoryIDFromRemoteNormalizesTransportAndCredentials(t *testing.T) {
	t.Parallel()
	values := []string{
		"https://token:super-secret@GitHub.com/Owner/repo.git?access_token=hidden#fragment",
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
