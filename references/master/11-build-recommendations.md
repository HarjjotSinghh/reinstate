# Build Recommendations — Reinstate

## Go / no-go

| Source | Verdict |
|--------|---------|
| ChatGPT deep research | **Yes** — adult version, not transcript Dropbox |
| ChatGPT conversation | **Yes** — control plane + three resume modes |
| Claude deep research | **Yes** — cross-vendor + path remap + E2EE wedge |
| Kimi | **Yes** — universal tool, not better claude-sync |
| Gemini deep research | **Yes** — phased; E2EE moat |
| Gemini (blunt) | **Yes** if dirty-state & security solved |
| GLM | **Worth building** — resist universal temptation in v1 |
| Grok | **Solid / validated pain** |
| Meta AI | **Very good idea** — 4 layers; don’t fight Omnigent-class |
| MiniMax | **Build** — center durable artifacts; sessions opt-in feature |
| Perplexity (+ deep) | **Yes** — not S3-for-sessions |
| DeepSeek | **Yes** but prefers **own agent** over wrappers |

**Master decision:** Build **Reinstate** as multi-agent local-first continuity + capability plane. Optional later: own agent (DeepSeek) only if meta-layer stalls.

---

## Build this first (ordered)

1. **`reinstate doctor` + device profiles** — detect agents, MCP/skills drift, path maps (no cloud required)  
2. **Claude Code same-agent cross-OS resume** with path remapping + E2EE + BYO storage  
3. **Codex same-agent resume** (second storage shape forces real adapters)  
4. **Atomic safety** — dry-run, backups, conflict forks  
5. **Capability-aware resume** — READY / DEGRADED / BLOCKED  
6. **Gemini + OpenCode adapters**  
7. **Portable checkpoints / handoffs** (Kontinuo-class)  
8. **Experimental native migration** (CASR-class)  
9. **Hosted Reinstate Cloud** (after OSS core proves trust)  
10. **Adapter SDK** for community agents  

---

## Explicit non-goals for v1

- Perfect cross-agent mid-session replay as marketed feature  
- Building on Remote Control / teleport as the offline story  
- Syncing auth tokens / account-bound encrypted blobs  
- Twelve agents day one  
- Polished multi-tenant dashboard before CLI magic works  
- Real-time multiplayer CRDT sessions  
- Competing as full Claude Code replacement (unless strategic pivot)  
- Grok adapter before privacy review  
- Cursor SQLite live sync as beachhead  

---

## Engineering principles

| Principle | Practice |
|-----------|----------|
| Local-first | Works offline; cloud is convenience |
| Fail-safe writes | Never silent overwrite of history |
| Honest modes | Native / handoff / experimental migration |
| Path identity | Git remote keys, not absolute paths |
| Overlays | Per-device config differences allowed |
| Hooks + files | Hybrid capture |
| Open inspectable core | Encryption/redaction code public if commercializing |
| Phase gates | Evidence before scope growth |

---

## Suggested repo layout (indicative)

```text
reinstate/                 # OSS core (Apache-2.0 suggested)
  cli/
  adapters/                # claude, codex, gemini, opencode
  canonical/               # schema, checkpoints
  crypto/
  sync/                    # manifest, CAS, backends
  doctor/

reinstate-cloud/           # private hosted (later)
```

---

## Validation sequence

1. Dogfood Windows ↔ MacBook daily for 2 weeks unattended  
2. Drop v0.1 into Claude/Codex FR threads that requested this  
3. Loom to r/ClaudeCode / r/ClaudeAI  
4. Comparison page for SEO  
5. Only then: hosted free tier + paid cloud  

---

## Kill / pivot triggers

| If… | Then… |
|-----|--------|
| Anthropic ships path-aware local session account sync | Keep multi-agent + profile; de-emphasize Claude session wedge |
| Adapter maintenance dominates | Hooks/SDK-only capture; fewer agents |
| Users only want skills/MCP sync | Lean MiniMax durable-center packaging under Reinstate |
| Trust incident | Stop feature work; security postmortem first |
| Zero activation outside founder | Narrow to personal tool; don’t force SaaS |

---

## First-week checklist

- [ ] Canonical schema v0 for session + checkpoint + capability profile  
- [ ] Claude adapter: list/read/write with path rewrite  
- [ ] age encryption + passphrase flow  
- [ ] S3/R2 backend interface  
- [ ] `reinstate doctor` for one machine  
- [ ] Backup-before-write  
- [ ] End-to-end Windows→Mac resume demo script  
- [ ] SECURITY.md threat model stub  
- [ ] Trademark search kickoff (name clearance)  

---

## One-sentence operating thesis

**Reinstate is the neutral, encrypted, path-aware control plane that reinstates agent workspaces across devices — with sessions as the hook and capabilities as the moat.**
