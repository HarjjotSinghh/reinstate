# Clarifications for the maintainer — v0.6.0

Questions the plan could not settle from the request, the code, or the
records. Each one names the assumption the work proceeds under, so nothing is
blocked on an answer. Overturning an answer before stable promotion is cheap;
after it, it is a `v0.6.1`.

Written 2026-09-05; status lines updated by the coordinator. Executors
append to the last section only.

**Where things stand (2026-09-06).** Candidate PR
[#404](https://github.com/HarjjotSinghh/reinstate/pull/404) (draft) carries
`release/v0.6.0-rc.1`; the deferred-macOS rows are
[#403](https://github.com/HarjjotSinghh/reinstate/issues/403). Every Go,
race, lint, vuln, and doc gate is green on the branch; the pre-tag Windows
matrix (W7) runs next and its report lands on the same branch. After that
the only steps left before a prerelease are yours: Q5 (sign the tag) and, if
you want them closed, Q12 (the elevated-shell daemon row) and Q11 (the host
variables).

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

If you would rather I merge the candidate PR myself before you tag, say so;
otherwise I leave the merge to you as well, since it is the step right before
your signature.

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

## Q11 — Your machine has lab variables in its user environment

`REINSTATE_BACKEND=memory` and `REINSTATE_MEMORY_BACKEND_DIR=D:\Projects\hop-10-lab\locker`
are set as persistent user-level environment variables on this host (and
`XDG_DATA_HOME` may be too). They silently redirect every `rein` command,
including the Hop journeys, to a local store, which is what made the first
lab pairing look broken. Every agent now unsets them per shell and the lab
strips them for the processes it launches. I have not changed your machine's
environment; if the variables are no longer needed, remove them with
`[Environment]::SetEnvironmentVariable('REINSTATE_BACKEND', $null, 'User')`
and the same for the other two.

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
