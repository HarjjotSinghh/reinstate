# Adversarial verification of #11 (revocation) and #12 (sync verify) — 2026-08-27

Windows 11 bench (NT 10.0.26200), Go 1.26.1, native amd64. Public CLI at
`hop/11-revocation` b451d611 and `hop/12-verify` 52907d40; private control
plane at 6c22b8a and c7ae807. This is the verification round the 2026-08-24
pause record left undone: both fixers had committed, both adversarial
verifiers were cut at a usage limit.

## Verdict

- **#11 — `FAIL`.** One blocker, three majors. The blocker refutes the
  ticket's central documented promise. The ticket's *functional* acceptance
  criteria both passed.
- **#12 — `FAIL`.** One blocker, five majors. The blocker reopens the exact
  hole commit 52907d40 was written to close.

Neither branch merges as it stands. Neither verdict is a regression from
these commits: both are properties the branches **document as holding**
that do not hold. The code-level separation between the two tickets is
clean, so the merge plan in section 5 survives both fixes.

## 1. Method

Two workflows and a re-run, fourteen agents. Six adversarial verifiers
(three lenses per ticket, client and control plane), a judge per ticket
that reproduced every blocker before accepting it, a completeness critic,
and five acceptance-criteria agents pointed at the issue text rather than
the commit claims. Each agent worked in its own git worktree so mutation
testing could not contaminate a neighbour.

The judges downgraded three verifier findings they could not sustain. One
acceptance finding was mis-scoped and is withdrawn in section 4.

## 2. #11 — the blocker

`checkKeyGenerationFloor` refuses only a **lower** generation. A party with
bucket write access appends a forged **higher** one instead.

Three code facts, confirmed directly against b451d611:

- `internal/cli/account.go:582` — `if observed < floor { refuse }`. Higher
  passes.
- `internal/keyring/keyring.go:185` — `Parse` validates structure only:
  positive numbers, no duplicates, wrap-format rules. There is no
  cryptographic link between generation N and N+1.
- `internal/keyring/keyring.go:270` — the wrap binding is
  `{profileID, generation.Number}`. Both are public. Nothing secret enters
  the binding.

Reproduced end to end, driving the real CLI against the fake control plane:
A enrols, pushes, approves B, revokes B to generation 2 and pins floor 2.
Using only `keyring.v1.json`, which is public, an attacker appends
generation 3 with a root key of its own choosing and one age wrap per
listed device sealed to that device's published public key. Observed:
`Parse` accepts it, `A push: exit=0`, and the post-revocation snapshot
opens under the attacker's root key. `observeKeyGeneration` then raises A's
floor to 3, so a genuine recovery below it is also refused.

The write access this needs is granted by the product's own documented
behaviour: a revoked device keeps already-minted bucket credentials for up
to an hour.

This contradicts `docs/hop.md:178`, `docs/security-model.md`, and
`CHANGELOG.md:197` ("cannot open anything pushed after the revocation").
The follow-up those docs name — carrying the generation floor on the
control plane — does not close it, because the forgery goes upward.

Closing it needs the keyring to authenticate each generation: a MAC over
generation N+1's header keyed by generation N's root key, checked on every
read path. A device revoked at generation N never holds N's root key and
cannot forge the link.

## 3. Findings

### #11 majors

- **The re-enrolment recipe this diff adds cannot succeed.**
  `docs/hop.md:205` documents recovery for a revoked device. Run verbatim,
  both `rein account recover` and `rein account join` exit 7, "this device
  is already enrolled": nothing in the CLI removes `account.json`.
  `existingInitFiles` does not list it and `BackupFiles` copies rather than
  deletes. The branch's own test only gets past it with a direct
  `os.Remove`.
- **Self-revocation is not concurrency-safe.** `store.RevokeDevice` guards
  only `deviceID == byDeviceID`, and `authDevice` reads before the
  transaction writes. 39 of 40 rounds of two devices revoking each other
  ended with zero unrevoked devices on the account, both tokens 401, and no
  device able to approve a replacement.
- **An in-flight credential mint outlives revocation.** 28 of 30 rounds
  handed the revoked device a fresh full-TTL credential after revocation
  committed. The public docs state this window accurately; private
  `docs/hop.md` section 4 and `SPEC.md` user story 11 contradict them.

### #11 minors

- `rein devices revoke`'s own help text says the revoked device "cannot
  push". That is false for up to an hour and contradicts `docs/hop.md`
  lines 174-178, which state the window correctly. The success message
  printed after a revocation has the same omission. Confirmed: new mints
  are refused instantly (401), but the credential already held stays valid
  for the rest of its TTL, and `storage.Provider` exposes no withdrawal
  method at all.
- Keyring growth per generation is unbounded and nothing compacts it. At 5
  surviving devices the object grew 4,535 bytes per generation (180,733
  bytes at depth 40). `keyring.Load` caps a read at 1 MiB, so at roughly
  231 revocations the account writes a keyring nobody can read — not a
  push, not a pull, not another revocation, not `rein account recover`.
  Extrapolated from measured growth; the cliff was not driven to.

### #12 blocker and majors

- **BLOCKER — the endpoint pin is a string compare between two values the
  control plane itself supplies, and nothing checks where the request
  landed.** One redirect rule on the pinned endpoint restores the hole, and
  the report returns PASS asserting the credential was refused by a foreign
  bucket when the credential was never sent. The pin must be on the
  response: refuse redirects, and require the refusal to have carried the
  credential.
- `isolationStep` does not require step 1 to have passed, so an always-403
  host yields `isolation: pass` and `IsolationChecked() == true`.
- The report calls the fetched object "the newest snapshot". Nothing
  observes recency — the pick is `sort.Strings` over random uuid v4 ids —
  so on a locker with n snapshots the claim is wrong about (n-1)/n of the
  time. Mutating the pick leaves the suite green.
- `verify_reports` stores unconstrained client free text. `did` and
  `observed` are clipped to 2048 bytes and stored verbatim, so the control
  plane holds exactly the metadata `0004_verify.sql` and `docs/hop.md`
  promise it never holds.
- The reference locker is advertised forever once its row exists. The
  honest `no_reference` degrade added by 205b9ae is startup-only and
  first-time-only; a deleted bucket is never re-provisioned or re-checked.
- The 20-report per-device cap bounds nothing an account controls:
  `plan.Devices` is enforced only in `mintCredentials`, never at sign-in,
  and neither login nor report posting is rate limited.

## 4. Acceptance criteria, against the issue text

| Issue | Criterion | Status |
| --- | --- | --- |
| #11 | Remaining devices read all pre-revocation history, write under the new generation | **met** |
| #11 | Keyring CAS discipline; concurrent revocations do not lose wraps | **met** |
| #11 | Revocation visible in the console device list | **blocked** — the console is #14, not built |
| #12 | Auto-run after first push on each new device; report stored | **met** |
| #12 | Output readable by a non-expert, reproducible step by step | **partially met** |
| #12 | Object-format spec and threat model match the protocol | **not met** |

#11's two functional criteria were demonstrated by execution and are
solid. History stayed readable at every depth probed: 12 generations with
one sealed envelope each, opened end to end after 11 rollovers, and 40
generations at the keyring seam. `rein account recover` with the original
recovery code after two revocations landed at generation 3 holding all
three and restored the full history. Under `-race`: two concurrent
revocations of different devices 30/30 clean, six-way 5/5, a revocation
racing an approval left the joiner in the current generation 30/30, and a
four-shell CLI race over one home 25/25 with zero lost wraps.

#12 auto-run was proven end to end — the client's own 2,099-byte report
bytes POSTed to the real `hopd` handler, written to sqlite, and read back.

Two blockers under readability, both real:

- Nothing pushed yet, and a mistyped BYO passphrase, both print
  `OUTCOME: FAIL … worth reporting to security@reinstate.dev`. The
  trust-establishing command tells users to report a security incident for
  a non-event.
- On Hop **no step is reproducible by hand**, though `docs/hop.md:201`
  promises it and `docs/hop/object-format.md:159` ships the recipe. No
  command yields the hourly locker credentials, and no command exports a
  root-key identity file — `crypto.RootKeyIdentity` is package-internal. A
  Hop user cannot perform documented step 3.

Also: `rein sync verify`'s documented failure exit code is `4` in three
places; the observed code is `7`. `4` is `AuthStorage`, so a user scripting
the documented value gets it backwards.

**Withdrawn.** One agent reported as a blocker that revocation "does not
exist anywhere in the repository". It was working in the #12 worktree.
`rein devices revoke` and `Keyring.Rollover` exist on **#11**; the docs
describing them exist only on **#12**. Each branch holds half of the claim,
and it resolves on merge — but it must be re-verified there, because no
test has ever run against a tree containing both.

## 5. Merge plan

Verified by trial merge, not predicted.

Public, **#11 first**:

- `#11` onto `hop/main` — clean.
- `#12` onto that — conflicts in `CHANGELOG.md`, `docs/hop.md`, and
  `internal/cli/hop_test.go`. The first two are union resolves, ordering
  #11's text first. `hop_test.go` is **not** a union resolve: both branches
  add cases to the same table and both sets must survive.

Private, **#11 first**:

- `#11` onto `main` — clean.
- `#12` onto that — conflicts in `docs/hop.md` and
  `internal/server/server.go`. Union both route blocks into the mux.
- **The trap: `internal/store/store_test.go` auto-merges and is then
  wrong.** Both branches independently changed the same line from `n != 3`
  to `n != 4`, so the blobs are byte-identical and git keeps `4` with no
  conflict marker — but the merged tree carries **five** migrations, and
  fails `TestOpenMigratesOnceAndReopens`. Set it to 5.
- `README.md` auto-merges and then goes stale: it points at "`docs/hop.md`
  section 4", which the merge renumbers.

The duplicate `0004` migrations are **functionally safe**. `migrate()` keys
`schema_migrations` on the full filename and sorts by it, so
`0004_revocation.sql` and `0004_verify.sql` both apply in that order, and
their schemas are disjoint — one adds columns to `devices`, the other adds
two new tables. Rename #12's to `0005_verify.sql` for legibility while no
database has recorded the name. This is the second reason to merge #11
first: if #12 lands and deploys first, the rename target flips and the
apply order changes with it.

`0004_revocation.sql` is not self-idempotent, and the tree has no down
migrations.

## 6. What this round did not cover

- **`make test-race` was never run on the public repo by anyone**, though
  `make verify` and CI both include it and this bench has cgo.
  `revocation_test.go:498` spawns two goroutines through one fake control
  plane and has never seen the detector. (The #11 forward-path probes did
  run under `-race`, but only over code they wrote themselves.)
- **No fuzzing**, against the project's own practice — the repo has eight
  `Fuzz*` targets, and both tickets add a parser fed by untrusted bytes:
  `keyring.Parse` (the surface of #11's blocker) and `sanitizeVerifyReport`.
- **Windows only.** Cross-*builds* ran; cross-*execution* did not.
  `internal/crypto/passphrase_fd_unix.go` carries #11's recovery-code read
  and has never executed, and `internal/crypto` is excluded from
  `FAST_PACKAGES`, so `make test` skips it entirely.
- **No two-device journey against real `hopd`, and no live R2.** Everything
  ran against fakes. The credential-window conclusion rests on the shape of
  `storage.Provider` and the fake's records, not on a real R2 temporary
  credential observed surviving a revocation. #9 and #10 each committed a
  physical journey record; neither #11 nor #12 adds one.
- **Nothing has ever run against a tree containing both tickets**, nor
  against either ticket under the daemon (#10), which runs
  `push --all --json` in-process and therefore fires both code paths.

## 7. Shipped claims with no evidence behind them

- `docs/hop.md`, `docs/hop/threat-model.md` and `docs/hop/object-format.md`
  are **absent from the doc gate** in
  `internal/doctest/phase4_cli_contract_test.go`, which lists fourteen docs
  and none of these three.
- #12's CHANGELOG claims the S3 backend's `List` "now follows continuation
  tokens". `List` is byte-identical between f9afc556 and 52907d40; that
  work shipped in #13. #12 added a doc comment.
- #12's CHANGELOG asserts a seven-journey Windows bench run on 2026-08-23
  with no committed record. The convention is `docs/testing/results/`.
- `internal/backend/s3/s3test/fake.go` places its new foreign-bucket
  refusal **before** credential validation, so the fake answers
  `AccessDenied` for a bad credential where real R2 answers
  `InvalidAccessKeyId` — the exact distinction #12 step 4 is built on. This
  shared fixture is used repo-wide and was not diff-reviewed.
