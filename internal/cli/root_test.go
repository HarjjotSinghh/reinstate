package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func runCLI(t *testing.T, name string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Execute(Options{
		Name:   name,
		Stdout: &out,
		Stderr: &errb,
		Args:   args,
	})
	return out.String(), errb.String(), code
}

func TestNoArgsShowsHelpExit2(t *testing.T) {
	out, _, code := runCLI(t, "reinstate")
	if code != ExitUsage {
		t.Fatalf("exit=%d want %d out=%q", code, ExitUsage, out)
	}
	if !strings.Contains(out, "Usage") && !strings.Contains(out, "usage") && !strings.Contains(strings.ToLower(out), "reinstate") {
		// Cobra help may go to stdout
		if out == "" {
			t.Fatalf("expected help output, got empty")
		}
	}
}

func TestHelpExit0(t *testing.T) {
	out, _, code := runCLI(t, "reinstate", "--help")
	if code != ExitOK {
		t.Fatalf("exit=%d want 0 out=%q", code, out)
	}
	if !strings.Contains(out, "version") {
		t.Fatalf("help missing version: %q", out)
	}
}

func TestUnknownCommandExit2(t *testing.T) {
	_, errb, code := runCLI(t, "reinstate", "not-a-real-command")
	if code != ExitUsage {
		t.Fatalf("exit=%d want %d stderr=%q", code, ExitUsage, errb)
	}
}

func TestVersionJSON(t *testing.T) {
	out, _, code := runCLI(t, "reinstate", "version", "--json")
	if code != ExitOK {
		t.Fatalf("exit=%d out=%q", code, out)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if m["name"] != "reinstate" {
		t.Fatalf("name=%q", m["name"])
	}
	if m["version"] == "" {
		t.Fatal("empty version")
	}
}

func TestExitErrorMapping(t *testing.T) {
	if ExitCodeFrom(nil) != ExitOK {
		t.Fatal("nil")
	}
	if ExitCodeFrom(NewExitError(ExitConfig, "x")) != ExitConfig {
		t.Fatal("config")
	}
	if ExitCodeFrom(NewExitError(ExitConflict, "c")) != ExitConflict {
		t.Fatal("conflict")
	}
}

func TestReinAndReinstateSame(t *testing.T) {
	out1, _, c1 := runCLI(t, "rein", "version", "--json")
	out2, _, c2 := runCLI(t, "reinstate", "version", "--json")
	if c1 != ExitOK || c2 != ExitOK {
		t.Fatalf("codes %d %d", c1, c2)
	}
	var a, b map[string]string
	_ = json.Unmarshal([]byte(out1), &a)
	_ = json.Unmarshal([]byte(out2), &b)
	if a["version"] != b["version"] {
		t.Fatalf("version mismatch %q vs %q", a["version"], b["version"])
	}
}
