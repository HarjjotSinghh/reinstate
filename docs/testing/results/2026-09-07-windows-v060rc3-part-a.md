# `v0.6.0-rc.3` native Windows acceptance — part A (gates, core, T0/T1)

Executor A of the tagged-artifact run. Covers section A automated gates,
Phase 5 core matrices A/B/G/H, T0 Matrix F (all 7 agents), and T1 Matrix C
(cline, copilot, cursor, pi). Other executors cover T2–T5 agent rows,
sections C/D, and the reconciliation.

## Header

| Field | Value |
| ----- | ----- |
| UTC date/time | `2026-09-07T02:36:00Z`–`2026-09-07T03:10:00Z` (session) / report filed `2026-09-07` |
| Device | `windows-amd64`, native x64, not WSL |
| OS/version/build | Windows 11 Pro 10.0.26200 |
| Host | acceptance host (hostname redacted) |
| Tested tag | `v0.6.0-rc.3` |
| Tested full commit | `202157c7877d33105dec700604dd23893b4d8b51` |
| Windows archive SHA-256 (checksums.txt, re-verified) | `5fc5188ea92706e841d9c022cfe29ab386430a1a54a90139eec16d9baf756cd0` |
| Installed binary SHA-256 (`rein.exe` == `reinstate.exe`) | `6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0` |
| `rein version --json` (own install, `D:\ReinstateAcceptanceProjects\v060-rc3-a\install`) | `{"name":"reinstate","version":"0.6.0-rc.3","commit":"202157c7877d33105dec700604dd23893b4d8b51","date":"2026-09-07T02:36:11Z"}` |
| Go toolchain | go1.26.1 (module pins `GOTOOLCHAIN=go1.25.13` for lint/vuln/test-install, honored) |
| Worktree | `D:\Projects\reinstate-worktrees\v060-rc3-tagged`, branch `v060/rc3-tagged` @ `202157c7877d33105dec700604dd23893b4d8b51` |
| Host contamination rule | Every shell in this report ran `unset REINSTATE_BACKEND REINSTATE_MEMORY_BACKEND_DIR` (bash) / `Remove-Item Env:REINSTATE_BACKEND, Env:REINSTATE_MEMORY_BACKEND_DIR` (PowerShell) before any `rein`/`go test`/Hop-lab invocation. `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `XDG_DATA_HOME` were left untouched throughout. |

**Bootstrap deviation sentence.** The live `https://reinstate.dev/install.ps1`
was downloaded fresh with `Invoke-WebRequest -OutFile` into
`D:\ReinstateAcceptanceProjects\v060-rc3-a\bootstrap\install.ps1`; its pinned
version line reads `$Version = "v0.6.0-rc.3"`. Run once (`INSTALL_DIR` and
`REINSTATE_SKIP_PATH_UPDATE=1` set so it never touched the persistent user
`PATH` or replaced the host's pre-existing install) into
`D:\ReinstateAcceptanceProjects\v060-rc3-a\bootstrap\bin`, it installed
`reinstate.exe`/`rein.exe` whose SHA-256
(`6517281bc5a5984e59f030238e1525d355201bb849762db00e403a990fcde7d0`) is
byte-identical to the checksummed `rc3-draft` archive's binaries and reports
the same `version`/`commit`/`date` — **no deviation** between the live
bootstrap route and the release-commit candidate. (Note: this host also
carries a pre-existing `rein`/`reinstate` install at
`%LOCALAPPDATA%\Programs\Reinstate\bin`, version `0.3.0-rc.6`, on the
persistent user `PATH` — the bootstrap's own canonical installer correctly
refused to silently replace it without `REINSTATE_CONFIRM_REPLACE=1`, which
this run never set, confirming the "never replace a user binary" rule holds
even for the bootstrap path. Every command in this report prepended the
candidate's own install directory to `PATH` and confirmed `which rein
reinstate` resolved there before use.)

All work happened under `D:\ReinstateAcceptanceProjects\v060-rc3-a\` and the
worktree. Isolated directories holding copies of real vendor session data
(used for Matrix C6 corruption tests, covering `cline`, `copilot`, `cursor`,
`pi`) were deleted immediately after use; nothing from them is committed or
quoted here. Two throwaway git projects (`pi-throwaway`, `cline-throwaway`)
hold only sessions this run itself created with a planted, non-sensitive
search token; they are left in place as evidence, not real vendor data.

---

## Section A — Automated gates

| # | Gate | Result | Evidence |
| - | ---- | ------ | -------- |
| A1 | Format, vet, tidy | PASS | `gofmt -l .`, `go vet ./...`, `go mod tidy -diff` all empty output, all exit 0 |
| A2 | Unit suite | PASS | `CGO_ENABLED=0 go test ./... -count=1 -p 4`: all packages `ok`, 0 `FAIL`, exit 0 |
| A3 | Race suite | PASS | `CGO_ENABLED=1 go test -race ./internal/... -count=1 -p 4`: all packages `ok`, 0 `FAIL`, exit 0 |
| A4 | Lint and vuln | PASS | `make lint` → golangci-lint v2.11.4, `0 issues`; `make vuln` → govulncheck v1.6.0, `0 vulnerabilities` in code (4 in required-but-uncalled modules, informational) |
| A5 | Doc gate | PASS | `go test ./internal/doctest/... -count=1` → `ok`; `scripts/check-docs.ps1` → exit 0 (wraps the same test) |
| A6 | Snapshot and artifacts | PASS | `scripts/verify-release.ps1 -DistDir <rc3-draft>` → `release artifacts verified (PowerShell): <rc3-draft>`, exit 0; `scripts/check-release-binary-identity.ps1 -DistDir <rc3-draft> -ExpectedCommit 202157c7877d33105dec700604dd23893b4d8b51 -ExpectedVersion 0.6.0-rc.3` → `release binary identity ok: version=0.6.0-rc.3 commit=202157c7877d33105dec700604dd23893b4d8b51`, exit 0 |
| A7 | Installers | PASS | `scripts/test-install.ps1 -DistDir <rc3-draft>` → verify-release re-run clean, `internal/doctest` `ok`, exit 0 |
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
| A5 | Every `Evidence` path exists in the tagged tree | PASS | `checkEvidence` `os.Stat`s every declared path per descriptor; exercised by every `TestXConformance`, all pass |
| A6 | `sessions --agent <key>` accepts exactly T1+ keys, rejects others exit 2 | PASS | Live: all 7 T0 keys and one bogus key → exit 2; all 11 T1–T5 keys (`cline,copilot,cursor,pi,gemini,kimi,grok,qwen,claude,codex,opencode`) → exit 0 |
| A7 | `resume` accepts exactly T3+ keys, refuses others exit 5 + reason | PASS | Live: `AGENT:SESSION_ID` syntax against a bogus id — T0/T1/T2 keys (`aider`,`amp`,`cline`,`copilot`,`cursor`,`pi`,`gemini`,`kimi`) → exit 5, `code:"compatibility"`, message names the tier and reason; T3+ keys (`grok`,`qwen`,`claude`,`codex`,`opencode`) → exit 2, `code:"usage"`, `"session not found: <agent>:bogus-session-id"` (key accepted, no such session) |
| A8 | Agent order deterministic across runs | PASS | Two successive `doctor --agents --json` runs a few seconds apart: identical 18-key alphabetical order both times. (Harness note: the two captures otherwise differed only in `generated_at` and one live directory's child/file count — real host agent activity between the two calls, not nondeterminism in the ordering itself.) |
| A9 | Broken-evidence descriptor fails conformance | PASS | `TestBrokenEvidencePathFails` and `TestRunFailsBrokenEvidence` (`internal/agents/conformance/run_test.go`) pass in A2 and re-run explicitly here |
| A10 | No catalog scan writes/renames/locks any agent-root file | PASS | Live filesystem audit: full recursive `mtime`+`size` snapshot of the real `%USERPROFILE%\.pi\agent` tree (46 files) before and after `rein sessions --agent pi`, `rein search`, and `rein doctor --agents --json` — zero diff |

**10/10 PASS.**

---

## Matrix B — Probe and redaction (9 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| B1 | `doctor --agents --json` conforms to `AGENT-PROBE-V1` | PASS | `"schema": "AGENT-PROBE-V1"` in every capture |
| B2 | No absolute path/home dir name/username | PASS | Full-document scan for the local username, `c:\users`/`c:/users`, and `d:\`/`d:/`-style drive paths: all absent |
| B3 | Only keys, never a JSON value, from session files | PASS | `first_line_keys` entries (e.g. Claude's `history.jsonl`→`["display","pastedContents","project","sessionId","timestamp"]`) carry field names, never field content |
| B4 | Names/dirs shape-normalized, no raw UUID/hash/slug | PASS | Verified specifically against a **real OpenCode Git object store**: `resolved_root` shows a live root; `name_shapes` includes `snapshot/*` → `<40-hex>`, `snapshot/*/*` → `<40-hex>`, `snapshot/*/*/objects/pack/*` → `pack-<40-hex>.idx` (3 real pack files); the raw `tree` entries under the same paths are wildcarded (`snapshot/*/*/...`), never a literal 40-hex object or pack name |
| B5 | No excluded subtree (credential/cache) anywhere in output | PASS | Full-document scan for `credential`, `.git-credentials`, `token.json`, `secret`, `auth.json`, `apikey`, `password`, `keyring`, `.ssh`: all absent |
| B6 | Absent agent reports absence without error | PASS | T0 agents (e.g. `aider`) show `resolved_root: null`, `tree: []`, `candidate_roots: []` with no exception; overall command exit 0 |
| B7 | Empty root distinguishable from absent root | PASS | Real observed data: Kimi's declared candidates show `.kimi-code` (`exists:true, marker_present:true`) alongside `.kimi` (`exists:true, marker_present:false`) — an existing-but-unmarked root reported distinctly from one that plain does not exist (`exists:false`) |
| B8 | `agent-storage-probe.sh`/`.ps1` identical to the binary | PASS | Both scripts are thin wrappers (`rein doctor --agents --json "$@"`); back-to-back diff against direct `rein.exe doctor --agents --json` invocation was byte-identical after normalizing `generated_at`, for both the `.ps1` and the `.sh` variant, with the candidate's own directory prepended to `PATH`. (Harness note: a first `.sh` attempt run a few seconds apart from its comparison capture showed one unrelated diff — a Codex `tmp/arg0/*` shape that needs 2+ samples to normalize, and had only 1 at the first capture — live host noise between two nearly-simultaneous calls, not a wrapper defect; the immediate back-to-back re-run matched exactly.) |
| B9 | Fixture secret scanner passes over every committed probe artifact | PASS | No standalone secret-scan script/binary exists in the tree to invoke directly; substituted a pattern scan (API-key/token/private-key/password shapes) over all 26 files under `docs/testing/results/agent-probes/`, zero hits, plus the passing `internal/secretscan` unit suite (part of A2, also re-run explicitly here) |

**9/9 PASS.**

---

## Matrix G — No regression (8 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| G1 | Claude/Codex native resume+fork behave as prior releases | PASS (regression evidence only) | Full `go test ./...` (A2) covers `internal/agentcheck`, `internal/agents/catalog` claude/codex descriptors unchanged; no live resume/fork journey performed by this executor — that physical journey is Matrix E (`claude:E1/E2`, `codex:E1/E2`), owned elsewhere in this run |
| G2 | Claude↔Codex structured handoff both complete | PASS (regression evidence only) | `internal/handoff/...` suite passes in A2; no fresh live handoff journey run by this executor (Matrix D scope, owned elsewhere) |
| G3 | Gemini/OpenCode/Grok remain handoff sources, read-only | PASS | Live: `gemini`(T2) and `kimi`(T2) refuse `resume` exit 5 naming the tier (§A7); OpenCode is T5 (its own native-agent tier, unchanged from rc.2) |
| G4 | `push`/`pull` carry only Claude/Codex(+OpenCode) | PASS | Live `rein __complete push --agent ""` and `pull --agent ""`: both list exactly `claude`, `codex`, `opencode`; no other agent key offered |
| G5 | Path remapping unchanged | PASS | `internal/pathmap/...` suite passes in A2, re-run explicitly here |
| G6 | Exit-code semantics unchanged | PASS | Live exit codes observed throughout this report are consistent with documented meanings: `2` (not found/invalid key), `5` (unsupported/blocked, names reason), `0` (success) — no new/undocumented code observed |
| G7 | An older-lineage index upgrades without loss (tested from a real `v0.5.1`-built index) | PASS | Built a 100-session index with the real `reinstate_0.5.1_windows_amd64` binary (checksum-verified against `checksums.txt`) in an isolated `$REINSTATE_HOME`: `rein sessions --agent all --json` → 100 sessions across 10 agents. Ran the `v0.6.0-rc.3` binary against that same home: `rein version --json` reports `0.6.0-rc.3`; `rein sessions --agent all --json` returns the identical 100-session, identical per-agent distribution, exit 0, no error/migration prompt — the v0.5.1-built `cache/session-index-v2.sqlite` opened and served cleanly under the new binary |
| G8 | Capsules excluded from sync | PASS | `TestRefuseHandoffsPush` (`internal/sync/sync_test.go`) passes in A2 and re-run explicitly here |

**8/8 PASS** (G1/G2 carry the caveat that this executor's own evidence is the regression suite, not a fresh physical journey — that mechanism is exercised live under Matrix D/E by whichever executor owns those rows this run).

---

## Matrix H — CLI contract and performance (6 rows)

| # | Assertion | Result | Evidence |
| - | --------- | ------ | -------- |
| H1 | `docs/cli-reference.md` matches shipped flags | PASS | `go test ./internal/doctest/... -count=1` → `ok` (part of A5) |
| H2 | `--json` shape for sessions/search/inspect/doctor matches documented shape | PASS | All four commands returned well-formed, schema-conformant JSON throughout this run (`AGENT-PROBE-V1` for doctor; consistent `sessions[]`/`session{}` shapes for the others), consumed programmatically without error at every step above |
| H3 | Full refresh across all installed agents within documented ceiling | PASS | Live cold refresh on a fresh `$REINSTATE_HOME`, 10 installed agents, real sessions: `11.046s` (rc.2's tagged run recorded `13.149s` under the same "reasonable ceiling" standard — consistent, no regression) |
| H4 | No-change refresh materially faster than cold | PASS | Same run, immediate re-invocation: `0.200s` (~98% faster than the `11.046s` cold run) |
| H5 | One unreachable/slow agent source does not block the others | PASS | `TestSlowAgentDoesNotDiscardTheRun` (`internal/agents/probe/collect_test.go`) passes in A2 and re-run explicitly here |
| H6 | Shell completion lists correct agent keys per command (exercised directly) | PASS | Live `rein __complete`: **`push --agent ""`** and **`pull --agent ""`** → `claude, codex, opencode` (exercised directly per this run's dispatch); `sessions --agent ""` → `all, claude, cline, codex, copilot, cursor, gemini, grok, kimi, opencode, pi, qwen` (T1+ only, no T0 key); `handoff --to ""` → `claude, codex, grok, opencode, qwen`. All surfaces agree with each other and with A6/G4's key sets |

**6/6 PASS.**

---

## T0 Matrix F — Per-T0-agent (7 agents × 2 rows = 14 rows)

All evidence is live, from the same `rein doctor --agents --json`/`--complete` captures used above, plus an explicit `sessions --agent`/`resume` exit-code sweep against every T0 key.

| Agent | F1 (appears in `doctor --agents` with reason) | F2 (no command offers a capability; nothing implies one is coming) |
| ----- | --- | --- |
| `aider` | PASS — `t0_reason=layout_unverified` | PASS — `sessions --agent aider` exits 2; `resume aider:bogus` exits 5 naming the tier; no completion surface lists `aider` |
| `amp` | PASS — `t0_reason=server_backed` | PASS — same sweep, clean |
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
a **copy** of each real tree (deleted immediately after use; the A10 audit
already proved `rein` itself never touches the source). `cline:C3` and
`pi:C3` created fresh throwaway sessions through the vendor's own live CLI in
an isolated project, with a planted, non-sensitive search token chosen by
this run. `cursor:C2`/`C3` and `copilot:C3` ran against the host's real store
under the T1-discovery exception (real data; counts, field names, and a
self-chosen search word only — never a quoted real prompt, title, path, or
session id).

| # | Assertion | cline | copilot | cursor | pi |
| - | --------- | ----- | ------- | ------ | -- |
| C1 | ≥2 real sessions, ≥2 distinct projects | PASS — 5 sessions / 4 distinct projects | PASS — 2 sessions / 2 distinct projects | PASS — 2 sessions / 2 distinct projects | PASS — 4 sessions / 3 distinct projects (before this run's own 5th session) |
| C2 | project/branch/title/timestamp/message-count match the agent's own store | PASS — real non-zero `message_count` (2,1,1,1,1) | PASS — real non-zero `message_count` (5, 26) | **PASS — fix confirmed against the real host store**: both real sessions now report non-zero `message_count` (3, 3), reading the store's actual `blobs`/`meta` schema. (`v0.6.0-rc.2`'s tagged run recorded `message_count: 0` for these same real sessions against the old `messages`/`message`/`bubbles` guess — this is the exact regression the candidate's dispatch targets, and it is fixed.) | PASS — real non-zero `message_count` (2,2,2,2) |
| C3 (message-body search) | **PASS** — `cline --cwd <isolated throwaway dir>` answered live (not `Unauthorized`) confirming Cline is authenticated on this host; created a fresh session with a planted token via `cline --cwd <dir> --json "<prompt with token>"`; `rein search <token> --agent cline --json` → exactly 1 match | **PASS** — real host session: searched `rein search "Reply" --agent copilot --json` (a word taken from that session's own bounded `prompt_preview`) and got exactly 1 match, confirming body-text search on real data | **PASS — fix confirmed against the real host store**: read one real session's own bounded `prompt_preview` (a boilerplate `<user_info>...Workspace Path:...` header), chose one distinctive word from it, ran `rein search <word> --agent cursor --json` → exactly 1 match, the correct session. (`v0.6.0-rc.2`'s tagged run recorded this as unable to succeed against a schema no real store had — fixed.) | **FAIL — new finding, not a host-credential gap.** `pi -p "<prompt with planted token>"` in a throwaway project completed live through Pi's own configured Anthropic login (`defaultProvider: anthropic`, model `claude-opus-4-7`) and replied normally, creating a real session file on disk containing the token in a `role":"user"` message. `rein search <token> --agent pi --json` returned zero matches, and `rein inspect` on that same session shows **no `prompt_preview` field at all** (all 5 real sessions on this host show this). Root-caused by reading `internal/agents/sources/pi/source.go`: this host's real Pi CLI (`0.73.1`) writes session-format `version:3`, where user/assistant turns are `{"type":"message","message":{"role":"...","content":[{"type":"text","text":"..."}]}}` — but `readConversation`'s text extraction calls `ExtractTextContent(item["message"])`, and `ExtractTextContent`'s `map[string]any` case only reads a top-level `"text"` key on that map, never descending into `message.content`. The committed test fixture (`testdata/sessionindex/pi/*/`) uses an older flat schema (`{"type":"user",...,"text":"..."}`, `version:1`) that the same code path *does* handle, which is why the unit suite (A2) passes while this real-data mechanism does not. `message_count` still comes out correct (2) because both turns share the outer `"type":"message"` and are counted regardless of extracted text, which is why this was not visible in C2. This is a Cursor-C2/C3-shaped defect (real vendor schema drifted from what the reader recognizes, silently degrading to empty rather than erroring) that this candidate does not touch. |
| C4 | bounded metadata only, no transcript body | PASS — `inspect` returns a fixed key set incl. a 77-char `prompt_preview`, no full body | PASS — same bounded shape, 30-char `prompt_preview` | PASS — same bounded shape, 160-char `prompt_preview` | PASS — bounded shape with no crash; `prompt_preview` field absent rather than any transcript body (consistent with the C3 finding above — bounded is satisfied, search is not) |
| C5 | below T3, every record read-only; resume refused exit 5 | PASS — live against a real session id: exit 5 | PASS — live against a real session id: exit 5 | PASS — live against a real session id: exit 5 | PASS — live against a real session id: exit 5 |
| C6 | corrupted/empty/absent root degrade cleanly, no panic, no partial record | PASS — truncated a copy of a real `.messages.json` sidecar: unaffected sessions still list correctly, exit 0; empty/absent root (`CLINE_DATA_DIR`) → `[]`, exit 0 | PASS — truncated a copy of a real `session.db`: no crash, sessions still list correctly, exit 0; empty/absent root (`COPILOT_HOME`) → `[]`, exit 0 | PASS — truncated a copy of a real `store.db` (via the `CURSOR_CONFIG_DIR` redirect): the truncated session degrades to `message_count: 0` (fail-closed, matches the catalog's documented behavior for an unreadable store), the unaffected session still reports `message_count: 3`, no panic, exit 0; empty/absent root → `[]`, exit 0 | PASS — truncated a copy of a real session `.jsonl` mid-record: no crash, all 5 sessions still list, exit 0; empty/absent root (`PI_CODING_AGENT_DIR`) → `[]`, exit 0 |

**Row count: 24. 23 PASS, 1 FAIL** (`pi:C3` — a confirmed reader defect, not a
host/credential/harness gap; see the Findings section below).

---

## Findings

### Product defect (release-blocking; new this run)

- **`pi:C3` fails on real data: Pi's real `version:3` session format
  (`"type":"message"` with nested `message.role`/`message.content`) is not
  parsed for searchable text.** `internal/agents/sources/pi/source.go`'s
  `readConversation` calls `sources.ExtractTextContent(item["message"])`
  when `item["type"] == "message"`. `ExtractTextContent`'s `map[string]any`
  branch (`internal/agents/sources/extract.go`) only reads a **top-level**
  `"text"` key on that map; it does not descend into `message.content`
  (itself a content-block array shaped `[{"type":"text","text":"..."}]`,
  the same shape `ExtractTextContent`'s own `[]any` branch already knows how
  to flatten). The two fallbacks (`item["content"]`, then a top-level
  `item["text"]`) also miss, because in this format both the text and the
  role live one level inside `item["message"]`, not at those paths. Net
  effect: `prompt_preview` and search-text are silently empty for every real
  session in this format — confirmed on a session this run created itself,
  with a known planted token verifiably present in the raw `.jsonl` on disk,
  live-completed through Pi's own configured Anthropic account. `message_count`
  is unaffected (both turns share the outer `"type":"message"` case, so the
  count is right by construction, independent of the text-extraction bug),
  which is why Matrix C2 does not surface it and only C3 does. The committed
  fixture at `testdata/sessionindex/pi/{macos,windows}/` uses an older flat
  schema (`{"type":"user","text":"...","version":1}`) that the same code
  correctly handles, so `go test ./internal/agents/sources/pi/...` (part of
  A2) stays green while the real-vendor-format path is broken — the same
  general shape of gap (a fixture frozen to a schema the vendor has since
  moved on from) as the `cursor:C2`/`C3` defect this candidate fixes, just
  in a different reader and not touched by this candidate. All 5 real Pi
  sessions on this host (this run's included) show an absent `prompt_preview`
  for the same reason. Recommend a fix in `readConversation`/
  `ExtractTextContent` (or a Pi-specific unwrap of `item["message"].content`)
  plus a new fixture using the real `version:3` shape, mirroring how this
  candidate's Cursor fix added `blobs`/`meta`-schema coverage.

### Harness notes (not product defects)

- Two apparent nondeterminism findings during this run turned out to be live
  host activity between two nearly-simultaneous `doctor --agents --json`
  calls, not defects: (1) the A8 determinism check's two captures differed
  only in `generated_at` and one directory's live child/file count (a
  real, in-progress codex/claude write on this actively-used host); (2) a
  first B8 wrapper-vs-binary diff showed one Codex `tmp/arg0/*` name-shape
  entry differ (`<slug>` vs. a literal name — the shape algorithm needs 2+
  samples to generalize and the second sample landed between the two calls).
  An immediate back-to-back re-run of each matched exactly. Recorded here per
  the dispatch's own "check the harness before the product" rule; agent key
  **ordering** itself was identical in both A8 captures, so A8 is still PASS.
- This host carries a pre-existing `rein`/`reinstate` install (`0.3.0-rc.6`)
  on the persistent user `PATH`. Every command in this report prepended the
  candidate's own directory to `PATH` and confirmed `which rein reinstate`
  resolved to the candidate before use; the live bootstrap's own
  "refusing to replace" guard was independently observed firing against that
  same pre-existing install when `INSTALL_DIR` was not set, confirming the
  guard itself works.
- `cline` and `cursor-agent` availability differs from the `v0.6.0-rc.2`
  tagged run: Cline now answers live (`cline --cwd <dir> --json` returns a
  real completion, not `Unauthorized`), which is why `cline:C3` is `PASS`
  this run rather than the `PARTIAL` rc.2 recorded. `cursor-agent`
  (`2026.08.11`) is still broken on this host — `Error: Cannot find module
  'tree-sitter'`, confirmed by direct invocation before this run started —
  so `cursor:C2`/`C3` correctly ran `rein`-only, read-only, against the real
  store per the dispatch's prescribed method rather than via a fresh
  vendor-launched session.

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
| Matrix C (T1 × 4) | 24 | 23 | 0 | 1 | 0 |
| **Total** | **79** | **78** | **0** | **1** | **0** |

One required row in this executor's assignment does not pass: `pi:C3`,
`FAIL`, a confirmed reader defect (see Findings). This is a **new** finding
relative to the `v0.6.0-rc.3` dispatch's expectation of `PASS` for this row,
and relative to `v0.6.0-rc.2`'s carried disposition (`PARTIAL`,
host/harness) — this run root-caused the gap as a code defect rather than a
credential or session-availability problem, so it does not qualify for
`NOT TESTED (host credential)` and blocks the required-row count as `FAIL`
pending the assembler's own reconciliation of every executor's rows. Two of
the three rc.2-carried Matrix-C findings this dispatch specifically expected
fixed are confirmed fixed against **real host data**: `cursor:C2`/`C3` (the
candidate's own intended fix) and `cline:C3` (the host's Cline credential is
now authenticated). `copilot:C3` remains `PASS` as at rc.2.
