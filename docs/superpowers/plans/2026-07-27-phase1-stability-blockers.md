# Phase 1 Stability Blockers Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development
> (if subagents are available and authorized) or superpowers:executing-plans to
> implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close every currently recorded actionable Phase 1 product blocker,
including the remaining POSIX installer hang, while preserving safe replacement
consent and leaving a fully verified local RC5 candidate branch.

**Architecture:** Keep the existing F1-F3 CLI/sync fixes unchanged. Bound the
canonical POSIX installer's `/dev/tty` confirmation read with a validated
timeout; if the active `/bin/sh` lacks timed reads, refuse immediately with the
existing explicit environment-variable escape hatch. Exercise the real
installer through an idle pseudo-terminal so the regression test reproduces the
Mac acceptance failure rather than merely checking script text.

**Tech Stack:** Go 1.25.12, POSIX shell, Go `testing`, `os/exec`, native
`script(1)` pseudo-terminal utility, Make verification gates.

**Execution constraint:** Repository instructions prohibit subagents unless the
user explicitly requests them. Execute this plan in the current agent session.

---

## Blocker inventory

- F1: fixed in `3f2cbf4`; default re-init refuses and `--force` backs up config
  plus state.
- F2: fixed in `3f2cbf4`; joined/established profiles reject a missing remote
  manifest.
- F3: fixed in `3f2cbf4`; `init --profile-id` verifies manifest existence before
  saving config.
- N1: still open; an unattended but readable `/dev/tty` can block
  `scripts/install.sh` forever.
- Live GitHub issues: none open as of the audit.
- RC2/RC3 report findings are either fixed by RC3/RC4, vendor/operator
  conditions, or non-product observations; none adds a current code blocker.

### Task 1: Reproduce N1 through the real installer

**Files:**

- Modify: `internal/doctest/install_contract_test.go`
- Test: `internal/doctest/install_contract_test.go`

- [ ] **Step 1: Add the idle-TTY contract case**

  Add a helper that runs `scripts/install.sh` under the platform's `script(1)`
  pseudo-terminal, keeps its input pipe open without sending bytes, and returns
  whether the command exceeded a bounded test deadline.

  Run an upgrade with:

  ```text
  REINSTATE_CONFIRM_TIMEOUT_SECONDS=1
  ```

  Require the command to finish before the test deadline, refuse replacement,
  preserve the installed binary, and print a timeout/refusal message.

- [ ] **Step 2: Add timeout-value validation coverage**

  Run an upgrade with:

  ```text
  REINSTATE_CONFIRM_TIMEOUT_SECONDS=invalid
  ```

  Require a non-zero exit, an actionable validation message, and no replacement.

- [ ] **Step 3: Verify RED**

  Run:

  ```bash
  go test ./internal/doctest -run TestInstallerConsumesGoReleaserAssetContract -count=1 -v
  ```

  Expected: FAIL because the current installer ignores the timeout variable and
  the idle-TTY run exceeds the test deadline.

### Task 2: Implement bounded, fail-closed replacement confirmation

**Files:**

- Modify: `scripts/install.sh`
- Test: `internal/doctest/install_contract_test.go`

- [ ] **Step 1: Validate the timeout**

  In `confirm_replace`, read
  `REINSTATE_CONFIRM_TIMEOUT_SECONDS` with a 30-second default. Accept only
  integer values from 1 through 300. Invalid values must refuse replacement
  before reading the TTY.

- [ ] **Step 2: Detect timed-read support**

  Probe the active shell's `read -t` behavior against `/dev/null`. A normal EOF
  status proves support; an unsupported-option status must fail closed
  immediately and direct the operator to
  `REINSTATE_CONFIRM_REPLACE=1` after reviewing the version change.

- [ ] **Step 3: Bound the real TTY read**

  Use the validated timeout for the controlling-terminal read. On timeout or
  closed input, print a newline plus an actionable refusal and return non-zero.
  Preserve the existing accepted answers and the explicit
  `REINSTATE_CONFIRM_REPLACE=1` bypass.

- [ ] **Step 4: Verify GREEN**

  Run:

  ```bash
  go test ./internal/doctest -run TestInstallerConsumesGoReleaserAssetContract -count=1 -v
  ```

  Expected: PASS, including the idle-TTY and invalid-timeout cases.

### Task 3: Document the stable installer contract

**Files:**

- Modify: `README.md`
- Modify: `docs/getting-started.md`
- Modify: `CHANGELOG.md`
- Modify: `docs/testing/results/2026-07-26-phase1-rc5-blocker-fix-handoff.md`
- Test: `internal/doctest/install_contract_test.go`

- [ ] **Step 1: Update user-facing installation guidance**

  Document the 30-second default, the 1-300 second override, and the
  `REINSTATE_CONFIRM_REPLACE=1` explicit approval path. State that unsupported
  timed-read shells refuse immediately rather than waiting forever.

- [ ] **Step 2: Update release evidence**

  Add N1 to `[Unreleased]` as fixed and revise the handoff so no known local
  product blocker remains open. Keep Phase 1 explicitly not stable until a
  fresh physical Mac/Windows RC5 run passes every mandatory row.

- [ ] **Step 3: Run documentation checks**

  Run:

  ```bash
  make docs-check
  git diff --check
  ```

  Expected: PASS.

### Task 4: Release-grade verification and local handoff

**Files:**

- Modify only if verification exposes a real regression.

- [ ] **Step 1: Run focused and full verification**

  ```bash
  go test ./internal/doctest -count=1
  make verify
  ```

  Expected: PASS with no reachable vulnerability.

- [ ] **Step 2: Cross-build every release target**

  Compile Darwin amd64/arm64, Windows amd64, and Linux amd64/arm64 with
  `CGO_ENABLED=0` and Go 1.25.12.

- [ ] **Step 3: Run isolated native Mac acceptance-style checks**

  Re-run the F1-F3 synthetic binary checks and add the N1 idle-TTY installer
  check. Do not touch real Reinstate homes, real agent trees, Keychain, or
  remote storage.

- [ ] **Step 4: Review and commit locally**

  Inspect the exact diff and secret scan, then create focused Conventional
  Commits. Leave the branch clean and ahead of `origin/main`; do not push,
  merge, tag, publish, or create RC5 artifacts.

- [ ] **Step 5: Record the remaining truth**

  Update the handoff with commit IDs and fresh evidence. The remaining gate is
  external acceptance: protected-main CI, signed RC5 artifacts, and the complete
  fresh-profile two-device run.
