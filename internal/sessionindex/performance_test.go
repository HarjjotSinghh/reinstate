package sessionindex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/fileidentity"
)

func TestBoundedPromptAccumulatorPreservesIndexedSearchText(t *testing.T) {
	parts := []string{
		" first\ncontrolled ",
		"second\t🙂 value",
		"\x1b[31mthird\x1b[0m",
		strings.Repeat("bounded ", MaxSearchTextBytes),
		"ignored after bound",
	}
	var optimized boundedText
	legacy := ""
	for _, part := range parts {
		optimized.Add(part)
		legacy = BuildSearchText(legacy, part)
	}
	got := BuildSearchText("prefix", optimized.String(), "suffix")
	want := BuildSearchText("prefix", legacy, "suffix")
	if got != want {
		t.Fatalf("optimized search text differs from the legacy result: got %d bytes, want %d", len(got), len(want))
	}
	if len(got) > MaxSearchTextBytes || !utf8.ValidString(got) {
		t.Fatalf("optimized search text is invalid or unbounded: %d bytes", len(got))
	}
}

func TestBoundedPromptAccumulatorCutoffMatchesLegacy(t *testing.T) {
	parts := []string{strings.Repeat("a", MaxSearchTextBytes-2), "🙂", "z"}
	assertBoundedPromptAccumulatorEquivalent(t, parts)
}

func TestBoundedPromptAccumulatorDeterministicEquivalence(t *testing.T) {
	random := rand.New(rand.NewSource(0x96))
	tokens := []string{"", "plain", " ", "\t", "\n", "🙂", "界", "\x1b[31m", "\x1b[0m", "\u202e", "a", "z"}
	for sample := range 256 {
		parts := make([]string, random.Intn(80))
		for index := range parts {
			var value strings.Builder
			for range random.Intn(24) {
				value.WriteString(tokens[random.Intn(len(tokens))])
			}
			parts[index] = value.String()
		}
		assertBoundedPromptAccumulatorEquivalent(t, parts)
		if t.Failed() {
			t.Fatalf("random equivalence sample %d failed", sample)
		}
	}

	// Exercise the byte cutoff with many independently sanitized chunks rather
	// than only one oversized value.
	boundary := make([]string, 300)
	for index := range boundary {
		boundary[index] = strings.Repeat(string(rune('a'+index%26)), 1_023)
	}
	boundary = append(boundary, "🙂", "z", "\n final")
	assertBoundedPromptAccumulatorEquivalent(t, boundary)
}

func assertBoundedPromptAccumulatorEquivalent(t *testing.T, parts []string) {
	t.Helper()
	var optimized boundedText
	legacy := ""
	for _, part := range parts {
		optimized.Add(part)
		legacy = BuildSearchText(legacy, part)
	}
	got := BuildSearchText("prefix", optimized.String(), "suffix")
	want := BuildSearchText("prefix", legacy, "suffix")
	if got != want {
		t.Fatalf("optimized accumulation differs from legacy: got %d bytes, want %d", len(got), len(want))
	}
}

func TestBoundedPromptAccumulatorStaysLinearAndBounded(t *testing.T) {
	const messages = 10_000
	prompt := "controlled synthetic prompt with unicode beta and bounded metadata"
	allocations := testing.AllocsPerRun(3, func() {
		var text boundedText
		for range messages {
			text.Add(prompt)
		}
		value := text.String()
		if len(value) > MaxSearchTextBytes || !utf8.ValidString(value) {
			panic("prompt accumulator became invalid or unbounded")
		}
	})
	// The exact number is intentionally generous across Go/OS versions. The
	// former quadratic implementation exceeds 170,000 allocations for this
	// fixture; linear bounded accumulation remains below 20,000.
	if allocations > 20_000 {
		t.Fatalf("prompt accumulator allocations = %.0f, want <= 20000", allocations)
	}
}

func BenchmarkBoundedPromptAccumulator(b *testing.B) {
	for _, messages := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("messages_%d", messages), func(b *testing.B) {
			prompt := "controlled synthetic prompt with unicode beta and bounded metadata"
			b.ReportAllocs()
			for range b.N {
				var text boundedText
				for range messages {
					text.Add(prompt)
				}
				if text.String() == "" {
					b.Fatal("empty prompt accumulation")
				}
			}
		})
	}
}

// BenchmarkExecLaunchRunnerImmediateChild separates the shell-free child
// creation/wait boundary from interactive vendor time. It executes the current
// test binary as an immediate, controlled child while retaining the production
// executable/workspace identity checks. No vendor or session source is read or
// mutated.
func BenchmarkExecLaunchRunnerImmediateChild(b *testing.B) {
	b.Setenv("REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		b.Fatal(err)
	}
	executableIdentity, err := fileidentity.CaptureExecutable(context.Background(), executable)
	if err != nil {
		b.Fatal(err)
	}
	workspacePath := b.TempDir()
	workspaceIdentity, err := fileidentity.Capture(workspacePath)
	if err != nil {
		b.Fatal(err)
	}
	plan := LaunchPlan{
		Agent: AgentClaude, SessionRef: "claude:controlled-performance-child",
		Operation: OperationResume, Executable: "controlled-test-helper",
		Args: []string{"-test.run=^TestPhase3PerformanceLaunchHelper$"}, Dir: workspacePath,
	}
	runner := ExecLaunchRunner{
		Executable: executable, ExecutableIdentity: executableIdentity,
		WorkspaceIdentity: workspaceIdentity, Stdin: bytes.NewReader(nil),
		Stdout: io.Discard, Stderr: io.Discard,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := RunLaunch(context.Background(), plan, runner); err != nil {
			b.Fatal(err)
		}
	}
}

func TestPhase3PerformanceLaunchHelper(t *testing.T) {}

func BenchmarkClaudeLargeCorpusRefresh(b *testing.B) {
	root := b.TempDir()
	projects := filepath.Join(root, "projects", "controlled")
	if err := os.MkdirAll(projects, 0o700); err != nil {
		b.Fatal(err)
	}
	for index := range 1_000 {
		path := filepath.Join(projects, fmt.Sprintf("controlled-%04d.jsonl", index))
		body := fmt.Sprintf("{\"type\":\"user\",\"sessionId\":\"controlled-%04d\",\"cwd\":\"/tmp/controlled\",\"message\":{\"role\":\"user\",\"content\":\"controlled prompt %04d\"}}\n", index, index)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	source := NewClaudeSource(root)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err := source.Scan(context.Background())
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Records) != 1_000 {
			b.Fatalf("records = %d", len(result.Records))
		}
	}
}

func BenchmarkClaudeLongSessionRefresh(b *testing.B) {
	root := b.TempDir()
	projects := filepath.Join(root, "projects", "controlled")
	if err := os.MkdirAll(projects, 0o700); err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(projects, "controlled-long.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		b.Fatal(err)
	}
	for index := range 10_000 {
		line := fmt.Sprintf("{\"type\":\"user\",\"sessionId\":\"controlled-long\",\"cwd\":\"/tmp/controlled\",\"message\":{\"role\":\"user\",\"content\":\"controlled prompt %05d with bounded unicode beta\"}}\n", index)
		if _, err := file.WriteString(line); err != nil {
			_ = file.Close()
			b.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		b.Fatal(err)
	}
	source := NewClaudeSource(root)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err := source.Scan(context.Background())
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Records) != 1 || result.Records[0].MessageCount != 10_000 {
			b.Fatalf("unexpected long-session result: %+v", result.Records)
		}
	}
}

func BenchmarkLargeCorpusIndexLayers(b *testing.B) {
	records := syntheticPerformanceRecords(1_000)
	path := filepath.Join(b.TempDir(), "index.sqlite")
	store, err := Open(path)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := store.ReplaceSource(context.Background(), AgentClaude, records); err != nil {
		b.Fatal(err)
	}
	if err := store.Close(); err != nil {
		b.Fatal(err)
	}

	b.Run("open", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			opened, err := Open(path)
			if err != nil {
				b.Fatal(err)
			}
			if err := opened.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("query_1000", func(b *testing.B) {
		opened, err := Open(path)
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = opened.Close() }()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			found, err := opened.Search(context.Background(), Filter{Query: "searchable marker", Limit: 1_000})
			if err != nil {
				b.Fatal(err)
			}
			if len(found) != 1_000 {
				b.Fatalf("records = %d", len(found))
			}
		}
	})

	b.Run("warm_refresh_1000", func(b *testing.B) {
		opened, err := Open(path)
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = opened.Close() }()
		index, err := NewIndex(opened, staticPerformanceSource{records: records})
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			result, err := index.Refresh(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if len(result.Sources) != 1 || result.Sources[0].Unchanged != 1_000 {
				b.Fatalf("unexpected refresh: %+v", result)
			}
		}
	})
}

type staticPerformanceSource struct {
	records []Record
}

func (source staticPerformanceSource) Name() string { return AgentClaude }

func (source staticPerformanceSource) Scan(ctx context.Context) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	return ScanResult{Records: append([]Record(nil), source.records...)}, nil
}

func syntheticPerformanceRecords(count int) []Record {
	records := make([]Record, count)
	for index := range records {
		id := fmt.Sprintf("controlled-%04d", index)
		records[index] = Record{
			Key:           AgentClaude + ":" + id,
			ID:            id,
			Agent:         AgentClaude,
			Title:         "controlled synthetic session " + id,
			Project:       "controlled-performance",
			Workspace:     "/tmp/controlled-performance",
			Branch:        "main",
			UpdatedAt:     time.Unix(1_800_000_000-int64(index), 0).UTC(),
			SizeBytes:     256,
			MessageCount:  2,
			PromptPreview: "controlled synthetic preview",
			CanResume:     true,
			CanFork:       true,
			SourcePath:    filepath.Join("/tmp", "controlled", id+".jsonl"),
			SourceModTime: int64(index + 1),
			SourceSize:    256,
			SearchText:    "controlled synthetic searchable marker",
		}
	}
	return records
}
