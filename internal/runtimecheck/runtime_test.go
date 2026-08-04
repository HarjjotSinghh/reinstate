package runtimecheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	versions map[string]string
	errors   map[string]error
	calls    []string
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

func TestParseConstraintRejectsUnsupportedSyntax(t *testing.T) {
	for _, value := range []string{"*", "latest", "20 || 22", "<=20", "https://secret.invalid"} {
		if _, ok := parseConstraint(value); ok {
			t.Fatalf("accepted %q", value)
		}
	}
}
