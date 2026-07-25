// Package doctor produces redacted diagnostics and synthetic self-tests.
package doctor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/codex"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
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

	// Adapter compatibility probes inspect only roots/layout markers.
	adapters := []adapter.Adapter{&claude.Adapter{}, &codex.Adapter{}}
	for _, selected := range adapters {
		install, compatibility, err := selected.Detect(ctx)
		name := selected.Name()
		if err != nil {
			rep.Agents[name] = string(adapter.CompatibilityUntested)
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + name, Status: "warn", Message: Redact(err.Error()),
			})
			continue
		}
		rep.Agents[name] = string(compatibility)
		switch compatibility {
		case adapter.CompatibilitySupported:
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + name, Status: "ok",
				Message: fmt.Sprintf("%s (%s)", compatibility, install.Version),
			})
		case adapter.CompatibilityNotInstalled:
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + name, Status: "skip", Message: "not installed",
			})
		case adapter.CompatibilityUntested:
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + name, Status: "warn", Message: "layout/version untested; writes blocked",
			})
		default:
			rep.Checks = append(rep.Checks, Check{
				Name: "agent." + name, Status: "fail", Message: "unsupported layout/version",
				Code: exitcode.Compatibility,
			})
		}
	}

	if err := credentials.NewKeyringStore().Probe(); err != nil {
		rep.Checks = append(rep.Checks, Check{
			Name: "keyring", Status: "warn", Message: Redact(err.Error()),
		})
	} else {
		rep.Checks = append(rep.Checks, Check{
			Name: "keyring", Status: "ok", Message: "OS keyring provider reachable",
		})
	}

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
