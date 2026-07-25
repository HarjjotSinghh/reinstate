package doctest

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

const publicBootstrapVersion = "v0.1.0-rc.2"

func TestPublicBootstrapStaticContract(t *testing.T) {
	tests := []struct {
		path      string
		extension string
		tagPath   string
		required  []string
		forbidden []string
	}{
		{
			path:      "website/public/install.sh",
			extension: "install.sh",
			tagPath:   `/${VERSION}/scripts/install.sh`,
			required: []string{
				publicBootstrapVersion,
				"https://raw.githubusercontent.com/HarjjotSinghh/reinstate",
				"PINNED_INSTALLER_SHA256",
				"REINSTATE_VERSION=\"$VERSION\"",
				"REINSTATE_SKIP_PATH_UPDATE",
				"Next:",
				"rein init",
			},
			forbidden: []string{
				"releases/latest",
				"api.github.com/repos",
				"\nrein init",
				"REINSTATE_BOOTSTRAP_ORIGIN",
				"REINSTATE_BOOTSTRAP_INSTALLER_SHA256",
			},
		},
		{
			path:      "website/public/install.ps1",
			extension: "install.ps1",
			tagPath:   `/${Version}/scripts/install.ps1`,
			required: []string{
				publicBootstrapVersion,
				"https://raw.githubusercontent.com/HarjjotSinghh/reinstate",
				"PinnedInstallerSha256",
				"$env:REINSTATE_VERSION = $Version",
				"REINSTATE_SKIP_PATH_UPDATE",
				"OrdinalIgnoreCase",
				"Next: rein init",
			},
			forbidden: []string{
				"releases/latest",
				"api.github.com/repos",
				"\nrein init",
				"& rein init",
				"REINSTATE_BOOTSTRAP_ORIGIN",
				"REINSTATE_BOOTSTRAP_INSTALLER_SHA256",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.extension, func(t *testing.T) {
			body := read(t, test.path)
			for _, value := range test.required {
				if !strings.Contains(body, value) {
					t.Errorf("%s is missing contract value %q", test.path, value)
				}
			}
			for _, value := range test.forbidden {
				if strings.Contains(body, value) {
					t.Errorf("%s contains forbidden value %q", test.path, value)
				}
			}
			if !strings.Contains(body, test.tagPath) {
				t.Errorf("%s must construct exact tag path %q", test.path, test.tagPath)
			}
		})
	}
}

func TestPOSIXPublicBootstrapContract(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX bootstrap supports macOS/Linux")
	}

	canonical := []byte(`#!/bin/sh
set -eu
if [ "${REINSTATE_VERSION:-}" != "v0.1.0-rc.2" ]; then
  echo "wrong bootstrap version: ${REINSTATE_VERSION:-missing}" >&2
  exit 91
fi
mkdir -p "$INSTALL_DIR"
cat >"$INSTALL_DIR/reinstate" <<'BINARY'
#!/bin/sh
if [ "${1:-}" = "init" ] && [ -n "${REINSTATE_INIT_MARKER:-}" ]; then
  : >"$REINSTATE_INIT_MARKER"
fi
BINARY
chmod 755 "$INSTALL_DIR/reinstate"
cp "$INSTALL_DIR/reinstate" "$INSTALL_DIR/rein"
`)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/"+publicBootstrapVersion+"/scripts/install.sh" {
			http.NotFound(writer, request)
			return
		}
		requests.Add(1)
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write(canonical)
	}))
	defer server.Close()

	bootstrapPath := materializeBootstrapForTest(
		t, "website/public/install.sh", server.URL, sha256Hex(canonical),
	)
	home := t.TempDir()
	installDir := filepath.Join(t.TempDir(), "reinstate bin")
	initMarker := filepath.Join(t.TempDir(), "init-ran")
	environment := publicBootstrapEnv(
		"HOME="+home,
		"INSTALL_DIR="+installDir,
		"SHELL=/bin/zsh",
		"REINSTATE_INIT_MARKER="+initMarker,
	)

	output, err := runPOSIXPublicBootstrap(bootstrapPath, environment)
	if err != nil {
		t.Fatalf("public bootstrap failed: %v\n%s", err, output)
	}
	for _, name := range []string{"reinstate", "rein"} {
		if _, err := os.Stat(filepath.Join(installDir, name)); err != nil {
			t.Fatalf("missing installed command %s: %v", name, err)
		}
	}
	if _, err := os.Stat(initMarker); !os.IsNotExist(err) {
		t.Fatalf("bootstrap launched interactive init; marker err=%v", err)
	}
	profile := filepath.Join(home, ".zshrc")
	profileBody, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(profileBody), installDir); count != 1 {
		t.Fatalf("PATH entry count = %d, want 1\n%s", count, profileBody)
	}
	expectedNext := "Next: '" + filepath.Join(installDir, "rein") + "' init"
	if !strings.Contains(string(output), expectedNext) {
		t.Fatalf("bootstrap omitted immediate next command %q:\n%s", expectedNext, output)
	}

	output, err = runPOSIXPublicBootstrap(bootstrapPath, environment)
	if err != nil {
		t.Fatalf("second public bootstrap failed: %v\n%s", err, output)
	}
	profileBody, err = os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(profileBody), installDir); count != 1 {
		t.Fatalf("second run duplicated PATH entry: count=%d\n%s", count, profileBody)
	}
	if requests.Load() != 2 {
		t.Fatalf("canonical installer request count = %d, want 2", requests.Load())
	}

	skipHome := t.TempDir()
	skipInstallDir := filepath.Join(t.TempDir(), "bin")
	skipEnvironment := publicBootstrapEnv(
		"HOME="+skipHome,
		"INSTALL_DIR="+skipInstallDir,
		"SHELL=/bin/zsh",
		"REINSTATE_SKIP_PATH_UPDATE=1",
	)
	if output, err = runPOSIXPublicBootstrap(bootstrapPath, skipEnvironment); err != nil {
		t.Fatalf("opt-out bootstrap failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(skipHome, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("PATH opt-out modified shell profile; err=%v", err)
	}
}

func TestPOSIXPublicBootstrapRejectsInstallerHashMismatch(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX bootstrap supports macOS/Linux")
	}

	canonical := []byte("#!/bin/sh\nexit 0\n")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(canonical)
	}))
	defer server.Close()

	bootstrapPath := materializeBootstrapForTest(
		t, "website/public/install.sh", server.URL, "",
	)
	home := t.TempDir()
	installDir := filepath.Join(t.TempDir(), "bin")
	environment := publicBootstrapEnv(
		"HOME="+home,
		"INSTALL_DIR="+installDir,
		"SHELL=/bin/zsh",
	)
	output, err := runPOSIXPublicBootstrap(bootstrapPath, environment)
	if err == nil || !strings.Contains(strings.ToLower(string(output)), "checksum mismatch") {
		t.Fatalf("bootstrap accepted installer hash mismatch: err=%v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(installDir, "rein")); !os.IsNotExist(err) {
		t.Fatalf("hash mismatch installed a binary; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("hash mismatch modified shell profile; err=%v", err)
	}
}

func TestWindowsPublicBootstrapContract(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows contract")
	}
	powerShell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("Windows PowerShell unavailable")
	}

	canonical := []byte(`
if ($env:REINSTATE_VERSION -ne "v0.1.0-rc.2") {
    throw "wrong bootstrap version: $env:REINSTATE_VERSION"
}
New-Item -ItemType Directory -Force -Path $env:INSTALL_DIR | Out-Null
Set-Content -Path (Join-Path $env:INSTALL_DIR "reinstate.exe") -Value "fixture"
Set-Content -Path (Join-Path $env:INSTALL_DIR "rein.exe") -Value "fixture"
`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/"+publicBootstrapVersion+"/scripts/install.ps1" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write(canonical)
	}))
	defer server.Close()

	installDir := filepath.Join(t.TempDir(), "Reinstate Bin")
	bootstrapPath := materializeBootstrapForTest(
		t, "website/public/install.ps1", server.URL, sha256Hex(canonical),
	)
	command := exec.Command(powerShell, "-NoProfile", "-Command",
		"& $env:REINSTATE_BOOTSTRAP_SCRIPT; & $env:REINSTATE_BOOTSTRAP_SCRIPT")
	command.Env = publicBootstrapEnv(
		"INSTALL_DIR="+installDir,
		"REINSTATE_BOOTSTRAP_SCRIPT="+bootstrapPath,
		"REINSTATE_BOOTSTRAP_PATH_SCOPE=Process",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Windows public bootstrap failed: %v\n%s", err, output)
	}
	for _, name := range []string{"reinstate.exe", "rein.exe"} {
		if _, err := os.Stat(filepath.Join(installDir, name)); err != nil {
			t.Fatalf("missing Windows command %s: %v", name, err)
		}
	}
	if count := strings.Count(string(output), "Added "+installDir+" to current process PATH."); count != 1 {
		t.Fatalf("Windows PATH update count = %d, want 1\n%s", count, output)
	}
	if count := strings.Count(string(output), "Next: rein init"); count != 2 {
		t.Fatalf("Windows next-step count = %d, want 2\n%s", count, output)
	}

	upperPath := strings.ToUpper(installDir) + string(os.PathSeparator)
	command = exec.Command(powerShell, "-NoProfile", "-Command", "& $env:REINSTATE_BOOTSTRAP_SCRIPT")
	command.Env = publicBootstrapEnv(
		"PATH="+upperPath,
		"INSTALL_DIR="+installDir,
		"REINSTATE_BOOTSTRAP_SCRIPT="+bootstrapPath,
		"REINSTATE_BOOTSTRAP_PATH_SCOPE=Process",
	)
	output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("case-insensitive PATH bootstrap failed: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "Added "+installDir) {
		t.Fatalf("case-insensitive PATH match added a duplicate:\n%s", output)
	}
}

func TestWindowsPublicBootstrapRejectsInstallerHashMismatch(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows contract")
	}
	powerShell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("Windows PowerShell unavailable")
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`throw "must not execute"`))
	}))
	defer server.Close()

	installDir := filepath.Join(t.TempDir(), "bin")
	bootstrapPath := materializeBootstrapForTest(
		t, "website/public/install.ps1", server.URL, "",
	)
	command := exec.Command(powerShell, "-NoProfile", "-File", bootstrapPath)
	command.Env = publicBootstrapEnv(
		"INSTALL_DIR="+installDir,
		"REINSTATE_BOOTSTRAP_PATH_SCOPE=Process",
	)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(strings.ToLower(string(output)), "checksum mismatch") {
		t.Fatalf("Windows bootstrap accepted installer hash mismatch: err=%v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(installDir, "rein.exe")); !os.IsNotExist(err) {
		t.Fatalf("hash mismatch installed a Windows binary; err=%v", err)
	}
}

func runPOSIXPublicBootstrap(scriptPath string, environment []string) ([]byte, error) {
	command := exec.Command("sh", scriptPath)
	command.Env = environment
	return command.CombinedOutput()
}

func materializeBootstrapForTest(t *testing.T, relativePath, origin, expectedHash string) string {
	t.Helper()
	body := read(t, relativePath)
	const officialOrigin = "https://raw.githubusercontent.com/HarjjotSinghh/reinstate"
	if !strings.Contains(body, officialOrigin) {
		t.Fatalf("%s is missing official origin", relativePath)
	}
	body = strings.ReplaceAll(body, officialOrigin, origin)
	if expectedHash != "" {
		var pinnedHash string
		switch filepath.Ext(relativePath) {
		case ".sh":
			pinnedHash = "8f68b0ad0707e5e710cb365849cf833f16eaea1ac76407905763747dae986c25"
		case ".ps1":
			pinnedHash = "4d6e422f36ef20f4378786b34a75c042223ebff3db13b3a05f7a97e1126d6781"
		default:
			t.Fatalf("unsupported bootstrap extension: %s", relativePath)
		}
		if !strings.Contains(body, pinnedHash) {
			t.Fatalf("%s is missing pinned installer hash", relativePath)
		}
		body = strings.ReplaceAll(body, pinnedHash, expectedHash)
	}
	path := filepath.Join(t.TempDir(), filepath.Base(relativePath))
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func publicBootstrapEnv(extra ...string) []string {
	blocked := []string{
		"HOME",
		"INSTALL_DIR",
		"REINSTATE_BOOTSTRAP_PATH_SCOPE",
		"REINSTATE_BOOTSTRAP_SCRIPT",
		"REINSTATE_INIT_MARKER",
		"REINSTATE_SKIP_PATH_UPDATE",
		"REINSTATE_VERSION",
		"SHELL",
	}
	environment := cleanEnv(os.Environ(), blocked...)
	return append(environment, extra...)
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%x", sum)
}
