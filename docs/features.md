# Features and commands (v0.1.0–v0.6.0)

Stable `v0.6.0` is the current release. It includes every shipped surface from
Phase 1 through Phase 6C. This page is the command map. Details live in
[CLI reference](cli-reference.md), [getting started](getting-started.md), and
[handoff](handoff.md).

`rein` and `reinstate` are the same binary.

A structured handoff starts a **new destination session continuing the same
task**. Native resume stays same-vendor; Reinstate does not reconstruct a
vendor session or translate one agent's transcript into another.

## What each stable release added

| Release | Phase | What you can do |
| ------- | ----- | ---------------- |
| `v0.1.0` | Encrypted sync | Push and pull same-vendor Claude Code and Codex sessions through client-side-encrypted S3-compatible storage you own. |
| `v0.2.0` | Local index | Find, search, inspect, and same-vendor resume/fork sessions on one machine with no `init` or bucket. |
| `v0.3.0` | Verified resume | Gate native launch on a local environment report (workspace, agent, capabilities, runtime) with exact warning acknowledgement. |
| `v0.4.0` | Structured handoff | Continue the same task in a *new* Claude Code or Codex session, including Claude ↔ Codex, with a visible projection and five-bullet first-reply. |
| `v0.5.0`/`v0.5.1` | Universal agent coverage | An 18-agent catalog with an explicit support tier per agent, `rein doctor --agents` with a redacted storage probe, session discovery for eleven agents, and structured handoff from five. `v0.5.1` is a patch on `v0.5.0` (pure-Go SQLite driver bump; no product-surface change). |
| `v0.6.0` | Cloud continuity (Hop) | The Reinstate Hop hosted-tier client (`login`/`whoami`, `init --hop`, `account init/recover/join`, `devices approve/revoke`, `sync verify/migrate`, `daemon`) and the interactive TUI switcher. OpenCode reaches T5 (encrypted same-vendor sync). Kimi Code CLI reaches T2 (handoff source). |

Stable `v0.6.0` is certified by native Windows x64 tagged-artifact acceptance
PASS (215/215 required rows) under the single-platform waiver in
[ADR 0005](adr/0005-v0.6.0-scope-and-windows-first-acceptance.md); Apple
Silicon macOS acceptance is deferred to
[#403](https://github.com/HarjjotSinghh/reinstate/issues/403) until that
hardware returns. Earlier stable releases (`v0.5.1` and before) passed
dual-platform tagged-artifact acceptance on both Apple Silicon macOS and
native Windows x64. Intel macOS and Linux/WSL2 remain optional and unverified
([#97](https://github.com/HarjjotSinghh/reinstate/issues/97),
[#98](https://github.com/HarjjotSinghh/reinstate/issues/98)).

## Command map

### Always available

| Command | Since | Purpose |
| ------- | ----- | ------- |
| `rein version [--json]` | v0.1.0 | Print the installed version. |
| `rein doctor [--json] [--self-test]` | v0.1.0 | Health-check the CLI. `--self-test` uses in-memory sync; it does not prove remote storage. |
| `rein setup check [--json]` | v0.1.0 | Report missing config or platform/agent issues. |
| `rein completion …` | v0.1.0 | Shell completion. |

### Phase 1 — encrypted sync (`v0.1.0`)

Requires `rein init` and a passphrase on each device.

| Command | Purpose |
| ------- | ------- |
| `rein init` | Configure profile, bucket, and project path roots. |
| `rein list` | Compatibility listing used by sync scripts. Prefer `rein sessions` for local work. |
| `rein status` | Compare local sessions with the remote manifest. |
| `rein diff` | Show divergence before a transfer. |
| `rein push` | Encrypt and upload selected Claude/Codex sessions. |
| `rein pull` | Download, decrypt, and restore with backups. |
| `rein conflicts list\|show\|resolve` | Inspect and resolve sync conflicts. |

Native resume after a pull stays same-vendor: Claude → Claude, Codex → Codex.

### Phase 2 — local continuity (`v0.2.0`)

No `init`, credentials, or network.

| Command | Purpose |
| ------- | ------- |
| `rein sessions` | Refresh the private derived index and list sessions. |
| `rein search QUERY…` | Literal, case-insensitive search of bounded user text and metadata. |
| `rein inspect AGENT:SESSION_ID` | Bounded inspect, including the Phase 3 `environment` report. |
| `rein last` | Plan a launch of the newest matching session. |
| `rein resume AGENT:SESSION_ID` | Same-vendor native resume (`claude --resume` / `codex resume`). |
| `rein fork AGENT:SESSION_ID` | Same-vendor native fork. |
| `rein` (TTY) | Numbered switcher: resume, inspect, fork, or handoff. |

Gemini CLI and OpenCode are read-only in the local index. They refuse
`resume`/`fork`.

### Phase 3 — verified resume (`v0.3.0`)

Same commands as Phase 2, plus:

| Flag / report | Purpose |
| ------------- | ------- |
| `environment` on inspect and native dry-runs | Local checks for workspace, installed agent, name-only capabilities, and declared runtimes. |
| `--allow-environment-warning ID` | Acknowledge one exact warning ID. Repeat per ID. |
| `--dry-run --json` | Required whenever JSON output would mix with a live vendor TUI. |

A first launch reports `baseline.unavailable` on purpose. Unacknowledged
warnings fail closed with exit `7`.

### Phase 4 — structured handoff (`v0.4.0`)

| Command | Purpose |
| ------- | ------- |
| `rein handoff SESSION --to claude\|codex` | Build a continuity capsule and start a **new** dest session. |
| `rein handoff --last --from AGENT --to AGENT` | Handoff the newest matching source. |
| `rein handoff --dry-run` | Preview; temporary files only. |
| `rein handoff --no-launch` | Store the capsule and print the dest command. |
| `rein handoff list` | List local handoff artifacts (`mode` / `handoffs`). |
| `rein handoff inspect ID` | Inspect one handoff; record acknowledgement. |
| `rein handoff export ID` | Export json or markdown. |
| `rein resume SESSION --with claude\|codex` | Alias for `handoff --to`; prints a structured-handoff notice. |

Sources: Claude Code, Codex, Gemini CLI, OpenCode, Grok Build.
Destinations: Claude Code and Codex only.

Dest first-reply must restate: (1) current goal and latest user request,
(2) critical constraints, (3) changed files and test state, (4) missing
capabilities or uncertain evidence, (5) proposed next action.

### Phase 5 — universal agent coverage (`v0.5.0`/`v0.5.1`)

| Command | Purpose |
| ------- | ------- |
| `rein doctor --agents [--json]` | Redacted per-agent storage probe across the full catalog. |
| `rein sessions` / `search` / `inspect` | Now cover eleven agents (read-only where native resume is not supported). |
| `rein handoff … --from AGENT` | Five structured-handoff sources: Claude Code, Codex, Gemini CLI, OpenCode, Grok Build. |

Bare `rein` gained development-verified interactive-switcher, handoff-studio,
and setup-wizard surfaces in the `v0.5.2-rc.1` candidate; that candidate was
published but never certified, and its content ships unchanged inside
`v0.6.0` below.

### Phase 6C — Reinstate Hop and the interactive CLI (`v0.6.0`)

| Command | Purpose |
| ------- | ------- |
| `rein login` / `rein whoami` | Passwordless Hop sign-in and identity. |
| `rein init --hop` | Wizard-driven setup of the hosted locker, with `--link`/`--paste` pairing. |
| `rein account init\|recover\|join\|status` | Create, recover, join, or inspect a Hop account and its keyring. |
| `rein devices` / `approve` / `revoke` | Pair and manage devices on the account. |
| `rein hop status` / `rein hop credentials` | Inspect the locker and current Hop credentials. |
| `rein sync verify` | Verify the locker's claims byte by byte. |
| `rein sync migrate --to byo` | Leave Hop for your own S3/R2 bucket. |
| `rein daemon` | Resident background sync process (launchd on macOS, systemd `--user` on Linux, Task Scheduler on Windows). |
| `rein` (TTY) | Interactive TUI switcher: per-row readiness, a handoff studio, a setup wizard, a `ctrl+k` palette. `--plain`/`REINSTATE_NO_TUI` keep every `--json` document and non-TTY stream byte-identical to `v0.5.1`. |

The hosted control plane this client talks to by default is not open: no
pricing, trial, or sign-up ships. `REINSTATE_HOP_URL` / `[hop] url` point the
client at a control plane you run yourself.

OpenCode reaches **T5** (encrypted same-vendor sync).
Kimi Code CLI reaches **T2** (handoff source).
OpenCode, Grok Build, and Qwen Code are structured-handoff destinations
alongside Claude Code and Codex. See [Reinstate Hop](hop.md).

## Not in stable v0.6.0

- reconstructed cross-agent conversation written into vendor storage
- Gemini CLI or Kimi Code CLI as handoff destinations or native-resume agents
- encrypted sync beyond Claude Code, Codex CLI, and OpenCode
- MCP, skill, plugin, marketplace, or credential sync; universal configuration
- Apple Silicon macOS acceptance for `v0.6.0` (deferred to
  [#403](https://github.com/HarjjotSinghh/reinstate/issues/403)); Intel macOS
  or Linux/WSL2 as certified platforms
- a Reinstate-owned agent runtime; team continuity
- pricing, a trial, or sign-up for the Hop hosted control plane

See [ROADMAP.md](../ROADMAP.md).
