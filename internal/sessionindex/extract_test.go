package sessionindex

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

func TestSafeTextRemovesTerminalControlsAndCollapsesWhitespace(t *testing.T) {
	t.Parallel()

	input := "\x1b[31mred\x1b[0m\t two\n\nwords \x1b]0;title\a after\u202esecret"
	if got, want := SafeText(input, 0), "red two words aftersecret"; got != want {
		t.Fatalf("SafeText() = %q, want %q", got, want)
	}
	if got := SafePreview(strings.Repeat("界", PromptPreviewRunes+20)); utf8.RuneCountInString(got) != PromptPreviewRunes {
		t.Fatalf("preview rune count = %d", utf8.RuneCountInString(got))
	}
}

func TestBuildSearchTextIsBoundedAndValidUTF8(t *testing.T) {
	t.Parallel()

	got := BuildSearchText(strings.Repeat("🙂", MaxSearchTextBytes))
	if len(got) > MaxSearchTextBytes {
		t.Fatalf("search text bytes = %d", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("bounded search text is invalid UTF-8")
	}
}

func TestCoalesceRecordsMergesNativeSessionSegmentsDeterministically(t *testing.T) {
	t.Parallel()
	old := testRecord(
		AgentCodex,
		"shared-id",
		time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
		"/sessions/old.jsonl",
		1,
	)
	old.Title = "Useful vendor title"
	old.PromptPreview = "first user prompt"
	old.SearchText = "old searchable marker"
	old.Files = []string{"old.go"}
	old.MessageCount = 2
	old.RecordedEnvironment = environment.RecordedEnvironment{
		RepositoryID: environment.RecordedField{
			Value:      "https://github.com/example/demo.git",
			Provenance: "codex.session_meta.git.repository_url",
		},
		GitHead: environment.RecordedField{
			Value:      "0123456789abcdef0123456789abcdef01234567",
			Provenance: "codex.session_meta.git.commit_hash",
		},
		Requirements: []environment.Requirement{{
			Kind: "mcp", Name: "github", Provenance: "codex.session_meta.mcp",
		}},
	}
	newer := testRecord(
		AgentCodex,
		"shared-id",
		time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC),
		"/sessions/new.jsonl",
		2,
	)
	newer.Title = newer.ID
	newer.SearchText = "new searchable marker"
	newer.Files = []string{"new.go"}
	newer.MessageCount = 3
	newer.RecordedEnvironment = environment.RecordedEnvironment{
		Branch: environment.RecordedField{
			Value:      "phase-three",
			Provenance: "codex.session_meta.git.branch",
		},
		Requirements: []environment.Requirement{{
			Kind: "mcp", Name: "github", Provenance: "codex.session_meta.mcp",
		}},
	}

	records, warnings := CoalesceRecords([]Record{newer, old})
	if len(records) != 1 || len(warnings) != 1 {
		t.Fatalf("records/warnings = %d/%d", len(records), len(warnings))
	}
	record := records[0]
	if record.Reference() != "codex:shared-id" ||
		record.Title != "Useful vendor title" ||
		record.PromptPreview != "first user prompt" ||
		record.MessageCount != 5 ||
		record.SourcePath != "/sessions/new.jsonl" {
		t.Fatalf("coalesced record = %+v", record)
	}
	for _, expected := range []string{"old searchable marker", "new searchable marker"} {
		if !strings.Contains(record.SearchText, expected) {
			t.Fatalf("search text %q is missing %q", record.SearchText, expected)
		}
	}
	if got := strings.Join(record.Files, ","); got != "new.go,old.go" {
		t.Fatalf("files = %q", got)
	}
	if record.RecordedEnvironment.RepositoryID.Value != environment.NormalizeRepositoryID("https://github.com/example/demo.git") ||
		record.RecordedEnvironment.Branch.Value != "phase-three" ||
		record.RecordedEnvironment.GitHead.Value != "0123456789abcdef0123456789abcdef01234567" ||
		len(record.RecordedEnvironment.Requirements) != 1 {
		t.Fatalf("coalesced recorded environment = %+v", record.RecordedEnvironment)
	}
	if warnings[0].Code != "coalesced_session_segments" {
		t.Fatalf("warning = %+v", warnings[0])
	}

	reversed, _ := CoalesceRecords([]Record{old, newer})
	if len(reversed) != 1 || reversed[0].SourcePath != record.SourcePath ||
		reversed[0].SourceModTime != record.SourceModTime {
		t.Fatalf("aggregate fingerprint is not deterministic: %+v / %+v", record, reversed)
	}

	changedOld := old
	changedOld.SourceModTime++
	changed, _ := CoalesceRecords([]Record{changedOld, newer})
	if changed[0].SourceModTime == record.SourceModTime {
		t.Fatal("aggregate freshness token did not change with an older segment")
	}
}

func TestScanJSONLinesBoundsAndToleratesConcurrentTail(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"id":1}`,
		`not-json`,
		`{"oversized":"` + strings.Repeat("x", 64) + `"}`,
		`{"id":2}`,
		`{"unfinished":`,
	}, "\n")
	var visited []string
	warnings, err := ScanJSONLines(strings.NewReader(input), 32, func(_ int, line []byte) error {
		visited = append(visited, string(line))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(visited, ","), `{"id":1},{"id":2}`; got != want {
		t.Fatalf("visited = %q, want %q", got, want)
	}
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	if got, want := strings.Join(codes, ","), "malformed_record,oversized_record,incomplete_trailing_record"; got != want {
		t.Fatalf("warning codes = %q, want %q", got, want)
	}
}

func TestScanJSONLinesAcceptsCompleteFinalValueWithoutNewline(t *testing.T) {
	t.Parallel()

	visited := 0
	warnings, err := ScanJSONLines(strings.NewReader(`{"complete":true}`), 1024, func(_ int, _ []byte) error {
		visited++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || visited != 1 {
		t.Fatalf("warnings=%v visited=%d", warnings, visited)
	}
}

func TestScanJSONLinesAcceptsRecordExactlyAtLimit(t *testing.T) {
	t.Parallel()

	line := `{"v":"x"}`
	visited := 0
	warnings, err := ScanJSONLines(strings.NewReader(line+"\n"), len(line), func(_ int, _ []byte) error {
		visited++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || visited != 1 {
		t.Fatalf("warnings=%v visited=%d", warnings, visited)
	}
}

func TestScanJSONLinesPropagatesVisitorFailure(t *testing.T) {
	t.Parallel()

	want := fmt.Errorf("stop")
	_, err := ScanJSONLines(strings.NewReader("{}\n"), 1024, func(_ int, _ []byte) error {
		return want
	})
	if err != want {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
