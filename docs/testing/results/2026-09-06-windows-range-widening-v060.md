# Verified-range widening — native Windows x64, 2026-09-06

`AGENT-TIER-JOURNEY-V1` · two agents (Claude Code, OpenCode), single platform.

Reproduces the method `v0.5.0-rc.5` used to widen its verified ranges
(`RELEASING.md` "v0.5.0-rc.5 candidate gate",
`docs/testing/v0.5.0-rc.5-agent-verification-prompts.md`, the E rows in
`docs/testing/results/2026-08-21-windows-phase5-V050RC6.md`, and the T5 round
trip in `docs/testing/results/2026-08-23-windows-opencode-t5.md`), under
[ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md) D3.

**macOS evidence: pending (ADR 0005).**

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | 2026-09-06 |
| Device | `windows-amd64`, native — not WSL |
| OS | Microsoft Windows 11 Pro, 10.0.26200 |
| Branch | `v060/w3-ranges` |
| Branch tip at time of this record | `cd3d7f305a4d7cbe448f09fb7a507926051ad65b` |
| Go toolchain | `go1.25.13` (`GOTOOLCHAIN=go1.25.13`; host default `go1.26.1`) |
| Claude Code installed | `2.1.261` |
| OpenCode installed | `1.18.27` |
| Lab root | `<lab-project>` (under `D:\ReinstateAcceptanceProjects\v060-w3\`) |

Both agents were run as the real installed vendor binary inside a throwaway
project directory under the lab root, never the developer's real agent
trees. Claude Code used an isolated `CLAUDE_CONFIG_DIR` seeded with only the
host's own `.credentials.json` (no session/project content copied in);
OpenCode used an isolated `XDG_DATA_HOME`. No real transcript is quoted
anywhere below; every token named is a marker planted by this record for the
sole purpose of proving continuation, redacted as `<token>`.

## 2. T-301 — Claude Code, `2.1.238` → `2.1.261`

| Step | Command (argv) | Result |
| ---- | --------------- | ------ |
| Create session | `claude -p "Reply with exactly this text and nothing else: <token>"` (cwd = `<lab-project>`, a real git repo) | PASS — real session created, vendor returned `<token>` |
| Find it | `rein sessions --agent claude --json` | PASS — the one session in `<lab-project>` listed, nothing else |
| Find it by content | `rein search <token> --json` | PASS — the session found; Claude's transcript reader indexes full message text |
| Inspect before the code change | `rein inspect claude:<id> --json` | PASS — `agent.version.status=untested`, message names range `2.1.219 to 2.1.238 inclusive` |
| Widen the ceiling | edit `maximumVerifiedClaudeVersion` (`internal/adapter/claude/claude.go`), `Max` (`internal/agentcheck/agent.go`, `internal/agents/catalog/claude.go`) to `2.1.261`; rebuild | PASS — `gofmt`/`go vet`/`go build` clean |
| Inspect after the code change | `rein inspect claude:<id> --json` | PASS — `agent.version.status=match/supported`, actual `2.1.261` |
| Dry-run launch plan | `rein resume claude:<id> --dry-run --json` | PASS — plan `claude --resume <id>`, cwd = the session's own workspace; decision `confirmation_required` (only the first-launch baseline warning; no block) |
| Non-`--dry-run` guard | `rein resume claude:<id> --json` (no `--dry-run`) | PASS — refused `--json requires --dry-run for native agent launches`, exit `2`; a real (non-JSON) launch would open the interactive TUI, not attempted here (no ConPTY driver run for this record) |
| Fork plan | `rein resume claude:<id> --fork --dry-run --json` | PASS — plan `claude --resume <id> --fork-session` |
| **Physical resume** | `claude --resume <id> -p "What token did you reply with earlier in this conversation? Reply with only the token, nothing else." --output-format json` — the dry-run plan's exact argv, plus the vendor's own non-interactive completion flag (`-p`), per the card's documented fallback when no ConPTY driver run backs this record | **PASS** — same `session_id` as `<id>`; result was exactly `<token>`, which exists nowhere but the original session's first turn |

## 3. T-302 — OpenCode, single `1.18.21` build → `1.18.21`–`1.18.27`

| Step | Command (argv) | Result |
| ---- | --------------- | ------ |
| Create session | `opencode run --model opencode/big-pickle "Reply with exactly this text and nothing else: <token>" --format json` (stdin redirected from empty; cwd = `<lab-project>`, a real git repo) | PASS — real session created, vendor returned `<token>` |
| Find it | `rein sessions --agent opencode --json` | PASS — the one session in `<lab-project>` listed, nothing else |
| Find it by title | `rein search "Echo token <token>" --agent opencode --json` | PASS — found by the recorded title. A raw-token query returned no rows: OpenCode's index builds `SearchText` from id/title/project/workspace/branch only (`internal/agents/sources/opencode/{source,sqlite}.go`), never message body text — pre-existing, documented behaviour, not a widening regression |
| Inspect before the code change | `rein inspect opencode:<id> --json` | PASS — `agent.version.status=untested`, message names range `1.18.21 to 1.18.21 inclusive` |
| Widen the ceiling | `Max` (`internal/agents/catalog/opencode.go`) `1.18.21` → `1.18.27`; rebuild | PASS |
| Inspect after the code change | `rein inspect opencode:<id> --json` | PASS — `agent.version.status=match/supported`, actual `1.18.27` |
| Dry-run launch plan | `rein resume opencode:<id> --dry-run --json` | PASS — plan `opencode --session <id>`, cwd = the session's own workspace; decision `confirmation_required` |
| Non-`--dry-run` guard | `rein resume opencode:<id> --json` (no `--dry-run`) | PASS — same refusal/exit `2` as Claude Code, above |
| **Physical resume** | `opencode run --session <id> "What token did you reply with earlier in this conversation? Reply with only the token, nothing else." --format json` (stdin redirected from empty) — the dry-run plan's `--session` argument, plus the vendor's `run` non-interactive form | **PASS** — same `sessionID` as `<id>`; result was exactly `<token>` |
| Handoff dry-run (T4) | `rein handoff claude:<claude-id> --to opencode --dry-run --json` (run from the Claude session's own workspace) | PASS — capsule + fidelity report produced, destination argv `opencode --prompt "..."` |

### T5 — push/pull round trip, two isolated homes on this host

Two isolated `REINSTATE_HOME`s (device A holding the source OpenCode store,
device B a separately vendor-initialised, pre-existing OpenCode store) shared
one local `scripts/testing/fakelocker` instance (a real HTTP fake-S3 server,
`go run ./scripts/testing/fakelocker`) as the BYO bucket, standing in for a
shared bucket the same way the original T5 record used a copied disk-backed
store between two real devices. The encryption passphrase reached `rein`
through `REINSTATE_PASSPHRASE_FD` via a throwaway, never-committed FD
launcher built for this record only (the same automation path
`docs/testing/results/2026-08-23-windows-opencode-t5.md` used: "a tiny local
launcher, never committed").

| Step | Command (argv) | Result |
| ---- | --------------- | ------ |
| Init device A | `rein init --yes --endpoint http://127.0.0.1:<port> --bucket w3lab --prefix profiles/w3lab-opencode --region auto` | PASS |
| Push from A | `rein push --agent opencode --session <id> --json` | PASS — snapshot `1c13f05a-7793-4c63-a4de-393cbeb00233` |
| Init device B (same profile) | `rein init --yes --endpoint http://127.0.0.1:<port> --bucket w3lab --prefix profiles/w3lab-opencode --region auto --profile-id <profile-id>` | PASS |
| Pull dry-run on B | `rein pull --agent opencode --session <id> --dry-run --json` | PASS — plan names snapshot `1c13f05a-7793-4c63-a4de-393cbeb00233` |
| Pull for real on B | `rein pull --agent opencode --session <id> --json` | PASS — **same** snapshot id as the dry-run and as the push: a stable revision |
| Vendor reads the restore back | `opencode export <id>` against device B's store | PASS — both messages, both parts, `<token>` intact in the message body, recorded `directory` unchanged (same host, no remap needed for this leg) |
| Live resume-and-answer on device B | `opencode run --session <id> "..." --format json` | **NOT COMPLETED** — see below |

The last row is disclosed rather than hidden. The identical command against
device A's own (unpulled) store answered in seconds; against device B's
pulled store it produced zero stdout for over five minutes across two
attempts. OpenCode's own log (`opencode.log` under device B's `XDG_DATA_HOME`)
shows the run entering its model loop and exiting it normally
(`loop step=1` / `exiting loop`, milliseconds apart), then a 50+ second gap
before an internal `cleanup prune=7.days` step, with no further log line
afterward within the observation window; one of the two attempts also logged
a `cleanup failed exitCode=66` warning at that same point. No Reinstate
process or code path is on the call stack at that point — this is entirely
inside the vendor binary's own post-response housekeeping — so it is recorded
as an open harness observation, not a Reinstate product defect, and is not
claimed as evidence either way. The push/pull round trip and the vendor's own
`export` read-back (both of which did complete, and are the pair the original
T5 record's own evidentiary bar rests on — "the vendor's own view of the
restored session ... read it back with both messages, both parts") are what
this record claims for "restores the session with a stable revision."

## 4. Range declarations moved

| Location | Old | New |
| -------- | --- | --- |
| `internal/adapter/claude/claude.go` (`maximumVerifiedClaudeVersion`) | `2.1.238` | `2.1.261` |
| `internal/agentcheck/agent.go` (`testFallbackDefinitions` claude `Max`) | `2.1.238` | `2.1.261` |
| `internal/agents/catalog/claude.go` (`VersionSpec.Max`) | `2.1.238` | `2.1.261` |
| `internal/agents/catalog/opencode.go` (`VersionSpec.Max`) | `1.18.21` | `1.18.27` |

Every test inside W3's `file-ownership.md` grant that pinned the old
ceilings, or the "one version past the old ceiling" fail-closed boundary,
moved with them so a fail-closed probe still refuses the next version up as
`UNTESTED`:

- `internal/adapter/claude/claude_test.go` (`internal/adapter/claude/**`,
  W3) — `2.1.238` stays `true`; the boundary case moves from `2.1.239`/`false`
  to `2.1.261`/`true` (new ceiling) and `2.1.262`/`false` (one past it). This
  is the test that now carries the fail-closed proof for Claude Code below.
- `internal/agentcheck/version_evidence_test.go`
  (`internal/agentcheck/**`, W3) — the refusal message's named range moves
  from `2.1.219`/`2.1.238` to `2.1.219`/`2.1.261`.

Four more tests pin the same old ceilings but sit outside W3's grant in
`file-ownership.md`: the row for `internal/agents/catalog/` names the files
`claude.go` and `opencode.go` specifically, not the directory, and
`internal/cli/**` (everything else) / `internal/transcript/**` are
Coordinator-owned or unlisted entirely. The task card's own step 5 said to
update every test that pins the old ceiling; `file-ownership.md`'s Gate 0 has
no size exception for that. Gate 0 wins: this record's branch does **not**
edit these four files, so each is now stale against the widened ranges above
and fails exactly as shown, with the one-line fix each needs to a future
owner:

| File | Failing test | Current (stale) assertion | One-line fix |
| ---- | ------------- | -------------------------- | ------------- |
| `internal/agents/catalog/catalog_test.go` | `TestShippedAgentsRegisterAtDeclaredTiers` | `claude` row still expects max `2.1.238`; got `2.1.261` | change the claude row's max to `2.1.261` |
| `internal/agents/catalog/opencode_resume_test.go` | `TestOpenCodeDescriptorIsT5` | still expects the single build `1.18.21`/`1.18.21`; got `1.18.21`/`1.18.27` | change the expected max to `1.18.27` |
| `internal/cli/handoff_claude_source_test.go` | `TestHandoffFromClaudeInstallJustOutsideVerifiedRange` | fake `claude --version` `2.1.239` is now **inside** the widened range, so the handoff succeeds (exit `0`) instead of refusing (`ExitCompatibility`, exit `5`) | move the fake version from `2.1.239` to `2.1.262` (one past the new ceiling) |
| `internal/transcript/claude_test.go` | `TestClaudeProbeVersionGate` | the `"outside range"` case fixes `2.1.239`, which now probes `SUPPORTED` instead of `UNTESTED` | move that case's fixed version from `2.1.239` to `2.1.262` |

None of these are behaviour changes to fix — each is the identical one-line
literal bump already proven correct in the prior draft of this branch, which
this record's author (W3) is not permitted to carry under
`file-ownership.md`. Until whoever is assigned those paths applies them, `go
test ./...` shows exactly these 4 failures in exactly these 3 packages and
no others (§5).

Confirmed fail-closed regardless: `2.1.262` (Claude Code) is still `UNTESTED`
against the widened range, proved at the unit level by the in-scope
`TestClaudeSupportedVersionRange` (`internal/adapter/claude/claude_test.go`,
line 128: `{version: "2.1.262", want: false}`), independent of the four stale
tests above. OpenCode's `1.18.28` boundary has no literal unit test on either
side of this change (the task card's "or the unit tests" allowance) and was
not re-verified against the real vendor binary since neither `2.1.262` nor
`1.18.28` is installed on this host.

## 5. Gates

| Gate | Command | Result |
| ---- | ------- | ------ |
| Format | `gofmt -l .` | PASS — empty |
| Vet | `GOTOOLCHAIN=go1.25.13 go vet ./...` | PASS |
| Tidy | `GOTOOLCHAIN=go1.25.13 go mod tidy -diff` | PASS — empty |
| Unit suite | `CGO_ENABLED=0 GOTOOLCHAIN=go1.25.13 go test ./... -count=1` | **FAIL as recorded** — 3 packages / 4 tests red: `internal/agents/catalog` (`TestShippedAgentsRegisterAtDeclaredTiers`, `TestOpenCodeDescriptorIsT5`), `internal/cli` (`TestHandoffFromClaudeInstallJustOutsideVerifiedRange`), `internal/transcript` (`TestClaudeProbeVersionGate`) — all four are the stale-pinned-literal tests documented in §4, outside W3's `file-ownership.md` grant; every other package (67 of 70) `ok` |
| Race suite | `CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test -race ./internal/... -count=1` | **FAIL as recorded** — the same 3 packages / 4 tests, nothing else; re-run with `TMP`/`TEMP`/`GOTMPDIR` redirected off the host's nearly-full system drive (see the executor report) |
| Cross-OS build | `GOOS=darwin go build ./...`, `GOOS=linux go build ./...` | PASS (Linux build likewise needed `GOTMPDIR` off the system drive) |
| Doc gate | `go test ./internal/doctest/... -count=1`; `scripts/check-docs.sh` | PASS |
| Secret scanner | `go test ./internal/fixture -count=1` | PASS |
| Diff secret grep | `git diff release/v0.6.0-rc.1...HEAD \| grep -nE 'AKIA\|-----BEGIN\|passphrase=\|Bearer [A-Za-z0-9]'` | PASS — no match against the actual branch range (the prior draft of this record used `git diff -- .`, a working-tree diff that is empty by construction on a clean, committed branch and never scanned the introduced diff at all; corrected here) |

### Addendum — scope correction after review

The resume/push/pull evidence in §§2–3 was gathered at branch tip
`cd3d7f305a4d7cbe448f09fb7a507926051ad65b` and is unchanged by anything
below; it does not depend on the four out-of-scope files. A later commit on
this branch reverted `internal/cli/handoff_claude_source_test.go`,
`internal/transcript/claude_test.go`,
`internal/agents/catalog/catalog_test.go`, and
`internal/agents/catalog/opencode_resume_test.go` to their pre-branch
content because they sit outside W3's row in `file-ownership.md` (see §4),
which is why the Unit suite and Race suite rows above show the resulting
4 failures rather than an all-`ok` run. Nothing else in this document changed
as a result.

**macOS evidence: pending (ADR 0005).**
