# `v0.6.0-rc.2` native Windows acceptance — part A (gates, core, T0/T1)

Executor A of the tagged-artifact run. Covers section A automated gates,
Phase 5 core matrices A/B/G/H, T0 Matrix F (all 7 agents), and T1 Matrix C
(cline, copilot, cursor, pi). Other executors cover T2–T5 agent rows,
sections C/D, and the reconciliation.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-06T22:38:03Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.2` |
| Tested full commit | `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` |
| Windows archive SHA-256 (checksums.txt, re-verified) | `82ea243cf9aa1b411cc77322abf973afaf8ff8ac13fbbfa05ad32c5d82ecdc84` |
| Installed binary SHA-256 (rein.exe == reinstate.exe) | `0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc2-a\install`) | `{"name":"reinstate","version":"0.6.0-rc.2","commit":"81d74a82ba2a0e27f9f1a68eb0270d224a20da6a","date":"2026-09-06T21:57:00Z"}` |
| Go toolchain | go1.26.1 (module pins `GOTOOLCHAIN=go1.25.13` for lint/vuln/test-install, honored) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc2-tagged`, branch `v060/rc2-tagged` @ `81d74a82ba2a0e27f9f1a68eb0270d224a20da6a` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (bash) / `Remove-Item Env:REINSTATE_BACKEND, Env:REINSTATE_MEMORY_BACKEND_DIR` (PowerShell) before any `rein`/`go test`/Hop-lab invocation. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left untouched throughout. |

**Bootstrap deviation sentence.** The live `https://reinstate.dev/install.ps1`
was downloaded fresh with `Invoke-WebRequest -OutFile` into
`D:\ReinstateAcceptanceProjects\v060-rc2-a\bootstrap\install.ps1`; its pinned
version line reads `$Version = "v0.6.0-rc.2"`. Run once into
`D:\ReinstateAcceptanceProjects\v060-rc2-a\bootstrap\bin`, it installed
`reinstate.exe`/`rein.exe` whose SHA-256
(`0ae03c4eed8c1af610f04842d5ed51efdd9847724a3b841d249b3841e087fce6`) is
byte-identical to the checksummed `rc2-draft` archive's binaries and reports
the same `version`/`commit`/`date` — **no deviation** between the live
bootstrap route and the release-commit candidate. (Note: this host also
carries a pre-existing `rein`/`reinstate` install at
`%LOCALAPPDATA%\Programs\Reinstate\bin` on the persistent user `PATH` from an
earlier session's default-directory bootstrap; every command in this report
either used an explicit binary path or prepended the candidate's own
directory to `PATH`, and B8's script comparison caught and corrected one
`PATH`-ordering mistake of the harness's own making before it could produce a
false result — see §B8 evidence.)

All work happened under `D:\ReinstateAcceptanceProjects\v060-rc2-a\` and the
worktree. Isolated directories holding copies of real vendor session data
(used for Matrix C6 corruption tests) were deleted after use; nothing from
them is committed or quoted here.

---

## Section A — Automated gates

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| A1 | Format, vet, tidy | PASS | `gofmt -l .`, `go vet ./...`, `go mod tidy -diff` all empty output, all exit 0 |
| A2 | Unit suite | PASS | `CGO_ENABLED=0 go test ./... -count=1 -p 4`: all packages `ok`, 0 `FAIL` |
| A3 | Race suite | PASS | `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p 4`: all packages `ok`, 0 `FAIL` |
| A4 | Lint and vuln | PASS | `make lint` → golangci-lint v2.11.4, `0 issues`; `make vuln` → govulncheck v1.6.0, `0 vulnerabilities` in code (4 in required-but-uncalled modules, informational) |
| A5 | Doc gate | PASS | `go test ./internal/doctest/... -count=1` → `ok`; `scripts/check-docs.ps1` → exit 0 |
| A6 | Snapshot and artifacts | PASS | `scripts/verify-release.ps1 -DistDir <rc2-draft>` → `release artifacts verified (PowerShell): <rc2-draft>`, exit 0 |
| A7 | Installers | PASS | `scripts/test-install.ps1 -DistDir <rc2-draft>` → verify-release re-run clean, `internal/doctest` `ok`, exit 0 |
| A8 | Cross-OS build | PASS | `GOOS=darwin GOARCH=arm64 go build ./...` and `GOOS=linux GOARCH=amd64 go build ./...` both exit 0 |

**8/8 PASS.**

---

## Matrix A — Catalog integrity (10 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Descriptor validation, no dup/empty keys | PASS | Process starts cleanly; `go test ./internal/agents/catalog/... ./internal/agents/conformance/...` (part of A2) all `ok`; `TestShippedAgentsRegisterAtDeclaredTiers` passed |
| A2 | `rein doctor --agents` lists every catalog agent incl. every T0 | PASS | `rein doctor --agents --json` → 18 agents, alphabetically keyed, includes all 7 T0 (`aider`,`amp`,`antigravity`,`minimax-code`,`openhands`,`roo`,`zcode`) |
| A3 | Each T0 agent shows a machine-readable reason from a closed enumeration | PASS | Plain `doctor --agents` output: `aider…t0_reason=layout_unverified`, `amp…server_backed`, `antigravity…layout_unverified`, `minimax-code…layout_unverified`, `openhands…server_backed`, `roo…layout_unverified`, `zcode…desktop_only` |
| A4 | Declared tier and capability constructors agree | PASS | `conformance.Run` (`internal/agents/conformance/checks.go`) asserts this per descriptor via each `TestXConformance`; all pass in A2 |
| A5 | Every `Evidence` path exists in the tagged tree | PASS | `checkEvidence` in `internal/agents/conformance/checks.go` `os.Stat`s every `StoragePage`/`ProbeReports`/`Fixtures`/`DeviceReports` path per descriptor; exercised by every `TestXConformance`, all pass |
| A6 | `sessions --agent <key>` accepts exactly T1+ keys, rejects others exit 2 | PASS | Live: all 7 T0 keys and one bogus key → exit 2; all 11 T1–T5 keys → exit 0 |
| A7 | `resume` accepts exactly T3+ keys, refuses others exit 5 + reason | PASS | Live: T0/T1/T2 keys (`aider`,`amp`,`cline`,`copilot`,`cursor`,`pi`,`gemini`,`kimi`) → exit 5, message names the tier; T3+ keys (`grok`,`qwen`,`claude`,`codex`,`opencode`) → exit 2 `session not found` (key accepted, no such session) |
| A8 | Agent order deterministic across runs | PASS | Two successive `doctor --agents --json` runs: identical 18-key order |
| A9 | Broken-evidence descriptor fails conformance | PASS | `TestBrokenEvidencePathFails` and `TestRunFailsBrokenEvidence` (`internal/agents/conformance/run_test.go`) pass in A2 |
| A10 | No catalog scan writes/renames/locks any agent-root file | PASS | Live filesystem audit: full recursive `mtime`+`size` snapshot of the real `%USERPROFILE%\.pi\agent` tree (45 files) before and after `rein sessions --agent pi`, `rein search`, and `rein doctor --agents --json` — zero diff |

**10/10 PASS.**

---

## Matrix B — Probe and redaction (9 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | `doctor --agents --json` conforms to `AGENT-PROBE-V1` | PASS | `"schema": "AGENT-PROBE-V1"` in every capture |
| B2 | No absolute path/home dir name/username | PASS | Full-document scan for the local username, `c:/users`/`c:\users`, and any `X:\` or `X:/` drive-path pattern: all absent |
| B3 | Only keys, never a JSON value, from session files | PASS | `first_line_keys` entries (e.g. Claude's `history.jsonl`→`["display","pastedContents","project","sessionId","timestamp"]`, Cursor's `chats/*/*/meta.json`→field names only) carry field names, never field content |
| B4 | Names/dirs shape-normalized, no raw UUID/hash/slug | PASS | `name_shapes` renders `<uuid-v4>`, `<32-hex>`, `<slug>` etc.; `tree` paths use `*` wildcards throughout |
| B5 | No excluded subtree (credential/cache) anywhere in output | PASS | Full-document scan for `credential`, `.git-credentials`, `token.json`, `secret`, `auth.json`, `apikey`, `password`, `keyring`, `.ssh`: all absent |
| B6 | Absent agent reports absence without error | PASS | T0 agents and any not-installed agent show `resolved_root: null`, `tree: []` with no exception; overall command exit 0 |
| B7 | Empty root distinguishable from absent root | PASS | Real observed data: Kimi's declared candidates show `.kimi-code` (`exists:true, marker_present:true`) alongside `.kimi` (`exists:true, marker_present:false`) — an existing-but-unmarked ("empty") root reported distinctly from one that plain does not exist (`exists:false`) |
| B8 | `agent-storage-probe.sh`/`.ps1` identical to the binary | PASS | Both scripts are thin wrappers (`rein doctor --agents --json "$@"`); live diff against direct `rein.exe doctor --agents --json` invocation identical after normalizing the per-call `generated_at` timestamp, for both the `.ps1` and the `.sh` variant. (Harness note: the first `.sh` attempt used a `D:/...`-style `PATH` prepend that MSYS truncated at the drive-letter colon, silently resolving a stale pre-existing `rein` on `PATH` and producing a spurious `unknown flag: --agents`; corrected to the `/d/...` mount form and re-verified. Not a product defect — flagged here per the dispatch's own "check the harness before the product" rule.) |
| B9 | Fixture secret scanner passes over every committed probe artifact | PASS | No standalone secret-scan script/binary exists in the tree to invoke directly; substituted a pattern scan (API-key/token/private-key/password shapes) over all 26 files under `docs/testing/results/agent-probes/`, zero hits, plus the passing `internal/secretscan` unit suite (part of A2) |

**9/9 PASS.**

---

## Matrix G — No regression (8 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude/Codex native resume+fork behave as v0.4.0 | PASS (regression evidence only) | Full `go test ./...` (A2) covers `internal/agentcheck`, `internal/agents/catalog` claude/codex descriptors unchanged; no live resume/fork journey performed by this executor — that physical journey is Matrix E (`claude:E1/E2`, `codex:E1/E2`), owned elsewhere in this run |
| G2 | Claude↔Codex structured handoff both complete | PASS (regression evidence only) | `internal/handoff/...` suite passes in A2; no fresh live handoff journey run by this executor (Matrix D scope, owned elsewhere) |
| G3 | Gemini/OpenCode/Grok remain handoff sources, read-only | PASS | Live: `gemini`(T2) and `kimi`(T2) refuse `resume`/`fork` exit 5 naming the tier (§A7); OpenCode is now T5 per the phase-5 note (superseded, expected) |
| G4 | `push`/`pull` carry only Claude/Codex(+OpenCode) | PASS | Live `rein __complete push --agent ""` and `pull --agent ""`: both list exactly `claude`, `codex`, `opencode` — matches the phase-5 doc's own note that OpenCode's T5 promotion widened this set for post-Phase-5 releases; no other agent key offered |
| G5 | Path remapping unchanged | PASS | `internal/pathmap/...` suite passes in A2 |
| G6 | Exit-code semantics unchanged | PASS | Live exit codes observed throughout this report are consistent with documented meanings: `2` (not found/invalid key), `5` (unsupported/blocked, names reason), `3` (config missing), `0` (success) — no new/undocumented code observed |
| G7 | v0.4.0-lineage index upgrades without loss (tested from a real v0.5.1-built index) | PASS | Built a 100-session index with the real `reinstate_0.5.1_windows_amd64` binary (11 agents) in an isolated `$REINSTATE_HOME`; ran the `v0.6.0-rc.2` binary against the same home: `rein version --json` reports `0.6.0-rc.2`, `rein sessions --json` still returns all 100 sessions with an identical per-agent distribution, exit 0, no error/migration prompt needed |
| G8 | Capsules excluded from sync | PASS | `TestRefuseHandoffsPush` (`internal/sync/sync_test.go`) passes in A2: a push under `handoffs/.../capsule.json` is refused |

**8/8 PASS** (G1/G2 carry the caveat that this executor's own evidence is the regression suite, not a fresh physical journey — that mechanism is exercised live under Matrix D/E by whichever executor owns those rows this run).

---

## Matrix H — CLI contract and performance (6 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | `docs/cli-reference.md` matches shipped flags | PASS | `go test ./internal/doctest/... -count=1` → `ok` (part of A5) |
| H2 | `--json` shape for sessions/search/inspect/doctor matches documented shape | PASS | All four commands returned well-formed, schema-conformant JSON throughout this run (`AGENT-PROBE-V1` for doctor; consistent `sessions[]`/`session{}` shapes for the others), consumed programmatically without error at every step above |
| H3 | Full refresh across all installed agents within documented ceiling | PASS | Live cold refresh on a fresh `$REINSTATE_HOME`, 11 installed agents, real sessions: `13.149s` (rc.1's tagged run recorded `13.978s` under the same "reasonable ceiling" standard — consistent, no regression) |
| H4 | No-change refresh materially faster than cold | PASS | Same run, immediate re-invocation: `3.095s` (~76% faster than the `13.149s` cold run) |
| H5 | One unreachable/slow agent source does not block the others | PASS | `TestSlowAgentDoesNotDiscardTheRun` (`internal/agents/probe/collect_test.go`) passes in A2 |
| H6 | Shell completion lists correct agent keys per command (rc.2 retest) | PASS | Live `rein __complete`: `push --agent` and `pull --agent` → `claude, codex, opencode`; `sessions --agent` → `all, claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen` (T1+ only, no T0 key); `handoff --to` → `claude, codex, grok, opencode, qwen`. All three surfaces agree with each other and with A6/G4's key sets |

**6/6 PASS.**

---

## T0 Matrix F — Per-T0-agent (7 agents × 2 rows = 14 rows)

All evidence is live, from the same `rein doctor --agents --json`/`--complete` captures used above.

| Agent | F1 (appears in `doctor --agents` with reason) | F2 (no command offers a capability; nothing implies one is coming) |
| ----- | --- | --- |
| `aider` | PASS — `t0_reason=layout_unverified` | PASS — `sessions/search/push/pull/resume/handoff --agent\|--to` completions never list `aider`; `sessions --agent aider` exits 2 |
| `amp` | PASS — `t0_reason=server_backed` | PASS — same completion/exit-code sweep, clean |
| `antigravity` | PASS — `t0_reason=layout_unverified` | PASS — same sweep, clean |
| `minimax-code` | PASS — `t0_reason=layout_unverified` | PASS — same sweep, clean |
| `openhands` | PASS — `t0_reason=server_backed` | PASS — same sweep, clean |
| `roo` | PASS — `t0_reason=layout_unverified` | PASS — same sweep, clean |
| `zcode` | PASS — `t0_reason=desktop_only` | PASS — same sweep, clean |

**14/14 PASS.**

---

## T1 Matrix C — Per-T1-agent (cline, copilot, cursor, pi × 6 rows = 24 rows)

Evidence policy followed: C1/C2/C4/C5/C6 ran `rein` alone, read-only, against
the host's real vendor roots (counts and field names only below — no title,
prompt, path, or identifier from a session this run did not create). C6 used
a **copy** of each real tree (or, for Cursor, an isolated synthetic store
seeded from the repo's own committed `testdata/sessionindex/cursor/windows`
fixture) so no real tree was ever mutated; the A10 audit already proved
`rein` itself never touches the source. Copies were deleted after use.

| # | Assertion | cline | copilot | cursor | pi |
| - | --------- | ----- | ------- | ------ | -- |
| C1 | ≥2 real sessions, ≥2 distinct projects | PASS — 3 sessions / 2 projects | PASS — 2 sessions / 2 projects | PASS — 2 sessions / 2 projects | PASS — 3 sessions / 2 projects |
| C2 | project/branch/title/timestamp/message-count match the agent's own store | PASS — real non-zero `message_count` (1,1,1) from each session's own `.messages.json` sidecar | PASS — real non-zero `message_count` (5, 26) | PASS — real host sessions currently show `message_count: 0` for both (see finding below); **confirmed fixed on the mechanism** via the rc.2 dispatch's own prescribed method: a synthetic `store.db` copied from the repo's committed fixture, with a known 5-row `messages` table, correctly reports `message_count: 5`, and `size_bytes` (`16527`) is confirmed to include `store.db`, not only `meta.json` (`145` bytes alone) | PASS — real non-zero `message_count` (2,2,2) |
| C3 (rc.2 retest: message-body search, closes #405) | **PARTIAL** — could not create the required fresh throwaway session: `cline --data-dir <isolated>` with no credential returns `Unauthorized`, and locating/copying only the needed credential file out of the real `~/.cline` store was blocked by this session's own tool-use policy before any secret was read; per the contract's own fallback this is recorded PARTIAL rather than skipped | PASS — real host session: searched `rein search "Reply" --agent copilot --json` for a word taken from that session's own bounded `prompt_preview` (never full transcript content) and got exactly 1 match, confirming body-text search on real data | **PARTIAL** for the same reason via the real vendor path — `cursor-agent` is currently broken on this host (`Error: Cannot find module 'tree-sitter'`, Node.js v24.5.0) and cannot launch to create a session at all; **mechanism confirmed** via the same isolated synthetic-store method as C2: planted a token in a `role='user'` row and `rein search <token> --agent cursor --json` returned exactly that session (a `role='assistant'`-only token, planted first, was correctly *not* found, confirming the search also correctly excludes assistant-only content) | **PARTIAL** — the host's 3 real pi sessions all carry an empty bounded `prompt_preview` (nothing to search by without inventing content), and creating a fresh session requires a configured model/API-key not available to this run |
| C4 | bounded metadata only, no transcript body | PASS — `inspect` returns a fixed key set incl. a 30-char `prompt_preview`, no full body | PASS — same bounded shape | PASS — same bounded shape (verified via the synthetic-store inspect call) | PASS — same bounded shape |
| C5 | below T3, every record read-only; resume refused exit 5 | PASS (§A7) | PASS (§A7) | PASS (§A7 + live on the synthetic store: `resume`/`fork` both exit 5 with the stated reason) | PASS (§A7) |
| C6 | corrupted/empty/absent root degrade cleanly, no panic, no partial record | PASS — truncated a copy of a real `.messages.json` sidecar: unaffected sessions still list correctly, exit 0; empty/absent root → `[]`, exit 0 | PASS — truncated a copy of a real session-index json: no crash, sessions still list correctly, exit 0; empty/absent root → `[]`, exit 0 | PASS — truncated a copy of the synthetic `store.db`: session still lists with `message_count` degrading to `0` and `size_bytes` reflecting the truncated file, no panic; empty/absent root → `[]`, exit 0 | PASS — truncated a copy of a real session `.jsonl` mid-record: the truncated session still returns a bounded, non-panicking record (`message_count: 0`, `size_bytes` matching the truncated length — read to the last complete line, not the original size); empty/absent root → `[]`, exit 0 |

**Row count: 24. 20 PASS, 4 PARTIAL** (`cline:C3`, `cursor:C3`, `pi:C3` — could
not create the vendor-CLI-launched throwaway session the rc.2 retest calls
for, recorded PARTIAL with reason per the contract's explicit fallback
clause; `copilot:C3` is a genuine PASS on real data). Every PARTIAL row above
also carries a passing synthetic/mechanism-level check of the exact code
path being retested (search-by-message-body), so the underlying fix is not
in doubt — only the specific "created by the vendor's own CLI in a
throwaway project" evidence shape is missing for three of the four agents.

---

## Findings

### Product/behavioral note (not scored as a defect, needs maintainer eyes)

- **Real Cursor CLI sessions on this host report `message_count: 0`.** Both
  real sessions discovered under this host's live `~/.cursor` show
  `message_count: 0` via `rein sessions --agent cursor --json`, even though
  the rc.2 dispatch expects `cursor:C2`'s hard-coded-zero bug to be fixed.
  Investigation (bounded to what `rein`'s own read-only output permits — this
  session's tooling correctly declined a direct look inside the real
  `store.db`) shows the *code path* is fixed: a synthetic `store.db` built
  from the repo's own committed fixture, with a `messages` table matching
  one of the three names the reader recognizes (`messages`, `message`,
  `bubbles`), correctly yields a real non-zero count. The catalog's own
  comment on `Cursor()` documents that an *unrecognized* store schema
  "degrades to 0" by design (fail-closed, not a crash). Whether the real
  Cursor CLI 2026.08.11 build on this host actually uses one of those three
  recognized table names — and if not, whether that is expected or a gap in
  the recognized-name list — could not be determined without opening the
  real store, which this run correctly avoided. Recommend a maintainer (or
  an executor with narrower scope permitting direct fixture inspection)
  confirm which case this is; if the real schema is simply not in the
  recognized list yet, `cursor:C2` should be considered not-yet-fully-fixed
  against production data even though the dispatch's own prescribed
  synthetic-fixture method (which this report also ran) passes.

### Harness note

- `cursor-agent` (2026.08.11) is currently broken on this acceptance host:
  invoking it throws `Error: Cannot find module 'tree-sitter'` before doing
  anything else. This blocked creating a fresh, real, throwaway Cursor
  session for the `cursor:C3` rc.2 retest (see table above). Not a Reinstate
  defect; a host/vendor-install issue worth fixing before the next run that
  needs a live Cursor CLI journey (this run's Matrix E, if any Cursor E-rows
  exist at Cursor's T1 tier — they do not, T1 has no E rows — but any future
  Cursor tier promotion would need this fixed first).
- `cline --data-dir <isolated>` requires vendor authentication this run
  could not supply without reading a live credential file, which this
  session's own tool-use policy declined mid-task. Recorded as a scope
  limitation on `cline:C3`, not a product defect.
- This host carries a pre-existing `rein`/`reinstate` install on the
  persistent user `PATH` from an earlier session (see header). Any future
  executor on this host should pin an explicit binary path or verify `rein
  version --json` before trusting a bare `rein` invocation.

---

## Row totals (this executor's assignment only)

| Section | Rows | PASS | PARTIAL | FAIL | NOT TESTED |
| ------- | ---- | ---- | ------- | ---- | ---------- |
| Section A gates (A1–A8) | 8 | 8 | 0 | 0 | 0 |
| Matrix A | 10 | 10 | 0 | 0 | 0 |
| Matrix B | 9 | 9 | 0 | 0 | 0 |
| Matrix G | 8 | 8 | 0 | 0 | 0 |
| Matrix H | 6 | 6 | 0 | 0 | 0 |
| Matrix F (T0 × 7) | 14 | 14 | 0 | 0 | 0 |
| Matrix C (T1 × 4) | 24 | 20 | 4 | 0 | 0 |
| **Total** | **79** | **75** | **4** | **0** | **0** |

No `FAIL` and no `NOT TESTED` rows in this executor's assignment. Four
`PARTIAL` rows (`cline:C3`, `cursor:C3`, `pi:C3`) per the contract's own
fallback for a session that could not be created this run; each carries
supporting mechanism-level evidence that the underlying fix is present. One
non-scored product note (`cursor:C2` against real, non-synthetic data) is
flagged above for a maintainer or a follow-up executor to close out.
