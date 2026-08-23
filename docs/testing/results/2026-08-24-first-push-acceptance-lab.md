# First-push acceptance suite against a running control plane — lab, 2026-08-24

The tagged acceptance `TestHopFirstPushJourneyStaging` (`internal/cli`,
`-tags hopacceptance`) is the same journey as the in-process
`TestHopFirstPushJourney` — sign in, `init --hop`, `account init`, push
three sessions, wipe, sign in again, `account recover`, pull, verified
resume — against a real control plane and a real locker address. Review of
the first version found it could never pass: it seeded one device token
without a device id (`rein init --hop` refuses that), and it reused the same
device for the wiped leg (`rein account recover` refuses a keyring entry this
machine holds no key for). This record is the proof that the rewritten suite
runs and passes, in both of its sign-in modes, against `hopd`.

No staging control plane exists yet (`hop-staging.reinstate.dev` does not
resolve), so the control plane was `hopd` from `reinstate-hosted` `main` at
`0fda8f3` on loopback with the fake storage provider, and the locker was
`scripts/testing/fakelocker` — the same lab as the Windows record. When a
staging deployment exists, the same two commands below run against it with
`HOP_STAGING_URL` changed; nothing in the suite is loopback-specific. Every
output below is real; the device tokens are the only redaction.

## Verdict

- **Email sign-in mode (`HOP_LOGIN_EMAIL`):** `PASS`. Both devices ran a
  real `rein login --email` and waited for the link; a lab script on the
  same machine did what the browser does (GET the confirm page, POST the
  form) about two seconds later. The wiped device was enrolled as a new
  device (`b4f2d487…`, not `eced5cbe…`), so `account recover` added it to
  the keyring instead of refusing.
- **Pre-issued token mode (`HOP_DEVICE_TOKEN` + `HOP_DEVICE_TOKEN_2`):**
  `PASS`. Device ids are resolved from `/v1/whoami` before anything is
  pushed; the suite refuses two tokens of one device up front (shown below)
  rather than failing at `account recover`.
- **first_push exactly once:** `PASS`. One `first_push` row per account in
  the control plane's `events` table after each full journey.
- **Sign-in to first push:** 165 ms and 169 ms (budget 120 s), measured from
  the signed-in device; the two-second wait for a person (here, the script)
  to approve the link is logged separately and excluded.
- **Verified resume with the real verifier:** Claude Code and Codex
  `verified, ready` (both installed on this Mac); OpenCode blocked only on
  `agent.executable` (not installed here), which the suite records rather
  than fails.
- **Lab locker:** three accounts used one running `fakelocker` without the
  second and third `rein account init` finding the first account's keyring;
  `AnyBucket` now serves each bucket from its own store.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-23` (IST 2026-08-24 early morning; hopd stamps UTC) |
| Machine | macOS `26.5.2`, `arm64`, `go1.26.5 darwin/arm64`; claude and codex on `PATH`, opencode not installed |
| Client | `hop/08-first-push-journey` worktree (this branch, the commit that adds this record) |
| Control plane | `hopd` @ `0fda8f3`: `HOPD_ADDR=127.0.0.1:8082 HOPD_BASE_URL=http://127.0.0.1:8082 HOPD_S3_ENDPOINT=http://127.0.0.1:9002 HOPD_EMAIL_SENDER=log HOPD_STORAGE=fake`, fresh `hopd.db` |
| Locker | `scripts/testing/fakelocker -addr 127.0.0.1:9002` (built from this branch) |
| Sign-in | email magic links printed by hopd's log sender; a lab script tails the log, GETs the confirm page, and POSTs the form; in token mode the same script minted the two tokens through `POST /v1/login/sessions` + poll |
| Isolation | `HOME`, `CODEX_HOME`, `XDG_DATA_HOME`, and `REINSTATE_HOME` under `t.TempDir()`; in-memory device token and secret stores; no real keyring or agent store touched |
| Cleanup | `hopd`, `fakelocker`, and the approver stopped; `hopd.db` kept for the event counts below |

## 2. Email sign-in mode

```text
$ HOP_STAGING_URL=http://127.0.0.1:8082 HOP_LOGIN_EMAIL=first-push-acceptance-1@example.com HOP_LOGIN_TIMEOUT=2m \
    go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v
=== RUN   TestHopFirstPushJourneyStaging
    hop_first_push_acceptance_test.go:150: device "acceptance-laptop": approve the sign-in link sent to first-push-acceptance-1@example.com within 2m0s
    hop_first_push_acceptance_test.go:150: device "acceptance-laptop" signed in as device eced5cbe-0da3-446a-b5a2-8f5d774d779a after 2.006s
    hop_first_push_acceptance_test.go:168: sign-in to first successful push against http://127.0.0.1:8082: 165ms (budget 2m0s; 2.006s more waiting for the sign-in link to be approved)
    hop_first_push_acceptance_test.go:179: first push recorded at 2026-08-23T17:30:53Z
    hop_first_push_acceptance_test.go:192: device "acceptance-laptop-wiped": approve the sign-in link sent to first-push-acceptance-1@example.com within 2m0s
    hop_first_push_acceptance_test.go:192: device "acceptance-laptop-wiped" signed in as device b4f2d487-e029-4a14-ad69-c38a8735a590 after 2.003s
    hop_first_push_acceptance_test.go:226: resume claude:session-first-push: verified, ready
    hop_first_push_acceptance_test.go:226: resume codex:rollout-first-push: verified, ready
    hop_first_push_acceptance_test.go:228: resume opencode:ses_fixture001: blocked only because the vendor executable is not installed here
--- PASS: TestHopFirstPushJourneyStaging (6.38s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	7.637s
```

hopd's log carried the two links, and the approver clicked them:

```text
hopd: email to first-push-acceptance-1@example.com — Sign in to Reinstate Hop
Open this link to sign in the device "acceptance-laptop" (darwin-arm64):
…
hopd: email to first-push-acceptance-1@example.com — Sign in to Reinstate Hop
Open this link to sign in the device "acceptance-laptop-wiped" (darwin-arm64):
…
23:00:52 approved /login/email/oezM6epQ… confirm page ok POST 200
23:00:55 approved /login/email/KpJWcclp… confirm page ok POST 200
```

(The AWS SDK's `WARN Response has no supported checksum` lines from the fake
locker are omitted from both runs; they are the fake's missing checksum
headers, not the product. The test-file line numbers differ between the two
runs because the up-front token check was added between them.)

## 3. Pre-issued token mode

Two tokens minted for a fresh account (`first-push-acceptance-3@example.com`),
one per device name:

```text
$ HOP_STAGING_URL=http://127.0.0.1:8082 HOP_DEVICE_TOKEN=<token of token-laptop> HOP_DEVICE_TOKEN_2=<token of token-laptop-wiped> \
    go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v
=== RUN   TestHopFirstPushJourneyStaging
    hop_first_push_acceptance_test.go:180: sign-in to first successful push against http://127.0.0.1:8082: 169ms (budget 2m0s; 0s more waiting for the sign-in link to be approved)
    hop_first_push_acceptance_test.go:191: first push recorded at 2026-08-23T17:32:52Z
    hop_first_push_acceptance_test.go:238: resume claude:session-first-push: verified, ready
    hop_first_push_acceptance_test.go:238: resume codex:rollout-first-push: verified, ready
    hop_first_push_acceptance_test.go:240: resume opencode:ses_fixture001: blocked only because the vendor executable is not installed here
--- PASS: TestHopFirstPushJourneyStaging (2.92s)
PASS
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	3.774s
```

The same token in both variables is refused before any push:

```text
$ HOP_STAGING_URL=http://127.0.0.1:8082 HOP_DEVICE_TOKEN=<token> HOP_DEVICE_TOKEN_2=<same token> go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v
    hop_first_push_acceptance_test.go:92: HOP_DEVICE_TOKEN and HOP_DEVICE_TOKEN_2 both belong to device afe9826a-133b-419b-9c96-61a541dde08f; the wiped device needs its own device, see the test comment
--- FAIL: TestHopFirstPushJourneyStaging (0.03s)
```

## 4. Control plane events

After all runs (`sqlite3 hopd.db`, events joined to accounts; the second
account was a token-mode run of the suite before the up-front token check
was added, same result):

```text
first-push-acceptance-1@example.com|device_enrolled|2
first-push-acceptance-1@example.com|first_push|1
first-push-acceptance-1@example.com|locker_provisioned|1
first-push-acceptance-1@example.com|sign_up|1
first-push-acceptance-2@example.com|device_enrolled|2
first-push-acceptance-2@example.com|first_push|1
first-push-acceptance-2@example.com|locker_provisioned|1
first-push-acceptance-2@example.com|sign_up|1
first-push-acceptance-3@example.com|device_enrolled|2
first-push-acceptance-3@example.com|first_push|1
first-push-acceptance-3@example.com|locker_provisioned|1
first-push-acceptance-3@example.com|sign_up|1
```

Two `device_enrolled` rows per account (day one and the wiped device), one
`first_push` each.

## 5. Skip

Without the environment the suite skips, on macOS and on the Windows bench
(see the Windows record):

```text
$ go test -tags hopacceptance ./internal/cli -run TestHopFirstPushJourneyStaging -count=1 -v
    hop_first_push_acceptance_test.go:54: HOP_STAGING_URL is not set; the staging first-push journey is skipped
--- SKIP: TestHopFirstPushJourneyStaging (0.00s)
```
