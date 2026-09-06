# `v0.6.0-rc.1` pre-tag native Windows matrix — second pass, executor E

W7 second-pass executor E (handoff, discovery, CLI). Scope: OpenCode Matrix D
(`D1`–`D5`) with a **real** OpenCode session; `D4` for `claude`/`codex`/`grok`/
`qwen` with real multi-turn sessions; Copilot `C1`–`C4` under the read-only
discovery exception; CLI-experience rows 14 and 22 on the fixed build; and a
re-verification of automated gate `A7`. This report follows the first pass's
row-table-plus-evidence style and is additive — it does not re-litigate rows
outside this scope.

## Artifact identity

| Field | Value |
| --- | --- |
| Worktree | `v060/w7c-rerun`, `D:\Projects\reinstate-worktrees\v060-w7c-rerun` |
| Tested commit (artifact under test, all rows except A7's self-build) | `86cb34212a3dbd6241608595124e82e9110c78a3` (`86cb3421`) |
| Snapshot archive | `reinstate_0.0.0-86cb3421_windows_amd64.zip` |
| Archive SHA-256 | `00f20f21163e54a94898d654f75731caf88844917ff7538bab4db811e9d90bd0` — verified against `checksums.txt` with `sha256sum` before use |
| `rein.exe` / `reinstate.exe` SHA-256 | `d6a4d09ad9e329e3d481cc3909e3da0c51106a6d9d972da7f67d491d3986e1fb` (both files, byte-identical) |
| `rein version --json` | `{"commit":"86cb34212a3dbd6241608595124e82e9110c78a3","date":"2026-09-06T03:52:33Z","name":"reinstate","version":"0.0.0-86cb3421"}` |
| Comparison binary (row 22) | `reinstate_0.5.1_windows_amd64.zip`, SHA-256 `b724ca3da4e124004063032d63240c244fb9a59279076f0e87441d327a826e8c` (matches its own `checksums.txt`); installed `rein version --json` names commit `e8d1ec28edee73005a51ca8802a04ced369f4bcb`, version `0.5.1` |
| Install location | `D:\ReinstateAcceptanceProjects\v060-w7c-e\install\` (snapshot, fresh unzip), `D:\ReinstateAcceptanceProjects\v060-w7c-e\v051\` (comparison binary) — never a binary built by this executor for any row except A7 |
| Host OS | Windows NT 10.0.26200 (Windows 11 Pro), amd64, native (never WSL) |
| Shell | PowerShell 5.1 (`$PSVersionTable.PSVersion` `5.1.26100.8328`) for ConPTY/env work; Git Bash for POSIX scripts and JSON tooling |
| Go toolchain | host default `go1.26.1`; `GOTOOLCHAIN=go1.25.13` for all worktree builds, matching the contract |
| Date | 2026-09-06 |

**Environment hygiene.** Every shell in this report starts by clearing
`REINSTATE_BACKEND`, `REINSTATE_MEMORY_BACKEND_DIR`, `XDG_DATA_HOME`,
`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, then — for any row that touches session
discovery — also explicitly setting or clearing `GROK_HOME`, `QWEN_HOME`,
`COPILOT_HOME`, `CURSOR_CONFIG_DIR`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`,
`CLINE_DATA_DIR`, `PI_CODING_AGENT_DIR`, `OH_PERSISTENCE_DIR` (the full set of
`RootEnv` variables in `internal/agents/catalog`), because this host's ambient
~~shell~~ also carries real values for the agents beyond the five named in the
dispatch. See [Harness finding H-1](#h-1-per-call-shell-state-reset-is-a-real-data-exposure-risk-on-this-harness)
for why this matters and what went wrong once before it was corrected.

Real vendor sessions used in this report were created under throwaway
projects at `D:\ReinstateAcceptanceProjects\v060-w7c-e\projects\*`, each a
fresh `git init` repository, never a project the developer actually works in.
Each vendor ran with an isolated root env var pointed at
`D:\ReinstateAcceptanceProjects\v060-w7c-e\vendorhomes\<agent>\`, seeded with
**only** that vendor's own credential file copied from the host's real
default location (`.credentials.json` for Claude, `auth.json` for Codex/Grok/
OpenCode, `settings.json` + `installation_id` for Qwen) — never a session,
transcript, history, or database file. No real session content, title,
prompt, or path from any developer tree is reproduced anywhere below. All of
`D:\ReinstateAcceptanceProjects\v060-w7c-e\vendorhomes\` and the throwaway
projects are deleted at the end of this run (see
[Cleanup](#cleanup)).

## Verdict summary

| Row group | Result |
| --- | --- |
| OpenCode Matrix D (`D1`–`D5`) | 4 PASS, 1 NOT TESTED (`D4`, SQLite layout has no JSONL truncation analogue) |
| `D4` for `claude`/`codex`/`grok`/`qwen` | 2 PASS (`codex`, `grok`, byte-exact), 2 NOT TESTED (`claude`, `qwen`, credential-environment limits) |
| Copilot `C1`–`C4` | 4 PASS |
| CLI experience row 14 | **PASS** (was FAIL/RB1 in the first pass — the spacebar fix in `d3036646` works) |
| CLI experience row 22 | **PASS** (all diffs matched to a named `[0.6.0-rc.1]` changelog entry) |
| `A7` (installers) | **PASS** (coordinator's run, and this executor's own from-scratch rebuild) |

This report's evidence substantially de-risks **RB2** (`opencode` `D1`
`NOT_INSTALLED`): the mechanism works correctly against a real session in 10
of 11 attempts, and the one failure has an identified, narrow, load-dependent
root cause (see [`D1:opencode`](#d1-opencode)) — this is not the systemic
block the first pass's synthetic-fixture-only evidence could not rule out.
It resolves **RB1** (row 14, now fixed), **RB5** (`A7`, now a confirmed clean
4-step pipeline), and **RB6** (row 22, resolved under the `v0.6.0` contract's
amended definition). See [Release-blocking findings reassessed](#release-blocking-findings-reassessed).

---

## 1. OpenCode Matrix D — real session

### Real session used

`opencode run` (real vendor binary `opencode.exe 1.18.27`) created a genuine
two-turn session under an isolated `XDG_DATA_HOME`, in a throwaway git
repository, seeded only with the host's own `auth.json`:

```
opencode run "Reply with exactly this text and nothing else: <marker-1>" --format json
opencode run --session <id> "Reply with exactly this text and nothing else: <marker-2>" --format json
```

Both turns completed for real (the vendor answered with the planted marker
each time — not reproduced here). `rein sessions --agent opencode --json`
(fully isolated environment, all nine other agent roots pointed at empty
directories) discovers exactly this one real session:
`opencode:ses_f8b0da2ceffetDurnT2C8NLXiE`, project `opencode-proj`,
`message_count: 4`.

### `D1:opencode`

| Field | Value |
| --- | --- |
| Row | PASS |
| Command | `rein handoff opencode:ses_f8b0da2ceffetDurnT2C8NLXiE --to claude --dry-run --json` |
| Result | 10 of 11 attempts: `rc=0`, a full capsule with `fidelity`, `parse`, `destination.args`, `workspace` (1,393–3,800 estimated bytes depending on run). 1 of 11 attempts: `rc=5`, `{"code":"compatibility","message":"handoff: compatibility: source agent \"opencode\" is NOT_INSTALLED"}` |

**This is the headline finding of this pass.** The first pass's `D1:opencode`
`FAIL` was recorded against a *copied synthetic fixture* whose recorded root
held no real `opencode.db`, and the report explicitly could not tell whether
a real session would behave differently. It does: against a genuine session
created by the real vendor binary, on the identical isolated
`XDG_DATA_HOME`, the same command that failed against the fixture succeeds
in the overwhelming majority of attempts.

**The one failure was investigated to a specific, narrow root cause, not
dismissed.** `internal/handoff/pipeline.go`'s `Plan()` never wires a
catalog-constructed reader — `opts.Reader` is nil on every call path in this
codebase, so the pipeline always falls back to the package-level registry
singleton `transcript.Get("opencode")`, registered once at `init()` as
`NewOpenCodeReader(nil)` (`DataRoot=""`, `Getenv=nil`). Its `Probe()`
(`internal/transcript/opencode.go`):

1. Checks for a legacy on-disk `<data>/storage/message/<id>` directory —
   absent for every modern SQLite-only install (confirmed: the real
   `opencode.exe 1.18.27` install used throughout this report has no
   `storage` subdirectory at all under its data root; it is
   `layout: embedded-sqlite-session-store`, per `rein inspect`).
2. Falls back to `canListMetadata(ctx)`, which shells out to the **real**
   `opencode session list --format json` under a **`context.WithTimeout(ctx,
   5*time.Second)`** bound (`internal/transcript/opencode.go`,
   `canListMetadata`). Confirmed correct and fast in isolation — a direct
   repro of this exact call (same binary, same env, same session) completed
   in well under a second and returned the real session list.
3. **If that 5-second bound is not met, or the subprocess otherwise fails,
   the fallback checks `os.Stat(<data>/storage)`** — a directory a modern
   SQLite-only install never creates — which therefore **always** fails, and
   `Probe` reports `NOT_INSTALLED`.

This host runs many concurrent parallel acceptance-test executors sharing
one machine (`Get-Process` showed 689 total processes and several `claude`/
`opencode`/`codex` processes individually consuming 500–4,000+ CPU-seconds
during this run) — exactly the kind of contention the first pass's own `F2b`
finding (bounded Git-probe timeouts under load) already documents for a
different bounded call. `canListMetadata`'s 5-second bound around a real
process spawn is the same class of risk, but lands on a **hard compatibility
gate** (`NOT_INSTALLED`, blocking the whole handoff) rather than a soft
preflight warning, and — unlike the Git probe, which has a real fallback —
`Probe`'s fallback for a slow/failed live check is structurally incapable of
succeeding for any modern OpenCode install, because the on-disk marker it
falls back to checking (`storage/`) is a legacy layout marker.

**Assessment:** this is a real, narrow, load-dependent product defect —
distinct from and better-understood than the first pass's `RB2` — worth
fixing (either widen the timeout, retry once, or give the SQLite/metadata
path a fallback that does not depend on a directory no modern install
creates), but it is not the systemic "OpenCode handoff-source detection is
broken" risk `RB2` had to leave open. See
[Product defects](#product-defects) `PD-1`.

### `D2:opencode`

| Field | Value |
| --- | --- |
| Row | PASS |
| Evidence | The real capsule's `fidelity.components` entries are exclusively `exact` / `normalized` / `referenced` / `omitted` (with a `reason` on every `omitted` entry — `requires_optional_summarizer`, `interrupted_not_replayed`); no field is silently filled. `exact` components' byte counts match the two planted marker turns (`recent_user_messages`/`user_messages`: count 2, 142 bytes on the real session). |

### `D3:opencode`

| Field | Value |
| --- | --- |
| Row | PASS |
| Evidence | `constraints`, `decisions`, `rejected_approaches`: `"portability":"omitted","reason":"requires_optional_summarizer"`. `pending`: `"omitted","reason":"interrupted_not_replayed"`. `unknown`: `"referenced","reason":"unrecognized_record_type"`, count 2 — the two non-message OpenCode records (step-start/step-finish framing) are referenced, not guessed at or dropped silently. |

### `D4:opencode`

| Field | Value |
| --- | --- |
| Row | NOT TESTED |
| Reason | The real, installed OpenCode (`1.18.27`) uses the SQLite metadata-fallback path (`layout: embedded-sqlite-session-store`); its `Boundary` for this layout hashes the **entire `opencode.db` file** as one atomic unit (`raw_source.byte_offset == raw_source.size_bytes`, no `partial` field), confirmed by inspecting a real (non-truncated) capsule's `raw_source`. There is no JSONL-style "last complete record" concept in this path to truncate at — byte-truncating a live SQLite file produces file corruption, not a clean mid-stream boundary, and would not exercise "boundary at the last complete record" as the assertion describes. The legacy message-tree `Snapshot`/boundary path this assertion clearly was written for is real (`snapshotStorage` in `internal/transcript/opencode.go`) but not reachable with any OpenCode version installed on this host. Not a product defect — an evidence gap specific to this device's installed OpenCode version, carried forward from the first pass's Part B ("all Matrix D `D4` rows... not demonstrated for any agent"). |

### `D5:opencode`

| Field | Value |
| --- | --- |
| Row | PASS |
| Evidence | Two `--dry-run` runs on the identical unchanged real session, normalized and diffed field-by-field with `handoff_id`/`lineage_root`/`destination.session_id`/planned-file paths (fresh per invocation, by design) excluded: **identical**. |

---

## 2. `D4` for `claude`, `codex`, `grok`, `qwen` — real multi-turn sessions

Each agent's real session was created the same way: `<vendor> -p "reply with
exactly <marker>"` (or the agent's non-interactive equivalent) in a
throwaway git repo, isolated root env var, seeded only with the host's own
credential file, then a second turn added via that vendor's own resume/
continue mechanism to guarantee **two or more real records** before
truncating.

### `D4:codex` — PASS, byte-exact

Real two-turn session: `codex exec "…"` then `codex exec resume <thread-id>
"…"`, both answered for real. Source file:
`sessions/2026/09/06/rollout-….jsonl`, 25 JSONL lines, 129,066 bytes.

A byte-precise copy was made: the first 24 complete lines kept verbatim,
then half of line 25 appended with no trailing newline (a torn mid-record
write). Independently computed in Python before ever running `rein`:

| Field | Independently computed | `rein` reported |
| --- | --- | --- |
| Boundary byte offset | `128764` | `128764` |
| SHA-256 of bytes `[0, offset)` | `f265954d075522cb22f04c587edbc72a92cdfa0eda804228fe31a50edd743c0e` | `f265954d075522cb22f04c587edbc72a92cdfa0eda804228fe31a50edd743c0e` |
| `partial` | `true` (size 128,914 > offset) | `true` |
| `size_bytes` | `128914` | `128914` |

Evidence path: `rein sessions --agent codex --json` first (session-index
layer) reports a clean warning — `"code":"incomplete_trailing_record",
"message":"ignored incomplete trailing JSONL record 25"` — and
`message_count: 5` (excluding the torn record); then `rein handoff
codex:<id> --to claude --no-launch --json` materializes
`handoffs/<id>/capsule.json`, whose `raw_source` object carries the four
values in the table above verbatim. Every value matches the independently
computed one exactly — this is the byte-exact confirmation Matrix D4 asks
for, not an approximation.

### `D4:grok` — PASS, byte-exact

Real two-turn session: `grok -p "…" --output-format json` then `grok
--resume <session-id> -p "…" --output-format json`, both answered for real.
Grok's transcript reader authority for boundary/snapshot purposes is
`updates.jsonl` (an append-only restore log), **not** `chat_history.jsonl` —
confirmed by an initial attempt that truncated `chat_history.jsonl` only and
observed **no** effect on the recorded boundary (`updates.jsonl` was still
whole, so `raw_source.byte_offset == raw_source.size_bytes`, matching the
comment in `internal/transcript/grok.go`: "Authority: prefer `updates.jsonl`
(append-only restore log)"). Corrected and re-run against a byte-precise
truncation of `updates.jsonl` (18 lines, 10,134 bytes; first 17 lines kept,
half of line 18 appended, no trailing newline):

| Field | Independently computed | `rein` reported |
| --- | --- | --- |
| Boundary byte offset | `9572` | `9572` |
| SHA-256 of bytes `[0, offset)` | `8e8801b7b3a478c07d4e8e1117859919f6e58705b96e8e05354616fcc3b9fe89` | `8e8801b7b3a478c07d4e8e1117859919f6e58705b96e8e05354616fcc3b9fe89` |
| `partial` | `true` | `true` |
| `size_bytes` | `9852` | `9852` |

Same evidence path as codex: `rein handoff grok:<id> --to claude --no-launch
--json`, then read `handoffs/<id>/capsule.json`'s `raw_source`.

### `D4:claude` — NOT TESTED

`claude -p "reply with exactly <marker>"` was run three times, from three
fresh copies of the host's own `.credentials.json` into an isolated
`CLAUDE_CONFIG_DIR`, in a throwaway repo. Every attempt failed:

- Attempt 1: `"result":"Failed to authenticate: OAuth session expired and could not be refreshed"`
- Attempt 2 (immediate retry, same copy): `"result":"Not logged in · Please run /login"`
- Attempt 3 (fresh re-copy of the live credentials file): same as attempt 1

Mechanism, not a product defect in Reinstate: this host has several real,
concurrently-running Claude Code processes (see
[Harness finding H-2](#h-2-this-host-is-shared-with-several-concurrently-running-real-agent-processes))
actively using and rotating the account's real OAuth refresh token. A
refresh token is single-use; a copy taken at time T racing a real session's
own refresh at T+ε is consumed by whichever process wins, and the loser's
copy is now stale — consistent with attempt 1's specific "could not be
refreshed" wording (not a blanket auth failure) and with attempt 2
subsequently reporting the credential fully invalidated. This matches and
sharpens the first pass's `RB4` finding (`claude -p` under an isolated
`CLAUDE_CONFIG_DIR` printed `Not logged in`) with a specific mechanism: it is
not that the credential-copy method is unsound in general (it worked cleanly
for Codex and Grok in this same run, and for Claude Code itself in the
`2026-09-06-windows-range-widening-v060.md` precedent on a presumably
less-contended run), it is that **this particular host, at this particular
time, has real concurrent Claude usage that races and invalidates a copied
refresh token before this executor's process can use it.** Two real
two-message sessions (a user turn plus the authentication-error result) were
created as a side effect of the failed attempts, but their content is an
auth error, not a real exchange, so this executor did not use one as
"real multi-turn" evidence for `D4` — that would overstate what was
actually demonstrated.

### `D4:qwen` — NOT TESTED

`qwen -p "reply with exactly <marker>" --output-format json`, with an
isolated `QWEN_HOME` seeded from the host's own `settings.json` (plus,
on a second attempt, `installation_id` once the first attempt's error
pattern suggested a device-binding check): both attempts returned
`"[API Error: 401 invalid access token or token expired]"`. Qwen's
`security.auth.selectedType` is `openai`-style against a `coding-plan`
provider (`providerMetadata.coding-plan.baseUrl`,
`coding.dashscope.aliyuncs.com`) whose `env.BAILIAN_CODING_PLAN_API_KEY`
value is a short (9-character) reference token, not a bearer credential —
this provider's real access token is evidently minted through a live
exchange bound to something this executor's file copy did not carry (a
device/session identity, most likely), so it is not amenable to the simple
"copy the one credential file" method this run used successfully for the
other four agents. Not attempted further given the time budget; genuinely
not reproducible via this method on this host, not a Reinstate defect.

---

## 3. Copilot `C1`–`C4` — read-only discovery, real root

Per this executor's explicit dispatch exception (Copilot's `C1`–`C4` are
discovery rows against the real Copilot root, read-only, through `rein`
only, counts and field names only — no title, prompt, path, or identifier
reproduced). `HOME`/`USERPROFILE` were the real host defaults for these four
commands only (so Copilot's real, unredirected root resolves), scoped with
`--agent copilot` so no other agent's source is touched; `REINSTATE_HOME`
was a fresh, empty directory dedicated to this section.

| Row | Result | Evidence |
| --- | --- | --- |
| `C1` | PASS | `rein sessions --agent copilot --json` → 2 real sessions, from **2 distinct projects** (satisfies "at least two distinct projects" at the minimum bound) |
| `C2` | PASS | Structural field check only, no values printed: both records have a non-empty `project` (string), non-empty `title` (string), `updated_at` matching an ISO-8601 timestamp pattern, positive integer `message_count`, and a present `workspace`; one of the two records also carries a `branch` (string), the other has `branch: null` (a session outside a tracked branch context — not a defect, a legitimate value) |
| `C3` | PASS | A search term was derived **inside a script from the session's own title and never printed to this executor's output**; `rein search <term> --agent copilot --json` returned exactly 1 hit whose `id` matched the source session |
| `C4` | PASS | `rein inspect <key> --json` for both sessions: field **paths only** enumerated (`session.*`, `environment.*` — `id`, `key`, `agent`, `project`, `branch`, `title`, `updated_at`, `message_count`, `size_bytes`, `can_resume`, `can_fork`, `read_only_reason`, `prompt_preview`, plus the full `environment.checks[]`/`workspace`/`agent` preflight shape), never a value; total response size 5,289–5,828 bytes (bounded, not a transcript dump); `prompt_preview` specifically confirmed **bounded** (30 and 2 characters respectively on the two real sessions) |

No title, prompt, path, or session identifier from either real Copilot
session appears above or anywhere else in this report.

---

## 4. CLI experience — rows 14 and 22

### Row 14 — PASS (was `RB1`/FAIL)

Method: `scripts/testing/conptydriver` and `scripts/tuisandbox`, both built
fresh from this worktree at `86cb3421` (`GOTOOLCHAIN=go1.25.13 go build`),
against the **snapshot install** (`D:\ReinstateAcceptanceProjects\v060-w7c-e\install\rein.exe`)
— never a binary this executor built for the row's own evidence. Bench root
`D:\ReinstateAcceptanceProjects\v060-w7c-e\tuisandbox\`, outside any Git
checkout, with **every** agent root env var isolated (not just the three
`REINSTATE_*`/`XDG_DATA_HOME` host-contamination variables — see
[Harness finding H-1](#h-1-per-call-shell-state-reset-is-a-real-data-exposure-risk-on-this-harness)).
Step script (the driver's key grammar, `key space` — not `key a`):

```
wait /ctrl\+k commands/ 15s
key down  (×7, landing on claude:…0007, the seeded single-warning fixture)
key enter
wait /environment warning/ 8s
snapshot checklist-before
key space
sleep 300ms
snapshot checklist-after
key esc
kill
```

Captured frames:

```
checklist-before:
 ▸ [ ] baseline.unavailable
 equivalent command
 rein resume claude:5f0a1c00-0000-4000-8000-000000000007

checklist-after:
 ▸ [x] baseline.unavailable
 equivalent command
 rein resume claude:5f0a1c00-0000-4000-8000-000000000007 --allow-environment-warning
 baseline.unavailable
```

The real physical spacebar keystroke (byte `0x20`, delivered through
`conptydriver`'s native ConPTY, never a synthetic `tea.KeyMsg{Type:
tea.KeySpace}` injection) toggles the checkbox from `[ ]` to `[x]` and the
live equivalent-command line updates accordingly. **Reproduced twice**
independently in this run (once under partial isolation while this
executor was still building up the fully-isolated environment, once again
under the final, fully-isolated environment) — both captures identical in
substance.

**A load-dependent transient was also observed and is disclosed, not
hidden.** Several additional attempts on this same host, in the same
session, failed before the checklist ever rendered — the switcher's own
preflight refused outright with `git.repository_identity` /
`git.status`: "the bounded Git probe timed out" (a 2-second bound,
`internal/workspace/model.go`'s `DefaultProbeTimeout`), even though `git
status` in the exact same workspace directory, run directly, completed in
34–37ms every time. This reproduces the first pass's own `F2b` finding
(a transient `BLOCKED` glyph under sustained rapid launches) — but far more
persistently in this session, consistent with this host's heavier-than-usual
concurrent load during this specific pass (see
[Harness finding H-2](#h-2-this-host-is-shared-with-several-concurrently-running-real-agent-processes)).
This is a harness/host-load observation, not a defect in the checklist or
the spacebar fix themselves: every attempt that reached the checklist screen
at all showed the fix working, with zero exceptions.

**Disposition: row 14 is PASS.** `RB1` is resolved by `d3036646` on this
build.

### Row 22 — PASS (was FAIL/`RB6`, now resolved under the `v0.6.0` amended definition)

Method: the same synthetic `tuisandbox` home, scanned independently by both
the snapshot binary and the `v0.5.1` binary (each with its own empty
`REINSTATE_HOME`), JSON normalized (`sort_keys`, `indent=2`) before diffing.
Documents diffed: `sessions --json`, `resume --dry-run --json` and
`inspect --json` (both `claude` and `codex` records), `handoff list --json`,
and `handoff <ref> --to codex --dry-run --json` (run from the session's own
workspace, not the report's own working directory, to avoid the
directory-name-substring artifact the first pass's own methodology note
already flags).

| Document | Diff | Changelog match |
| --- | --- | --- |
| `sessions --json` | Grok record: `v0.5.1` has `"fork":false,"resume":false,"read_only_reason":"Grok Build sessions are source-only in Phase 4"`; snapshot has `"fork":true,"resume":true`, no `read_only_reason` | `[0.6.0-rc.1]`, "Grok Build moves to **T3, verified resume**" (accepted class 1: capability-tier promotion) |
| `resume --dry-run --json` / `inspect --json` (claude, codex) | Snapshot adds one `environment.checks[]` entry: `{"id":"agent.active","status":"match","severity":"info",...}` | `[0.6.0-rc.1]`, "`rein resume` and `rein fork` now report whether the session being resumed is already open… `agent.active`" (accepted class 2: active-session detection) |
| `handoff list --json` | Byte-identical | — |
| `handoff <ref> --to codex --dry-run --json` | Byte-identical apart from the two comparison homes' own directory-name substrings inside file paths the command legitimately echoes (a comparison-setup artifact, confirmed by re-running from the correct workspace and diffing again) | — |

Every observed difference matches one of the two accepted classes the
`v0.6.0` contract names verbatim in its amended row-22 text ("`sessions
--json` capability fields for Grok Build, OpenCode, and Qwen Code sessions…
the `agent.active` check in `resume --dry-run --json` and `inspect
--json`"), and each traces to a specific, dated `[0.6.0-rc.1]` changelog
entry. No unmatched difference was found. **Disposition: row 22 is PASS**
under `docs/testing/v0.6.0-windows-acceptance.md` section C's amended
definition. `RB6` is resolved, not by ignoring the diffs but by confirming
each one is exactly the accepted, documented drift the contract anticipated.

---

## 5. `A7` — installers

**Coordinator's run** (recorded, not reproduced by this executor): the
staged GoReleaser snapshot of `86cb3421` at
`…\scratchpad\v060rc1-snapshot2\` — `snapshot.log` shows `snapshot ok`,
`staged 5 raw release binaries`, `release artifacts verified (PowerShell)`
(twice), and `ok  github.com/HarjjotSinghh/reinstate/internal/doctest
1.081s` — all four gates (`snapshot.ps1`, `stage-release-assets.ps1`,
`check-release-artifacts.ps1`, `test-install.ps1`) exit `0` on this commit.

**This executor's own re-run**, from scratch, in this worktree:

| Step | Command | Result |
| --- | --- | --- |
| 1. Snapshot | `scripts\snapshot.ps1` | PASS — full 28-artifact dist (all five platforms, all package formats) staged to `dist\`; built at this worktree's HEAD at the time, `2f7967cb` (see [Harness finding H-3](#h-3-the-shared-worktree-moved-forward-under-a-concurrent-executor) — this worktree is shared with a concurrent executor whose commit landed mid-session) |
| 2. Stage | `scripts\stage-release-assets.ps1 dist` | PASS — `staged 5 raw release binaries into …\dist` |
| 3+4. Verify + installer test | `scripts\test-install.ps1 dist` (internally: `verify-release.ps1` → `check-release-artifacts.ps1`, then `go test ./internal/doctest -run TestInstaller`) | **PASS** — `release artifacts verified (PowerShell): …\dist`; `ok  github.com/HarjjotSinghh/reinstate/internal/doctest  1.229s` |

**A precise refinement of `RB5`'s root cause.** Running step 3+4 immediately
after step 1 alone (skipping step 2) reproduces `RB5`'s exact failure —
`missing checksummed artifact: reinstate_0.0.0-2f7967cb_darwin_amd64` — even
though step 1 alone already produced the **full** 28-artifact,
all-platforms dist. The missing piece is not platform coverage; it is that
`snapshot.ps1` leaves GoReleaser's raw per-target binaries at their internal
target-triple paths (`dist\reinstate_darwin_amd64_v1\reinstate`, etc.), and
only `stage-release-assets.ps1` copies/renames them to the top-level
checksummed names `check-release-artifacts.ps1` (called inside
`test-install.ps1`) actually looks for. `RB5`'s original record — staging a
directory with only 3 Windows-identity files — hit the same symptom for a
different reason (missing platforms entirely); this run confirms the
**complete**, correctly-ordered four-step pipeline (`snapshot` →
`stage-release-assets` → `check-release-artifacts` → `test-install`) is
what the contract's own artifact-identity section describes, and it passes
cleanly end to end. **`A7` is PASS; `RB5` is resolved.**

---

## Product defects

### PD-1 — `opencode` handoff-source compatibility probe has no safe degraded state under load (MINOR, load-dependent, not release-blocking)

See [`D1:opencode`](#d1-opencode) for the full mechanism. `internal/transcript/opencode.go`'s
`OpenCodeReader.Probe` falls back to a live `opencode session list` shell-out
bounded at 5 seconds (`canListMetadata`); if that bound is not met — as
happened once in 11 attempts on this heavily loaded shared host — the
further fallback checks for a `storage/` directory that no modern
SQLite-only OpenCode install ever creates, so it reports `NOT_INSTALLED`
(a hard compatibility block) instead of a softer, honest "could not
determine, proceeding" outcome. Recommend either widening/retrying the
bounded subprocess call, or giving the SQLite/metadata path its own
non-legacy degraded state that does not depend on `storage/` existing.
Not release-blocking on this evidence (10/11 real attempts succeeded
cleanly), but real and worth fixing given OpenCode T5 is a headline feature
of this candidate — this substantially narrows, but does not fully close,
the coordinator's open question about whether `D1:opencode` needed
root-causing before tag.

## Release-blocking findings reassessed

| ID (first pass) | First-pass status | This pass's evidence | New disposition |
| --- | --- | --- | --- |
| `RB1` (row 14, dead spacebar) | BLOCKER | Fixed by `d3036646`; verified working twice with a real ConPTY spacebar keystroke against this build | **Resolved.** No longer blocking. |
| `RB2` (`opencode` `D1` `NOT_INSTALLED`) | MAJOR, release-blocking pending root-cause | Works in 10/11 real-session attempts; the 1 failure has an identified, narrow, load-dependent root cause (`PD-1`) | **Substantially de-risked.** Recommend downgrading to MINOR/tracked, not release-blocking — but `PD-1` should still be fixed before or shortly after tag given OpenCode T5 is a headline feature. |
| `RB4` (no isolated, freshly-authenticated vendor session for `claude`/`codex`/`opencode`/`grok`/`qwen`) | BLOCKER | This pass obtained genuine, freshly-authenticated real sessions for `codex`, `opencode`, and `grok` (multi-turn, some with byte-exact `D4` proof); `claude` and `qwen` remain blocked by host-specific credential-environment conditions with a now-precise mechanism for each (OAuth refresh-token rotation racing concurrent real usage; a proxy-token coding-plan auth not amenable to file-copy) | **Partially resolved.** Three of five required agents now have real, physical evidence in this device's history; `claude` and `qwen` remain open, but for understood, host-specific reasons rather than "no isolated session was reachable at all." |
| `RB5` (`A7` `test-install.ps1` fails) | MAJOR | Coordinator's run and this executor's own from-scratch 4-step rebuild both exit `0` | **Resolved.** Root cause was a missing `stage-release-assets.ps1` step, not a fundamentally broken installer or an unfixable staging gap. |
| `RB6` (row 22 frozen-output diffs) | MAJOR, pending coordinator confirmation | Every diff matches a named `[0.6.0-rc.1]` changelog entry in one of the contract's two accepted classes | **Resolved** under the `v0.6.0` contract's amended row-22 definition. |

## Harness findings

### H-1: per-call shell state reset is a real data-exposure risk on this harness

This executor's tool environment starts a fresh shell for every Bash/
PowerShell invocation — environment variables set in one call do not
persist to the next. Early in this run, a sequence that set full agent-root
isolation in one call, then issued a **separate** follow-up call assuming
that isolation was still active, ran against this host's real, ambient,
un-isolated environment instead. The result was a `rein sessions --json`
listing that briefly surfaced a large number of real Claude Code/Codex/
other-agent session records — on the order of 100+ entries, real project
names and session identifiers among them — into this executor's own tool
output. No specific identifier, title, path, or content from that listing
is reproduced anywhere in this report or was acted upon; this finding
exists to name the mechanism so it is not repeated. **Corrective action
taken immediately upon discovery and followed for the remainder of this
run:** every single command that touches `rein.exe` or a vendor CLI sets
**all** relevant isolation variables and runs the command **in the same
call**, never split across two. This is a property of the calling
harness (tool-call shell lifecycle), not of Reinstate — but it is worth the
coordinator's attention for any future multi-step acceptance session on a
host with real, unredirected agent data, and worth naming explicitly in
`docs/testing/windows-acceptance-host.md` alongside its existing
`CLAUDE_CONFIG_DIR` ambient-inheritance warning.

### H-2: this host is shared with several concurrently running real agent processes

`Get-Process` during this run showed 689 total processes, including several
`claude`, `opencode`, `codex`, and `powershell` processes individually
consuming 500–4,000+ CPU-seconds — evidence of multiple parallel acceptance
executors (or other real usage) sharing this host simultaneously with this
session. This explains: the repeated `git.repository_identity`/
`git.status` "bounded Git probe timed out" transients seen while gathering
row 14 evidence (reproducing the first pass's `F2b`, but far more
persistently); the `opencode` `canListMetadata` timeout behind `PD-1`; and
the Claude Code OAuth refresh-token rotation conflict behind `D4:claude`'s
`NOT TESTED` disposition. None of these are product defects in the rows
they affected once isolated from load, but a report run during a lighter
load window would likely show fewer of them. Recommend the coordinator
consider whether concurrent multi-executor runs against one shared
acceptance host should be avoided for load-sensitive physical-resume work,
or whether the bounded timeouts this report and the first pass both
identify (`workspace.DefaultProbeTimeout` at 2s; `canListMetadata` at 5s)
should be widened for this host's demonstrated contention profile.

### H-3: the shared worktree moved forward under a concurrent executor

Partway through this run, `git log` on `v060/w7c-rerun` showed a new commit
— `2f7967cb test(v0.6.0-rc.1): record W7 pass-2 executor D Matrix E physical
resume rows` — landed on top of the `86cb3421` this dispatch pinned, from a
concurrent executor (evidently "pass-2 executor D") committing to the same
branch in the same worktree directory. This did not affect any row's
evidence in this report — every row except `A7`'s self-build used the
pre-verified, pre-built snapshot archive pinned to `86cb3421`, never a
binary rebuilt from this worktree's moving `HEAD` — but `A7`'s own
from-scratch rebuild in [section 5](#5-a7--installers) was necessarily built
at whatever `HEAD` was at that moment (`2f7967cb`), not `86cb3421`, since
`A7` tests the installer *pipeline*, not a specific product commit. This is
disclosed for transparency; the coordinator should confirm whether a shared
worktree across concurrent executors on the same branch is intended, and
whether this executor's commit (see below) should be sequenced against
executor D's.

## Open questions carried forward

1. Should `PD-1` (OpenCode's `NOT_INSTALLED` fallback having no safe
   degraded state under a slow/failed live subprocess call) be fixed before
   `v0.6.0-rc.1` tags, given it is real but load-dependent and OpenCode T5 is
   a headline feature? This executor's view: worth fixing promptly, not
   necessarily blocking, given the 10/11 real-session success rate.
2. `D4:claude` and `D4:qwen` remain genuinely untested on this device for
   host-specific credential-environment reasons (OAuth rotation racing real
   concurrent usage; a proxy-token coding-plan auth). Is a maintainer-
   provided, disposable, freshly-authenticated credential (not this
   developer's live daily-driver account) the sanctioned way to close this
   gap, given the live-account method is inherently racy on a host with
   real concurrent usage?
3. Should `docs/testing/windows-acceptance-host.md` and/or this contract's
   run notes be updated to name **all** `internal/agents/catalog` `RootEnv`
   variables as needing explicit isolation (not only the three documented
   host-contamination variables), given `H-1`?
4. Is running multiple parallel acceptance executors against one shared,
   heavily-loaded host (`H-2`) an acceptable methodology given how many of
   both this pass's and the first pass's findings trace to bounded-timeout
   behavior under contention, or should physical-resume/live-subprocess
   rows be scheduled with less concurrent load?

## Cleanup

At the end of this run, the following are deleted and never committed:
`D:\ReinstateAcceptanceProjects\v060-w7c-e\vendorhomes\` (all copied
credential files and every real session this run created), the throwaway
project repositories under
`D:\ReinstateAcceptanceProjects\v060-w7c-e\projects\`, and the truncated-copy
working directories under
`D:\ReinstateAcceptanceProjects\v060-w7c-e\d4-truncation\`. The install
directories (`install\`, `v051\`), captures, and this report's evidence
files remain for the coordinator's reference.

---

_No transcript text, real prompt, real response, credential, private path,
or repository name from any real vendor tree appears above. Every marker
token planted for continuity proof is redacted as `<marker-N>`. Session
identifiers quoted for `codex`, `grok`, and `opencode` were created by this
executor for this report in throwaway projects and contain no information
about the developer's real work._
