package doctest

import (
	"crypto/sha256"
	"encoding/json"
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

const (
	publicBootstrapVersion       = "v0.5.0-rc.4"
	publicPOSIXInstallerSHA256   = "7776adb4ace8aa333745cd3f3e42b3a10d1400b9394c612d065c20a739db2e66"
	publicWindowsInstallerSHA256 = "02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe"
)

func TestWebsiteCIInstallerPinMatchesPublicBootstrapVersion(t *testing.T) {
	workflow := read(t, ".github/workflows/ci.yml")
	want := "grep -F '" + publicBootstrapVersion + "'"
	if got := strings.Count(workflow, want); got != 2 {
		t.Fatalf("website CI must verify both public installers against %s; found %d matching assertions", publicBootstrapVersion, got)
	}
}

func TestPublicBootstrapVercelHeaders(t *testing.T) {
	var config struct {
		Git struct {
			DeploymentEnabled *bool `json:"deploymentEnabled"`
		} `json:"git"`
		IgnoreCommand string `json:"ignoreCommand"`
		Headers       []struct {
			Source  string `json:"source"`
			Headers []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"headers"`
	}
	if err := json.Unmarshal([]byte(read(t, "website/vercel.json")), &config); err != nil {
		t.Fatal(err)
	}
	if config.Git.DeploymentEnabled == nil || *config.Git.DeploymentEnabled {
		t.Error("automatic Vercel Git deployments must remain disabled")
	}
	if config.IgnoreCommand != "node scripts/vercel-ignore-production-branch.mjs" {
		t.Errorf("unexpected Vercel deployment gate: %q", config.IgnoreCommand)
	}

	for _, route := range []string{"/install.sh", "/install.ps1"} {
		found := false
		for _, definition := range config.Headers {
			if definition.Source != route {
				continue
			}
			found = true
			headers := map[string]string{}
			for _, header := range definition.Headers {
				headers[strings.ToLower(header.Key)] = strings.ToLower(header.Value)
			}
			if headers["content-type"] != "text/plain; charset=utf-8" {
				t.Errorf("%s Content-Type = %q", route, headers["content-type"])
			}
			if headers["x-content-type-options"] != "nosniff" {
				t.Errorf("%s X-Content-Type-Options = %q", route, headers["x-content-type-options"])
			}
			if !strings.Contains(headers["cache-control"], "must-revalidate") {
				t.Errorf("%s Cache-Control = %q", route, headers["cache-control"])
			}
		}
		if !found {
			t.Errorf("missing Vercel headers for %s", route)
		}
	}
}

func TestProductionDeploymentVerifiesBeforePromotion(t *testing.T) {
	body := read(t, "scripts/deploy-website-production.sh")
	for _, required := range []string{
		`^website-v[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[1-9][0-9]*$`,
		`git branch --show-current`,
		`git status --porcelain`,
		`verify-tag "$deployment_tag"`,
		`verify-tag "$cli_version"`,
		`git merge-base --is-ancestor "$cli_version^{}" "$deployment_commit"`,
		`git diff --exit-code "$cli_version^{}"`,
		`check-cli-release.mjs --tag "$cli_version"`,
		`check-vercel-project-link.mjs`,
		`npm exec --yes --package=vercel@57.0.0 -- vercel`,
		`vercel_cli deploy`,
		`--prod --skip-domain`,
		`parse-vercel-deployment-url.mjs`,
		`verify-live-installers.sh" "$cli_version"`,
		`npm run check:freshness`,
		`npm run check:lighthouse`,
		`--allow-missing-previous`,
		`artifacts/indexnow/$deployment_tag-plan.json`,
		`npm run check:production-discovery`,
		`--allow-vercel-preview-noindex`,
		`vercel_cli promote`,
		`https://reinstate.dev`,
	} {
		if !strings.Contains(body, required) {
			t.Errorf("production deployment script is missing %q", required)
		}
	}
	if strings.Contains(body, `npx --yes vercel`) {
		t.Error("production deployment must use the lockfile-pinned local Vercel CLI")
	}
	verifyIndex := strings.Index(body, `"$repo_directory/scripts/verify-live-installers.sh"`)
	promoteIndex := strings.Index(body, `vercel_cli promote`)
	if verifyIndex == -1 || promoteIndex == -1 || verifyIndex > promoteIndex {
		t.Error("immutable installer verification must run before production promotion")
	}
	smokeIndex := strings.Index(body, `--base-url "$deployment_url"`)
	if smokeIndex == -1 || smokeIndex > promoteIndex {
		t.Error("immutable production-discovery smoke must pass before production promotion")
	}
	productionSmokeIndex := strings.LastIndex(body, `npm run check:production-discovery`)
	if productionSmokeIndex == -1 || productionSmokeIndex < promoteIndex {
		t.Error("promoted production origin must receive a discovery smoke test")
	}
}

func TestProductionDeploymentRejectsInvalidWebsiteTagDate(t *testing.T) {
	// The deployment script is POSIX shell. A host without one cannot exercise
	// the contract at all, and running it anyway reports "executable not found"
	// as though the script had accepted the invalid date.
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no POSIX shell on this host; deployment script contract not exercisable")
	}
	command := exec.Command(
		"sh",
		filepath.Join(repoRoot(t), "scripts", "deploy-website-production.sh"),
		"website-v2026.02.30.1",
	)
	command.Dir = repoRoot(t)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("production deployment accepted an invalid calendar date")
	}
	if !strings.Contains(string(output), "invalid website deployment date") {
		t.Fatalf("unexpected invalid-date failure:\n%s", output)
	}
}

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
				"Security.Cryptography.SHA256",
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
				"Get-FileHash",
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
if [ "${REINSTATE_VERSION:-}" != "v0.5.0-rc.4" ]; then
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
if ($env:REINSTATE_VERSION -ne "v0.5.0-rc.4") {
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
			pinnedHash = publicPOSIXInstallerSHA256
		case ".ps1":
			pinnedHash = publicWindowsInstallerSHA256
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
