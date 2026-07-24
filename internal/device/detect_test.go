package device

import "testing"

func TestDetectMacOS(t *testing.T) {
	info, err := Detect(Probes{GOOS: "darwin", GOARCH: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	if info.PlatformID != "darwin-arm64" || !info.Native || !info.Supported {
		t.Fatalf("%+v", info)
	}
	info2, _ := Detect(Probes{GOOS: "darwin", GOARCH: "amd64"})
	if info2.PlatformID != "darwin-amd64" {
		t.Fatalf("%+v", info2)
	}
}

func TestDetectWindows(t *testing.T) {
	info, err := Detect(Probes{GOOS: "windows", GOARCH: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if info.PlatformID != "windows-amd64" || info.IsWSL {
		t.Fatalf("%+v", info)
	}
}

func TestDetectWSL2(t *testing.T) {
	info, err := Detect(Probes{
		GOOS:   "linux",
		GOARCH: "amd64",
		Env: map[string]string{
			"WSL_DISTRO_NAME": "Ubuntu",
			"WSL_INTEROP":     "/run/WSL/1_interop",
			"WSL_VERSION":     "2",
		},
		ProcVersion: "Linux version 5.15.0-microsoft-standard-WSL2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsWSL || info.WSLVersion != 2 || !info.Supported {
		t.Fatalf("%+v", info)
	}
	if info.PlatformID != "linux-wsl2-amd64" {
		t.Fatalf("platform %q", info.PlatformID)
	}
}

func TestDetectWSL1Refused(t *testing.T) {
	info, err := Detect(Probes{
		GOOS:   "linux",
		GOARCH: "amd64",
		Env: map[string]string{
			"WSL_DISTRO_NAME": "Ubuntu",
			"WSL_VERSION":     "1",
		},
		ProcVersion: "Linux version 4.4.0-19041-Microsoft",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsWSL || info.WSLVersion != 1 {
		t.Fatalf("%+v", info)
	}
	if info.Supported || info.RefuseReason == "" {
		t.Fatalf("expected WSL1 refusal: %+v", info)
	}
}

func TestDetectLinuxNonWSL(t *testing.T) {
	info, err := Detect(Probes{
		GOOS:        "linux",
		GOARCH:      "amd64",
		ProcVersion: "Linux version 6.5.0-generic",
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.IsWSL || info.PlatformID != "linux-amd64" {
		t.Fatalf("%+v", info)
	}
}
