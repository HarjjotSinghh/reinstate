// Package doctor produces redacted diagnostics and synthetic self-tests.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/device"
	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/version"
)

// Check is one diagnostic item.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok, warn, fail, skip
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Report is a redacted diagnostic report.
type Report struct {
	Version   string            `json:"version"`
	Platform  string            `json:"platform"`
	Home      string            `json:"home"`
	Summary   string            `json:"summary"`
	Checks    []Check           `json:"checks"`
	Agents    map[string]string `json:"agents"`
	SelfTest  string            `json:"self_test,omitempty"`
	Generated string            `json:"generated_at"`
}

// Options configure doctor.
type Options struct {
	Home     string
	SelfTest bool
}

// Run builds a redacted report. It never reads real vendor session contents.
func Run(ctx context.Context, opts Options) (*Report, error) {
	_ = ctx
	home := opts.Home
	if home == "" {
		h, err := config.Home()
		if err != nil {
			return nil, err
		}
		home = h
	}
	info, _ := device.Detect(device.DefaultProbes())
	rep := &Report{
		Version:   version.Version,
		Platform:  info.PlatformID,
		Home:      RedactPath(home),
		Agents:    map[string]string{},
		Generated: time.Now().UTC().Format(time.RFC3339),
	}

	// Config check
	cfgPath := config.ConfigPath(home)
	if _, err := os.Stat(cfgPath); err != nil {
		rep.Checks = append(rep.Checks, Check{
			Name: "config", Status: "fail", Message: "config missing", Code: exitcode.Config,
		})
	} else if _, err := config.LoadConfig(home); err != nil {
		rep.Checks = append(rep.Checks, Check{
			Name: "config", Status: "fail", Message: Redact(err.Error()), Code: exitcode.Config,
		})
	} else {
		rep.Checks = append(rep.Checks, Check{Name: "config", Status: "ok", Message: "config valid"})
	}

	// Device support
	if !info.Supported {
		rep.Checks = append(rep.Checks, Check{
			Name: "device", Status: "fail", Message: Redact(info.RefuseReason), Code: exitcode.Compatibility,
		})
	} else {
		rep.Checks = append(rep.Checks, Check{Name: "device", Status: "ok", Message: info.PlatformID})
	}

	// Agent roots (existence only — no session reads)
	for _, agent := range []string{"claude", "codex"} {
		root := probeAgentRoot(agent)
		if root == "" {
			rep.Agents[agent] = string(compatNotInstalled)
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + agent, Status: "skip", Message: "not installed",
			})
			continue
		}
		rep.Agents[agent] = "PRESENT"
		rep.Checks = append(rep.Checks, Check{
			Name: "agent." + agent, Status: "ok", Message: "root present (path redacted)",
		})
	}

	// Keyring availability (informational; no secrets)
	rep.Checks = append(rep.Checks, Check{
		Name: "keyring", Status: "ok", Message: "keyring abstraction available (not probed for secrets)",
	})

	if opts.SelfTest {
		if err := SelfTest(home); err != nil {
			rep.SelfTest = "fail"
			rep.Checks = append(rep.Checks, Check{
				Name: "self_test", Status: "fail", Message: Redact(err.Error()), Code: exitcode.Runtime,
			})
		} else {
			rep.SelfTest = "ok"
			rep.Checks = append(rep.Checks, Check{Name: "self_test", Status: "ok", Message: "synthetic self-test passed"})
		}
	}

	rep.Summary = summarize(rep)
	return rep, nil
}

const compatNotInstalled = "NOT_INSTALLED"

func probeAgentRoot(agent string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	var candidates []string
	switch agent {
	case "claude":
		candidates = []string{
			filepath.Join(home, ".claude"),
			filepath.Join(home, ".config", "claude"),
		}
	case "codex":
		candidates = []string{
			filepath.Join(home, ".codex"),
			filepath.Join(home, ".config", "codex"),
		}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

func summarize(rep *Report) string {
	fails := 0
	for _, c := range rep.Checks {
		if c.Status == "fail" {
			fails++
		}
	}
	if fails == 0 {
		return "all checks passed"
	}
	return fmt.Sprintf("%d check(s) failed", fails)
}

// ExitCode maps a report to a process exit code.
func ExitCode(rep *Report) int {
	if rep == nil {
		return exitcode.Runtime
	}
	code := exitcode.OK
	for _, c := range rep.Checks {
		if c.Status != "fail" {
			continue
		}
		if c.Code > code {
			code = c.Code
		}
		if c.Code == 0 {
			code = exitcode.Runtime
		}
	}
	return code
}

// FormatHuman returns a redacted human report.
func FormatHuman(rep *Report) string {
	if rep == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "reinstate doctor\nversion: %s\nplatform: %s\nhome: %s\nsummary: %s\n",
		rep.Version, rep.Platform, rep.Home, rep.Summary)
	for _, c := range rep.Checks {
		fmt.Fprintf(&b, "- [%s] %s: %s\n", c.Status, c.Name, c.Message)
	}
	if rep.SelfTest != "" {
		fmt.Fprintf(&b, "self_test: %s\n", rep.SelfTest)
	}
	_ = runtime.GOOS
	return b.String()
}
