# W6 — Website cards

One executor, **after every code task has merged**. The website asserts product
truth, and product truth is not settled until the tiers are final.

Build and test with the Node version the website pins in
`website/package.json`. A mismatched Node version fails `npm ci` in ways that
look like unrelated build errors.

---

## T-060 — Compatibility data

**Owns:** `website/src/data/compatibility.json`,
`website/src/data/agent-version-history.ts`

1. Add every catalog agent with its final tier, storage family, and evidence
   references.
2. Update `reinstateVersion`, `lastReviewed`, `lastTested`, and
   `fixtureCommit`.
3. Add version-history entries for any agent promoted to T3.
4. Keep the file consistent with `docs/compatibility.md` and the adapter
   constants. `website/src/lib/linkable-assets.test.ts` asserts all three agree,
   which is exactly the guard you want here.

---

## T-061 — Integrations pages and the agent matrix

**Owns:** `website/src/pages/integrations/**`,
`website/src/pages/compatibility*.astro`, the agent matrix component

1. Render the tier ladder as a matrix. One row per agent, one column per tier,
   with the evidence linked.
2. Add integration pages only for agents at T1 or above. A T0 agent belongs in
   the matrix with its reason, not on a page that implies an integration.
3. Every capability statement names its tier.

**The copy rule.** State breadth as a matrix, never as an absolute.

Acceptable:

> Reinstate indexes sessions from eleven coding agents. Encrypted sync covers
> Claude Code and Codex CLI.

Rejected by `website/src/lib/comparison-pages.test.ts`:

> Works with all coding agents. Seamless cross-agent sync.

Those tests are correct. Do not weaken them to fit a headline.

---

## T-062 — Roadmap page

**Owns:** `website/src/pages/roadmap.astro`

1. Renumber to match `ROADMAP.md`: Phase 5 universal agent coverage, Phase 6
   universal configuration, Phase 7 Console, Phase 8 team continuity.
2. Add the Phase 5 rows with accurate statuses.
3. `website/src/lib/evidence-pages.test.ts` asserts a minimum count of
   `<td>Stable</td>` cells. Adding planned rows must not reduce the stable
   count below the assertion. Read the test before editing the page.

---

## T-063 — Contract tests and copy review

**Owns:** `website/src/lib/*.test.ts`

1. Extend `linkable-assets.test.ts` to assert the catalog, `compatibility.json`,
   and `docs/compatibility.md` agree on every agent's tier.
2. Add an assertion that no page claims a capability above an agent's tier.
3. Update `product-truth.test.ts` for the `v0.5.0` release line.
4. Run the full suite: `npm --prefix website test`.
5. Read every changed page's copy by eye. Mechanical tests catch absolutes;
   they do not catch a sentence that is technically true and misleading.

**Do not delete or relax an existing assertion to make a page pass.** If an
assertion blocks accurate copy, that is a finding for the coordinator. Every
one of these tests exists because a specific overclaim shipped once.
