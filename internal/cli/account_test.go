package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

var recoveryCodePattern = regexp.MustCompile(`\b(?:[0-9A-Z]{4}-){7}[0-9A-Z]{4}\b`)

// accountJourney drives the real CLI entrypoint for one "device": its own
// Reinstate home, sharing one disk-backed memory locker and one in-process
// secret store with the other devices in the test.
type accountJourney struct {
	t       *testing.T
	home    string
	secrets credentials.SecretStore
	// prompt answers the hidden recovery-code prompt; nil leaves the
	// production path (REINSTATE_RECOVERY_CODE_FD or a terminal).
	prompt func(prompt string, stderrSoFar func() string) ([]byte, error)
}

func (j *accountJourney) run(args ...string) (stdout, stderr string, code int) {
	j.t.Helper()
	j.t.Setenv("REINSTATE_HOME", j.home)
	var out, errb bytes.Buffer
	opts := Options{
		Name: "reinstate", Stdout: &out, Stderr: &errb, Args: args,
		AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
		DeviceSecrets:       j.secrets,
	}
	if j.prompt != nil {
		opts.RecoveryCodePrompt = func(prompt string) ([]byte, error) {
			return j.prompt(prompt, errb.String)
		}
	}
	code = Execute(opts)
	return out.String(), errb.String(), code
}

// TestAccountJourneyInitPushRecoverPull is the primary-seam journey for the
// hosted key model: init on device A, push a session, recover on a fresh
// device B from the recovery code alone, pull, and read the content.
func TestAccountJourneyInitPushRecoverPull(t *testing.T) {
	locker := t.TempDir()
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("REINSTATE_PASSPHRASE_FD", "")
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", "")

	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)
	sourceProject := filepath.Join(userHome, "Projects", "hop-source")
	targetProject := filepath.Join(userHome, "Projects", "hop-target")
	for _, dir := range []string{sourceProject, targetProject} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	claudeProjectsRoot := filepath.Join(userHome, ".claude", "projects")
	sourceClaudeRoot := filepath.Join(claudeProjectsRoot, claudeProjectDirectoryForTest(sourceProject))
	if err := os.MkdirAll(sourceClaudeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".claude", "version"), []byte("2.1.219\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": sourceProject})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"synthetic hop journey"}}`+"\n")...)
	sessionPath := filepath.Join(sourceClaudeRoot, "session-hop.jsonl")
	if err := os.WriteFile(sessionPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	secrets := credentials.NewMemorySecrets()
	var shownCode string
	deviceA := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	deviceA.prompt = func(prompt string, stderrSoFar func() string) ([]byte, error) {
		if !strings.Contains(prompt, "Re-enter") {
			return nil, errors.New("unexpected prompt " + prompt)
		}
		shownCode = recoveryCodePattern.FindString(stderrSoFar())
		if shownCode == "" {
			return nil, errors.New("recovery code was not shown before the confirmation prompt")
		}
		// Typed the way a person would: lower case, spaces instead of dashes.
		return []byte(strings.ToLower(strings.ReplaceAll(shownCode, "-", " "))), nil
	}

	endpoint := "https://example.r2.cloudflarestorage.com"
	out, errb, code := deviceA.run("init", "--endpoint", endpoint, "--bucket", "hop-test",
		"--project", "local/hop="+sourceProject, "--yes")
	if code != ExitOK {
		t.Fatalf("device A init exit=%d out=%q err=%q", code, out, errb)
	}
	cfgA, err := config.LoadConfig(deviceA.home)
	if err != nil {
		t.Fatal(err)
	}
	profileID := cfgA.ProfileID

	// Before enrolment a root-key push cannot happen and the BYO path still
	// asks for a passphrase, which this non-terminal run cannot supply.
	out, errb, code = deviceA.run("account", "status")
	if code != ExitOK || !strings.Contains(out, "not enrolled") || !strings.Contains(out, "keyring: not found") {
		t.Fatalf("status before init: exit=%d out=%q err=%q", code, out, errb)
	}

	out, errb, code = deviceA.run("account", "init")
	if code != ExitOK {
		t.Fatalf("account init exit=%d out=%q err=%q", code, out, errb)
	}
	if shownCode == "" || strings.Count(errb, shownCode) != 1 {
		t.Fatalf("recovery code must be shown exactly once on the prompt stream: %q", errb)
	}
	if strings.Contains(out, shownCode) {
		t.Fatalf("recovery code leaked to stdout: %q", out)
	}
	for _, phrase := range []string{"nobody can", "recover the locker", "not the operator", "Local session copies", "unaffected"} {
		if !strings.Contains(errb, phrase) {
			t.Fatalf("init did not state the recovery policy (%q missing): %q", phrase, errb)
		}
	}
	assertRecoveryCodeNotOnDisk(t, shownCode, deviceA.home, locker)
	cfgA, _ = config.LoadConfig(deviceA.home)
	if cfgA.Encryption.Type != schema.EncryptionRootKey {
		t.Fatalf("encryption.type = %q after account init", cfgA.Encryption.Type)
	}
	accountA, err := config.LoadAccount(deviceA.home)
	if err != nil || !accountA.RecoveryCodeConfirmed || accountA.EnrolledVia != "init" || accountA.KeyGeneration != 1 {
		t.Fatalf("account record: %+v %v", accountA, err)
	}
	out, errb, code = deviceA.run("account", "init")
	if code != ExitSafety {
		t.Fatalf("second account init should refuse: exit=%d out=%q err=%q", code, out, errb)
	}

	// Push without any passphrase: the root key seals the envelope.
	out, errb, code = deviceA.run("push", "--agent", "claude", "--session", "session-hop", "--json")
	if code != ExitOK {
		t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
	}

	// Device B: fresh home, joins the profile, enrols from the recovery code
	// delivered through the automation descriptor.
	deviceB := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	out, errb, code = deviceB.run("init", "--endpoint", endpoint, "--bucket", "hop-test",
		"--profile-id", profileID, "--project", "local/hop="+targetProject, "--yes")
	if code != ExitOK {
		t.Fatalf("device B init exit=%d out=%q err=%q", code, out, errb)
	}
	out, errb, code = deviceB.run("pull", "--agent", "claude", "--session", "session-hop", "--json")
	if code == ExitOK || !strings.Contains(errb, "Encryption passphrase") && !strings.Contains(errb, "interactive terminal") {
		// Not yet enrolled and still on the BYO model: the command must not
		// reach storage with the wrong key model.
		t.Fatalf("pull before recover: exit=%d out=%q err=%q", code, out, errb)
	}

	withRecoveryCodeFD(t, shownCode)
	out, errb, code = deviceB.run("account", "recover")
	if code != ExitOK {
		t.Fatalf("account recover exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(out, "devices=2") || !strings.Contains(errb, "nobody can") {
		t.Fatalf("recover output: out=%q err=%q", out, errb)
	}
	assertRecoveryCodeNotOnDisk(t, shownCode, deviceB.home, locker)
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", "")

	out, errb, code = deviceB.run("account", "status", "--json")
	if code != ExitOK {
		t.Fatalf("status exit=%d out=%q err=%q", code, out, errb)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("status json: %v %q", err, out)
	}
	if status["key_generation"] != float64(1) || status["enrolled_devices"] != float64(2) ||
		status["recovery_code_confirmed"] != true || status["device_in_keyring"] != true ||
		status["enrolled_via"] != "recover" || status["encryption_type"] != schema.EncryptionRootKey {
		t.Fatalf("status = %v", status)
	}
	if !strings.Contains(status["account_path"].(string), "/") || strings.Contains(status["account_path"].(string), `\`) {
		t.Fatalf("account_path should be slash-normalized on every host: %v", status["account_path"])
	}

	// Device B pulls and reads what device A wrote.
	targetSessionPath := filepath.Join(claudeProjectsRoot, claudeProjectDirectoryForTest(targetProject), "session-hop.jsonl")
	out, errb, code = deviceB.run("pull", "--agent", "claude", "--session", "session-hop", "--json")
	if code != ExitOK {
		t.Fatalf("pull exit=%d out=%q err=%q", code, out, errb)
	}
	restored, err := os.ReadFile(targetSessionPath)
	if err != nil {
		t.Fatalf("session not restored on device B: %v", err)
	}
	if !bytes.Contains(restored, []byte("synthetic hop journey")) {
		t.Fatalf("restored content: %q", restored)
	}
	// Storage holds only ciphertext plus the keyring's wrapped blobs.
	assertRecoveryCodeNotOnDisk(t, "synthetic hop journey", locker)

	// Device C: a wrong code with a valid checksum fails closed and writes
	// nothing; a code with a typo is rejected by the checksum first.
	deviceC := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	out, errb, code = deviceC.run("init", "--endpoint", endpoint, "--bucket", "hop-test",
		"--profile-id", profileID, "--project", "local/hop="+targetProject, "--yes")
	if code != ExitOK {
		t.Fatalf("device C init exit=%d out=%q err=%q", code, out, errb)
	}
	wrongCode, err := keyring.GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	withRecoveryCodeFD(t, wrongCode)
	out, errb, code = deviceC.run("account", "recover")
	if code != ExitAuthStorage || !strings.Contains(errb, "does not match") {
		t.Fatalf("wrong code: exit=%d out=%q err=%q", code, out, errb)
	}
	typo := shownCode[:len(shownCode)-1] + map[bool]string{true: "2", false: "3"}[strings.HasSuffix(shownCode, "3")]
	withRecoveryCodeFD(t, typo)
	out, errb, code = deviceC.run("account", "recover")
	if code != ExitUsage || !strings.Contains(errb, "checksum") {
		t.Fatalf("typo: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := config.LoadAccount(deviceC.home); !os.IsNotExist(err) {
		t.Fatalf("failed recover wrote an account record: %v", err)
	}
	cfgC, _ := config.LoadConfig(deviceC.home)
	if cfgC.Encryption.Type != schema.EncryptionPassphrase {
		t.Fatalf("failed recover switched the key model: %q", cfgC.Encryption.Type)
	}
	out, errb, code = deviceC.run("account", "status", "--json")
	if code != ExitOK {
		t.Fatalf("status exit=%d out=%q err=%q", code, out, errb)
	}
	_ = json.Unmarshal([]byte(out), &status)
	if status["enrolled_devices"] != float64(2) || status["device_in_keyring"] != false || status["device_key_present"] != false {
		t.Fatalf("failed recover changed the keyring or left a device key: %v", status)
	}
}

// TestAccountInitMismatchWritesNothing covers the forced re-entry: a
// confirmation that does not match the shown code aborts before the keyring,
// device key, config, or account record exist.
func TestAccountInitMismatchWritesNothing(t *testing.T) {
	locker := t.TempDir()
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", "")
	secrets := credentials.NewMemorySecrets()
	device := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	out, errb, code := device.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "hop-test", "--yes")
	if code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	cases := map[string]func(shown string) string{
		"different valid code": func(string) string {
			other, err := keyring.GenerateRecoveryCode()
			if err != nil {
				t.Fatal(err)
			}
			return other
		},
		"typo":  func(shown string) string { return "0000" + shown[4:] },
		"empty": func(string) string { return "" },
	}
	for name, answer := range cases {
		t.Run(name, func(t *testing.T) {
			device.prompt = func(_ string, stderrSoFar func() string) ([]byte, error) {
				shown := recoveryCodePattern.FindString(stderrSoFar())
				if shown == "" {
					t.Fatal("code not shown before confirmation")
				}
				return []byte(answer(shown)), nil
			}
			out, errb, code := device.run("account", "init")
			if code != ExitSafety || !strings.Contains(errb, "nothing was written") {
				t.Fatalf("exit=%d out=%q err=%q", code, out, errb)
			}
			shown := recoveryCodePattern.FindString(errb)
			if _, err := config.LoadAccount(device.home); !os.IsNotExist(err) {
				t.Fatalf("account record written: %v", err)
			}
			cfg, _ := config.LoadConfig(device.home)
			if cfg.Encryption.Type != schema.EncryptionPassphrase {
				t.Fatalf("key model switched: %q", cfg.Encryption.Type)
			}
			if _, err := secrets.GetSecret(deviceSecretRef(cfg.ProfileID, cfg.DeviceID)); !errors.Is(err, credentials.ErrSecretNotFound) {
				t.Fatalf("device key stored: %v", err)
			}
			out, errb, code = device.run("account", "status")
			if code != ExitOK || !strings.Contains(out, "keyring: not found") {
				t.Fatalf("keyring written: exit=%d out=%q err=%q", code, out, errb)
			}
			assertRecoveryCodeNotOnDisk(t, shown, device.home, locker)
		})
	}
}

// TestRootKeyPushRefusesUnenrolledDevice: a home configured for the root-key
// model without a device key or account record must explain how to enrol
// instead of prompting for a passphrase or touching storage.
func TestRootKeyPushRefusesUnenrolledDevice(t *testing.T) {
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", t.TempDir())
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	device := &accountJourney{t: t, home: t.TempDir(), secrets: credentials.NewMemorySecrets()}
	if _, errb, code := device.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "hop-test", "--yes"); code != ExitOK {
		t.Fatalf("init exit=%d err=%q", code, errb)
	}
	cfg, _ := config.LoadConfig(device.home)
	cfg.Encryption.Type = schema.EncryptionRootKey
	if err := config.SaveConfig(device.home, cfg); err != nil {
		t.Fatal(err)
	}
	out, errb, code := device.run("status")
	if code != ExitConfig || !strings.Contains(errb, "rein account recover") || strings.Contains(errb, "passphrase") {
		t.Fatalf("exit=%d out=%q err=%q", code, out, errb)
	}
}

func withRecoveryCodeFD(t *testing.T, code string) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "recovery-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.WriteString(code + "\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", strconv.FormatUint(uint64(file.Fd()), 10))
}

// assertRecoveryCodeNotOnDisk walks every file under roots and fails when
// needle appears in any of them.
func assertRecoveryCodeNotOnDisk(t *testing.T, needle string, roots ...string) {
	t.Helper()
	if needle == "" {
		t.Fatal("empty needle")
	}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if bytes.Contains(raw, []byte(needle)) || bytes.Contains(raw, []byte(strings.ReplaceAll(needle, "-", ""))) {
				t.Fatalf("%q found on disk at %s", needle, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestAccountRecoverOnListedDevice: a device whose keyring wrap already
// exists must never have its device key overwritten or deleted. Losing the
// local record re-attaches; a missing or foreign key refuses without writes.
func TestAccountRecoverOnListedDevice(t *testing.T) {
	locker := t.TempDir()
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", locker)
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	t.Setenv("REINSTATE_RECOVERY_CODE_FD", "")
	secrets := credentials.NewMemorySecrets()
	device := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	if _, errb, code := device.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "hop-test", "--yes"); code != ExitOK {
		t.Fatalf("init exit=%d err=%q", code, errb)
	}
	var shownCode string
	device.prompt = func(_ string, stderrSoFar func() string) ([]byte, error) {
		shownCode = recoveryCodePattern.FindString(stderrSoFar())
		return []byte(shownCode), nil
	}
	if out, errb, code := device.run("account", "init"); code != ExitOK {
		t.Fatalf("account init exit=%d out=%q err=%q", code, out, errb)
	}
	device.prompt = nil
	cfg, err := config.LoadConfig(device.home)
	if err != nil {
		t.Fatal(err)
	}
	ref := deviceSecretRef(cfg.ProfileID, cfg.DeviceID)
	original, err := secrets.GetSecret(ref)
	if err != nil {
		t.Fatal(err)
	}
	assertKeyUnchanged := func(t *testing.T) {
		t.Helper()
		now, err := secrets.GetSecret(ref)
		if err != nil || !bytes.Equal(now, original) {
			t.Fatalf("device key changed or removed: %v", err)
		}
	}
	assertKeyringUnchanged := func(t *testing.T) {
		t.Helper()
		out, errb, code := device.run("account", "status", "--json")
		if code != ExitOK {
			t.Fatalf("status exit=%d err=%q", code, errb)
		}
		var status map[string]any
		_ = json.Unmarshal([]byte(out), &status)
		if status["enrolled_devices"] != float64(1) || status["key_generation"] != float64(1) {
			t.Fatalf("keyring changed: %v", status)
		}
	}

	// Lost local record, key still present: recover re-attaches.
	if err := os.Remove(config.AccountPath(device.home)); err != nil {
		t.Fatal(err)
	}
	withRecoveryCodeFD(t, shownCode)
	out, errb, code := device.run("account", "recover")
	if code != ExitOK || !strings.Contains(out, "already enrolled") || !strings.Contains(out, "devices=1") {
		t.Fatalf("re-attach: exit=%d out=%q err=%q", code, out, errb)
	}
	assertKeyUnchanged(t)
	assertKeyringUnchanged(t)
	account, err := config.LoadAccount(device.home)
	if err != nil || account.KeyGeneration != 1 || account.EnrolledVia != "recover" || !account.RecoveryCodeConfirmed {
		t.Fatalf("account record after re-attach: %+v %v", account, err)
	}
	if out, errb, code := device.run("status"); code != ExitOK {
		t.Fatalf("status after re-attach: exit=%d out=%q err=%q", code, out, errb)
	}

	// Listed device whose key the OS keyring no longer holds: refuse, and do
	// not touch the keyring.
	if err := os.Remove(config.AccountPath(device.home)); err != nil {
		t.Fatal(err)
	}
	if err := secrets.DeleteSecret(ref); err != nil {
		t.Fatal(err)
	}
	withRecoveryCodeFD(t, shownCode)
	out, errb, code = device.run("account", "recover")
	if code != ExitSafety || !strings.Contains(errb, "holds no key") || !strings.Contains(errb, "nothing was written") {
		t.Fatalf("missing key: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, err := secrets.GetSecret(ref); !errors.Is(err, credentials.ErrSecretNotFound) {
		t.Fatalf("recover wrote a device key: %v", err)
	}
	if _, err := config.LoadAccount(device.home); !os.IsNotExist(err) {
		t.Fatalf("recover wrote an account record: %v", err)
	}
	assertKeyringUnchanged(t)

	// A key the keyring does not list: refuse without overwriting it.
	foreign, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	if err := secrets.SetSecret(ref, []byte(foreign.String())); err != nil {
		t.Fatal(err)
	}
	withRecoveryCodeFD(t, shownCode)
	out, errb, code = device.run("account", "recover")
	if code != ExitSafety || !strings.Contains(errb, "not the key held") {
		t.Fatalf("foreign key: exit=%d out=%q err=%q", code, out, errb)
	}
	if now, err := secrets.GetSecret(ref); err != nil || string(now) != foreign.String() {
		t.Fatalf("foreign key overwritten or deleted: %v", err)
	}
	assertKeyringUnchanged(t)

	// account init never overwrites an existing device key either.
	original = []byte(foreign.String())
	fresh := &accountJourney{t: t, home: t.TempDir(), secrets: secrets}
	t.Setenv("REINSTATE_MEMORY_BACKEND_DIR", t.TempDir())
	if _, errb, code := fresh.run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "hop-test",
		"--yes"); code != ExitOK {
		t.Fatalf("fresh init exit=%d err=%q", code, errb)
	}
	// Same profile and device ids as the enrolled device, pointed at an
	// empty locker, so the only thing standing in the way is the stored key.
	freshCfg, err := config.LoadConfig(fresh.home)
	if err != nil {
		t.Fatal(err)
	}
	freshCfg.ProfileID = cfg.ProfileID
	freshCfg.DeviceID = cfg.DeviceID
	if err := config.SaveConfig(fresh.home, freshCfg); err != nil {
		t.Fatal(err)
	}
	fresh.prompt = func(_ string, stderrSoFar func() string) ([]byte, error) {
		return []byte(recoveryCodePattern.FindString(stderrSoFar())), nil
	}
	out, errb, code = fresh.run("account", "init")
	if code != ExitSafety || !strings.Contains(errb, "already holds a device key") {
		t.Fatalf("init over existing key: exit=%d out=%q err=%q", code, out, errb)
	}
	assertKeyUnchanged(t)
}
