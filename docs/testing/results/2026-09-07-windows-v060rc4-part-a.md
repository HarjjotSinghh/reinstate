# `v0.6.0-rc.4` tagged-artifact Windows acceptance — part A (gates, core matrices, T0/T1)

`PHASE5-DEVICE-REPORT-V1` (partial — this executor's assignment only)

This is executor A's part file for the **published, signed GitHub prerelease
`v0.6.0-rc.4`**, satisfying the rows of
[`../v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) this
executor was assigned: section A automated gates (A1–A8); Phase 5 core
matrices A (10), B (9), G (8), H (6); T0 agents F1/F2 (`aider`, `amp`,
`antigravity`, `minimax-code`, `openhands`, `roo`, `zcode`); and T1 agents
C1–C6 (`cline`, `copilot`, `cursor`, `pi`). Per
[`../v0.6.0-rc.4-agent-verification-prompts.md`](../v0.6.0-rc.4-agent-verification-prompts.md),
this candidate specifically re-tests `pi:C3` (the `F-PI-CONTENT-EXTRACTION`
fix); this part carries that row and confirms the fix on a fresh
planted-token session.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.4` |
| Full commit | `561ec133e7fd040d7937d555a75f0bd7dd7b878e` |
| Release workflow run | `34092326851` |
| Archive | `reinstate_0.6.0-rc.4_windows_amd64.zip` |
| Archive SHA-256 (re-verified against `checksums.txt` before install) | `f5ae24e2edf0364f0e6f9b0df3c5b2c21f47222ecd394b9c40fc5dbd1ae14d87` — matched |
| Installed `rein.exe` / `reinstate.exe` SHA-256 | `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef` (both; `fc.exe /b` confirmed byte-identical, "no differences encountered") |
| `rein version --json` | `{"commit":"561ec133e7fd040d7937d555a75f0bd7dd7b878e","date":"2026-09-07T06:48:08Z","name":"reinstate","version":"0.6.0-rc.4"}` |
| Install source (section B–H rows) | `checksums.txt`-verified copy of the coordinator's staged draft (`…/scratchpad/rc4-draft/reinstate_0.6.0-rc.4_windows_amd64.zip`), extracted into this executor's own fresh `D:\ReinstateAcceptanceProjects\v060-rc4-a\install\` |
| Bootstrap deviation | Executor A alone installs from the live bootstrap and records that identity (below); this is the artifact-identity run for the whole tagged pass. All Phase 5 matrix/T0/T1 rows below were exercised against the checksummed-archive install in `…\v060-rc4-a\install\`, with that directory placed first on `PATH` under both `rein` and `reinstate` names (`Get-Command`/`where.exe` confirmed resolution order before every row) |

### Live-bootstrap artifact-identity evidence (executor A only)

- Saved `https://reinstate.dev/install.ps1` via `Invoke-WebRequest -OutFile` to
  `D:/ReinstateAcceptanceProjects/v060-rc4-a/bootstrap/install.ps1`. Its pinned
  version line reads `$Version = "v0.6.0-rc.4"`, and it downloads
  `scripts/install.ps1` from
  `raw.githubusercontent.com/HarjjotSinghh/reinstate/v0.6.0-rc.4/scripts/install.ps1`
  with a pinned installer SHA-256.
- Ran it with `INSTALL_DIR` pointed at
  `D:/ReinstateAcceptanceProjects/v060-rc4-a/bootstrap/dist` (never the
  default `%LOCALAPPDATA%\Programs\Reinstate\bin`, which already held an
  unrelated stale dev install on this host — the installer correctly refused
  to touch it, confirming the "never replace a user binary" default is
  intact). "installer checksum ok", "checksum ok", exit 0.
- Installed `rein.exe`/`reinstate.exe` in that fresh directory: byte-identical
  (SHA-256 `fbea94615dabcbc30fc1ebb6e93259ddd330b62573555e551fd3112b31b9d3ef`
  for both, `fc.exe /b` "no differences encountered"), `version --json` names
  commit `561ec133e7fd040d7937d555a75f0bd7dd7b878e` and version
  `0.6.0-rc.4` — identical to the checksummed-archive install used for every
  other row in this part.

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| Device | `windows-amd64`, native process, no emulation |
| OS/version/build | Windows 11 Pro, `10.0.26200` |
| Go toolchain | `go1.25.13` via `GOTOOLCHAIN` pin in `Makefile`/scripts (host `go` reports `go1.26.1`) |
| Git version | `2.52.0.windows.1` |
| Filesystem | NTFS |
| Date | 2026-09-07 (UTC evidence timestamps in command output) |
| Every shell | `Remove-Item Env:REINSTATE_BACKEND`, `Env:REINSTATE_MEMORY_BACKEND_DIR` (PowerShell) or `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (Git Bash) run first in every shell that invoked `rein`, `go test`, or a release script; confirmed empty before each row. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left untouched throughout — every T1 discovery row (C1–C4) ran against the host's real live agent roots, read-only, through `rein` alone |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc4-tagged`, branch `v060/rc4-tagged`, shared by the other tagged-run executors (this file only adds its own commit) |
| Throwaway project roots used | `D:\ReinstateAcceptanceProjects\v060-rc4-a\` (bootstrap, install, matrix/gate evidence, T0/T1 probe/copy directories); `C:\Users\admin\rein-accept\proj-e5` and prior throwaway acceptance directories re-used read-only as real-session sources for handoff/search rows |

## Section A — Automated gates (required)

All gates run once against this candidate, per the dispatch ("Do not run the
Go test suite; executor A alone runs the section A automated gates once with
`-p 4`"). A2/A3 ran with `-p 4`; A1, A4, A5, A8 ran from the worktree; A6/A7
ran against the coordinator's checksummed `rc4-draft` directory.

| # | Gate | Command | Result | Evidence |
| - | ---- | ------- | ------ | -------- |
| A1 | Format, vet, tidy | `gofmt -l .`; `go vet ./...`; `go mod tidy -diff` | **PASS** | All three produced empty output, exit 0 |
| A2 | Unit suite | `CGO_ENABLED=0 go test ./... -count=1 -p 4` | **PASS** | Every package `ok` (or `[no test files]`); exit 0. Longest packages: `internal/cli` 71.4s, `internal/workspace` 15.3s, `scripts/testing/phase3perf` 21.9s |
| A3 | Race suite | `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p 4` | **PASS** | Every package `ok`; zero `DATA RACE` or `FAIL` lines; exit 0 |
| A4 | Lint and vuln | `make lint`; `make vuln` | **PASS** | `golangci-lint v2.11.4`: "0 issues."; `govulncheck v1.6.0`: "No vulnerabilities found... 0 vulnerabilities in packages you import" (4 unreached vulnerabilities in required-but-uncalled modules, not code the binary calls) |
| A5 | Doc gate | `go test ./internal/doctest/... -count=1`; `scripts/check-docs.ps1` | **PASS** | `ok internal/doctest`; `check-docs.ps1` exit 0 (re-runs the same doctest package) |
| A6 | Snapshot and artifacts | `scripts/verify-release.ps1 -DistDir <rc4-draft>`; `scripts/check-release-binary-identity.ps1 -DistDir <rc4-draft> -ExpectedCommit 561ec1... -ExpectedVersion 0.6.0-rc.4` | **PASS** | "release artifacts verified (PowerShell): …\rc4-draft"; "release binary identity ok: version=0.6.0-rc.4 commit=561ec133e7fd040d7937d555a75f0bd7dd7b878e"; both exit 0 |
| A7 | Installers | `scripts/test-install.ps1 -DistDir <rc4-draft>` | **PASS** | Re-runs `verify-release.ps1`, then `go test ./internal/doctest -run TestInstaller -count=1` → `ok`; exit 0 |
| A8 | Cross-OS build | `GOOS=darwin GOARCH=arm64 go build ./...`; `GOOS=linux GOARCH=amd64 go build ./...` | **PASS** | Both exit 0, no output |

**Section A: 8/8 PASS.**

## Phase 5 Matrix A — Catalog integrity (10 rows, required)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Every catalog descriptor validates at process start; no duplicate/empty keys | **PASS** | Implicit in every successful `rein` invocation below plus `internal/agents/catalog` and `internal/agents/conformance` (`TestShippedAgentsConformance`) passing in the A2 unit suite |
| A2 | `rein doctor --agents` lists every catalog agent, including every T0 agent | **PASS** | Human `rein doctor --agents` lists all 18 catalog keys (5 T0 shown as `no`/`t0_reason=...`, plus `desktop_only`/`server_backed`/`layout_unverified` reasons) |
| A3 | Each T0 agent shows a machine-readable reason from the closed enumeration | **PASS** | All 7 T0 agents show `t0_reason=layout_unverified` (`aider`, `antigravity`, `minimax-code`, `roo`), `t0_reason=server_backed` (`amp`, `openhands`), or `t0_reason=desktop_only` (`zcode`) |
| A4 | Declared tier and present capability constructors agree for every agent | **PASS** | `TestShippedAgentsConformance` (`internal/agents/conformance/shipped_test.go`) passed in the A2 suite |
| A5 | Every `Evidence` path in every descriptor exists in the tagged tree | **PASS** | `TestShippedEvidenceIsComplete` passed in the A2 suite |
| A6 | `rein sessions --agent <key>` accepts exactly the T1+ keys, rejects others exit `2` | **PASS** | Accepted list is exactly `claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen` (T1+T2+T4+T5, 11 keys); `aider` (T0) and a nonsense key both refused with `invalid agent; expected …` and exit code `2` |
| A7 | `rein resume` accepts exactly the T3+ keys, refuses others exit `5` with a reason | **PASS** | `resume pi:x`/`gemini:x` (T1/T2) and `resume aider:x` (T0) all refused `native session action is unsupported: <Agent> is tier T<N>[, (reason)]; native resume is unsupported`, exit `5`; `resume claude:x` took the different, session-lookup code path (`session not found`, exit `2`), confirming T5 keys are accepted syntactically |
| A8 | Agent order in all output is deterministic across runs | **PASS** | Three consecutive `rein doctor --agents` runs produced byte-identical output (same SHA-256 all three) |
| A9 | A descriptor with a deliberately broken evidence path fails the conformance suite | **PASS** | `TestBrokenEvidencePathIsCaught` / `TestBrokenEvidencePathFails` passed in the A2 suite |
| A10 | No catalog agent's scan writes, renames, or locks any file under an agent root | **PASS** | SHA-256 digest (relative path + size + mtime for every file) of 8 real agent roots taken before and after `rein sessions --agent all --json` (a full index refresh): all 8 digests identical |

**Matrix A: 10/10 PASS.**

## Phase 5 Matrix B — Probe and redaction (9 rows, required)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | `rein doctor --agents --json` conforms to `AGENT-PROBE-V1` | **PASS** | Top-level `"schema": "AGENT-PROBE-V1"` |
| B2 | Output contains no absolute path, home directory name, or username | **PASS** | Programmatic scan of the full probe JSON: zero occurrences of the host username, zero `C:\`-style absolute paths; every path field uses `"relative_to": "home"` or `"relative_to": "env"` with a relative `suffix` |
| B3 | Output contains no JSON value from any session file, only keys | **PASS** | `first_line_keys` entries are arrays of field names only (e.g. `["schema","tracking"]`, `["data","id","parentId","timestamp","type"]`), never values |
| B4 | Directory/file names are shape-normalized, no raw UUID/hash/path slug | **FAIL** | For the `gemini` agent's `tmp/` subtree, the probe's `tree` output includes **two top-level directory segments that are real, human-chosen project-name strings copied verbatim from this host**, not shape-normalized — every path segment *beneath* each of those two directories is correctly normalized (`*`, `<uuid-v4>`, `<n>`), but the top segment itself is not. This is a real (not synthetic) project-identifying string reaching the "redacted" probe output; not quoted here per the no-private-names rule. Reproducible: `rein doctor --agents --json` on this host, `agents[].tree` for `key=="gemini"`, entries under `tmp/` |
| B5 | No excluded subtree, incl. credential/cache paths, appears anywhere in output | **PASS** | Full-text scan for credential/token/secret/`.ssh`/password/keyring substrings found only benign usage-metric field names (`cachedTokens`, `contextTokensUsed`, etc.); no credential-subtree paths or values |
| B6 | Probe against an absent agent reports absence without error | **PASS** | 7 T0 agents (none installed) all show `resolved_root: null`, `executable_on_path: false`, no error, exit 0 |
| B7 | A root that exists but is empty is distinguishable from an absent root | **FAIL** | Pointed `CLINE_DATA_DIR` at (a) a real, empty directory and (b) a nonexistent path. Both `rein doctor --agents --json` outputs are **byte-identical apart from the `generated_at` timestamp** — `resolved_root: null`, empty `tree`, human view shows `root=no` in both cases. The override correctly stops silent fallback to the real home root (a different, non-null `resolved_root` when unset, confirming the override is read), but "exists and empty" and "does not exist" are not distinguishable in either output |
| B8 | `agent-storage-probe.sh`/`.ps1` produce output identical to the binary | **PASS** | Both wrapper scripts are thin argv-forwarders to `rein/reinstate doctor --agents --json`; ran both and diffed against a direct binary invocation (ignoring only the per-invocation `generated_at` timestamp) — identical |
| B9 | The fixture secret scanner passes over every committed probe artifact | **PASS** | `go run github.com/zricethezav/gitleaks/v8@v8.30.1 dir --config .gitleaks.toml docs/testing/results/agent-probes/`: "scanned ~377633 bytes… no leaks found", exit 0 |

**Matrix B: 7/9 PASS, 2 FAIL (B4, B7 — see findings below).**

## Phase 5 Matrix G — No regression (8 rows, required)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude Code and Codex CLI native resume/fork behave as in `v0.4.0` | **PASS** | `resume claude:x`/`resume codex:x` (bogus ids) both take the session-lookup path (`session not found`, exit `2`), confirming native resume/fork remain wired for both, consistent with the passing regression suite |
| G2 | Claude→Codex and Codex→Claude structured handoff both complete | **PASS** | Ran both directions for real (not `--dry-run`) with `--no-launch --json` from each source session's own real workspace: `claude:<real id> --to codex` → exit 0, `"destination_session_mode":"new"`, capsule stored, `projection.md` written to disk; `codex:<real id> --to claude` → exit 0, same shape, destination `session_id` minted. Neither session/title/path is quoted here (not created by this run) |
| G3 | Gemini/OpenCode/Grok remain handoff sources; Gemini stays read-only | **PASS**, with a note | `resume gemini:x` refused `unsupported…T2`, exit `5` (Gemini remains read-only, as written); `handoff --from gemini --last --to claude --dry-run` engaged the real Gemini source machinery (refused only on a workspace-mismatch check, a correct refusal per the contract's "reading a refusal correctly" list, not an "unsupported agent" rejection). **Note:** `grok` is now T4 and `resume grok:x` takes the session-lookup path (not "unsupported"), i.e. Grok itself no longer reads as read-only-only — this matches its already-published T4 census tier and is not a new regression, but it means G3's literal "Grok remains read-only" clause is stale, in the same way the contract itself already flags for OpenCode's T5 promotion |
| G4 | `rein push`/`pull` carry Claude and Codex sessions, and no other agent's | **FAIL (stale contract text, not a new regression)** | `rein __complete push --agent ""` and `rein __complete pull --agent ""` both return exactly `claude`, `codex`, `opencode` (never any other agent). Traced to `internal/cli/commands_impl.go`'s `defaultRegistry()`, which builds the push/pull agent set from `agents.Capable(agents.CapabilitySync)` — OpenCode's catalog descriptor now declares a sync adapter. A real `push --agent opencode --dry-run` could not be driven to completion without Hop login (`config missing`, exit 3), so this is a completion/registry-level finding, not a confirmed data-plane push. Flagged for the maintainer to reconcile G4's wording with G3's OpenCode note, not scored as a regression this candidate introduced |
| G5 | Path remapping across macOS and Windows is unchanged | **PASS** | `internal/pathmap` package tests passed in the A2 unit suite (pure remap-logic tests, run without needing physical macOS hardware); no change to this package in this candidate |
| G6 | Existing exit-code semantics unchanged; no new exit codes | **PASS** | Every exit code observed across this part's rows (`0`, `1`(none hit), `2`, `3`, `5`, `6`(none hit), `7`(none hit)) matches `docs/cli-reference.md`'s documented table, itself asserted against the shipped binary by the A5 doctest gate |
| G7 | An index built by an older release upgrades without data loss, or migrates with an explicit message | **PASS** | Built a real index with the checksum-verified `v0.5.1` Windows binary in a fresh isolated `REINSTATE_HOME` (100+ real sessions across all installed agents), then opened the **same** home with the `v0.6.0-rc.4` binary. At the default `--limit 100` a naive single-shot diff showed "1 lost / 1 new," which further investigation showed was **only** the default-100-row listing window shifting by one live session created on the host during the test — re-run with `--limit 5000` (uncapped) on both sides: **530/530 v0.5.1 sessions present after the rc.4 open, 0 lost, +1 new** (a real session created live on the host during the test window). No data loss from the upgrade itself |
| G8 | Capsules remain excluded from sync | **PASS** | `TestRefuseHandoffsPush` (`internal/sync/sync_test.go`) passed in the A2 suite |

**Matrix G: 7/8 PASS, 1 FAIL (G4 — stale contract text; see findings below).**

## Phase 5 Matrix H — CLI contract and performance (6 rows, required)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | `docs/cli-reference.md` matches the shipped flags | **PASS** | Covered by the A5 doctest gate (`internal/doctest`), which asserts this directly; re-ran standalone, `ok` |
| H2 | `--json` output for `sessions`, `search`, `inspect`, `doctor` matches its documented shape | **PASS** | Exercised all four live: `sessions --json` → `{"sessions":[...]}`; `search "the" --agent claude --json` → `{"sessions":[...]}` (1 real match); `inspect claude:<id> --json` → `{"session":{...},"environment":{...}}`; `doctor --agents --json` → `AGENT-PROBE-V1` shape (Matrix B). All well-formed, all exit 0 |
| H3 | Full index refresh across all installed agents completes within the documented ceiling | **PASS** | Cold full refresh (`sessions --agent all --limit 5000 --json`, fresh isolated `REINSTATE_HOME`, 11 installed agents, 530+ real sessions): **14.2s**, well inside the ~20–30s range `docs/testing/phase-3-cli-performance.md`'s run notes reference for the original (unregressed) Windows cold-refresh observation |
| H4 | Refresh with no changes is materially faster than a cold refresh | **PASS** | Same home, immediately repeated: **0.27s** warm vs **14.2s** cold — ~52× faster |
| H5 | A single unreachable/slow agent source does not block the others | **PASS** | Pointed `QWEN_HOME` at an unreachable UNC path (`\\nonexistent-host-xyz-9182\share\qwen`) and ran a full refresh: exit 0, **15.6s** (comparable to the unmodified cold-refresh baseline, not hung), 522 sessions returned across the 10 other installed agents (`qwen` correctly absent, nothing else blocked) |
| H6 | Shell completion lists the correct agent keys per command | **PASS** | Exercised directly via Cobra's hidden completion path: `rein __complete push --agent ""` / `pull --agent ""` → exactly `claude, codex, opencode` (deterministic, both commands identical — see the G4 finding above for what that list itself implies); `rein __complete sessions --agent ""` → the wider, correct T1+ list (`all, claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen`) with no T0 key present in either case |

**Matrix H: 6/6 PASS.**

## T0 agents — Matrix F (F1/F2), 7 agents × 2 rows = 14 rows (required)

All 7 T0 agents (`aider`, `amp`, `antigravity`, `minimax-code`, `openhands`,
`roo`, `zcode`) are **not installed** on this host (`executable_on_path:
false`, `resolved_root: null` for all 7) — a scope reduction per the Phase 5
contract, not a `FAIL`. Neither row required installing one.

| Agent | F1 (`doctor --agents` lists it with a reason) | F2 (no command offers a capability; none implies one is coming) |
| ----- | ---------------------------------------------- | ----------------------------------------------------------------- |
| `aider` | **PASS** — `t0_reason=layout_unverified` | **PASS** — `resume aider:x` refused `unsupported…T0 (layout_unverified)` exit `5`; `sessions --agent aider` and `handoff --from aider` both refused `invalid agent…` exit `2` (not offered in the accepted-key lists at all) |
| `amp` | **PASS** — `t0_reason=server_backed` | **PASS** — same three-command pattern, `t0_reason=server_backed` in the resume refusal |
| `antigravity` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same pattern |
| `minimax-code` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same pattern |
| `openhands` | **PASS** — `t0_reason=server_backed` | **PASS** — same pattern |
| `roo` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same pattern |
| `zcode` | **PASS** — `t0_reason=desktop_only` | **PASS** — same pattern |

**T0 F1/F2: 14/14 PASS.**

## T1 agents — Matrix C (C1–C6), 4 agents × 6 rows = 24 rows (required)

C1–C4 for every agent below ran read-only through `rein` alone against this
host's real, live agent roots, per the evidence policy for T1 discovery rows.
C6 ran against **copies** of each real tree (never the live data); corrupted
copies were discarded after the test.

### `cline` (7 real sessions on this host, 5 distinct projects)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | **PASS** | `sessions --agent cline --json`: 7 sessions, 5 distinct projects |
| C2 | **PASS** | `sessions`/`inspect` fields present and populated: `project`, `title`, `updated_at`, `message_count` (no `branch` field for this agent — not applicable to its store shape) |
| C3 | **PASS** | Probed `cline --cwd <dir> --json "Reply PONG"` with the live host config: answered "PONG" (live credential works). Created a fresh session with a planted, unique token via the same live `cline` binary in a throwaway project; `rein search <token> --agent cline --json` found exactly 1 session (the one just created) |
| C4 | **PASS** | `inspect cline:<id> --json` → bounded `session`/`environment` object; fields are metadata only (`prompt_preview` is a short preview string, no message array/transcript body) |
| C5 | **PASS** | `resume cline:<id> --dry-run` refused `unsupported: Cline sessions are read-only until a device journey verifies native resume`, exit `5`; every listed session carries a populated `read_only_reason` |
| C6 | **PASS** | Copied `~/.cline/data` (db + sessions), truncated the last 500 bytes off `sessions.db`: `sessions --agent cline --json` against the copy completed within 30s, all 9 real+synthetic sessions in the copy still enumerated, no panic, no error. Empty-root and absent-root redirects (`CLINE_DATA_DIR`) both returned cleanly (`root=no`, exit 0, no error) |

### `copilot` (2 real sessions, 2 distinct projects)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | **PASS** | `sessions --agent copilot --json`: 2 sessions, 2 distinct projects |
| C2 | **PASS** | Fields present: `project`, `branch`, `title`, `updated_at`, `message_count` |
| C3 | **PASS** | Read one real session's `prompt_preview`, picked a plain word from it (not quoted here), `rein search "<word>" --agent copilot --json` → found 1 |
| C4 | **PASS** | `inspect copilot:<id> --json` → bounded metadata only, `prompt_preview` short, no transcript array |
| C5 | **PASS** | `resume copilot:<id> --dry-run` refused `unsupported: GitHub Copilot CLI sessions are read-only…`, exit `5` |
| C6 | **PASS** | Copied `~/.copilot`, truncated the last 500 bytes off two `session.db` files: `sessions --agent copilot --json` against the copy completed within 30s, sessions still enumerated, no panic/error. Empty-root and absent-root redirects (`COPILOT_HOME`) both clean |

### `cursor` (2 real sessions, 2 distinct projects — real Cursor CLI store; `cursor-agent` itself is broken on this host, `Cannot find module tree-sitter`, so no fresh session could be created; every row below ran through `rein` alone, read-only)

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | **PASS** | `sessions --agent cursor --json`: 2 sessions, 2 distinct projects |
| C2 | **PASS** | `message_count` values only, per the dispatch's redaction rule: **3, 3** |
| C3 | **PASS** | Read one real session's `prompt_preview` myself, picked a word from it, `rein search "<word>" --agent cursor --json` → **found 1**. Neither the word, the preview, a title, nor an id is quoted here |
| C4 | **PASS** | `inspect cursor:<id> --json` → bounded metadata (`session`/`environment`), no transcript body |
| C5 | **PASS** | `resume cursor:<id> --dry-run` refused `unsupported: Cursor CLI sessions are read-only…`, exit `5` |
| C6 | **PASS** | (CURSOR_CONFIG_DIR redirect method.) Copied `~/.cursor/chats` only, truncated the last 500 bytes off both real `store.db` files. Ran `rein sessions --agent cursor --json` against the copy via `CURSOR_CONFIG_DIR`, bounded to a 30s job timeout: completed well within the bound, both sessions still returned with correct `message_count` (3, 3), no panic, no error. (A first attempt that copied the *entire* `~/.cursor` tree — including a ~700KB Statsig cache file — via a plain `cp -r` appeared to hang past 120s; re-running against a narrower, faster copy showed this was the copy step itself being slow on this host, not `rein` — noted so it isn't mistaken for a product hang.) Empty-root and absent-root redirects both returned `{"sessions":[]}`, exit 0, no error |

### `pi` (7 real sessions before this run, 8 after; 6+ distinct projects) — **the dispatch's specific re-test**

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | **PASS** | `sessions --agent pi --json`: 7 pre-existing sessions, 6 distinct projects |
| C2 | **PASS** | Fields present: `project`, `title`, `updated_at`, `message_count` |
| C3 | **PASS — confirms the `F-PI-CONTENT-EXTRACTION` fix** | Created a fresh session with `pi -p "<prompt containing a planted, unique token>"` in a throwaway git project. Confirmed the raw session file on disk is real `version:3` shape (`"version":3` at the top level; `"content":[{"type":"text","text":"…"}]` parts, not a legacy top-level `"text"` key) and contains the token. `rein search <token> --agent pi --json` → found exactly 1 (the session just created). `rein inspect pi:<id> --json` → `prompt_preview` is **populated** (59 characters) and **contains the planted token** — not empty, not null. This is the opposite of the `v0.6.0-rc.3` tagged result (search found nothing, no `prompt_preview`) |
| C4 | **PASS** | `inspect pi:<id> --json` → bounded metadata only |
| C5 | **PASS** | `resume pi:<id> --dry-run` refused `unsupported: Pi is tier T1; native resume is unsupported` / `…sessions are read-only…`, exit `5` |
| C6 | **PASS** | Copied `~/.pi/agent`, truncated the last ~20 bytes off the newly created session's `.jsonl` (torn last JSON line): `sessions --agent pi --json` against the copy still enumerated all 8 sessions (including the truncated one), no panic, no error, exit 0. Empty-root and absent-root redirects (`PI_CODING_AGENT_DIR`) both clean |

**T1 C1–C6: 24/24 PASS.**

## Verdict for this part

- **Rows in this part:** 8 (section A gates) + 33 (Matrix A/B/G/H core) + 14
  (T0 F1/F2) + 24 (T1 C1–C6) = **79**
- **Section A gates:** 8/8 PASS
- **Matrix A:** 10/10 PASS
- **Matrix B:** 7/9 PASS — **2 FAIL** (`B4`, `B7`, both real product findings, neither release-blocking on their own; see below)
- **Matrix G:** 7/8 PASS — **1 FAIL** (`G4`, stale contract text describing a pre-existing OpenCode sync capability, not a regression introduced by this candidate; see below)
- **Matrix H:** 6/6 PASS
- **T0 F1/F2:** 14/14 PASS
- **T1 C1–C6:** 24/24 PASS, including the dispatch's specific re-test `pi:C3` — **confirmed fixed**
- **Required-row PASS count for this part:** 76/79 (3 non-PASS rows, all documented below with reproduction evidence; none is a `NOT TESTED` or harness gap)

## Findings from this part

1. **`B4` FAIL — product, probe redaction gap.** The `gemini` agent's probed
   `tree` includes two top-level directory names under `tmp/` that are real,
   un-normalized project-name strings copied verbatim from this host, while
   every path segment beneath them is correctly shape-normalized. This is a
   real string reaching output the contract requires to be free of "raw …
   path slug"s. Not release-blocking by itself (the rest of the redaction
   surface — B1/B2/B3/B5/B6/B9 — held), but worth a fix before any report
   from this host's probe could be shared externally.
2. **`B7` FAIL — product, correctness gap.** An overridden agent root that
   exists but is empty and one that does not exist at all produce
   byte-identical `doctor --agents --json`/human output (apart from the
   timestamp) — not distinguishable as the contract requires. Reproduced
   twice on `cline` via `CLINE_DATA_DIR`.
3. **`G4` FAIL — stale contract text, not a new regression.** `push`/`pull`
   agent completion (and the underlying `defaultRegistry()`) already include
   `opencode` alongside `claude`/`codex`. This tracks OpenCode's existing T5
   promotion (the same promotion Matrix G3's own carried note already
   acknowledges) rather than anything this candidate changed; flagged so the
   maintainer can reconcile G4's and G3's wording rather than leaving a
   silent mismatch between the written contract and the shipped registry. A
   full data-plane `push --agent opencode` could not be driven to completion
   in this part without a Hop login (out of this part's scope).
4. **Harness note, not scored.** A naive `push --agent opencode --dry-run`
   probe returned `config missing` (exit `3`, no Hop profile configured in
   this shell) before reaching agent validation — this blocked confirming
   whether a real opencode push would actually transfer data, so the G4
   finding above is a completion/registry-level finding only, not a
   confirmed data-plane regression.
5. **Harness trap avoided (recorded so it isn't repeated).** A first
   `cursor:C6` attempt that copied the *entire* `~/.cursor` tree (including
   a large Statsig telemetry cache file) via `cp -r` through Git Bash on this
   host appeared to hang past 120 seconds. Re-running the same corruption
   test with a narrower copy (`chats/` only) and a bounded PowerShell job
   completed in well under 30 seconds with a clean result — the slow step was
   the file copy on this host, not `rein`. Recorded so a future run does not
   mistake this for a product hang.

No panics, crashes, or unbounded hangs were observed in `rein` itself across
any row in this part.

## Cleanup

All throwaway/isolated directories used for corruption, empty-root, and
absent-root probes in this part
(`D:\ReinstateAcceptanceProjects\v060-rc4-a\c6\*-copy*`,
`empty-cline`, `pi-probe-dir`, `cline-probe-dir`) held only copies made by
this run or newly created synthetic probe sessions; none was committed. No
real host agent data was modified — every corruption/empty/absent test ran
against a copy, never the live root.

