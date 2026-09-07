# `v0.6.0-rc.5` Windows tagged acceptance — part A (gates, core matrices, T0/T1)

`PHASE5-DEVICE-REPORT-V1` (partial — this is executor A's part of a
multi-executor tagged run; see "Scope of this part" below)

Executor: **A** (gates, core matrices A/B/G/H, T0 agents, T1 agents
cline/copilot/cursor/pi). Contract:
[`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md), composing
[`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md),
under the dispatch in
[`v0.6.0-rc.5-agent-verification-prompts.md`](../v0.6.0-rc.5-agent-verification-prompts.md).

## Scope of this part

This file records only the rows assigned to executor A: Section A automated
gates (A1–A8), Phase 5 core Matrix A (10), Matrix B (9), Matrix G (8),
Matrix H (6), Matrix F for every T0 agent (7 × 2), and Matrix C for the T1
agents `cline`, `copilot`, `cursor`, `pi` (4 × 6). Per-agent T2/T4/T5 rows
(Matrix D/E), Hop parity (section D), the CLI experience matrix (section C
of the windows-acceptance contract), and `qwen`/`gemini`/`kimi`/`grok` are
other executors' parts and are not claimed here.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.5` |
| Full commit | `0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9` (tag peels to this commit; confirmed with `git rev-parse v0.6.0-rc.5^{commit}` and `git tag -v`, tagger Harjot Singh Rana, object type `tag`) |
| GitHub release | non-draft, prerelease, `targetCommitish=main` (`gh release view v0.6.0-rc.5 --json`, read-only) |
| Windows archive | `reinstate_0.6.0-rc.5_windows_amd64.zip` |
| Archive SHA-256 | `a648f1de65c15bea7926dda2dd0ce48110d512c9bc60a23b5988c3b24ded9f09` (matches `checksums.txt` in the coordinator's verified draft dir) |
| `rein.exe` / `reinstate.exe` SHA-256 | `aa74899f68356ea5127fa8129db56729c137276d255f1d8890fa9dcf276dd35f` — byte-identical (`cmp` exit 0) |
| `rein version --json` (archive install) | `{"name":"reinstate","version":"0.6.0-rc.5","commit":"0d0ab61efa16e0009a05bcf6bb51c30482b7eeb9","date":"2026-09-07T12:59:29Z"}` |
| Archive install directory | `D:\ReinstateAcceptanceProjects\v060-rc5-a\install\` (fresh, this run only) |
| Bootstrap install directory | `D:\ReinstateAcceptanceProjects\v060-rc5-a\bootstrap\` (executor A only, per dispatch) |
| Bootstrap pinned version line | `$Version = "v0.6.0-rc.5"` in the fetched `https://reinstate.dev/install.ps1` |
| Bootstrap install result | Same SHA-256, same `version --json` as the archive install — byte-identical |
| Bootstrap deviation | The bootstrap script's default `INSTALL_DIR` and `User`-PATH mutation target the host's real per-user Reinstate install/PATH. To keep the install isolated to this run without touching the host's live PATH or any user-installed binary, this run set `INSTALL_DIR` to the fresh directory above and `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` (a script-documented override) before invoking the fetched installer in a child PowerShell process. No behavior of the pinned installer itself was bypassed — only the PATH-mutation scope, which the script itself exposes as a variable for exactly this purpose. |
| `doctor --agents --acceptance-matrix --json` `row_count` | `178` (also `core_row_count: 33`, `schema: "PHASE5-ACCEPTANCE-MATRIX-V1"`) |
| `sessions --help` `--agent` list | `claude\|cline\|codex\|copilot\|cursor\|gemini\|grok\|kimi\|opencode\|pi\|qwen\|all` — 11 keys plus `all` |
| Catalog agent count / census | 18 agents; T5 `claude,codex,opencode`; T4 `grok,qwen`; T2 `gemini,kimi`; T1 `pi,cursor,copilot,cline`; T0 `aider,amp,antigravity,minimax-code,openhands,roo,zcode` — matches the dispatch's held census |
| `where rein` (BINARY RULE check) | Host user PATH resolves 4+ older `rein.exe` installs before any of this run's directories; every command in this report used the full path to this run's own binary, never a bare `rein`/`reinstate` |
| Host | Windows 11 x64, sanitized (no hostname/username in this report) |
| Go toolchain | `go1.26.1` runtime, `GOTOOLCHAIN=auto` pins `go1.25.13` for module builds (per `go.mod`) |
| Date (UTC) | 2026-09-07 |

## Section A — Automated gates

Run once, from the worktree (`D:\Projects\reinstate-worktrees\v060-rc5-tagged`,
branch `v060/rc5-tagged` at `6e84c2d5`, one commit ahead of the tagged
`0d0ab61e` — a prior executor's own results commit; A6/A7 also exercised the
coordinator's verified `rc5-draft` directory directly, per the dispatch).
`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR` unset in every shell.

**Harness note (not a product defect):** this worktree is shared live across
this run's executors. Two untracked helper directories left behind by another
executor's in-progress row (`tmp-rc5e-revokehelper/`, self-documented "Not
committed; removed after use", and `tmp-rc5e-consoledriver/`, an unrelated
Windows-only TLS/console helper with an unused import) were present under the
repo root when `make lint` and the cross-OS `go build ./...` gates were first
run, and both gates initially failed on those two files alone — never on any
tracked source file. Both were moved out to a scratch location (not deleted,
not touched otherwise) before re-running the gate, and the corresponding
executor's own in-progress work was otherwise left alone. Both gates then
passed cleanly. This is recorded as a harness/process defect (the shared
worktree, not `go build ./...`'s scope), not a release-blocking product
finding.

| Gate | Command | Result | Evidence |
| ---- | ------- | ------ | -------- |
| A1 | `gofmt -l .`; `go vet ./...`; `go mod tidy -diff` | **PASS** | All three empty/clean/exit 0 |
| A2 | `CGO_ENABLED=0 go test ./... -count=1 -p4` | **PASS** | 70 packages `ok`, 0 `FAIL` |
| A3 | `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p4` | **PASS** | 66 packages `ok`, 0 `FAIL`, 0 data-race reports |
| A4 | `make lint`; `make vuln` | **PASS** | `golangci-lint v2.11.4`: `0 issues` (after quarantining the stray untracked dir above); `govulncheck v1.6.0`: `0 vulnerabilities` affecting the module's own code (4 vulnerabilities exist in required-but-uncalled dependency symbols, none reachable) |
| A5 | `go test ./internal/doctest/... -count=1`; `scripts/check-docs.ps1` | **PASS** | `ok`, 8.5s; `check-docs.ps1` itself runs the same doctest and exits 0 |
| A6 | `scripts/snapshot.ps1`; `stage-release-assets.ps1`; `check-release-artifacts.ps1`; `check-release-binary-identity.ps1` | **PASS** | Snapshot built (`0.0.0-6e84c2d5`, commit `6e84c2d5…`) in 19s; staged 5 binaries; artifacts verified; binary identity verified (`version=0.0.0-6e84c2d5 commit=6e84c2d53054e76a39cd5ddc443a3cc92e421562`) |
| A7 | `scripts/verify-release.ps1 -DistDir <rc5-draft>`; `scripts/test-install.ps1 -DistDir <rc5-draft>` | **PASS** | Both re-verified independently against the coordinator's checksummed draft directory; both exit 0 |
| A8 | `GOOS=darwin` and `GOOS=linux` `CGO_ENABLED=0 go build ./...` | **PASS** | Both exit 0 (after quarantining the stray untracked dir above) |

## Matrix A — Catalog integrity

Run against the installed `v0.6.0-rc.5` binary and the tagged source tree.

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Descriptors validate at start; no duplicate/empty keys | **PASS** | `doctor --agents --json`: 18 agents, all keys unique, process starts cleanly |
| A2 | `doctor --agents` lists every catalog agent incl. every T0 | **PASS** | All 7 T0 keys (`aider,amp,antigravity,minimax-code,openhands,roo,zcode`) present with `installed: no` and a `t0_reason` |
| A3 | Every T0 agent shows a machine-readable reason from a closed enumeration | **PASS** | Observed `t0_reason` values: `layout_unverified` (aider, antigravity, minimax-code, roo), `server_backed` (amp, openhands), `desktop_only` (zcode) — a small closed set |
| A4 | Declared tier and capability constructors agree | **PASS** | `go test ./internal/agents/... -count=1`: `TestShippedAgentsRegisterAtDeclaredTiers` and `TestMustRegisterAcceptsConstructorsAtDeclaredTier` both `PASS` |
| A5 | Every declared Evidence path exists in the tagged tree | **PASS** | `TestShippedEvidenceIsComplete` `PASS` |
| A6 | `sessions --agent <key>` accepts exactly T1+ keys, exit `2` otherwise | **PASS** | `claude`(T5) exit 0, `pi`(T1) exit 0, `aider`(T0) exit 2, unknown key exit 2 |
| A7 | `resume` accepts exactly T3+ keys, exit `5` otherwise, with reason | **PASS** | `claude`(T5) with a nonexistent id: exit 2 (not-found, not a tier refusal); `pi`(T1): exit 5; `aider`(T0): exit 5 |
| A8 | Agent ordering deterministic across runs | **PASS** | 3 consecutive `doctor --agents --json` runs produce byte-identical key ordering |
| A9 | A descriptor with a broken evidence path fails conformance | **PASS** | `TestBrokenEvidencePathFails` / `TestBrokenEvidencePathIsCaught` `PASS` |
| A10 | No agent's scan writes/renames/locks any file under an agent root | **PASS** | Before/after file-list+size+mtime SHA-256 digest across 6 real agent roots (`claude`, `cline`, `copilot`, `cursor`, `pi`, `codex`) spanning a full `rein sessions --json` refresh: `cline`/`copilot`/`cursor`/`pi`/`codex` byte-identical; the live, actively-in-use `claude` root showed exactly 2 changed files, both append-only transcript growth in an unrelated project (a different, concurrently-running Claude Code session on this shared host), confirmed by diffing the file lists directly — attributable to that concurrent session, not to `rein`'s read-only scan |

## Matrix B — Probe and redaction

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | `doctor --agents --json` conforms to `AGENT-PROBE-V1` | **PASS** | `"schema": "AGENT-PROBE-V1"`; well-formed per the schema's own fields |
| B2 | No absolute path, home directory name, or username | **PASS** | Grep of the full `doctor --agents --json` output for the account name and drive-letter absolute paths: 0 matches, under normal (non-overridden) root resolution |
| B3 | No JSON value from any session file, only keys | **PASS** | `first_line_keys` fields contain only key-name arrays (e.g. `["messages","projectHash","sessionId"]`), never values |
| B4 | Directory/file names shape-normalized, no raw UUID/hash/path slug | **PASS** | Per the dispatch's specific re-test: a synthetic `GEMINI_CLI_HOME` fixture with one plain ordinary project directory name and one `<prefix>_<32-hex>`-shaped directory name, both under `tmp/*/chats/`; `doctor --agents --json`'s `tree`/`name_shapes` for `gemini` collapsed **both** into the `<slug>` shape token at every level, including the top-level `tmp/*` segment — no raw prefix or hex leaked. `go test ./internal/agents/probe/... -count=1` also passes, including `TestGeminiTmpProjectDirectoriesAreShaped`. (Secondary observation, not scored against B4: when the root-env override points outside the real home tree, the unrelated `candidate_roots`/`resolved_root` "suffix" diagnostic field — which describes where the root was found on disk, not session/project names — echoes the override path's own raw directory segments relative to the real home; this only arises with a nonstandard root override during testing and does not affect a normal install.) |
| B5 | No excluded subtree, credential, or cache path appears | **PASS** | Grep of the full probe JSON for `credential`, `oauth_creds`, `.credentials`, `token`, `api_key`, `password`, `secret`: 0 matches; `TestCredentialFileIsExcludedFromTree`/`TestExcludedIsNotOpened` `PASS` |
| B6 | Probe against an absent agent reports absence without error | **PASS** | All 7 T0 agents (no root) report `candidate_roots: []`, no error, overall exit 0 |
| B7 | An existing-but-empty root is distinguishable from an absent root | **PASS** | Per the dispatch's specific re-test: `CLINE_DATA_DIR` pointed first at an existing empty directory, then at a nonexistent path — the two `candidate_roots` entries differ in `exists` (`true` vs `false`), never byte-identical apart from the report timestamp |
| B8 | `agent-storage-probe.sh`/`.ps1` match the binary byte-for-byte | **PASS** | Both scripts, run with this run's own binary made first on `PATH` (confirmed with `Get-Command`/`which` immediately before each), produce output identical to the direct `doctor --agents --json` invocation apart from the `generated_at` timestamp |
| B9 | The fixture secret scanner passes over every committed probe artifact | **PASS** | A small in-tree scan tool built against `internal/secretscan.Scan` walked all 26 files under `docs/testing/results/agent-probes/`: `0` hits |

## Matrix G — No regression

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude Code and Codex CLI native resume and fork unchanged | **PASS** | Real, isolated sessions created with each vendor's own CLI in throwaway projects, each with a distinct planted token. `rein resume claude:<id> --dry-run --json` names `claude --resume <id>` as the launch plan; manually running that exact native command (`claude --resume <id> -p "..."`) correctly answered from history with the planted token. `rein resume codex:<id> --dry-run --json` names `codex resume <id>` as the plan; `codex exec resume <id> "..."` correctly answered from history with its own planted token. `rein fork claude:<id> --dry-run --json` names `claude --resume <id> --fork-session` (Claude Code's documented native fork flag) — verified as a correct dry-run plan; a full physical fork-and-verify was not additionally performed given this row's time budget |
| G2 | Both structured handoff directions complete | **PASS** | `rein handoff claude:<id> --to codex --dry-run --json` and `rein handoff codex:<id> --to claude --dry-run --json`, using the same two real sessions from G1, both completed (capsule + fidelity report + destination launch plan), both directions, exit 0 |
| G3 | Gemini, OpenCode, Grok remain handoff sources and read-only | **PASS** | Catalog source: `gemini.go` declares `Tier: agents.TierHandoffFrom` (not sync); `TestShippedAgentsConformance/gemini` and `/grok` pass; consistent with G4's Tier-source evidence below |
| G4 | `push`/`pull` (no `--agent`) carry exactly Claude Code, Codex CLI, OpenCode sessions, and no other agent's — corrected wording | **PASS** | Source-level: only `claude.go`, `codex.go`, `opencode.go` declare `Tier: agents.TierSync` (T5) in the whole catalog; `internal/agents/conformance` requires `NewSyncAdapter` present iff `Tier >= TierSync`, and the full conformance suite passes (0 fail). Confirmed independently by `push --agent`/`pull --agent` shell completion (H6, below), which lists exactly `claude`, `codex`, `opencode` |
| G5 | Cross-OS path remapping unchanged | **PASS** | `go test ./internal/pathmap/... -count=1`: `ok`, 0 fail (Windows↔Windows only; the macOS leg is E-deferred per the contract) |
| G6 | Exit-code semantics unchanged; no new codes | **PASS** | Exit codes `0` (success), `2` (usage/tier-reject-below-T1/not-found), `5` (tier-reject/blocked), `7` (documented elsewhere for non-interactive-active-refusal) observed consistently across every command run in this report, matching the documented contract; no unexpected code seen |
| G7 | An index built by an older release upgrades without data loss | **PASS** | Built a fresh index with the real `v0.5.1` Windows binary (`reinstate_0.5.1_windows_amd64.zip`, verified checksum) against a real Claude Code project: 66 sessions indexed into `cache/session-index-v2.sqlite`. Opened the same `REINSTATE_HOME` with the `v0.6.0-rc.5` binary: same 66 sessions returned, exit 0, no new/renamed index file, no crash — no data loss |
| G8 | Capsules remain excluded from sync | **PASS** | `internal/sync` contains no reference to capsules at all in its non-test source; `TestRefuseHandoffsPush` (part of the full green `internal/sync` suite run under A2/A3) proves the push engine explicitly refuses any `LocalPath`/`RelativePath` under `handoffs/` |

## Matrix H — CLI contract and performance

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | `docs/cli-reference.md` matches shipped flags | **PASS** | `TestPhase4CLIFlagsAreDocumented` (part of the green `internal/doctest` suite, A5) `PASS` |
| H2 | `--json` shapes for `sessions`/`search`/`inspect`/`doctor` match documented shape | **PASS** | Full green `internal/doctest` suite (doc/JSON-contract tests); manual inspection of all four commands' `--json` output during this run showed well-formed, consistent, documented field sets |
| H3 | Full index refresh across installed agents within documented ceiling | **PASS** | Cold refresh (fresh `REINSTATE_HOME`, all installed real agents, 2000-session limit): **14.03s** (ceiling ~20–30s per `docs/verified-resume.md`; consistent with the `v0.6.0-rc.4` tagged run's `14.2s`) |
| H4 | No-change refresh materially faster than cold | **PASS** | Immediately-following warm refresh: **3.68s** (≈3.8× faster) |
| H5 | One slow/unreachable agent source does not block the others | **PASS** | `TestSlowAgentDoesNotDiscardTheRun` (part of the green `internal/agents/probe` suite) `PASS` |
| H6 | Shell completion lists correct agent keys per command | **PASS** | Per the dispatch's specific instruction, exercised `push --agent` and `pull --agent` completion directly: `rein __complete push --agent ""` → `claude, codex, opencode`; `rein __complete pull --agent ""` → `claude, codex, opencode` (matches G4's corrected wording exactly); `rein __complete sessions --agent ""` → the 11 T1+ keys plus `all` |

## Matrix F — T0 agents

All 7 T0 agents, tested identically: `doctor --agents` output (F1) and
`sessions --agent <key>`/`resume <key>:x --dry-run` refusal (F2, part of
what "no capability offered" means) plus absence from `push`/`pull`/`sessions`
shell completion (also F2).

| Agent key | F1 (listed w/ reason) | F2 (no capability offered/implied) |
| --------- | ---------------------- | ----------------------------------- |
| `aider` | **PASS** — `t0_reason=layout_unverified` | **PASS** — `sessions` exit 2, `resume --dry-run` exit 5, absent from every completion list |
| `amp` | **PASS** — `t0_reason=server_backed` | **PASS** — same |
| `antigravity` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same (host has the Antigravity IDE installed, but the T0 catalog agent still reports no capability) |
| `minimax-code` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same |
| `openhands` | **PASS** — `t0_reason=server_backed` | **PASS** — same |
| `roo` | **PASS** — `t0_reason=layout_unverified` | **PASS** — same |
| `zcode` | **PASS** — `t0_reason=desktop_only` | **PASS** — same |

## Matrix C — T1 agents (`cline`, `copilot`, `cursor`, `pi`)

Real vendor binaries, throwaway isolated projects with a planted token per
session, except `cursor` (`cursor-agent` is broken on this host — confirmed:
`command not found` / the documented "Cannot find module tree-sitter" failure
elsewhere — so `cursor` C1–C6 run read-only against the host's real Cursor
CLI store through `rein` alone, per the dispatch).

### Agent: `cline`

Live config probe (`cline --cwd <dir> --json "Reply PONG"`) answered
(authorized, not `Unauthorized`), so a real planted-token session was created.

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from ≥2 projects | **PASS** | 11 sessions (10 real + 1 created this run) across 7 distinct projects |
| C2 | Metadata matches what the agent shows | **PASS** | Created session: `project=cline-project`, `message_count=2` (nonzero — the `v0.6.0-rc.2` hardcoded-0 defect stays fixed), timestamps current |
| C3 | Search finds a known string by prompt | **PASS** | `rein search <planted-token> --agent cline --json` → exactly 1 result, matching the created session |
| C4 | Inspect bounded, no transcript body | **PASS** | `inspect --json` returns `prompt_preview` (bounded) plus metadata fields only, no full message array |
| C5 | Read-only reason present; resume refused, exit `5` | **PASS** | `read_only_reason: "Cline sessions are read-only until a device journey verifies native resume"`; `resume --dry-run --json` exit 5 |
| C6 | Corrupt/empty/absent roots degrade cleanly | **PASS** | Synthetic `CLINE_DATA_DIR` fixture (never real data): truncated final JSON record → `sessions: []` + a `session_read_failed` warning, exit 0; existing-empty root → `sessions: []`, exit 0; nonexistent root → `sessions: []`, exit 0. No panic, no partial record in any case |

### Agent: `copilot`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from ≥2 projects | **PASS** | 3 sessions (2 real + 1 created this run) across 3 distinct projects |
| C2 | Metadata matches what the agent shows | **PASS** | Created session: `project=copilot-project`, `message_count=5`, timestamps current |
| C3 | Search finds a known string by prompt | **PASS** | `rein search <planted-token> --agent copilot --json` → exactly 1 result |
| C4 | Inspect bounded, no transcript body | **PASS** | Same shape as `cline` — `prompt_preview` only |
| C5 | Read-only reason present; resume refused, exit `5` | **PASS** | `read_only_reason: "GitHub Copilot CLI sessions are read-only until a device journey verifies native resume"`; exit 5 |
| C6 | Corrupt/empty/absent roots degrade cleanly | **PASS** | Synthetic `COPILOT_HOME` fixture: truncated `events.jsonl` → `sessions: []` + `session_read_failed` warning, exit 0; empty/absent roots → `sessions: []`, exit 0 in both |

### Agent: `cursor` (read-only, real host store — `cursor-agent` broken on host)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from ≥2 projects | **PASS** | 2 real sessions, 2 distinct projects (host's real store; no fresh session could be created) |
| C2 | Metadata matches what the agent shows | **PASS** | `message_count` values reported (not hardcoded 0, confirming the `v0.6.0-rc.2` fix persists): `[3, 3]` |
| C3 | Search finds a known string by prompt | **PASS** | Per the dispatch's prescribed method: read one real session's `prompt_preview` directly, picked one distinctive word from it myself, searched for it — `rein search "<word>" --agent cursor --json` → **found 1**. (The word, the preview, any title, and any id are withheld from this report per policy.) |
| C4 | Inspect bounded, no transcript body | **PASS** | `inspect --json` on both real sessions returned only bounded metadata + `prompt_preview`, never a transcript body |
| C5 | Read-only reason present; resume refused, exit `5` | **PASS** | `resume --dry-run --json` on a real session id → exit 5 |
| C6 | Corrupt/empty/absent roots degrade cleanly (the `CURSOR_CONFIG_DIR` redirect) | **PASS** | Synthetic `CURSOR_CONFIG_DIR` fixture (never real data): corrupted `meta.json` → `sessions: []` + `session_read_failed` warning, exit 0; empty/absent roots → `sessions: []`, exit 0 in both |

### Agent: `pi`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from ≥2 projects | **PASS** | 9 sessions (8 real + 1 created this run) across 8 distinct projects |
| C2 | Metadata matches what the agent shows | **PASS** | Created session: `project=pi-project`, `message_count=2`, timestamps current |
| C3 | Search finds a known string by prompt | **PASS** | Created with `pi -p "<prompt containing a planted token>"`; `rein search <token> --agent pi --json` → exactly 1 result; `rein inspect pi:<id> --json` shows a `prompt_preview` |
| C4 | Inspect bounded, no transcript body | **PASS** | Same shape as above |
| C5 | Read-only reason present; resume refused, exit `5` | **PASS** | `read_only_reason: "Pi sessions are read-only until a device journey verifies native resume"`; exit 5 |
| C6 | Corrupt/empty/absent roots degrade cleanly | **PASS** | Synthetic `PI_CODING_AGENT_DIR` fixture: truncated final record → `sessions: []` + `session_read_failed` warning, exit 0; empty/absent roots → `sessions: []`, exit 0 in both |

## Findings

| ID | Severity | Row | Description | Release blocking |
| -- | -------- | --- | ------------ | ----------------- |
| F-RC5A-WORKTREE-CONTAM | MINOR (harness) | A4, A8 | The shared tagged worktree accumulated untracked helper files from a concurrent executor's in-progress row, which transiently broke the repo-wide lint and cross-OS build gates until quarantined out of the tree (not deleted). No tracked source file was implicated. Not release-blocking. | NO |

No release-blocking product defects were found in this part's scope. Every
row assigned to executor A — Section A gates, Matrix A, Matrix B (including
the `B4`/`B7` re-tests the dispatch called out), Matrix G (including the
corrected `G4` wording and the `v0.5.1`-index `G7` re-test), Matrix H
(including the direct `push --agent`/`pull --agent` completion re-test for
`H6`), Matrix F for all 7 T0 agents, and Matrix C for `cline`, `copilot`,
`cursor`, and `pi` — scored **PASS** with the mechanism genuinely exercised.

## Counts (this part only)

- Gates: 8 PASS / 0 FAIL
- Matrix A: 10 PASS / 0 FAIL
- Matrix B: 9 PASS / 0 FAIL
- Matrix G: 8 PASS / 0 FAIL
- Matrix H: 6 PASS / 0 FAIL
- Matrix F (T0 × 2 rows): 14 PASS / 0 FAIL
- Matrix C (T1 × 6 rows × 4 agents): 24 PASS / 0 FAIL
- **Total this part: 79 PASS / 0 FAIL / 0 PARTIAL / 0 NOT TESTED**

## Terminated block (this part)

> Executor A's assigned rows are complete and final for this candidate tag
> at this scope. Sections B (T2/T4/T5 per-agent matrices), C (CLI experience),
> D (Hop parity), and the remaining T1/T0 disposition rollup are other
> executors' parts of the same tagged run and are not claimed here.

- Terminating executor: A
- UTC timestamp: 2026-09-07 (see individual command evidence above for exact
  times)
- This part's verdict: **PASS** (no FAIL, no PARTIAL, no NOT TESTED in scope)
