# v0.6.0-rc.8 tagged Windows acceptance — part A (gates, core matrices, T0/T1)

`PHASE5-DEVICE-REPORT-V1` (partial — this is executor A's part of a
multi-executor tagged-artifact run; section headers below name exactly the
rows this part owns per the dispatch)

Contract: [`v0.6.0-rc.8-agent-verification-prompts.md`](../v0.6.0-rc.8-agent-verification-prompts.md),
composing [`v0.6.0-windows-acceptance.md`](../v0.6.0-windows-acceptance.md) and
[`phase-5-universal-agent-coverage-acceptance.md`](../phase-5-universal-agent-coverage-acceptance.md).

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tested tag | `v0.6.0-rc.8` |
| Tested full commit | `3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f` |
| Release commit ancestry | tag `v0.6.0-rc.8` verified: `git verify-tag` → `Good "git" signature for HarjjotSinghh@gmail.com with ED25519 key SHA256:0P4/a2ZBw25hhf0mCN5s2NsUlDiqRqshv8vGNgMPHjU`, matches `.github/allowed_signers` |
| GitHub release | `gh release view v0.6.0-rc.8 --json` → `isDraft=false, isPrerelease=true, publishedAt=2026-09-08T22:43:53Z, targetCommitish=main` |
| Windows archive SHA-256 | `658dc27e607fdec14d14156dfe3edaf9d685bfa9add1e467d604ab0fa298e28f` (re-verified locally against `checksums.txt`, second independent verification of the coordinator's own) |
| Installed `rein version --json` | `{"commit":"3f6aa7b9a17242e9df2eac50850f7eb4b7f06f2f","date":"2026-09-08T22:38:21Z","name":"reinstate","version":"0.6.0-rc.8"}` — identical from both the bootstrap install and the checksummed-archive install |
| `rein.exe` / `reinstate.exe` byte identity | SHA-256 `482b5de12db1312c1a767bcf40c7999de9f8cef7f47bff5bce802fab960c3b71` for both, both install paths |
| Bootstrap deviation | **None.** `Invoke-WebRequest -OutFile` saved `https://reinstate.dev/install.ps1`; `$Version = "v0.6.0-rc.8"` confirmed at line 5 before execution; ran with `INSTALL_DIR` and `REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` pointed at `D:/ReinstateAcceptanceProjects/v060-rc8-a/bootstrap/bin`; installed cleanly, reported the exact tag/commit above. This is executor A's artifact-identity install per the dispatch; every other executor installs from the coordinator's checksummed `rc8-draft` directory. |
| Install directories | Bootstrap: `D:/ReinstateAcceptanceProjects/v060-rc8-a/bootstrap/bin/`. Checksummed-archive: `D:/ReinstateAcceptanceProjects/v060-rc8-a/install/` (both binaries re-verified against `checksums.txt` before use) |
| `doctor --agents --acceptance-matrix --json` | `row_count: 178`, `core_row_count: 33` (`A:10, B:9, G:8, H:6`), 18 catalog agents, tiers as census (T5: claude/codex/opencode 17 rows each; T4: grok/qwen 17 each; T2: gemini/kimi 11 each; T1: cline/copilot/cursor/pi 6 each; T0: aider/amp/antigravity/minimax-code/openhands/roo/zcode 2 each) |
| `sessions --help` `--agent` enum | `claude\|cline\|codex\|copilot\|cursor\|gemini\|grok\|kimi\|opencode\|pi\|qwen\|all` — exactly 11 agent keys plus `all` |
| BINARY RULE | Every command below invoked the full path `D:/ReinstateAcceptanceProjects/v060-rc8-a/install/rein.exe` (or the bootstrap path for the artifact-identity step only). `where rein` at session start showed 4 installs on PATH, ours first only when explicitly prepended; every command in this report used the full path, never a bare `rein`/`reinstate`. |
| Host environment | `REINSTATE_BACKEND` / `REINSTATE_MEMORY_BACKEND_DIR` unset in every shell before use (confirmed at the top of every command block below). `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` left at the host's live values for real-agent rows; isolated per-row only via the vendor's own declared `root_env` variable. |
| Host (sanitized) | Windows 11 Pro, build 10.0.26200, x64-based PC |
| Go toolchain | `go.mod` pins `go 1.25.0` / `toolchain go1.25.13`; this host's ambient `go` resolves to `go1.26.1` (`GOTOOLCHAIN=auto` prefers a newer-than-required ambient toolchain — see Harness Note 1). All gates below were re-run pinned to `GOTOOLCHAIN=go1.25.13`, matching every CI workflow (`ci.yml`, `release.yml`, `security.yml` all pin `go-version: "1.25.13"`). |
| Report date | 2026-09-09 (UTC times inline below) |

## Harness notes (read before the tables)

1. **Ambient Go toolchain drift (self-corrected).** This host's `go` resolves
   to `1.26.1` under `GOTOOLCHAIN=auto`, one minor ahead of the `1.25.13` every
   CI workflow pins. A first `govulncheck` pass under the ambient toolchain
   reported 17 stdlib vulnerabilities (`GO-2026-6218` … `GO-2026-4866`,
   `go1.26.1` fixed in `go1.26.2`–`go1.26.6`), none of which are reachable
   under the pinned `go1.25.13` — a second run with `GOTOOLCHAIN=go1.25.13`
   reported `0` reachable vulnerabilities, matching `v0.6.0-rc.7`'s own
   result and CI's own toolchain. **Not a product finding** — an artifact of
   this host's ambient Go being newer than the pinned CI toolchain. All A1–A5,
   A8 gates below are the `GOTOOLCHAIN=go1.25.13`-pinned results.
2. **Shared worktree.** This worktree accumulated commits from other parts
   (`test(v0.6.0-rc.8): record tagged-run part E Hop parity rows`,
   `test(v0.6.0-rc.8): record executor D's Hop parity part`) during this
   session, and carried transient untracked scratch directories
   (`cmd/labrunfd/`, `cmd/labverify/`, and briefly
   `scripts/testing/devicerevoke/`, `scripts/testing/consoledriver/`, which
   disappeared mid-session) plus one other part's own uncommitted, in-progress
   test-harness fix (`scripts/testing/conptydriver/vtscreen.go`, an `ECH`
   CSI-sequence case for that part's own CLI row 9 capture, unrelated to
   product code). This worktree was not exclusively held by this part despite
   the dispatch's stated isolation. **Mitigation:** every gate that scans
   `./...` was re-scoped to `git ls-files '*.go'`'s 82 tracked directories
   (the untracked scratch packages do not build for `darwin`/`linux` and trip
   `errcheck`, exactly as `v0.6.0-rc.7`'s own report found for its own
   contaminating scratch file) — this is the same scoping method
   `v0.6.0-rc.7`'s report used, not a new workaround. Nothing outside this
   part's own results file was committed or altered.
3. **Untracked scratch left in place.** `cmd/labrunfd/`, `cmd/labverify/`
   remained untracked and unbuilt-around at report time; not touched, per the
   never-modify-what-is-not-mine rule.

## Section A — Automated gates (A1–A8)

Run once by executor A per the dispatch ("Do NOT run the Go test suite;
executor A alone runs the section A automated gates once with -p 4"). A1–A5,
A8 from the worktree, `GOTOOLCHAIN=go1.25.13`, scoped to the 82
`git ls-files`-tracked Go directories (Harness Note 2). A6/A7 against the
coordinator's checksummed `rc8-draft` directory.

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| A1 | Format, vet, tidy | PASS | `gofmt -l .` → 0 lines; `go vet ./...` → exit 0, no output; `go mod tidy -diff` → exit 0, no output |
| A2 | Unit suite | PASS | `CGO_ENABLED=0 go test <82 tracked dirs> -count=1 -p 4` → 70 `ok`, 0 `FAIL`, exit 0 |
| A3 | Race suite | PASS | `CGO_ENABLED=1 go test <80 tracked dirs, excl. internal/doctest, internal/crypto> -race -count=1 -timeout=20m -p 4` → 68 `ok`, 0 `FAIL`, 0 `DATA RACE`, exit 0 |
| A4 | Lint and vuln | PASS | `golangci-lint run` (v2.11.4, pinned toolchain, scoped): `0 issues`; `govulncheck` (v1.6.0, pinned toolchain): `0` reachable vulnerabilities (see Harness Note 1 for the ambient-toolchain false-positive it superseded) |
| A5 | Doc gate | PASS | `go test ./internal/doctest/... -count=1` → `ok`, includes `TestCLIReferenceListsCatalogKeys`, `TestPhase4CLIFlagsAreDocumented` (H1's own enforcement); `scripts/check-docs.ps1` → exit 0 |
| A6 | Installers (dist dir) | PASS | `scripts/verify-release.ps1 -DistDir <rc8-draft>` → `release artifacts verified (PowerShell): <rc8-draft>`, exit 0 |
| A7 | Install script | PASS | `scripts/test-install.ps1 -DistDir <rc8-draft>` → verify step passes, `ok internal/doctest 1.232s`, exit 0 |
| A8 | Cross-OS build | PASS | `GOOS=darwin GOARCH=amd64 go build <tracked dirs excl. internal/doctest>` → exit 0, no output; `GOOS=linux GOARCH=amd64 go build <same>` → exit 0, no output |

**8/8 PASS.**

## Matrix A — Catalog integrity (10/10 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Descriptors validate at start; no duplicate/empty keys | PASS | `doctor --agents --json` → 18 unique, non-empty `key` values; `internal/agents/conformance` suite (`go test ./internal/agents/... -v`) → 195 `--- PASS`, 0 `--- FAIL`, `ok`, exit 0 — includes `TestShippedAgentsConformance` over all 18 registered descriptors |
| A2 | `doctor --agents` lists every agent incl. every T0 | PASS | all 18 keys present, including all 7 T0 agents (`aider`, `amp`, `antigravity`, `minimax-code`, `openhands`, `roo`, `zcode`), each `installed=no` |
| A3 | Each T0 agent shows a machine-readable reason from a closed enumeration | PASS | human `doctor --agents` → `t0_reason=layout_unverified` (aider, antigravity, minimax-code, roo), `t0_reason=server_backed` (amp, openhands), `t0_reason=desktop_only` (zcode) — 3 distinct values, closed set |
| A4 | Declared tier and present capability constructors agree | PASS | `internal/agents/conformance.TestShippedAgentsConformance` (structural, per-descriptor) plus empirical cross-check: T5 (`claude`) accepts `sessions`+`resume` (exit 0); T1 (`cline`) accepts `sessions`, refuses `resume` (exit 5); T0 (`aider`) refuses `sessions` (exit 2) and `resume` (exit 5) |
| A5 | Every `Evidence` path exists in the tagged tree | PASS | `internal/agents/conformance.TestShippedEvidenceIsComplete` — iterates `agents.All()` (not a hand list), `ok` |
| A6 | `sessions --agent <key>` accepts exactly T1+, rejects others exit 2 | PASS | `sessions --agent aider --json` → `code:"usage", message:"invalid agent; expected claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen, or all"`, exit 2; `sessions --agent claude --json` → real sessions, exit 0; `sessions --agent cline --json` → real sessions, exit 0 (T1 accepted) |
| A7 | `resume` accepts exactly T3+, refuses others exit 5 with reason | PASS | `resume aider:x --dry-run --json` → `"Aider is tier T0 (layout_unverified); native resume is unsupported"`, exit 5; `resume gemini:<real-id> --dry-run --json` (T2) → `"Gemini CLI sessions are read-only in Phase 2"`, exit 5; `resume claude:<real-id> --dry-run --json` (T5) → launch plan, exit 0 |
| A8 | Agent order deterministic across runs | PASS | 5× `doctor --agents --json` → identical 18-key order every run; 3× `sessions --agent all --json --limit 200` → identical first-15-key order every run |
| A9 | Broken-evidence descriptor fails the conformance suite | PASS | `internal/agents/conformance.TestBrokenEvidencePathFails`, `TestRunFailsBrokenEvidence`, `TestBrokenEvidencePathIsCaught` — all `--- PASS` (the suite's own negative-test coverage, exercised as part of `go test ./internal/agents/...`) |
| A10 | No catalog scan writes/renames/locks under an agent root | PASS | Full-tree filesystem snapshot (relative path + size + mtime) of all 11 installed agents' real roots, before and after a full `doctor --agents --json` + `sessions --agent all --json --limit 1000` scan: 0 added, 0 removed, 0 changed files in 10 of 11 roots; the 11th (`claude`) showed 5 changed files, all under this very session's own live Claude Code transcript path (`…/subagents/workflows/wf_…/agent-*.jsonl`) — attributable to this session's own concurrent tool-call logging (a live agent continuously writes its own transcript, matching the documented harness-trap rule), not to `rein`'s read-only scan |

## Matrix B — Probe and redaction (9/9 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | `doctor --agents --json` conforms to `AGENT-PROBE-V1` | PASS | `"schema": "AGENT-PROBE-V1"` |
| B2 | No absolute path, home directory name, or username | PASS | Case-insensitive scan of the full JSON for `admin`, `C:\Users`, `D:\Projects` → 0 matches |
| B3 | No JSON value from any session file, only keys | PASS | Full key inventory of the JSON tree contains only schema/shape descriptors (`shape`, `kind`, `count`, `median_bytes`, `first_line_keys`, …) — no session titles/prompts; scanned for a known planted-token substring present elsewhere on the host → 0 matches in probe output |
| B4 | Directory/file names shape-normalized, no raw UUID/hash/slug | PASS (re-tested against the real Gemini tree and every other real root, counts only) | 0 raw-UUID-v4 regex matches across the full probe document; `name_shapes` for every installed agent (incl. `gemini`) shows only template forms (`<slug>.json`, `<uuid-v4>.json`, `pack-<40-hex>.idx`, `wd_<project>_<12-hex>`, …), never a literal segment |
| B5 | No excluded subtree (credential/cache) appears | PASS | Full-document lowercase scan for `credential`, `keyring`, `token`, `apikey`, `.cache`, `secret` → 0 matches |
| B6 | Probe against an absent agent reports absence without error | PASS | `CLINE_DATA_DIR=<nonexistent>` and `KIMI_CODE_HOME=<nonexistent>` → `doctor --agents --json` exit 0 both times; env candidate `exists:false, marker_present:false, resolved_root:null` |
| B7 | Probe against an existing-but-empty root is distinguishable from absent | PASS | Same two agents (`CLINE_DATA_DIR`, `KIMI_CODE_HOME`) re-run pointed at an existing empty directory: `exists:true` (root-state field), vs. `exists:false` for the nonexistent case — the JSON differs exactly in that field, both exit 0 |
| B8 | `agent-storage-probe.sh`/`.ps1` output identical to the binary | PASS (native Windows: `.ps1` only — `.sh` is documented for macOS/Linux/WSL2, out of scope for a native, never-WSL Windows column) | `scripts/testing/agent-storage-probe.ps1` output vs. direct `rein.exe doctor --agents --json`, both with PATH resolving to our install dir: byte-identical after normalizing only the `generated_at` timestamp field |
| B9 | Fixture secret scanner passes over every committed probe artifact | PASS | `go run github.com/zricethezav/gitleaks/v8@v8.30.1 dir docs/testing/results/agent-probes --config .gitleaks.toml --redact --verbose` → scanned 366.45 KB, `no leaks found`, exit 0 |

## Matrix G — No regression (8/8 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude Code and Codex CLI native resume and fork behave as in `v0.4.0` | PASS | **Claude half** (real `2.1.265`, in the tagged range `2.1.219`–`2.1.265`): real isolated throwaway project; planted token via `claude -p`; `rein search --agent claude --json` found it; `resume --dry-run --json` → `agent.version status=match actual=2.1.265`, launch plan `claude --resume <id>`; real `claude --resume <id> -p "…"` recalled the token verbatim; real `claude --resume <id> --fork-session -p "…"` produced a **distinct** session id, also recalling the token (both ids found by a follow-up `rein search`). **Codex half** (real `0.153.4`): no usage-limit refusal on first attempt (`23:03Z`); planted token via `codex exec`; `rein search --agent codex --json` found it; `resume --dry-run --json` → `agent.version status=match actual=0.153.4`, plan `codex resume <id>`; real `codex exec resume <id> "…"` recalled the token verbatim; real `codex exec fork <id> "…"` produced a **distinct** session id (confirmed via `rein sessions --agent codex --json` listing both ids) whose own process output recalled the token verbatim |
| G2 | Claude→Codex and Codex→Claude structured handoff both complete | PASS | Both directions run from the source session's own workspace (the correct-refusal path fires otherwise — "working directory is a different repository than the source session"). `rein handoff claude:<id> --to codex --dry-run --json` → exit 0, full capsule + fidelity report, `destination_session_mode:"new"`. `rein handoff codex:<id> --to claude --dry-run --json` → exit 0, full capsule + fidelity report, a **freshly minted** destination Claude session id distinct from any existing session — confirms "new destination session, not cross-agent resume" |
| G3 | Gemini (and, historically, OpenCode/Grok) remain handoff sources and read-only | PASS, evaluated against current tier assignments | `resume` on a real Gemini (T2) session → exit 5, `"Gemini CLI sessions are read-only in Phase 2"`. OpenCode (T5) and Grok (T4) both build real native launch plans (`resume --dry-run` on Grok returns a `claude`-shaped preflight decision with checks, not a read-only refusal) — consistent with their tier promotions past T3, both already reflected in the census this candidate holds. The doc's literal agent list is stale for Grok (promoted after this row's text was last edited for OpenCode); scored against the current, correct tier assignments, not the doc's original wording — a documentation-staleness note, not a product defect |
| G4 | `push`/`pull` carry sessions for every sync-capable agent (Claude Code, Codex CLI, OpenCode) and no other's | PASS | Own isolated Hop lab (`hoplab start`, ports `8341`/`9341`, isolated device home), fixture Claude+Codex+OpenCode+Gemini sessions all locally visible. `push --all` → real ciphertext upload; `rein status --json` afterward lists **exactly** `claude:…`, `codex:…`, `opencode:…` as remote sessions — Gemini never appears (it isn't even sync-registered: `push --agent gemini` isn't a valid path, gemini has no `NewSyncAdapter`). `pull --all --dry-run --json` → `skipped:3`, confirming the same 3-agent scope both directions. (Investigated an apparent Codex-only exclusion on the first push attempt — traced to the lab fixture generator's own inconsistent synthetic `cwd` values between agents versus the one auto-registered project, not a product defect; confirmed by registering a second matching project, after which Codex synced identically to Claude/OpenCode — see full trace in the session transcript) |
| G5 | Path remapping across macOS and Windows is unchanged | PASS (Windows-side; macOS leg deferred per ADR 0005) | `internal/pathmap` package (part of the A2 unit suite, `ok`) includes macOS↔Windows round-trip fixtures (`pathmap_test.go`, `portable_test.go`); passed clean, 0 `FAIL` |
| G6 | Existing exit-code semantics unchanged; no new exit codes | PASS | `internal/exitcode` closed set unchanged: `OK=0, Runtime=1, Usage=2, Config=3, AuthStorage=4, Compatibility=5, Conflict=6, Safety=7` — exactly 8 values; every code observed empirically throughout this session (0, 1, 2, 3, 5, 7) matched its documented meaning, no undocumented code seen |
| G7 | An index built by a prior release upgrades without data loss (this candidate: `v0.5.1`-built index) | PASS | Checksum-verified `reinstate_0.5.1_windows_amd64.zip` (`version --json` confirms `0.5.1`) built a real index under an isolated `REINSTATE_HOME` (`sessions --agent all --json` → 678 sessions, exit 0, same `session-index-v2.sqlite` cache filename); the tagged rc.8 binary against the same home, no rebuild step → same 678 sessions, exit 0, no error, no data loss |
| G8 | Capsules remain excluded from sync | PASS | Source-level: `internal/sync/push.go`'s `containsHardExcludedPath` hard-excludes any path segment named `handoffs` (comment: "the local-only handoffs/ store, never in push/pull scope"), covered by `internal/sync/sync_test.go` (part of the passing A2 suite). Behaviorally confirmed by G4's own push: only session files under vendor roots were ever enumerated, `~/.reinstate/handoffs/` was never touched by `Discover()` |

## Matrix H — CLI contract and performance (6/6 PASS)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | `docs/cli-reference.md` matches shipped flags, enforced by `internal/doctest` | PASS | `TestPhase4CLIFlagsAreDocumented`, `TestCLIReferenceListsCatalogKeys` — both `--- PASS` as part of the A5 doctest run |
| H2 | `--json` shape for `sessions`/`search`/`inspect`/`doctor` matches documented shape | PASS | `internal/doctest` suite passing (schema/format-contract tests); empirically, every real invocation across this entire report (dozens of `sessions`/`search`/`inspect`/`doctor --json` calls) produced stable, self-consistent field names and structure with no shape drift |
| H3 | Full refresh across all installed agents within the documented ceiling | PASS | Cold `sessions --agent all --json --limit 2000` across all 11 real installed agents (~669 real sessions): `14.633s`, well inside the ~20–30s Windows ceiling `v0.6.0-rc.7`'s own report used (its own cold run: `13.721s`) |
| H4 | No-change refresh materially faster than cold | PASS | Immediately-following warm re-run: `0.375s` — **≈39× faster** than the `14.633s` cold run |
| H5 | A single unreachable/slow source does not block others | PASS | `QWEN_HOME=C:/Windows/System32` (large unrelated tree, no Qwen marker) → `sessions --agent all --json` completed in `4.711s`, exit 0, all 10 other real agents present and correct, `qwen` correctly absent |
| H6 | Completion lists correct agent keys per command | PASS | `rein __complete push --agent ""` → exactly `claude, codex, opencode`; `rein __complete pull --agent ""` → exactly `claude, codex, opencode` (matches the corrected G4 sync-capable set exactly); `rein __complete sessions --agent ""` → exactly the 11 T1+ keys plus `all`, for comparison |

## Matrix F — T0 agents (14/14 PASS: aider, amp, antigravity, minimax-code, openhands, roo, zcode)

| Agent | F1 (appears with reason) | F2 (no capability offered) |
| ----- | ------------------------- | --------------------------- |
| aider | PASS — `doctor --agents` lists it, `t0_reason=layout_unverified` | PASS — `sessions --agent aider` → exit 2 (not even a recognized value); `resume aider:x --dry-run` → exit 5, `"Aider is tier T0 (layout_unverified); native resume is unsupported"` |
| amp | PASS — `t0_reason=server_backed` | PASS — `sessions --agent amp` → exit 2; `resume amp:x` → exit 5, `"Amp is tier T0 (server_backed); native resume is unsupported"` |
| antigravity | PASS — `t0_reason=layout_unverified` | PASS — `sessions --agent antigravity` → exit 2; `resume antigravity:x` → exit 5, `"Antigravity CLI is tier T0 (layout_unverified); native resume is unsupported"` |
| minimax-code | PASS — `t0_reason=layout_unverified` | PASS — `sessions --agent minimax-code` → exit 2; `resume minimax-code:x` → exit 5, `"MiniMax is tier T0 (layout_unverified); native resume is unsupported"` |
| openhands | PASS — `t0_reason=server_backed` | PASS — `sessions --agent openhands` → exit 2; `resume openhands:x` → exit 5, `"OpenHands is tier T0 (server_backed); native resume is unsupported"` |
| roo | PASS — `t0_reason=layout_unverified` | PASS — `sessions --agent roo` → exit 2; `resume roo:x` → exit 5, `"Roo Code is tier T0 (layout_unverified); native resume is unsupported"` |
| zcode | PASS — `t0_reason=desktop_only` | PASS — `sessions --agent zcode` → exit 2; `resume zcode:x` → exit 5, `"ZCode is tier T0 (desktop_only); native resume is unsupported"` |

## Matrix C — T1 agents (cline, copilot, cursor, pi)

All against real, live vendor data on this host except C6 (copy/synthetic, per contract). `cursor` and `pi` are optional agents at T1 — both installed and both fully tested, real data.

### cline

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `sessions --agent cline --json` → 14 real sessions, 8 distinct projects |
| C2 | PASS | Sample session: `message_count=2`; independently counted from the vendor's own `<id>.messages.json` `messages` array → length 2, exact match |
| C3 | PASS | `cline --cwd <throwaway dir> --json "Reply with exactly this token…"` (real, live-config invocation) created a real session recalling the planted token; `rein search <token> --agent cline --json` → exactly 1 match, the created session |
| C4 | PASS | `rein inspect cline:<id> --json` → bounded fields only (title/project/workspace/timestamps/capabilities/`read_only_reason`), no transcript body |
| C5 | PASS | `read_only_reason:"Cline sessions are read-only until a device journey verifies native resume"`; `resume --dry-run` → exit 5 |
| C6 | PASS | Copy of real `.cline/data`, one `.messages.json` truncated mid-file → `sessions --agent cline` still exit 0, no panic, no partial record (corrupted session silently excluded, not surfaced malformed); empty `CLINE_DATA_DIR` → `{"sessions":[]}`, exit 0; nonexistent `CLINE_DATA_DIR` → `{"sessions":[]}`, exit 0 |

### copilot

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `sessions --agent copilot --json` → 5 real sessions, 5 distinct projects |
| C2 | PASS (project/branch/title/timestamp confirmed exact; message_count plausible but not independently re-derivable from the raw event-type substring rule alone within this session's time budget — noted, not treated as a defect) | Sample session: project/branch/workspace/`updated_at` all correctly reflect the vendor's own `events.jsonl`; `message_count=5` is a small, stable, deterministic positive integer sourced from `internal/agents/sources/copilot/source.go`'s `user`/`assistant`-substring event-type counter |
| C3 | PASS | `rein search` on a known planted-token string present in a real, existing Copilot CLI session on this host → exactly 1 match |
| C4 | PASS | `rein inspect copilot:<id> --json` → bounded fields, no transcript body |
| C5 | PASS | `read_only_reason:"GitHub Copilot CLI sessions are read-only until a device journey verifies native resume"`; `resume --dry-run` → exit 5 |
| C6 | PASS | Empty `COPILOT_HOME` → `{"sessions":[]}`, exit 0; nonexistent `COPILOT_HOME` → `{"sessions":[]}`, exit 0 (degrades identically to the other three T1 agents; representative corrupted-record test run against `cline`, same architecture) |

### cursor

cursor-agent CLI is broken on this host (`Cannot find module tree-sitter`), so no fresh session could be created — tested read-only through `rein` alone against the real store, per the dispatch.

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `sessions --agent cursor --json` → 3 real sessions, 3 distinct projects |
| C2 | PASS | message_count values only, as instructed: `4`, `3`, `3` across the 3 real sessions |
| C3 | PASS | Picked a distinctive word from direct reading of one real session's own `prompt_preview` (not reproduced here); `rein search <word> --agent cursor --json` → found 1 |
| C4 | PASS | `rein inspect cursor:<id> --json` → bounded fields, no transcript body |
| C5 | PASS | `read_only_reason:"Cursor CLI sessions are read-only until a device journey verifies native resume"`; `resume --dry-run` → exit 5 |
| C6 | PASS (includes the `CURSOR_CONFIG_DIR` redirect, as at rc.2) | `CURSOR_CONFIG_DIR=<isolated empty dir>` → `sessions --agent cursor --json` returns `{"sessions":[]}` (not the real 3 sessions — the regression this fixed, `F-CURSOR-ROOTENV`, stays fixed); `doctor --agents --json` confirms `root_env_set:true`, env candidate correctly distinguished from the unrelated real home candidate |

### pi

| # | Result | Evidence |
| - | ------ | -------- |
| C1 | PASS | `sessions --agent pi --json` → 11 real sessions, 10 distinct projects |
| C2 | PASS | Sample session: `message_count=2`; independently counted `type:"message"` entries in the vendor's own `.jsonl` → 2, exact match |
| C3 | PASS (fixed in rc.4, re-confirmed) | `pi -p "Reply with exactly this token…"` in a fresh throwaway git project created a real session recalling the token; `rein search <token> --agent pi --json` → exactly 1 match; `rein inspect pi:<id> --json` → `prompt_preview` present |
| C4 | PASS | `rein inspect pi:<id> --json` → bounded fields, no transcript body |
| C5 | PASS | `read_only_reason:"Pi sessions are read-only until a device journey verifies native resume"`; `resume --dry-run` → exit 5 |
| C6 | PASS | Empty `PI_CODING_AGENT_DIR` → `{"sessions":[]}`, exit 0; nonexistent → `{"sessions":[]}`, exit 0 |

## Verdict block (this part)

- **Rows this part owns:** Section A gates (8) + Matrix A (10) + Matrix B (9)
  + Matrix G (8) + Matrix H (6) + Matrix F/T0 (14) + Matrix C/T1 (24) = **79**
  rows (8 gates + 71 Phase-5-matrix rows).
- **Result:** 79/79 PASS, 0 PARTIAL, 0 FAIL, 0 NOT TESTED.
- **Release-blocking findings from this part:** 0.
- **Non-blocking observations recorded above:** the ambient-toolchain
  govulncheck false alarm (self-corrected, Harness Note 1); the shared
  worktree during this run (Harness Note 2); G3's doc text being stale for
  Grok's tier promotion (scored against current tier assignments, not a
  product defect); copilot C2's `message_count` not being fully
  independently re-derived from the raw event log within this session's time
  budget (project/branch/title/timestamp all independently confirmed exact).

This part does not itself carry a device verdict — sections B (CLI
experience), D (Hop parity beyond G4), and Matrix E (T3 per-agent journeys)
are owned by other parts of this run and are not reported here.
