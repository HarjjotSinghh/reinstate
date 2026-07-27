# Reinstate Phase 1 RC5 — Windows Device B acceptance

**Date:** 2026-07-27
**Evidence owner:** native Windows Device B
**Release:** `v0.1.0-rc.5`
**Peeled tag commit:** `b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065`
**Verdict:** **FAIL — stopped during W1**

Stock RC5 could not initialize the additional device with the confirmed correct
storage coordinates. Its remote-manifest `HeadObject` probe returned HTTP
`400 Bad Request` even though the remote manifest already existed. This is a
release-blocking W1 failure.

Execution stopped at that gate. The remaining W1 checks and all W2/W3 checks
are `NOT TESTED`, never `PASS`.

No Reinstate product code was modified. This report is the only repository
change.

## 1. Evidence boundary

This report uses only contemporaneous native-PowerShell output captured while
the installed stock RC5 binary was under test.

The evidence does not contain secret inputs, storage-coordinate values, remote
object names, session contents, encrypted payloads, or private absolute paths.
The private project path in the recorded command is replaced by an explicit
redaction.

The numeric process exit code was not printed or captured when the failure
occurred. It is recorded as `NOT CAPTURED`; it is not inferred from source code
and the failed acceptance step was not rerun.

## 2. Release identity and pre-init evidence

The installed command resolved through the normal user-local Windows install
location represented as:

```text
%LOCALAPPDATA%\Programs\Reinstate\bin\rein.exe
```

Stock release identity:

```json
{
  "commit": "b4ebd8d",
  "name": "reinstate",
  "version": "0.1.0-rc.5"
}
```

Pre-init `rein setup check` reported:

```text
version: 0.1.0-rc.5
platform: windows-amd64
config: config missing
device: windows-amd64
agent.claude: SUPPORTED (2.1.220)
agent.codex: SUPPORTED (0.145.0)
keyring: OS keyring provider reachable
exit=3
```

The missing configuration was the expected pre-init state. The stock binary,
platform, adapters, and keyring provider were recognized.

The public-bootstrap-only gate was later explicitly waived. RC5 installation
used the separately inspected exact-tag canonical installer whose recorded
SHA-256 matched:

```text
ce46d3a22d4d9349c7e6847ed65b5e8ff93b51e7f035ad3ff93b7dc19d2f1232
```

Therefore the public Windows bootstrap and complete installer idempotency/PATH
row are not claimed as fully passed by this report.

## 3. W1 correct-coordinate additional-device initialization — FAIL

The operator confirmed the correct storage coordinates were entered separately
at Reinstate's private prompts. Their values and all secret inputs are omitted.

Exact stock-RC5 command structure, with only the prohibited private absolute
project path redacted:

```powershell
rein init --profile-id 451177b7-9eda-4e49-b74d-239773916f77 --project 'local/reinstate-phase1-acceptance-rc5=<ABSOLUTE_PRIVATE_WINDOWS_PROJECT_PATH>'
```

Actual numeric exit code:

```text
NOT CAPTURED
```

Sanitized contemporaneous output:

```text
[private prompt values omitted]
remote profile manifest probe failed: operation error S3: HeadObject, https response error StatusCode: 400, RequestID: , HostID: , api error BadRequest: Bad Request
```

Observed result:

```text
correct_coordinate_additional_device_init=FAIL
remote_manifest_existed=true
config_initialized=false
head_object_http_status=400
```

The shell remained in the uninitialized-home state after the command. No
successful initialization message was emitted.

## 4. Post-RC5 development validation — EXCLUDED

PR
[#26](https://github.com/HarjjotSinghh/reinstate/pull/26) changed the
additional-device probe after the RC5 tag. A later initialization and restore
using that post-RC5 development code succeeded.

That result is useful development validation, but it is not stock
`v0.1.0-rc.5` acceptance evidence. It is explicitly excluded from this
verdict, all gate counts, and all W1/W2/W3 results.

The next release candidate must contain the reviewed fix and run a fresh
physical-device evidence chain. RC5 must not be retroactively relabeled as
passing.

## 5. Blocked work

### 5.1 Remaining W1 — NOT TESTED

The following stock-RC5 checks were blocked by the failed initialization and
were not executed as acceptance evidence:

- post-init `rein setup check`;
- post-init `rein doctor --self-test`;
- default re-init refusal and no-mutation proof;
- deliberate wrong-secret refusal and no-mutation proof;
- correct-secret status with exactly two selected sessions;
- missing-remote-manifest strict-status probe;
- scoped Claude and Codex dry-runs;
- scoped Claude and Codex pulls;
- mapped Windows discovery;
- exact-ID Claude and Codex vendor resume; and
- visual confirmation of the two selected test sessions.

`WINDOWS-RC5-W1-PASS` was not emitted.

### 5.2 W2 — NOT TESTED

No active-agent overwrite test, backup replacement test, Windows update, or
Windows-to-Mac push was executed.

### 5.3 W3 — NOT TESTED

No divergence, conflict, conflict record, or `--keep-both` test was executed.

No M2, M3, or M4 continuation is authorized for RC5.

## 6. Mandatory-gate reconciliation

| # | Gate | Result | Evidence |
| ---: | --- | --- | --- |
| 1 | Public installer returns 200 and installs RC5 on Mac | NOT TESTED | Not evaluated by this Windows report |
| 2 | Public installer returns 200 and installs RC5 on Windows | PARTIAL | Exact-tag canonical installer verified; public-bootstrap-only gate waived |
| 3 | Both installers are idempotent and PATH-safe | NOT TESTED | Complete two-device row not captured |
| 4 | Pre-init missing-config failure is accurate | PASS | Exit `3`; only configuration missing |
| 5 | Post-init setup check and self-test pass on both devices | NOT TESTED | Windows blocked at initialization |
| 6 | Claude setup prompt completes on the Mac | NOT TESTED | Not evaluated by this Windows report |
| 7 | Codex setup prompt completes on Windows | FAIL | Correct-coordinate stock-RC5 initialization returned `HeadObject` HTTP 400 |
| 8 | Only two selected test sessions reach remote storage | NOT TESTED | No stock-RC5 Windows status evidence |
| 9 | Remote acceptance data satisfies the confidentiality check | NOT TESTED | Not evaluated by this Windows report |
| 10 | Wrong secret fails without mutation | NOT TESTED | Blocked by initialization failure |
| 11 | Claude Mac-to-Windows resume succeeds | NOT TESTED | Blocked by initialization failure |
| 12 | Codex Mac-to-Windows resume succeeds | NOT TESTED | Post-RC5 development result excluded |
| 13 | Active-agent overwrite is refused | NOT TESTED | W2 not executed |
| 14 | Existing Windows target is backed up before restore | NOT TESTED | W2 not executed |
| 15 | Claude Windows-to-Mac resume succeeds | NOT TESTED | W2 not executed |
| 16 | Codex Windows-to-Mac resume succeeds | NOT TESTED | W2 not executed |
| 17 | Existing Mac targets are backed up before restore | NOT TESTED | W2 not executed |
| 18 | Unchanged pushes skip without new snapshots | NOT TESTED | W2 not executed |
| 19 | Divergence records a conflict without overwrite | NOT TESTED | W3 not executed |
| 20 | `--keep-both` preserves both branches | NOT TESTED | W3 not executed |
| 21 | All required GitHub checks are green | NOT TESTED | Acceptance stopped during W1 |

Totals:

```text
PASS=1
PARTIAL=1
FAIL=1
NOT_TESTED=18
ALL_21_PASSED=false
```

## 7. Release-blocking finding

### `WINDOWS-RC5-W1-FAIL` — stock RC5 remote-manifest probe rejected valid coordinates

Severity: **release blocking**

The stock RC5 additional-device initialization path used `HeadObject` as its
remote-manifest gate. Against the existing remote manifest and confirmed
correct coordinates, the provider returned HTTP `400 Bad Request`.
Initialization did not complete, so downstream restore safety and same-vendor
resume could not be tested on the released artifact.

PR #26 is post-tag development evidence only. RC6 or a later candidate must
contain the fix and repeat the complete acceptance chain from fresh state.

## 8. Sanitized stopping handoff

```text
WINDOWS-RC5-W1-FAIL
release=v0.1.0-rc.5
tag_commit=b4ebd8dcf8b47e7dcbbc0fc40c4ef9adf9ea5065
device=windows-amd64
canonical_project_id=local/reinstate-phase1-acceptance-rc5
correct_coordinate_additional_device_init=FAIL
failed_operation=HeadObject
failed_http_status=400
failed_error=BadRequest
actual_exit_code=NOT_CAPTURED
config_initialized=false
w1_remainder=NOT_TESTED
w2=NOT_TESTED
w3=NOT_TESTED
post_rc5_pr_26_validation=EXCLUDED
overall_verdict=FAIL
pass_count=1
partial_count=1
fail_count=1
not_tested_count=18
all_21_passed=false
report_path=docs/testing/results/2026-07-27-windows-phase1-rc5.md
END-WINDOWS-RC5-W1-FAIL
```

Phase 1 RC5 acceptance is **FAIL**. All 21 mandatory rows did not pass.
