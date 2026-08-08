package doctest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsStageReleaseAssetsSkipsArtifactsWithoutExtra(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows PowerShell contract")
	}
	powerShell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("Windows PowerShell unavailable")
	}

	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	rawDir := filepath.Join(dist, "raw")
	if err := os.MkdirAll(rawDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fixtures := []struct {
		name     string
		artifact string
		contents string
		staged   bool
	}{
		{
			name:     "metadata",
			artifact: `{"name":"metadata","path":"dist/artifacts.json","internal_type":"Metadata","type":"Metadata"}`,
		},
		{
			name:     "not-raw.exe",
			artifact: `{"name":"not-raw.exe","path":"dist/raw/not-raw.exe","internal_type":"Binary","type":"Binary"}`,
		},
		{
			name:     "rein-upper.exe",
			artifact: `{"name":"rein-upper.exe","path":"dist/raw/rein-upper.exe","internal_type":"Binary","type":"Binary","extra":{"ID":"raw"}}`,
			contents: "synthetic uppercase raw release binary",
			staged:   true,
		},
		{
			name:     "rein-lower.exe",
			artifact: `{"name":"rein-lower.exe","path":"dist/raw/rein-lower.exe","internal_type":"Binary","type":"Binary","extra":{"id":"raw"}}`,
			contents: "synthetic lowercase raw release binary",
			staged:   true,
		},
	}
	artifacts := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		artifacts = append(artifacts, fixture.artifact)
		if !fixture.staged {
			continue
		}
		if err := os.WriteFile(filepath.Join(rawDir, fixture.name), []byte(fixture.contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dist, "artifacts.json"), []byte("["+strings.Join(artifacts, ",")+"]"), 0o600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(
		powerShell,
		"-NoProfile",
		"-File", filepath.Join(repoRoot(t), "scripts", "stage-release-assets.ps1"),
		"-DistDir", "dist",
	)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("stage-release-assets.ps1 failed: %v\n%s", err, output)
	}
	for _, fixture := range fixtures {
		if !fixture.staged {
			continue
		}
		staged, err := os.ReadFile(filepath.Join(dist, fixture.name))
		if err != nil {
			t.Fatal(err)
		}
		if string(staged) != fixture.contents {
			t.Fatalf("staged %s = %q, want %q", fixture.name, staged, fixture.contents)
		}
	}
}
