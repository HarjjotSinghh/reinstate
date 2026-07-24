// Package device detects OS, architecture, and WSL environment.
package device

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Info describes the current (or injected) device environment.
type Info struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	PlatformID   string `json:"platform_id"`
	IsWSL        bool   `json:"is_wsl"`
	WSLVersion   int    `json:"wsl_version,omitempty"`
	Native       bool   `json:"native"`
	Supported    bool   `json:"supported"`
	RefuseReason string `json:"refuse_reason,omitempty"`
}

// Probes allow tests to inject environment signals without reading real sessions.
type Probes struct {
	GOOS        string
	GOARCH      string
	Env         map[string]string
	ProcVersion string // /proc/version content (Linux)
	OSRelease   string // /etc/os-release content
}

// DefaultProbes uses the real runtime and environment.
func DefaultProbes() Probes {
	return Probes{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
		Env:    map[string]string{},
	}
}

func (p Probes) env(k string) string {
	if p.Env != nil {
		if v, ok := p.Env[k]; ok {
			return v
		}
	}
	return os.Getenv(k)
}

// Detect classifies the device environment.
func Detect(p Probes) (Info, error) {
	if p.GOOS == "" {
		p.GOOS = runtime.GOOS
	}
	if p.GOARCH == "" {
		p.GOARCH = runtime.GOARCH
	}
	info := Info{
		OS:        p.GOOS,
		Arch:      p.GOARCH,
		Native:    true,
		Supported: true,
	}

	// WSL detection
	if p.GOOS == "linux" {
		wsl := strings.TrimSpace(p.env("WSL_DISTRO_NAME"))
		wslVerEnv := strings.TrimSpace(p.env("WSL_INTEROP"))
		proc := strings.ToLower(p.ProcVersion)
		if p.ProcVersion == "" {
			if b, err := os.ReadFile("/proc/version"); err == nil {
				proc = strings.ToLower(string(b))
			}
		} else {
			proc = strings.ToLower(p.ProcVersion)
		}
		isWSL := wsl != "" || wslVerEnv != "" || strings.Contains(proc, "microsoft") || strings.Contains(proc, "wsl")
		if isWSL {
			info.IsWSL = true
			info.Native = false
			// WSL1 vs WSL2: WSL2 has WSL_INTEROP or "microsoft-standard-wsl2" / "microsoft-standard"
			// WSL1 historically lacks WSL_INTEROP and reports "Microsoft" without WSL2 markers.
			ver := 2
			if strings.Contains(proc, "wsl2") || strings.Contains(proc, "microsoft-standard") || wslVerEnv != "" {
				ver = 2
			} else if strings.Contains(proc, "microsoft") && !strings.Contains(proc, "wsl2") && wslVerEnv == "" && p.env("WSL_VERSION") == "1" {
				ver = 1
			}
			if v := p.env("WSL_VERSION"); v == "1" {
				ver = 1
			}
			if v := p.env("WSL_VERSION"); v == "2" {
				ver = 2
			}
			info.WSLVersion = ver
			if ver == 1 {
				info.Supported = false
				info.RefuseReason = "WSL1 is unsupported; use native Windows or WSL2"
			}
			info.PlatformID = fmt.Sprintf("linux-wsl%d-%s", ver, p.GOARCH)
			return info, nil
		}
		info.PlatformID = fmt.Sprintf("linux-%s", p.GOARCH)
		return info, nil
	}

	switch p.GOOS {
	case "darwin":
		info.PlatformID = fmt.Sprintf("darwin-%s", p.GOARCH)
	case "windows":
		info.PlatformID = fmt.Sprintf("windows-%s", p.GOARCH)
	default:
		info.PlatformID = fmt.Sprintf("%s-%s", p.GOOS, p.GOARCH)
	}
	return info, nil
}

// PlatformID is a stable identifier for envelope metadata.
func PlatformID() string {
	info, err := Detect(DefaultProbes())
	if err != nil {
		return runtime.GOOS + "-" + runtime.GOARCH
	}
	return info.PlatformID
}
