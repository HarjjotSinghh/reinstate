# Architecture & MVP Plan

## Strategic product shapes (choose deliberately)

### Shape 1 — Universal session sync wrapper (literal idea)

Multi-agent adapters, E2EE, BYO storage, path rewrite, push/pull.  
**Risk:** scope explosion; races vendor clouds on sessions.

### Shape 2 — Skills / rules / agents center (MiniMax preferred center)

Git or object store as home for durable artifacts; sessions opt-in.  
**Risk:** weaker emotional demo than resume.

### Shape 3 — Own cloud-native agent (DeepSeek preferred)

Don’t wrap others; build `reinstate agent` multi-provider with native cloud sessions.  
**Risk:** compete with Claude/Codex feature velocity; huge product surface.

### Shape 4 — Dev-environment-as-code (chezmoi for agents)

Pure OSS apply/diff; no SaaS.  
**Risk:** monetization; slower growth.

**Master synthesis default:** Shape 1+2 hybrid — **profile (durable) + same-agent sessions (hook) + checkpoints**, not Shape 3 at start.

---

## Recommended pipeline (Kimi)

```text
Adapters → Watcher (fsnotify, debounced, append-aware)
        → Normalizer (path tokens, ignore rules, redaction)
        → Encryption (age / client keys)
        → Sync engine (manifest SQLite, CAS chunks, delta)
        → Backends (R2 / S3 / GCS / WebDAV / Gist / git)
```

Restore invert: pull → decrypt → path-rewrite → atomic swap + backup.

### Technology choices

| Layer | Recommendation | Rationale |
|-------|----------------|-----------|
| Language | Go or Rust static binary | Zero-deps install; claude-sync pattern |
| Crypto | age + Argon2 passphrase KDF; 0600 files | Same passphrase = same key UX |
| Backends | S3-compatible first (R2 free tier) | BYO trust |
| Sync | Manifest + CAS + tail-append JSONL | Multi-GB histories |
| Conflicts | Never overwrite; conflict forks | Fail-safe |
| Adapters | Golden fixtures + defensive parse | Format churn |
| Distribution | npm optional + GitHub releases + curl | Windows/macOS/Linux |

**Omit v1:** CRDTs, multiplayer, multi-tenant custom protocol, mobile apps, full cross-agent translation.

---

## Agent adapter interface (ChatGPT)

```ts
interface AgentAdapter {
  detectInstalled(): Promise<AgentInstall | null>
  listSessions(filter?: Filter): Promise<SessionMeta[]>
  readSession(id: string): Promise<CanonicalSession>
  writeSession(session: CanonicalSession, opts: WriteOpts): Promise<void>
  listCapabilities(): Promise<CapabilityProfile>
  applyCapabilityOverlay(profile: Profile): Promise<void>
  supportsNativeResume(): boolean
  doctor(): Promise<DoctorReport>
}
```

---

## Canonical representation (sketch)

```json
{
  "schema_version": 1,
  "session_id": "ds_01...",
  "source": {
    "agent": "claude-code",
    "agent_version": "...",
    "adapter_version": "..."
  },
  "intent": { "goal": "...", "decisions": [], "next_action": "..." },
  "conversation": { "events": [], "summary": "..." },
  "workspace": {
    "repo_key": "github.com/org/repo",
    "head": "abc123",
    "dirty": true,
    "changed_files": []
  },
  "capabilities": { "mcp": [], "skills": [], "hooks": [] },
  "environment": { "os": "windows", "path_map": {} },
  "provenance": { "created_at": "...", "fork_of": null }
}
```

Separate **metadata** (SQLite index, search, cursors) from **blobs** (encrypted payloads).

---

## Phased MVP (consensus merge)

### Phase 0 — Local doctor & profiles (no cloud required)

- Detect agents on macOS / Windows / WSL  
- Diff MCP/skills/hooks/instructions across tools  
- Device profile + overlays  
- **Ship DevSync heritage here** (MCP/skills/profile plane inside Reinstate)  

**Gate:** useful on a single machine; config drift visible.

### Phase 1 — Same-agent cross-device resume (the wedge)

Agents: **Claude Code + Codex** first (largest + different storage shapes).

Commands:

```text
init | push | pull [--dry-run] | status | conflicts | resume | doctor
```

Must-have:

- E2EE + BYO storage  
- **OS-aware path remapping** (folder keys + in-file paths)  
- Atomic writes + timestamped backups  
- Benchmark: Windows session → Mac `claude --resume` / Codex equivalent **&lt;10s**, no corruption, survives 3 agent releases  

**Gate:** stranger installs and completes cross-OS resume in &lt;5 minutes.

### Phase 2 — Expand adapters + fold config scope

- Gemini CLI (project hash remap)  
- OpenCode (safe SQLite/API)  
- Config scope: one canonical definition → each tool’s native files  
- Optional shell-hook auto push/pull  

**Gate:** fresh machine bootstrap restores sessions **and** capabilities that make them behave.

### Phase 3 — Portable checkpoints / handoffs

```text
checkpoint | handoff --agent <target>
```

Kontinuo-class compact packages; best-effort cross-agent continuity **without** claiming full replay.

### Phase 4 — Experimental native migration

```text
migrate --to codex
```

CASR-like IR; directions separate (Claude↔Codex first); golden fixtures; read-after-write verify.

### Phase 5 — Open adapter SDK + convenience

```text
adapter init | test | publish
```

Hosted zero-knowledge relay optional; local web session browser; team share of **selected** sessions.

---

## Alternative phase numbering (Claude research)

| Stage | Focus |
|-------|--------|
| 0 | Fold unified profile into Reinstate (MCP+skills+agents+instructions+settings; DevSync heritage) |
| 1 | Claude session sync + path remap + E2EE (founder Windows↔Mac) |
| 2 | Codex + Gemini + OpenCode adapters |
| 3 | Profile + best-effort context handoff |

Same substance as above.

---

## Gemini deep research roadmap (IDE later)

1. Terminal agents beachhead  
2. MCP + skills path tokenization  
3. Cursor/IDE SQLite (hardest moat attempt)  
4. Market on zero-knowledge security  

---

## What not to build first

Compiled denylist:

1. Polished multi-tenant web dashboard as v1  
2. Team realtime collaboration  
3. Semantic search across everything  
4. Twelve agent adapters day one  
5. Cross-agent perfect replay marketing  
6. Depending on Remote Control / teleport  
7. Syncing auth tokens  
8. Plaintext remote storage  
9. Custom CRDT sync protocol  
10. Competing as “another Claude Code” (unless explicitly Shape 3)

---

## Validation plan (cheap, targeted)

1. Post v0.1 into open FR threads (Claude #cross-device, Codex #14067)  
2. r/ClaudeCode / r/ClaudeAI multi-machine threads + 2-min Loom  
3. Dogfood metrics:

| Metric | Why |
|--------|-----|
| Time desktop→laptop resume | Core value |
| % resumes READY vs DEGRADED vs BLOCKED | Capability plane quality |
| Conflict rate | Sync model fitness |
| Adapter break rate per agent release | Maintenance load |
| Secrets redacted count | Security posture |
| Setup time to first success | Distribution |

---

## Kill / pivot signals

| Signal | Response |
|--------|----------|
| Anthropic ships account-level **local** session sync with **automatic cross-OS path remapping** | Double down on multi-vendor profile + handoffs; de-emphasize Claude-only session wedge |
| OpenAI-only users satisfied by native account sync | Still sell multi-agent + Claude/Gemini/OpenCode |
| Trust incident | Existential — design so architecture makes plaintext impossible |
| Format churn unmaintainable for files-first | Shift weight to hooks/SDK |

---

## CLI sketch (product: **Reinstate**, short alias: **`rein`**)

Prefer `rein` in examples; `reinstate` is the full binary name (same tool).

```bash
rein init
rein doctor
rein push
rein pull --dry-run
rein status
rein conflicts
rein list
rein resume <id>
rein checkpoint <id>
rein handoff <id> --to codex
rein migrate <id> --to claude   # experimental
```
