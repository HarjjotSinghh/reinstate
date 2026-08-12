package handoff

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
)

// Fixed checklist IDs the destination must restate before mutation.
// Order is part of the contract and must stay stable.
var acknowledgementRequirementIDs = []string{
	"goal",
	"latest_request",
	"changed_files",
	"tests",
	"missing_caps",
	"next_action",
}

// Acknowledgement is the destination-agent checklist plus any recorded
// confirmation from rein handoff inspect.
//
// rc.1 enforces this at the prompt level only — Reinstate cannot police
// another agent's loop. `rein handoff inspect <id> --acknowledged` /
// `--not-acknowledged` records the user's answer so the metric stays honest.
// Confirmed stays nil until RecordAcknowledgement sets it explicitly; it never
// defaults to true.
type Acknowledgement struct {
	Required   []string  // goal | latest_request | changed_files | tests | missing_caps | next_action
	Confirmed  *bool     // nil = not recorded
	RecordedAt time.Time // zero until Confirmed is set
}

// AcknowledgementRequirements returns the deterministic checklist IDs the
// destination must restate before mutation. Order is stable across runs and
// OSes. The capsule is accepted for call-site symmetry; rc.1 always returns
// the full fixed set (prompt-level contract, not a derived filter).
func AcknowledgementRequirements(c capsule.Capsule) []string {
	_ = c
	out := make([]string, len(acknowledgementRequirementIDs))
	copy(out, acknowledgementRequirementIDs)
	return out
}

// GetAcknowledgement loads the latest acknowledgement state for handoffID.
// Confirmed remains nil until RecordAcknowledgement is called.
func GetAcknowledgement(store *Store, handoffID string) (Acknowledgement, error) {
	var zero Acknowledgement
	if store == nil || store.root == "" {
		return zero, errors.New("handoff store is nil")
	}
	if err := validateHandoffID(handoffID); err != nil {
		return zero, err
	}
	c, _, err := store.Get(handoffID)
	if err != nil {
		return zero, err
	}
	ack := Acknowledgement{Required: AcknowledgementRequirements(c)}

	entries, err := store.List(maxListLimit)
	if err != nil {
		return zero, err
	}
	// List is newest-first. Prefer the newest entry that explicitly recorded
	// acknowledgement so Confirmed never flips back to nil from a later null.
	for _, e := range entries {
		if e.HandoffID != handoffID || e.Acknowledged == nil {
			continue
		}
		v := *e.Acknowledged
		ack.Confirmed = &v
		ack.RecordedAt = e.CreatedAt.UTC()
		return ack, nil
	}
	return ack, nil
}

// RecordAcknowledgement appends a lineage update with the user's confirmation.
// Confirmed is never defaulted to true. Recording the same value twice is
// idempotent (no additional lineage line).
func RecordAcknowledgement(store *Store, handoffID string, confirmed bool) error {
	if store == nil || store.root == "" {
		return errors.New("handoff store is nil")
	}
	if err := validateHandoffID(handoffID); err != nil {
		return err
	}
	c, _, err := store.Get(handoffID)
	if err != nil {
		return err
	}

	current, err := GetAcknowledgement(store, handoffID)
	if err != nil {
		return err
	}
	if current.Confirmed != nil && *current.Confirmed == confirmed {
		return nil
	}

	entry, ok, err := latestLineageEntry(store, handoffID)
	if err != nil {
		return err
	}
	if !ok {
		entry = LineageEntry{
			HandoffID:   handoffID,
			LineageRoot: strings.TrimSpace(c.Identity.LineageRoot),
			Source: LineageEndpoint{
				Agent:          c.Identity.Parent.Agent,
				SessionID:      c.Identity.Parent.SessionID,
				ArtifactSHA256: c.Identity.Parent.ArtifactSHA256,
			},
			Policy:          c.Projection.Policy,
			FidelityOverall: string(c.Fidelity.Overall),
		}
		if entry.LineageRoot == "" {
			entry.LineageRoot = handoffID
		}
	}
	v := confirmed
	entry.Acknowledged = &v
	entry.HandoffID = handoffID
	entry.CreatedAt = time.Now().UTC().Truncate(time.Second)
	if err := store.AppendLineage(entry); err != nil {
		return fmt.Errorf("record acknowledgement: %w", err)
	}
	return nil
}

func latestLineageEntry(store *Store, handoffID string) (LineageEntry, bool, error) {
	entries, err := store.List(maxListLimit)
	if err != nil {
		return LineageEntry{}, false, err
	}
	for _, e := range entries {
		if e.HandoffID == handoffID {
			return e, true, nil
		}
	}
	return LineageEntry{}, false, nil
}
