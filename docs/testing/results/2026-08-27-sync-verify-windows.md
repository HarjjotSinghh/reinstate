# `rein sync verify` on the Windows bench — 2026-08-27

In-process bench run for ticket #12, recorded because the `[Unreleased]`
CHANGELOG asserted a Windows run of these journeys on 2026-08-23 with no
committed artefact behind it. The convention is a record under
`docs/testing/results/`, so this is that run, performed now, on the tree
that ships. Everything below is real output, trimmed only of the AWS SDK's
`WARN Response has no supported checksum` lines (the S3 fake sends no
checksum header) and of `ok`/`no test files` lines where a summary says
so.

Nothing physical here: no second device, no live R2, no real `hopd`. Every
seam is in-process — the fake control plane in `internal/cli`, the shared
S3 fake in `internal/backend/s3/s3test`, synthetic agent stores under the
test's own temporary home. That is the scope of the claim this record
supports, and it is narrower than #9's and #10's physical journeys.

## Verdict

- **The `rein sync verify` journeys:** `PASS`, eleven of eleven, driving
  the real CLI end to end.
- **`make test-race`:** no data race reported, on a bench with cgo. This
  is the first time it has been run on the public repository at all —
  the 2026-08-27 verification round recorded that nobody had, although
  `make verify` and CI both include it.
- **`go vet ./...`:** clean.
- **Cross-builds** for `windows`, `darwin` and `linux` (`amd64`): clean.
- **Full suite:** two failures, both carried by this branch before any of
  this ticket's work and both fixed on `hop/main`
  (`TestDaemonJourneyHop`, `TestMigrateJourneyLeaveHopForBYO`). No other
  failure.

## 1. Test record

| Field | Value |
| ----- | ----- |
| UTC date | `2026-08-27` |
| Bench | `Microsoft Windows 11 Pro` `10.0.26200.8328`, `windows-amd64` native (not WSL) |
| Toolchain | `go version go1.26.1 windows/amd64`; `make test-race` pins `GOTOOLCHAIN=go1.25.13`; `gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh) 16.1.0`, `CGO_ENABLED=1` available |
| Tree | `hop/12-verify` at `b1c8f8f2`, worktree `D:\Projects\reinstate-worktrees\12` |
| Control plane | the in-process fake in `internal/cli/hop_locker_test.go` and `verify_fake_test.go`; no `hopd`, no network beyond loopback |
| Storage | `internal/backend/s3/s3test` (fake S3, loopback `httptest` server) for the hosted journeys; `internal/backend/memory` on disk for the BYO journey |
| Environment | every run under `env -u REINSTATE_BACKEND -u REINSTATE_MEMORY_BACKEND_DIR -u REINSTATE_S3_ACCESS_KEY_ID -u REINSTATE_S3_SECRET_ACCESS_KEY`, so no leftover lab variable reached a test |
| Isolation | each test sets its own `REINSTATE_HOME`, `HOME`/`USERPROFILE`, `CLAUDE_CONFIG_DIR` and `CODEX_HOME` under `t.TempDir()`; no real `~/.claude` or `~/.reinstate` was read |

## 2. The journeys

```
go test ./internal/cli/... -run 'TestSyncVerify|TestHopCredentials' -count=1 -v

--- PASS: TestHopCredentialsMakesTheByHandRecipeReal (0.26s)
--- PASS: TestHopCredentialsNeedsASignedInDevice (0.03s)
--- PASS: TestSyncVerifyJourneyHosted (0.38s)
--- PASS: TestSyncVerifyHostedReportGolden (0.27s)
--- PASS: TestSyncVerifyJourneyTamperedObjects (0.29s)
--- PASS: TestSyncVerifyJourneyReferenceReachable (0.24s)
--- PASS: TestSyncVerifyJourneyReferenceRejectsTheCredential (0.96s)
--- PASS: TestSyncVerifyJourneyNoReferenceLocker (0.27s)
--- PASS: TestSyncVerifyBeforeAnyPush (0.21s)
--- PASS: TestSyncVerifyExitCodes (0.27s)
--- PASS: TestSyncVerifyJourneyBYO (0.20s)
ok  	github.com/HarjjotSinghh/reinstate/internal/cli	4.756s
```

What each one drives, so the list is readable without the source:

| Journey | What it proves |
| --- | --- |
| `JourneyHosted` | sign in, `init --hop`, `account init`, push; the first push verifies once and posts once; `rein sync verify` reads as a report, posts again on demand, names the same access key id in steps 1 and 4, and uploads no session name |
| `HostedReportGolden` | the `--json` document for a Hop locker, pinned by `internal/cli/testdata/verify/hop-report.golden.json` |
| `JourneyTamperedObjects` | plaintext in place of ciphertext fails step 2; a flipped byte fails step 3; both exit `7`; the restored object verifies again |
| `JourneyReferenceReachable` | a storage endpoint that lets the credential into another bucket fails the run, and the failing report is still posted |
| `JourneyReferenceRejectsTheCredential` | `InvalidAccessKeyId`, `SignatureDoesNotMatch`, `ExpiredToken`, `InvalidToken` each fail step 4 rather than reading as a scope refusal, and none asks for a security report |
| `JourneyNoReferenceLocker` | a control plane with no reference locker makes step 4 not applicable, and neither the report nor the push-hook line claims isolation |
| `BeforeAnyPush` | a profile that has pushed nothing: four NOT APPLICABLE steps, `OUTCOME: NOT YET VERIFIABLE`, exit `0`, nothing posted |
| `ExitCodes` | a passing run exits `0` and a failed one exits `7` (`exitcode.Safety`), not the `4` three documents used to claim |
| `JourneyBYO` | BYO over the memory backend with the passphrase: steps 1–3 pass, step 4 is not applicable, nothing is posted, and the `--json` document matches `byo-report.golden.json` |
| `HopCredentialsMakesTheByHandRecipeReal` | `rein hop credentials` mints and prints the live locker credential; a push afterwards still succeeds, and the S3 fake accepts only the newest mint |
| `HopCredentialsNeedsASignedInDevice` | it refuses like every other hosted command instead of printing an empty credential |

The unit checks behind them, all passing:

```
go test ./internal/verify/... -count=1 -v      # 25 tests, ok … 1.160s
```

## 3. `make test-race`

```
env -u REINSTATE_BACKEND -u … make test-race
CGO_ENABLED=1 GOTOOLCHAIN=go1.25.13 go test <FAST_PACKAGES> -race -count=1 -timeout=20m

--- FAIL: TestDaemonJourneyHop (1.96s)
    daemon_test.go:320: daemon status missing "daemon:   running (pid":
--- FAIL: TestMigrateJourneyLeaveHopForBYO (0.71s)
    migrate_test.go:307: restored session-locker (err=<nil>):
FAIL	github.com/HarjjotSinghh/reinstate/internal/cli	62.189s
```

**No `DATA RACE` block was printed.** The two failures are the ones this
branch already carried (see the verdict). `FAST_PACKAGES` excludes
`internal/doctest` and `internal/crypto`, so those two packages are still
not covered by `-race` on this bench; `internal/verify` and
`internal/cli`, which hold this ticket's work, are.

## 4. `go vet` and the cross-builds

```
go vet ./...                                   # exit 0, no output
GOOS=windows GOARCH=amd64 go build ./...       # exit 0
GOOS=darwin  GOARCH=amd64 go build ./...       # exit 0
GOOS=linux   GOARCH=amd64 go build ./...       # exit 0
gofmt -l .                                     # no output
```

Cross-*builds* only. Nothing was cross-*executed*:
`internal/crypto/passphrase_fd_unix.go` still has not run anywhere.

## 5. Full suite

```
env -u REINSTATE_BACKEND -u … go test ./...

--- FAIL: TestDaemonJourneyHop (1.40s)
--- FAIL: TestMigrateJourneyLeaveHopForBYO (0.60s)
FAIL	github.com/HarjjotSinghh/reinstate/internal/cli	28.404s
```

Every other package passes. Both failures reproduce on the branch point
and are fixed on `hop/main`; neither touches `rein sync verify`.

## 6. What this record does not cover

- **No physical journey.** One machine, no second device, no live R2, no
  real `hopd`. The credential-scope conclusion of step 4 rests on the S3
  fake's behaviour (which answers a foreign bucket `AccessDenied` only
  after the signature validates, as R2 does) and not on a real R2
  temporary credential being refused by a real foreign bucket.
- **No `-race` over `internal/crypto` or `internal/doctest`**, which
  `FAST_PACKAGES` excludes.
- **No fuzzing.** The repository has eight `Fuzz*` targets and none was
  run here.
- **Nothing was run against a tree containing both #11 and #12.** That
  has to be re-verified on the merge; the object-format page's
  key-generation sentence is scoped to this branch precisely because of
  it.
