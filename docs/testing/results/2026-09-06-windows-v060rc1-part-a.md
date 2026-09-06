# v0.6.0-rc.1 tagged Windows acceptance — part A (gates, core, T0/T1)

`PHASE5-DEVICE-REPORT-V1` (subset). This is executor A's part file for the
tagged-artifact run against `v0.6.0-rc.1`. It covers section A automated
gates (A1-A8), the Phase 5 core matrices A/B/G/H, the T0 agent matrix F, and
the T1 agent matrix C (agents `cline`, `copilot`, `cursor`, `pi`). Sections
B's per-T2/T3/T4/T5 rows, section C (CLI experience), and section D (Hop
journeys) are other executors' part files against this same tag.

## Deviation from dispatch

The dispatch (`docs/testing/v0.6.0-rc.1-agent-verification-prompts.md`) asks
for an install from the live bootstrap (reinstate.dev) after proving it pins
`v0.6.0-rc.1`. At run time the live bootstrap still pins `v0.5.2-rc.1`
because the guarded website deploy's test gate fails on three Windows-only
test files (`website/src/lib/api-routes.test.ts` and
`website/src/lib/repository-social-preview.test.ts`, per this contract's own
Run Notes, plus a third file this deviation note generalizes over). The
install used for this report therefore came from the published GitHub
prerelease assets (`reinstate_0.6.0-rc.1_windows_amd64.zip`), whose sha256
was independently re-verified against `checksums.txt` and whose GitHub
attestation had already been verified by the coordinator, rather than from
the live bootstrap.

## Artifact identity

| Field | Value |
| ----- | ----- |
| Tag | `v0.6.0-rc.1` (published GitHub prerelease, non-draft, `isPrerelease: true`, `targetCommitish: main`, confirmed via `gh release view`) |
| Full commit | `63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0` |
| Windows archive | `reinstate_0.6.0-rc.1_windows_amd64.zip` |
| Archive SHA-256 | `3d1ebf243c6d6f234ffe95955b9504502c340bfda16f42f776db97893bc26542` — matches `checksums.txt`, re-verified by this executor with `sha256sum` before extracting |
| Installed binary SHA-256 | `d13d683eda6e1d59082be08c0326be196ffaa856cb8b83a35ef181d3ec624e2c` — `rein.exe` and `reinstate.exe` byte-identical (`cmp` exit 0) |
| `rein version --json` | `{"commit":"63ac5a5b02bacc5ad52ea0826b4d3d77fdc597e0","date":"2026-09-06T11:17:45Z","name":"reinstate","version":"0.6.0-rc.1"}` — both binary names report the same document |
| Install location | `D:\ReinstateAcceptanceProjects\v060-w7b-A\install\` (this executor's own fresh directory; no user binary replaced) |
| Bootstrap deviation | see above — install came from published release assets, not the live bootstrap, because reinstate.dev still pins `v0.5.2-rc.1` |

## Host (sanitized)

| Field | Value |
| ----- | ----- |
| OS | Windows 11 Pro, `10.0.26200`, native `windows/amd64`, never WSL |
| Git | `2.52.0.windows.1` |
| Go toolchain | `go1.25.13` (pinned by `go.mod`/`GOTOOLCHAIN`; host default `go1.26.1`) |
| Worktree | `v060-w7b-tagged`, branch `v060/w7b-tagged` at `63ac5a5b` (release commit itself) |
| UTC date | 2026-09-06 |
| Environment hygiene | every shell in this part ran with `REINSTATE_BACKEND` and `REINSTATE_MEMORY_BACKEND_DIR` unset (confirmed empty at the start of each shell); `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left at the host's live values throughout, per the ground rules — isolation for a row used the vendor's own declared `root_env` variable instead |
| Vendor versions re-checked | Claude Code `2.1.263`, Codex CLI `0.149.0`, OpenCode `1.18.27`, Grok `1.0.5`, Qwen `0.21.12`, Gemini CLI `0.53.0`, Kimi Code CLI `0.36.1`, Copilot CLI `1.0.80`, Cursor CLI `3.19.7`, Cline `3.0.61`, Pi `0.73.1` — all match or fall within the versions this candidate verifies |
| Catalog agent count | 18 |
| Catalog tier census | T0: 7, T1: 4, T2: 2, T3: 0, T4: 2, T5: 3 |
| Generated required row count | `rein doctor --agents --acceptance-matrix --json`: `row_count: 178`, `core_row_count: 33` (A:10, B:9, G:8, H:6) |

## Verdict for this part

- **Rows in this part:** 79 (8 gates + 33 core + 14 Matrix F + 24 Matrix C)
- **PASS:** 73
- **FAIL:** 6
- **PARTIAL:** 0
- **NOT TESTED:** 0
- **Release-blocking candidate findings:** 1 (F-CURSOR-ROOTENV, see Findings)

`PARTIAL` and `NOT TESTED` do not pass a required row; none were recorded in
this part — every row below was either fully exercised (`PASS`/`FAIL`) or,
where a mechanism genuinely could not be exercised, that is called out in the
row's evidence and the row is scored `FAIL` rather than left unscored.

## Section A — Automated gates

All commands ran from the worktree root with `GOTOOLCHAIN=go1.25.13` and
`REINSTATE_BACKEND`/`REINSTATE_MEMORY_BACKEND_DIR` unset. A1-A5 and A8 ran
with `-p 4` from the worktree; A6/A7 ran against the coordinator-staged
release directory (`…\scratchpad\rc1-draft`) via `scripts/verify-release.ps1
-DistDir` and `scripts/test-install.ps1 -DistDir`.

**Harness note.** Two untracked, uncommitted Go files were already present in
this shared worktree at the start of this part
(`cmd/h3probe/main.go`, `cmd/hopfdrunner/main.go`) — self-documented in their
own header comments as "a throwaway acceptance-test helper (not part of the
product; deleted from the worktree before the results commit)", evidently
left behind by another in-flight executor's Hop-journey work on this same
shared worktree (Section D is a different executor's assignment). They were
**not deleted** by this executor, to avoid destroying another executor's
in-progress state; instead their effect on `./...`-scoped gates was isolated
and confirmed structurally (below). `git status --porcelain` at the end of
this part shows only these two pre-existing untracked paths, nothing else.

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| A1 | Format, vet, tidy | PASS | `gofmt -l .` empty (exit 0); `go vet -p 4 ./...` exit 0, no output; `go mod tidy -diff` exit 0, empty diff |
| A2 | Unit suite | PASS | `CGO_ENABLED=0 go test ./... -count=1 -p 4`: every package `ok` or `[no test files]`, exit 0; `internal/cli` 65.9s was the longest package |
| A3 | Race suite | PASS | `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p 4`: every package `ok`, exit 0; `internal/cli` 97.8s longest |
| A4 | Lint and vuln | PASS (harness note) | `make lint` on the literal `./...` tree fails (3 `errcheck` issues) — all 3 are inside the two untracked non-product helper files above, none inside `cmd/reinstate`, `internal`, or `scripts`. Re-run scoped to `./cmd/reinstate/... ./internal/... ./scripts/...` (the actual release/test module tree): `0 issues`, exit 0. `make vuln` (`govulncheck ./...`, unaffected by the untracked files): `0 vulnerabilities` in code, 4 in unused transitive dependencies, exit 0 |
| A5 | Doc gate | PASS | `go test ./internal/doctest/... -count=1 -v`: 18 tests, all PASS, exit 0 (includes `TestPhase4CLIFlagsAreDocumented`, `TestCLIReferenceListsCatalogKeys`, `TestReleaseAndSupportClaims`, `TestWebsiteReleaseTruthStaysSynchronized`); `scripts/check-docs.ps1` exit 0 |
| A6 | Snapshot and artifacts | PASS | `scripts/verify-release.ps1 -DistDir <rc1-draft>`: `"release artifacts verified (PowerShell): <rc1-draft>"`, exit 0 (this wraps `check-release-artifacts.ps1` in full parity per the script's own header) |
| A7 | Installers | PASS | `scripts/test-install.ps1 -DistDir <rc1-draft>`: re-runs verify-release.ps1 (exit 0) then `go test ./internal/doctest -run TestInstaller -count=1`: `ok`, exit 0 |
| A8 | Cross-OS build | PASS (harness note) | `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./...` and the `linux/amd64` equivalent both fail on the literal `./...` tree — both failures are `cmd/hopfdrunner` (the same untracked helper) importing `golang.org/x/sys/windows`, which cannot cross-compile by design. Re-run scoped to `./cmd/reinstate/... ./internal/... ./scripts/...`: both `darwin/arm64` and `linux/amd64` build clean, exit 0 |

## Matrix A — Catalog integrity

Evidence source: `rein doctor --agents --json` (schema `AGENT-PROBE-V1`) and
`rein doctor --agents` (human table) on the installed candidate binary, with
`rein`/`reinstate` both confirmed on `PATH` ahead of any other installed
release before every command in this part.

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| A1 | Descriptors validate at start; no duplicate or empty keys | PASS | 18 agent keys returned, 18 unique, none empty |
| A2 | `doctor --agents` lists every agent including T0 | PASS | Human table lists all 18, including all 7 T0 agents (`aider`, `amp`, `antigravity`, `minimax-code`, `openhands`, `roo`, `zcode`) with `installed: no`, `root: no` |
| A3 | Every T0 agent shows an enumerated reason | PASS | Human table `notes` column: `t0_reason=layout_unverified` (aider, antigravity, minimax-code, roo), `t0_reason=server_backed` (amp, openhands), `t0_reason=desktop_only` (zcode) — a closed, three-value enumeration |
| A4 | Declared tier and capability constructors agree | PASS | `go test ./internal/agents/conformance/... -run TestShippedAgentsConformance -v`: subtests for claude/codex/gemini/opencode/grok/kimi all PASS |
| A5 | Every declared evidence path exists in the tagged tree | PASS | `go test ./internal/agents/conformance/... -run TestShippedEvidenceIsComplete -v`: PASS |
| A6 | `sessions --agent` accepts exactly T1+ keys, else exit `2` | PASS | `rein sessions --agent aider --json` (T0): exit `2`, `"invalid agent; expected claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen, or all"` — the accepted list is exactly the 11 T1+ keys; `rein sessions --agent cline --json` (T1): exit `0` with real session data; an unknown key also exit `2` with the same message |
| A7 | `resume` accepts exactly T3+ keys, else exit `5` with reason | PASS | `rein resume aider:x --dry-run --json` (T0): exit `5`, `"...Aider is tier T0 (layout_unverified); native resume is unsupported"`; `cline:x` (T1): exit `5`, tier-named reason; `gemini:x` (T2): exit `5`, tier-named reason; `claude:doesnotexist` (T5, valid tier): exit `2` (`session not found`, i.e. it clears the tier gate and fails on lookup instead — proving T3+ keys are accepted past the gate) |
| A8 | Agent ordering deterministic across runs | PASS | Two independent `rein doctor --agents --json` invocations: identical key ordering, byte-for-byte after excluding nothing (order list compared programmatically, equal) |
| A9 | Broken evidence path fails conformance | PASS | `go test ./internal/agents/conformance/... -run "TestBrokenEvidencePathFails|TestBrokenEvidencePathIsCaught" -v`: both PASS |
| A10 | No scan writes, renames, or locks under any agent root | PASS | Live filesystem audit: recursive file count, total size, and max `LastWriteTimeUtc` for four real T1 agent roots (`.cline\data`, `.copilot`, `.cursor`, `.pi\agent`) captured before and after a full `rein sessions --agent all --json` refresh (12.8s) — all three metrics identical before/after for all four roots. Also: `go test ./internal/agents/conformance/... -run "TestIsolationFSRejectsWritesAndOutsideRoot|TestIsolationFSOpenFileWriteRejected" -v`: both PASS |

## Matrix B — Probe and redaction

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | Output conforms to `AGENT-PROBE-V1` | PASS | `doctor --agents --json` top-level `"schema": "AGENT-PROBE-V1"`; structure matches the documented shape (`schema`, `generated_at`, `platform`, `reinstate_version`, `agents[]`) |
| B2 | No absolute path, home name, or username | PASS | Full-document scan for the host username and Windows absolute-path patterns (`C:\Users\...`, `C:/Users`): zero hits |
| B3 | No JSON values from session files, only keys | PASS | `first_line_keys` fields list field **names** only (e.g. `cwd`, `id`, `timestamp`, `type`, `version`; `createdAtMs`, `hasConversation`, `schemaVersion`); spot-checked every occurrence of ambiguous substrings (`token`, `cache`) in the document — all were field names (`contextTokensUsed`, `avgTimeToFirstTokenMs`, `models_cache.json` as a filename) or path-shape components, never a value from a session |
| B4 | Names shape-normalized | PASS | `name_shapes` fields show normalized tokens only: `<slug>`, `<32-hex>`, `<uuid-v4>`, `<slug>-<uuid-v4>.jsonl`, `<n>` — no raw UUID, hash, or real path segment observed anywhere in the document |
| B5 | No excluded subtree appears anywhere in output | PASS | Cross-referenced each of `claude`, `codex`, `cline`, `copilot`, `cursor`, `pi`, `grok`'s declared `Excluded` pattern lists (read from `internal/agents/sources/*/source.go`) against every `"path"` value in the probe document: no declared-excluded filename (`auth.json`, `credentials.json`, `.env`, `cache/**`, `mcp-secrets`, `session-store.db*`, `mcp.json`, `hooks.json`, etc.) appears as an exact path/leaf match anywhere; every substring hit on a generic word (`logs`, `cache`, `git`, `hooks.json`) resolved to a *different*, non-excluded path under a *different* agent whose own exclusion list does not name it (e.g. `codex`'s tree legitimately shows `hooks.json` because codex's exclusion list only names `auth.json`/`.env`/`cache/**`, not `hooks.json`) |
| B6 | Absent agent reported without error | PASS | All 7 T0 agents (`installed: false`/absent) reported cleanly inside the same successful `doctor --agents --json` run, exit 0, no error |
| B7 | Empty root distinguishable from absent root | PASS | Same-run `candidate_roots[]` entries carry independent `exists`/`marker_present` booleans per candidate: `kimi`'s `.kimi` candidate shows `exists: true, marker_present: false` (present but unmarked/empty) in the same document where `antigravity`'s `.gemini/antigravity-cli` and `claude`'s/`codex`'s secondary `.config/*` candidates show `exists: false` (absent) — the two states are distinguishable in one live run without any override needed |
| B8 | Wrapper scripts match binary output | PASS | `scripts/testing/agent-storage-probe.sh` and `.ps1` both `exec`/invoke the installed binary directly; live runs of each compared to a direct `doctor --agents --json` call are identical after excluding the `generated_at` timestamp (the only per-invocation field) |
| B9 | Secret scanner passes over committed probes | PASS | `internal/secretscan.Scan` run (via a temporary in-tree driver, deleted immediately after use) over all 26 files under `docs/testing/results/agent-probes/`: `scanned 26 files, 0 with matches`, exit 0 |

## Matrix G — No regression

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude Code and Codex CLI native resume and fork unchanged | PASS | Isolated `CLAUDE_CONFIG_DIR` seeded only with `testdata/sessionindex/claude/windows`'s committed fixture (never real `~/.claude`): `resume --dry-run --json` → `executable: "claude"`, `args: ["--resume","claude-syn-windows"]`; `fork --dry-run --json` → `args: [..., "--fork-session"]`; `agent.version` matched the live `2.1.263`. Isolated `CODEX_HOME` seeded only with `testdata/adapters/codex/windows`'s committed fixture: `resume --dry-run --json` → `executable: "codex"`, `args: ["resume","rollout-syn-001"]`; `fork --dry-run --json` → `args: ["fork","rollout-syn-001"]`. Both argv shapes match `docs/cli-reference.md`'s documented table exactly |
| G2 | Both structured handoff directions complete | PASS | Same isolated homes: `rein handoff claude:claude-syn-windows --to codex --dry-run --json` exit 0, full capsule+fidelity document, `destination_session_mode: "new"`; `rein handoff codex:rollout-syn-001 --to claude --dry-run --json` exit 0, same shape. Both directions produce a `structured handoff` / `new` destination session (never a cross-agent resume), matching the dispatch's framing |
| G3 | Gemini remains a source and read-only (Grok has since been promoted past this gate on this candidate) | PASS | `rein resume gemini:x --dry-run --json`: exit `5`, `"...Gemini CLI is tier T2; native resume is unsupported"`. Grok is declared `T4` in this candidate's catalog (census above) — the contract's original G3 wording predates that promotion; per the dispatch's own instruction to read G3 against the candidate's current census, Gemini is the row's live subject this candidate and it verifies clean |
| G4 | Push and pull carry only the T5 agents (Claude, Codex, and — since this candidate — OpenCode), no other agent's | PASS | `push --agent`'s completion/registry is built from `agents.Capable(agents.CapabilitySync)` (`internal/cli/commands_impl.go:defaultRegistry`), not a hardcoded list — dynamically scoped to catalog `T5` declarations, currently `claude`/`codex`/`opencode`. `internal/cli/opencode_sync_test.go` directly exercises `push --agent opencode`, and `internal/cli/hop_first_push_test.go`/`hop_first_push_acceptance_test.go` exercise all three agents together — both already passing under A2/A3 |
| G5 | Cross-OS path remapping unchanged | PASS | `go test ./internal/pathmap/... -v`: `TestRoundTripPOSIX`, `TestWindowsDrive`, `TestSpacesAndUnicode`, `TestNormalizePortable*`, `TestNormalizeKeepsUnmatchedPathsForVendorRewriting`, `TestIsToken` all PASS. Full physical Windows↔macOS proof is section D/H11's mechanism (different executor); this row's "unchanged" claim rests on the unit-level remap engine, which is what G5 (as distinct from H11) actually asserts |
| G6 | Exit-code semantics unchanged; no new codes | PASS | `internal/exitcode` defines exactly 8 codes (`0,1,2,3,4,5,6,7`); `docs/cli-reference.md`'s exit-code table lists exactly the same 8 with the same meanings — no undocumented code observed anywhere across this entire part's ~150 live command invocations |
| G7 | A prior-stable index upgrades without data loss | PASS | Built a 100-session index (11 agents, real host roots) with the verified `v0.5.1` release binary (`e8d1ec28`, checksum-verified against its own `checksums.txt`) under an isolated `REINSTATE_HOME`; reopened with the `v0.6.0-rc.1` candidate binary against the same home: `100/100` sessions, identical per-agent breakdown. Reopened again with the `v0.5.1` binary afterward: still `100/100` — forward- and backward-compatible, no loss |
| G8 | Capsules remain excluded from sync | PASS | A real (non-dry-run, `--no-launch`) `claude→codex` structured handoff was created (`handoff_id: 43db733df59ccb18f17bc0db050eff68`). It appears in `rein handoff list --json`'s own lineage listing but is absent from `rein sessions --json`'s listing of the same home (`grep` for the handoff id in the sessions output: 0 matches) — the session index that `push`/`pull` operate over never sees it. Structurally, `internal/sync/*.go` contains zero references to capsules at all |

## Matrix H — CLI contract and performance

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | CLI reference matches shipped flags | PASS | `go test ./internal/doctest/... -run "TestCLIReferenceListsCatalogKeys|TestPhase4CLIFlagsAreDocumented" -v`: both PASS; independently, this part's live `claude`/`codex` resume/fork argv (G1) matched `docs/cli-reference.md`'s documented table exactly |
| H2 | `--json` shapes match documentation | PASS | Live `sessions`, `search`, `inspect`, `doctor --agents` all produce the documented field sets (`inspect` correctly shows a capped `prompt_preview`, never a full transcript, matching the "Phase 2 has no full-transcript dump" documentation); covered structurally by the full `internal/cli` unit/golden suite passing under A2/A3 |
| H3 | Full refresh within a reasonable ceiling | PASS | Cold full refresh across all installed agents (fresh isolated `REINSTATE_HOME`, 11 agents, real sessions): `13.978s` |
| H4 | No-change refresh materially faster | PASS | Same isolated home, immediately re-run twice: `4.158s`, then `3.740s` — roughly 70% faster than the cold run. (The residual ~4s floor is native vendor `--version` subprocess probes for the T3+ agents, not index I/O.) |
| H5 | One slow/unreachable source does not block others | PASS | Pointed `CODEX_HOME` at a file (not a directory) to force a hard-broken source, then ran `rein sessions --agent all --json`: exit `0`, every other installed agent's real sessions still returned (codex alone absent) — one broken source did not block or error the rest |
| H6 | Completion lists correct agent keys per command | **FAIL** | `rein __complete sessions --agent ""` and `rein __complete search --agent ""` both correctly list exactly the 11 T1+ keys; `rein __complete handoff --to ""` correctly lists exactly the 5 T4+ keys (`claude, codex, grok, opencode, qwen`). But `rein __complete push --agent ""` and `rein __complete pull --agent ""` both return `ShellCompDirectiveDefault` with **no candidates at all** — neither command has a `RegisterFlagCompletionFunc` wired for `--agent`, so shell completion offers nothing (falls back to file-path completion) for exactly the two commands whose `--agent` scope is narrowest (T5 only) and arguably most in need of a visible, correct key list. Not release-blocking by itself (no wrong values are offered, only none), but the row's assertion — "the correct agent keys per command" — fails for `push`/`pull` |

## Matrix F — Per T0 agent (`aider`, `amp`, `antigravity`, `minimax-code`, `openhands`, `roo`, `zcode`)

| Agent key | F1 listed with reason | F2 no capability offered or implied |
| --------- | --------------------- | ------------------------------------ |
| `aider` | PASS | PASS |
| `amp` | PASS | PASS |
| `antigravity` | PASS | PASS |
| `minimax-code` | PASS | PASS |
| `openhands` | PASS | PASS |
| `roo` | PASS | PASS |
| `zcode` | PASS | PASS |

**F1 evidence (all 7):** each appears in `rein doctor --agents`'s human table
with `installed: no`, `root: no`, and a `t0_reason` note from the closed
enumeration (see Matrix A3 above for the exact values per agent).

**F2 evidence (all 7):** for every one of the 7 keys, three independent
commands were run and all three refused cleanly:
- `rein sessions --agent <key> --json` → exit `2`, `"invalid agent; expected
  claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi,
  qwen, or all"` (the key is not even in the accepted list)
- `rein resume <key>:x --dry-run --json` → exit `5`,
  `"...<DisplayName> is tier T0 (<reason>); native resume is unsupported"`
- `rein handoff claude:claude-syn-windows --to <key> --dry-run --json` →
  exit `2` (not an accepted handoff destination)

No output across any of the 21 commands implied an upcoming capability for
any T0 agent.

## Matrix C — Per T1 agent

Per the ground rules, C1-C4 ran read-only through `rein` alone against the
host's real root (this is the documented definition of the T1 discovery
rows); C5 used the same real data; C6 used an isolated root pointed at by
each agent's own declared `root_env` variable and, where noted, a **copy**
of one real session file, never the live tree itself. No title, prompt,
path, or identifier from a session this executor did not itself create or
that was not already a matter of committed-fixture record appears below;
session ids quoted are opaque UUIDs/vendor-assigned tokens, and the only
prompt-shaped strings quoted (e.g. `"Reply with the single word: ok"`) are
short, generic acceptance-probe instructions already present in this
program's own prior committed evidence, not private content.

### Agent: `cline`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from two or more projects | PASS | `rein sessions --agent cline --json`: 3 sessions across 2 distinct projects |
| C2 | Metadata matches what the agent itself shows | **FAIL** | `updated_at` and `title` matched the vendor's own `sessions.db`/`.json` metadata exactly for the spot-checked session, but `message_count` was `0` for all 3 sessions while the vendor's own `<id>.messages.json` sidecar (opened directly, read-only, real data) contains 1 real message for each. Source review confirms this is not a documented exclusion: `internal/agents/sources/cline/source.go` never assigns a `MessageCount` field at all (stays at the Go zero value), unlike `pi`/`copilot`'s sources, which do |
| C3 | Search finds a known string by prompt, and by file where recorded | **FAIL** | A search that happened to match the session's *title* returned the session, but source review shows `cline`'s `SearchText` is built from `id, title, project, workspace` only (`internal/agents/sources/cline/source.go:163`) — the vendor's own message-body content (confirmed present, above) is never included, unlike `pi`/`copilot` which explicitly append `parsed.prompts.String()` |
| C4 | Inspect is bounded and prints no transcript body | PASS | `rein inspect cline:<id> --json`: bounded fields only, capped `prompt_preview`, no transcript body |
| C5 | Below T3, read-only reason present, resume refused exit `5` | PASS | Every session record carries `read_only_reason: "Cline sessions are read-only until a device journey verifies native resume"`; `rein resume cline:x --dry-run --json` exit `5` |
| C6 | Corrupt, empty, and absent roots degrade cleanly | PASS | Isolated `CLINE_DATA_DIR` pointed at an empty directory and at a nonexistent directory: both returned `{"sessions": []}`, exit `0`. A **copy** of one real session's `db/sessions.db` plus its `<id>.json`/`<id>.messages.json` sidecars, with the copied database's `messages_path` column repointed at the copy and the copied `.messages.json` truncated mid-record: `rein sessions --agent cline --json` still returned the session cleanly, exit `0`, no panic, no partial record (consistent with C3's finding — the sidecar's content is never opened at all, so its corruption has no effect either way) |

### Agent: `copilot`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from two or more projects | PASS | 2 sessions across 2 distinct projects |
| C2 | Metadata matches what the agent itself shows | PASS | `message_count: 5` for the spot-checked session is derived from real `events.jsonl` content (`internal/agents/sources/copilot/source.go` computes `parsed.messages` from the vendor event stream, confirmed non-zero and non-hardcoded); `title`, `project`, `branch`, `updated_at` all matched the vendor's own `workspace.yaml`/`events.jsonl` |
| C3 | Search finds a known string by prompt, and by file where recorded | PASS | `SearchText` construction includes `parsed.prompts.String()` (`internal/agents/sources/copilot/source.go:168`) — confirmed by source, and live search by a known project token returned the correct session |
| C4 | Inspect is bounded and prints no transcript body | PASS | Bounded fields, capped preview |
| C5 | Below T3, read-only reason present, resume refused exit `5` | PASS | `read_only_reason: "GitHub Copilot CLI sessions are read-only until a device journey verifies native resume"`; resume exit `5` |
| C6 | Corrupt, empty, and absent roots degrade cleanly | PASS | Isolated `COPILOT_HOME` pointed at an empty directory and at a nonexistent directory: both `{"sessions": []}`, exit `0` — genuine isolation confirmed (unlike `cursor`, see below), since `COPILOT_HOME` is correctly wired as `RootEnv` in `internal/agents/sources/copilot/source.go:103` |

### Agent: `cursor`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from two or more projects | PASS | 2 sessions across 2 distinct projects |
| C2 | Metadata matches what the agent itself shows | **FAIL** | `message_count: 0` for both sessions, while each session's own `store.db` (opened directly, read-only, real data) contains 11 rows in its `blobs` table — non-zero real content. `size_bytes` also reflects only the tiny `meta.json` sidecar (145/140 bytes), never `store.db` (86016 bytes), confirming `store.db` is never opened for metadata at all |
| C3 | Search finds a known string by prompt, and by file where recorded | **FAIL** | Same root cause as C2/cline: `SearchText` is built from `id, title, project, workspace` only (`internal/agents/sources/cursor/source.go:173`), never message content |
| C4 | Inspect is bounded and prints no transcript body | PASS | Bounded fields, capped preview |
| C5 | Below T3, read-only reason present, resume refused exit `5` | PASS | `read_only_reason: "Cursor CLI sessions are read-only until a device journey verifies native resume"`; resume exit `5` |
| C6 | Corrupt, empty, and absent roots degrade cleanly | **FAIL** | Setting `CURSOR_CONFIG_DIR` to an empty directory, and separately to a nonexistent directory, had **no effect** on `rein sessions --agent cursor --json` — it kept returning the same 2 real sessions from the true host root both times, proving the override is not honored by the discovery path (see finding **F-CURSOR-ROOTENV** below). Because this candidate's own isolation mechanism does not work for `cursor`, this executor could not exercise C6's required "against a copy, never the tester's real agent data" — the row's mechanism was not exercised, so it does not pass, independent of whether any crash would have occurred |

### Agent: `pi`

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| C1 | Lists real sessions from two or more projects | PASS | 3 sessions across 2 distinct projects |
| C2 | Metadata matches what the agent itself shows | PASS | Spot-checked session: raw `.jsonl` file has exactly 2 `message`-type events (of 5 total events incl. `session`/`model_change`/`thinking_level_change`); rein reports `message_count: 2` — exact match. `updated_at` (`2026-08-17T11:49:09Z`) matches the last raw event's timestamp (`11:49:09.744Z`) to the second |
| C3 | Search finds a known string by prompt, and by file where recorded | PASS | `SearchText` includes `parsed.prompts.String()` (`internal/agents/sources/pi/source.go:146`); confirmed by source and by a live search hit |
| C4 | Inspect is bounded and prints no transcript body | PASS | Bounded fields, capped preview, `read_only_reason` present |
| C5 | Below T3, read-only reason present, resume refused exit `5` | PASS | `read_only_reason: "Pi sessions are read-only until a device journey verifies native resume"`; resume exit `5` |
| C6 | Corrupt, empty, and absent roots degrade cleanly | PASS | Isolated `PI_CODING_AGENT_DIR` pointed at an empty directory and at a nonexistent directory: both `{"sessions": []}`, exit `0` — genuine isolation confirmed (`PI_CODING_AGENT_DIR` correctly wired as `RootEnv` in `internal/agents/sources/pi/source.go:95`) |

## Findings

| ID | Severity | Matrix row | Description | Release blocking |
| -- | -------- | ---------- | ------------ | ----------------- |
| F-CURSOR-ROOTENV | **MAJOR** (candidate for BLOCKER — recommend coordinator review) | `cursor:C6`, and any other row/report that relied on `CURSOR_CONFIG_DIR` to isolate Cursor CLI testing from the real host root | `internal/agents/sources/cursor/source.go`'s `config()` (lines 108-120) builds its `hometree.Config` without setting the `RootEnv` field at all, unlike every sibling T1 source (`cline`, `copilot`, `pi` all set `RootEnv: "<VAR>"` at the equivalent call site). Confirmed live: `rein doctor --agents --json`'s probe path correctly shows `resolved_root: null` for an empty/absent `CURSOR_CONFIG_DIR` override (a *different* code path, `internal/agents/probe/collect.go`, which resolves the override independently and correctly), but `rein sessions --agent cursor --json` (and by extension `search`/`inspect`/`resume`/`fork` for cursor, which share the same `defaultLocalSources()`/`NewIndexSource(agents.Env{})` call site in `internal/cli/sessions.go:376`) silently ignores the variable and always reads the real `~/.cursor`. This means the documented isolation mechanism the whole acceptance program (and this candidate's own ground rules) relies on to test Cursor CLI without touching a developer's real data **does not work** for any command except `doctor --agents`. Not caught by the existing test suite: no conformance test asserts `RootEnv` is set for every source, and `cursor`'s own `source_test.go` has no test exercising the override. | NO (does not block *this* release mechanically — no exploit path, no data written; the real risk is to *testers*, not end users, since it silently defeats an isolation control this program depends on) but flagged for coordinator judgment given how many prior/parallel acceptance passes may have unknowingly exercised the real host root under a `CURSOR_CONFIG_DIR` override believing it was isolated |
| F-CLINE-CURSOR-MSGCOUNT | MINOR | `cline:C2`, `cursor:C2` | `message_count` is unconditionally `0` for both `cline` and `cursor` sessions — never derived from the vendor's own message-bearing file (`*.messages.json` for cline, `store.db` for cursor), confirmed against real content on this host. `pi` and `copilot` (the other two T1 agents) correctly compute a non-zero count from equivalent vendor content, so this is not an inherent T1 limitation | NO |
| F-CLINE-CURSOR-SEARCH | MINOR | `cline:C3`, `cursor:C3` | Search for `cline` and `cursor` only indexes `id, title, project, workspace` — never message/prompt body content, confirmed by source (`SearchText` construction) and consistent with the C2 finding above. `pi` and `copilot` include real prompt text in `SearchText`. This mirrors the pretag report's own pre-existing, disclosed `opencode:C3` finding (search-by-title-only, not full body) but is undocumented for `cline`/`cursor` specifically | NO |
| F-COMPLETION-PUSHPULL | MINOR | `H6` | `push --agent` and `pull --agent` have no shell-completion candidates at all (no `RegisterFlagCompletionFunc` wired), while `sessions --agent`, `search --agent`, and `handoff --to` all correctly complete to their tier-scoped key lists | NO |

## Methodology notes (disclosed)

- **Capability-baseline cross-contamination, disclosed rather than hidden.**
  `resume`/`fork`/`handoff` dry-runs for a `codex`-source or `codex`/`claude`-
  destination operation legitimately surface the live `CLAUDE_CONFIG_DIR`'s
  full skill inventory inside the `environment.capabilities`/fidelity
  document — this is documented, intentional behavior (`docs/cli-reference.md`:
  "a subsequent preflight can compare repository identity, ... capabilities,
  and recognized runtimes with that private baseline"), not a product defect,
  and it happens because the ground rules correctly forbid unsetting the
  host's live `CLAUDE_CONFIG_DIR`. Two local files that captured this host's
  real, private skill-name inventory as a side effect of a G1/G2 dry-run were
  deleted immediately after the needed structural fields (`executable`,
  `args`, `operation`, `decision`, `mode`, `fidelity`) were extracted; no
  skill or plugin name from that inventory appears anywhere in this report.
- **Absolute-path Python readback quirk.** Several `python3 -c` invocations
  against files addressed by an absolute `/d/...` Git Bash path
  intermittently raised `FileNotFoundError` even though the same path read
  fine with `cat`/`wc`/`head` in the same shell; every case was worked around
  by `cd`-ing into the directory and using a relative filename. This looks
  like a harness/tool quirk of this session's Python invocation, not a
  filesystem or product issue — flagged for completeness, not as a finding
  against the candidate.

## Cleanup

All isolated homes, throwaway git projects, copied/truncated fixture files,
and the temporary in-tree secret-scanner driver (deleted immediately after
use, never committed) lived under `D:\ReinstateAcceptanceProjects\v060-w7b-A\`
on the test host and are deleted at the end of this part; the unzipped
install itself is left in place as the artifact under test. The two
untracked non-product helper files left by another executor
(`cmd/h3probe/`, `cmd/hopfdrunner/`) were **not** touched by this executor.
No transcript text, real prompt, real response, credential value, private
path, or vendor skill/session name from a developer's real tree appears
above; every session id, handoff id, and timestamp quoted originates from a
committed `testdata/` fixture, a real session this executor could attribute
to prior committed acceptance-probe activity on this shared host (opaque
UUIDs/vendor tokens only, per the T1 discovery evidence policy), or a
process/session this executor created itself in its own throwaway lab
directory.

## Terminated block for this part

> Part-A testing (section A gates, Phase 5 core matrices A/B/G/H, Matrix F
> for all 7 T0 agents, Matrix C for `cline`/`copilot`/`cursor`/`pi`) is
> terminated at `MATRIX_COMPLETE` for this executor's assignment. Results
> above are final for this part and this tag. T0/T1 agents `aider`, `amp`,
> `antigravity`, `minimax-code`, `openhands`, `roo`, `zcode`, `cline`,
> `copilot`, `cursor`, `pi` are fully covered by this part; every other
> agent's rows, section C (CLI experience), and section D (Hop journeys) are
> other executors' part files against this same `63ac5a5b` / `v0.6.0-rc.1`
> tag.

- Terminating tester: tagged-run executor A
- UTC timestamp: 2026-09-06T12:08:20Z
- Part verdict: **FAIL** — 6 of 79 rows in this part do not pass (see
  Verdict for this part). This does not by itself determine the combined
  device verdict, which depends on every part file plus sections C and D.
