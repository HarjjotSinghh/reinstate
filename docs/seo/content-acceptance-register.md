# Published editorial content acceptance register

This register records repository-local technical and editorial review for every
published guide and blog article. It does not claim physical-device
acceptance, external fact-checking, production traffic, or maintainer approval.
Those remain separate evidence gates.

## Review identity

| Field | Value |
| ----- | ----- |
| Reviewer | Codex coding agent |
| Review date | 2026-07-27 |
| Product baseline | `v0.1.0-rc.6` |
| Sources of truth | `PRODUCT.md`, `docs/product-strategy.md`, implementation, deterministic tests |
| Automated evidence | `make verify`; website tests/build; SEO, link, freshness, media, performance, Lighthouse, and schema-parity gates |
| Boundary | Source and generated-artifact review; no physical two-device or external-provider test claimed |

## Acceptance dimensions

Each accepted page was checked for:

1. an answer-first opening that remains accurate when extracted;
2. unique intent and original explanatory value;
3. current-versus-planned product truth and stated limitations;
4. scoped commands with dry-run/verification and no silent broad transfer;
5. security boundaries, including transcript-content risk;
6. visible content matching emitted article, HowTo, FAQ, and breadcrumb data;
7. canonical internal links to installation, integration, security,
   compatibility, and troubleshooting sources where relevant; and
8. a route-specific 1200×630 Reinstate card plus passing build gates.

## Page decisions

| Canonical | Type | Commands | Original value and limitations | Schema/answer evidence | Decision |
| --------- | ---- | -------- | ------------------------------ | ---------------------- | -------- |
| `/guides/sync-claude-code-sessions-across-devices` | Guide | Scoped Claude session push/pull with dry-runs, verification, rollback, and failure paths | Explains project-slug portability and same-vendor resume; open native-device gates are explicit | Shared visible `HowTo` steps, `TechArticle`, breadcrumbs, answer-first intro | Accepted within stated boundary |
| `/guides/sync-codex-sessions-across-devices` | Guide | Scoped Codex rollout push/pull with dry-runs, verification, rollback, and failure paths | Explains structural `session_meta.cwd` rewriting without transcript rewriting; open gates explicit | Shared visible `HowTo` steps, `TechArticle`, breadcrumbs, answer-first intro | Accepted within stated boundary |
| `/guides/move-a-coding-agent-session-from-mac-to-windows` | Guide | Agent/session-scoped transfer and destination verification | Explains canonical project IDs and cross-OS path mapping; WSL and physical acceptance qualified | Shared visible `HowTo` steps, `TechArticle`, breadcrumbs, answer-first intro | Accepted within stated boundary |
| `/guides/use-s3-for-coding-agent-session-storage` | Guide | Reviewed S3 setup and scoped encrypted push | Separates endpoint, bucket, keyring, encryption, retention, and least privilege; no provider guarantee | Shared visible `HowTo` steps, `TechArticle`, breadcrumbs, answer-first intro | Accepted within stated boundary |
| `/guides/use-cloudflare-r2-for-coding-agent-session-storage` | Guide | Reviewed R2 setup and scoped encrypted push | Documents R2-specific setup without claiming a Reinstate-hosted service; secret handling explicit | Shared visible `HowTo` steps, `TechArticle`, breadcrumbs, answer-first intro | Accepted within stated boundary |
| `/blog/why-git-does-not-sync-coding-agent-sessions` | Article | No mutating workflow; command criterion not applicable | Distinguishes source history from vendor session state without framing Git as a replacement target | `BlogPosting`, breadcrumbs, visible headline/description, answer-first thesis | Accepted within stated boundary |

## Evidence-gated follow-up

- Maintainer editorial approval is required before production promotion.
- Physical macOS/Windows and two-device acceptance must not be inferred from
  this review.
- External vendor documentation and compatibility ranges must be rechecked
  when the freshness gate warns.
- Any command, schema, title, description, or product-scope change reopens the
  affected row.
