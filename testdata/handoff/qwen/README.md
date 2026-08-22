# Qwen Code handoff fixtures

Synthetic. No real transcripts, credentials, or private paths.

Layout matches the vendor's own store: `projects/<sanitized-cwd>/chats/<uuid>.jsonl`.
The project directory is `cwd` with every non-alphanumeric byte replaced by `-`
(lower-cased first on Windows), so `/Users/fixture-user/code/demo` becomes
`-Users-fixture-user-code-demo`.

## Record shape

Top-level keys match Claude Code (`uuid`, `parentUuid`, `sessionId`, `cwd`,
`type`, `timestamp`, `version`). The **message body does not**: Qwen carries a
Gemini `Content` value, `{"role":"user"|"model","parts":[…]}`, where a part is
`{"text":…}`, `{"functionCall":{id,name,args}}`, or
`{"functionResponse":{id,name,response}}`. `type` is one of `user`,
`assistant`, `tool_result`, `system`; `system` records carry a `subtype` and a
`systemPayload` instead of a message.

## Cases

| Directory | What it covers |
| --------- | -------------- |
| `basic/` | text, one successful tool call/result pair, one failing pair, `ui_telemetry` and `custom_title` system records |
| `rewound/` | a `/rewind` that re-roots `parentUuid`, leaving two `DEAD_BRANCH_*` records off the live chain |
| `partial-final-record/` | a torn trailing record with no newline |
| `unknown-records/` | an unknown `type`, an unknown system `subtype`, an unknown part type, and a record with neither `uuid` nor `type` |

`rewound/` is the case that matters most. Qwen does not write a `$rewindTo`
marker the way Gemini does — it re-points `lastRecordUuid` at the record before
the rewound turn and appends a `subtype:"rewind"` system record there, so the
discarded turns stay on disk on a dead branch of the uuid tree. A reader that
walks the file line by line would replay turns the user explicitly threw away.
The live conversation is the `parentUuid` chain walked back from the last
record.
