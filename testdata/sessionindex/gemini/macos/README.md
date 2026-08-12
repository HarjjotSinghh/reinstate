# Gemini CLI `$rewindTo` fixture

JSONL is append-only. `$rewindTo` does **not** delete prior lines from the
file; it appends a marker. Active conversation projection truncates at that
marker (vendor `ChatRecordingService.rewindTo` removes from the target id
onward in memory and calls `appendRecord({ $rewindTo })`).

See `docs/cli/rewind.md` and
`packages/core/src/services/chatRecordingService.ts` in google-gemini/gemini-cli.
