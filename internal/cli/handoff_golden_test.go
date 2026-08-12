package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestHandoffCLIGoldens(t *testing.T) {
	tests := []struct {
		name   string
		asJSON bool
	}{
		{name: "human"},
		{name: "json", asJSON: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home, vendorHome, sources, transcriptPath := handoffCLIFixture(t)
			args := []string{"handoff", "codex:source-session", "--to", "claude", "--dry-run"}
			if test.asJSON {
				args = append(args, "--json")
			}
			stdout, stderr, code := runHandoffCLI(t, home, vendorHome, sources, &recordingLaunchRunner{}, args...)
			if code != ExitOK || stderr != "" {
				t.Fatalf("exit=%d stderr=%q", code, stderr)
			}
			got := normalizeHandoffGolden(t, stdout, test.asJSON, map[string]string{
				home:                         "${REINSTATE_HOME}",
				filepath.Dir(transcriptPath): "${WORKSPACE}",
			})
			compareCLIGolden(t, "handoff-dry-run-"+test.name+map[bool]string{true: ".json", false: ".txt"}[test.asJSON], got)
		})
	}
}

func TestHandoffExecutedOutputMatchesDryRunByteForByte(t *testing.T) {
	home, vendorHome, sources, _ := handoffCLIFixture(t)
	runner := &recordingLaunchRunner{}
	dryOut, dryErr, dryCode := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude", "--dry-run")
	if dryCode != ExitOK || dryErr != "" {
		t.Fatalf("dry-run exit=%d stderr=%q", dryCode, dryErr)
	}
	executedOut, executedErr, executedCode := runHandoffCLI(t, home, vendorHome, sources, runner,
		"handoff", "codex:source-session", "--to", "claude")
	if executedCode != ExitOK || executedErr != "" {
		t.Fatalf("executed exit=%d stderr=%q", executedCode, executedErr)
	}
	if len(runner.plans) != 1 {
		t.Fatalf("fake runner launches=%d want 1", len(runner.plans))
	}
	if dryOut != executedOut {
		t.Fatalf("dry-run/executed output differs\n--- dry-run ---\n%s\n--- executed ---\n%s", dryOut, executedOut)
	}
}

func normalizeHandoffGolden(t *testing.T, raw string, asJSON bool, replacements map[string]string) []byte {
	t.Helper()
	return normalizeHandoffGoldenForOS(t, raw, asJSON, replacements, runtime.GOOS == "windows")
}

func normalizeHandoffGoldenForOS(t *testing.T, raw string, asJSON bool, replacements map[string]string, windows bool) []byte {
	t.Helper()
	type replacement struct{ from, to string }
	var variants []replacement
	seen := map[string]bool{}
	for root, token := range replacements {
		clean := filepath.Clean(root)
		for _, variant := range []string{root, clean, filepath.ToSlash(clean), strings.ReplaceAll(clean, `\`, "/")} {
			for _, candidate := range []string{variant, jsonQuoted(t, variant)} {
				if candidate != "" && !seen[candidate] {
					seen[candidate] = true
					variants = append(variants, replacement{from: candidate, to: token})
				}
			}
		}
	}
	sort.Slice(variants, func(i, j int) bool { return len(variants[i].from) > len(variants[j].from) })
	for _, variant := range variants {
		raw = strings.ReplaceAll(raw, variant.from, variant.to)
	}
	if asJSON {
		if windows {
			raw = strings.ReplaceAll(raw, `\\`, "/")
		}
		return []byte(raw)
	}
	normalize := func(value string) string {
		if windows {
			value = strings.ReplaceAll(value, `\n`, "\x00newline\x00")
			value = strings.ReplaceAll(value, `\\`, "/")
			value = strings.ReplaceAll(value, `\`, "/")
			value = strings.ReplaceAll(value, "\x00newline\x00", `\n`)
		}
		return value
	}
	return []byte(normalize(raw))
}

func jsonQuoted(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Trim(string(encoded), `"`)
}

func TestNormalizeHandoffGoldenWindowsQuotedPaths(t *testing.T) {
	raw := `command "claude" "Read C:\\Users\\fixture\\reinstate\\handoffs\\id\\projection.md\nnext"` + "\n" +
		`file C:\Users\fixture\reinstate\handoffs\id\capsule.json cwd=C:\Users\fixture\workspace` + "\n"
	got := string(normalizeHandoffGoldenForOS(t, raw, false, map[string]string{
		`C:\Users\fixture\reinstate`: "${REINSTATE_HOME}",
		`C:\Users\fixture\workspace`: "${WORKSPACE}",
	}, true))
	for _, want := range []string{"${REINSTATE_HOME}/handoffs/id/projection.md", "${WORKSPACE}", `projection.md\nnext`} {
		if !strings.Contains(got, want) {
			t.Fatalf("normalized output omitted %q: %s", want, got)
		}
	}
	for _, forbidden := range []string{`C:\`, "C:/", "${REINSTATE_HOME}//"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("normalized output contains %q: %s", forbidden, got)
		}
	}
}

func compareCLIGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	if bytes.Contains(got, []byte{'\r'}) || bytes.Contains(got, []byte("/Users/")) || bytes.Contains(got, []byte("/home/")) {
		t.Fatalf("golden %s contains a non-portable newline or host path", name)
	}
	path := filepath.Join("..", "..", "testdata", "handoff", "golden", "cli", name)
	if os.Getenv("UPDATE_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (set UPDATE_GOLDENS=1 to create)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
