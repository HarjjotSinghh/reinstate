package doctest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if err := os.WriteFile(filepath.Join(rawDir, "rein.exe"), []byte("synthetic release binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	const artifacts = `[
  {"name":"metadata","path":"dist/artifacts.json","internal_type":"Metadata","type":"Metadata"},
	  {"name":"not-raw.exe","path":"dist/raw/not-raw.exe","internal_type":"Binary","type":"Binary"},
  {"name":"rein.exe","path":"dist/raw/rein.exe","internal_type":"Binary","type":"Binary","extra":{"ID":"raw"}}
]`
	if err := os.WriteFile(filepath.Join(dist, "artifacts.json"), []byte(artifacts), 0o600); err != nil {
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
	staged, err := os.ReadFile(filepath.Join(dist, "rein.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if string(staged) != "synthetic release binary" {
		t.Fatalf("staged binary = %q", staged)
	}
}
