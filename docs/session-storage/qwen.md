# Qwen Code (Alibaba)

**Confidence: Unverified** — official product identified; no device probe;
no Reinstate reader. Vendor documentation is not a tier promotion.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T2

Catalog key is `qwen`.

## Research outcome

**T0, reason `layout_unverified`.** That is the completed result for T-022.

The official product is identified. Dual-platform probes are required for T1+.
This task has no native Windows device, so both probes cannot be committed.
Do not invent a reader. Do not reuse the Gemini CLI reader.

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

The descriptor's discovery marker is `tmp`, carried over from the Gemini-fork
hypothesis and therefore itself unverified. It exists so that a bare `~/.qwen`
does not resolve as an installation: on a machine with no Qwen Code at all, a
skill installer had already created `~/.qwen/skills`, and an unmarked root
reported that as an installed agent. If a probe shows `exists: true` with
`marker_present: false` on a machine where Qwen Code *is* installed, the marker
is wrong and item 1 above is what corrects it.

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
