package workspace

import (
	"errors"
	"strings"
	"testing"
)

func TestBoundedBufferDrainsWithoutRetainingExcess(t *testing.T) {
	t.Parallel()
	var buffer boundedBuffer
	input := []byte(strings.Repeat("x", maxGitOutputBytes+10))
	written, err := buffer.Write(input)
	if err != nil || written != len(input) || buffer.Len() != maxGitOutputBytes || !buffer.exceeded {
		t.Fatalf("written=%d len=%d exceeded=%t err=%v", written, buffer.Len(), buffer.exceeded, err)
	}
}

func TestExecGitRunnerRejectsCommandsOutsideReadOnlyAllowlist(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"fetch"},
		{"pull", "origin"},
		{"clone", "https://example.invalid/repo"},
		{"-c"},
		{"config", "--global", "user.name", "attacker"},
		{"config", "--unset", "remote.origin.url"},
		{"status", "--porcelain=v1"},
		{"rev-list", "--objects", "HEAD"},
	} {
		if _, err := (ExecGitRunner{}).Run(t.Context(), t.TempDir(), args...); !errors.Is(err, ErrUnsafeGitCommand) {
			t.Fatalf("args=%v error=%v", args, err)
		}
	}
}

func TestSafeGitEnvironmentRemovesRepositoryAndConfigInjection(t *testing.T) {
	t.Parallel()
	environment := safeGitEnvironment([]string{
		"PATH=/bin", "GIT_DIR=/attacker", "GIT_WORK_TREE=/wrong",
		"GIT_CONFIG_COUNT=2", "GIT_CONFIG_KEY_1=core.fsmonitor",
		"GIT_CONFIG_VALUE_1=malicious", "GIT_CONFIG_PARAMETERS='alias.status=!steal'",
		"GIT_TRACE=/tmp/private-trace", "GIT_CEILING_DIRECTORIES=/wrong", "LC_ALL=host",
	})
	joined := strings.Join(environment, "\n")
	for _, forbidden := range []string{"/attacker", "/wrong", "malicious", "steal", "private-trace", "LC_ALL=host"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("unsafe environment retained %q: %s", forbidden, joined)
		}
	}
	for _, required := range []string{"PATH=/bin", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1", "LC_ALL=C"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("safe environment missing %q: %s", required, joined)
		}
	}
}
