# Reinstate One-Line Installers Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish checksum-pinned one-line installers at `reinstate.dev` and provide a strict MacBook/Windows Phase 1 acceptance runbook.

**Architecture:** Two static website bootstraps download and hash-check the canonical installers from the exact `v0.1.0-rc.2` tag. The canonical installers continue to own platform selection, release checksum verification, replacement safety, and binary installation; the bootstraps add user-level PATH quality of life and print `rein init` without launching it.

**Tech Stack:** POSIX `sh`, PowerShell 5.1+, Go 1.25 doctests, Astro 7, Vercel static assets, GitHub Actions, Markdown.

---

## File Map

### Create

- `website/public/install.sh` — public macOS/POSIX bootstrap.
- `website/public/install.ps1` — public native Windows bootstrap.
- `internal/doctest/bootstrap_install_contract_test.go` — static and
  platform-specific behavioral contracts for both bootstraps.
- `docs/testing/phase-1-mac-windows-acceptance.md` — authoritative two-device
  manual acceptance runbook.

### Modify

- `.github/workflows/ci.yml` — run website tests/build and assert both public
  files survive the Astro production build.
- `README.md` — show both one-line commands and current RC status.
- `docs/getting-started.md` — replace placeholder release installation with
  public commands plus inspect-before-execute alternatives.
- `website/src/content/docs/getting-started.md` — make public website guidance
  match the implemented CLI and release.
- `docs/README.md` — link the acceptance runbook.
- `website/src/content/docs/README.md` — link the acceptance runbook in the
  website documentation index.
- `scripts/install.sh` — add the stable public entrypoint to the usage comment
  while retaining the exact-tag canonical invocation.
- `scripts/install.ps1` — add the stable public entrypoint to the usage comment
  while retaining the exact-tag canonical invocation.

## Chunk 1: Executable Installer Contract

### Task 1: Add failing public-bootstrap contract tests

**Files:**

- Create: `internal/doctest/bootstrap_install_contract_test.go`
- Reference: `internal/doctest/install_contract_test.go`
- Reference: `website/public/install.sh`
- Reference: `website/public/install.ps1`

- [ ] **Step 1: Write the static contract test**

Add `TestPublicBootstrapStaticContract`. It must read both public scripts and
assert:

```go
required := []string{
    "v0.1.0-rc.2",
    "scripts/install.",
    "SHA256",
    "rein init",
}
forbidden := []string{
    "releases/latest",
    "api.github.com/repos",
}
```

It must additionally assert that neither script contains an executable
invocation of `rein init`; the words may appear only in completion output.

- [ ] **Step 2: Write the POSIX behavior test**

Add `TestPOSIXPublicBootstrapContract`, skipped outside Darwin/Linux. Use
`httptest.Server` to serve a synthetic canonical installer at:

```text
/v0.1.0-rc.2/scripts/install.sh
```

The synthetic installer must assert that `REINSTATE_VERSION` equals
`v0.1.0-rc.2`, create executable `reinstate` and `rein` files under
`INSTALL_DIR`, and exit successfully.

Run the public bootstrap with isolated `HOME`, `INSTALL_DIR`, `SHELL`, and
`PATH`; point `REINSTATE_BOOTSTRAP_ORIGIN` and
`REINSTATE_BOOTSTRAP_INSTALLER_SHA256` at the fixture. Assert:

- the exact tagged path was requested;
- both commands were installed;
- `~/.zshrc` gained exactly one safe PATH export;
- a second run does not duplicate the entry;
- output prints the absolute `rein init` command;
- no `init` process was launched; and
- `REINSTATE_SKIP_PATH_UPDATE=1` leaves a fresh profile untouched.

- [ ] **Step 3: Write hash-mismatch failure coverage**

Serve a fixture whose digest does not equal the configured expected digest.
Assert non-zero exit, a checksum-mismatch message, no installed binary, and no
shell-profile mutation.

- [ ] **Step 4: Write the native Windows behavior test**

Add `TestWindowsPublicBootstrapContract`, skipped outside Windows or when
`powershell.exe` is missing. Serve a synthetic canonical PowerShell installer
at:

```text
/v0.1.0-rc.2/scripts/install.ps1
```

Run the bootstrap twice in one PowerShell process with
`REINSTATE_BOOTSTRAP_PATH_SCOPE=Process`. Assert:

- the exact tag path was requested;
- both `.exe` fixture files were installed;
- the exact release version reached the canonical installer;
- PATH was added once and compared case-insensitively;
- output prints `Next: rein init`; and
- no interactive initialization ran.

Add a mismatch case that proves no binary or PATH update occurs.

- [ ] **Step 5: Run the focused tests and confirm RED**

Run:

```sh
GOTOOLCHAIN=go1.25.12 go test ./internal/doctest \
  -run 'Test(PublicBootstrapStaticContract|POSIXPublicBootstrapContract|WindowsPublicBootstrapContract)' \
  -count=1
```

Expected: FAIL because `website/public/install.sh` and
`website/public/install.ps1` do not exist.

### Task 2: Implement the POSIX bootstrap

**Files:**

- Create: `website/public/install.sh`
- Test: `internal/doctest/bootstrap_install_contract_test.go`

- [ ] **Step 1: Add the pinned download contract**

Implement a `/bin/sh` script with:

```sh
set -eu
VERSION="v0.1.0-rc.2"
PINNED_INSTALLER_SHA256="8f68b0ad0707e5e710cb365849cf833f16eaea1ac76407905763747dae986c25"
ORIGIN="${REINSTATE_BOOTSTRAP_ORIGIN:-https://raw.githubusercontent.com/HarjjotSinghh/reinstate}"
EXPECTED_INSTALLER_SHA256="${REINSTATE_BOOTSTRAP_INSTALLER_SHA256:-$PINNED_INSTALLER_SHA256}"
INSTALLER_URL="${ORIGIN}/${VERSION}/scripts/install.sh"
```

Download to a `mktemp -d` directory, clean it with a trap, calculate SHA-256
using `sha256sum` or `shasum -a 256`, compare exactly, and only then run:

```sh
REINSTATE_VERSION="$VERSION" sh "$installer_path"
```

- [ ] **Step 2: Add safe idempotent PATH setup**

After successful installation:

- derive `INSTALL_DIR` exactly as the canonical installer does;
- skip changes when it is already in current `PATH`;
- honor `REINSTATE_SKIP_PATH_UPDATE=1`;
- select `.zshrc`, `.bashrc`, or `.profile` from `$SHELL`;
- quote custom installation paths as POSIX shell literals;
- append only a marked export line;
- never rewrite existing profile contents; and
- print an absolute immediate command plus the new-terminal command.

- [ ] **Step 3: Validate shell syntax**

Run:

```sh
sh -n website/public/install.sh
```

Expected: exit 0.

- [ ] **Step 4: Run POSIX contract tests and confirm GREEN**

Run:

```sh
GOTOOLCHAIN=go1.25.12 go test ./internal/doctest \
  -run 'Test(PublicBootstrapStaticContract|POSIXPublicBootstrapContract)' \
  -count=1
```

Expected: PASS on macOS/Linux.

### Task 3: Implement the Windows bootstrap

**Files:**

- Create: `website/public/install.ps1`
- Test: `internal/doctest/bootstrap_install_contract_test.go`

- [ ] **Step 1: Add the pinned PowerShell download contract**

Implement:

```powershell
$ErrorActionPreference = "Stop"
$Version = "v0.1.0-rc.2"
$PinnedInstallerSha256 = "4d6e422f36ef20f4378786b34a75c042223ebff3db13b3a05f7a97e1126d6781"
```

Construct the exact tag URL, download to a GUID-named temporary directory,
verify with `Get-FileHash`, and execute the verified text as a child
`ScriptBlock` with `$env:REINSTATE_VERSION` temporarily set. Restore the
caller's previous environment variable in `finally`.

- [ ] **Step 2: Add idempotent Windows PATH setup**

Normalize PATH entries case-insensitively after trimming quotes and trailing
separators. Default persistence scope is `User`; allow
`REINSTATE_BOOTSTRAP_PATH_SCOPE=Process` only for isolated tests. Honor
`REINSTATE_SKIP_PATH_UPDATE=1` for persistence, always make the command
available in the current process, and print `Next: rein init`.

- [ ] **Step 3: Parse and smoke-test locally when PowerShell is available**

Run:

```sh
pwsh -NoProfile -Command \
  "[void][scriptblock]::Create((Get-Content -Raw website/public/install.ps1))"
```

Expected: exit 0. If `pwsh` is unavailable locally, record that native execution
will be supplied by the Windows CI matrix.

- [ ] **Step 4: Run all bootstrap tests**

Run:

```sh
GOTOOLCHAIN=go1.25.12 go test ./internal/doctest -run Bootstrap -count=1
```

Expected: POSIX and static contracts pass locally; Windows behavioral coverage
runs on Windows CI.

- [ ] **Step 5: Commit the executable installer slice**

```sh
git add website/public/install.sh website/public/install.ps1 \
  internal/doctest/bootstrap_install_contract_test.go
git commit -m "feat(installer): add pinned one-line bootstraps"
```

## Chunk 2: Build and Documentation Integration

### Task 4: Add the website production gate

**Files:**

- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add a website CI job**

Add an Ubuntu job using:

```yaml
- uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020 # v4.4.0
  with:
    node-version: "22.12.0"
    cache: npm
    cache-dependency-path: website/package-lock.json
```

Run `npm ci`, `npm test`, and `npm run build` in `website/`. Then assert:

```sh
test -s dist/client/install.sh
test -s dist/client/install.ps1
grep -F 'v0.1.0-rc.2' dist/client/install.sh
grep -F 'v0.1.0-rc.2' dist/client/install.ps1
```

- [ ] **Step 2: Run website tests**

Run:

```sh
cd website && npm test
```

Expected: all Vitest tests pass.

- [ ] **Step 3: Run and inspect the production build**

Run:

```sh
cd website && npm run build
```

Expected: build succeeds and both files exist under `dist/client/`.

- [ ] **Step 4: Commit CI integration**

```sh
git add .github/workflows/ci.yml
git commit -m "ci(website): verify public installer assets"
```

### Task 5: Replace stale install instructions

**Files:**

- Modify: `README.md`
- Modify: `docs/getting-started.md`
- Modify: `website/src/content/docs/getting-started.md`
- Modify: `scripts/install.sh`
- Modify: `scripts/install.ps1`

- [ ] **Step 1: Add the public commands to README**

Document:

```sh
curl -fsSL https://reinstate.dev/install.sh | sh
```

```powershell
irm https://reinstate.dev/install.ps1 | iex
```

State that they install `v0.1.0-rc.2`, configuration starts with `rein init`,
and the release remains an RC until native acceptance is signed off.

- [ ] **Step 2: Update both getting-started guides**

Keep the commands platform-specific. Add an inspect-first alternative that
downloads the public bootstrap to disk, lets the user inspect it, and executes
it. Remove moving-`main`, moving-`latest`, unshipped adapters, and target-UX
claims from the website copy.

- [ ] **Step 3: Update canonical installer comments**

Make the first usage example the stable public command and retain the advanced
exact-tag invocation for audits or mirrors. Do not change installer runtime
behavior in this task.

- [ ] **Step 4: Run docs contracts**

Run:

```sh
GOTOOLCHAIN=go1.25.12 go test ./internal/doctest -count=1
```

Expected: PASS.

## Chunk 3: Two-Device Acceptance Runbook

### Task 6: Write the Phase 1 Mac/Windows guide

**Files:**

- Create: `docs/testing/phase-1-mac-windows-acceptance.md`
- Modify: `docs/README.md`
- Modify: `website/src/content/docs/README.md`
- Reference: `docs/cli-reference.md`
- Reference: `docs/prompts/claude-code-setup.md`
- Reference: `docs/prompts/codex-setup.md`

- [ ] **Step 1: Document prerequisites and evidence**

List native macOS, native Windows PowerShell, Claude Code, Codex, a disposable
project on each machine, an R2/S3 bucket, exact profile ID sharing, passphrase
handling, and a result table with date/device/command/outcome/evidence fields.

Explicitly prohibit pasting credentials or passphrases into an AI-agent prompt.

- [ ] **Step 2: Add installation and preflight acceptance**

For each device include the public one-liner, expected installation path,
`rein version --json`, `rein doctor --self-test`, and `rein setup check`.
Separate mandatory results from optional diagnostics.

- [ ] **Step 3: Add first-device and second-device setup**

Use one canonical project ID with two different absolute paths. Capture the
non-secret profile ID from the first device. Require identical storage
coordinates and encryption passphrase on the second device.

- [ ] **Step 4: Add bidirectional agent acceptance**

Create a harmless recognizable Claude Code session and Codex session on the
source device. Test:

1. `status`;
2. `push --all --dry-run`;
3. `push --all`;
4. `pull --all --dry-run`;
5. `pull --all`;
6. `claude --resume`; and
7. `codex resume`.

Repeat in the reverse direction with new markers.

- [ ] **Step 5: Add safety and failure cases**

Cover no-op repeat sync, local backups, deliberate concurrent edits/conflicts,
wrong passphrase, checksum/tamper expectations, and safe cleanup. Never instruct
the user to delete real agent data.

- [ ] **Step 6: Add strict sign-off**

Define mandatory pass criteria. Any failed checksum, missing backup, plaintext
remote object, failed resume, destructive overwrite, or unexplained command
error keeps Phase 1 open.

- [ ] **Step 7: Link the runbook from both docs indexes**

Add descriptive links without breaking the repository markdown-link checker.

- [ ] **Step 8: Commit documentation**

```sh
git add README.md docs website/src/content/docs scripts/install.sh scripts/install.ps1
git commit -m "docs(phase1): add two-device acceptance runbook"
```

## Chunk 4: Verification and Publication

### Task 7: Run focused and full local verification

**Files:** none unless a gate exposes a defect.

- [ ] **Step 1: Check formatting and diffs**

Run:

```sh
gofmt -w internal/doctest/bootstrap_install_contract_test.go
git diff --check
git status --short
```

Expected: no formatting or whitespace errors and only intended files changed.

- [ ] **Step 2: Run bootstrap, docs, and website gates**

Run:

```sh
sh -n website/public/install.sh
GOTOOLCHAIN=go1.25.12 go test ./internal/doctest -count=1
(cd website && npm test && npm run build)
```

Expected: all pass.

- [ ] **Step 3: Run the repository merge gate**

Run:

```sh
make verify
```

Expected: `verify ok`.

- [ ] **Step 4: Inspect the production artifacts**

Assert that the built static files are non-empty, pin the RC, contain no
`latest` resolver, and match the source files byte-for-byte.

- [ ] **Step 5: Perform a local network smoke against the published RC**

Run the public POSIX bootstrap logic in an isolated temporary `HOME` and
`INSTALL_DIR`, while preserving any existing real installation. Verify:

```sh
<temporary-install-dir>/rein version --json
```

Expected version: `0.1.0-rc.2`.

### Task 8: Publish through pull request

**Files:** none.

- [ ] **Step 1: Confirm clean branch and commit history**

Run:

```sh
git status --short --branch
git log --oneline origin/main..HEAD
```

Expected: clean branch with only the planned commits.

- [ ] **Step 2: Push and open the PR**

```sh
git push -u origin feat/one-line-installers
gh pr create --base main --head feat/one-line-installers
```

The PR body must state the security model, tests run, manual Windows follow-up,
and that the endpoint installs `v0.1.0-rc.2`.

- [ ] **Step 3: Wait for all required checks**

Inspect each check. Fix failures before merge; do not merge around a failing
Windows bootstrap contract or website production build.

- [ ] **Step 4: Merge and verify ancestry**

Merge the PR, fetch `origin/main`, and prove the merge commit is reachable from
the remote main branch.

- [ ] **Step 5: Verify Vercel publication**

Poll both routes without waits longer than 60 seconds:

```sh
curl -fsSL https://reinstate.dev/install.sh
curl -fsSL https://reinstate.dev/install.ps1
```

Assert HTTP 200, expected script content, pinned RC version, and no HTML/404
response.

- [ ] **Step 6: Run a live isolated macOS installation**

Pipe the live public installer into `sh` with isolated `HOME` and `INSTALL_DIR`.
Verify the installed binary reports `0.1.0-rc.2`. Do not alter the user's real
CLI or shell profiles.

- [ ] **Step 7: Hand off native acceptance**

Give the user the runbook path and state plainly that Phase 1 is not complete
until its mandatory Mac/Windows checklist passes.
