# W6 — Hop parity journeys on native Windows (hosted #16)

**Executors:** two Sonnet agents (A: H1–H5 and H9–H11; B: H6–H8), each in
its own worktree and its own lab root. **Verifier:** one Sonnet agent
(check the harness before the product: every PASS row must have exercised
the mechanism it names).
**Branch:** `v060/w6-hop-parity` (results doc and any fixture); defects go to
the coordinator as fix cards.
**Contract:** [Windows acceptance, section D](../../../testing/v0.6.0-windows-acceptance.md#d--hop-parity-journeys-required).
**Prior records to read:** `2026-08-24-first-push-windows.md`,
`2026-08-24-pairing-macos-windows.md`, `2026-08-24-daemon.md`,
`2026-08-27-sync-verify-windows-round-three.md`,
`2026-08-27-key-generation-floor-crossplane.md`.

## Ground rules

- Use W4's `hoplab` and two homes. Fresh `hopd.db` per journey group.
- Real vendor binaries for session creation and for the `--dry-run` launch
  plan; a real resume for at least one agent per journey group through the
  ConPTY driver.
- Sessions live in throwaway projects under `D:\ReinstateAcceptanceProjects\`.
  Never the developer's own trees.
- Every command in the report is real; only the recovery code, the device
  token, and the lab project path are redacted.
- A row whose mechanism was not exercised (a keyring that never rolled, a
  revoke with no lagging device) is `FAIL`, not `PASS`.

## T-601 — H1–H5 (executor A)

Sign-in by email with the approver (and one refused sign-in, H1b);
`rein init --hop` provisions the locker exactly once (H2); `rein account init`
shows the recovery code once and writes `keyring.v1.json` (H3); first push
of one Claude Code, one Codex, one OpenCode session, `first_push` reported to
the control plane exactly once, `rein hop status` shows it (H4); wipe the
home, the agent stores, the token, and the device key; sign in as a new
device; `rein account recover`; `rein pull --all`; verified `rein resume
--dry-run` for all three and one real resume (H5).

## T-602 — H6–H8 (executor B)

Pairing: device B `rein account join`, device A `rein devices approve`,
B pulls (H6); an expired request is refused or rolled back (H6b). Daemon:
`rein daemon install|status|stop|start|uninstall` round trip through the
real Task Scheduler from an elevated shell if the host allows, and the
foreground loop's push-on-change and scheduled pull (H7). Revocation: A
revokes B; the keyring rolls a generation; B's token is refused; the
lagging-device attack in `keygeneration_crossplane_test.go` run with
`-tags hopacceptance` and `REINSTATE_HOPD_BIN` against the local `hopd`
(H8); the Console-initiated pending request completes only after the
recovery-code command (H8b).

## T-603 — H9–H11 (executor A)

`rein sync verify` human and `--json` reports name only observed objects,
and the 404-floor proxy journey reproduces the documented residual (H9);
`rein sync migrate --to byo` to a second `fakelocker` bucket, `--switch`,
`--forget-hop` (H10); path remap: device A's project at one Windows path,
device B's mapping at another, pull rewrites session paths and the resumed
session's workspace verification passes (H11; the macOS half is deferred).

## T-604 — Defects

Each Windows-only defect: reproduce, minimal fix on a coordinator-assigned
branch with a regression test, re-run the row. If it cannot be fixed within
the workstream, it is listed in the results doc under "Release blockers" with
the exact failing command.

## Done when

`docs/testing/results/2026-09-DD-windows-hop-parity-v060.md` with every row
`PASS` or a listed blocker; Gates 3 and 4; the verifier's re-run of H5 and H8.
