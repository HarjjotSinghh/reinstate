package catalog

import (
	"strings"
	"testing"

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

func TestQwenIsHandoffSourceOnly(t *testing.T) {
	d := Qwen()
	if d.Tier != agents.TierHandoffFrom {
		t.Fatalf("tier = %s, want T2", d.Tier)
	}
	if d.T0Reason != "" {
		t.Fatalf("T0Reason = %q, want empty above T0", d.T0Reason)
	}
	if d.NewIndexSource == nil || d.NewReader == nil {
		t.Fatal("T2 requires an index source and a transcript reader")
	}
	if d.NewTarget != nil || d.NewSyncAdapter != nil {
		t.Fatal("T2 descriptor must not ship target or sync constructors")
	}
	if d.Native != nil || d.Version != nil {
		t.Fatal("native resume and a version range are T3 claims")
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
