// Package handoff implements portable cross-agent session handoffs.
//
// BindWorkspace converts Phase 3 preflight truth into capsule workspace state.
// DeriveCheckpoint builds a deterministic task checkpoint from canonical events
// and live Git porcelain with no model or network calls. Store persists capsules
// and append-only lineage under $REINSTATE_HOME/handoffs (local-only in v0.4.0).
// CodexTarget launches a new Codex session via argv bootstrap and reconciles
// the vendor-assigned session ID after launch without writing ~/.codex.
package handoff
