# Features and commands (v0.1.0–v0.5.0)

Stable `v0.5.0` is the current release. It includes every shipped surface from
Phase 1 through Phase 5. This page is the command map. Details live in
[CLI reference](cli-reference.md), [getting started](getting-started.md), and
[handoff](handoff.md).

`rein` and `reinstate` are the same binary.

A structured handoff starts a **new destination session continuing the same
task**. It is not native resume, not the same session, and not lossless
transfer.

## What each stable release added

| Release | Phase | What you can do |
| ------- | ----- | ---------------- |
| `v0.1.0` | Encrypted sync | Push and pull same-vendor Claude Code and Codex sessions through client-side-encrypted S3-compatible storage you own. |
| `v0.2.0` | Local index | Find, search, inspect, and same-vendor resume/fork sessions on one machine with no `init` or bucket. |
| `v0.3.0` | Verified resume | Gate native launch on a local environment report (workspace, agent, capabilities, runtime) with exact warning acknowledgement. |
| `v0.4.0` | Structured handoff | Continue the same task in a *new* Claude Code or Codex session, including Claude ↔ Codex, with a visible projection and five-bullet first-reply. |

Supported mandatory platforms: Apple Silicon macOS and native Windows x64.
Intel macOS and Linux/WSL2 remain optional and unverified
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

## Not in v0.4.0

- reconstructed cross-agent conversation written into vendor storage
- Gemini / OpenCode / Grok as destinations
- MCP, skill, plugin, marketplace, or credential sync
- Intel macOS or Linux/WSL2 as certified platforms
- a Reinstate-owned agent runtime

See [ROADMAP.md](../ROADMAP.md).
