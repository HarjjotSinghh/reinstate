package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	_ "modernc.org/sqlite"
)

// DeviceHome is one synthetic, isolated device identity for the Hop lab:
// its own Reinstate home (config/state/device id -- REINSTATE_HOME is a
// first-class override the product itself honours, internal/config.Home),
// its own agent config roots, and a session set that points at a project
// path no other device's session set uses, so pairing, revocation, the
// lagging device, and path-remap scenarios can tell device A and device B
// apart on one host.
//
// What this does NOT isolate: the OS keyring device token. Two real,
// simultaneously signed-in `rein` processes on one Windows account collide
// there (credentials.KeyringStore uses one fixed service name and one fixed
// entry name regardless of REINSTATE_HOME -- see internal/credentials/
// devicetoken.go and keyring.go, neither of which W4 owns). Sequential use
// of the real binary works today with `hoplab keyring save/load` (this
// package, keyring.go) swapping which device's token is the active one;
// truly simultaneous real-binary devices need the in-process test harness
// pattern instead (internal/cli's hopDevice: a MemoryDeviceTokenStore per
// device, REINSTATE_HOME switched per call -- see hop_first_push_test.go
// and keygeneration_crossplane_test.go, which already run two and three
// devices this way against a real hopd).
type DeviceHome struct {
	Name            string // "device-a", "device-b", ...
	Root            string
	ReinstateHome   string
	ClaudeConfigDir string
	CodexHome       string
	XDGDataHome     string
	Project         string // this device's distinct fake project path (Windows-shaped)
}

// BuildDeviceHome computes a DeviceHome's paths under labRoot without
// touching the filesystem.
func BuildDeviceHome(labRoot, name string) DeviceHome {
	root := filepath.Join(labRoot, name)
	home := filepath.Join(root, "home")
	return DeviceHome{
		Name:            name,
		Root:            root,
		ReinstateHome:   filepath.Join(root, "reinstate"),
		ClaudeConfigDir: filepath.Join(home, ".claude"),
		CodexHome:       filepath.Join(home, ".codex"),
		XDGDataHome:     filepath.Join(home, "xdgdata"),
		Project:         `C:\Users\fixture-user\code\` + name,
	}
}

// Seed plants one Claude, one Codex, and one OpenCode session under h, each
// pointing at h.Project and each carrying an id suffixed with h.Name so no
// two devices' sessions collide if their lockers are ever compared side by
// side. repoRoot locates testdata/adapters/*/windows, the same fixtures
// hoplab's own module tests hydrate from.
func (h DeviceHome) Seed(repoRoot string) error {
	for _, dir := range []string{h.ReinstateHome, h.ClaudeConfigDir, h.CodexHome, h.XDGDataHome} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	suffix := strings.TrimPrefix(h.Name, "device-")
	fixtures := filepath.Join(repoRoot, "testdata", "adapters")

	if err := h.seedClaude(filepath.Join(fixtures, "claude", "windows"), suffix); err != nil {
		return fmt.Errorf("seed claude: %w", err)
	}
	if err := h.seedCodex(filepath.Join(fixtures, "codex", "windows", "sessions", "rollout-syn-001.jsonl"), suffix); err != nil {
		return fmt.Errorf("seed codex: %w", err)
	}
	if err := h.seedOpenCode(filepath.Join(fixtures, "opencode", "windows", "store.sql"), suffix); err != nil {
		return fmt.Errorf("seed opencode: %w", err)
	}
	return nil
}

func rewriteFixture(content, suffix string) string {
	content = strings.ReplaceAll(content, `C:\\Users\\fixture-user\\code\\demo`, `C:\\Users\\fixture-user\\code\\device-`+suffix)
	content = strings.ReplaceAll(content, `C:\Users\fixture-user\code\demo`, `C:\Users\fixture-user\code\device-`+suffix)
	content = strings.ReplaceAll(content, "session-syn-001", "session-syn-001-"+suffix)
	content = strings.ReplaceAll(content, "rollout-syn-001", "rollout-syn-001-"+suffix)
	content = strings.ReplaceAll(content, "ses_fixture001", "ses_fixture001"+suffix)
	content = strings.ReplaceAll(content, "msg_fixtureasst001", "msg_fixtureasst001"+suffix)
	content = strings.ReplaceAll(content, "msg_fixtureuser001", "msg_fixtureuser001"+suffix)
	content = strings.ReplaceAll(content, "prt_fixtureasst001", "prt_fixtureasst001"+suffix)
	content = strings.ReplaceAll(content, "prt_fixtureuser001", "prt_fixtureuser001"+suffix)
	return content
}

func (h DeviceHome) seedClaude(claudeFixtureDir, suffix string) error {
	raw, err := os.ReadFile(filepath.Join(claudeFixtureDir, "projects", "fixture-project", "session-syn-001.jsonl"))
	if err != nil {
		return err
	}
	content := rewriteFixture(string(raw), suffix)
	dir := filepath.Join(h.ClaudeConfigDir, "projects", claudeProjectSlug(h.Project))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "session-syn-001-"+suffix+".jsonl"), []byte(content), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(h.ClaudeConfigDir, "version"), []byte("2.1.228\n"), 0o600)
}

func (h DeviceHome) seedCodex(fixturePath, suffix string) error {
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}
	content := rewriteFixture(string(raw), suffix)
	dir := filepath.Join(h.CodexHome, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "rollout-syn-001-"+suffix+".jsonl"), []byte(content), 0o600)
}

func (h DeviceHome) seedOpenCode(sqlFixturePath, suffix string) error {
	raw, err := os.ReadFile(sqlFixturePath)
	if err != nil {
		return err
	}
	script := rewriteFixture(string(raw), suffix)
	dir := filepath.Join(h.XDGDataHome, "opencode")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "opencode.db")
	_ = os.Remove(dbPath)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(script); err != nil {
		return fmt.Errorf("hydrate %s: %w", dbPath, err)
	}
	return nil
}

// claudeProjectSlug replicates Claude Code's own project-directory naming
// (internal/cli/e2e_test.go's claudeProjectDirectoryForTest, which mirrors
// the vendor: every UTF-16 code unit that is not a-z/A-Z/0-9 becomes '-'),
// so a seeded fixture lands where the real adapter looks for it.
func claudeProjectSlug(projectPath string) string {
	var b strings.Builder
	for _, unit := range utf16.Encode([]rune(projectPath)) {
		if unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z' || unit >= '0' && unit <= '9' {
			b.WriteByte(byte(unit))
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}
