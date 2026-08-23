package daemon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Spec is everything a service definition needs. Rendering is pure so
// golden tests cover every platform from any host.
type Spec struct {
	// Label identifies the service to the OS (launchd label, systemd unit
	// name, scheduled task name).
	Label string
	// Executable is the absolute path of the rein binary.
	Executable string
	// Args are the arguments after the executable ("daemon", "run", ...).
	Args []string
	// Home is the Reinstate home the daemon serves; exported as
	// REINSTATE_HOME so the service sees the same home as the shell that
	// installed it. Empty keeps the default home.
	Home string
	// LogPath receives stdout and stderr of the process.
	LogPath string
	// Path is the PATH the service inherits; launchd agents start with a
	// minimal one that cannot find vendor CLIs otherwise.
	Path string
	// UserID is the account the Windows task runs as (DOMAIN\user or a
	// SID). Task Scheduler refuses a /XML create whose principal names no
	// user with "Access is denied" under a non-elevated token, so the
	// installer always sets it. Ignored off Windows.
	UserID string
	// Env are extra environment variables (lab fixtures such as
	// REINSTATE_BACKEND, or a non-production control plane). Task
	// Scheduler has no per-task environment, so Windows ignores them.
	Env map[string]string
}

// EnvPairs is the environment block in definition order: the home, PATH,
// then Env sorted by name.
func (s Spec) EnvPairs() [][2]string {
	var pairs [][2]string
	if s.Home != "" {
		pairs = append(pairs, [2]string{"REINSTATE_HOME", s.Home})
	}
	pairs = append(pairs, [2]string{"PATH", s.Path})
	keys := make([]string, 0, len(s.Env))
	for k := range s.Env {
		if k == "REINSTATE_HOME" || k == "PATH" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pairs = append(pairs, [2]string{k, s.Env[k]})
	}
	return pairs
}

// DefaultLabel is the service name for the default home. A non-default
// home (REINSTATE_HOME) gets a suffix derived from its path, so two homes
// on one machine can each run a daemon and tests can register a throwaway
// service beside the real one.
const DefaultLabel = "com.reinstate.daemon"

// LabelFor returns the service label for home. defaultHome is what
// config.Home() yields with no override.
func LabelFor(home, defaultHome string) string {
	clean := filepath.Clean(home)
	if clean == filepath.Clean(defaultHome) || home == "" {
		return DefaultLabel
	}
	sum := sha256.Sum256([]byte(clean))
	return DefaultLabel + "." + hex.EncodeToString(sum[:])[:8]
}

// Runner executes an OS command and returns its combined output. It is
// injectable so unit tests never touch launchctl, systemctl, or schtasks.
type Runner func(ctx context.Context, name string, args ...string) (string, error)

// ExecRunner runs real commands.
func ExecRunner(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return text, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, text)
		}
		return text, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return text, nil
}

// State is what `rein daemon status` reports about the registration.
type State struct {
	// Installed reports that the service definition exists.
	Installed bool
	// Running reports that the OS says the process is running.
	Running bool
	// Definition is the path of the plist, unit, or the task name.
	Definition string
	// Detail is the manager's own words, when any.
	Detail string
}

// Manager registers the daemon with the OS so it starts at login.
type Manager interface {
	// Kind is "launchd", "systemd", or "schtasks".
	Kind() string
	// DefinitionPath is where Install writes the definition (task name on
	// Windows, where the definition lives inside the scheduler).
	DefinitionPath(spec Spec) string
	// Render produces the definition text.
	Render(spec Spec) ([]byte, error)
	Install(ctx context.Context, spec Spec) error
	Uninstall(ctx context.Context, spec Spec) error
	Start(ctx context.Context, spec Spec) error
	Stop(ctx context.Context, spec Spec) error
	Status(ctx context.Context, spec Spec) (State, error)
}

// ErrUnsupportedOS reports a platform without a service manager.
var ErrUnsupportedOS = errors.New("no service manager for this platform; run rein daemon run under your own supervisor")

// NewManager returns the manager for goos ("darwin", "linux", "windows").
// userHome is the account's home directory; run executes commands.
func NewManager(goos, userHome string, run Runner) (Manager, error) {
	if run == nil {
		run = ExecRunner
	}
	switch goos {
	case "darwin":
		return &launchdManager{userHome: userHome, run: run}, nil
	case "linux":
		return &systemdManager{userHome: userHome, run: run}, nil
	case "windows":
		return &schtasksManager{run: run}, nil
	}
	return nil, ErrUnsupportedOS
}

func writeDefinition(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Owner-only: the definition carries the service environment.
	return os.WriteFile(path, content, 0o600)
}

// ---------- launchd (macOS) ----------

type launchdManager struct {
	userHome string
	run      Runner
}

func (m *launchdManager) Kind() string { return "launchd" }

func (m *launchdManager) DefinitionPath(spec Spec) string {
	return filepath.Join(m.userHome, "Library", "LaunchAgents", spec.Label+".plist")
}

var plistTemplate = template.Must(template.New("plist").Funcs(template.FuncMap{"esc": xmlEscape}).Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{esc .Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{esc .Executable}}</string>
{{- range .Args}}
		<string>{{esc .}}</string>
{{- end}}
	</array>
	<key>EnvironmentVariables</key>
	<dict>
{{- range .EnvPairs}}
		<key>{{esc (index . 0)}}</key>
		<string>{{esc (index . 1)}}</string>
{{- end}}
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>ThrottleInterval</key>
	<integer>10</integer>
	<key>ProcessType</key>
	<string>Background</string>
	<key>StandardOutPath</key>
	<string>{{esc .LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{esc .LogPath}}</string>
</dict>
</plist>
`))

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func (m *launchdManager) Render(spec Spec) ([]byte, error) {
	var buf bytes.Buffer
	if err := plistTemplate.Execute(&buf, spec); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *launchdManager) domain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func (m *launchdManager) Install(ctx context.Context, spec Spec) error {
	content, err := m.Render(spec)
	if err != nil {
		return err
	}
	path := m.DefinitionPath(spec)
	// A stale registration would keep the old definition.
	_, _ = m.run(ctx, "launchctl", "bootout", m.domain()+"/"+spec.Label)
	if err := writeDefinition(path, content); err != nil {
		return err
	}
	if _, err := m.run(ctx, "launchctl", "bootstrap", m.domain(), path); err != nil {
		return err
	}
	return nil
}

func (m *launchdManager) Uninstall(ctx context.Context, spec Spec) error {
	path := m.DefinitionPath(spec)
	if _, err := m.run(ctx, "launchctl", "bootout", m.domain()+"/"+spec.Label); err != nil {
		// Not loaded is fine; a missing plist is also fine.
		if _, statErr := os.Stat(path); statErr == nil && !strings.Contains(err.Error(), "No such process") && !strings.Contains(err.Error(), "not find") {
			return err
		}
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *launchdManager) Start(ctx context.Context, spec Spec) error {
	path := m.DefinitionPath(spec)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("the daemon is not installed (%s missing); run rein daemon install", path)
	}
	// A loaded agent is restarted in place; an unloaded one is loaded,
	// which starts it (RunAtLoad) without a second spawn.
	if _, err := m.run(ctx, "launchctl", "print", m.domain()+"/"+spec.Label); err == nil {
		_, err := m.run(ctx, "launchctl", "kickstart", "-k", m.domain()+"/"+spec.Label)
		return err
	}
	_, err := m.run(ctx, "launchctl", "bootstrap", m.domain(), path)
	return err
}

func (m *launchdManager) Stop(ctx context.Context, spec Spec) error {
	// bootout unloads the agent; the plist stays so start can bootstrap it
	// again. launchctl stop alone would let KeepAlive respawn it.
	if _, err := m.run(ctx, "launchctl", "bootout", m.domain()+"/"+spec.Label); err != nil {
		if strings.Contains(err.Error(), "No such process") || strings.Contains(err.Error(), "not find") {
			return nil
		}
		return err
	}
	return nil
}

func (m *launchdManager) Status(ctx context.Context, spec Spec) (State, error) {
	path := m.DefinitionPath(spec)
	state := State{Definition: path}
	if _, err := os.Stat(path); err == nil {
		state.Installed = true
	}
	out, err := m.run(ctx, "launchctl", "print", m.domain()+"/"+spec.Label)
	if err != nil {
		state.Detail = "not loaded"
		return state, nil
	}
	// The first "state =" line is the service's own; nested ones belong
	// to its endpoints.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "state = ") && state.Detail == "" {
			state.Detail = strings.TrimPrefix(line, "state = ")
			switch state.Detail {
			case "running":
				state.Running = true
			case "xpcproxy":
				// launchd has handed the job to xpcproxy and the process is
				// about to exec; for the first second after install the
				// daemon is starting, not stopped.
				state.Running = true
				state.Detail = "starting"
			}
		}
	}
	if state.Detail == "" {
		state.Detail = "loaded"
	}
	return state, nil
}

// ---------- systemd --user (Linux) ----------

type systemdManager struct {
	userHome string
	run      Runner
}

func (m *systemdManager) Kind() string { return "systemd" }

func (m *systemdManager) unit(spec Spec) string { return spec.Label + ".service" }

func (m *systemdManager) DefinitionPath(spec Spec) string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(m.userHome, ".config")
	}
	return filepath.Join(base, "systemd", "user", m.unit(spec))
}

var unitTemplate = template.Must(template.New("unit").Funcs(template.FuncMap{"q": systemdQuote}).Parse(
	`[Unit]
Description=Reinstate daemon (session sync and device approvals)
Documentation=https://reinstate.dev/docs/hop

[Service]
Type=simple
ExecStart={{q .Executable}}{{range .Args}} {{q .}}{{end}}
{{- range .EnvPairs}}
Environment={{index . 0}}={{q (index . 1)}}
{{- end}}
Restart=on-failure
RestartSec=10
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}

[Install]
WantedBy=default.target
`))

// systemdQuote quotes one ExecStart/Environment word when it needs it.
func systemdQuote(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\"'\\$;") {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
	}
	return s
}

func (m *systemdManager) Render(spec Spec) ([]byte, error) {
	var buf bytes.Buffer
	if err := unitTemplate.Execute(&buf, spec); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *systemdManager) Install(ctx context.Context, spec Spec) error {
	content, err := m.Render(spec)
	if err != nil {
		return err
	}
	if err := writeDefinition(m.DefinitionPath(spec), content); err != nil {
		return err
	}
	if _, err := m.run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	_, err = m.run(ctx, "systemctl", "--user", "enable", "--now", m.unit(spec))
	return err
}

func (m *systemdManager) Uninstall(ctx context.Context, spec Spec) error {
	_, _ = m.run(ctx, "systemctl", "--user", "disable", "--now", m.unit(spec))
	if err := os.Remove(m.DefinitionPath(spec)); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, _ = m.run(ctx, "systemctl", "--user", "daemon-reload")
	return nil
}

func (m *systemdManager) Start(ctx context.Context, spec Spec) error {
	_, err := m.run(ctx, "systemctl", "--user", "restart", m.unit(spec))
	return err
}

func (m *systemdManager) Stop(ctx context.Context, spec Spec) error {
	_, err := m.run(ctx, "systemctl", "--user", "stop", m.unit(spec))
	return err
}

func (m *systemdManager) Status(ctx context.Context, spec Spec) (State, error) {
	path := m.DefinitionPath(spec)
	state := State{Definition: path}
	if _, err := os.Stat(path); err == nil {
		state.Installed = true
	}
	out, _ := m.run(ctx, "systemctl", "--user", "is-active", m.unit(spec))
	state.Detail = strings.TrimSpace(out)
	state.Running = state.Detail == "active"
	return state, nil
}

// ---------- Task Scheduler (Windows) ----------

type schtasksManager struct {
	run Runner
}

func (m *schtasksManager) Kind() string { return "schtasks" }

func (m *schtasksManager) DefinitionPath(spec Spec) string { return `\` + spec.Label }

// schtasksTemplate is the Task Scheduler 1.2 XML schtasks /Create /XML
// accepts: a logon trigger for the installing user, no time limit, no
// battery conditions, restart on failure, and a single instance.
var schtasksTemplate = template.Must(template.New("task").Funcs(template.FuncMap{"esc": xmlEscape}).Parse(
	`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Reinstate daemon (session sync and device approvals)</Description>
    <URI>\{{esc .Label}}</URI>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
{{- if .UserID}}
      <UserId>{{esc .UserID}}</UserId>
{{- end}}
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>true</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>10</Count>
    </RestartOnFailure>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>{{esc .Executable}}</Command>
      <Arguments>{{esc .Arguments}}</Arguments>
    </Exec>
  </Actions>
</Task>
`))

type schtasksView struct {
	Spec
	Arguments string
}

// schtasksArguments joins the arguments for the task's Exec action. The
// home travels as an argument (--home) because a scheduled task has no
// per-task environment block.
func schtasksArguments(spec Spec) string {
	args := append([]string{}, spec.Args...)
	if spec.Home != "" {
		args = append(args, "--home", spec.Home)
	}
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			a = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		}
		quoted = append(quoted, a)
	}
	return strings.Join(quoted, " ")
}

func (m *schtasksManager) Render(spec Spec) ([]byte, error) {
	var buf bytes.Buffer
	if err := schtasksTemplate.Execute(&buf, schtasksView{Spec: spec, Arguments: schtasksArguments(spec)}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *schtasksManager) Install(ctx context.Context, spec Spec) error {
	content, err := m.Render(spec)
	if err != nil {
		return err
	}
	// schtasks reads the XML as UTF-16 with a BOM when the declaration says
	// so; write exactly that.
	xmlPath := filepath.Join(os.TempDir(), spec.Label+".xml")
	if err := os.WriteFile(xmlPath, utf16LE(content), 0o600); err != nil {
		return err
	}
	defer func() { _ = os.Remove(xmlPath) }()
	if _, err := m.run(ctx, "schtasks", "/Create", "/TN", spec.Label, "/XML", xmlPath, "/F"); err != nil {
		return err
	}
	_, err = m.run(ctx, "schtasks", "/Run", "/TN", spec.Label)
	return err
}

func (m *schtasksManager) Uninstall(ctx context.Context, spec Spec) error {
	_, _ = m.run(ctx, "schtasks", "/End", "/TN", spec.Label)
	if _, err := m.run(ctx, "schtasks", "/Delete", "/TN", spec.Label, "/F"); err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "cannot find") {
			return nil
		}
		return err
	}
	return nil
}

func (m *schtasksManager) Start(ctx context.Context, spec Spec) error {
	_, err := m.run(ctx, "schtasks", "/Run", "/TN", spec.Label)
	return err
}

func (m *schtasksManager) Stop(ctx context.Context, spec Spec) error {
	_, err := m.run(ctx, "schtasks", "/End", "/TN", spec.Label)
	return err
}

func (m *schtasksManager) Status(ctx context.Context, spec Spec) (State, error) {
	state := State{Definition: m.DefinitionPath(spec)}
	out, err := m.run(ctx, "schtasks", "/Query", "/TN", spec.Label, "/FO", "LIST")
	if err != nil {
		state.Detail = "not registered"
		return state, nil
	}
	state.Installed = true
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			state.Detail = strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
			state.Running = state.Detail == "Running"
		}
	}
	return state, nil
}

// utf16LE encodes UTF-8 text as UTF-16LE with a byte-order mark.
func utf16LE(text []byte) []byte {
	runes := []rune(string(text))
	out := make([]byte, 0, 2+len(runes)*2)
	out = append(out, 0xFF, 0xFE)
	for _, r := range runes {
		if r >= 0x10000 {
			r -= 0x10000
			hi, lo := 0xD800+(r>>10), 0xDC00+(r&0x3FF)
			out = append(out, byte(hi), byte(hi>>8), byte(lo), byte(lo>>8))
			continue
		}
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}
