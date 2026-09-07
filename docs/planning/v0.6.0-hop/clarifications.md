# Clarifications for the maintainer — v0.6.0

Questions the plan could not settle from the request, the code, or the
records. Each one names the assumption the work proceeds under, so nothing is
blocked on an answer. Overturning an answer before stable promotion is cheap;
after it, it is a `v0.6.1`.

Written 2026-09-05; status lines updated by the coordinator. Executors
append to the last section only.

**Where things stand (2026-09-07).** `v0.6.0-rc.1` was published 2026-09-06
as a signed GitHub prerelease; its tagged-artifact native Windows acceptance
([`docs/testing/results/2026-09-06-windows-v060rc1.md`](../../testing/results/2026-09-06-windows-v060rc1.md))
ended at 201 of 216 required rows `PASS` — every Hop parity row (16/16) and
every CLI experience row (22/22) passed; all 7 required-row failures were in
the Phase 5 generated matrix (Cursor CLI root-env isolation, Cline/Cursor
`message_count`, `push`/`pull --agent` completion, and OpenCode handoff
determinism), plus one fixture gap (`grok:D4`, no committed
`partial-final-record` fixture). That report does not authorize stable
`v0.6.0`. `v0.6.0-rc.2` (2026-09-07) fixed those seven rows and the fixture
gap, plus one addition beyond that scope — `cline:C3`/`cursor:C3` (search
excluded message body) and the pre-existing `opencode:C3` gap (passed by
title only) were fixed too (closes #405, `search_text` indexes message body
for all three sources). Its own tagged-artifact native Windows acceptance
([`docs/testing/results/2026-09-07-windows-v060rc2.md`](../../testing/results/2026-09-07-windows-v060rc2.md))
ended device verdict `FAIL`: **203 PASS / 5 PARTIAL / 2 FAIL / 5 NOT TESTED**
of **215** required rows. All 22 CLI rows and 15 of 16 Hop rows passed; the
seven rc.1 required-row failures cleared. A post-commit coordinator finding
(report §0.12) found the real Cursor CLI `store.db` schema — `blobs(id,
data)` / `meta(key, value)` — is not the `messages`/`message`/`bubbles`
shape the `v0.6.0-rc.2` reader guessed at and no real store ever had, so
real Cursor CLI sessions kept reporting `message_count: 0` and were not
found by search; `cursor:C2`/`cursor:C3` were re-scored `FAIL` on real data.
The other nine blocking rows — `cline:C3`, `pi:C3` (host/harness, could not
create a real vendor-CLI session), `grok:E1`/`E2`/`E3` (new
`F-GROK-MCP-RECONNECT` finding, host MCP-reconnect stall), `qwen:E1`/`E2`/
`E3` (host credential, excused only when `grok`'s same row passes, which it
did not this run), and `MatrixH:H7` (no operator available to accept UAC) —
are host/harness/operator-availability gaps unrelated to the Cursor store
schema. `v0.6.0-rc.3` (2026-09-07) changed exactly one thing beyond
`v0.6.0-rc.2` — the Cursor CLI store reader (`c03337bc`) — and nothing else:
no agent's tier moved, no compatibility range widened (`RELEASING.md`,
"v0.6.0-rc.2 candidate evidence" and "v0.6.0-rc.3 candidate gate"). Its own
tagged-artifact native Windows acceptance
([`docs/testing/results/2026-09-07-windows-v060rc3.md`](../../testing/results/2026-09-07-windows-v060rc3.md))
ended device verdict `FAIL`: **201 PASS / 3 PARTIAL / 1 FAIL / 10 NOT
TESTED** of **215** required rows. All 22 CLI rows and all 16 Hop rows
passed, and this candidate's own Cursor store-schema fix was confirmed
`PASS` on real data (`cursor:C2`, `cursor:C3`). Fourteen required rows
blocked the verdict: a confirmed code defect (`pi:C3` — the reader never
read `message.content` for real `version:3` Pi sessions, so `prompt_preview`
and search text were silently empty); a version-compatibility block
(`qwen:D1`–`D5`, `E1`/`E2`/`E3`/`E5`/`E6` — the host's real Qwen Code
self-updated to `0.23.0`, above the verified `0.21.13` ceiling, correctly
refusing before building a launch plan); and a headless-harness gap
(`codex:E5`, `opencode:E5`, `grok:E5` — no PTY available to attach a
genuinely active vendor process). A lab incident also occurred during this
run: the `H7` daemon Task Scheduler started ran with the host's login
environment (not the isolated home `rein daemon install` ran under) and
pulled synthetic sessions into the live Claude, Codex, and OpenCode stores;
cleaned up, tracked as
[issue #424](https://github.com/HarjjotSinghh/reinstate/issues/424).
`v0.6.0-rc.4` (2026-09-07) is the current candidate: it changes exactly two
things beyond `v0.6.0-rc.3` — the Pi reader, and the verified Qwen Code
range (`0.21.12`–`0.23.0`, Windows evidence) — and nothing else: no other
agent's tier moves, no other compatibility range widens.

Two dispositions were adopted autonomously (Q19) so the stable gate is not
permanently unreachable on this host: `opencode:D4` is `N/A (definitional)`
— its SQLite-only store has no JSONL boundary the row's mechanism can apply
to, regardless of fix — excluded from the required row count (215 of 216
required going forward); and a row for an optional agent that cannot
complete because of a host credential is recorded `NOT TESTED (host
credential)` and does not block the verdict when its condition holds. At
the `v0.6.0-rc.3` tagged run, zero rows qualified for that second
disposition — the `qwen` gap that run found was a version-compatibility
block, not a credential gap, so it does not match the disposition's own
text regardless of whether `grok`'s same row passed (report §0.9, §20).
Full rules in
[`docs/testing/v0.6.0-windows-acceptance.md`](../../testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict)
and [ADR 0005's amendment](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md#amendment-2026-09-07).

**Q18, Q20, Q22, and Q23 were answered by you on 2026-09-07**: you refreshed
the Qwen Code login, re-authenticated Cline, and confirmed no account or
xAI-side change behind the Grok stall; you accepted the UAC prompt for the
`v0.6.0-rc.3` tagged run's `MatrixH:H7` (that round trip's mechanism passed,
though its incidental live-store write is the `#424` incident above, not a
row defect). Their update notes below record what each answer's tagged run
then showed. **Q21 remains open** — `cursor-agent` is still broken on the
acceptance host (`Error: Cannot find module 'tree-sitter'`); not needed,
since `cursor:C2`/`cursor:C3` run `rein`-only against the real store instead.
**Q24 is done** — the two synthetic/incidental OpenCode session rows the
`v0.6.0-rc.3` run's harness incidents left in the live `opencode.db` were
removed by you after that report was committed. **Q25 is new**: the
`MatrixH:H7` lab-isolation rule this candidate's dispatch adds (a fresh lab
account, before/after listing digests of the live agent roots, and a
go-ahead file poll for the UAC prompt) and issue #424 itself, for you to
confirm.

What is left is yours, in order: confirm the `MatrixH:H7` lab-isolation
method (Q25) and whether/when to fix `#424` itself (a separate, smaller
ticket, not part of this candidate); decide whether to reinstall/upgrade the
broken `cursor-agent` install (Q21) — still not required; accept or reject
the disposition rules (Q19, unchanged since the `v0.6.0-rc.2` update); merge
the `v0.6.0-rc.4` release commit; sign and push the `v0.6.0-rc.4` tag (Q5);
then run the tagged dispatch
([`docs/testing/v0.6.0-rc.4-agent-verification-prompts.md`](../../testing/v0.6.0-rc.4-agent-verification-prompts.md))
against native Windows x64, or tell me to.

---

## Q1 — Is `v0.6.0` "Hop plus the interactive CLI"?

**Assumed: yes.** The tree that exists is `hop/main` (the Hop client,
OpenCode T5, Kimi T2) and `main` (`v0.5.2-rc.1`). Nothing of Phase 6A
(universal configuration) is built, so `v0.6.0` is Phase 6C shipping first and
6A moves to `v0.7.0`. If instead you meant `v0.6.0` to be universal
configuration, the Hop release becomes `v0.6.0` anyway and configuration is
`v0.7.0` — the same work, different roadmap wording.

## Q2 — Ship the Hop client before the hosted service is open?

**Assumed: yes**, with every surface saying the service is not open
(ADR 0005 D1). The alternative — stripping the commands behind a build tag —
contradicts hosted ADR 0007 and touches every Hop file. Consequence for
users: `rein login` against the default URL fails with a one-line message
pointing at the docs until `hopd` is deployed. If you would rather the stable
wait for staging or production deployment (hosted #6, founder procurement
items in `reinstate-hosted/docs/procurement.md`), say so and `v0.6.0` stops
at the candidate.

## Q3 — Widen the verified ranges on Windows evidence alone?

**Assumed: yes** (ADR 0005 D3), because this host runs Claude Code `2.1.261`
and OpenCode `1.18.27`, both outside the verified ranges, and I will not
downgrade your working tools. The widened part is marked "native Windows
evidence; macOS pending" everywhere it appears. The alternative is to run the
resume rows with `--allow-untested`, which makes them vacuous.

## Q4 — No stable `v0.5.2`?

**Assumed: correct.** `v0.5.2-rc.1`'s content is certified inside `v0.6.0`
and #366 is re-pointed at the `v0.6.0` Windows run. If you want a stable
`v0.5.2` for the package registries first, it needs its own Windows run of the
22 rows on the tagged `v0.5.2-rc.1` artifact; that is about a day and I can
do it after `v0.6.0-rc.1` is cut.

## Q5 — Tag signing: only you can do it

The release signing key (`.github/allowed_signers`, key `…OKWvFz02…`) is not
on this machine; `~/.ssh/id_ed25519` here is a different key. I will not
copy a signing key anywhere. When the candidate PR is green and the pre-tag
Windows report is committed, these are your steps, from a clean `main`
checkout after merging the PR:

```bash
git fetch origin && git checkout main && git pull --ff-only
export REINSTATE_SIGNING_KEY="$HOME/.ssh/reinstate_release_signing"
git -c gpg.format=ssh -c user.signingkey="$REINSTATE_SIGNING_KEY" \
  tag -s v0.6.0-rc.1 -m "Reinstate v0.6.0-rc.1"
git push origin v0.6.0-rc.1
```

Then let the release workflow publish the draft, verify assets and
attestations per `RELEASING.md` step 3, publish it as a **prerelease**, and
tell me. The tagged-artifact Windows run (W7b) starts from there. The same
shape applies to `v0.6.0` stable and the `website-vYYYY.MM.DD.N` tag.

The merge is yours too, for one more reason: it is the largest branch this
repository has ever taken. The repository's convention is a squash merge
(`.gitleaks.toml` relies on it), which folds the Hop client's ticket-by-ticket
history into one commit on `main`; `hop/main` and `release/v0.6.0-rc.1` keep
that history on the remote either way. If you would rather keep it on `main`,
use a merge commit instead. PR #404 is marked ready for review with CI green;
merging it, then signing the tag, are the two steps that remain before the
tagged Windows run.

## Q6 — The ConPTY host state (#367)

The acceptance host's pseudo-console subsystem broke on 2026-08-23 and a
reboot "has not been tried". This machine is that host. W4 probes it first;
if it is still broken, a reboot is the cheapest fix and only you can do it
safely — please reboot when convenient and say so. Until then W4 builds a Go
ConPTY driver, which is needed anyway for Grok GD8 on Windows (#368) and for
keystroke injection the scheduled-task workaround cannot do.

## Q7 — The hosted `LAUNCH-PLAN.md` says "merge and release after production"

Your launch plan (assessed earlier today) sequences the release after
production deployment and Windows↔macOS parity. Your instruction today
sequences `v0.6.0` first. I follow today's instruction and keep the launch
plan's later phases (billing wiring, pricing page, legal, production) out of
`v0.6.0`. Nothing in `v0.6.0` mentions price, trial, or sign-up.

## Q8 — Dependabot PRs (#373–#402, twenty open)

Not touched. Go dependency bumps (aws-sdk-go-v2, smithy-go, age, modernc
sqlite) would change the release's module graph mid-flight. Recommend merging
them after `v0.6.0` stable, in one pass, with the suite re-run. Say if you
want the SQLite driver bump (`modernc.org/sqlite` 1.56 → 1.58) in `v0.6.0`;
the `v0.5.1` precedent re-ran the storage rows for exactly that change.

## Q9 — Stray local edit

`assets/logo/concepts/14-diptych-bridge.svg` is modified in the shared
checkout and not committed. Nobody in this plan touches it. Commit or discard
it when you are back.

## Q11 — Your machine had lab variables in its user environment — resolved

`REINSTATE_BACKEND=memory` and `REINSTATE_MEMORY_BACKEND_DIR` were set as
persistent user-level variables and silently redirected every `rein` command
to a local store. With your approval on 2026-09-06 I removed those two.
`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, and `XDG_DATA_HOME` (all under
`D:\Projects\hop-10-lab\`) are your live agent homes, so I left them; the
acceptance rules now say to isolate with a run's own value rather than unset
them.

## Q12 — Two rows need you at the keyboard

- **H7, the daemon's Task Scheduler round trip.** `rein daemon install`
  registers a scheduled task, which needs an elevated shell; no agent session
  here can elevate. The foreground loop (push on change, scheduled pull, pull
  before resume) passed. When convenient, from an elevated PowerShell in a
  throwaway home: `rein daemon install`, `status`, `stop`, `start`,
  `uninstall`, and paste the output into the run notes, or tell me and I will
  drive it while you hold the elevation prompt.
- **ConPTY (#367)** allocates again on this host: W4's driver drove the
  interactive switcher and a Claude Code resume, so the reboot happened or the
  state cleared. I will close #367 with that evidence when the PR opens.

## Q14 — This host could not list its own processes — resolved

During the pre-tag passes `Get-CimInstance Win32_Process` and `tasklist`
failed with "Critical error", which is how the active-session defect was
found and fixed. With your approval I ran `winmgmt /verifyrepository`
elevated (you accepted the prompt): the repository is consistent, no salvage
was needed, and non-elevated enumeration works again. The E5 rows run on the
tagged artifact.

## Q15 — Claude Code auto-updates faster than the ceiling moves

Claude Code went `2.1.238` → `2.1.261` → `2.1.263` on this host within two
days, and each step refused every Claude resume row by design. I widened the
ceiling twice with real resume evidence, and the tagged run will likely meet
a newer build again. Options for you: (a) pin the host during acceptance
(`DISABLE_AUTOUPDATER=1` in the environment Claude Code starts with), (b)
accept a patch series (`2.1.x` up to a stated ceiling) as verified rather
than an exact build, which is a policy change to the fail-closed rule in
`docs/compatibility.md`, or (c) keep widening per candidate as `RELEASING.md`
already anticipates. I proceed with (c).

## Q16 — Claude Code's sign-in failed for spawned runs — resolved

The spawned `claude -p …` runs failed with "OAuth session expired and could
not be refreshed" because the acceptance rules told every run to unset
`CLAUDE_CONFIG_DIR`, so they fell back to a stale `~/.claude` instead of your
live configuration at `D:\Projects\hop-10-lab\claude`. With the live
directory a spawned run authenticates and answers (verified 2026-09-06). The
rules are corrected; the Claude resume, fork, and truncation rows (`E2`,
`E3`, `D4`) are collectable on the tagged artifact.

## Q17 — The release signing key is gone; a replacement is ready but needs your hands

The old key existed only on the MacBook, so the tags it signed stay valid
and nothing else can be signed with it. On 2026-09-06 I generated a
replacement on this host: `C:\Users\admin\.ssh\reinstate_release_signing`
(ed25519, no passphrase, public key
`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINteNJhvsI+dxEOVj+Lnc4jD…`, comment
`reinstate-release-signing-2026-09-06`). The permission classifier stops me
from committing a change to `.github/allowed_signers`, which is right for a
trust anchor, so three steps are yours:

1. Add the key to the allowed signers and merge it (the branch
   `chore/rotate-release-signing-key-2026-09` in the worktree
   `D:\Projects\reinstate-worktrees\rotate-key` already holds the edited
   file, old key kept so earlier tags still verify):
   `! cd D:\Projects\reinstate-worktrees\rotate-key && git add .github/allowed_signers && git commit -m "chore(release): add a replacement SSH signing key" && git push -u origin chore/rotate-release-signing-key-2026-09 && gh pr create --base main --fill && gh pr merge --squash`
2. Register the public key on your GitHub account as a **signing** key so
   the tag shows "Verified" (Settings → SSH and GPG keys → New SSH key → type
   Signing), or `! gh auth refresh -h github.com -s admin:ssh_signing_key`
   then `! gh ssh-key add C:\Users\admin\.ssh\reinstate_release_signing.pub --type signing`.
3. Back the private key up to your password manager; this host is its only
   copy. Consider a hardware-backed key later.

Then I sign and push `v0.6.0-rc.1` (or you run the two commands from Q5).

## Q18 — Refresh the Qwen Code login on the acceptance host

Non-interactive `qwen -p` on the acceptance host returns `[API Error: 401
invalid access token or token expired]`. Qwen Code's OAuth token there is
expired, and refreshing it requires an interactive browser login — nothing
I can do headlessly. Without a refresh, `qwen:E1`/`E2`/`E3`/`E5` cannot
complete this candidate's tagged run and are recorded `NOT TESTED (host
credential)` (Q19's disposition rule), which does not block the verdict
since Qwen Code is an optional agent, but does mean those four rows stay
untested rather than passing outright. Please sign into Qwen Code
interactively on the acceptance host before, or during, the `v0.6.0-rc.3`
tagged run, if you want those rows to actually run.

**Update (`v0.6.0-rc.2` tagged run, 2026-09-07):** still unrefreshed. All
four rows were recorded `NOT TESTED (host credential)`; the disposition's
second condition (another T4 agent passing the same row) held only for
`qwen:E5` (`grok:E5` was `PASS`) — `grok:E1`/`E2`/`E3` did not themselves
pass this run (Q22), so `qwen:E1`/`E2`/`E3` stayed blocking despite the
rule existing. Still yours to refresh whenever convenient.

**Answered (2026-09-07):** you refreshed the Qwen Code login interactively
before the `v0.6.0-rc.3` tagged run. That run's own probe (`qwen -p "Reply
PONG"`-style) and its planted-token `C3` session both completed
successfully against the live default home — the credential is no longer
the blocker. What the refreshed login exposed instead: the host's real
Qwen Code had also self-updated to `0.23.0`, above the in-tree verified
ceiling `0.21.12`–`0.21.13`, so every launch-plan-building row (`D1`–`D5`,
`E1`/`E2`/`E3`/`E5`/`E6`) refused with `exit 5` (`agent.version` block, not
a credential failure) instead of running. Zero rows qualified for the
`NOT TESTED (host credential)` disposition this run — a row that does not
match the disposition's own definition does not get its exemption,
regardless of `grok`'s own results. `v0.6.0-rc.4` widens the verified range
to `0.21.12`–`0.23.0` (see
[`docs/testing/results/2026-09-07-windows-range-widening-qwen-v060.md`](../../testing/results/2026-09-07-windows-range-widening-qwen-v060.md)),
which is expected to let all ten rows run for real against the same
refreshed login.

## Q19 — Two disposition rules adopted without your sign-off, to unblock the stable gate

As written, the required-row rule in ADR 0005 D2 makes a `PASS` device
verdict impossible on this host: `opencode:D4`'s mechanism (a byte-exact
JSONL truncation boundary) cannot exist for OpenCode's SQLite-only store,
regardless of any fix, and `qwen:E1`/`E2`/`E3`/`E5` cannot complete while
the host's Qwen Code credential is expired (Q18) and nobody but you can
refresh it. Neither gap is a product defect or something an executor can
close. I adopted two narrow dispositions autonomously, recorded as an
[amendment to ADR 0005](../../adr/0005-v0.6.0-scope-and-windows-first-acceptance.md#amendment-2026-09-07)
and detailed in
[`docs/testing/v0.6.0-windows-acceptance.md`](../../testing/v0.6.0-windows-acceptance.md#dispositions-that-do-not-block-the-device-verdict):
a row whose mechanism cannot exist for an agent's store layout is `N/A
(definitional)` and excluded from the required count (today, only
`opencode:D4`); and a row for an optional agent that cannot complete for
lack of a host credential is `NOT TESTED (host credential)` and does not
block the verdict, provided every required agent and at least one other
optional agent at the same tier pass the same row. **You may reject
either or both.** Rejecting (b) means the stable gate waits for the Qwen
Code login (Q18) to actually complete those four rows. Rejecting (a) means
`opencode:D4` reverts to `NOT TESTED (definitional)`, which never passes a
required row, so stable then waits for you to drop that row from the
required set explicitly — no fix makes a JSONL boundary exist in a store
that has none.

**Update (`v0.6.0-rc.2` tagged run, 2026-09-07):** both rules were applied
mechanically to a real run for the first time (report §0.7, §12).
`opencode:D4` stayed `N/A (definitional)`, unchanged. The `qwen` rule's
second condition depends on `grok` passing the same row each time, which is
not guaranteed run to run — this run it held for `qwen:E5` only, not
`qwen:E1`/`E2`/`E3` (Q18's update, above). The rule itself is unchanged and
still awaits your acceptance or rejection; what changed is evidence that it
behaves as written, including the case where it does not excuse a row.

## Q20 — Cline account needs re-authentication

Non-interactive `cline` on the acceptance host returns `Unauthorized: Please
make sure you're using the latest version of Cline and re-authenticate your
Cline account`, even against the live Cline configuration
(`CLINE_CONFIG_DIR` unset, the host's own real Cline install). Nothing I can
do headlessly re-authenticates a Cline account — it needs an interactive
sign-in. Without it, `cline:C3`'s real-vendor-session evidence stays
`NOT TESTED (host credential)` on the `v0.6.0-rc.3` tagged run (excused, per
the contract's disposition rule, once any *one* other optional T1 agent —
`copilot:C3`, `cursor:C3`, or `pi:C3` — `PASS`es the same run; it is not a
conjunction of two named agents. This candidate's Cursor store fix is
expected to flip `cursor:C3` to `PASS`, which alone would excuse this row).
Please sign into Cline interactively on the acceptance host before, or
during, that run if you want the row to run for real regardless.

**Answered (2026-09-07):** you re-authenticated Cline interactively before
the `v0.6.0-rc.3` tagged run. Cline answered live and a fresh
planted-token session was created and found by `rein search` — `cline:C3`
`PASS`ed outright this run rather than falling back to the host-credential
disposition. No further action needed; carried forward as `PASS` on `v0.6.0-rc.4`'s
dispatch unless something regresses.

## Q21 — `cursor-agent` is broken on this host

`cursor-agent` `2026.08.11` fails immediately with `Error: Cannot find
module 'tree-sitter'` from its own bundled `index.js`; the installed
versions directory under this host's Cursor CLI installation has no
`node_modules` at all, so no fresh real Cursor CLI session can be created
here. `v0.6.0-rc.3`'s `cursor:C2`/`cursor:C3` rows work around this by
reading the host's real `~/.cursor` `store.db` through `rein` alone,
read-only, under the T1-discovery exception — that evidence does not need a
working `cursor-agent`. Reinstalling or upgrading `cursor-agent` on this
host is a separate decision I have not made for you: it touches a real
vendor CLI install outside this repository, and I do not know whether you
use this host's Cursor CLI for anything else that a reinstall could
disturb. Tell me if you want it reinstalled, and whether before or after
the `v0.6.0-rc.3` tagged run.

**Still open (2026-09-07):** `cursor-agent` remains broken on this host; you
have not asked for a reinstall. Not needed — `cursor:C2`/`cursor:C3` ran
`rein`-only, read-only against the real store at both the `v0.6.0-rc.3`
tagged run (`PASS`) and are expected to again on `v0.6.0-rc.4`. Still yours
whenever you want it fixed for other reasons.

## Q22 — Grok Build started stalling on every prompt on this host

Two isolated-home probes (`GROK_HOME` seeded only with `auth.json`, stdin
closed, no shared MCP servers) with `grok -p` never produced a completed
reply within 150 s or 400 s respectively. The session's own
`logs/unified.jsonl` shows the prompt queued (`shell.prompt.queued`) for
minutes before `shell.handle_prompt.start` fires, then nothing further in
the budget. `grok` passed all four `E`-row journeys (`E1`/`E2`/`E3`/`E5`) on
this same host at `v0.6.0-rc.1` (2026-09-06) with no such delay. Nothing
about this host's Grok Build install, `PATH`, or credential changed between
that run and this one that I made or can see. Do you know of an account
change, an xAI-side rate limit or maintenance window, or anything else on
your end that could explain a new multi-minute stall between the CLI
queuing a prompt and starting to handle it? Without an answer, `v0.6.0-rc.3`
retries with a longer per-turn budget and records whatever stall evidence
it finds; it does not otherwise change the Grok Build reader or client.

**Answered (2026-09-07):** you confirmed no account or xAI-side change on
your end. The `v0.6.0-rc.3` tagged run's retry (stdin closed, isolated
`GROK_HOME`, ten-minute budget) completed `grok:E1`/`E2`/`E3` — the same
symptom family as before (a completed reply after roughly eleven minutes),
just inside the longer budget this time, confirming the isolated-`GROK_HOME`
retry method rather than an account or platform fix. No further action
needed on your end; `v0.6.0-rc.4`'s dispatch keeps the same retry method.

## Q23 — `MatrixH:H7` needs you at the keyboard for one UAC prompt

The daemon's Task Scheduler round trip (`rein daemon install`) needs an
elevated shell; the `v0.6.0-rc.2` tagged run declined the elevation prompt
twice with nobody available to accept it (the identical mechanism passed in
full on this same host at `v0.6.0-rc.1`, so this is availability of an
acceptor, not a regression). Please give me a time window when you can sit
at this machine for a few minutes to accept one UAC prompt during the
`v0.6.0-rc.3` tagged run, or tell me to keep recording the row `PARTIAL`
with that reason until you are available.

**Answered (2026-09-07):** you gave a time window and accepted the UAC
prompt during the `v0.6.0-rc.3` tagged run. `MatrixH:H7`'s own mechanism
completed in full and was scored `PASS` (report §18). That same round
trip's elevated `daemon stop`/`daemon start` half is what wrote synthetic
content into this host's real, live agent stores — the Task Scheduler task
`rein daemon install` created pins only `--home`, not the agent-root
environment variables, so it silently followed the login environment
instead of the isolated device home once running under Task Scheduler
rather than the shell `install` ran in. That is not something accepting the
UAC prompt could have prevented; it is filed as
[issue #424](https://github.com/HarjjotSinghh/reinstate/issues/424) (see
Q25). No further action needed from you on the UAC prompt itself;
`v0.6.0-rc.4`'s dispatch adds a lab-isolation method (Q25) so the row's
mechanism can be re-verified without the same incident recurring, and still
needs you at the keyboard for the same one prompt.

## Q24 — Two stray OpenCode session rows the `v0.6.0-rc.3` harness incidents left in the live store

The same incidents behind Q25/`#424` — the `H7` daemon pull and the T5
push/pull round trip's unisolated device-B pull — wrote two rows into this
host's real, live `opencode.db` alongside the files the report's own
cleanup already removed: the synthetic `ses_fixture001a` (the `H7` daemon
pull) and one throwaway T5 session an executor created directly against
that database. The live store was compared read-only against a pristine
pre-incident backup by session id and holds every backup session plus
exactly those two additions — nothing from the backup is missing, so the
store did not need rolling back, only those two rows removing (report
§0.11(c)). I do not have write access to your live OpenCode store, so I
could not remove them myself.

**Answered (2026-09-07):** you removed both rows from the live
`opencode.db` yourself after the `v0.6.0-rc.3` report was committed. No
further action needed.

## Q25 — `MatrixH:H7` lab isolation and issue #424

The `v0.6.0-rc.3` tagged run's `H7` incident (Q23's answer, above;
[`docs/testing/results/2026-09-07-windows-v060rc3.md`](../../testing/results/2026-09-07-windows-v060rc3.md)
§16, §21) showed that `rein daemon install`'s Task Scheduler task
definition pins `--home` but not `CLAUDE_CONFIG_DIR`/`CODEX_HOME`/
`XDG_DATA_HOME`, so a daemon Task Scheduler restarts (a `daemon stop`/
`daemon start`, or an ordinary login-triggered relaunch) silently follows
whichever agent-root environment variables are live in the process's
ambient environment at that moment — the operator's real login environment
for a real end-user, which is arguably correct, but it is exactly what
turned this lab's intended-isolated `H7` test into a write into this host's
real Claude, Codex, and OpenCode stores. I filed
[issue #424](https://github.com/HarjjotSinghh/reinstate/issues/424) to fix
this properly (record the agent-root variables — or resolved roots — at
install time, show them in `rein daemon status`, and refuse to start when
the resolved roots differ unless `--allow-root-change` is given); this
candidate does not attempt that fix.

In the meantime, `v0.6.0-rc.4`'s dispatch
([`docs/testing/v0.6.0-rc.4-agent-verification-prompts.md`](../../testing/v0.6.0-rc.4-agent-verification-prompts.md),
"Lab isolation for `MatrixH:H7`") adds a run-method guard so the row can be
re-verified without depending on `#424` landing first: run `H7` from a
fresh lab account with zero pushed sessions (so the daemon's pull, even if
it does follow the login environment, is a no-op with nothing synthetic to
write); record a sorted-listing SHA-256 digest (paths and sizes) of that
account's live Claude, Codex, and OpenCode roots before and after the round
trip, and require them identical; and poll
`D:/ReinstateAcceptanceProjects/h7-go.txt` for your go-ahead before
attempting the elevation, so the row still waits for you at the keyboard
for the one UAC prompt rather than attempting to route around it. Please
confirm: (a) this method is acceptable for re-verifying `H7` on
`v0.6.0-rc.4`, and (b) when you would like `#424` itself scheduled — it is
a small, separable fix and does not need to block this candidate's tagged
run.

**Wording corrected (2026-09-07), after the `v0.6.0-rc.4` tagged run.**
"Fresh lab account" was ambiguous and that run's executor read it as a
fresh **Windows** user account — which needs administrator rights to
create — could not set one up, and abandoned the row before its own UAC
prompt was even reached, polling the go-signal file for only a few minutes
instead of the intended long wait. The rule always meant a fresh **Hop**
account (created through the lab's `hopd`, via `hoplab pair init` or the
account init flow) with zero pushed sessions — not a new Windows login —
needing no elevation beyond the row's own single UAC prompt. The corrected
rule, plus the precise go-signal poll cadence (every 60 s, for up to 150
minutes, starting only after every other row in the executor's part is
done), is now recorded as the contract text in
[`docs/testing/v0.6.0-windows-acceptance.md`](../../testing/v0.6.0-windows-acceptance.md)
(Run notes) rather than left to each candidate's dispatch to restate. See
Q26 and Q27 below.

## Q26 — `MatrixH:H7` needs you at the keyboard during the `v0.6.0-rc.5` run

`H7`'s UAC prompt cannot be accepted unattended (Q23), and the corrected
lab-isolation rule (Q25, above) still routes through that same single
elevation. Beyond accepting the prompt itself, `H7` now depends on the
operator go-signal file
(`D:/ReinstateAcceptanceProjects/h7-go.txt`, polled every 60 s for up to
150 minutes, starting only once every other row in the executor's part is
done) so the row does not attempt the elevation before you are actually
available to accept it. Please confirm you (or another maintainer with
admin rights on the acceptance host) will be reachable near the keyboard
for a window inside the `v0.6.0-rc.5` run so this go-signal mechanism has
someone to signal for; if no maintainer is reachable, the row records
`PARTIAL` (operator/harness availability) after the full 150-minute wait,
the same disposition `H7` has carried on more than one prior run.

## Q27 — Should the acceptance lab pin vendor CLI versions for a run?

Vendor self-updates keep moving the compatible-range ceiling mid-run,
independent of anything this candidate changes: this cycle alone saw
Claude Code self-update to `2.1.263`, Qwen Code to `0.23.0`, and OpenCode
toward `1.18.29` (`website/src/data/compatibility.json` currently verifies
OpenCode only through `1.18.27`) — each discovered because a real vendor
binary on the acceptance host updated itself between runs, not because the
lab requested a newer version. Widening the verified range each time is
real evidence, not padding, but it also means a run's outcome can depend on
exactly when a vendor happened to auto-update relative to when the row ran,
which makes two runs against the "same" dispatch not strictly comparable.
Please decide: (a) pin vendor CLI versions for the duration of a single
acceptance run (disabling or deferring vendor auto-update on the lab
account for that window) so a run's evidence is tied to one known version
per agent, or (b) keep the current practice of widening the verified range
on whatever version the host's vendor binary has already self-updated to,
treating each new ceiling as evidence rather than drift. Either is
workable; this only needs your call before it becomes an inconsistency
between reports.

## Q13 — GitGuardian on the candidate PR

GitGuardian's check on #404 flags the synthetic keyring goldens the Hop tree
carries (two test age identities, the pairing-protocol golden keys, the
fixture keyring's account key). They are fixtures that protect nothing;
gitleaks now allowlists each literal by path in `.gitleaks.toml` and its
check is green. GitGuardian is configured only in its dashboard, which I
cannot reach: mark those incidents as false positives there, or tell me
the check is not required for merge.

## Q10 — Commit identity

The Hop-era commits on this host are authored `Harjot Singh Rana
<HarjjotSinghh@gmail.com>` (the git config here) while the private repo's
STATUS.md asks for the `noreply` identity and the `v0.5.2-rc.1` tag used it.
Executors use the configured identity plus the required trailers. Tell me if
the public repo should switch.

---

## Executor questions

_(Relayed by the coordinator from executor reports, newest first. Each
entry: date, workstream, the question, and the decision taken.)_

- **2026-09-06, W4.** The OS keyring held one device token per host, so two
  Reinstate homes on one machine (the lab's "device A" and "device B")
  overwrote each other's sign-in, and a fresh home could inherit a token
  pointing at another control plane. **Decided:** fixed in the product
  (`internal/credentials`, commit 2521485f): a home selected with
  `REINSTATE_HOME` gets its own keyring entry; the default home is unchanged.
  The lab's keyring save/load swap becomes unnecessary.
- **2026-09-06, W4.** Should `hoplab pair` drive `rein account recover` or
  the live `rein account join` + `rein devices approve` flow? **Decided:**
  both, as separate subcommands; the H6 row needs join/approve, H5 needs
  recover.
- **2026-09-06, W2.** `internal/preflight` observers each default their own
  sub-timeout to 2 s regardless of the shared `Options.Timeout`; two rounds
  of test flakes came from that. Should `Verify` propagate the shared budget
  to unset sub-timeouts? **Decided:** not in `v0.6.0`; production callers all
  use the defaults, so there is no live gap. Worth a follow-up card after
  stable.
- **2026-09-06, W5.** Bumping `product.currentRelease` for a candidate forces
  `CITATION.cff` and the changelog heading to move with it (release-truth
  guards). **Decided:** the coordinator moves `CITATION.cff` on the release
  branch (matching the `v0.5.2-rc.1` precedent) and W1's changelog heading
  satisfies the other guard; W8 sets the dates.
- **2026-09-06, W2.** Two `internal/workspace` tests skip on Windows because
  symlink creation is unreliable there; symlinked-workspace probing is
  therefore untested on the Windows host. **Decided:** carried as a known
  gap in the Windows contract's run notes; not a `v0.6.0` change.
