package transcript

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestSnapshotJSONLPartialTrailingRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	complete := []byte("{\"type\":\"user\",\"text\":\"hello\"}\n{\"type\":\"assistant\",\"text\":\"hi\"}\n")
	partial := []byte("{\"type\":\"assistant\",\"text\":\"truncat")
	if err := os.WriteFile(path, append(append([]byte{}, complete...), partial...), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	b, err := SnapshotJSONL(path, "claude", "sess-1", 0)
	if err != nil {
		t.Fatalf("SnapshotJSONL: %v", err)
	}
	if !b.Partial {
		t.Fatalf("Partial = false, want true")
	}
	if b.ByteOffset != int64(len(complete)) {
		t.Fatalf("ByteOffset = %d, want %d", b.ByteOffset, len(complete))
	}
	if b.SizeBytes != int64(len(complete)+len(partial)) {
		t.Fatalf("SizeBytes = %d, want %d", b.SizeBytes, len(complete)+len(partial))
	}
	if b.Path() != path {
		t.Fatalf("Path = %q, want %q", b.Path(), path)
	}

	// Frozen prefix must parse cleanly and never surface the partial record.
	var lines [][]byte
	warnings, err := VisitCompleteJSONL(b, 0, func(_ int, line []byte) error {
		lines = append(lines, append([]byte(nil), line...))
		return nil
	})
	if err != nil {
		t.Fatalf("VisitCompleteJSONL: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d complete lines, want 2", len(lines))
	}
	for i, line := range lines {
		if bytes.Contains(line, []byte("truncat")) {
			t.Fatalf("line %d surfaced partial content: %s", i, line)
		}
	}
}

func TestSnapshotJSONLDigestCoversExactPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	body := []byte("{\"a\":1}\n{\"b\":2}\n{\"partial\":")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	b, err := SnapshotJSONL(path, "codex", "sess-2", 0)
	if err != nil {
		t.Fatalf("SnapshotJSONL: %v", err)
	}

	wantOffset := int64(len("{\"a\":1}\n{\"b\":2}\n"))
	if b.ByteOffset != wantOffset {
		t.Fatalf("ByteOffset = %d, want %d", b.ByteOffset, wantOffset)
	}

	sum := sha256.Sum256(body[:wantOffset])
	wantDigest := hex.EncodeToString(sum[:])
	if b.SHA256 != wantDigest {
		t.Fatalf("SHA256 = %q, want %q", b.SHA256, wantDigest)
	}

	recomputed, err := DigestPrefix(b)
	if err != nil {
		t.Fatalf("DigestPrefix: %v", err)
	}
	if recomputed != wantDigest {
		t.Fatalf("DigestPrefix = %q, want %q", recomputed, wantDigest)
	}

	reader, err := PrefixReader(b)
	if err != nil {
		t.Fatalf("PrefixReader: %v", err)
	}
	defer func() { _ = reader.Close() }()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll prefix: %v", err)
	}
	if !bytes.Equal(got, body[:wantOffset]) {
		t.Fatalf("prefix bytes mismatch")
	}
}

func TestSnapshotJSONLAppendDoesNotChangeFrozenBoundary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	initial := []byte("{\"type\":\"user\",\"text\":\"one\"}\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	b, err := SnapshotJSONL(path, "gemini", "sess-3", 0)
	if err != nil {
		t.Fatalf("SnapshotJSONL: %v", err)
	}
	frozen := b

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.Write([]byte("{\"type\":\"user\",\"text\":\"two\"}\n{\"partial\"")); err != nil {
		_ = f.Close()
		t.Fatalf("append: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close append: %v", err)
	}

	// The returned Boundary value is immutable; digest of the frozen prefix is stable.
	if b.ByteOffset != frozen.ByteOffset || b.SHA256 != frozen.SHA256 || b.Partial != frozen.Partial {
		t.Fatalf("boundary mutated after append: %+v vs %+v", b, frozen)
	}
	digest, err := DigestPrefix(frozen)
	if err != nil {
		t.Fatalf("DigestPrefix after append: %v", err)
	}
	if digest != frozen.SHA256 {
		t.Fatalf("frozen digest changed after append: got %q want %q", digest, frozen.SHA256)
	}

	// Snapshot of a copy taken at freeze time also matches.
	copyPath := filepath.Join(dir, "copy.jsonl")
	if err := os.WriteFile(copyPath, initial, 0o600); err != nil {
		t.Fatalf("write copy: %v", err)
	}
	copyBoundary, err := SnapshotJSONL(copyPath, "gemini", "sess-3", 0)
	if err != nil {
		t.Fatalf("SnapshotJSONL copy: %v", err)
	}
	if copyBoundary.ByteOffset != frozen.ByteOffset || copyBoundary.SHA256 != frozen.SHA256 {
		t.Fatalf("copy boundary mismatch: %+v vs %+v", copyBoundary, frozen)
	}
}

func TestRegistryRejectsDuplicateAndEmptyNames(t *testing.T) {
	name := "test-reader-" + t.Name()
	r := &stubReader{name: name}

	if err := Register(r); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := Register(r); err == nil {
		t.Fatal("second Register succeeded, want duplicate error")
	}
	if err := Register(&stubReader{name: ""}); err == nil {
		t.Fatal("Register empty name succeeded, want error")
	}
	if err := Register(nil); err == nil {
		t.Fatal("Register nil succeeded, want error")
	}

	got, ok := Get(name)
	if !ok || got.Name() != name {
		t.Fatalf("Get(%q) = (%v, %v)", name, got, ok)
	}
	if _, ok := Get("missing-" + t.Name()); ok {
		t.Fatal("Get missing reader unexpectedly ok")
	}
}

func TestNormalizeHelpers(t *testing.T) {
	t.Parallel()

	if got := NormalizeActor("Human"); got != capsule.ActorUser {
		t.Fatalf("NormalizeActor = %q, want user", got)
	}
	if got := NormalizeKind("tool_use"); got != capsule.KindToolCall {
		t.Fatalf("NormalizeKind = %q, want tool_call", got)
	}

	text := TextBlock("hello")
	if text.Type != capsule.BlockTypeText || text.Text != "hello" || text.Size != 5 {
		t.Fatalf("TextBlock = %+v", text)
	}
	ref := RefBlock("sidecar:1")
	if ref.Type != capsule.BlockTypeRef || ref.Ref != "sidecar:1" {
		t.Fatalf("RefBlock = %+v", ref)
	}

	actor, kind, port, reason := ClassifyUnknown("weird")
	if actor != capsule.ActorUnknown || kind != capsule.KindUnknown || port != capsule.PortabilityReferenced || reason == "" {
		t.Fatalf("ClassifyUnknown = %q %q %q %q", actor, kind, port, reason)
	}

	events := LinkToolResults([]capsule.Event{
		{Kind: capsule.KindToolCall, CallID: "c1"},
		{Kind: capsule.KindToolCall, CallID: "c2"},
		{Kind: capsule.KindToolResult},
		{Kind: capsule.KindToolResult},
	})
	if events[2].LinkedCallID != "c1" || events[3].LinkedCallID != "c2" {
		t.Fatalf("LinkToolResults = %+v", events)
	}

	long := TextBlock(stringsRepeat("x", 20))
	truncated := TruncateBlock(long, 12)
	if !bytes.HasSuffix([]byte(truncated.Text), []byte(TruncationMarker)) {
		t.Fatalf("TruncateBlock missing marker: %q", truncated.Text)
	}
	if len(truncated.Text) != 12 {
		t.Fatalf("TruncateBlock len = %d, want 12", len(truncated.Text))
	}
	unchanged := TruncateBlock(TextBlock("short"), 100)
	if unchanged.Text != "short" {
		t.Fatalf("TruncateBlock altered short text: %q", unchanged.Text)
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}

type stubReader struct {
	name string
}

func (s *stubReader) Name() string { return s.name }

func (s *stubReader) Probe(context.Context, sessionindex.Record) (Compatibility, error) {
	return CompatibilitySupported, nil
}

func (s *stubReader) Snapshot(context.Context, sessionindex.Record) (Boundary, error) {
	return Boundary{}, nil
}

func (s *stubReader) Parse(context.Context, Boundary) ([]capsule.Event, ParseReport, error) {
	return nil, ParseReport{}, nil
}
