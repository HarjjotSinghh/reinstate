package agentcheck

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	output VersionOutput
	err    error
	name   string
	args   []string
	ctxErr error
}

func (runner *fakeRunner) Version(ctx context.Context, name string, args ...string) (VersionOutput, error) {
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
			runner := &fakeRunner{output: VersionOutput{Stdout: test.version}}
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
		{name: "missing layout", agent: "codex", output: "codex-cli 0.146.0", want: StatusUntested, message: "layout"},
		{name: "unknown version", agent: "codex", makeLayout: true, output: "secret vendor response", want: StatusUntested, message: "unrecognized"},
		{name: "unverified version", agent: "claude", makeLayout: true, output: "9.9.9 (Claude Code)", want: StatusUntested, message: "verified range"},
		{name: "probe error", agent: "claude", makeLayout: true, runErr: errors.New("token-secret"), want: StatusError, message: "probe failed"},
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
			runner := &fakeRunner{output: VersionOutput{Stdout: test.output}, err: test.runErr}
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
	runner := &fakeRunner{output: VersionOutput{Stdout: "2.1.220 (Claude Code)"}}
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

func TestVendorVersionGrammarRejectsAmbiguousOutput(t *testing.T) {
	tests := []struct {
		name   string
		parse  func(VersionOutput) (string, bool)
		output VersionOutput
		want   string
		ok     bool
	}{
		{name: "claude canonical", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220 (Claude Code)\n"}, want: "2.1.220", ok: true},
		{name: "claude windows newline", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220 (Claude Code)\r\n"}, want: "2.1.220", ok: true},
		{name: "codex canonical", parse: parseCodexVersion, output: VersionOutput{Stdout: "codex-cli 0.146.0\n"}, want: "0.146.0", ok: true},
		{name: "hostile prefix", parse: parseClaudeVersion, output: VersionOutput{Stdout: "hostile 2.1.220 (Claude Code)\n"}},
		{name: "hostile suffix", parse: parseCodexVersion, output: VersionOutput{Stdout: "codex-cli 0.146.0 trusted\n"}},
		{name: "multiple lines", parse: parseCodexVersion, output: VersionOutput{Stdout: "codex-cli 0.146.0\ncodex-cli 0.145.0\n"}},
		{name: "multiple versions", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220 (Claude Code) 9.9.9"}},
		{name: "ansi prefix", parse: parseCodexVersion, output: VersionOutput{Stdout: "\x1b[32mcodex-cli 0.146.0\x1b[0m\n"}},
		{name: "embedded control", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220\x00 (Claude Code)\n"}},
		{name: "stderr warning", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220 (Claude Code)\n", Stderr: "warning: update available\n"}},
		{name: "stderr control whitespace", parse: parseClaudeVersion, output: VersionOutput{Stdout: "2.1.220 (Claude Code)\n", Stderr: "\n"}},
		{name: "wrong vendor grammar", parse: parseCodexVersion, output: VersionOutput{Stdout: "0.146.0\n"}},
		{name: "leading whitespace", parse: parseClaudeVersion, output: VersionOutput{Stdout: " 2.1.220 (Claude Code)\n"}},
		{name: "noncanonical component", parse: parseCodexVersion, output: VersionOutput{Stdout: "codex-cli 00.146.0\n"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.parse(test.output)
			if got != test.want || ok != test.ok {
				t.Fatalf("parse(%q, %q) = %q, %t", test.output.Stdout, test.output.Stderr, got, ok)
			}
		})
	}
}

func TestInspectDoesNotSkipAnEarlierDefaultVendorRoot(t *testing.T) {
	home := t.TempDir()
	first := filepath.Join(home, ".claude")
	second := filepath.Join(home, ".config", "claude", "projects")
	if err := os.MkdirAll(first, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o700); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{output: VersionOutput{Stdout: "2.1.220 (Claude Code)\n"}}
	result := Inspect(context.Background(), "claude", Options{
		Home:     home,
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "/verified/claude", nil },
		Runner:   runner,
	})
	if result.Status != StatusUntested || result.LayoutRecognized || runner.name != "" {
		t.Fatalf("result/runner = %+v / %q", result, runner.name)
	}
}

func TestInspectUsesVendorEnvironmentRootWithoutDisclosingIt(t *testing.T) {
	tests := []struct {
		agent       string
		environment string
		marker      string
		version     string
	}{
		{agent: "claude", environment: "CLAUDE_CONFIG_DIR", marker: "projects", version: "2.1.220 (Claude Code)\n"},
		{agent: "codex", environment: "CODEX_HOME", marker: "sessions", version: "codex-cli 0.146.0\n"},
	}
	for _, test := range tests {
		t.Run(test.agent, func(t *testing.T) {
			configuredRoot := filepath.Join(t.TempDir(), "private-agent-config")
			if err := os.MkdirAll(filepath.Join(configuredRoot, test.marker), 0o700); err != nil {
				t.Fatal(err)
			}
			var requested []string
			result := Inspect(context.Background(), test.agent, Options{
				Home: t.TempDir(),
				LookPath: func(value string) (string, error) {
					return "/verified/" + value, nil
				},
				Getenv: func(name string) string {
					requested = append(requested, name)
					if name == test.environment {
						return configuredRoot
					}
					return ""
				},
				Runner: &fakeRunner{output: VersionOutput{Stdout: test.version}},
			})
			if result.Status != StatusSupported || strings.Join(requested, ",") != test.environment {
				t.Fatalf("result/requested = %+v / %v", result, requested)
			}
			if strings.Contains(fmt.Sprintf("%+v", result), configuredRoot) {
				t.Fatalf("result disclosed configured root: %+v", result)
			}
		})
	}
}

func TestInspectExplicitRootOverridesVendorEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	result := Inspect(context.Background(), "codex", Options{
		Root: root,
		Getenv: func(string) string {
			t.Fatal("Getenv called with an explicit root")
			return ""
		},
		LookPath: func(string) (string, error) { return "/verified/codex", nil },
		Runner:   &fakeRunner{output: VersionOutput{Stdout: "codex-cli 0.146.0\n"}},
	})
	if result.Status != StatusSupported {
		t.Fatalf("result = %+v", result)
	}
}

func TestInspectRejectsSymlinkedLayoutPaths(t *testing.T) {
	targetRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(targetRoot, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		root func(*testing.T) string
	}{
		{
			name: "root symlink",
			root: func(t *testing.T) string {
				link := filepath.Join(t.TempDir(), "claude-root")
				if err := os.Symlink(targetRoot, link); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				return link
			},
		},
		{
			name: "escaping marker symlink",
			root: func(t *testing.T) string {
				root := t.TempDir()
				if err := os.Symlink(filepath.Join(targetRoot, "projects"), filepath.Join(root, "projects")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				return root
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeRunner{output: VersionOutput{Stdout: "2.1.220 (Claude Code)\n"}}
			result := Inspect(context.Background(), "claude", Options{
				Root:     test.root(t),
				LookPath: func(string) (string, error) { return "/verified/claude", nil },
				Runner:   runner,
			})
			if result.Status != StatusUntested || result.LayoutRecognized || runner.name != "" {
				t.Fatalf("result/runner = %+v / %q", result, runner.name)
			}
		})
	}
}

func TestExecRunnerSeparatesAndBoundsOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GO_WANT_AGENTCHECK_HELPER_PROCESS", "1")
	t.Setenv("AGENTCHECK_HELPER_STDOUT", "codex-cli 0.146.0\n")
	t.Setenv("AGENTCHECK_HELPER_STDERR", "warning\n")
	t.Setenv("AGENTCHECK_HELPER_EXIT", "0")
	output, err := (ExecRunner{}).Version(context.Background(), executable, "-test.run=^TestAgentCheckHelperProcess$")
	if err != nil || output.Stdout != "codex-cli 0.146.0\n" || output.Stderr != "warning\n" {
		t.Fatalf("output/error = %+v / %v", output, err)
	}

	for _, test := range []struct {
		name   string
		stdout string
		stderr string
	}{
		{name: "stdout", stdout: "abcdef"},
		{name: "stderr", stderr: "abcdef"},
	} {
		t.Run(test.name+" overflow", func(t *testing.T) {
			t.Setenv("AGENTCHECK_HELPER_STDOUT", test.stdout)
			t.Setenv("AGENTCHECK_HELPER_STDERR", test.stderr)
			output, err := (ExecRunner{MaxOutput: 3}).Version(context.Background(), executable, "-test.run=^TestAgentCheckHelperProcess$")
			if err == nil || output != (VersionOutput{}) {
				t.Fatalf("bounded output/error = %+v / %v", output, err)
			}
		})
	}
}

func TestAgentCheckHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_AGENTCHECK_HELPER_PROCESS") != "1" {
		return
	}
	_, _ = fmt.Fprint(os.Stdout, os.Getenv("AGENTCHECK_HELPER_STDOUT"))
	_, _ = fmt.Fprint(os.Stderr, os.Getenv("AGENTCHECK_HELPER_STDERR"))
	if os.Getenv("AGENTCHECK_HELPER_EXIT") != "0" {
		os.Exit(7)
	}
	os.Exit(0)
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
