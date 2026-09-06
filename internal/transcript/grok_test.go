package transcript

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestGrokReaderParsesBasicAndCompactedFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		root           string
		wantSession    string
		wantUserSubstr string
		wantCompacted  bool
	}{
		{
			name:           "handoff basic",
			root:           filepath.Join("..", "..", "testdata", "handoff", "grok", "basic"),
			wantSession:    "01987654-basic-0000-0000-000000000001",
			wantUserSubstr: "Basic Grok handoff user prompt",
		},
		{
			name:           "handoff compacted",
			root:           filepath.Join("..", "..", "testdata", "handoff", "grok", "compacted"),
			wantSession:    "01987654-3210-7890-abcd-ef0123456789",
			wantUserSubstr: "Post-compaction continuation prompt",
			wantCompacted:  true,
		},
		{
			name:           "sessionindex macos",
			root:           filepath.Join("..", "..", "testdata", "sessionindex", "grok", "macos"),
			wantSession:    "01987654-3210-7890-abcd-ef0123456789",
			wantUserSubstr: "Post-compaction continuation prompt",
			wantCompacted:  true,
		},
	}

	reader := NewGrokReader()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := sessionindex.NewGrokSource(test.root).Scan(context.Background())
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			compat, err := reader.Probe(context.Background(), record)
			if err != nil {
				t.Fatalf("Probe() error = %v", err)
			}
			if compat != CompatibilitySupported {
				t.Fatalf("Probe() = %q, want supported", compat)
			}
			boundary, err := reader.Snapshot(context.Background(), record)
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if boundary.SessionID != test.wantSession {
				t.Fatalf("boundary session = %q, want %q", boundary.SessionID, test.wantSession)
			}
			events, report, err := reader.Parse(context.Background(), boundary)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if report.Events != len(events) || len(events) == 0 {
				t.Fatalf("events = %d report=%d", len(events), report.Events)
			}
			foundUser := false
			foundPreCompact := false
			foundCheckpoint := false
			for _, ev := range events {
				if ev.Kind == capsule.KindMessage && ev.Actor == capsule.ActorUser {
					for _, block := range ev.Blocks {
						if strings.Contains(block.Text, test.wantUserSubstr) {
							foundUser = true
						}
						if strings.Contains(block.Text, "PRE_COMPACT_USER_TURN") {
							foundPreCompact = true
						}
					}
				}
				if ev.Kind == capsule.KindCheckpoint {
					foundCheckpoint = true
				}
				if ev.ID == "" || ev.ContentHash == "" {
					t.Fatalf("event missing id/hash: %#v", ev)
				}
				if ev.Portability != capsule.PortabilityExact && ev.Reason == "" {
					t.Fatalf("non-exact event missing reason: %#v", ev)
				}
			}
			if !foundUser {
				t.Fatalf("missing user prompt %q in events", test.wantUserSubstr)
			}
			if test.wantCompacted && !foundPreCompact {
				t.Fatal("compacted fixture missing pre-compact user turn")
			}
			if test.wantCompacted && !foundCheckpoint {
				t.Fatal("compacted fixture missing compaction checkpoint event")
			}

			again, _, err := reader.Parse(context.Background(), boundary)
			if err != nil {
				t.Fatalf("second Parse() error = %v", err)
			}
			if len(again) != len(events) {
				t.Fatalf("second parse event count = %d, want %d", len(again), len(events))
			}
			for i := range events {
				if again[i].ID != events[i].ID || again[i].ContentHash != events[i].ContentHash {
					t.Fatalf("parse not deterministic at %d", i)
				}
			}
		})
	}
}

// TestGrokReaderPartialFinalRecordExcluded covers grok:D4: updates.jsonl (the
// boundary authority) truncated mid-record, on both a macOS-shaped and a
// native-Windows-shaped workspace, with the boundary offset and the SHA-256
// of the bytes before it cross-checked against an independent computation.
func TestGrokReaderPartialFinalRecordExcluded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		root        string
		wantSession string
	}{
		{
			name:        "macos",
			root:        filepath.Join("..", "..", "testdata", "handoff", "grok", "partial-final-record"),
			wantSession: "01987654-pf0m-0000-0000-000000000001",
		},
		{
			name:        "windows",
			root:        filepath.Join("..", "..", "testdata", "handoff", "grok", "partial-final-record-windows"),
			wantSession: "01987654-pf0w-0000-0000-000000000002",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := sessionindex.NewGrokSource(tc.root).Scan(context.Background())
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if len(result.Records) != 1 {
				t.Fatalf("records = %d, want 1", len(result.Records))
			}
			record := result.Records[0]
			if record.ID != tc.wantSession {
				t.Fatalf("record ID = %q, want %q", record.ID, tc.wantSession)
			}

			reader := NewGrokReader()
			boundary, err := reader.Snapshot(context.Background(), record)
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if !boundary.Partial {
				t.Fatal("Partial = false, want true")
			}
			if !strings.HasSuffix(boundary.Path(), "updates.jsonl") {
				t.Fatalf("boundary path = %q, want the updates.jsonl authority file", boundary.Path())
			}

			raw, err := os.ReadFile(boundary.Path())
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			wantOffset, wantDigest := expectedJSONLBoundary(t, raw)
			if boundary.ByteOffset != wantOffset {
				t.Fatalf("ByteOffset = %d, want %d (independently computed)", boundary.ByteOffset, wantOffset)
			}
			if boundary.SHA256 != wantDigest {
				t.Fatalf("SHA256 = %q, want %q (independently computed)", boundary.SHA256, wantDigest)
			}
			if recomputed, err := DigestPrefix(boundary); err != nil || recomputed != wantDigest {
				t.Fatalf("DigestPrefix = %q, err=%v, want %q", recomputed, err, wantDigest)
			}

			events, _, err := reader.Parse(context.Background(), boundary)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			for _, ev := range events {
				for _, block := range ev.Blocks {
					if strings.Contains(block.Text, "TRUNCATED_PARTIAL_ONLY") {
						t.Fatalf("partial record surfaced: %+v", ev)
					}
				}
			}
		})
	}
}

func TestGrokForcedSecurityAndNoRedactRefusal(t *testing.T) {
	t.Parallel()
	security := NewGrokReader().ForcedSecurity()
	if security.DestinationWarning != DestinationWarningGrok {
		t.Fatalf("DestinationWarning = %q", security.DestinationWarning)
	}
	if !security.RedactionForced {
		t.Fatal("RedactionForced = false")
	}
	if !security.SourceInstructionsAreUntrustedHistory {
		t.Fatal("SourceInstructionsAreUntrustedHistory = false")
	}
	if err := RefuseNoRedact(sessionindex.AgentGrok); !errors.Is(err, ErrNoRedactRefused) {
		t.Fatalf("RefuseNoRedact(grok) = %v", err)
	}
	if err := RefuseNoRedact("claude"); err != nil {
		t.Fatalf("RefuseNoRedact(claude) = %v", err)
	}
}

func TestGrokReaderRegistered(t *testing.T) {
	t.Parallel()
	reader, ok := Get(sessionindex.AgentGrok)
	if !ok || reader == nil || reader.Name() != sessionindex.AgentGrok {
		t.Fatalf("Get(grok) = %#v, %v", reader, ok)
	}
}
