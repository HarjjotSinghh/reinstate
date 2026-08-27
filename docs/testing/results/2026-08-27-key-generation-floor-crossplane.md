# The account key generation floor, client against control plane — 2026-08-27

Windows 11 bench (NT 10.0.26200), Go 1.26.1, native amd64. Public client at
the round-three integration tree (`#11` then `#12` merged); control plane
`hopd` built from the matching private integration tree at `e571b61`.

## Why this run exists

The floor is the one thing in tickets #11 and #12 that neither repository
can exercise. The client half decides what to refuse; the control plane half
decides what number to serve. Every test on either side talks to a fake of
the other — the CLI journeys drive a fake control plane written in
`internal/cli`, and `hopd`'s tests drive its store directly — so a
wire-shape mistake between them passes both suites and fails on the first
real device. Round three coordinated the shape by reading the other
repository; this is the run that checks it.

## What ran

`internal/cli/keygeneration_crossplane_test.go`, built with
`-tags hopacceptance` and skipped unless `REINSTATE_HOPD_BIN` names a `hopd`
binary. Two journeys, both driving the real `rein` code in process against
the real `hopd` binary over HTTP:

```
REINSTATE_HOPD_BIN=<hopd.exe> go test -tags hopacceptance ./internal/cli/ \
  -run 'TestKeyGenerationFloor.*RealHopd' -v -count=1
```

`hopd` ran with `HOPD_STORAGE=fake`, `HOPD_EMAIL_SENDER=log`, a throwaway
SQLite database and a free loopback port. Sign-in was the real device
authorization flow: the CLI opened a session and polled, the magic link was
read out of `hopd`'s own log and POSTed the way a person clicking "Approve
this device?" does.

The bucket was stood in for by `REINSTATE_BACKEND=memory` with one
`REINSTATE_MEMORY_BACKEND_DIR` shared between the three device homes. That
seam short-circuits the S3 client and nothing else: `confirmKeyGenerationFloor`
keys off `cfg.Storage.Type`, so every command still asked the control plane
for the floor. Writing into that directory directly is what stands in for a
revoked device using the locker credential it was already given.

## Journey 1 — the lagging device, against the real control plane

```
A: rein login / init --hop / account init / push --all
B: rein login / init --hop / account recover
C: rein login / init --hop / account recover / account status
   (C records key generation 1 and the floor hopd served then, which is 0)
[the locker is copied aside — the genuine generation-1 keyring]
A: rein devices revoke <B>     → keyring generation 2, hopd's floor → 2
[the genuine generation-1 keyring is written back over the current one]
C: push / pull / sync verify / account status / devices
```

Observed, all of it:

- `GET /v1/account/key-generation` answered `0` before the revocation and
  `2` after it, read with a device token over the real route.
- The revocation changed the keyring object; the restored copy parses and
  every signature in it verifies, which is what makes this different from
  planting a forgery.
- C — which had run nothing since generation 1 — refused on `push`, `pull`
  and `sync verify` with exit 7, each naming the control plane, and reported
  the refusal without exiting on it in `account status` and `devices`.
  Nothing was written.
- A, which saw the rollover itself, refused the same object from its own
  record.

`devices approve` is in the same class and is not driven here: it answers
"no pending pairing requests" before it reads a keyring, and opening one
needs a fourth device sitting in an interactive poll.
`TestControlPlaneFloorReachesADeviceThatLagsBehind` covers it against the
in-package fake.

**Falsified.** With the two floor lines deleted from `keyringAnchor.check`,
the same journey shows the hole exactly as the round-two re-attack described
it: `push` exits 0 and writes a snapshot, `pull` exits 0, `account status`
reports "key generation 1", `devices` lists the revoked device without
complaint, and `rein sync verify` prints a full `OUTCOME: PASS` report and
posts it. The revoked device's generation-1 root key opens what C wrote.

## Journey 2 — a control plane that does not carry the floor

The same `hopd`, behind a proxy that answers `404` to
`/v1/account/key-generation` and passes everything else through, which is a
deployment older than the route.

- The floor route was asked for four times and answered `404` every time, so
  the fallback path is the one that ran.
- `rein devices revoke` still succeeded and still said, on stderr, that the
  control plane carries no floor — rather than reporting a protection it did
  not get.
- The documented residual reproduced: a device that has never confirmed a
  floor read the restored generation-1 keyring as current.
- The device that *had* read generation 2 still refused it, on its own
  record. That is what the residual leaves in place.

## What this run does not establish

- **Live R2.** The locker was a directory and `hopd`'s storage was its
  in-memory fake. Nothing here observed a real R2 temporary credential, a
  real bucket-scope refusal, or R2's answer for a read of a missing bucket.
- **A real deployment.** `hopd` ran as a local process against SQLite with
  the log email sender. Nothing here exercised Cloudflare, TLS termination,
  the Caddy front, or a restart under load.
- **Two physical devices.** Three device homes on one machine, with device
  tokens in per-process memory stores rather than in Windows Credential
  Manager, which holds one entry per user and could not have held three.
- **darwin or linux.** Cross-builds ran; cross-execution did not.
- **The credential window itself.** Writing into the shared directory is a
  stand-in for a revoked device's still-valid credential. That the credential
  really does keep working against R2 for the rest of its hour is a property
  of R2 and is stated, not demonstrated.

## Bench gates on the merged trees

Public: full suite, `make test-race`, `go vet`, `gofmt` (clean), and the
windows, darwin and linux amd64 cross-builds — all clean.
`TestDaemonJourneyHop` and `TestMigrateJourneyLeaveHopForBYO`, which fail on
both ticket branches, pass here.

Private: full suite, `go test -race ./...`, `go vet`, and the same three
cross-builds — all clean. `gofmt -l` lists files in that repository because
it is checked out with CRLF line endings; with the endings normalised it is
clean, and it is equally "dirty" on `main`.

Website: `npx vitest run` reports 8 files / 13 tests failing on this tree and
**the same 8 files / 13 tests failing on `hop/main`** — Windows file locking
in the waitlist tests, and gates that want a `.vercel/project.json` and other
deploy-time inputs this bench does not have. Not this round's, and unchanged
by it. `src/lib/security-section.test.ts`, which this round edits, passes.
