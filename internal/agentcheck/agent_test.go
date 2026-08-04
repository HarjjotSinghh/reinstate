package agentcheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	output string
	err    error
	name   string
	args   []string
	ctxErr error
}

func (runner *fakeRunner) Version(ctx context.Context, name string, args ...string) (string, error) {
	runner.name = name
	runner.args = append([]string(nil), args...)
	runner.ctxErr = ctx.Err()
	return runner.output, runner.err
}

func TestInspectSupportedAgents(t *testing.T) {
	tests := []struct {
		agent   string
		marker  string
		version string
		layout  string
	}{
		{agent: "claude", marker: "projects", version: "2.1.220 (Claude Code)\n", layout: "projects-jsonl"},
		{agent: "codex", marker: "sessions", version: "codex-cli 0.146.0\n", layout: "sessions-rollout-jsonl"},
	}
	for _, test := range tests {
		t.Run(test.agent, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, test.marker), 0o700); err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{output: test.version}
			result := Inspect(context.Background(), test.agent, Options{
				Root: root,
				LookPath: func(value string) (string, error) {
					return "/verified/" + value, nil
				},
				Runner: runner,
			})
			if result.Status != StatusSupported || !result.ExecutablePresent ||
				!result.LayoutRecognized || result.Layout != test.layout {
				t.Fatalf("result = %+v", result)
			}
			if runner.name != "/verified/"+test.agent || strings.Join(runner.args, " ") != "--version" {
				t.Fatalf("runner = %q %v", runner.name, runner.args)
			}
		})
	}
}

func TestInspectFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		agent      string
		lookErr    error
		makeLayout bool
		output     string
		runErr     error
		want       Status
		message    string
	}{
		{name: "missing executable", agent: "claude", lookErr: errors.New("secret path"), want: StatusNotInstalled, message: "executable"},
		{name: "missing layout", agent: "codex", output: "0.146.0", want: StatusUntested, message: "layout"},
		{name: "unknown version", agent: "codex", makeLayout: true, output: "secret vendor response", want: StatusUntested, message: "unrecognized"},
		{name: "unverified version", agent: "claude", makeLayout: true, output: "9.9.9", want: StatusUntested, message: "verified range"},
		{name: "probe error", agent: "claude", makeLayout: true, runErr: errors.New("token-secret"), want: StatusUntested, message: "probe failed"},
		{name: "read only agent", agent: "gemini", want: StatusUntested, message: "does not support"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.makeLayout {
				marker := "projects"
				if test.agent == "codex" {
					marker = "sessions"
				}
				if err := os.Mkdir(filepath.Join(root, marker), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			runner := &fakeRunner{output: test.output, err: test.runErr}
			result := Inspect(context.Background(), test.agent, Options{
				Root: root,
				LookPath: func(value string) (string, error) {
					if test.lookErr != nil {
						return "", test.lookErr
					}
					return "/verified/" + value, nil
				},
				Runner: runner,
			})
			if result.Status != test.want || !strings.Contains(result.Message, test.message) {
				t.Fatalf("result = %+v", result)
			}
			rendered := result.Message + result.Version + result.Layout
			for _, secret := range []string{"secret path", "secret vendor response", "token-secret"} {
				if strings.Contains(rendered, secret) {
					t.Fatalf("result leaked %q: %+v", secret, result)
				}
			}
		})
	}
}

func TestInspectUsesSingleBoundedContext(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{output: "2.1.220"}
	result := Inspect(context.Background(), "claude", Options{
		Root: root,
		LookPath: func(string) (string, error) {
			return "/verified/claude", nil
		},
		Runner:  runner,
		Timeout: time.Second,
	})
	if result.Status != StatusSupported || runner.ctxErr != nil {
		t.Fatalf("result/context = %+v / %v", result, runner.ctxErr)
	}
}

func TestBoundedBufferCapsOutput(t *testing.T) {
	var buffer boundedBuffer
	buffer.limit = 3
	if count, err := buffer.Write([]byte("abcdef")); err != nil || count != 6 {
		t.Fatalf("write = %d, %v", count, err)
	}
	if buffer.String() != "abc" || !buffer.overflow {
		t.Fatalf("buffer = %q overflow=%t", buffer.String(), buffer.overflow)
	}
}
