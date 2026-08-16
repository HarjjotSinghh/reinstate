# Agent storage probe

**Status:** Phase 5 evidence contract · **Decided by:**
[ADR 0004](../adr/0004-universal-agent-coverage.md), decision 3

Before Reinstate ships a reader for an agent, someone must confirm on a real
machine where that agent actually keeps its sessions. Vendor documentation is
not sufficient: the Kimi Code CLI docs mirrors currently disagree with each
other about the data root.

The probe is how that confirmation is captured without ever copying a real
transcript into the repository.

---

## Surfaces

| Surface | Purpose |
| ------- | ------- |
| `rein doctor --agents` | Human-readable inventory of every catalog agent on this machine |
| `rein doctor --agents --json` | The committable evidence artifact |
| `scripts/testing/agent-storage-probe.sh` | Wrapper for contributors without a Go toolchain (macOS, Linux, WSL2) |
| `scripts/testing/agent-storage-probe.ps1` | Native Windows twin |

The wrappers exist so a tester on a machine with no Go installed can still
produce evidence from a release binary.

---

## What the probe emits

For each agent in the catalog, one record:

```json
{
  "schema": "AGENT-PROBE-V1",
  "generated_at": "2026-08-16T00:00:00Z",
  "platform": {"os": "darwin", "arch": "arm64", "device_class": "macos"},
  "reinstate_version": "0.5.0-rc.1",
  "agents": [
    {
      "key": "kimi",
      "display_name": "Kimi Code CLI",
      "declared_tier": "T0",
      "root_env": "KIMI_CODE_HOME",
      "root_env_set": false,
      "candidate_roots": [
        {"relative_to": "home", "suffix": ".kimi-code", "exists": true,  "marker_present": true},
        {"relative_to": "home", "suffix": ".kimi",      "exists": false, "marker_present": false}
      ],
      "resolved_root": {"relative_to": "home", "suffix": ".kimi-code"},
      "executable_on_path": true,
      "version_raw": "kimi 1.4.2",
      "tree": [
        {"path": "sessions", "kind": "dir", "children": 14},
        {"path": "sessions/*/", "kind": "dir", "children": 3, "sample_count": 14},
        {"path": "sessions/*/*/state.json", "kind": "file", "count": 41, "median_bytes": 812},
        {"path": "sessions/*/*/agents/main/wire.jsonl", "kind": "file", "count": 41, "median_bytes": 184320},
        {"path": "session_index.jsonl", "kind": "file", "count": 1, "median_bytes": 6144}
      ],
      "name_shapes": [
        {"path": "sessions/*", "shape": "wd_<32-hex>", "samples": 14},
        {"path": "sessions/*/*", "shape": "<uuid-v4>", "samples": 41}
      ],
      "first_line_keys": {
        "sessions/*/*/state.json": ["title", "createdAt", "updatedAt", "workDir"],
        "sessions/*/*/agents/main/wire.jsonl": ["type", "timestamp", "role"]
      }
    }
  ]
}
```

`first_line_keys` carries **JSON object keys only**, never values. That is
enough to write a fixture and a parser, and it cannot leak content.

---

## Redaction rules

These are hard requirements, verified by a test over the probe output, not by
the operator's care.

The probe **must never** emit:

1. transcript content, prompt text, assistant text, tool arguments, or tool
   results;
2. any JSON **value** from a session file — keys only;
3. absolute paths, home directory names, or usernames — paths are emitted
   relative to the resolved agent root, and roots are emitted as
   `{relative_to, suffix}` pairs;
4. file or directory names that are not shape-normalized — a UUID becomes
   `<uuid-v4>`, a hash becomes `<32-hex>`, a path slug becomes `<slug>`, and
   the operating-system account name becomes `<user>` wherever it appears
   inside a name;
5. repository names, branch names, or remote URLs;
6. environment variable **values**, only whether each is set;
7. anything from a path listed in the descriptor's `Excluded` set, including
   credential and cache subtrees.

Rule 4 covers the account name because vendors put it inside bucket directory
names: Kimi Code buckets a workspace as `wd_<user>_<hash>`. Nothing about an
account name looks like a UUID, a hash, or a slug, so the normalizer preserves
it verbatim unless it is removed explicitly. An absolute path is not the only
way an identity reaches a committed artifact.

The probe opens every file read-only, reads at most the first line of a
sampled file, and never writes, renames, or locks anything under an agent
root.

`make verify` runs the existing fixture secret scanner over
`docs/testing/results/agent-probes/`, so a probe artifact carrying a secret
fails CI.

---

## Running a probe

1. Use the agent normally for a while, in at least two different projects.
   A probe against an agent with zero sessions proves nothing.
2. Run the probe:

```bash
rein doctor --agents --json > probe.json
```

```powershell
rein doctor --agents --json | Out-File -Encoding utf8 probe.json
```

3. Read `probe.json` yourself before committing it. You are the last redaction
   check.
4. Commit it as
   `docs/testing/results/agent-probes/<YYYY-MM-DD>-<macos|windows|wsl>-<agent-key>.json`.

A T1 promotion needs a macOS probe and a native Windows probe. WSL2 is
additionally required when the agent runs there, because WSL2 and native
Windows are different devices with different trees.

---

## Turning a probe into a storage page

A probe answers four questions, and the agent's page in
[../session-storage/](../session-storage/) records the answers with the probe
cited:

| Question | Probe field |
| -------- | ----------- |
| Which root is real on this OS? | `resolved_root`, `candidate_roots[].marker_present` |
| What is the session path shape? | `tree[].path`, `name_shapes[]` |
| How is the project bucket derived? | `name_shapes` for the bucket directory |
| What record fields exist? | `first_line_keys` |

Rows the probe confirms move from `Unverified` to `Documented`. Rows the probe
**contradicts** are deleted, and the contradiction is noted on the page —
that is the probe's most valuable output.

Rows the probe cannot reach stay `Unverified`, and the agent's tier is capped
accordingly.

### Budgets are per agent

`--agent-timeout` bounds a single agent, not the whole run, and defaults to ten
seconds. The probe spawns one `--version` subprocess per installed agent, so on
the machine this phase targets — a dozen harnesses installed side by side — a
single whole-run budget guarantees failure exactly where evidence matters most.

An agent that exceeds its budget is recorded with `timed_out: true` and
whatever was gathered before the deadline. Its partial fields are not a
negative finding, and the surrounding agents are unaffected. Only the operator
interrupting the command aborts the run.

### Why a candidate root can exist and still not resolve

Discovery is marker-gated: a declared root only becomes `resolved_root` when the
descriptor's `Storage.Marker` is present inside it. A bare `~/.<agent>` directory
is not evidence of an installation, because skill installers, dotfile managers,
and other agents routinely plant one. Every descriptor that declares a root must
declare a marker, and registration panics otherwise.

So `exists: true` with `marker_present: false` is a meaningful result, not a
failure: the directory is there but the layout the descriptor expects is not.
Either the agent has never been run, or the declared marker is wrong — and the
`tree` of a sibling probe from a machine where the agent *is* installed settles
which.

For the same reason the `installed` column of `rein doctor --agents` reports
only whether the executable is on `PATH`. Root presence is the separate `root`
column. An explicit `RootEnv` or fixture root bypasses the marker gate, because
pointing the probe at a directory is an instruction rather than a guess.

---

## What the probe does not do

It does not install agents, launch them, log in, or make network calls. It does
not decide a tier. It is evidence; the tier decision is
[../agent-support-tiers.md](../agent-support-tiers.md).
