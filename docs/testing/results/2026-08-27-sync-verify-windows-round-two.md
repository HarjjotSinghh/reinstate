# `rein sync verify` on the Windows bench, round two — 2026-08-27

The second bench run for ticket #12, on the tree after the round-two
fixes. The first record —
[`2026-08-27-sync-verify-windows.md`](2026-08-27-sync-verify-windows.md) —
stands as what was true when it was written; three of the behaviours it
describes have deliberately changed since, and this record says which.

Everything below is real output, trimmed only of the AWS SDK's
`WARN Response has no supported checksum` lines (the S3 fake sends no
checksum header) and of `ok`/`no test files` lines where a summary says
so.

Nothing physical here either: no second device, no live R2, no real
`hopd`, and the AWS CLI is not installed on this bench. Every seam is
in-process.

## Verdict

- **The `rein sync verify` and `rein hop credentials` journeys:** `PASS`,
  fifteen of fifteen, driving the real CLI end to end.
- **`internal/verify` unit checks:** 29 tests, 119 cases, all passing.
- **`make test-race`:** no data race reported, on a bench with cgo.
- **`go vet ./...`:** clean.
- **Cross-builds** for `windows/amd64`, `darwin/arm64` and `linux/amd64`:
  clean.
- **Full suite:** two failures, both carried by this branch before any of
  this ticket's work and both fixed on `hop/main`
  (`TestDaemonJourneyHop`, `TestMigrateJourneyLeaveHopForBYO`), each
  confirmed on the stashed tree as well. No other failure.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-27` |
| Bench | `Microsoft Windows 11 Pro` `10.0.26200`, `windows-amd64` native (not WSL) |
| Toolchain | `go version go1.26.1 windows/amd64`; `make test-race` pins `GOTOOLCHAIN=go1.25.13`; `gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh) 16.1.0`, `CGO_ENABLED=1` available |
| Shell | Git Bash `sh` (`/usr/bin/sh`), which the recipe journey needs |
| Tree | `hop/12-verify` at `d23db398` plus this round's commits; worktree `D:\Projects\reinstate-worktrees\12` |
| Control plane | the in-process fake in `internal/cli/hop_locker_test.go` and `verify_fake_test.go`; no `hopd`, no network beyond loopback |
| Storage | `internal/backend/s3/s3test` (fake S3, loopback `httptest` server); `internal/backend/memory` on disk for the BYO journey |
| Environment | every run under `env -u REINSTATE_BACKEND -u REINSTATE_MEMORY_BACKEND_DIR -u REINSTATE_S3_ACCESS_KEY_ID -u REINSTATE_S3_SECRET_ACCESS_KEY` |
| Isolation | each test sets its own `REINSTATE_HOME`, `HOME`/`USERPROFILE`, `CLAUDE_CONFIG_DIR` and `CODEX_HOME` under `t.TempDir()`; no real `~/.claude` or `~/.reinstate` was read |

## 2. The journeys

```
go test ./internal/cli/ -run 'TestSyncVerify|TestHopCredentials|TestByHandRecipe' -count=1 -v

--- PASS: TestHopCredentialsMakesTheByHandRecipeReal (0.31s)
--- PASS: TestHopCredentialsNeedsASignedInDevice (0.03s)
--- PASS: TestByHandRecipeRunsAsPrinted (0.63s)
--- PASS: TestSyncVerifyJourneyHosted (0.42s)
--- PASS: TestSyncVerifyHostedReportGolden (0.33s)
--- PASS: TestSyncVerifyJourneyTamperedObjects (0.36s)
--- PASS: TestSyncVerifyJourneyReferenceReachable (0.29s)
--- PASS: TestSyncVerifyJourneyReferenceRejectsTheCredential (1.15s)
--- PASS: TestSyncVerifyJourneyNoReferenceLocker (0.32s)
--- PASS: TestSyncVerifyControlPlaneFaultsDoNotFailTheVerification (0.60s)
--- PASS: TestSyncVerifyRefusesAReferenceEndpointStepOneDidNotList (0.63s)
--- PASS: TestSyncVerifyBeforeAnyPush (0.23s)
--- PASS: TestSyncVerifyUnreachableControlPlanePrintsAReport (0.28s)
--- PASS: TestSyncVerifyExitCodes (0.28s)
--- PASS: TestSyncVerifyJourneyBYO (0.23s)
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	7.499s
```

Three of them are new in this round, and one changed its expectations:

| Journey | What it proves |
| --- | --- |
| `ControlPlaneFaultsDoNotFailTheVerification` | the control plane answering `500` on `GET /v1/verify/reference`, and a reference row naming the account's **own** bucket, each leave step 4 NOT APPLICABLE with a "Could not run" reason, exit `0`, post a report whose outcome is `pass`, and claim nothing about bucket scope |
| `RefusesAReferenceEndpointStepOneDidNotList` | a reference endpoint that is not the one step 1 listed — including the same host over plaintext `http` — fails step 4, and the credential is never signed for it (nothing reaches the S3 fake as a foreign-bucket request) |
| `ByHandRecipeRunsAsPrinted` | `docs/hop/object-format.md`'s shell, run through `sh` against the fake locker, followed by the same four requests made for real with the credentials the recipe printed |
| `JourneyReferenceRejectsTheCredential` | **changed**: `InvalidAccessKeyId`, `SignatureDoesNotMatch`, `ExpiredToken` and `InvalidToken` now leave step 4 NOT APPLICABLE and exit `0` rather than failing the run. A locker credential lasts an hour; one that died between step 1 and step 4 is a non-event, and the previous record's line for this journey no longer describes the branch |

## 3. What the three fixed reports now say

Real output, from a run of the report writer over each arrangement:

```
===== the control plane answered 500 =====
Step 4: Prove this account's credentials are refused from another bucket
  What was seen:  Could not run: the control plane did not say where its reference locker is (internal error), so nothing about bucket scope was shown. That is a fault on the control plane's side, not a finding about this locker. The first three steps above ran entirely against storage and the key on this device and stand on their own; run rein sync verify again later.
  Result:         NOT APPLICABLE

OUTCOME: PASS. The objects checked (the index and the newest snapshot in the index) are ciphertext this device can open. … Whether the credentials reach other buckets was not checked (step 4 above says why), so nothing is claimed about that.

===== the reference row names the account's own bucket =====
  What was seen:  Could not run: the control plane named this account's own bucket (lk-1) as its reference locker. These credentials are supposed to reach that bucket, so nothing it answers says anything about other buckets. That is a fault on the control plane's side, not a finding about this locker. …
  Result:         NOT APPLICABLE

===== the pinned endpoint is plaintext http =====
  What was seen:  The control plane pointed this step at http://s3.example, which is plaintext http. This step signs its request with the same temporary credentials this device pushes with — a secret key and a session token — and those are never sent over an unencrypted connection, so no request was made and nothing about bucket scope was shown. Step 1 listed this account's locker at http://s3.example, so those credentials are already travelling unencrypted on every push: report this to the operator.
  Result:         FAIL

OUTCOME: FAIL. At least one step did not hold. The failed step above names what was seen and what to do about it.
```

## 4. The unit checks

```
go test ./internal/verify/ -count=1 -v      # 29 tests, 119 cases, ok
go test ./internal/doctest/ -count=1        # ok, including the new object-format gate
```

Each fix was also mutation-tested: the fix was reverted in place, the new
test was confirmed to fail, and the fix restored. That is recorded in the
commit messages rather than here.

## 5. `make test-race`

```
env -u REINSTATE_BACKEND -u … make test-race
CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test <FAST_PACKAGES> -race -count=1 -timeout=20m

--- FAIL: TestDaemonJourneyHop (2.07s)
--- FAIL: TestMigrateJourneyLeaveHopForBYO (0.70s)
FAIL	github.com/HarjjotSinghh/reinstate/internal/cli	64.887s
```

**No `DATA RACE` block was printed.** `FAST_PACKAGES` excludes
`internal/doctest` and `internal/crypto`, so those two are still not
covered by `-race` on this bench; `internal/verify` and `internal/cli`,
which hold this ticket's work, are.

## 6. `go vet` and the cross-builds

```
go vet ./...                                   # exit 0, no output
GOOS=windows GOARCH=amd64 go build ./...       # exit 0
GOOS=darwin  GOARCH=arm64 go build ./...       # exit 0
GOOS=linux   GOARCH=amd64 go build ./...       # exit 0
gofmt -l ./cmd ./internal                      # no output
```

Cross-*builds* only. Nothing was cross-*executed*.

## 7. Full suite

```
env -u REINSTATE_BACKEND -u … go test ./...

--- FAIL: TestDaemonJourneyHop (1.33s)
--- FAIL: TestMigrateJourneyLeaveHopForBYO (0.58s)
FAIL	github.com/HarjjotSinghh/reinstate/internal/cli	33.183s
```

Every other package passes. Both failures were reproduced on this branch
with the round-two work stashed, so neither is this round's.

## 8. What this record does not cover

- **The AWS CLI itself.** `aws` is not installed on this bench and is not
  a dependency of this repository, so the recipe journey runs the page's
  shell with `aws` replaced by a shim that records what it was given and
  serves the object bodies, and makes the same requests separately with
  the SDK. The recipe's shell, its environment, and its requests are
  covered; the AWS CLI's own argument handling is not.
- **No physical journey.** One machine, no second device, no live R2, no
  real `hopd`. Step 4's conclusions rest on the S3 fake's behaviour, which
  answers a foreign bucket `AccessDenied` only after the signature
  validates, as R2 does.
- **The plaintext refusal was not driven through the CLI.** Every real
  endpoint in these tests is a loopback `httptest` server, which is the
  one address the refusal exempts, so the refusal is covered by the
  `internal/verify` table (four non-loopback plaintext endpoints refused,
  three loopback ones allowed) and by the CLI only through the endpoint
  pin that rejects a plaintext endpoint step 1 did not list.
- **No `-race` over `internal/crypto` or `internal/doctest`.**
- **No fuzzing.**
- **Nothing has been run against a tree containing both #11 and #12.**
  That still has to be verified on the merge. The two protocol pages are
  written so that the merge cannot silently falsify them: they state no
  release property they cannot check, and their format examples are held
  to the constants the code defines, which was confirmed by bumping
  `keyring.SchemaVersion` to `3` — the value #11 sets — and re-running the
  gate, which passed.
