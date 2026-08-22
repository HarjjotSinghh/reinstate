package conformance

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
)

func TestShippedAgentsConformance(t *testing.T) {
	cases := []struct {
		name     string
		desc     agents.Descriptor
		fixtures Fixtures
	}{
		{"claude", catalog.Claude(), Fixtures{Root: "testdata/sessionindex/claude", OS: []string{"macos", "windows"}}},
		{"codex", catalog.Codex(), Fixtures{Root: "testdata/sessionindex/codex/forks"}},
		{"gemini", catalog.Gemini(), Fixtures{Root: "testdata/sessionindex/gemini", OS: []string{"macos", "windows"}}},
		{"opencode", catalog.OpenCode(), Fixtures{Root: "testdata/sessionindex/opencode", OS: []string{"macos", "windows"}}},
		{"grok", catalog.Grok(), Fixtures{Root: "testdata/sessionindex/grok", OS: []string{"macos", "windows"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Every check fails the suite, evidence included. It used to be
			// collected and logged, so a descriptor could name a storage page,
			// probe report, or fixture that did not exist and nothing caught it.
			for _, check := range Evaluate(tc.desc, tc.fixtures) {
				if check.Err != nil {
					t.Errorf("%s: %v", check.Name, check.Err)
				}
			}
		})
	}
}

// TestShippedEvidenceIsComplete requires every shipped descriptor to name
// evidence that actually exists. The previous form allowed any error whose
// text mentioned StoragePage or ProbeReports, which is what every descriptor
// reported, so a missing fixture path rode along undetected.
func TestShippedEvidenceIsComplete(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	// Every registered agent, not a hand-maintained list. The list held five
	// descriptors and had to be edited by hand whenever an agent was promoted,
	// so an agent raised to T3 was simply absent from the evidence gate — the
	// gate reported nothing because it was never asked. A promotion is exactly
	// when this check matters, and it is exactly when a list like that is
	// forgotten.
	all := agents.All()
	if len(all) == 0 {
		t.Fatal("no agents registered; the catalog import is not taking effect")
	}
	for _, desc := range all {
		if err := checkEvidence(desc, root); err != nil {
			t.Errorf("%s: %v", desc.Key, err)
		}
	}
}

// TestBrokenEvidencePathIsCaught is the negative control for Matrix A9.
// TestSinglePlatformDeviceReportIsCaught is the negative control for the T3+
// device-journey rule.
//
// checkEvidence used to accept any non-empty DeviceReports, so a verified-resume
// or handoff-destination claim could rest on one macOS journey and pass. The
// tier ladder is not an honour system at exactly this rung: T3 hands a live
// session to a vendor binary and T4 writes a new one, and argv quoting, path
// separators and process identity all diverge across platforms.
func TestSinglePlatformDeviceReportIsCaught(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}

	// Real paths, so the only thing under test is the platform rule rather than
	// a file that happens not to exist.
	const (
		macOSReport   = "docs/testing/results/2026-08-11-macos-phase3-V030.md"
		windowsReport = "docs/testing/results/2026-08-11-windows-phase3-V030.md"
	)

	for _, testCase := range []struct {
		name    string
		reports []string
		wantGap string
	}{
		{name: "macos only", reports: []string{macOSReport}, wantGap: "windows"},
		{name: "windows only", reports: []string{windowsReport}, wantGap: "macos"},
		{
			// A journey on neither device is the case the old check missed
			// least visibly: the list is non-empty, so it read as evidence.
			name:    "neither platform named",
			reports: []string{"docs/testing/results/2026-08-11-phase1-inventory.md"},
			wantGap: "macos and windows",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			desc := catalog.Claude()
			desc.Evidence.DeviceReports = testCase.reports
			err := checkEvidence(desc, root)
			if err == nil {
				t.Fatalf("a %s T3 claim passed checkEvidence", testCase.name)
			}
			if want := "DeviceReports on " + testCase.wantGap; !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not name the missing platform %q", err, want)
			}
		})
	}

	t.Run("both platforms pass", func(t *testing.T) {
		desc := catalog.Claude()
		desc.Evidence.DeviceReports = []string{macOSReport, windowsReport}
		if err := checkEvidence(desc, root); err != nil {
			t.Fatalf("a two-platform T3 claim was rejected: %v", err)
		}
	})

	t.Run("wsl does not substitute for native windows", func(t *testing.T) {
		// WSL2 is a separate device with a separate tree. A journey run there
		// says nothing about native Windows argv or path handling, which is
		// most of what T3 and T4 actually exercise.
		desc := catalog.Claude()
		desc.Evidence.DeviceReports = []string{macOSReport, "docs/testing/results/2026-08-11-wsl-phase3-V030.md"}
		err := checkEvidence(desc, root)
		if err == nil {
			t.Fatal("a WSL journey satisfied the native Windows requirement")
		}
		if !strings.Contains(err.Error(), "DeviceReports on windows") {
			t.Fatalf("error %q does not report native Windows as missing", err)
		}
	})

	t.Run("a tier below T3 is unaffected", func(t *testing.T) {
		// T1 and T2 require no device journey at all; the rule must not start
		// demanding one.
		desc := catalog.Claude()
		desc.Tier = agents.TierHandoffFrom
		desc.NewTarget, desc.NewSyncAdapter = nil, nil
		desc.Evidence.DeviceReports = nil
		if err := checkEvidence(desc, root); err != nil {
			t.Fatalf("a T2 descriptor with no device report was rejected: %v", err)
		}
	})
}

func TestBrokenEvidencePathIsCaught(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	desc := catalog.Claude()
	desc.Evidence.Fixtures = append(
		append([]string(nil), desc.Evidence.Fixtures...),
		"testdata/sessionindex/claude/this-path-does-not-exist",
	)
	if err := checkEvidence(desc, root); err == nil {
		t.Fatal("a descriptor naming a nonexistent evidence path passed checkEvidence")
	}
}
