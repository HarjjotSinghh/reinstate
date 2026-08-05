# Phase 3 CLI performance and issue #96

This document records the reproducible development baseline for
[issue #96](https://github.com/HarjjotSinghh/reinstate/issues/96). It does not
replace tagged-artifact measurements on the two supported release platforms.
The RC1 dispatch requires fresh Apple Silicon macOS and native Windows x64
measurements before stable promotion.

## Optimization

The measured long-session bottleneck was prompt-search accumulation in the
Claude, Codex, and Gemini metadata readers. Each message previously rebuilt,
re-sanitized, and copied the entire accumulated prefix. For `n` similarly sized
messages that performed quadratic `O(n²)` character work and allocation.

The Phase 3 implementation uses an append-only `strings.Builder`, sanitizes
each new bounded value exactly once, and stops accepting input at the existing
256 KiB search-text boundary. Work is now amortized `O(n)` up to that fixed
bound. The final private search text remains UTF-8-valid, sanitized, bounded,
and semantically identical after the canonical whitespace normalization. No
source scan, freshness check, record, or privacy field is skipped or cached.

## Reproducible development results

Environment: Apple M4 Pro, `darwin/arm64`, Go `1.25.12`, source benchmark,
2026-08-05. Every corpus is generated from controlled synthetic metadata; no
real transcript or private path is read or emitted. Times are wall-clock Go
benchmark times. These are development measurements, not installed-artifact or
native-Windows claims.

### Before and after: long-session accumulator

The 100- and 1,000-message values are medians of three one-iteration samples.
The 10,000-message baseline is one sample because it was already prohibitively
slow; the optimized value is the median of three samples.

| Messages | Before | After | Before allocated bytes | After allocated bytes |
| ---: | ---: | ---: | ---: | ---: |
| 100 | 3.61 ms | 69.9 us | 1.43 MB | 41.5 KB |
| 1,000 | 349.8 ms | 816 us | 212.9 MB | 446 KB |
| 10,000 | 21.18 s | 2.89 ms | 13.77 GB | 1.81 MB |

The 10,000-message path is about 7,300 times faster in this controlled sample.
An allocation ceiling regression test rejects restoration of the quadratic
algorithm without relying on a fragile wall-clock threshold.

### Current layer measurements

| Layer or command | Synthetic corpus | Typical result |
| --- | ---: | ---: |
| Claude source scan | 1,000 one-message files | 28.6 ms |
| Claude source scan | one 10,000-message file | 24.1 ms |
| Open private SQLite index | 1,000 records | 1.12 ms |
| Query and decode matching rows | 1,000 records | 9.42 ms |
| No-change derived-index replacement | 1,000 records | 5.41 ms |
| `version --json` orchestration | no index | 46–77 us warm |
| `--help` orchestration | no index | about 60 us warm |
| `sessions --limit 1000 --json` | 1,000 records | 21.0 ms warm |
| `search ... --limit 1000 --json` | 1,000 records | 21.5 ms warm |
| `inspect ... --json` | 1,000 records | 8.71 ms warm |
| resume dry-run JSON | 1,000 records | 9.05 ms warm |
| fork dry-run JSON | 1,000 records | 8.76 ms warm |
| Shell-free child creation/wait | immediate controlled child | 11.1 ms |

The command benchmarks run Cobra, refresh, SQLite, rendering, and deterministic
preflight orchestration in-process. They deliberately exclude process startup,
physical antivirus behavior and real vendor-file scanning. Source benchmarks
measure the vendor reader separately. A dedicated shell-free child benchmark
uses the production launch runner with verified executable/workspace identities
and an immediate controlled child; it separates child creation/wait cost from
an interactive vendor session. The tagged artifact matrix measures the complete
installed process on real devices.

## Commands

Run the portable benchmark suite unchanged on macOS, Linux, or native Windows:

```bash
GOTOOLCHAIN=go1.25.12 go test ./internal/sessionindex \
  -run '^$' \
  -bench 'Benchmark(BoundedPromptAccumulator|ClaudeLargeCorpusRefresh|ClaudeLongSessionRefresh|LargeCorpusIndexLayers|ExecLaunchRunnerImmediateChild)' \
  -benchmem -count=3

GOTOOLCHAIN=go1.25.12 go test ./internal/cli \
  -run '^$' -bench BenchmarkLocalMetadataCommands -benchmem -count=3
```

For a quick profile of the long-session reader:

```bash
GOTOOLCHAIN=go1.25.12 go test ./internal/sessionindex \
  -run '^$' -bench BenchmarkClaudeLongSessionRefresh \
  -cpuprofile cpu.out -memprofile mem.out
go tool pprof -top cpu.out
go tool pprof -top -alloc_space mem.out
```

Generated profile files are local evidence and must not be committed.

For tagged installed-artifact evidence, run the checked-in harness from an
exact clean tagged checkout. `ABS_NEW_EVIDENCE_ROOT` must not already exist;
the alias paths and every curated PATH directory must be absolute, canonical,
physical, and outside the source/evidence roots. The curated PATH must resolve
trusted Git, Claude, and Codex executables but deliberately omit OpenCode:

```text
go run ./scripts/testing/phase3perf run --root ABS_NEW_EVIDENCE_ROOT --rein ABS_INSTALLED_REIN --reinstate ABS_INSTALLED_REINSTATE --source-root ABS_TAGGED_SOURCE --expected-commit TEST_COMMIT --expected-version 0.3.0-rc.1 --path CURATED_PATH
```

The command generates exact normal and 1,000-record corpora, verifies the
embedded canonical digest and root-bound materialized digests, creates frozen
clean Git workspaces, measures version/help startup plus all seven corpus
commands, preserves cold index families, and emits only a private aggregate
`results.json`. It fails closed on source changes, untrusted paths, ambient
capabilities, unexpected OpenCode visibility, output/schema differences,
timeouts, source mutation, or workspace drift.

## Platform status and remaining evidence

| Platform | Development benchmark status | Tagged installed-artifact status |
| --- | --- | --- |
| Apple Silicon macOS | Measured above | Required after RC1 publication |
| Native Windows x64 | Portable suite provided; not measured in this report | Required after RC1 publication |
| Linux | Portable suite provided; local physical result unavailable | Unsupported/unverified optional evidence |
| Intel macOS / WSL2 | Not measured | Unsupported/unverified optional evidence |

The original Windows 20–30 second operator observation is not overwritten by a
macOS source benchmark. Issue #96 is only physically resolved for the release
line when the installed RC artifact completes the fixed normal and 1,000-record
cold/warm matrix on Apple Silicon and native Windows without a timeout, an
unbounded-growth signal, or a comparable same-host regression.
