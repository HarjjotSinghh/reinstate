package sessionindex

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodexForkKeepsItsOwnIdentity covers the two rollout shapes Codex produces
// for `codex fork`. Both replay the source's records, so a fork's session_meta
// may carry the source id either alongside or instead of its own. Attributing a
// fork to the source made it unaddressable and merged its turns into the
// session it was forked from.
func TestCodexForkKeepsItsOwnIdentity(t *testing.T) {
	t.Parallel()

	const (
		sourceID = "00000000-0000-4000-8000-00000000a001"
		forkAID  = "00000000-0000-4000-8000-00000000b001"
		forkBID  = "00000000-0000-4000-8000-00000000c001"
	)

	source := NewCodexSource(filepath.Join(
		"..", "..", "testdata", "sessionindex", "codex", "forks",
	))
	scan, err := source.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	records, _ := CoalesceRecords(scan.Records)
	byID := make(map[string]Record, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}

	if len(records) != 3 {
		t.Fatalf("records after coalescing = %d, want 3 (source + two forks); ids=%v",
			len(records), keysOf(byID))
	}
	for _, want := range []string{sourceID, forkAID, forkBID} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("record %q missing; ids=%v", want, keysOf(byID))
		}
	}

	// A fork's turns must not leak into the record it was forked from.
	sourceText := strings.ToUpper(byID[sourceID].PromptPreview + " " + byID[sourceID].SearchText)
	for _, leaked := range []string{"FORK-A-ONLY-PROMPT", "FORK-B-ONLY-PROMPT"} {
		if strings.Contains(sourceText, leaked) {
			t.Fatalf("source record absorbed fork content %q", leaked)
		}
	}

	// Each fork must still be independently resumable and forkable.
	for _, id := range []string{forkAID, forkBID} {
		record := byID[id]
		if !record.CanResume || !record.CanFork {
			t.Fatalf("fork %q lost native capabilities: resume=%v fork=%v",
				id, record.CanResume, record.CanFork)
		}
		if record.Workspace == "" {
			t.Fatalf("fork %q lost its recorded workspace", id)
		}
	}
}

func TestCodexSessionIDFromFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "rollout name ending in a uuid",
			path: "rollout-2026-01-02T03-04-05-00000000-0000-4000-8000-00000000a001.jsonl",
			want: "00000000-0000-4000-8000-00000000a001",
		},
		{
			name: "synthetic fixture name without a uuid",
			path: "rollout-syn-001.jsonl",
			want: "",
		},
		{
			name: "uuid-shaped but non-hexadecimal tail",
			path: "rollout-2026-01-02T03-04-05-zzzzzzzz-0000-4000-8000-00000000a001.jsonl",
			want: "",
		},
		{
			name: "too few hyphen separated fields",
			path: "rollout.jsonl",
			want: "",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := codexSessionIDFromFilename(test.path); got != test.want {
				t.Fatalf("codexSessionIDFromFilename(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func keysOf(records map[string]Record) []string {
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	return ids
}
