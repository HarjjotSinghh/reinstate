# Reference Docs — Head-to-Head Evaluation

Scored on: verifiability, technical depth on the crux (path remapping / formats),
competitive coverage, strategic judgment, intellectual honesty, usability as a working doc.

## Measured

| File | kW | Redundancy | URLs | Issue refs | Paths | Tables | Structure |
|---|---|---|---|---|---|---|---|
| kimi/kimi.md | 8.9 | 0% | 57 (+footnotes, 5 diagrams) | — | 17 | 33 | 30 headings |
| claude-deep-research.md | 3.3 | 0% | 0 | 28 | 17 | 9 | 11 headings |
| chatgpt-deep-research.md | 6.6 | 0% | **621** | 4 | 2 | 33 | 20 headings |
| minimax.md | 7.5 | 0% | 26 | — | 34 | 26 | 24 headings |
| chatgpt.md | 23.5 | 1% | 265 | 0 | 0 | 142 | design chat, not research |
| claude.md | 175.9 | **61%** | 0 | 1368 | 845 | 441 | same report pasted ~10× |
| gemini-deep-research.md | 4.2 | 0% | **0** | 0 | 9 | 0 | **0 newlines — one blob** |
| glm / gemini / grok / metaai / deepseek / perplexity×2 | 0.6–2.0 | 0% | 0–38 | ~0 | few | few | short takes |

## Unique coverage (what only one doc found)

- **chatgpt-deep-research** — sole source on ACP (`session/resume` vs `session/load`, capability
  checks), SpecStory, Kontinuo, CASR. 18 ACP / 31 SpecStory / 31 Kontinuo mentions; zero elsewhere
  except its own sibling chat.
- **kimi** — sole source on **scale & delta sync**: 6.1 GB `~/.codex/sessions`, 328 MB single
  rollouts, 821-session archives → append-aware chunked delta sync. Also retention knobs, conflict
  fail-safe (`.conflict` fork), secrets hygiene.
- **minimax** — sole deep coverage of **hooks-first architecture** (43 mentions; full lifecycle for
  Claude/Codex/Gemini) and the **path-slug collision bug class** (dash and dot both → `-`, so two
  real paths slugify identically; `cwd` in-transcript is the only unambiguous key).
- **claude-deep-research** — densest vendor matrix (8 tools), 25+ competitor teardown, 28 verifiable
  issue numbers, pass/fail benchmarks per stage, explicit kill/pivot signal.
- **gemini.md** (600 words) — "dirty-state desync is the final boss." Highest insight-per-word.
- **gemini-deep-research** — Cursor `.vscdb` SQLite/WAL internals, `cleanupPeriodDays` 30-day purge.

## Verdict

**1. kimi/kimi.md — best overall.** Only doc that is simultaneously cited, structured, complete
(demand → landscape → incumbent risk → feasibility → strategy → MVP → risk register → verdict), and
operationally deep. It's the only one that would survive contact with production (delta sync,
conflicts, secrets, retention).

**2. claude-deep-research.md — best signal-to-length.** 3.3k words carrying more actionable
specifics than docs 5× its size, plus the only self-critical Caveats section. Loses on zero
clickable sources.

**3. chatgpt-deep-research.md — best sourced, most original thinking.** 621 URLs and the only doc
that reframes the problem correctly ("a session is not a safe continuation state", three resume
modes, capability-aware resume). But blind to the competitive field: zero mentions of claude-sync,
teleport, path remapping, Cursor, Grok, Omnara, TokenRip.

**4. minimax.md** — sharpest contrarian architecture take (hooks > file parsing).

**Worst of the deep-research tier: gemini-deep-research.md.** Zero citations, zero document
structure (single 31 KB paragraph), consultant-inflated prose, and it promotes cross-tool session
transpilation as "the ultimate moat" — the exact claim every better-sourced doc rates impractical.

**claude.md** is the largest but 61% duplicate; treat `claude-deep-research.md` as its clean extract.

No single doc is sufficient — that's why `master/` exists. kimi + claude-deep + chatgpt-deep
together cover ~all of it.
