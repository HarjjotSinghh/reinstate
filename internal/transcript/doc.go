// Package transcript defines the shared vendor-transcript Reader contract,
// complete-record boundary snapshot helpers, and normalization utilities used
// by agent-specific readers.
//
// Snapshot opens source files read-only. It never locks, renames, truncates, or
// writes vendor transcripts. A Boundary freezes the byte offset of the last
// newline-terminated parseable JSONL record; any trailing partial line is
// excluded and reported with Partial set.
//
// Agent-specific readers (Claude, Codex, Gemini, OpenCode, Grok) live in sibling
// files and register via Register. This package itself provides only the shared
// contract and helpers.
package transcript
