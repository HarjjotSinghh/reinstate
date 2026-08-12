# Gemini `$rewindTo` handoff fixture

JSONL is append-only. `$rewindTo` does not delete prior lines; it appends a
marker. Capsule parse replays the marker and truncates **from and including**
the target id (vendor `ChatRecordingService.rewindTo`), so rewound turns never
reach the capsule.

`session-rewind.jsonl` rewinds to `model-1`, so only `user-1` survives.
`session-rewind-unknown.jsonl` targets a missing id (no-op + warning).
