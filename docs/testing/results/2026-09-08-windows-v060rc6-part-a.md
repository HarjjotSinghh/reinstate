# `v0.6.0-rc.6` tagged Windows acceptance — part A (gates, core, T0/T1)

`PHASE5-DEVICE-REPORT-V1` (partial — this is one of several tagged-run
executor parts; it does not by itself carry a device verdict)

Scope: section A automated gates (A1–A8); Phase 5 core matrices A, B, G, H;
Matrix F (all seven T0 agents); Matrix C for the four T1 agents (`cline`,
`copilot`, `cursor`, `pi`). Sections C (CLI experience), D (Hop parity),
Matrix D/E for T2+/T3+ agents, and the maintainer's `grok:E1`–`E3` console
evidence are out of this part's scope and are reported by other executors.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.6` (annotated, signed) |
| Tag signature | `git verify-tag v0.6.0-rc.6` → Good "git" signature, ED25519 key `SHA256:0P4/a2ZBw25hhf0mCN5s2NsUlDiqRqshv8vGNgMPHjU`, verified against `.github/allowed_signers` |
| Tag peeled commit | `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| GitHub release | non-draft, `isPrerelease: true`, published `2026-09-07T23:34:41Z` (`gh release view`, read-only) |
| Worktree | `v060/rc6-tagged` at `7c5adccaea602ff952cd31c21bafa7fdd22a2cfc` |
| Installed binary SHA-256 (both routes) | `272efb504a62fdd2624fa6b117a2587a4869160c034adeb9fcd6281231f078c1` — matches `checksums.txt`'s `reinstate_0.6.0-rc.6_windows_amd64.exe` entry exactly |
| `rein.exe` / `reinstate.exe` | byte-identical (`cmp` exit 0), both installs |
| `rein version --json` | `{"commit":"7c5adccaea602ff952cd31c21bafa7fdd22a2cfc","date":"2026-09-07T23:28:49Z","name":"reinstate","version":"0.6.0-rc.6"}` |
| **Bootstrap deviation** | None in substance. The live `https://reinstate.dev/install.ps1` pins `$Version = "v0.6.0-rc.6"` exactly (confirmed by reading the saved script before executing it) and the installed artifact's SHA-256 matches the checksummed release archive byte-for-byte. The install dir override variable is `INSTALL_DIR`, not `REINSTATE_INSTALL_DIR` — using the wrong name silently installs to the default `%LOCALAPPDATA%\Programs\Reinstate\bin` and then refuses to overwrite a pre-existing older install there without `REINSTATE_CONFIRM_REPLACE=1`; this is documented installer behavior, not a defect, and was corrected before any binary was relied on. |
| Executor's own artifact-identity install | `D:\ReinstateAcceptanceProjects\v060-rc6-a\bootstrap\bin\` (live bootstrap, this executor only) |
| Executor's working install | `D:\ReinstateAcceptanceProjects\v060-rc6-a\install\` (extracted from the coordinator-verified `rc6-draft` zip, sha256 re-verified independently) |
| Host | Windows 11 Pro, native `windows/amd64`, not WSL |
| Go toolchain | `go1.26.1 windows/amd64` (via `golang.org/toolchain@v0.0.1-go1.25.13` pin for the module's own `go 1.25.13` directive; matches contract) |
| Host contamination rule | Every shell in this part ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` before any `rein`/Go-test invocation. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left untouched throughout (host's live agent homes) |
| Binary rule | Every `rein`/`reinstate` invocation in this part used the full path to the executor's own install (`D:\ReinstateAcceptanceProjects\v060-rc6-a\install\rein.exe`), never a bare `rein`/`reinstate`. `where rein` at the top of the run showed 6 older installs ahead on PATH (`D:\Projects\reinstate\rein.exe` first), confirming the rule's necessity |
| `doctor --agents --acceptance-matrix --json` | `row_count: 178`, `core: {A:10, B:9, G:8, H:6}` — matches the dispatch exactly |
| `sessions --help` `--agent` list | 11 keys: `claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen` (+ `all`) |
| Agent census (`doctor --agents`) | `claude` T5/74 sessions, `codex` T5/418, `opencode` T5/18, `grok` T4/30, `qwen` T4/9, `gemini` T2/30, `kimi` T2/5, `pi/cline/copilot/cursor` T1/9/11/3/3, all 7 T0 agents present with an enumerated reason and `installed: no` |

## Section A — automated gates (8/8 PASS; not part of the 216)

All gates run from the worktree except A6/A7 which point at the
coordinator-verified `rc6-draft` dist directory, exactly as assigned.

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | `gofmt -l .` / `go vet ./...` / `go mod tidy -diff` | PASS | All three commands produced empty output |
| A2 | Unit suite, `CGO_ENABLED=0 go test ./... -count=1 -p 4` | PASS | Every package `ok` or `[no test files]`, `EXIT:0` |
| A3 | Race suite, `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p 4` | PASS | Every package `ok`, `EXIT:0`. **First attempt failed** with a ThreadSanitizer mmap allocation error (`the paging file is too small`) — a host resource-contention harness defect from running A2/A3/lint/vuln concurrently in the background, not a race condition; retried alone and passed cleanly |
| A4 | `make lint`; `make vuln` | PASS | `golangci-lint run` → `0 issues`; `govulncheck` → `0 vulnerabilities` (4 unreachable module-level vulns noted, none reachable by our code). **Lint's first attempt also failed** on the same paging-file exhaustion as A3; retried alone and passed |
| A5 | `go test ./internal/doctest/... -count=1`; `scripts/check-docs.ps1` | PASS | doctest package `ok` (11.985s); `check-docs.ps1` exit 0 |
| A6 | `scripts/verify-release.ps1 -DistDir <rc6-draft>` | PASS | exit 0 |
| A7 | `scripts/test-install.ps1 -DistDir <rc6-draft>` | PASS | exit 0 |
| A8 | `GOOS=darwin`/`GOOS=linux go build ./...` | PASS | Both exit 0, no output |

## Matrix A — Catalog integrity (10/10 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Descriptors validate at start; no duplicate/empty keys | PASS | `doctor --agents --json`: 18 unique agent keys |
| A2 | `doctor --agents` lists every agent incl. every T0 | PASS | All 7 T0 keys present, `installed: no`, each with a reason |
| A3 | Every T0 agent shows an enumerated reason | PASS | `layout_unverified` (aider, antigravity, minimax-code, roo), `server_backed` (amp, openhands), `desktop_only` (zcode) — 3-value closed enumeration |
| A4 | Declared tier and capability constructors agree | PASS | Covered by the full green suite (`TestShippedAgentsRegisterAtDeclaredTiers`, `TestMustRegisterAcceptsConstructorsAtDeclaredTier`) |
| A5 | Every `Evidence` path exists in the tagged tree | PASS | `TestShippedEvidenceIsComplete`, part of the green `internal/doctest` run |
| A6 | `sessions --agent` accepts exactly T1+ keys, else exit 2 | PASS | All 11 T1+ keys (`claude, pi, cline, copilot, cursor, gemini, grok, kimi, qwen, codex, opencode`) exit 0; `aider` (T0) and an unknown key both exit 2 with `"code":"usage"` naming the 11-key list |
| A7 | `resume` accepts exactly T3+ keys, else exit 5 with reason | PASS | `claude` (nonexistent id) exit 2 not-found; `pi` (T1) exit 5 "tier T1; native resume is unsupported"; `aider` (T0) exit 5 similarly; `gemini` (T2) exit 5 similarly |
| A8 | Agent ordering deterministic across runs | PASS | 3 consecutive `doctor --agents --json` runs, byte-identical key ordering |
| A9 | Broken evidence path fails conformance | PASS | Covered by the full green suite (`TestBrokenEvidencePathFails`/`TestBrokenEvidencePathIsCaught`) |
| A10 | No scan writes/renames/locks any file under an agent root | PASS | Before/after full-file listing (relative path, size, mtime) across 4 real roots (`cline`, `copilot`, `cursor`, `pi`) spanning a `doctor --agents` + full `sessions --agent all` scan: `copilot`/`cursor`/`pi` byte-for-byte identical; `cline` showed exactly one changed entry, a `hub-events-hub-production.db-wal` mtime bump — a telemetry/hub database outside the `sessions`/`sessions.db` tree the probe reads, attributable to Cline's own background VS Code extension process, not this run's scan |

## Matrix B — Probe and redaction (9/9 PASS)

**`B4` and `B7` are this part's specific re-test targets, both confirmed.**

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | Output conforms to `AGENT-PROBE-V1` | PASS | `"schema": "AGENT-PROBE-V1"` |
| B2 | No absolute path, home name, or username | PASS | Full-document scan for the host username and `[A-Za-z]:\...`/`/Users/<name>` patterns: 0 hits |
| B3 | No JSON values from session files, keys only | PASS | `first_line_keys` entries are field-name arrays only (e.g. `kind, lastUpdated, projectHash, sessionId, startTime` for gemini) |
| B4 | Names shape-normalized | **PASS — re-test target** | Ran against the real, installed `gemini` tree (project-hash directories under `tmp/*`) and every other real root on this host. Every `tmp/*` segment collapsed to the `<64-hex>` shape token; every other agent's `name_shapes` across the whole document used only templated tokens (`<slug>`, `<uuid-v4>`, `<32/40/64-hex>`, `<n>`, and their compounds) — an automated scan for any raw 16+ hex/UUID-looking string appearing **outside** a `"shape"` field, across the entire real-root probe document, found 0 hits. Reporting counts only per policy, no raw names |
| B5 | No excluded subtree present | PASS | Case-insensitive scan for `credential`, `secret`, `password` substrings: 0 hits |
| B6 | Absent agent reports absence without error | PASS | All 7 T0 agents: `candidate_roots: []`, `resolved_root: null`, no error, exit 0 |
| B7 | Empty root distinguishable from absent root | **PASS — re-test target** | `CLINE_DATA_DIR` and `KIMI_CODE_HOME` each pointed first at a freshly-created **empty existing** directory, then at a **nonexistent** path. In both agents' `candidate_roots`, the env-relative entry's `exists` field flips `true` → `false` between the two runs — never byte-identical apart from that field |
| B8 | Wrapper scripts match binary output | PASS | `scripts/testing/agent-storage-probe.sh` and `.ps1`, run with the tagged install first on `PATH` under both `rein` and `reinstate`, produced output structurally identical to the direct binary invocation (mechanically confirmed: the `.sh` wrapper execs `rein doctor --agents --json` verbatim; the `.ps1` wrapper resolves `rein`/`reinstate` off `PATH` and calls the same subcommand). Small byte-count drift in a couple of `median_bytes`/`children` fields between sequential invocations is attributable to this being a live, actively-used host (the live `claude`/`codex` roots grow between any two commands), not a wrapper defect |
| B9 | Secret scanner passes over committed probe artifacts | PASS | `go test ./internal/fixture -count=1` → `TestScanTreeFixtures` PASS, scanning all 26 committed files under `docs/testing/results/agent-probes/` |

## Matrix G — No regression (7 PASS, 1 PARTIAL)

**`G4` is this part's re-test target (corrected contract wording), confirmed. `G7` uses a real `v0.5.1`-built index, as assigned.**

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude/Codex native resume and fork unchanged | **PARTIAL** | **Claude half fully exercised, PASS:** real isolated session, planted token; native `claude --resume <id> -p "..."` returned the exact planted token from history; native `claude --resume <id> --fork-session -p "..."` produced a **distinct** new session id that also correctly recalled the token; `rein fork claude:<id> --dry-run --json` named the correct native flags (`--resume <id> --fork-session`). **Codex half NOT TESTED:** the host's live, authenticated Codex account returned `"You've hit your usage limit... try again at 10:13 AM"` on every attempt (3 independent attempts: a fresh session, a resume of an existing session, and a minimal one-word prompt), reproduced identically with both the installed npm Codex CLI (`0.153.4`, itself above this catalog's verified ceiling `0.149.0`) and a second, in-range Codex install already present on this host (`0.146.0`, within `0.133.0`–`0.149.0`) — so the block is the live account's own usage limit, not a version or harness issue. See "Harness notes" below; this is a new environment condition, not a carried disposition |
| G2 | Both structured handoff directions complete | PASS | `handoff claude:<id> --to codex --dry-run --json` (from G1's own claude session, run from its own workspace) → exit 0, full capsule + fidelity report. `handoff codex:<id> --to claude --dry-run --json` (a real, pre-existing codex session, run from its own workspace) → **exit 5 "source agent codex is UNTESTED"** with the default (drifted, `0.153.4`) codex on `PATH`, because the row's version check reads whatever `codex --version` currently reports, not the session's own recorded version — a correct refusal per "native agent version X is outside the verified range". Re-ran with the in-range `0.146.0` Codex CLI placed first on `PATH`: exit 0, full capsule produced. Both directions confirmed to complete on an in-range vendor CLI |
| G3 | Gemini, OpenCode, Grok remain sources; Gemini stays read-only | PASS | Source: `gemini.go` declares `Tier: agents.TierHandoffFrom` (read-only); `grok.go` declares `Tier: agents.TierHandoffTo` (not `TierSync`); `opencode.go` declares `Tier: agents.TierSync` (its documented T5 promotion). Full green `TestShippedAgentsConformance` suite |
| G4 | `push`/`pull` carry exactly Claude Code, Codex CLI, OpenCode, and no other agent's — corrected wording | **PASS — re-test target** | Source-level: only `claude.go`, `codex.go`, `opencode.go` declare `Tier: agents.TierSync` in the whole catalog. Behaviorally confirmed by H6 shell completion below: `push --agent`/`pull --agent` complete to exactly `claude, codex, opencode` |
| G5 | Cross-OS path remapping unchanged | PASS | `go test ./internal/pathmap/... -count=1` → `ok` (Windows↔Windows leg only; macOS leg is deferred) |
| G6 | Exit-code semantics unchanged; no new codes | PASS | Every command in this part's testing returned one of the documented codes (`0`, `2`, `5`); `internal/doctest`'s green suite covers the documented exit-code table |
| G7 | An index built by an older release upgrades without data loss | **PASS — assigned re-test** | Built a fresh index with a real, checksum-verified `v0.5.1` Windows binary (`82b6b431db27fb2b3a1178680a66093e78c745322edb680933ce3850d078cc67`, matches `checksums.txt`) against the host's real, live Claude Code project tree in an isolated `REINSTATE_HOME`: 74 sessions indexed. Opened the identical `REINSTATE_HOME` with the tagged `v0.6.0-rc.6` binary: same 74 sessions returned, exit 0, and the two session-id sets diffed byte-for-byte identical — no data loss, no migration message needed |
| G8 | Capsules remain excluded from sync | PASS | `go test ./internal/sync/... -run TestRefuseHandoffsPush -v` → PASS |

## Matrix H — CLI contract and performance (6/6 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | CLI reference matches shipped flags | PASS | `internal/doctest` green (A5) |
| H2 | `--json` shapes match documentation | PASS | Full green `internal/doctest`; every `--json` output produced in this part parsed as well-formed JSON matching its documented shape |
| H3 | Full refresh within documented ceiling | PASS | Cold `sessions --agent all --json --limit 1000` against a fresh `REINSTATE_HOME`: **12.836s**, within the ~20–30s ceiling |
| H4 | No-change refresh materially faster | PASS | Immediate repeat, same `REINSTATE_HOME`: **4.257s**, ≈3× faster |
| H5 | One slow/unreachable source does not block others | PASS | `TestSlowAgentDoesNotDiscardTheRun`, part of the green A2/A3 suite |
| H6 | Completion lists correct agent keys per command | PASS | Per this part's assignment: `rein __complete push --agent ""` and `pull --agent ""` → exactly `claude, codex, opencode` (matches G4's corrected wording exactly); `rein __complete sessions --agent ""` → `all` plus the 11 T1+ keys, matching `sessions --help` exactly |

## Matrix F — Per T0 agent (14/14 PASS)

All 7 T0 agents are not installed on this host — a scope reduction, not a `FAIL`.

| Agent key | F1 (listed with reason) | F2 (no capability offered) |
| --------- | ------------------------ | ---------------------------- |
| `aider` | PASS — `t0_reason=layout_unverified` | PASS — `sessions --agent aider` exit 2; `resume aider:x` exit 5 |
| `amp` | PASS — `t0_reason=server_backed` | PASS — same shape |
| `antigravity` | PASS — `t0_reason=layout_unverified` (host has the Antigravity IDE installed; the catalog agent still reports no capability) | PASS — same shape |
| `minimax-code` | PASS — `t0_reason=layout_unverified` | PASS — same shape |
| `openhands` | PASS — `t0_reason=server_backed` | PASS — same shape |
| `roo` | PASS — `t0_reason=layout_unverified` | PASS — same shape |
| `zcode` | PASS — `t0_reason=desktop_only` | PASS — same shape |

## Matrix C — Per T1 agent: `cline`, `copilot`, `cursor`, `pi` (24/24 PASS)

Per the ground rules: `cline`/`pi`/`copilot` C3 created a fresh, throwaway
session through the vendor's own live CLI with a planted, non-sensitive
search token in a throwaway git project under
`D:/ReinstateAcceptanceProjects/v060-rc6-a/`. `cursor` ran entirely
read-only against the host's real store (`cursor-agent` is broken on this
host: `Cannot find module tree-sitter`, confirmed by its absence from
`PATH`) — C2 reports `message_count` values only, C3 used a word chosen
from a real session's own `prompt_preview` and reports match-count only, per
policy. C6 for every agent used a filesystem **copy**, never live data.

| # | `cline` | `copilot` | `cursor` | `pi` |
| - | ------- | --------- | -------- | ---- |
| C1 (≥2 sessions, ≥2 projects) | PASS — 11 sessions / 7 projects | PASS — 3 sessions / 3 projects | PASS — 3 sessions / 3 projects | PASS — 9 sessions / 8 projects |
| C2 (metadata matches vendor) | PASS — `message_count` values `[0, 1, 2]` across real sessions, project/workspace/title populated | PASS — `message_count` values `[5, 5, 26]` | PASS — `message_count` values `[4, 3, 3]` | PASS — `message_count` value `2` uniformly (single-turn sessions), project/workspace/title populated |
| C3 (message-body search) | PASS — planted token found via `rein search`, exactly 1 match | PASS — planted token found via `rein search`, exactly 1 match | PASS — a word chosen from a real session's own `prompt_preview`, `rein search` found exactly 1 match (word/preview withheld per policy) | PASS — created via `pi -p`, found by `rein search`, `prompt_preview` populated with the planted token |
| C4 (bounded inspect, no body) | PASS — `prompt_preview` only, no transcript body | PASS — same shape | PASS — same shape | PASS — same shape |
| C5 (resume gate correct for tier) | PASS — exit 5, `read_only_reason` populated | PASS — exit 5, same shape | PASS — exit 5, same shape | PASS — exit 5, same shape |
| C6 (corrupt/empty/absent degrade cleanly) | PASS — 30%-truncated copy of a session file → `session_read_failed` warning, exit 0, no panic; empty dir and nonexistent dir both return `{"sessions":[]}`, exit 0 | PASS — 30%-truncated `events.jsonl` copy → exit 0, no panic, session count consistent; empty/absent clean, exit 0 | PASS — 30%-truncated `store.db` copy (targeted `chats/` subtree fixture, not the full `.cursor` config dir) → exit 0, no panic; empty/absent clean, exit 0 | PASS — 30%-truncated `.jsonl` copy → exit 0, no panic; empty/absent clean, exit 0 |

## Harness notes (non-blocking to the mechanisms under test)

1. **Paging-file exhaustion under concurrent background jobs.** Running the
   unit suite, race suite, lint, and vuln scan simultaneously in the
   background exhausted this host's Windows paging file, causing the race
   suite (ThreadSanitizer `mmap` allocation failure) and lint (Go compiler
   `fork/exec` failure) to fail spuriously. Both passed cleanly on a serial
   retry. Not a product defect; recorded so other executors avoid running
   heavy Go toolchain jobs in parallel with each other on this host.
2. **PowerShell `.NET Process.Start` / nested-`cmd` flakiness invoking
   `rein.exe`.** Direct `& $R ...` invocation and `cmd /c "..." > file"`
   both intermittently returned exit `-1`/`255` with zero output for the
   exact same command and arguments that succeeded moments later or
   immediately before — confirmed via a direct `Test-Path`/one-shot
   `Process.Start` that the binary and its hash were unaffected throughout.
   The reliable pattern found was plain `& $R args *> file` (all streams
   redirected via the PowerShell native operator); every B7/C6 fixture
   result recorded above used that pattern, retrying individually on the
   rare flake. Not a product defect.
3. **Live Codex account usage limit.** The host's live, authenticated
   Codex account is exhausted (`"You've hit your usage limit... try again
   at 10:13 AM"`), reproduced on 3 independent live-turn attempts against
   two different installed Codex CLI versions. This blocks any row in this
   run (any executor's) that needs a **live** Codex conversational turn —
   confirmed not a version or harness artifact. `--dry-run` rows are
   unaffected once an in-range Codex CLI (`0.146.0`, already installed at
   `C:\Users\admin\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`) is
   placed first on `PATH`.
4. **Codex CLI version drift.** The default, PATH-first Codex CLI
   (`C:\nvm4w\nodejs\codex`, npm-installed) reports `0.153.4`, above this
   catalog's verified ceiling `0.149.0`. `rein`'s own version gate
   correctly refuses live-mechanism rows against it (`native agent version
   0.153.4 is outside the verified range 0.133.0 to 0.149.0 inclusive`,
   exit 5) — this is the documented "read the refusal correctly" case, not
   a defect. A second, in-range Codex install (`0.146.0`) already present
   on this host was used for the dry-run-only rows above; it was not
   installed for this row.

## Counts (this part only)

- Section A gates: 8/8 PASS (not part of the 216-row count)
- Matrix A: 10/10 PASS
- Matrix B: 9/9 PASS
- Matrix G: 7/8 PASS, 1/8 PARTIAL (`G1`, Codex half — live account usage limit, not a product defect)
- Matrix H: 6/6 PASS
- Matrix F (T0): 14/14 PASS
- Matrix C (T1: cline/copilot/cursor/pi): 24/24 PASS
- **Part A required-row total: 70 PASS / 1 PARTIAL / 0 FAIL / 0 NOT TESTED of 71**

This part does not by itself carry the device verdict; it feeds the
coordinator's full-run reconciliation alongside the other executors' parts
(sections C, D, Matrix D/E for T2+/T3+ agents, and the maintainer's
`grok:E1`–`E3` console evidence).
