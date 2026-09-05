# W2 — Code fixes

**Executor:** one Sonnet agent. **Verifier:** one Sonnet agent (reproduce
each defect before the fix, confirm the fix, try to break it).
**Branch:** `v060/w2-fixes` from `release/v0.6.0-rc.1`.

## T-201 — Preflight shared-deadline flake

`internal/preflight/performance_adversarial_test.go`,
`TestVerifyHonorsParentCancellationAndSharedDeadline/shared_deadline`: with
`Timeout = 25ms` the test asserts `Verify` returns within 500 ms. Under a
parallel `go test ./...` on this host it took 1.19 s; alone it takes 0.4 s.
Widen the bound the way 2bc0367f widened `TestHugeTreeFinishes` (2 s → 8 s):
wide enough for a loaded machine, tight enough to catch an unbounded wait.
Keep the 25 ms timeout, keep the `DecisionBlocked` / exit-code assertions,
and write the reason in a comment beside the bound. Run the package ten
times under load (`go test ./... -count=1` in parallel with it) and report
the maximum observed.

## T-202 — Unreachable control plane

Reproduce: `REINSTATE_HOP_URL=http://127.0.0.1:1 rein login --email a@b.c
--no-browser` and `rein whoami` with a token bound to an unresolvable host.
Today the error is whatever `net/http` says. Make `internal/hop` classify
DNS failure, connection refused, and TLS handshake failure into one error
type, and make `rein login` / `rein whoami` print exactly one line of this
shape (final wording is yours; keep it timeless — no "not open yet"):

```text
could not reach the Reinstate Hop control plane at https://hop.reinstate.dev: <cause>
If you are not enrolled in Reinstate Hop, see https://reinstate.dev/docs/hop. To use another control plane, set REINSTATE_HOP_URL or [hop] url in config.toml.
```

The `--json` form carries `error.kind = "control_plane_unreachable"` and the
URL. Exit code stays what the command uses for network failure today (find
it; do not invent one). Test with an `httptest` server that is closed before
the call, and with a dialer that returns a DNS error. Add the message to
`docs/hop.md` ("Choosing the control plane") and `docs/troubleshooting.md`.

## T-203 — Cross-OS builds and tidiness

`GOOS=darwin`, `GOOS=linux`, `GOOS=windows` × `go build ./...` and `go vet`
clean; `go mod tidy -diff` empty. Fix build-tag or `//go:build` slips; do not
add dependencies (ask the coordinator).

## T-204 — Sweep

`grep -rn 'TODO\|FIXME\|XXX' internal cmd scripts --include=*.go | grep -i
'v0.6\|hop\|windows'` and `grep -rn 't.Skip' internal --include=*_test.go |
grep -i windows`. Every hit either has a comment naming the reason and an
issue, or gets one, or is resolved. Report the list.

## Done when

Gate 1 including race on `internal/...`; the verifier's report; changelog
bullets for T-201 and T-202 handed to W1 in the PR description.
