package doctest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerConsumesGoReleaserAssetContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		testPowerShellInstallerContract(t)
		return
	}
	testPOSIXInstallerContract(t)
}

func testPOSIXInstallerContract(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX installer supports macOS/Linux")
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("unsupported test architecture")
	}

	releaseDir := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir(releaseDir)))
	defer server.Close()
	installDir := filepath.Join(t.TempDir(), "bin")

	asset010 := makePOSIXArchive(t, releaseDir, "0.1.0")
	writeChecksums(t, releaseDir, asset010)
	output, err := runPOSIXInstaller(t, installDir, server.URL, "v0.1.0", nil)
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	destination := filepath.Join(installDir, "reinstate")
	alias := filepath.Join(installDir, "rein")
	for _, path := range []string{destination, alias} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing installed command %s: %v", path, err)
		}
	}
	if !strings.Contains(string(output), "Add "+installDir+" to PATH") {
		t.Fatalf("installer omitted PATH guidance: %s", output)
	}

	marker := []byte("\n# preserve-existing-install\n")
	file, err := os.OpenFile(destination, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(marker); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	output, err = runPOSIXInstaller(t, installDir, server.URL, "v0.1.0", nil)
	if err != nil {
		t.Fatalf("same-version reinstall failed: %v\n%s", err, output)
	}
	assertContainsFile(t, destination, marker)

	asset020 := makePOSIXArchive(t, releaseDir, "0.2.0")
	writeChecksums(t, releaseDir, asset020)
	output, err = runPOSIXInstaller(t, installDir, server.URL, "v0.2.0", nil)
	if err == nil || !strings.Contains(string(output), "refusing to replace") {
		t.Fatalf("unconfirmed replacement was not refused: err=%v\n%s", err, output)
	}
	assertContainsFile(t, destination, marker)
	output, err = runPOSIXInstaller(t, installDir, server.URL, "v0.2.0", []string{"REINSTATE_CONFIRM_REPLACE=1"})
	if err != nil {
		t.Fatalf("confirmed replacement failed: %v\n%s", err, output)
	}
	versionOutput, err := exec.Command(destination, "version", "--json").Output()
	if err != nil || !bytes.Contains(versionOutput, []byte(`"0.2.0"`)) {
		t.Fatalf("installed version mismatch: err=%v output=%s", err, versionOutput)
	}

	writeChecksums(t, releaseDir, asset010)
	output, err = runPOSIXInstaller(t, installDir, server.URL, "v0.1.0", nil)
	if err == nil || !strings.Contains(string(output), "refusing to replace") {
		t.Fatalf("unconfirmed downgrade was not refused: err=%v\n%s", err, output)
	}
	versionOutput, err = exec.Command(destination, "version", "--json").Output()
	if err != nil || !bytes.Contains(versionOutput, []byte(`"0.2.0"`)) {
		t.Fatalf("downgrade refusal replaced existing binary: err=%v output=%s", err, versionOutput)
	}

	asset030 := makePOSIXArchive(t, releaseDir, "0.3.0")
	if err := os.WriteFile(filepath.Join(releaseDir, "checksums.txt"), []byte(strings.Repeat("0", 64)+"  "+asset030+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err = runPOSIXInstaller(t, installDir, server.URL, "v0.3.0", []string{"REINSTATE_CONFIRM_REPLACE=1"})
	if err == nil || !strings.Contains(string(output), "checksum mismatch") {
		t.Fatalf("checksum mismatch was accepted: err=%v\n%s", err, output)
	}
	versionOutput, err = exec.Command(destination, "version", "--json").Output()
	if err != nil || !bytes.Contains(versionOutput, []byte(`"0.2.0"`)) {
		t.Fatalf("checksum failure replaced existing binary: err=%v output=%s", err, versionOutput)
	}

	output, err = runPOSIXInstaller(t, filepath.Join(t.TempDir(), "missing"), server.URL, "v9.9.9", nil)
	if err == nil {
		t.Fatalf("missing asset was accepted: %s", output)
	}
}

func testPowerShellInstallerContract(t *testing.T) {
	powerShell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("PowerShell unavailable")
	}
	releaseDir := t.TempDir()
	asset := makeWindowsArchive(t, releaseDir, "0.1.0")
	writeChecksums(t, releaseDir, asset)
	server := httptest.NewServer(http.FileServer(http.Dir(releaseDir)))
	defer server.Close()

	localAppData := t.TempDir()
	command := exec.Command(powerShell, "-NoProfile", "-File", filepath.Join(repoRoot(t), "scripts", "install.ps1"))
	command.Env = cleanEnv(os.Environ(), "INSTALL_DIR", "REINSTATE_CONFIRM_REPLACE")
	command.Env = append(command.Env,
		"LOCALAPPDATA="+localAppData,
		"REINSTATE_VERSION=v0.1.0",
		"REINSTATE_RELEASE_BASE_URL="+server.URL,
		"REINSTATE_SKIP_VERSION_CHECK=1",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	installDir := filepath.Join(localAppData, "Programs", "Reinstate", "bin")
	for _, name := range []string{"reinstate.exe", "rein.exe"} {
		if _, err := os.Stat(filepath.Join(installDir, name)); err != nil {
			t.Fatalf("missing Windows command %s: %v", name, err)
		}
	}

	badInstallDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(badInstallDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(badInstallDir, "reinstate.exe")
	if err := os.WriteFile(sentinel, []byte("preserve me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "checksums.txt"), []byte(strings.Repeat("0", 64)+"  "+asset+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command = exec.Command(powerShell, "-NoProfile", "-File", filepath.Join(repoRoot(t), "scripts", "install.ps1"))
	command.Env = append(cleanEnv(os.Environ(), "REINSTATE_CONFIRM_REPLACE"),
		"INSTALL_DIR="+badInstallDir,
		"REINSTATE_VERSION=v0.1.0",
		"REINSTATE_RELEASE_BASE_URL="+server.URL,
		"REINSTATE_SKIP_VERSION_CHECK=1",
	)
	if output, err := command.CombinedOutput(); err == nil || !bytes.Contains(output, []byte("checksum mismatch")) {
		t.Fatalf("checksum mismatch was accepted: err=%v\n%s", err, output)
	}
	assertContainsFile(t, sentinel, []byte("preserve me"))
}

func makePOSIXArchive(t *testing.T, releaseDir, version string) string {
	t.Helper()
	asset := fmt.Sprintf("reinstate_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	file, err := os.Create(filepath.Join(releaseDir, asset))
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	body := []byte(fmt.Sprintf(`#!/bin/sh
if [ "${1:-}" = "version" ] && [ "${2:-}" = "--json" ]; then
  printf '{"version":"%s"}\n'
  exit 0
fi
exit 0
`, version))
	if err := tarWriter.WriteHeader(&tar.Header{Name: "reinstate", Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return asset
}

func makeWindowsArchive(t *testing.T, releaseDir, version string) string {
	t.Helper()
	asset := "reinstate_" + version + "_windows_amd64.zip"
	file, err := os.Create(filepath.Join(releaseDir, asset))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(file)
	entry, err := zipWriter.Create("reinstate.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("synthetic installer fixture")); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return asset
}

func runPOSIXInstaller(t *testing.T, installDir, baseURL, version string, extraEnv []string) ([]byte, error) {
	t.Helper()
	command := exec.Command("sh", filepath.Join(repoRoot(t), "scripts", "install.sh"))
	command.Env = append(cleanEnv(os.Environ(), "REINSTATE_CONFIRM_REPLACE", "REINSTATE_SKIP_VERSION_CHECK"),
		"REINSTATE_VERSION="+version,
		"REINSTATE_RELEASE_BASE_URL="+baseURL,
		"INSTALL_DIR="+installDir,
	)
	command.Env = append(command.Env, extraEnv...)
	return command.CombinedOutput()
}

func cleanEnv(environment []string, keys ...string) []string {
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if !blocked[key] {
			result = append(result, entry)
		}
	}
	return result
}

func assertContainsFile(t *testing.T, path string, expected []byte) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, expected) {
		t.Fatalf("%s does not contain expected marker", path)
	}
}

func writeChecksums(t *testing.T, dir, asset string) {
	t.Helper()
	file, err := os.Open(filepath.Join(dir, asset))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf("%x  %s\n", hash.Sum(nil), asset)
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerScriptsDoNotRequestElevation(t *testing.T) {
	for _, relative := range []string{"scripts/install.sh", "scripts/install.ps1"} {
		body := strings.ToLower(read(t, relative))
		for _, forbidden := range []string{"sudo ", "runas", "-verb runas"} {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s contains elevation path %q", relative, forbidden)
			}
		}
	}
}
