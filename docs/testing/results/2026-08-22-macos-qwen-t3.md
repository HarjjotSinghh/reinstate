# Qwen Code T3 journey — macOS arm64, 2026-08-22

`AGENT-TIER-JOURNEY-V1` · single agent, single platform.

**This is not a `PHASE5-DEVICE-REPORT-V1`.** It does not cover a release
candidate, an acceptance matrix, or any agent other than Qwen Code. It records
one thing: the physical evidence gathered on macOS for Qwen's T3 claim, and
what that claim is still missing.

## Verdict

- **macOS journey:** `PASS`
- **native Windows journey:** `NOT RUN` — coordinated separately
- **Tier claim complete:** **NO.** [`agent-support-tiers.md`](../../agent-support-tiers.md)
  requires a physical resume journey on macOS *and* native Windows for T3. Only
  the macOS half exists. Do not read this file as the whole gate.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-22` |
| Device | `macos-arm64` |
| Agent | Qwen Code (`qwen`), distributed via official npm `@qwen-code/qwen-code` |
| Vendor version, bundled npm install | `0.21.12` |
| Vendor version, managed self-update in the default root | `0.21.13` |
| Vendor version, managed self-update installed during this journey | `0.21.15` |
| Declared range under test | `0.21.12`–`0.21.13` |
| Go toolchain | `go1.25.13` |

## 2. Isolation

Every vendor invocation ran with `QWEN_HOME` pointed at a throwaway directory
outside any real agent tree. No real `~/.qwen` was read and no real credential
was used. The vendor confirms the redirect itself: with `QWEN_HOME` set it
prints a warning that its existing configuration and OAuth tokens remain at the
default root and are not migrated.

Model traffic went to a local stub that speaks the OpenAI chat-completions
shape, selected with `--auth-type openai` and `OPENAI_BASE_URL`. The stub is
what let a real turn complete offline. It does not affect what is under test
here: Reinstate launches the vendor CLI against the vendor's own session store,
and the model provider is not part of that contract. It does mean the *content*
of the replies is not vendor output, which is why nothing in this report claims
anything about model behaviour.

## 3. Native control surface, measured

Read from `qwen --help` and from the vendor's own yargs option table, then
exercised.

| Operation | Argv | Result |
| --------- | ---- | ------ |
| Resume by id | `qwen --resume <id>` | prior turns replayed into the model request; exit `0` |
| Resume by id, attached form | `qwen --resume=<id>` | identical |
| Resume most recent | `qwen --continue` | continued the same session |
| Fork | `qwen --resume <id> --fork-session` | wrote a **new** `chats/<uuid>.jsonl` whose records carry `forkedFrom {sessionId, messageUuid}` |
| New session at a chosen id | `qwen --session-id <uuid>` | created `chats/<uuid>.jsonl` at exactly that id |
| New session at an id already present | `qwen --session-id <existing>` | refused: "Session Id … already exists (active or archived). Delete or unarchive it first." |
| New session at a non-UUID id | `qwen --session-id not-a-uuid` | rejected as a usage error |
| Resume an unknown id | `qwen --resume <absent>` | exit `1`, "No saved session found with ID …" |

`--resume` is a yargs string option, so it accepts both the separated and
attached forms; used with no value it opens an interactive picker, which is why
the descriptor always substitutes an id.

**Continuation was observed, not assumed.** The stub logged each request body.
After a resume, the request carried the earlier user and model turns ahead of
the new prompt; the turn count grew by two on each subsequent resume. That is
the vendor reconstructing the conversation from its own store.

## 4. Reinstate journey

Run with `rein` built from this branch, from a workspace whose sessions were
created in step 3.

| Step | Result |
| ---- | ------ |
| `rein sessions --agent qwen --json` | all three sessions listed, titled from their first user prompt |
| `rein inspect qwen:<id> --json` → `agent.executable` | `present` / info |
| … `agent.layout` | `match` / info |
| … `agent.version` | `match` / info — the installed version resolved inside the declared range |
| … `agent.active` (nothing running) | `match` / info, "no running qwen instance is using this session" |
| `rein resume qwen:<id> --dry-run --json` | `executable: qwen`, `args: ["--resume", "<id>"]`, cwd = recorded workspace |
| `rein fork qwen:<id> --dry-run --json` | `args: ["--resume", "<id>", "--fork-session"]` |
| `rein resume qwen:<id>` (real launch, acknowledged warnings) | vendor TUI started against the recorded session |
| … `agent.active` while that session is open | `present` / **warning**, "a running qwen instance is already using this session", repair names `--allow-environment-warning agent.active` |
| … `agent.active` after the agent exits | back to `match` |

The `agent.active` row is the T3 requirement that a session is not resumed
underneath the operator, and it is **scoped** — Reinstate identified the running
instance as using *this* session, not merely that some Qwen was running.

Process shape, for the record: the launcher runs as `node …/bin/qwen`, and the
two workers it re-execs run as
`node --expose-gc …/@qwen-code/qwen-code/cli.js`. The workers are what the
descriptor's node marker matches; the launcher alone would not be recognised.

## 5. Finding: the managed updater can outrun the version gate

Qwen installs its own updates into `<QWEN_HOME>/updates/npm/<id>/versions/<v>/`
and then execs that copy. Two consequences, both observed here:

1. **One machine reports several versions.** `qwen --version` answered `0.21.12`
   with `QWEN_HOME` pointed at a fresh root, and `0.21.13` with the default
   root, on the same host in the same minute. The range in the descriptor spans
   both for that reason, not for slack.
2. **The probed version is not necessarily the running one.** Reinstate's
   version probe deliberately strips vendor root variables from the child
   environment, so it read `0.21.13` — in range, `agent.version match` — while
   the process the launch actually started was running `0.21.15` out of the
   redirected root's `updates/npm` tree. The gate passed on a version that was
   not the one executing.

That is not specific to Qwen's storage layout; it is a property of any agent
whose self-updater is rooted inside the directory the operator can redirect. It
is recorded here rather than fixed, because the fix is a decision about the
version-probe environment contract and belongs with whoever owns that contract.

## 6. What this journey does not establish

- **Native Windows.** Nothing here was run on Windows. Qwen's project bucket is
  lower-cased before sanitising on Windows only, so the directory name differs
  between platforms; the Windows fixture encodes that rule from the vendor's
  source, not from an observed Windows tree.
- **Rewind.** No `/rewind` was performed against a live session, so the
  `systemPayload` body of a `subtype:"rewind"` record remains unobserved. The
  reader treats it as opaque for that reason.
- **Vendor model behaviour.** A local stub answered every request.
- **Archived sessions.** `chats/archive/<id>.jsonl` was not exercised; the
  session glob does not reach it today.
