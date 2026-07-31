# Source Map

Maps every original reference file to what it contributed to this master synthesis. Use this to verify nothing important was dropped.

| Original file | Role | Unique / heavy contributions |
|---------------|------|------------------------------|
| **chatgpt-deep-research.md** | Deep research report | Ecosystem matrix; SpecStory/CASR/Kontinuo/ACP/Warp; security table; strong vs weak product; portable checkpoint fields; final rec table |
| **chatgpt.md** | Long product design chat | Six-state model; 3 resume modes; capability-aware resume; architecture phases; security deep dive; **Reinstate naming decision**; Carryover comparison; `reinstate.dev`; open-core vs closed vs OSS; business scores |
| **(product decision)** | Short alias | **`rein`** chosen as daily CLI alias for Reinstate — see `10-naming.md` |
| **claude-deep-research.md** | Competitive/technical landscape | Vendor capability matrix; storage formats; competitor teardown (broad list); demand FRs; strategic assessment; staged recommendations; caveats |
| **claude.md** / **claude.pdf** | Expanded Claude research conversation | Overlaps deep research; extra competitor names; repeated matrices — treat deep research as clean extract |
| **deepseek.md** | Strategic alternative | **Approach A vs B** (sync wrapper vs **own cloud agent**); architecture for own agent; risks |
| **gemini-deep-research.md** | Long-form strategic eval | Developer retrieval problem; Claude/Cursor storage anatomy; path hashing; E2EE emphasis; phased IDE moat; zero-knowledge marketing |
| **gemini.md** | Blunt risk note | **Dirty-state desync** as final boss; hijacking agents; latency/token bloat; security yikes |
| **glm.md** | Condensed build advice | Per-CLI storage why “just sync file” breaks; warm competitive space; hard problems; MVP resist-universal; moat beyond file sync |
| **grok.md** | Short reality check | Field validation 2025–26; go-build yes |
| **kimi/kimi.md** + **assets/** | Full deep research + diagrams | Demand timeline; 4 categories empty quadrant; traction numbers; incumbent risk; 5 hard problems; MVP pipeline; risk register; verdict; footnotes/URLs |
| **metaai.md** | Architecture sketch | 4 layers to sync; hard problems; Dev Sync v2 architecture; positioning vs Omnigent-class |
| **minimax.md** | Contrarian depth | Vendor cloud coverage; Depot/Omnara/claude-sync; format landmines; path-slug collisions; hooks-first; **3 defensible positions**; durable artifacts center; rename off Dev Sync; full artifact table |
| **perplexity-deep-research.md** | Differentiation table | Reality check exists; still has teeth; build/not build; security MCP; business traction |
| **perplexity.md** | Product sketch | Context capsule; MVP; data model; who pays; validation; build order |

---

## Coverage checklist (topics → sources)

| Topic | Primary sources |
|-------|-----------------|
| Problem / multi-device pain | All |
| Dirty workspace desync | gemini.md, chatgpt, kontinuo via chatgpt-deep |
| Demand FRs / DIY | kimi, claude-deep, minimax |
| Competitor inventory | kimi, claude-deep, minimax, chatgpt |
| Vendor native features | kimi, chatgpt-deep, minimax, claude-deep |
| Session formats / paths | gemini-deep, claude-deep, minimax, glm, kimi |
| Architecture / MVP | kimi, chatgpt, claude-deep, gemini-deep, metaai, perplexity |
| Security | chatgpt, chatgpt-deep, kimi, gemini-deep, perplexity |
| Naming → **Reinstate** | **chatgpt.md only (decisive)** |
| Open-core business split | chatgpt.md |
| Own-agent alternative | deepseek.md |
| Durable-first strategy | minimax.md |
| Diagrams | kimi/assets → master/assets |

---

## Explicit disagreements preserved in master

| Dispute | Where resolved in master |
|---------|---------------------------|
| Sync wrappers vs new agent | 00, 06, 11 — default wrappers; DeepSeek as alternative |
| Sessions-first vs durable-first | 00, 05, 09 — hybrid; MiniMax weight on durable center |
| Keep DevSync name vs rename | 10 — **Reinstate decided** |
| OSS vs closed vs open-core | 09 — open core + cloud product |

---

## What was *not* fully re-copied

- Full 500+ URL source dumps from chatgpt-deep-research Sources section (linked from original)  
- Full duplicate repeated blocks inside claude.md (same report many times)  
- Raw ChatGPT thinking/tool-call noise except conclusions  
- Entire kimi footnote list (key URLs retained in competitive/demand docs; full list in `../kimi/kimi.md`)  

If you need a raw citation URL, open the original file in the table above.

---

## Suggested reading order for humans

1. `00-executive-summary.md`  
2. `10-naming.md` (Reinstate is the product)  
3. `03-competitive-landscape.md` + `08-strengths-weaknesses-risks.md`  
4. `06-architecture-and-mvp.md` + `07-security-and-trust.md`  
5. `11-build-recommendations.md`  
6. Dive into originals via this map when implementing adapters  

---

## Assets

| File | From |
|------|------|
| [`assets/01_landscape.svg`](../../assets/01_landscape.svg) | kimi |
| [`assets/02_demand_timeline.svg`](../../assets/02_demand_timeline.svg) | kimi |
| [`assets/03_traction.svg`](../../assets/03_traction.svg) | kimi |
| [`assets/04_market.svg`](../../assets/04_market.svg) | kimi |
| [`assets/05_architecture.svg`](../../assets/05_architecture.svg) | kimi |
