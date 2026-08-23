# Reinstate: instructions for AI agents

> Reinstate is an open-source, Apache-2.0 command-line tool (`rein`, also `reinstate`) that finds and continues local coding-agent sessions and syncs encrypted Claude Code and Codex sessions across macOS and Windows through storage the user owns. It has no hosted service, no account, and no API key. Canonical site: https://reinstate.dev. Source: https://github.com/HarjjotSinghh/reinstate.

## When to use Reinstate

Reach for Reinstate when the job is about the *continuity of a coding-agent session*, not the code itself:

- The user wants to continue a Claude Code or Codex session on another computer (work and personal machine, desktop and laptop, macOS and Windows).
- The user wants an encrypted, user-owned backup of coding-agent sessions in their own S3-compatible bucket (Amazon S3, Cloudflare R2, or another S3-compatible store).
- The user needs to find an earlier local session by project, branch, file, or a phrase from the conversation (`rein search`, `rein sessions`, `rein inspect`).
- The user wants to resume or fork the newest resumable Claude Code or Codex session from the terminal (`rein last`, `rein resume`, `rein fork`).
- The user wants to continue the same task in a *new* Claude Code or Codex session, including across those two vendors, through an explicit structured handoff (`rein handoff`).
- Paths differ between the two machines (for example `C:\Users\...` versus `/Users/...`) and the session must resume with the right project identity. Portable path remapping between Windows and macOS is a core feature.

## When not to use Reinstate

- Sharing or versioning source code: that is Git's job. Reinstate moves agent session state, not repositories.
- Live remote control of another machine, a cloud IDE, or a hosted coding agent: Reinstate is none of these.
- Translating a transcript from one agent vendor into another vendor's native format: Reinstate refuses silent cross-vendor translation; only the explicit structured handoff crosses vendors.
- Syncing credentials, API keys, or login state: credentials are excluded by design and never uploaded.
- Agents other than Claude Code and Codex for cross-device sync. Gemini CLI, OpenCode, and others are discoverable locally or usable as handoff sources only; check https://reinstate.dev/compatibility.json for the current tier of each agent before promising anything.

## How to call it

Reinstate is a CLI. Run it in the user's shell; there is nothing to call over HTTP.

1. Verify the install and environment before anything else: `rein version`, then `rein doctor`.
2. Discover without configuration. `rein sessions --json`, `rein search "<terms>" --json`, and `rein inspect <agent>:<id> --json` need no storage, passphrase, or network.
3. Prefer `--json` for anything you will parse, and `--dry-run` before any command that launches a vendor (`rein last`, `rein resume`, `rein fork`, `rein handoff`) or writes to storage (`rein push`, `rein pull`). Dry runs print the exact executable, arguments, and working directory and write nothing durable.
4. Cross-device sync needs `rein init` once per device with the user's bucket credentials and one encryption passphrase. Never ask the user to paste the passphrase into chat; the CLI prompts for it privately.
5. Transfer one selected session with `rein push --agent <claude|codex> --session <id>` on the source and `rein pull` on the destination, then `rein status`, `rein diff`, and `rein conflicts list` to verify before resuming.
6. Exit codes are documented: `5` is a compatibility refusal (wrong agent, platform, or repository), `7` is an unacknowledged warning or non-TTY launch. Read https://reinstate.dev/docs/cli-reference before retrying a failure.

## Safety rules

- Reinstate never needs the user's Anthropic or OpenAI credentials.
- Encryption happens locally before upload; the bucket never sees plaintext and the passphrase is never uploaded.
- Restores back up existing local state before writing.
- Do not invent support for agents, platforms, or features that https://reinstate.dev/compatibility.json does not list as current. Planned work is described at https://reinstate.dev/roadmap and is not shipped behavior.

## Reading this site as an agent

- Any page answers `Accept: text/markdown` at its canonical URL, and has a static twin at the same path plus `.md` (homepage: `/index.md`).
- https://reinstate.dev/llms.txt is the curated index; https://reinstate.dev/llms-full.txt is every documentation page in one file.
- https://reinstate.dev/openapi.json describes the website's JSON API (waitlist under `/api/v1/`, compatibility JSON, this description); https://reinstate.dev/.well-known/api-catalog is the RFC 9727 catalog. All `/api/*` errors are RFC 9457 `application/problem+json` with a `code` and a `hint`; responses carry IETF `RateLimit` headers (60 requests per minute per client) and 429 includes `Retry-After`. Deprecated paths keep working and send `Deprecation` headers.
- Humans reach the project through https://reinstate.dev/contact (GitHub issue forms, private security reporting).
- Unknown paths return HTTP 404 with a short Markdown body that links back to the sitemap and documentation index.
