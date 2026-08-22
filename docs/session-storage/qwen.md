# Qwen Code (Alibaba)

**Confidence: Documented on macOS and native Windows** —
official product identified; both platforms have a real JSONL conversation
with matching first-line keys; T1 index source, T2 transcript reader, T3 native
resume, and T4 handoff destination shipped; macOS journeys recorded, native
Windows journey outstanding.
**Current tier:** T4 (handoff destination) · **Phase 5 target:** T2 (exceeded)

Catalog key is `qwen`.

## Research outcome

Promoted **T0 → T1** on 2026-08-19 (dual-platform probes) and **T1 → T2** on
2026-08-22 (transcript reader). The sections below are kept in the order they
were written; the record shape is settled in
[Record shape (2026-08-22)](#record-shape-2026-08-22), and the earlier
"do not invent a reader" warnings are what that section is the answer to.

The Claude reader is still not reusable. Same-shaped keys are not the same
format.

## Device evidence (2026-08-16, macOS arm64)

Artifact:
[`2026-08-16-macos-qwen.json`](../testing/results/agent-probes/2026-08-16-macos-qwen.json)

| Check | Result |
| ----- | ------ |
| `qwen` on PATH | yes |
| `qwen --version` | `0.21.12` — a bare semver line |
| Resolved root | `~/.qwen` |
| Signed-in session | **no** — sign-up unavailable in the tester's region |
| macOS AGENT-PROBE-V1 | captured, but without a real conversation |
| native Windows AGENT-PROBE-V1 | [`2026-08-17-windows-qwen.json`](../testing/results/agent-probes/2026-08-17-windows-qwen.json) |

```
~/.qwen/
  projects/<slug>/chats/<slug>.json      one file, provenance unknown
  tmp/<64-hex>/logs.json
  tmp/<64-hex>/scheduled_tasks.lock
  extensions/extension-enablement.json
  installation_id, output-language.md, tip_history.json, <slug>.md
```

This **corrects the descriptor's discovery marker from `tmp` to `projects`.**
The Gemini-fork hypothesis predicted conversations under `tmp/<hash>/chats/`,
and `tmp/<64-hex>` does exist — but it holds `logs.json` and
`scheduled_tasks.lock`, not chats. Conversations live under
`projects/<slug>/chats/<slug>.json`, and the project bucket is a **slug, not a
hash**, which is the opposite of Gemini's `ProjectKeyPathHash`.

The single file under `chats/` was not produced by a session the tester ran, so
its record shape is not evidence. `skills/` and `extension-store/` are excluded
in the descriptor: they are configuration, and a populated skills library buries
the real tree in 176 directories of noise.

## Device evidence (2026-08-17, native Windows amd64)

Artifact:
[`2026-08-17-windows-qwen.json`](../testing/results/agent-probes/2026-08-17-windows-qwen.json)

An earlier dump the same day drowned in `updates/**/node_modules` (the
self-updater's npm tree). `updates` is excluded; this artifact is the re-probe
that reached the conversations.

The conversation layout contradicts the macOS reading:

| Aspect | macOS (2026-08-16) | Windows (2026-08-17) |
| ------ | ------------------ | -------------------- |
| Conversation path | `projects/<slug>/chats/<slug>.json` | `projects/<slug>/chats/<uuid-v4>.jsonl` |
| Count | 1 file, provenance unknown | 2 files, from real sessions |
| `qwen --version` | `0.21.12` | `0.21.13` — it self-updated past the pinned install |

The macOS file was **not** produced by a session the tester ran, which is why
that page recorded its shape as unverified. It was right to. The real format is
**JSONL, one file per session, named by UUID** — not a single JSON document.

The record shape is the interesting part:

```
cwd, message, parentUuid, provenance, sessionId, timestamp, type, uuid, version
```

That is Claude Code's transcript schema, near enough to be worth saying out
loud: a `uuid` / `parentUuid` chain, a `sessionId`, a `cwd`, a `message`, and a
`type`. The Gemini-fork hypothesis was rejected on storage-location grounds
already; this suggests the conversation format was taken from somewhere else
again. `provenance` is a field Claude Code does not have.

**This does not make Qwen readable by the Claude reader.** Same-shaped keys are
not the same format, and a reader that assumes otherwise will mis-parse
silently. It is a strong hint for whoever writes the Qwen reader, and nothing
more.

Also worth recording: `usage_record.jsonl` exists with
`durationMs, files, models, project, sessionId, skills, startTime, timestamp,
tools, totalLatencyMs, version` — a per-session usage ledger that could supply
message counts and file references without parsing a transcript at all.

**Promoted to T1 on 2026-08-19.** Dual-platform JSONL conversations exist and
an F1 index source now walks `projects/**/chats/*.jsonl`. macOS also writes
`<uuid-v4>-runtime.json` sidecars that are not conversations. Resume and
fork stay refused. Do not reuse the Claude reader.

## Device evidence (2026-08-17, macOS arm64)

Artifact:
[`2026-08-17-macos-qwen.json`](../testing/results/agent-probes/2026-08-17-macos-qwen.json)

A signed-in session completed on this host (`qwen` 0.21.13). The conversation
file is JSONL. First-line keys match Windows:

```
cwd, message, parentUuid, provenance, sessionId, timestamp, type, uuid, version
```

The probe collapsed three files under `projects/*/chats/*`. `name_shapes`
kept `<uuid-v4>-runtime.json` (one shape per glob); the JSONL schema is in
`first_line_keys`. Sidecars are not the conversation store.

| Check | Result |
| ----- | ------ |
| `qwen` on PATH | yes |
| `qwen --version` | `0.21.13` |
| Resolved root | `~/.qwen` |
| Signed-in session | **yes** — one real JSONL conversation |
| macOS AGENT-PROBE-V1 | this artifact |
| native Windows AGENT-PROBE-V1 | [`2026-08-17-windows-qwen.json`](../testing/results/agent-probes/2026-08-17-windows-qwen.json) |

## Record shape (2026-08-22)

Measured on macOS arm64 with `qwen` 0.21.13 driven against a **throwaway
`QWEN_HOME`** and a local stub model server. No real `~/.qwen` tree was read and
no real credential was used; the vendor prints its own warning that
`QWEN_HOME` is redirected, which is the confirmation that the override works.

A record's top-level keys match Claude Code's. **The body does not.**

```jsonc
{
  "uuid": "…", "parentUuid": "…"|null, "sessionId": "…",
  "cwd": "/Users/…", "gitBranch": "main"|null,
  "timestamp": "2026-08-21T23:47:19.664Z",
  "version": "0.21.12",              // CLI version string, not a schema number
  "type": "user"|"assistant"|"tool_result"|"system",
  "provenance": "real_user"|"assistant_output"|"tool_result"|"system",
  "message": { "role": "user"|"model", "parts": [ … ] }
}
```

`message` is a **Gemini `Content` value**, not a Claude content-block array. A
part is `{"text":…}`, `{"functionCall":{"id","name","args"}}`,
`{"functionResponse":{"id","name","response"}}`, or an inline/file data part.
`assistant` records add `model` and `usageMetadata`; `tool_result` records add
`toolCallResult` (`callId`, `status`, `resultDisplay`, `executionStatus`).
`system` records carry `subtype` plus `systemPayload` and no message at all.

That is why reading Qwen as Claude finds **no text whatsoever**: it looks for
`message.content[]` and Qwen writes `message.parts[]`.

### Rewind is structural, not a marker

Gemini writes a `$rewindTo` record. Qwen does not. The vendor's
`ChatRecordingService.rewindRecording()` sets `lastRecordUuid` back to the uuid
that was current just before the rewound user turn and then appends a
`subtype:"rewind"` system record there. The discarded turns stay on disk on a
**dead branch of the uuid tree**, and the vendor's own resume path
(`walkTranscriptUuidChain`) decides what is live by walking `parentUuid` back
from the last record.

A reader that walks the file line by line replays turns the user explicitly
threw away. `transcript.QwenReader` walks the same chain the vendor does and
reports the excluded records as a `qwen_rewound_records_excluded` warning.

### Project bucket

`sanitizeCwd` replaces every non-alphanumeric byte of `cwd` with `-`, after
lower-casing the whole path **on Windows only**. `/Users/u/code/demo` becomes
`-Users-u-code-demo`; `C:\Users\u\code\demo` becomes
`c--users-u-code-demo`. That is `ProjectKeyPathSlug` with a Windows-specific
case fold.

### Sidecars and archives

`0.21.13` writes the runtime status sidecar as `<sessionId>.runtime.json`
beside the conversation (the 2026-08-17 probe recorded the older
`<uuid>-runtime.json` shape). Both are `.json`, so neither matches the
`*.jsonl` session glob. Archived sessions move to `chats/archive/<id>.jsonl`,
which the glob `projects/**/chats/*.jsonl` also does not reach — archived Qwen
sessions are **not indexed today**.

## Identity

| Aspect | Value | Source |
| ------ | ----- | ------ |
| Vendor | Alibaba (QwenLM) | [qwen.ai/qwencode](https://qwen.ai/qwencode), [Alibaba Cloud Model Studio](https://www.alibabacloud.com/help/en/model-studio/qwen-code) |
| Product | Qwen Code | Official docs and product page |
| Binary | `qwen` | [Overview](https://qwenlm.github.io/qwen-code-docs/en/users/overview/) |
| Official repository | [QwenLM/qwen-code](https://github.com/QwenLM/qwen-code) | Product page and docs |
| Official docs | [qwenlm.github.io/qwen-code-docs](https://qwenlm.github.io/qwen-code-docs/en/users/overview/) | Linked from qwen.ai and Alibaba Cloud |
| npm package | `@qwen-code/qwen-code` | Official docs badge and uninstall page |
| Distribution | Official standalone installer (Alibaba OSS) and official npm | [Overview](https://qwenlm.github.io/qwen-code-docs/en/users/overview/), [Uninstall](https://qwenlm.github.io/qwen-code-docs/en/users/support/Uninstall/) |
| Storage family | F1 expected (home tree) | `QWEN_HOME` default `~/.qwen`; unverified as a session store |

## Why T0 was `layout_unverified` (historical)

At T0 the product, binary, repository, and official docs were settled; the
on-disk conversation layout was not. That was `layout_unverified`. The
2026-08-17 probes and the 2026-08-22 measurement above settled it.

## The managed self-updater moves the version underfoot

Qwen installs its own updates into `<QWEN_HOME>/updates/npm/<id>/versions/<v>/`
and then execs that copy. Consequences worth knowing before trusting a version
number:

- `qwen --version` answers differently on one machine depending on which root
  is in scope — 0.21.12 from the bundled npm install, 0.21.13 from the managed
  update in the default root, both observed on the same host within a minute.
  The descriptor's range spans both for that reason.
- Reinstate's version probe strips vendor root variables from the child
  environment by design, so the version it reads is not necessarily the version
  the launch runs. During the 2026-08-22 journey the probe read 0.21.13 (in
  range) while the launched process ran 0.21.15 out of the redirected root's
  `updates/npm` tree.

## Gemini-fork hypothesis

The 2025-07-22 Qwen3-Coder announcement described Qwen Code as a CLI "adapted
from Gemini CLI". Official Qwen Code docs now describe a **different** home
tree than Gemini's `~/.gemini` / `$GEMINI_CLI_HOME` layout.

| Claim | Official Qwen Code docs | Gemini layout already in Reinstate |
| ----- | ----------------------- | ---------------------------------- |
| Home root | `~/.qwen` via `$QWEN_HOME` | `~/.gemini` via `$GEMINI_CLI_HOME` |
| Runtime / conversations | `$QWEN_RUNTIME_DIR` (default: `QWEN_HOME`); "conversations, logs, todos" | `<root>/tmp/<project-hash>/chats/session-*.json(l)` |
| Rewind backups | `~/.qwen/file-history/` used by `/rewind` | JSONL `$rewindTo` records inside the session file |
| Documented `tmp/<hash>` file | `~/.qwen/tmp/<project_hash>/shell_history` | session JSON/JSONL plus checkpoints |

A fork of the CLI surface is not a fork of the session recording service.
Reusing `internal/transcript/gemini.go` without a probe would silently find
nothing, or worse, index the wrong files. **Hypothesis rejected as a shipping
basis.** Keep it as a later parser hint only after both probes confirm shape.

The 2026-08-16 macOS probe settles this concretely: Qwen Code keeps
conversations in `projects/<slug>/chats/`, while Gemini CLI keeps them in
`tmp/<project-hash>/chats/`. Qwen's `tmp/<64-hex>` exists but holds logs and a
task lock. Same fork ancestry, different store, different project-key kind.

## Claimed layout

Every row below is **Unverified**. No probe has observed the tree. Do not
treat these as a support claim.

| Aspect | Official claim | Notes |
| ------ | -------------- | ----- |
| Config root override | `$QWEN_HOME` | Absolute or relative; `~` expanded; empty string treated as unset |
| Config root default | `~/.qwen` | Uninstall preserves this directory by default |
| Runtime override | `$QWEN_RUNTIME_DIR` | "conversations, logs, todos"; defaults to `QWEN_HOME` when unset |
| User settings | `~/.qwen/settings.json` | May store API keys in the `env` object |
| Project settings | `<project>/.qwen/settings.json` | Not a session store |
| User env file | `<QWEN_HOME>/.env` | Secrets |
| Chat recording | `general.chatRecording` (default true) | Disabling also disables `--continue` and `--resume` |
| Rewind backups | `~/.qwen/file-history/` | Retention via `general.cleanupPeriodDays` (default 30) |
| Shell history | `~/.qwen/tmp/<project_hash>/shell_history` | Project hash from project root path. Not a transcript. |
| Project summary | `.qwen/PROJECT_SUMMARY.md` | Welcome-back flow; not a session |
| Session file format | **not documented** | `/status paths` claims to print current session file and log paths |
| Export | `/export html`, `/export md`, `/export json`, `/export jsonl` | Export is not the live store |

## Native control surface (measured 2026-08-22, macOS)

Exercised against a throwaway `QWEN_HOME` with `qwen` 0.21.12/0.21.13. Full
journey: [`2026-08-22-macos-qwen-t3.md`](../testing/results/2026-08-22-macos-qwen-t3.md).

| Aspect | Argv | Status |
| ------ | ---- | ------ |
| Resume by id | `qwen --resume <id>` | **verified on macOS** — prior turns replayed; unknown id exits 1 |
| Resume by id, attached | `qwen --resume=<id>` | verified, equivalent |
| Resume most recent | `qwen --continue` | verified |
| Fork | `qwen --resume <id> --fork-session` | verified — writes a new `chats/<uuid>.jsonl` with `forkedFrom` |
| New session at a chosen id | `qwen --session-id <uuid>` | verified — refuses an id that already exists, rejects a non-UUID |
| Initial prompt | `-p <text>` (non-interactive), `-i <text>` (interactive) | verified |
| Session listing | `qwen sessions list --json` | verified |
| In-session | `/resume`, `/continue`, `/rewind` (alias `/rollback`), `/branch` | documented, not exercised |
| Child-process identity | `QWEN_CODE=1` on `!` shell children | documented, not exercised |

`general.chatRecording` (or `--chat-recording=false`) disables recording, and
the vendor's own help says `--continue` and `--resume` stop working when it is
off. A session that was never recorded never appears in the index either, so
there is nothing to offer resume for.

### Handoff destination (T4)

`rein handoff --to qwen` launches
`qwen --session-id <uuid> --prompt-interactive "<briefing>"` in the verified
workspace: a **new** Qwen session seeded with a briefing, never a cross-agent
resume and never a reconstruction of the source thread. Journey:
[`2026-08-22-macos-qwen-t4.md`](../testing/results/2026-08-22-macos-qwen-t4.md).

The destination session id is knowable at launch, so lineage resolves rather
than guessing. The project bucket is always recomputed from the destination
workspace — a source device's directory name is never reused, because the
vendor lower-cases the path before sanitising it on Windows and only there.

Reinstate writes nothing under `$QWEN_HOME`. Qwen did not prompt for workspace
trust on a first launch in a fresh root, so there is no trust record to
pre-accept.

**The T3 and T4 claims are still one platform short.** A native Windows journey
has not been run.

## Secrets in the same tree

If a later task ever reads `$QWEN_HOME` or `$QWEN_RUNTIME_DIR`, these go in
`Excluded` **before** any read:

- `settings.json` — documented to hold API keys under `env`
- `.env` — documented credential file
- any OAuth token cache left from the discontinued Qwen OAuth flow

## What a later probe must settle

1. Whether conversations live under `$QWEN_HOME`, `$QWEN_RUNTIME_DIR`, or
   both, on macOS and native Windows.
2. The live session filename and record shape. Do not assume
   `tmp/*/chats/session-*.json*`.
3. Whether rewind is `file-history/` backups, in-file `$rewindTo` records,
   or both. A T2 reader must replay discards before emitting capsule events.
4. Subagent session files, if any, and how to exclude them.
5. Whether `tmp/<project_hash>` is also the conversation bucket or only
   shell history.

A probe is **required** to leave T0. Vendor documentation alone is not
enough. One-platform evidence is not enough.

The descriptor's discovery marker is `projects`, corrected from the
hypothesised `tmp` by the 2026-08-16 macOS probe. A marker is mandatory
because a bare `~/.qwen` is not evidence of an installation: before Qwen Code
was installed, a skill installer had already created `~/.qwen/skills`, and an
unmarked root reported that as an installed agent.

## Sources

- [Qwen Code product page](https://qwen.ai/qwencode)
- [Qwen Code overview](https://qwenlm.github.io/qwen-code-docs/en/users/overview/)
- [Settings (`QWEN_HOME`, `QWEN_RUNTIME_DIR`, `file-history`, `chatRecording`)](https://qwenlm.github.io/qwen-code-docs/en/users/configuration/settings/)
- [Commands (`/resume`, `/rewind`, `/export`, `/status paths`)](https://qwenlm.github.io/qwen-code-docs/en/users/features/commands/)
- [Authentication (credentials in `settings.json` and `.env`)](https://qwenlm.github.io/qwen-code-docs/en/users/configuration/auth/)
- [Uninstall (official binary and `~/.qwen` preserved)](https://qwenlm.github.io/qwen-code-docs/en/users/support/Uninstall/)
- [Official repository](https://github.com/QwenLM/qwen-code)
- [Alibaba Cloud Model Studio: Qwen Code](https://www.alibabacloud.com/help/en/model-studio/qwen-code)
- [Qwen3-Coder announcement (Gemini CLI adaptation claim)](https://qwen.ai/blog?id=qwen3-coder)
