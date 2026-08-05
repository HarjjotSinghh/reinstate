package runtimecheck

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	versions map[string]string
	errors   map[string]error
	calls    []string
}

type contextBlockingRunner struct{}

func (contextBlockingRunner) Version(ctx context.Context, _ string, _ ...string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (r *fakeRunner) Version(_ context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if err := r.errors[name]; err != nil {
		return "", err
	}
	return r.versions[name], nil
}

func TestInspectRecognizedNodeDeclarations(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		contents    string
		actual      string
		wantStatus  Status
		wantSource  string
		wantDeclare string
	}{
		{name: "exact", file: ".nvmrc", contents: "v20.11.1\n", actual: "v20.11.1\n", wantStatus: StatusMatch, wantSource: "nvmrc", wantDeclare: "v20.11.1"},
		{name: "major", file: ".node-version", contents: "20\n", actual: "v20.19.0\n", wantStatus: StatusMatch, wantSource: "node_version_file", wantDeclare: "20"},
		{name: "caret mismatch", file: "package.json", contents: `{"engines":{"node":"^20.11.0"}}`, actual: "v21.0.0\n", wantStatus: StatusChanged, wantSource: "package_json_engines", wantDeclare: "^20.11.0"},
		{name: "major range", file: "package.json", contents: `{"engines":{"node":">=20 <21"}}`, actual: "v20.12.2\n", wantStatus: StatusMatch, wantSource: "package_json_engines", wantDeclare: ">=20 <21"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			if err := os.WriteFile(filepath.Join(workspace, test.file), []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{versions: map[string]string{"node": test.actual}}
			results := Inspect(context.Background(), workspace, Options{Runner: runner})
			if len(results) != 1 {
				t.Fatalf("results = %+v", results)
			}
			got := results[0]
			if got.Status != test.wantStatus || got.SourceKind != test.wantSource || got.Declared != test.wantDeclare {
				t.Fatalf("result = %+v", got)
			}
			if len(runner.calls) != 1 || runner.calls[0] != "node --version" {
				t.Fatalf("calls = %v", runner.calls)
			}
		})
	}
}

func TestInspectGoToolchainPrecedesGoDirective(t *testing.T) {
	workspace := t.TempDir()
	contents := "module example.com/demo\n\ngo 1.24.0\ntoolchain go1.25.12\n"
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{versions: map[string]string{"go": "go version go1.25.13 darwin/arm64\n"}}
	results := Inspect(context.Background(), workspace, Options{Runner: runner})
	if len(results) != 1 || results[0].Status != StatusMatch ||
		results[0].Declared != "1.25.12" || results[0].Actual != "1.25.13" ||
		results[0].SourceKind != "go_mod_toolchain" {
		t.Fatalf("results = %+v", results)
	}
}

func TestInspectGoDirectivesAreCommentAwareAndStrict(t *testing.T) {
	tests := []struct {
		name       string
		contents   string
		wantStatus Status
		wantSource string
		wantValue  string
		wantCalls  int
	}{
		{
			name:       "trailing comments",
			contents:   "module example.com/demo\n\ngo 1.24.0 // language floor\ntoolchain go1.25.12 // preferred local toolchain\n",
			wantStatus: StatusMatch,
			wantSource: "go_mod_toolchain",
			wantValue:  "1.25.12",
			wantCalls:  1,
		},
		{
			name:       "default toolchain falls back to go directive",
			contents:   "module example.com/demo\n\ngo 1.25.0\ntoolchain default\n",
			wantStatus: StatusMatch,
			wantSource: "go_mod_go",
			wantValue:  "1.25.0",
			wantCalls:  1,
		},
		{
			name:       "duplicate go directive",
			contents:   "module example.com/demo\n\ngo 1.24.0\ngo 1.25.0\n",
			wantStatus: StatusUnknown,
			wantSource: "go_mod_go",
			wantCalls:  0,
		},
		{
			name:       "malformed toolchain directive",
			contents:   "module example.com/demo\n\ngo 1.25.0\ntoolchain 1.25.12\n",
			wantStatus: StatusUnknown,
			wantSource: "go_mod_toolchain",
			wantCalls:  0,
		},
		{
			name:       "node range syntax is invalid for go",
			contents:   "module example.com/demo\n\ngo ^1.25.0\n",
			wantStatus: StatusUnknown,
			wantSource: "go_mod_go",
			wantCalls:  0,
		},
		{
			name:       "go directive requires minor version",
			contents:   "module example.com/demo\n\ngo 1\n",
			wantStatus: StatusUnknown,
			wantSource: "go_mod_go",
			wantCalls:  0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{versions: map[string]string{"go": "go version go1.25.12 test/arch\n"}}
			results := Inspect(context.Background(), workspace, Options{Runner: runner})
			if len(results) != 1 || results[0].Status != test.wantStatus ||
				results[0].SourceKind != test.wantSource || len(runner.calls) != test.wantCalls {
				t.Fatalf("results/calls = %+v / %v", results, runner.calls)
			}
			if test.wantValue != "" && results[0].Declared != test.wantValue {
				t.Fatalf("declared = %q, want %q", results[0].Declared, test.wantValue)
			}
		})
	}
}

func TestInspectMissingAndUnknownDoNotLeak(t *testing.T) {
	workspace := t.TempDir()
	secret := "token-should-never-appear"
	if err := os.WriteFile(filepath.Join(workspace, ".nvmrc"), []byte("\x1b[31m"+secret+"\u202e\nsecond-line"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{
		versions: map[string]string{},
		errors:   map[string]error{"node": errors.New("raw-secret-from-path")},
	}
	results := Inspect(context.Background(), workspace, Options{Runner: runner})
	if len(results) != 1 || results[0].Status != StatusUnknown {
		t.Fatalf("results = %+v", results)
	}
	rendered := results[0].Message + results[0].Declared + results[0].Actual
	if strings.Contains(rendered, secret) || strings.Contains(rendered, "raw-secret") || strings.Contains(rendered, "\x1b") || strings.Contains(rendered, "\u202e") {
		t.Fatalf("unsafe result = %+v", results[0])
	}
}

func TestInspectDoesNotFollowDeclarationSymlink(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret-version")
	if err := os.WriteFile(outside, []byte("99.0.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, ".nvmrc")); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{versions: map[string]string{"node": "v99.0.0"}}
	results := Inspect(context.Background(), workspace, Options{Runner: runner})
	if len(results) != 1 || results[0].Status != StatusUnknown || len(runner.calls) != 0 {
		t.Fatalf("results/calls = %+v / %v", results, runner.calls)
	}
}

func TestReadBoundedRegularFileRejectsSymlinkSwapAfterLstat(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, ".nvmrc")
	outside := filepath.Join(t.TempDir(), "outside-version")
	if err := os.WriteFile(path, []byte("20.11.1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("99.0.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	var symlinkErr error
	data, state := readBoundedRegularFileWithOpener(path, func(value string) (*os.File, error) {
		if err := os.Remove(value); err != nil {
			return nil, err
		}
		symlinkErr = os.Symlink(outside, value)
		if symlinkErr != nil {
			return nil, symlinkErr
		}
		return os.Open(value)
	})
	if symlinkErr != nil {
		t.Skipf("symlink replacement unavailable: %v", symlinkErr)
	}
	if state != declarationUnknown || data != nil {
		t.Fatalf("state/data = %v / %q", state, data)
	}
}

func TestReadBoundedRegularFileRejectsDifferentOpenedIdentity(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "declared-version")
	other := filepath.Join(directory, "different-version")
	if err := os.WriteFile(path, []byte("20.11.1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("99.0.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, state := readBoundedRegularFileWithOpener(path, func(string) (*os.File, error) {
		return os.Open(other)
	})
	if state != declarationUnknown || data != nil {
		t.Fatalf("state/data = %v / %q", state, data)
	}
}

func TestInspectRejectsInvalidWorkspaceWithoutRunning(t *testing.T) {
	notDirectory := filepath.Join(t.TempDir(), "workspace-file")
	if err := os.WriteFile(notDirectory, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, workspace := range []string{"", notDirectory, filepath.Join(t.TempDir(), "missing")} {
		runner := &fakeRunner{versions: map[string]string{"node": "v20.11.1"}}
		results := Inspect(context.Background(), workspace, Options{Runner: runner})
		if len(results) != 0 || len(runner.calls) != 0 {
			t.Fatalf("workspace %q results/calls = %+v / %v", workspace, results, runner.calls)
		}
	}
}

func TestInspectAbsentDeclarationsRunsNothing(t *testing.T) {
	runner := &fakeRunner{versions: map[string]string{}}
	results := Inspect(context.Background(), t.TempDir(), Options{Runner: runner})
	if len(results) != 0 || len(runner.calls) != 0 {
		t.Fatalf("results/calls = %+v / %v", results, runner.calls)
	}
}

func TestLimitedBufferBoundsOutput(t *testing.T) {
	var buffer limitedBuffer
	buffer.limit = 4
	if n, err := buffer.Write([]byte("123456")); err != nil || n != 6 {
		t.Fatalf("write = %d, %v", n, err)
	}
	if buffer.String() != "1234" || !buffer.overflow {
		t.Fatalf("buffer = %q overflow=%t", buffer.String(), buffer.overflow)
	}
}

func TestLimitedBufferBoundsIOCopyFastPath(t *testing.T) {
	var buffer limitedBuffer
	buffer.limit = 4
	count, err := io.Copy(&buffer, strings.NewReader("123456"))
	if err != nil || count != 6 {
		t.Fatalf("copy = %d, %v", count, err)
	}
	if buffer.String() != "1234" || !buffer.overflow {
		t.Fatalf("buffer = %q overflow=%t", buffer.String(), buffer.overflow)
	}
}

func TestInspectDistinguishesMissingRuntimeFromProbeFailure(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".nvmrc"), []byte("20.11.1"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		err  error
		want Status
	}{
		{name: "missing", err: ErrExecutableNotFound, want: StatusMissing},
		{name: "execution failure", err: errors.New("private child failure"), want: StatusError},
		{name: "output overflow", err: ErrOutputLimit, want: StatusError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeRunner{errors: map[string]error{"node": test.err}}
			results := Inspect(context.Background(), workspace, Options{Runner: runner})
			if len(results) != 1 || results[0].Status != test.want {
				t.Fatalf("results = %+v", results)
			}
			if strings.Contains(results[0].Message, "private") {
				t.Fatalf("private runner error leaked: %+v", results[0])
			}
		})
	}
}

func TestInspectTimeoutIsBoundedInfrastructureError(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".nvmrc"), []byte("20.11.1"), 0o600); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	results := Inspect(context.Background(), workspace, Options{Runner: contextBlockingRunner{}, Timeout: 25 * time.Millisecond})
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timeout took %s", elapsed)
	}
	if len(results) != 1 || results[0].Status != StatusError {
		t.Fatalf("results = %+v", results)
	}
}

func TestSanitizedEnvironmentRemovesExecutionAndNetworkControls(t *testing.T) {
	node := sanitizedEnvironment("node", []string{"Path=/bin", "Node_Options=--require=secret.js", "SAFE=value"})
	if joined := strings.Join(node, "\n"); strings.Contains(strings.ToUpper(joined), "NODE_OPTIONS=") || !strings.Contains(joined, "SAFE=value") {
		t.Fatalf("node environment = %v", node)
	}

	goEnvironment := sanitizedEnvironment("go", []string{
		"PATH=/bin", "GoToolchain=go1.99.0+auto", "GOENV=secret", "GOWORK=secret", "GOPROXY=https://network.invalid", "SAFE=value",
	})
	joined := strings.Join(goEnvironment, "\n")
	for _, required := range []string{"GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS="} {
		if strings.Count(joined, required) != 1 {
			t.Fatalf("go environment missing unique %q: %v", required, goEnvironment)
		}
	}
	if strings.Contains(joined, "network.invalid") || strings.Contains(joined, "go1.99.0") || !strings.Contains(joined, "SAFE=value") {
		t.Fatalf("unsafe go environment = %v", goEnvironment)
	}
}

func TestExecRunnerSanitizesNodeEnvironmentAndWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	unsafeDirectory := t.TempDir()
	scriptDirectory := t.TempDir()
	writeProbeScript(t, filepath.Join(scriptDirectory, "node"), `#!/bin/sh
if [ -n "${NODE_OPTIONS+x}" ]; then exit 40; fi
if [ "$PWD" = "$UNSAFE_PROBE_CWD" ]; then exit 41; fi
printf 'v20.11.1\n'
`)
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(unsafeDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })
	t.Setenv("PATH", scriptDirectory)
	t.Setenv("NODE_OPTIONS", "--require=private-project-code.js")
	t.Setenv("UNSAFE_PROBE_CWD", unsafeDirectory)

	output, err := (ExecRunner{}).Version(context.Background(), "node", "--version")
	if err != nil || strings.TrimSpace(output) != "v20.11.1" {
		t.Fatalf("output/error = %q / %v", output, err)
	}
}

func TestExecRunnerForcesLocalOfflineGoProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	scriptDirectory := t.TempDir()
	writeProbeScript(t, filepath.Join(scriptDirectory, "go"), `#!/bin/sh
if [ "$GOTOOLCHAIN" != "local" ]; then exit 50; fi
if [ "$GOENV" != "off" ]; then exit 51; fi
if [ "$GOWORK" != "off" ]; then exit 52; fi
if [ "$GOPROXY" != "off" ]; then exit 53; fi
if [ "$GOSUMDB" != "off" ]; then exit 54; fi
if [ -n "$GOFLAGS" ]; then exit 55; fi
printf 'go version go1.25.12 test/arch\n'
`)
	t.Setenv("PATH", scriptDirectory)
	t.Setenv("GOTOOLCHAIN", "go1.99.0+auto")
	t.Setenv("GOENV", "private-go-env")
	t.Setenv("GOWORK", "private-go-work")
	t.Setenv("GOPROXY", "https://network.invalid")
	t.Setenv("GOSUMDB", "sum.network.invalid")
	t.Setenv("GOFLAGS", "-mod=mod")

	output, err := (ExecRunner{}).Version(context.Background(), "go", "version")
	if err != nil || strings.TrimSpace(output) != "go version go1.25.12 test/arch" {
		t.Fatalf("output/error = %q / %v", output, err)
	}
}

func TestExecRunnerDoesNotHonorForcedGoToolchainSelection(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go executable unavailable: %v", err)
	}
	t.Setenv("GOTOOLCHAIN", "go1.99.0+auto")
	t.Setenv("GOPROXY", "https://network.invalid")
	output, err := (ExecRunner{}).Version(context.Background(), "go", "version")
	if err != nil {
		t.Fatalf("local Go version probe failed: %v", err)
	}
	if strings.Contains(strings.ToLower(output), "download") {
		t.Fatalf("Go probe attempted toolchain selection: %q", output)
	}
	if _, ok := parseGoOutput(output); !ok {
		t.Fatalf("Go probe returned unrecognized output: %q", output)
	}
}

func TestExecRunnerTimeoutAndOutputLimitAreTyped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	t.Run("timeout", func(t *testing.T) {
		directory := t.TempDir()
		writeProbeScript(t, filepath.Join(directory, "node"), "#!/bin/sh\nwhile :; do :; done\n")
		t.Setenv("PATH", directory)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, err := (ExecRunner{}).Version(ctx, "node", "--version")
		if elapsed := time.Since(started); elapsed > time.Second {
			t.Fatalf("timeout took %s", elapsed)
		}
		if !errors.Is(err, ErrProbeFailed) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("overflow", func(t *testing.T) {
		directory := t.TempDir()
		writeProbeScript(t, filepath.Join(directory, "node"), "#!/bin/sh\nprintf '%5000s' x\n")
		t.Setenv("PATH", directory)
		output, err := (ExecRunner{MaxOutput: 32}).Version(context.Background(), "node", "--version")
		if !errors.Is(err, ErrProbeFailed) || !errors.Is(err, ErrOutputLimit) {
			t.Fatalf("output length/error = %d / %v", len(output), err)
		}
	})
}

func writeProbeScript(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestParseConstraintSemantics(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		actual     string
		wantMatch  bool
	}{
		{name: "caret major", constraint: "^20.11.0", actual: "20.99.0", wantMatch: true},
		{name: "caret major upper bound", constraint: "^20.11.0", actual: "21.0.0", wantMatch: false},
		{name: "caret zero minor", constraint: "^0.2.3", actual: "0.3.0", wantMatch: false},
		{name: "caret zero patch", constraint: "^0.0.3", actual: "0.0.4", wantMatch: false},
		{name: "caret partial zero", constraint: "^0.0", actual: "0.1.0", wantMatch: false},
		{name: "tilde major", constraint: "~20", actual: "20.99.0", wantMatch: true},
		{name: "tilde major upper bound", constraint: "~20", actual: "21.0.0", wantMatch: false},
		{name: "tilde minor", constraint: "~20.11", actual: "20.12.0", wantMatch: false},
		{name: "partial minor", constraint: "20.11", actual: "20.11.9", wantMatch: true},
		{name: "exclusive lower excludes boundary", constraint: ">20 <21", actual: "20.0.0", wantMatch: false},
		{name: "exclusive lower accepts next patch", constraint: ">20 <21", actual: "20.0.1", wantMatch: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			constraint, ok := parseConstraint(test.constraint)
			if !ok {
				t.Fatalf("constraint %q was rejected", test.constraint)
			}
			actual, _, ok := parseVersion(test.actual)
			if !ok {
				t.Fatalf("actual %q was rejected", test.actual)
			}
			if got := constraint.match(actual); got != test.wantMatch {
				t.Fatalf("match(%q, %q) = %t, want %t", test.constraint, test.actual, got, test.wantMatch)
			}
		})
	}
}

func TestRuntimeOutputParsersRejectAmbiguousOrPrereleaseVersions(t *testing.T) {
	for _, output := range []string{"v20.11.0-rc.1", "warning v20.11.0", "v20.11.0 extra"} {
		if _, ok := parseNodeOutput(output); ok {
			t.Fatalf("accepted Node output %q", output)
		}
	}
	for _, output := range []string{
		"warning go1.25.12 go version go1.99.0 test/arch",
		"go version go1.25.12-rc.1 test/arch",
		"go version go1.25.12 test/arch trailing",
	} {
		if _, ok := parseGoOutput(output); ok {
			t.Fatalf("accepted Go output %q", output)
		}
	}
}

func TestParseConstraintRejectsUnsupportedSyntax(t *testing.T) {
	for _, value := range []string{
		"*", "latest", "20 || 22", "<=20", "https://secret.invalid",
		"20.11.0-rc.1", "20.11.0+private-build", "020.11.0", "020.x", ">=020 <021",
	} {
		if _, ok := parseConstraint(value); ok {
			t.Fatalf("accepted %q", value)
		}
	}
}
