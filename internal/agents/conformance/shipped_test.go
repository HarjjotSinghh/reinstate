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
		{"kimi", catalog.Kimi(), Fixtures{Root: "testdata/sessionindex/kimi", OS: []string{"macos", "windows"}}},
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

// TestTierJourneyGapReadsWhatAReportCovers is the negative control for the
// content-aware half of the evidence gate.
//
// devicePlatformGap refuses a claim whose reports do not span both platforms,
// but it cannot tell what those reports are about. Both failures below actually
// shipped: Grok cited two release-acceptance reports that mention it only in
// index and handoff-source rows, and Qwen's T4 claim passed conformance while
// its only Windows report covered T3, because a Windows filename was present
// either way.
func TestTierJourneyGapReadsWhatAReportCovers(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}

	const (
		macOSJourneyT3   = "docs/testing/results/2026-08-22-macos-qwen-t3.md"
		windowsJourneyT3 = "docs/testing/results/2026-08-22-windows-qwen-t3.md"
		macOSJourneyT4   = "docs/testing/results/2026-08-22-macos-qwen-t4.md"
		windowsJourneyT4 = "docs/testing/results/2026-08-22-windows-qwen-t4.md"
		macOSRelease     = "docs/testing/results/2026-08-21-macos-phase5-V050RC6.md"
		windowsRelease   = "docs/testing/results/2026-08-21-windows-phase5-V050RC6.md"
	)

	qwen := func() agents.Descriptor {
		for _, d := range agents.All() {
			if d.Key == "qwen" {
				return d
			}
		}
		t.Fatal("qwen is not in the catalog")
		return agents.Descriptor{}
	}

	for _, testCase := range []struct {
		name    string
		reports []string
		wantGap string
	}{
		{
			name:    "every rung on both platforms",
			reports: []string{macOSJourneyT3, windowsJourneyT3, macOSJourneyT4, windowsJourneyT4},
			wantGap: "",
		},
		{
			// The hole that shipped: a Windows report exists, but it covers T3.
			name:    "the windows report covers a lower rung",
			reports: []string{macOSJourneyT3, windowsJourneyT3, macOSJourneyT4},
			wantGap: "T4 on windows",
		},
		{
			// The Grok trap: reports from both platforms that are about a
			// release, not about this agent reaching this rung.
			name:    "release acceptance is not a tier journey",
			reports: []string{macOSRelease, windowsRelease},
			wantGap: "T3 on macos and windows, T4 on macos and windows",
		},
		{
			name:    "a skipped lower rung is still a gap",
			reports: []string{macOSJourneyT4, windowsJourneyT4},
			wantGap: "T3 on macos and windows",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			desc := qwen()
			desc.Evidence.DeviceReports = testCase.reports
			if got := tierJourneyGap(desc, root); got != testCase.wantGap {
				t.Fatalf("tierJourneyGap = %q, want %q", got, testCase.wantGap)
			}
		})
	}
}

// TestLegacyEvidenceStaysAccepted keeps the grandfathered reports working. They
// predate AGENT-TIER-JOURNEY-V1 and carry no tier vocabulary at all, so no rule
// could read a rung from them; rejecting them would falsify shipped history
// rather than improve it.
func TestLegacyEvidenceStaysAccepted(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"claude", "codex"} {
		var desc agents.Descriptor
		for _, d := range agents.All() {
			if d.Key == key {
				desc = d
			}
		}
		if desc.Key == "" {
			t.Fatalf("%s is not in the catalog", key)
		}
		if gap := tierJourneyGap(desc, root); gap != "" {
			t.Fatalf("%s (%s) reported a journey gap: %s", key, desc.Tier, gap)
		}
	}
}

// TestBelowT3NeedsNoJourney keeps the rule from spreading. T1 and T2 require no
// device journey, and a reader or an index source is not evidence of a resume.
func TestBelowT3NeedsNoJourney(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range agents.All() {
		if d.Tier >= agents.TierResume {
			continue
		}
		if gap := tierJourneyGap(d, root); gap != "" {
			t.Fatalf("%s at %s was asked for a journey: %s", d.Key, d.Tier, gap)
		}
	}
}
