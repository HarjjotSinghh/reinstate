# W4 — Windows lab harness

**Executor:** one Sonnet agent. **Verifier:** one Sonnet agent (run the
harness cold from the README alone; anything it has to guess is a defect).
**Branch:** `v060/w4-harness` from `release/v0.6.0-rc.1`.

The private control plane is at `D:\Projects\reinstate-hosted` (Go module,
`cmd/hopd`; `go build ./cmd/hopd` works on this host with `CGO_ENABLED=0`).
Its README, `deploy/RUNBOOK.md`, and `docs/` describe `hopd`'s environment.
Never commit anything from that repo into this one; refer to it by path
through an environment variable.

## T-401 — `scripts/testing/hoplab`

A Go program (so it works from PowerShell 5.1 and Git Bash alike) plus a
one-line `.ps1` and `.sh` wrapper that:

1. Builds or locates `hopd` (`REINSTATE_HOPD_BIN`, else `go build` from
   `REINSTATE_HOSTED_DIR`, default `D:\Projects\reinstate-hosted`).
2. Starts `scripts/testing/fakelocker` on `127.0.0.1:9002` and `hopd` on
   `127.0.0.1:8082` with `HOPD_STORAGE=fake`, `HOPD_EMAIL_SENDER=log`,
   `HOPD_BASE_URL=http://127.0.0.1:8082`, `HOPD_S3_ENDPOINT=http://127.0.0.1:9002`,
   and a fresh `hopd.db` under a lab root the caller names (never under a Git
   checkout).
3. Tails `hopd`'s log for the sign-in emails and exposes an **approver**: given
   the address `rein login --email` used, it follows the emailed link the way
   the 2026-08-24 lab's approver did (read
   `docs/testing/results/2026-08-24-first-push-acceptance-lab.md` and the
   private repo's sign-in handler to get the confirm step right; the link is
   confirmed by POST, not by GET). A `--refuse` mode exercises the refused
   sign-in path.
4. Prints the environment block a client shell needs
   (`REINSTATE_HOP_URL=http://127.0.0.1:8082` and the isolated homes from
   T-402), and stops both processes on `Ctrl+C` or `hoplab stop`.

Read `internal/cli/hop_first_push_acceptance_test.go` first: `hoplab` should
make the `-tags hopacceptance` suites runnable with `HOP_STAGING_URL` pointed
at the local `hopd`, and `keygeneration_crossplane_test.go` runnable with
`REINSTATE_HOPD_BIN`.

## T-402 — Two homes on one host

Document and script (in `hoplab`) two isolated device identities on this
machine: separate Reinstate homes, separate OS-keyring service names or an
isolated keyring backend if the code offers one (find how the CLI journeys
isolate the device token; reuse that), separate `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, `XDG_DATA_HOME` roots seeded from `testdata/` fixtures or
`scripts/tuisandbox`. Pairing, revocation, the lagging device, and path
remap all need "device A" and "device B" to be distinguishable and to hold
different project paths.

## T-403 — ConPTY

First, probe #367 on this host: run any interactive program under a
pseudo-console and confirm it executes (the issue's `probe` write test). Record
the result either way in the results section of
`docs/testing/windows-acceptance-host.md`.

Then build `scripts/testing/conptydriver` (Go, `golang.org/x/sys/windows`
`CreatePseudoConsole`; no new module dependency): runs a command under a
pseudo-console of a given size, answers the Bubble Tea startup queries
(`ESC ] 11 ; ?` background colour, `ESC [ 6n` cursor position), executes a
step script (`wait /regex/ 10s`, `send "text"`, `key enter|esc|tab|ctrl+k|up|down|space|f`,
`snapshot path`, `sleep 500ms`, `kill`), and writes the raw stream plus a
per-snapshot rendering through a real VT parser (not a regex strip — conhost
rewrites spaces as cursor-forward moves, see the `v0.5.2` contract). It is
the Windows twin of `scripts/testing/vendor-tty-driver.py`. Prove it by
driving `scripts/tuisandbox`'s bare `rein` to the switcher and snapshotting
one frame, and by driving a Claude Code `--resume` to completion in a
throwaway project (needed by W3 and the CLI matrix).

## T-404 — Document the bench

`docs/testing/windows-acceptance-host.md`: a "Hop lab" section (how to start,
the env block, the approver, the two homes) and a "ConPTY driver" section
(usage, the step-script grammar, the two traps). Put the `hoplab` and
`conptydriver` usage in their own `README.md` files too.

## Done when

The verifier ran `hoplab` cold, signed in twice, paired the two homes, and
snapshotted one switcher frame through `conptydriver`, all from the docs.
