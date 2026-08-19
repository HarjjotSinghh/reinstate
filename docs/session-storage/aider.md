# Aider

**Confidence: Binary documented on macOS; Unverified on native Windows.**
No home-root tree. **Current tier:** T0 (`layout_unverified`) · **Phase 5
target:** T1

Catalog key is `aider`. Aider is the roster's only F4 agent: published
history files live **inside the repository**, not under a home root.

## Research outcome

**T0, reason `layout_unverified`.** That is the completed result for T-033.

The official product, binary, repository, and history filenames are settled
from vendor documentation. Dual-platform `AGENT-PROBE-V1` artifacts are
required for T1. This executor has no `aider` on PATH and no native Windows
host, so both probes cannot be committed. Do not invent a reader. Do not
walk the filesystem looking for `.aider.*`. Do not promote T2.

## Identity

| Aspect | Value | Source |
| ------ | ----- | ------ |
| Vendor | Aider AI LLC | [FAQ](https://aider.chat/docs/faq.html#what-is-aider-ai-llc) |
| Product | Aider | [aider.chat](https://aider.chat/) |
| Binary | `aider` | [Options](https://aider.chat/docs/config/options.html) |
| Official repository | [Aider-AI/aider](https://github.com/Aider-AI/aider) | FAQ |
| Official docs | [aider.chat/docs](https://aider.chat/docs/) | Product site |
| License | Apache 2.0 | FAQ / repository |
| Distribution | Official, open source | FAQ |
| Storage family | F4 (per-repository files) expected | Options "History Files"; unverified on disk |

## Device evidence (2026-08-19, macOS arm64)

Artifact:
[`2026-08-19-macos-aider.json`](../testing/results/agent-probes/2026-08-19-macos-aider.json)

| Check | Result |
| ----- | ------ |
| `aider` on PATH | yes — Homebrew `aider` 0.86.2 |
| macOS AGENT-PROBE-V1 | this artifact (executable + version only) |
| native Windows AGENT-PROBE-V1 | **absent** |
| Home root | none (F4). Probe `candidate_roots` is empty by design |
| Known-project files after a failed one-shot | `.aider.chat.history.md`, `.aider.input.history` in the repo cwd |

T1 is still forbidden: no native Windows probe, and no index source.
The home directory was not walked.

## Why F4, and why Roots stays empty

Every other catalog family answers "list my sessions" from a home root or a
vendor CLI. Aider cannot: the published defaults are files in the working
repository. Consequences the descriptor records now:

1. **Discovery scope.** A later F4 `projectfiles` scanner may look only
   inside workspaces Reinstate already tracks. Walking `$HOME` or the disk
   for `.aider.*` is a privacy hazard and is forbidden.
2. **No `Storage.Roots`.** A home-root candidate would make
   `rein doctor --agents` walk the user's home. T0 Aider emits empty
   `candidate_roots`.
3. **`AIDER_CHAT_HISTORY_FILE` is a file path, not a data root.** It is not
   `Storage.RootEnv`.

## Claimed layout

Every on-disk row stays **Unverified**. No probe has observed a tree. Do not
treat these as a support claim.

| Aspect | Official claim | Notes |
| ------ | -------------- | ----- |
| Chat history file | `.aider.chat.history.md` | `--chat-history-file`, env `AIDER_CHAT_HISTORY_FILE` |
| Input history file | `.aider.input.history` | `--input-history-file`, env `AIDER_INPUT_HISTORY_FILE`. Readline-style input, not a transcript |
| LLM debug log | none by default | `--llm-history-file` / `AIDER_LLM_HISTORY_FILE` (example `.aider.llm.history`) |
| Config search | `.aider.conf.yml` in home, then git root, then cwd | Later files win. `--config` loads only that file |
| Env file | `.env` in git root | `--env-file` / `AIDER_ENV_FILE`. API keys |
| Model settings | `.aider.model.settings.yml` | Not a session |
| Model metadata | `.aider.model.metadata.json` | Not a session |
| Ignore file | `.aiderignore` in git root | Not a session |
| Gitignore of history | `--gitignore` default true adds `.aider*` to `.gitignore` | Files may still be committed. Reinstate never writes, moves, or gitignores them |
| Session ID | **not documented** | |
| Session file format | rendered Markdown | FAQ shares `.aider.chat.history.md` as the transcript |
| JSON / JSONL store | **not documented** | Optional LLM log is a debug file, not a session catalog |

## Session identity

Vendor docs describe **one appended chat-history file per repository**
(unless relocated). They do not document a session ID, a session directory,
or a way to address one run inside the file.

`--restore-chat-history` restores "the previous chat history messages" from
that file. It is not `--session <id>`.

If a later probe cannot separate distinct runs, the record is **one session
per repository**. Do not fabricate session IDs to make the data model fit.

## Native control surface (documented, not T3)

| Aspect | Official claim |
| ------ | -------------- |
| Restore prior messages | `--restore-chat-history` (default **false**), env `AIDER_RESTORE_CHAT_HISTORY` |
| Resume a session ID | **none documented** |
| Continue last session | **none documented** as a session-ID argv |
| Clear in-chat history | `/clear`, `/reset` |
| Version flag | `--version` — output shape not captured |
| Process image | `aider` |

`--restore-chat-history` is a boolean reload of the whole Markdown file. It
is not a T3 `NativeSpec`. T3 needs a documented session-ID argv and
dual-platform physical resume journeys. **T3 is permanently unreachable**
through the current launch mechanism until the vendor publishes a
session-ID resume.

T2 is a separate decision. A rendered Markdown log is lossy by construction.
This task does not promote T2.

## Secrets and non-session files in the same tree

If a later task ever reads a known project, these go in `Excluded` **before**
any read:

- `.aider.conf.yml` — documented to hold OpenAI and Anthropic API keys
- `.env` — documented credential file
- `.aider.input.history` — input history, not the chat transcript
- `.aider.model.settings.yml`, `.aider.model.metadata.json`
- `.aider.tags.cache*` — repo-map cache, not a session
- `.aider.llm.history` — optional debug log, not the session catalog

## What a later probe must settle

1. Exact filenames on macOS and native Windows inside a **known** project
   after a real `aider` run. Confirm both the chat log and the input log.
2. Whether a machine-readable history exists anywhere. A JSON or JSONL
   alternative would change the tier ceiling.
3. Whether session boundaries are recoverable from the Markdown file. If
   not, one session per repository.
4. Whether `--chat-history-file` / `AIDER_CHAT_HISTORY_FILE` is used in
   practice to leave the repository, and whether any global default
   location exists beyond the documented per-repo file.
5. Whether `--restore-chat-history` is usable as anything other than
   "reload the whole file".

A probe is **required** to leave T0. Vendor documentation alone is not
enough. One-platform evidence is not enough. Do not walk the disk.

## Sources

- [Aider documentation](https://aider.chat/docs/)
- [Options (`--chat-history-file`, `--input-history-file`, `--restore-chat-history`, `--llm-history-file`)](https://aider.chat/docs/config/options.html)
- [YAML config (search order, history keys, `.gitignore` of `.aider*`)](https://aider.chat/docs/config/aider_conf.html)
- [`.env` config](https://aider.chat/docs/config/dotenv.html)
- [FAQ (share `.aider.chat.history.md`, Aider AI LLC, official repo)](https://aider.chat/docs/faq.html)
- [In-chat commands (`/clear`, `/reset`)](https://aider.chat/docs/usage/commands.html)
- [Official repository](https://github.com/Aider-AI/aider)
