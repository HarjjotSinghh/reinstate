package sessionindex

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestCompositeReference(t *testing.T) {
	t.Parallel()

	ref := CompositeReference(AgentClaude, "session:child")
	if ref != "claude:session:child" {
		t.Fatalf("reference = %q", ref)
	}
	agent, id, ok := ParseCompositeReference(ref)
	if !ok || agent != AgentClaude || id != "session:child" {
		t.Fatalf("parse = %q, %q, %v", agent, id, ok)
	}
	for _, invalid := range []string{"", "claude", ":id", "claude:"} {
		if _, _, ok := ParseCompositeReference(invalid); ok {
			t.Fatalf("ParseCompositeReference(%q) unexpectedly succeeded", invalid)
		}
	}
}

func TestNormalizeRecordBoundsPrivateContent(t *testing.T) {
	t.Parallel()

	record, err := NormalizeRecord(Record{
		ID:             "session-1",
		Agent:          " CLAUDE ",
		Title:          "\x1b[31mSensitive\x1b[0m\n title",
		Project:        "project",
		Workspace:      "/tmp/project",
		Branch:         "main",
		PromptPreview:  strings.Repeat("🙂", PromptPreviewRunes+5),
		Files:          []string{"z.go", "a.go", "z.go", "\x1b[2Jbad.go"},
		CanFork:        true,
		CanResume:      true,
		SourcePath:     "/tmp/source",
		SourceModTime:  12,
		SourceSize:     20,
		SearchText:     "literal prompt",
		UpdatedAt:      time.Date(2026, 7, 30, 1, 2, 3, 4, time.FixedZone("test", 3600)),
		MessageCount:   2,
		ReadOnlyReason: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Key != "claude:session-1" {
		t.Fatalf("key = %q", record.Key)
	}
	if record.Title != "Sensitive title" {
		t.Fatalf("title = %q", record.Title)
	}
	if utf8.RuneCountInString(record.PromptPreview) != PromptPreviewRunes {
		t.Fatalf("preview runes = %d", utf8.RuneCountInString(record.PromptPreview))
	}
	if got := strings.Join(record.Files, ","); got != "a.go,bad.go,z.go" {
		t.Fatalf("files = %q", got)
	}
	if record.UpdatedAt.Location() != time.UTC {
		t.Fatalf("UpdatedAt location = %v", record.UpdatedAt.Location())
	}
	for _, expected := range []string{"literal prompt", "claude:session-1", "a.go"} {
		if !strings.Contains(record.SearchText, expected) {
			t.Fatalf("search text does not contain %q", expected)
		}
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"SourcePath", "/tmp/source", "literal prompt"} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("JSON leaked private value %q: %s", private, encoded)
		}
	}
}

func TestNormalizeRecordRejectsUnsafeIdentityAndCapabilities(t *testing.T) {
	t.Parallel()

	tests := []Record{
		{ID: "", Agent: AgentClaude},
		{ID: "id", Agent: "bad:agent"},
		{ID: "id", Agent: AgentClaude, Key: "codex:id"},
		{ID: "id", Agent: AgentClaude, CanFork: true},
		{ID: "id", Agent: AgentClaude, SizeBytes: -1},
	}
	for _, record := range tests {
		if _, err := NormalizeRecord(record); err == nil {
			t.Fatalf("NormalizeRecord(%+v) unexpectedly succeeded", record)
		}
	}
}

func TestFilterLimitAndRecordOrdering(t *testing.T) {
	t.Parallel()

	if got := (Filter{}).EffectiveLimit(); got != DefaultLimit {
		t.Fatalf("default limit = %d", got)
	}
	if got := (Filter{Limit: MaxLimit + 1}).EffectiveLimit(); got != MaxLimit {
		t.Fatalf("bounded limit = %d", got)
	}

	same := time.Unix(100, 0)
	records := []Record{
		{Agent: AgentCodex, ID: "b", UpdatedAt: same},
		{Agent: AgentClaude, ID: "z", UpdatedAt: same},
		{Agent: AgentClaude, ID: "a", UpdatedAt: same.Add(time.Second)},
	}
	SortRecords(records)
	got := []string{records[0].Reference(), records[1].Reference(), records[2].Reference()}
	want := []string{"claude:a", "claude:z", "codex:b"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestAmbiguousReferenceErrorSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	err := &AmbiguousReferenceError{
		Reference: "same",
		Matches:   []string{"claude:same", "codex:same"},
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatal("ambiguity does not match ErrAmbiguous")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("ambiguity unexpectedly matches ErrNotFound")
	}
}
