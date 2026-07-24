# Vendor Capabilities Matrix

*As of research window mid/late 2026. Verify before building adapters.*

## Snapshot matrix

| Capability | Claude Code | Codex | Gemini CLI | OpenCode | Grok Build | Cursor |
|------------|-------------|-------|------------|----------|------------|--------|
| Local session persistence | Yes (JSONL) | Yes (rollouts + index) | Yes (project-hash) | Yes | Yes (flags) | Yes (SQLite) |
| Same-machine resume | `--continue` / `--resume` / picker | Thread resume / fork | UUID / index / latest | API + local | `-r` `/resume` | In-app |
| Cross-device **local** store sync | Partial / FR pressure; SessionStore SDK | Account-level surfaces strong | Weak cloud | None first-party | None | Weak/manual |
| Cross-device **cloud execution** | Web sessions, teleport | Cloud threads | Limited | — | Upload traces | — |
| Live remote control | Remote Control | Mobile↔desktop relay | — | — | Community | — |
| Documented external session store | SessionStore adapters | App-server APIs | Docs for local only | HTTP OpenAPI | Evolving | Opaque |
| MCP config | `~/.claude.json` etc. | `config.toml` | settings | config dirs | varies | settings |
| Hooks lifecycle | Mature | Beta (`codex_hooks`) | Wide + OTel | varies | limited | IDE |
| Skills / agents / commands | First-class dirs | `.codex/skills` etc. | dirs + GEMINI.md | skills dirs | limited | product-specific |
| Format stability promise | **Explicitly unstable** JSONL | Evolving SQLite/JSONL | Evolving | Evolving | New | Opaque DB |

---

## Claude Code (deepest coverage in sources)

### Storage

- Base: `~/.claude/projects/` (macOS/Linux); `%USERPROFILE%\.claude\projects\` (Windows); override via `CLAUDE_CONFIG_DIR`
- Project key: **flattened absolute path** — non-alphanumeric → `-`  
  - **Slug collision risk:** `.` and `-` both become `-` → unrelated paths can collide; use in-file `cwd` as ground truth
- Sessions: `*.jsonl` event streams (prompts, tools, results, …)
- Retention: default cleanup (~30 days `cleanupPeriodDays`) → cloud sync also = archival
- Desktop / web / VS Code maintain **own** session histories (not one unified local picker)

### Resume & cross-host

- `--continue`, `--resume`, naming, branching, pickers  
- Cross-host: move session files to same path/`cwd` **or** mirror via **SessionStore** (S3, Redis, DB)  
- Docs: conversation ≠ filesystem state  

### Cloud / multi-surface features

- Web sessions persist across devices  
- `--teleport`: typically **cloud → local** one-way from CLI  
- Desktop “Continue in” reverse in some cases; clean tree / subscription constraints  
- Remote Control: live attach; transcript on Anthropic servers while connected; origin process must run  
- Routines, Channels, Cowork, auto-memory, `--cloud`, background `&` sessions (feature flood Q1 2026 claims)

### Hooks (durable capture path)

SessionStart, SessionEnd, UserPromptSubmit, PreToolUse, PostToolUse, PreCompact, PostCompact, Stop, SubagentStop, Notification

---

## OpenAI Codex

### Storage

- Rollouts under date-sharded paths e.g. `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`  
- Index/state: SQLite (e.g. `state_5.sqlite` mentioned) — multi-GB histories reported  
- Config: `~/.codex/config.toml` + project `.codex/config.toml`  
- Some content may be account-encrypted (`encrypted_content`) — **breaks across accounts**; don’t invent token bridging  

### Continuity

- App-server: `thread/start`, `thread/resume`, `thread/fork`  
- Resume can **fail if required MCP missing** → validates capability-aware resume  
- Shared MCP config across Codex surfaces  
- Import settings/instructions/skills/plugins/projects/chats from other agents (vendor claim)  
- Mobile app syncs with desktop environment without exposing machines to public internet (relay)

### Hooks

Beta: SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop (`codex_hooks = true`)

---

## Gemini CLI

### Storage

- Project-scoped under hash path (e.g. `~/.gemini/tmp/<hash>/chats/` or docs-equivalent)  
- Complete history: tools, outputs, tokens, reasoning when available  
- Resume: latest / index / full UUID  
- Session browser, checkpoints  
- Default retention window ~30 days  

### Continuity

- Strong **local** story  
- Cloud “push conversation” still FR territory (Antigravity forum)  
- Project hash must be remapped when path/OS changes  

### Hooks

Very wide surface + OpenTelemetry streaming — strong for event-based capture without parsing unstable files

---

## OpenCode

- Local: `~/.local/share/opencode/` (and config under `~/.config/opencode/`)  
- Headless server: list/create/fork/share/diff/summarize sessions; messages; send; optional basic auth  
- Friendly for third-party integration  
- SQLite-ish storage in some community notes — use safe backup patterns (`.backup`)  

---

## Grok Build / xAI tooling

- Young OSS CLI; session flags: `-s`, `-r`, `-c`, `/resume`, `/sessions`, `/fork`  
- Research claim: default upload of session traces to GCS bucket `grok-code-session-traces`; “Improve the model” toggle may not fully disable  
- **Recommendation:** delay adapter or strip uploads / quarantine until privacy story is clean  

---

## Cursor / IDEs

- SQLite `state.vscdb` under platform Application Support paths  
- Workspace keyed by **MD5 of absolute path** → same repo different path = different chat  
- Syncing running IDE DBs is high-risk (WAL, locks)  
- Phase **late** (Gemini deep research Phase 3); use hash translation + offline backup, never live corrupt  

---

## Agent Client Protocol (ACP)

- Evolving standard for session lifecycle including **`session/resume`**  
- Resume expects session ID + working directory + MCP reconnection correctness  
- Align canonical model with ACP concepts where possible  

---

## Implications for product

1. **Beachhead:** file-based terminal agents (Claude JSONL, Codex rollouts, Gemini project stores) before IDE SQLite.  
2. **Cooperate with vendor APIs** (SessionStore, App Server, OpenCode HTTP, hooks) rather than only reverse-engineering.  
3. **Never depend** on Remote Control / teleport / subscription-only cloud as the core offline story.  
4. **Treat format churn as permanent** — adapters + fixtures + defensive parse.
