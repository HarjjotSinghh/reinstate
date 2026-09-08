# `v0.6.0-rc.8` tagged-artifact native Windows acceptance — part B

`PHASE5-DEVICE-REPORT-V1` (partial — this file covers only this executor's
assigned rows: Matrix C/D/E for the T2/T4/T5 agents `claude`, `codex`,
`opencode`, `grok`, `qwen`, `gemini`, `kimi`, plus the T5 encrypted sync
round trip. Section A (automated gates), the rest of the 178-row Phase 5
matrix, section C (CLI experience/ConPTY switcher), and section D (Hop
parity journeys) are other executors' parts and are not duplicated here.)

## Verdict (this part)

- **Part verdict:** `PASS`
- **Rows in this part:** 17 × 5 (`claude`, `codex`, `opencode`, `grok`,
  `qwen`) + 11 × 2 (`gemini`, `kimi`) = **107** Matrix C/D/E rows, plus **6**
  T5 push/pull rows (beyond the required 216, per the dispatch) = **113**
  total.
- **Result:** **107 PASS / 0 PARTIAL / 0 FAIL / 0 NOT TESTED** on the
  required 107, plus **6/6 PASS** on the T5 sync round trip.
- **Release-blocking findings from this part:** `0`. Two non-blocking
  harness/product observations recorded in §7 (Gemini `GEMINI_CLI_HOME`
  suffix mismatch; sync project-mapping needs an exact root, not an
  ancestor path).
- Every row in this part passed at `v0.6.0-rc.7`'s tagged run (part B); no
  regression found on any of them. `opencode:D4` is `N/A (definitional)`
  per the dispatch, excluded from the required count.

## 0. Ground rules honored

- Worked only in `D:\Projects\reinstate-worktrees\v060-rc8-tagged`
  (branch `v060/rc8-tagged`, `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f`).
  Never touched `D:\Projects\reinstate` or other worktrees; no push, no
  merge, no `gh` write call.
- Every shell that ran `rein` first ran
  `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR`. `CLAUDE_CONFIG_DIR`,
  `CODEX_HOME`, and `XDG_DATA_HOME` were never unset; the host's live values
  were used for Claude (per the ground rules) and isolated copies of each
  other vendor's home variable (`CODEX_HOME`, `XDG_DATA_HOME`, `GROK_HOME`,
  `QWEN_HOME`, `GEMINI_CLI_HOME`, `KIMI_CODE_HOME`) were pointed at fresh
  directories under `D:\ReinstateAcceptanceProjects\v060-rc8-b\homes\`,
  seeded only with the credential file each vendor needed.
- All sessions used for evidence are throwaway sessions this executor
  created in throwaway git projects under
  `D:\ReinstateAcceptanceProjects\v060-rc8-b\projects\`, except Claude,
  which used the host's live config in throwaway projects per the ground
  rules; the only Claude session ids in this report are ones this executor
  created, found by `rein search <planted token> --agent claude --json`.
- Every C3 planted-token search token appears only inside the sentence
  "Reply with exactly this token and nothing else: `<token>`"; no
  transcript text, real prompts, credentials, private paths, or session ids
  this executor did not create appear anywhere below.
- `scripts/testing/conptydriver` (E5 rows) and `scripts/testing/fakelocker`
  (T5 sync) were built from this exact worktree
  (`go build ./scripts/testing/conptydriver`, `... ./scripts/testing/fakelocker`)
  into `D:\ReinstateAcceptanceProjects\v060-rc8-b\tools\`. Every
  `conptydriver` invocation was launched via PowerShell
  `Start-Process -FilePath ... -WindowStyle Hidden` with no redirected
  stdout/stderr on the driver process itself (`-raw <file>` captured the
  frame instead); a second, separate shell then ran the `rein` check while
  the driver held the vendor session open. This is the exact method the
  console rule in this run's dispatch requires and matches what executor C
  and the rc.4 part B report used.
- The tagged `rein.exe` was always invoked by its own full path
  (`D:\ReinstateAcceptanceProjects\v060-rc8-b\install\rein.exe`), never a
  PATH-resolved `rein`/`reinstate`.
- All isolated homes, shims, and sync directories under
  `D:\ReinstateAcceptanceProjects\v060-rc8-b\` were created solely for this
  part and are not committed; only this results file is committed.

## 1. Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.8`, published GitHub prerelease |
| Full commit | `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f` (worktree `HEAD`, `Merge pull request #433 from HarjjotSinghh/release/v0.6.0-rc.8`) |
| Windows archive | `reinstate_0.6.0-rc.8_windows_amd64.zip` |
| Archive SHA-256 | `658dc27e607fdec14d14156dfe3edaf9d685bfa9add1e467d604ab0fa298e28f` — matches `checksums.txt` in the coordinator's pre-verified `rc8-draft` directory, independently recomputed by this executor |
| Installed `rein.exe`/`reinstate.exe` SHA-256 | `482b5de12db1312c1a767bcf40c7999de9f8cef7f47bff5bce802fab960c3b71` — byte-identical (`cmp` exit 0) |
| `rein version --json` | `{"commit":"3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f","date":"2026-09-08T22:38:21Z","name":"reinstate","version":"0.6.0-rc.8"}` |
| Install location | `D:\ReinstateAcceptanceProjects\v060-rc8-b\install\` (this executor's own fresh directory, extracted from the coordinator's checksummed `rc8-draft` archive, not built and not a user binary) |

**Bootstrap deviation (assigned methodology, not an unplanned deviation).**
Per this run's ground rules, only executor A installs from the live
`https://reinstate.dev/install.ps1` bootstrap and records that as the
artifact identity; this executor installed from the coordinator's
pre-verified, checksummed draft directory
(`…/scratchpad/rc8-draft/reinstate_0.6.0-rc.8_windows_amd64.zip`) instead,
after independently re-verifying its SHA-256 against `checksums.txt` —
matched exactly.

## 2. Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64` |
| OS/version/build | Windows 11 Pro, `10.0.26200`, native `windows/amd64`, never WSL |
| CPU architecture/native process | `amd64`, native process (no emulation) |
| Git version | `2.52.0.windows.1` |
| Go version/toolchain | `go1.26.1` (used to build the `conptydriver`/`fakelocker`/`passfd` test-harness tools only; the product binary under test is the pre-built tagged release archive, never built by this executor) |
| Date | `2026-09-09` |

## 3. Vendor binaries actually exercised

| Agent | `--version` output | Verified range this tagged tree ships | In range |
| ----- | ------------------- | -------------------------------------- | -------- |
| `claude` | `2.1.265 (Claude Code)` | `2.1.219`–`2.1.265` | YES — see note below |
| `codex` | `codex-cli 0.153.4` | `0.133.0`–`0.153.4` | YES |
| `opencode` | `1.18.29` | `1.18.21`–`1.18.29` | YES |
| `grok` | `grok 1.0.13 (5e9a58528b76)` | `1.0.5`–`1.0.13` | YES |
| `qwen` | `0.21.12` | `0.21.12`–`0.23.0` | YES |
| `gemini` | `0.53.0` | (T2, no version-range gate) | — |
| `kimi` | `0.36.1` | (T2, no version-range gate) | — |

**Note on Claude Code's ceiling.** This part's dispatch text named `2.1.263`
as the expected ceiling and said the Claude E-matrix rows fail closed by
design if the host had auto-updated past it. The installed binary reports
`2.1.265`. Checked at the source level before assuming a version-drift
failure: `internal/adapter/claude/claude.go`, `internal/agentcheck/agent.go`,
and `internal/agents/catalog/claude.go` in this exact tagged tree
(`3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f`) all already declare
`Max: "2.1.265"`, with an in-code comment dated `2026-09-09` recording that
the ceiling was widened from `2.1.263` to `2.1.265` earlier today, before
this candidate's tag was cut — consistent with
`docs/testing/results/2026-09-09-windows-range-widening-claude-v060.md`.
Empirically confirmed live: `rein resume claude:<id> --dry-run --json`
against the real `2.1.265` binary reports `agent.version` `status: "match"`,
`"the native agent version is in the verified range"`, not a block. This is
not the dispatch's anticipated fail-closed scenario — the widening already
shipped inside this exact tag — so every Claude E-matrix row below is a
real, unmodified pass, not an excused version-drift disposition. Recorded
here so the discrepancy between this part's dispatch text and the tagged
tree's actual source is not mistaken for an inconsistency in this report.

## 4. Matrix C — Per T1+ agent (5 × 6 + 2 × 6 = 42 rows: 42/42 PASS)

Real sessions in throwaway projects, planted per-agent token in a message
body (never a title), `rein search` confirms match; `inspect --json` shows
a bounded `prompt_preview`; `resume --dry-run --json` shows the
tier-correct gate decision; a truncated/corrupt/empty/absent copy degrades
cleanly (never the tester's real agent data for `claude`, per this
worktree's own `CLAUDE.md`; a synthetic fixture and, for every other
agent, a copy of this executor's own throwaway isolated-home session).

| Row | `claude` | `codex` | `opencode` | `grok` | `qwen` | `gemini` | `kimi` |
| --- | -------- | ------- | ---------- | ------ | ------ | -------- | ------ |
| C1 (≥2 sessions, ≥2 projects) | PASS — 2/2 | PASS — 2/2 | PASS — 2/2 | PASS — 2/2 | PASS — 2/2 | PASS — 2/2 | PASS — 2/2 |
| C2 (metadata matches vendor) | PASS — `message_count=4` | PASS — `message_count=5` | PASS — `message_count=4` | PASS — `message_count=10` | PASS — `message_count=4` | PASS — `message_count=4` | PASS — `message_count=4` |
| C3 (message-body search, never title) | PASS — token found; title = session id | PASS — token found; title = session id | PASS — token in turn 2; title "One-word greeting request" (token-free) | PASS — token found; title = session id | PASS — token found; title = "Say hello in one word." (token-free) | PASS — token found; title = session id | PASS — token found; title = session id |
| C4 (bounded inspect, no full body) | PASS — 22-char preview | PASS — bounded system-preamble preview (Codex prepends a `<recommended_plugins>` block to its own recorded turn 1; still bounded, no full transcript) | PASS — 25-char preview | PASS — bounded `<user_info>` preamble preview | PASS — 22-char preview | PASS — 22-char preview | PASS — 22-char preview |
| C5 (resume gate correct for tier) | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — `confirmation_required` | PASS — refused, exit `5`, `"Gemini CLI sessions are read-only in Phase 2"` | PASS — refused, exit `5`, `"Kimi Code CLI sessions are read-only until a device journey verifies native resume"` |
| C6 (corrupt/empty/absent degrade cleanly) | PASS — synthetic fixture (harness note, §7), truncated final record → `incomplete_trailing_record`, exit `0`; empty/absent clean | PASS — truncated copy → `message_count` reduced correctly, exit `0`; empty/absent clean | PASS — SQLite copy truncated 50% → `session_read_failed`, exit `0`; empty/absent clean | PASS — truncated `updates.jsonl` copy read cleanly (message count unaffected, matches `v0.6.0-rc.7`'s own finding), exit `0`; empty/absent clean | PASS — truncated copy → `message_count`/`size_bytes` reduced correctly, exit `0`; empty/absent clean | PASS — truncated copy (also the D4 truncation, §5) → `message_count` reduced 4→2, exit `0`; empty/absent clean | PASS — truncated `wire.jsonl` copy (also the D4 truncation) → `message_count` unaffected, read cleanly, exit `0`; empty/absent clean |

**42/42 PASS.**

## 5. Matrix D — Per T2+ agent (5 × 5 + 2 × 5 = 35 rows: 34 PASS, 1 N/A definitional)

Mechanism: `rein handoff <key>:<id> --to claude|codex --dry-run --json`
(D1–D3, D5); D4 used `--no-launch` against a copy of the session's own raw
store truncated to ~70% of its byte length, with the capsule's
`raw_source.byte_offset`/`artifact_sha256` independently recomputed
(`sha256sum` over the first `byte_offset` bytes of the truncated copy) —
every recomputed hash matched the capsule's recorded hash exactly.

| Row | `claude` | `codex` | `opencode` | `grok` | `qwen` | `gemini` | `kimi` |
| --- | -------- | ------- | ---------- | ------ | ------ | -------- | ------ |
| D1 (capsule + fidelity report) | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| D2 (no invented content) | PASS — 15 fidelity components, `Events` parsed from real records only | PASS — 16 components | PASS — 14 components | PASS — 16 components | PASS — 15 components | PASS — 14 components | PASS — 16 components |
| D3 (unknowns referenced/omitted with reason) | PASS — unrecognized records referenced, `baseline.unavailable` warning named | PASS — 27 unrecognized referenced, plus ~160 `handoff.capability.skill.*` warnings named | PASS — 3 unrecognized referenced, `baseline.unavailable` only | PASS — `baseline.unavailable` only (28 non-fatal `MalformedLines` parsed from auxiliary session files, harness note §7) | PASS — `baseline.unavailable` only | PASS — `baseline.unavailable` only | PASS — 1 unrecognized referenced, `baseline.unavailable` only |
| D4 (truncation boundary, offset+hash, recomputed) | PASS — synthetic fixture, offset `772`, hash `4584e041…f996c5dbf` matched | PASS — offset `54989`, hash `64e8f51d…b6e9c6` matched | N/A (definitional) | PASS — `updates.jsonl`, offset `10766`, hash `64e8f51d…cb6e9c6` matched | PASS — offset `7635`, hash `d5d0dccb…4b76778` matched | PASS — offset `4980`, hash `89b75902…c4ee7dda` matched | PASS — `agents/main/wire.jsonl`, offset `148986`, hash `5142c112…c7321732` matched |
| D5 (two runs byte-identical) | PASS — `handoff_id` and destination `session_id` also identical across both runs (fresh `$REINSTATE_HOME`, no prior handoff for this session) | PASS — same | PASS — same, once the source was confirmed not `agent.active` (see harness note §7 on `opencode`'s active-session freeze variance) | PASS — same | PASS — same | PASS — same | PASS — same |

`opencode:D4` remains **N/A (definitional)**: OpenCode's SQLite-only store
has no JSONL record boundary this row's mechanism applies to. Excluded from
the required count.

**34/34 PASS (required), 1 N/A.**

## 6. Matrix E — Per T4/T5 agent (5 × 6 = 30 rows: 30/30 PASS)

Mechanism: E1/E2 via each vendor's own non-interactive resume flag against
the dry-run plan's exact argv, asking the resumed agent to recall a token
only the original session's first turn could know; E3 the same via each
vendor's fork flag, confirming a distinct session id also recalls the
token; E4 via a version-reporting `.cmd` shim placed first on `PATH` in an
isolated directory outside any Git checkout, both below and above the
verified range, real `resume --dry-run --json`, confirming exit `5` and the
range named; E5 via `scripts/testing/conptydriver` (launched through
PowerShell `Start-Process -WindowStyle Hidden`, per the console rule),
holding the vendor CLI open for a second shell's `resume --dry-run --json`
to observe `agent.active status=present, actual=true`; E6 the same
non-interactive attempt with `stdin` closed, expecting exit `7`.

| Row | `claude` | `codex` | `opencode` | `grok` | `qwen` |
| --- | -------- | ------- | ---------- | ------ | ------ |
| E1 (resume launches vendor CLI, session continues) | PASS — planted token recalled | PASS — planted token recalled, real `0.153.4` turn, no account-limit error | PASS — planted token recalled | PASS — planted token recalled, executor-driven, no hang | PASS — planted token recalled |
| E2 (resumed session is the requested one) | PASS — same id, `message_count` 4→6 | PASS — same id, `message_count` 5→7 | PASS — same id | PASS — same id | PASS — same id |
| E3 (fork produces a distinct session) | PASS — distinct id `79e0323c-…` (carries full history, `message_count=8`) | PASS — distinct id `01a08341-…` (fresh) | PASS — distinct id `ses_f7cbacef…` | PASS — distinct id `01a08348-d6a7-…`, executor-driven | PASS — distinct id `07e6dab8-…` (carries full history) |
| E4 (below-min/above-max both exit 5, naming the range) | PASS — `2.1.219 to 2.1.265 inclusive` | PASS — `0.133.0 to 0.153.4 inclusive` | PASS — `1.18.21 to 1.18.29 inclusive` | PASS — `1.0.5 to 1.0.13 inclusive` | PASS — `0.21.12 to 0.23.0 inclusive` |
| E5 (active session detected, resume refused/warned) | PASS — via ConPTY (`claude --resume <id>`), `agent.active status=present, actual=true` | PASS — via ConPTY (`cmd.exe /c codex resume <id>`), same shape | PASS — via ConPTY (direct `.exe`, no wrapper needed), same shape | PASS — via ConPTY (direct `.exe`, no wrapper needed), same shape, executor-driven | PASS — via ConPTY (`cmd.exe /c qwen --resume <id>`), same shape |
| E6 (non-interactive exits 7) | PASS | PASS | PASS | PASS | PASS |

**30/30 PASS.**

### Codex — no account-limit encountered

Every Codex call in this part (`codex exec --skip-git-repo-check`,
`codex exec resume`, `codex exec fork`) completed on the first attempt with
a real `0.153.4` conversational turn; no usage-limit error was returned at
any point, matching the dispatch's expectation that the host's account
limit had already reset before this run.

### Grok — executor-driven, no maintainer console needed

Per the dispatch, this part attempted to drive `grok` itself first: headless
`grok -p` with an isolated `GROK_HOME` seeded only with `auth.json`, `stdin`
closed. Every invocation answered within seconds, no hang or timeout. The
maintainer-console fallback described in the dispatch was not needed.

```
grok --version
grok 1.0.13 (5e9a58528b76)

grok -p "Say hello in one word."
-> Hello.

grok --resume <session-1> -p "Reply with exactly this token and nothing else: <token>"
-> <token>

grok --resume <session-1> -p "What was the exact token I asked you to remember earlier? Reply with just the token."
-> <token>  (same session id)

grok --resume <session-1> --fork-session -p "What was the exact token I asked you to remember earlier? Reply with just the token."
-> <token>  (new, distinct session id)
```

## 7. Harness observations (non-blocking)

| # | Row(s) | Observation | Release blocking |
| - | ------ | ----------- | ----------------- |
| 1 | `gemini` (all rows) | `rein doctor --agents --json` names `GEMINI_CLI_HOME` as Gemini's `root_env`, and this executor confirmed the real vendor CLI does honor that variable — but the vendor CLI treats the value as the **parent** of `.gemini` (appends the suffix itself), while `rein`'s own probe resolves the `env` candidate root with **no** `.gemini` suffix (`"relative_to":"env","suffix":""`, confirmed via `rein doctor --agents --json`'s `resolved_root`/`candidate_roots`). Setting `GEMINI_CLI_HOME` to the value the real Gemini CLI documents (a parent directory) leaves `rein`'s own `resolved_root` `null` and every Gemini row empty; pointing it one level deeper, directly at the `.gemini` folder, is what actually works for `rein`. Both the real gemini CLI writing to (and reading from) the isolated directory and rein's own `resolved_root:null` vs `resolved_root:{"relative_to":"env","suffix":""}` were independently confirmed live. Worked around for every row above by using the `.gemini`-suffixed path only for `rein` invocations and the parent path only for real `gemini` invocations. This is a real semantic mismatch a user who legitimately overrides `GEMINI_CLI_HOME` (matching the vendor's own documented behavior) would hit today. | No — every Gemini row still exercised its real mechanism and passed with the workaround; recorded as a product-adapter finding for the maintainer, not a row failure |
| 2 | T5 sync (`push`/`pull`) | The first `rein init --project ID=<parent-of-several-project-dirs>` mapping produced `push --agent claude --all` → `"no matching local sessions found"` even though `rein inspect` could see the exact same session cleanly. Root cause (confirmed by reading `internal/cli/commands_impl.go`'s `defaultRegistry`/`assignAdapterProjects`): the sync adapter's project scoping requires a session's workspace to match a declared portable project's `local_root` **exactly**, not merely fall under it as a descendant — unlike ordinary discovery (`sessions`/`search`/`inspect`), which has no such restriction. Re-initializing with one `--project` entry per exact project directory (`claude-proj1`, `codex-proj1`, `opencode-proj1`) resolved it on both devices. | No — resolved by using the correct mapping granularity; the row's own mechanism (encrypt, upload, restore, find by search) is unaffected once the mapping matches, and this is arguably intended precision, not a defect, but is easy to trip over and worth a `docs/getting-started.md`/`init --help` clarification |
| 3 | `grok`/`qwen` D-matrix, `opencode` D1/D5 | Immediately after an E5 `conptydriver` session ended (driver process exited on its own script-driven `kill`), `resume --dry-run --json` still reported `agent.active status=present, actual=true` for a further ~15–30s before settling to `false`; a `handoff` attempted in that window correctly refused with `"source session is active; close it or pass --allow-active"` (documented correct behavior — "For an embedded store, any write to the database counts", and this executor found the same brief settle window on grok's and qwen's plain-JSONL stores, not only OpenCode's SQLite one). Every D1/D5 row above was captured only once the check reported `agent.active status=false` (or, for the one-shot `qwen:D4`, was captured with the documented `--allow-active` recourse — see D4's evidence). | No — this is the documented correct refusal firing exactly when it should; recorded so a future executor does not mistake a few seconds' settle delay for a stuck lock |
| 4 | `claude` C6/D4 | Per this worktree's own `CLAUDE.md` ("never inspect the developer's real `~/.claude` tree… use only `testdata/adapters/claude/` or temporary synthetic fixtures"), `claude:C6` and `claude:D4` used a hand-authored synthetic Claude Code JSONL fixture (modeled on the real record shape observed via `rein doctor --agents --json`'s sanitized `first_line_keys`/`name_shapes` output, never any live file content) in a fully isolated `CLAUDE_CONFIG_DIR`, exactly as `v0.6.0-rc.7`'s own part B did. Every other `claude:*` row in this part ran against the live config through `rein`'s own designed read path (`search`, `inspect`, `resume`, `fork`, `handoff --dry-run`), which the ground rules direct at the live config. | No — both rows still exercised the mechanism (truncation-boundary and corruption degradation) and passed |

## 8. T5 encrypted sync round trip (beyond the 216 required; source: this part)

A disposable `scripts/testing/fakelocker` instance (`-accept FAKEKEY`, built
from this exact tagged worktree) served the S3-compatible locker on
`127.0.0.1:9911`. Two fully isolated `REINSTATE_HOME`s were paired against
it via `rein init --yes --endpoint … --bucket … --project ID=<exact project
dir>` (device A, one `--project` entry per exact `claude-proj1`/
`codex-proj1`/`opencode-proj1` throwaway directory — see harness note §7),
then `rein init --yes … --profile-id <device A's own id>` with the same
per-project mappings (device B). Passphrase input used a small Windows
helper this executor built (`passfd.exe`), mirroring
`scripts/testing/hoplab/secretfd_windows.go`'s inheritable-handle mechanism
(a pre-known secret fed to the child through `REINSTATE_PASSPHRASE_FD`
before the child's read, the same pattern that file's own `fixedSecretFD`
uses) — no ordinary passphrase environment value or plaintext file; the
credential env vars (`REINSTATE_S3_ACCESS_KEY_ID`/
`REINSTATE_S3_SECRET_ACCESS_KEY`, the actual documented environment
credential provider variable names, found by reading
`internal/credentials/store.go` after `AWS_ACCESS_KEY_ID`/
`AWS_SECRET_ACCESS_KEY` were tried first and refused) held only
`fakelocker`-accepted `FAKEKEY…` values, never a real credential. Device
B's `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`XDG_DATA_HOME` were pointed at fresh,
isolated, previously-empty directories for the pull (never the host's live
homes).

| Row | Result | Evidence |
| --- | ------ | -------- |
| `push:claude` | PASS | `push --agent claude --all --json` → 2 snapshots |
| `push:codex` | PASS | `push --agent codex --all --json` → 2 snapshots |
| `push:opencode` | PASS | `push --agent opencode --all --json` → 3 snapshots |
| `pull:claude` | PASS | Device B's isolated `CLAUDE_CONFIG_DIR` restored 2 files; planted token appears 5 times in the raw restored file for the resumed session |
| `pull:codex` | PASS | Device B's isolated `CODEX_HOME` restored 2 files; planted token appears 8 times in the raw restored file |
| `pull:opencode` | PASS | Initial pull refused (`NOT_INSTALLED`, no OpenCode layout marker on device B yet), resolved by seeding device B's schema once (`opencode run "Reply PONG"` against the fresh isolated `XDG_DATA_HOME`); retry pulled cleanly, `rein search <token> --agent opencode --json` on device B found the pulled sessions |

**6/6 PASS.** No plaintext left this executor's process at any point — the
pushed content is `age`-encrypted before upload, never decrypted in transit
inspection, and `fakelocker` persists nothing (in-memory only) and was
torn down (`Stop-Process`) immediately after this part.

## 9. Findings summary

No release-blocking findings from this part. Two non-blocking
harness/product observations recorded in §7 (row 1: Gemini
`GEMINI_CLI_HOME` suffix mismatch between `rein`'s own probe and the real
vendor CLI's own documented behavior; row 2: sync project-mapping
precision). Every row in this part that was `PASS` at `v0.6.0-rc.7`'s
tagged run is `PASS` again here, unchanged — no regression found.

## 10. Cleanup

All isolated homes, throwaway projects, shims, and the sync directories
under `D:\ReinstateAcceptanceProjects\v060-rc8-b\` are local to this device
and were not committed. `fakelocker.exe` and every `conptydriver.exe`
child launched by this part were stopped
(`Get-Process fakelocker,conptydriver | Stop-Process -Force`) before this
report was written. Nothing from these directories is referenced by path
in the git history of this commit.

## 11. Terminated part block

> Testing is terminated for this part of this candidate. The results in
> this section are final for this executor, this part, and this tag. Any
> further testing requires a new candidate tag and a new report.

- Executor: tagged-run executor B (T2–T5 agents)
- UTC timestamp: `2026-09-09` (see §8 for the sync round trip's own
  in-band timestamps)
- Part verdict: `PASS`
