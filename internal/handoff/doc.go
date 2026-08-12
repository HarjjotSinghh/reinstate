// Package handoff implements portable cross-agent session handoffs.
//
// BindWorkspace converts Phase 3 preflight truth into capsule workspace state.
// DeriveCheckpoint builds a deterministic task checkpoint from canonical events
// and live Git porcelain with no model or network calls. Store persists capsules
// and append-only lineage under $REINSTATE_HOME/handoffs (local-only in v0.4.0).
// RenderBootstrap / RenderProjection / RenderJSON emit the destination briefing
// as a structured handoff (never native resume / lossless).
// ClaudeTarget and CodexTarget launch new destination sessions via vendor CLI
// argv (no vendor-internal writes); Codex reconciles its session ID after launch.
package handoff
