package catalog

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/conformance"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestQwenConformance(t *testing.T) {
	conformance.Run(t, Qwen(), conformance.Fixtures{
		Root: "testdata/sessionindex/qwen",
		OS:   []string{"macos", "windows"},
	})
}

func TestQwenIsResumeButNotADestination(t *testing.T) {
	d := Qwen()
	if d.Tier != agents.TierResume {
		t.Fatalf("tier = %s, want T3", d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.NewIndexSource == nil || d.NewReader == nil {
		t.Fatal("T3 requires an index source and a transcript reader")
	}
	if d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T3 descriptor must not ship target or sync constructors")
	}
	if d.Native == nil || d.Version == nil {
		t.Fatal("T3 requires a native launch spec and a version range")
	}
	if d.Native.NewSession != nil {
		t.Fatal("NewSession is a T4 claim")
	}
}

func TestQwenNativeArgvMatchesTheMeasuredVendorSurface(t *testing.T) {
	native := Qwen().Native
	if native.Executable != "qwen" {
		t.Fatalf("executable = %q", native.Executable)
	}
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{"resume", native.Resume, []string{"--resume", "{{.SessionID}}"}},
		{"fork", native.Fork, []string{"--resume", "{{.SessionID}}", "--fork-session"}},
		{"continue", native.Continue, []string{"--continue"}},
	}
	for _, tt := range tests {
		if len(tt.got) != len(tt.want) {
			t.Fatalf("%s argv = %v, want %v", tt.name, tt.got, tt.want)
		}
		for i := range tt.want {
			if tt.got[i] != tt.want[i] {
				t.Fatalf("%s argv = %v, want %v", tt.name, tt.got, tt.want)
			}
		}
	}
	// A resume template that never substitutes an id would open the vendor's
	// interactive session picker instead of the session the operator named.
	if !contains(native.Resume, "{{.SessionID}}") || !contains(native.Fork, "{{.SessionID}}") {
		t.Fatalf("resume/fork argv must substitute the session id: %v / %v", native.Resume, native.Fork)
	}
}

func TestParseQwenVersion(t *testing.T) {
	tests := []struct {
		name   string
		output agents.VersionOutput
		want   string
		ok     bool
	}{
		{name: "bundled npm install", output: agents.VersionOutput{Stdout: "0.21.12\n"}, want: "0.21.12", ok: true},
		{name: "managed self-update", output: agents.VersionOutput{Stdout: "0.21.13\n"}, want: "0.21.13", ok: true},
		{name: "windows newline", output: agents.VersionOutput{Stdout: "0.21.13\r\n"}, want: "0.21.13", ok: true},
		{name: "no trailing newline", output: agents.VersionOutput{Stdout: "0.21.13"}, want: "0.21.13", ok: true},
		// The QWEN_HOME redirect warning lands on stderr, and a version read
		// alongside a warning is not an authoritative version.
		{name: "root redirect warning on stderr", output: agents.VersionOutput{Stdout: "0.21.13\n", Stderr: "Warning: QWEN_HOME points to …\n"}},
		{name: "prerelease suffix", output: agents.VersionOutput{Stdout: "0.22.0-nightly\n"}},
		{name: "name prefix", output: agents.VersionOutput{Stdout: "qwen 0.21.13\n"}},
		{name: "leading zeros", output: agents.VersionOutput{Stdout: "00.21.13\n"}},
		{name: "two lines", output: agents.VersionOutput{Stdout: "0.21.13\n0.21.12\n"}},
		{name: "ansi", output: agents.VersionOutput{Stdout: "\x1b[32m0.21.13\x1b[0m\n"}},
		{name: "empty", output: agents.VersionOutput{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseQwenVersion(tt.output)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parseQwenVersion() = %q, %t, want %q, %t", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestQwenVersionRangeSpansTheSelfUpdater is the reason the range is not a
// single version: Qwen installs updates into <QWEN_HOME>/updates/npm and runs
// them, so the same machine answers --version differently depending on which
// root is in scope. Both ends were measured on macOS on 2026-08-22.
func TestQwenVersionRangeSpansTheSelfUpdater(t *testing.T) {
	version := Qwen().Version
	if version.Min != "0.21.12" || version.Max != "0.21.13" {
		t.Fatalf("range = %s–%s, want the measured 0.21.12–0.21.13", version.Min, version.Max)
	}
	for _, in := range []string{"0.21.12", "0.21.13"} {
		if !adapter.StableVersionInRange(in, version.Min, version.Max) {
			t.Fatalf("%s is a measured version but falls outside the declared range", in)
		}
	}
	// 0.21.15 exists and was seen installing itself, but only its --version
	// output has been observed. Fail closed until its layout is verified.
	if adapter.StableVersionInRange("0.21.15", version.Min, version.Max) {
		t.Fatal("0.21.15 is unverified and must report UNTESTED, not SUPPORTED")
	}
}

func TestQwenProcessSpecRecognizesTheRelaunchedWorker(t *testing.T) {
	process := Qwen().Process
	if !contains(process.Images, "qwen") {
		t.Fatalf("images = %v", process.Images)
	}
	// Observed argv of the worker the launcher re-execs. The launcher itself
	// runs as `node …/bin/qwen` and is not what this marker matches.
	const worker = "/users/u/.local/lib/node_modules/@qwen-code/qwen-code/cli.js"
	matched := false
	for _, marker := range process.NodeMarkers {
		if strings.Contains(worker, strings.ToLower(marker)) {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("node markers %v do not match the observed worker command line", process.NodeMarkers)
	}
}

func TestQwenCitesAMacOSDeviceJourney(t *testing.T) {
	d := Qwen()
	if len(d.Evidence.DeviceReports) == 0 {
		t.Fatal("T3 requires a device report")
	}
	var macOS, windows bool
	for _, report := range d.Evidence.DeviceReports {
		macOS = macOS || strings.Contains(report, "-macos-")
		windows = windows || strings.Contains(report, "-windows-")
	}
	if !macOS {
		t.Fatalf("device reports = %v, want a macOS journey", d.Evidence.DeviceReports)
	}
	if windows {
		// Reaching here is good news, and means this test has outlived its
		// purpose: replace it with a dual-platform assertion.
		t.Log("a native Windows device report is now cited; tighten this test")
	}
}

func TestQwenReaderIsTheQwenReader(t *testing.T) {
	reader, err := Qwen().NewReader(agents.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if reader == nil {
		t.Fatal("NewReader returned no reader")
	}
	if reader.Name() != sessionindex.AgentQwen {
		t.Fatalf("reader = %q, want the Qwen reader", reader.Name())
	}
	// Qwen's record keys match Claude Code's and its storage layout is the same
	// shape, which is exactly why reusing the Claude reader is the tempting
	// wrong answer: it would find no message text at all.
	if _, isClaude := reader.(*transcript.ClaudeReader); isClaude {
		t.Fatal("Qwen must not be wired to the Claude reader")
	}
}

func TestQwenExcludesSubagentTranscripts(t *testing.T) {
	d := Qwen()
	if !contains(d.Storage.Excluded, "subagents") {
		t.Fatalf("excluded = %v, missing subagents (per-subagent transcripts are not sessions)", d.Storage.Excluded)
	}
}

func TestQwenCitesBothPlatformProbes(t *testing.T) {
	d := Qwen()
	var macOS, windows bool
	for _, report := range d.Evidence.ProbeReports {
		macOS = macOS || strings.Contains(report, "-macos-")
		windows = windows || strings.Contains(report, "-windows-")
	}
	if !macOS || !windows {
		t.Fatalf("probe reports = %v, want one macOS and one native Windows", d.Evidence.ProbeReports)
	}
	var indexMacOS, indexWindows bool
	for _, path := range d.Evidence.Fixtures {
		indexMacOS = indexMacOS || path == "testdata/sessionindex/qwen/macos"
		indexWindows = indexWindows || path == "testdata/sessionindex/qwen/windows"
	}
	if !indexMacOS || !indexWindows {
		t.Fatalf("fixtures = %v, want one index fixture per platform", d.Evidence.Fixtures)
	}
}

func TestQwenExcludesUpdaterTree(t *testing.T) {
	d := Qwen()
	if !contains(d.Storage.Excluded, "updates") {
		t.Fatalf("excluded = %v, missing updates (npm self-updater drowned the Windows probe)", d.Storage.Excluded)
	}
}
