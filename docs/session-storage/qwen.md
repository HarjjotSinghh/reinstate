# Qwen Code (Alibaba)

**Confidence: Documented on macOS and native Windows** —
official product identified; both platforms have a real JSONL conversation
with matching first-line keys; no Reinstate reader.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T2

Catalog key is `qwen`.

## Research outcome

**T0, reason `layout_unverified`.** Unchanged by the 2026-08-17 macOS re-probe.

A real conversation now exists on macOS and Windows, and the JSONL first-line
keys match. That is not a reader. Do not invent one. Do not reuse the Claude
reader.

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

**Tier unchanged at T0.** Dual-platform JSONL conversations exist, but there
is no index source and no reader. macOS also writes `<uuid-v4>-runtime.json`
sidecars that the Windows artifact did not record. Do not reuse the Claude
reader.

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

## Why T0 is `layout_unverified`, not `unidentified_product`

The product, binary, repository, and official docs are settled. What is not
settled is the on-disk conversation layout: file names, record shape, rewind
encoding, and whether `QWEN_RUNTIME_DIR` is the authoritative store. That is
`layout_unverified`. A later dual-platform probe can promote rows; it cannot
be skipped.

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

## Native control surface (documented, unverified)

| Aspect | Official claim |
| ------ | -------------- |
| Resume most recent | `--continue` (blocked when `general.chatRecording` is false) |
| Resume picker / named | `--resume` (same) |
| In-session resume | `/resume` or `/continue` |
| Rewind turns | `/rewind` (alias `/rollback`) |
| Fork conversation | `/branch` |
| Child-process identity | `QWEN_CODE=1` on `!` shell children |

These argv strings are not a T3 `NativeSpec`. T3 also needs a version probe
and dual-platform physical resume journeys.

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
