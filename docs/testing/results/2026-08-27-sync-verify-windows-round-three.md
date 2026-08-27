# `rein sync verify` on the Windows bench, round three — 2026-08-27

The third bench run for ticket #12, on the tree after the round-three
fixes. The two earlier records —
[round one](2026-08-27-sync-verify-windows.md) and
[round two](2026-08-27-sync-verify-windows-round-two.md) — stand as what
was true when they were written; several sentences they quote have
deliberately changed since, and this record says which.

Windows 11 (NT 10.0.26200), Go 1.26.1 with `GOTOOLCHAIN=go1.25.13`, native
amd64, branch `hop/12-verify`. Everything below is real output, trimmed
only of the AWS SDK's `WARN Response has no supported checksum` lines (the
S3 fake sends no checksum header) and of `ok` / `no test files` lines where
a summary says so.

Still nothing physical: no second device, no live R2, no real `hopd`, and
the AWS CLI is not installed on this bench. Every seam is in-process — with
one deliberate exception recorded below, where the fake locker is bound to
a **non-loopback** address of this machine so that one code path can run
for the first time.

## What this round changed

Round two closed the reported findings and left narrower cousins. Round
three takes the five surfaces that make two claims about the locker and
makes them tell one story, closes the could-not-run rule on steps 1 and 2,
orders the isolation step's alarm behind its pin, and puts a gate over the
claims so the next surface is held to the rule the day it is written.

## The plaintext refusal, driven through the real client for the first time

The refusal exempts loopback. Every fake locker in `internal/cli` is an
httptest server, and httptest listens on loopback — the one address the
carve-out lets through. So the refusal had never run through the CLI at
all; its only coverage was a unit test of the predicate.

`s3test.NonLoopbackListener` finds an address of this machine that is not
127.0.0.0/8, `::1` or `localhost`, binds it, and dials it to prove it is
reachable. On this bench that is the Hyper-V/WSL host address. The fake
locker is served there, the fake control plane advertises it as the locker
**and** as the reference locker, and the real `rein sync verify` runs
against it:

```text
locker endpoint: http://172.29.80.1:56524
exit=7
Result:         PASS      (step 1)
Result:         PASS      (step 2)
Result:         PASS      (step 3)
Step 4: Prove this account's credentials are refused from another bucket
What was seen:  The control plane pointed this step at http://172.29.80.1:56524,
  which is plaintext http. This step signs its request with the same temporary
  credentials this device pushes with — a secret key and a session token — and it
  sends those over an unencrypted connection to nothing but this machine's own
  loopback address, so no request was made and nothing about bucket scope was
  shown. Step 1 listed this account's locker at http://172.29.80.1:56524, so those
  credentials are already travelling unencrypted on every push: report this to the
  operator.
Result:         FAIL
OUTCOME: FAIL. At least one step did not hold. The failed step above names what
  was seen and what to do about it.
```

The fake's request log is checked after the run: no request naming the
reference bucket reached it, so the credential was not signed for the
foreign bucket before the refusal.

**What this does not cover.**
`TestSyncVerifyRefusesAPlaintextEndpointThroughTheRealClient` **skips** on
a machine with no non-loopback address it can bind and dial — a container
with only `lo`, a laptop with every interface down. On such a machine the
refusal is again covered only by the unit tests of the predicate, and by
`TestSyncVerifyExemptsLoopbackFromThePlaintextRefusal`, which pins the
other half of the carve-out. CI has not been observed running this test.

## The could-not-run rule on steps 1 and 2

`docs/cli-reference.md` promised "a check that could not run is never
reported as a check that failed" while steps 1 and 2 reported a timeout or
a dropped connection as a failure. They now draw step 4's line: a refusal
is an answer and fails the step; no answer at all is `NOT APPLICABLE` with
a reason beginning "Could not run".

Through the real CLI, with the control plane up and the locker's endpoint
closed mid-journey
(`TestSyncVerifyOnAStorageEndpointThatAnswersNothing`):

```text
Step 1: List the locker with this device's credentials
  What was done:  Nothing: the check did not start.
  What was seen:  Could not run: the storage endpoint gave no answer (keyring:
    fetch: operation error S3: GetObject, exceeded maximum number of attempts, 3,
    https response error StatusCode: 0, ..., request send failed, ... connectex: No
    connection could be made because the target machine actively refused it.), so
    the locker could not be opened or listed.
  Result:         NOT APPLICABLE
...
OUTCOME: NOT VERIFIED. Nothing was checked, because the storage endpoint gave no
  answer (...). No step failed and nothing here says anything about what the
  locker holds. Run rein sync verify again when the storage endpoint answers.
exit=1
```

Before this round that run ended with a bare SDK dial error and exit `4`
(`AuthStorage`), with no report at all.

A Hop profile fetches the keyring while it opens, so a storage outage
stops the command before step 1; that is the path above. The other path —
the profile opens and a step gets no answer — is BYO storage, and it is
covered in `internal/verify` by
`TestStepsOneAndTwoDoNotFailOnAnEndpointThatDidNotAnswer`, which also
pins the two refusal cases that must still fail.

**The exception, stated rather than left to be found.** One check that
cannot run is still reported as a failure: step 3 with no key available on
this device. `rein sync verify` resolves a key before it runs, so the
command does not reach that state; the `verify` package called without one
does. `docs/cli-reference.md`, `docs/hop.md` and the CHANGELOG all say so.

## The isolation step's alarm now waits for its pin

`r.foreignBucket` and `degrade(&step, Fail)` were set by the List and
fetch observations *before* `pinToResponse` ran, and `degrade` cannot lift
a `Fail` — so a report could assert that credentials reached a foreign
bucket, and ask for a mail to `security@reinstate.dev`, on the strength of
an observation the pin then invalidated.

`TestTheForeignBucketAlarmWaitsForThePin` drives four transports against a
reference locker that answers the credential:

| what the transport saw | step | summary names the foreign bucket |
| --- | --- | --- |
| landed at the pinned endpoint | FAIL | yes |
| landed at another host | FAIL | no |
| a redirect the probe refused | FAIL | no |
| nothing recorded | FAIL | no |

With the gate removed, the last three rows produce
`FAIL. This account's credentials reached a bucket that is not its own …
send it to security@reinstate.dev.` The step still reports the successful
probe in every row; only the conclusion about *buckets* is withheld.

## The recipe on a prefixed locker

`docs/hop/object-format.md` asserted that a Hop locker has no prefix while
`internal/hop.Locker` carries the field and `rein sync verify` honours it,
and `--export` printed none — so the printed commands listed nothing and
fetched nothing on a locker that had one.
`TestByHandRecipeRunsAsPrinted` now runs the page's shell twice, against a
locker with no prefix and one with `team/a`:

```text
--- PASS: TestByHandRecipeRunsAsPrinted (1.16s)
    --- PASS: TestByHandRecipeRunsAsPrinted/no_prefix (0.57s)
    --- PASS: TestByHandRecipeRunsAsPrinted/prefix_team/a (0.59s)
```

With `REIN_LOCKER_PREFIX` removed from `--export`, both subtests fail: the
first because the name never reaches a child process, the second because
no `aws` command names `team/a/manifest.age`.

The AWS CLI is still not installed here. The recipe's shell runs under
`sh` against shims that record argv and the inherited environment, so the
page's shell and the requests it describes are covered and the `aws`
binary's own argument handling is not.

## The claim gate

Five surfaces state what the locker holds and when the credential goes on
the wire. `internal/doctest/locker_claims_test.go` walks every shipped
page, every non-test Go file under `internal/`, `cmd/` and the website's
content, and the rendered `rein --help` tree, and fails on a paragraph
that says the locker holds "only ciphertext" without naming
`keyring.v1.json`, or describes the plaintext-`http` refusal without
naming the loopback exemption.

It found two surfaces nobody had reported: `docs/architecture.md` and its
copy under `website/`, whose "Zero-knowledge remote — only ciphertext on
object storage" design principle is falsified by the plaintext keyring.
Both now name it.

Proven to bite by adding the claim to a page that did not have it:

```text
--- FAIL: TestLockerClaimsCarryTheirExceptions (0.14s)
    docs\faq.md makes the "what the locker holds" claim without its exception; …
    docs\faq.md makes the "when the locker credential is sent" claim without its
      exception; …
```

and, for the help tree, by stripping the qualifying sentence from `rein
hop`'s `Long`:

```text
--- FAIL: TestLockerClaimsInCommandHelp (0.00s)
    rein hop help makes the "what the locker holds" claim without its exception; …
```

The gate reads neither `references/` (third-party research about other
products, quoted as written) nor `docs/testing/` (bench records, including
this one, kept verbatim) nor released CHANGELOG sections. Each exclusion
is named in the test with its reason; there are three, and adding a fourth
means editing the gate.

## Suite, race, vet, cross-builds

```text
env -u REINSTATE_BACKEND -u REINSTATE_MEMORY_BACKEND_DIR \
    -u REINSTATE_S3_ACCESS_KEY_ID -u REINSTATE_S3_SECRET_ACCESS_KEY go test ./...
--- FAIL: TestDaemonJourneyHop
--- FAIL: TestMigrateJourneyLeaveHopForBYO
FAIL	github.com/HarjjotSinghh/reinstate/internal/cli	41.115s
(every other package ok)

make test-race        same two failures, no data race reported, 65.6s
go vet ./...          clean
GOOS=windows|darwin|linux GOARCH=amd64 go build ./...   all clean
```

Those two failures are the ticket-branch failures the round-two record
already documents: they are not caused by anything in this round and
resolve on merge. `internal/preflight`'s shared-deadline test was re-run in
isolation and passed (`ok … 0.442s`); it is a wall-clock flake under load.

`internal/crypto` and `internal/doctest` are excluded from
`FAST_PACKAGES`, so `make test-race` does not cover them; `go test ./...`
above does.
