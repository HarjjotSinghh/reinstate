package handoff

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAcknowledgementRequirementsDeterministic(t *testing.T) {
	t.Parallel()
	want := []string{
		"goal",
		"latest_request",
		"changed_files",
		"tests",
		"missing_caps",
		"next_action",
	}
	c := testCapsule("ackreq00112233445566778899aabbcc")
	first := AcknowledgementRequirements(c)
	second := AcknowledgementRequirements(c)
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("AcknowledgementRequirements = %#v, want %#v", first, want)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("AcknowledgementRequirements not stable: %#v vs %#v", first, second)
	}
	// Mutating the returned slice must not affect a later call.
	first[0] = "mutated"
	third := AcknowledgementRequirements(c)
	if !reflect.DeepEqual(third, want) {
		t.Fatalf("AcknowledgementRequirements shared backing: %#v", third)
	}
}

func TestUnrecordedAcknowledgementStaysNil(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := "acknil00112233445566778899aabbcc"
	c := testCapsule(id)
	if _, err := store.Put(c, Artifacts{
		ProjectionMD: []byte("projection"),
		Bootstrap:    []byte("boot"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendLineage(LineageEntry{
		HandoffID:   id,
		LineageRoot: id,
		CreatedAt:   time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC),
		Source:      LineageEndpoint{Agent: "claude", SessionID: "s1"},
		Destination: LineageEndpoint{Agent: "codex", State: "resolved"},
		Policy:      "balanced",
		Launched:    true,
		// Acknowledged intentionally omitted: must stay nil, never default true.
	}); err != nil {
		t.Fatal(err)
	}

	ack, err := GetAcknowledgement(store, id)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Confirmed != nil {
		t.Fatalf("Confirmed = %v, want nil (unrecorded)", *ack.Confirmed)
	}
	if !ack.RecordedAt.IsZero() {
		t.Fatalf("RecordedAt = %v, want zero", ack.RecordedAt)
	}
	if !reflect.DeepEqual(ack.Required, AcknowledgementRequirements(c)) {
		t.Fatalf("Required = %#v", ack.Required)
	}

	entries, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Acknowledged != nil {
		t.Fatalf("lineage Acknowledged = %v, want nil", entries[0].Acknowledged)
	}
}

func TestRecordAcknowledgementIdempotent(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := "ackdup00112233445566778899aabbcc"
	c := testCapsule(id)
	if _, err := store.Put(c, Artifacts{
		ProjectionMD: []byte("projection"),
		Bootstrap:    []byte("boot"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendLineage(LineageEntry{
		HandoffID:   id,
		LineageRoot: id,
		CreatedAt:   time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC),
		Source:      LineageEndpoint{Agent: "claude", SessionID: "s1"},
		Destination: LineageEndpoint{Agent: "codex", State: "resolved"},
		Policy:      "balanced",
		Launched:    true,
	}); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(filepath.Join(store.Root(), lineageFileName))
	if err != nil {
		t.Fatal(err)
	}

	if err := RecordAcknowledgement(store, id, true); err != nil {
		t.Fatal(err)
	}
	ack, err := GetAcknowledgement(store, id)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Confirmed == nil || !*ack.Confirmed {
		t.Fatalf("Confirmed = %v, want true", ack.Confirmed)
	}
	if ack.RecordedAt.IsZero() {
		t.Fatal("RecordedAt is zero after record")
	}

	mid, err := os.ReadFile(filepath.Join(store.Root(), lineageFileName))
	if err != nil {
		t.Fatal(err)
	}
	if len(mid) <= len(before) {
		t.Fatal("first RecordAcknowledgement did not append lineage")
	}

	if err := RecordAcknowledgement(store, id, true); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(store.Root(), lineageFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(mid) {
		t.Fatal("second RecordAcknowledgement mutated lineage; want idempotent no-op")
	}

	ack2, err := GetAcknowledgement(store, id)
	if err != nil {
		t.Fatal(err)
	}
	if ack2.Confirmed == nil || !*ack2.Confirmed {
		t.Fatalf("Confirmed after idempotent record = %v, want true", ack2.Confirmed)
	}
}

func TestRecordAcknowledgementFalseDoesNotDefaultTrue(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := "ackfal00112233445566778899aabbcc"
	if _, err := store.Put(testCapsule(id), Artifacts{
		ProjectionMD: []byte("projection"),
		Bootstrap:    []byte("boot"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := RecordAcknowledgement(store, id, false); err != nil {
		t.Fatal(err)
	}
	ack, err := GetAcknowledgement(store, id)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Confirmed == nil {
		t.Fatal("Confirmed is nil, want false")
	}
	if *ack.Confirmed {
		t.Fatal("Confirmed defaulted to true; want false")
	}
}
