# Reinstate Phase 0 and Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents are explicitly authorized) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking. Reinstate's root `AGENTS.md` prohibits unsolicited subagents, so the user's explicit authorization is always required before delegation.

**Goal:** Ship Reinstate `v0.1.0` as a trustworthy cross-device CLI that installs on macOS and Windows, safely syncs same-vendor Claude Code and OpenAI Codex sessions through encrypted R2/S3-compatible storage, and can be set up through either complete manual instructions or a versioned copy-paste AI-agent prompt.

**Architecture:** Reinstate preserves vendor-native session files as opaque payloads and applies only schema-aware path and project-identity transformations. A versioned local manifest tracks immutable encrypted remote snapshots; pull operations verify, decrypt, preview, back up, and atomically restore without silently overwriting divergent work. Phase 0 establishes contracts, diagnostics, installers, fixtures, documentation, and release trust; Phase 1 completes Claude and Codex end-to-end sync.

**Tech Stack:** Go; Cobra; TOML; `filippo.io/age`; AWS SDK for Go v2; OS credential stores through a cross-platform keyring abstraction; SHA-256; GitHub Actions; GitHub Releases; R2/S3-compatible object storage; Markdown documentation.

---

## 1. Authority and stopping rules

This document is the approved authority for Reinstate Phase 0 and Phase 1.

The work is complete only when the documented release gates pass. A green
scaffold, a successful cross-compile, or a plausible adapter implementation is
not evidence that the product works.

### Approved product decisions

1. Phase 0 is the verified foundation and bootstrap layer.
2. Phase 1 is the complete Claude Code and Codex sessions-only product.
3. The first stable public release is `v0.1.0`.
4. The primary architecture is opaque same-vendor snapshots with schema-aware
   transformations.
5. R2/S3-compatible storage is the first remote backend.
6. Hosted Reinstate storage is not part of Phase 0 or Phase 1.
7. Native Windows and WSL2 are distinct device environments.
8. Authentication files and credentials are never synced.
9. Documentation is part of each user-facing feature's definition of done.
10. SemVer tags identify installable, documented, tested releases—not arbitrary
    intermediate commits.

### Phase 1 non-goals

- Gemini, OpenCode, Grok, Cursor, or IDE adapters
- MCP, skills, hooks, instructions, or settings synchronization
- cross-agent transcript translation
- background services or automatic shell hooks
- real-time collaboration or CRDTs
- hosted accounts, billing, relay, dashboard, or team features
- telemetry
- syncing Claude/Codex auth, API keys, OAuth tokens, `.env` files, or OS
  credential-store contents

### Execution safety

Before implementation:

- preserve every existing tracked and untracked user change;
- do not mix unrelated dirty-tree changes into feature commits;
- create an isolated feature worktree from the intended `main` commit unless the
  user explicitly chooses another workflow;
- verify `main` and `origin/main` divergence before starting;
- never publish a tag or release until its specific gate passes.

---

## 2. Current repository baseline

At plan approval:

- `cmd/reinstate/main.go` implements only `version` and `help`;
- `init`, `push`, `pull`, `status`, `diff`, and `conflicts` are stubs;
- `internal/adapter/adapter.go` is a sketch with no concrete adapters;
- `internal/config`, `internal/crypto`, `internal/pathmap`, and `internal/sync`
  contain documentation stubs only;
- the only test confirms the version string is non-empty;
- the Go scaffold passes `go test`, `go vet`, and `make build`;
- no local or remote SemVer tags/releases exist;
- the repository contains user-owned tracked and untracked changes that must be
  preserved;
- current CI tests three operating systems, but lint is soft and docs checks
  only assert file existence;
- the POSIX installer does not verify downloads and there is no native
  PowerShell installer.

These checks prove that the starting scaffold builds. They do not prove any sync
behavior.

---

## 3. Support contract

### Primary tested matrix

| Environment | Claude Code | Codex CLI |
| --- | --- | --- |
| macOS native, arm64 | Required | Required |
| macOS native, amd64 | Required before stable release | Required before stable release |
| Windows 11 native, amd64 | Required | Required |
| Windows 11 WSL2, amd64 | Documented and release-smoked separately | Documented and release-smoked separately |

### Explicitly unsupported in `v0.1.0`

- WSL1
- treating native Windows and WSL as the same Reinstate device
- automatic sharing of one agent-state directory between Windows and WSL
- untested agent versions/layouts during restore
- other coding agents

### Compatibility states

Every adapter discovery result must report one of:

```text
SUPPORTED     exact layout/version range has release evidence
UNTESTED      recognizable layout, newer or otherwise unverified version
UNSUPPORTED   known-incompatible layout/version
NOT_INSTALLED no local installation/root found
```

Rules:

- `SUPPORTED` permits discovery, push, and pull.
- `UNTESTED` permits read-only discovery and dry-run export but refuses restore
  unless the user supplies an explicit compatibility override.
- `UNSUPPORTED` fails closed and links to compatibility documentation.
- `NOT_INSTALLED` is informational.

---

## 4. Public CLI contract

### Commands

```text
rein version [--json]
rein doctor [--json] [--self-test]
rein setup check [--json]
rein init
rein list [--agent claude|codex|all] [--json]
rein status [--json]
rein diff [--agent ...] [--session ...] [--json]
rein push [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein pull [--agent ...] [--session ...|--all] [--dry-run] [--json]
rein conflicts list [--json]
rein conflicts show <id> [--json]
rein conflicts resolve <id> --keep-local|--keep-remote|--keep-both
rein completion bash|zsh|fish|powershell
```

`reinstate` invokes the same executable and behavior.

### Stable exit codes

```text
0 success
1 unexpected runtime failure
2 usage or invalid arguments
3 missing/invalid config
4 authentication or storage failure
5 agent/layout compatibility failure
6 sync conflict
7 safety refusal
```

### Output rules

- Human output is concise and actionable.
- `--json` is stable, machine-readable, and contains no ANSI control sequences.
- JSON errors include `code`, `message`, `details`, and `safe_to_retry`.
- `diff` shows metadata by default, not transcript text.
- diagnostics redact usernames, credentials, transcript content, and sensitive
  endpoint query strings.
- no command prints a passphrase, storage secret, agent token, or keyring value.

---

## 5. Local filesystem contract

### Reinstate home

Default:

```text
macOS/Linux/WSL: ~/.reinstate
Windows:         %USERPROFILE%\.reinstate
```

Override:

```text
REINSTATE_HOME=/absolute/path
```

Layout:

```text
~/.reinstate/
  config.toml
  state.json
  device.json
  cache/
  backups/
  conflicts/
  locks/
  logs/
```

Rules:

- directories use owner-only permissions where supported;
- config and state files use owner read/write permissions where supported;
- Windows uses user-scoped ACLs rather than pretending POSIX modes are enough;
- writes use temporary siblings, `fsync` where applicable, and atomic rename;
- backups are created before any vendor-state replacement;
- lock files prevent overlapping Reinstate mutations;
- vendor agent files are never edited in place.

### Config schema v1

```toml
schema_version = 1
profile_id = "generated-opaque-id"
device_id = "generated-opaque-id"

[storage]
type = "s3"
endpoint = "https://<account>.r2.cloudflarestorage.com"
region = "auto"
bucket = "reinstate"
prefix = "profiles/<opaque-profile-id>"
credential_ref = "reinstate/<profile-id>/s3"

[encryption]
type = "age-scrypt"

[agents.claude]
enabled = true

[agents.codex]
enabled = true

[[projects]]
id = "github.com/example/repository"
local_root = "/local/platform/path"
```

Secrets are not valid config fields.

### Local state schema v1

```json
{
  "schema_version": 1,
  "last_remote_etag": "",
  "last_manifest_revision": "",
  "sessions": {},
  "updated_at": "RFC3339 timestamp"
}
```

State changes must be migration-tested.

---

## 6. Remote storage contract

### Object layout

```text
<prefix>/
  manifest.age
  snapshots/<opaque-snapshot-id>.age
  probes/<random-id>
```

Properties:

- snapshot objects are immutable;
- the encrypted manifest maps sessions to immutable snapshot revisions;
- the manifest update uses conditional remote version/ETag semantics;
- object names avoid plaintext repository, user, or session names;
- failed manifest updates do not delete a successfully uploaded snapshot;
- unreferenced snapshots may be garbage-collected only by a future explicit
  maintenance command, not during ordinary push/pull;
- init verifies CRUD using a random disposable probe and deletes it afterward.

### Encrypted envelope v1

Plaintext before age encryption:

```json
{
  "schema_version": 1,
  "kind": "reinstate-session-snapshot",
  "snapshot_id": "opaque-id",
  "parent_revision": "opaque-id-or-empty",
  "agent": "claude|codex",
  "adapter_schema": 1,
  "source_agent_version": "string",
  "source_platform": "darwin-arm64|windows-amd64|linux-wsl2-amd64",
  "project_id": "canonical-project-id",
  "session_id": "vendor-session-id",
  "created_at": "RFC3339 timestamp",
  "files": [
    {
      "path": "portable/relative/path",
      "mode": 384,
      "size": 123,
      "sha256": "hex"
    }
  ]
}
```

The serialized envelope and file payloads are streamed through compression and
age passphrase encryption. The passphrase is entered through a hidden TTY prompt
or a deliberately configured passphrase-file descriptor; it is never accepted
as a command-line argument.

---

## 7. Adapter architecture

Replace the shallow export/import sketch with a planning-oriented contract:

```go
type Compatibility string

const (
    CompatibilitySupported   Compatibility = "SUPPORTED"
    CompatibilityUntested    Compatibility = "UNTESTED"
    CompatibilityUnsupported Compatibility = "UNSUPPORTED"
    CompatibilityNotInstalled Compatibility = "NOT_INSTALLED"
)

type Adapter interface {
    Name() string
    Detect(context.Context) (Install, Compatibility, error)
    Discover(context.Context, DiscoverOptions) ([]Session, error)
    PlanExport(context.Context, Session, ExportOptions) (ExportPlan, error)
    Export(context.Context, ExportPlan, io.Writer) error
    PlanRestore(context.Context, Snapshot, RestoreOptions) (RestorePlan, error)
    Restore(context.Context, RestorePlan, io.Reader) error
    Exclusions() []Exclusion
}
```

Detailed types belong in focused files under `internal/adapter`.

Adapter rules:

- preserve unknown vendor fields and event types;
- parse defensively and never crash on malformed/unknown records;
- transform only known structural path fields;
- never globally replace arbitrary transcript strings;
- hard-exclude auth, credentials, tokens, caches, logs, and regenerable
  dependencies;
- fixtures are synthetic or aggressively redacted and validated by a secret
  scanner;
- every release records exact vendor versions/layout signatures tested.

---

## 8. Path mapping contract

### Canonical project identity

Priority:

1. normalized Git remote plus repository-relative root;
2. explicit user alias;
3. generated opaque local identity when neither is available.

Never use a raw absolute path as global identity.

### Portable tokens

```text
${HOME}
${REPO:<canonical-project-id>}
${WORK:<user-defined-alias>}
```

### Required path cases

- Windows drive letters
- UNC paths
- WSL `/mnt/<drive>` paths
- macOS/Linux paths
- spaces
- Unicode
- slash direction
- case differences
- long paths
- repositories moved or renamed between devices
- multiple worktrees

Round-trip invariants:

```text
denormalize(normalize(platform_path), same_platform) == clean(platform_path)
normalize(denormalize(portable_path, platform)) == portable_path
```

Path mapping must not alter user prose, code blocks, command output, or unknown
JSON fields.

---

## 9. Sync and conflict contract

### Push

1. acquire a local mutation lock;
2. load and validate config/state;
3. detect adapter compatibility;
4. discover requested sessions;
5. enforce exclusions;
6. generate and display a dry-run plan when requested;
7. fetch the current encrypted remote manifest and ETag;
8. compare local base revision with remote session revision;
9. reject divergence as a conflict;
10. export and transform into a temporary streamed snapshot;
11. encrypt and upload the immutable snapshot;
12. conditionally update the remote manifest;
13. atomically persist local state;
14. release the lock.

### Pull

1. acquire a local mutation lock;
2. fetch and authenticate the remote manifest;
3. select requested snapshots;
4. verify schema and adapter compatibility;
5. download and decrypt into a private temporary directory;
6. validate hashes and exclusion rules;
7. produce the restore plan;
8. return without mutation for `--dry-run`;
9. detect local divergence and active vendor processes;
10. create timestamped backups;
11. restore through temporary files and atomic rename;
12. validate restored hashes/layout;
13. atomically persist local state;
14. release the lock.

### Conflict behavior

- divergence never silently resolves with last-writer-wins;
- conflict records contain metadata and snapshot references, not plaintext
  transcript content;
- `--keep-local` publishes a new revision based on the current remote head;
- `--keep-remote` backs up local data before restore;
- `--keep-both` restores the alternate session under a vendor-safe forked
  identity;
- conflict resolution itself creates auditable local and remote state.

---

## 10. Documentation and AI-assisted setup contract

### User documentation tree

```text
docs/
  README.md
  cli-reference.md
  configuration.md
  storage-backends.md
  path-mapping.md
  backup-and-recovery.md
  compatibility.md
  testing.md
  uninstall.md
  troubleshooting.md
  security-model.md
  install/
    manual-macos.md
    manual-windows.md
    manual-wsl.md
    agent-assisted.md
    verify-installation.md
  prompts/
    README.md
    claude-code-setup.md
    codex-setup.md
    contributor-setup.md
  adapters/
    README.md
    claude-code.md
    codex.md
    contributing-an-adapter.md
  contributing/
    development.md
    testing.md
    documentation.md
    release-process.md
  adr/
  spec/
```

### Prompt architecture

The canonical prompt is thin and version-pinned. It tells the host agent to:

1. run inside an empty bootstrap directory;
2. detect and report host agent/version, OS, architecture, shell, and
   native-Windows-versus-WSL status;
3. inventory existing `rein`/`reinstate` without modifying it;
4. show planned changes and request approval before elevation or profile edits;
5. download only the exact official Reinstate release;
6. verify the published checksum/attestation;
7. install into the documented user-local path;
8. hand interactive secret entry to the human through `rein init`;
9. run `rein doctor --self-test`;
10. finish with a redacted report of files changed, versions, checks, and
    remaining human actions.

The prompt must explicitly prohibit:

- disabling sandbox/approval protections;
- `--yolo`, permission bypass, or danger-full-access modes;
- reading, copying, printing, or syncing auth files and credential stores;
- asking the user to paste passphrases or storage secrets into chat;
- installing from `main`, an unpinned branch, or an unverified latest artifact;
- modifying unrelated repositories;
- publishing, committing, or pushing anything.

### Two prompt modes

- End-user setup installs a release binary. It does not clone the repository or
  require Go.
- Contributor setup clones the repository, installs the supported Go toolchain,
  runs checks, and never configures real storage or reads real sessions by
  default.

### Documentation release rule

Every user-visible PR must update, as applicable:

- README happy path;
- CLI reference;
- manual installation;
- AI-agent prompt;
- troubleshooting;
- security model;
- adapter/compatibility matrix;
- changelog `[Unreleased]`.

Docs are checked against actual CLI help and test fixtures to prevent drift.

---

## 11. Versioning and release contract

### SemVer policy

```text
v0.1.0-alpha.1  authority and corrected contracts
v0.1.0-alpha.2  CLI/schema/doctor
v0.1.0-alpha.3  verified installers
v0.1.0-alpha.4  tested agent-assisted setup
v0.1.0-beta.1   Claude end-to-end
v0.1.0-beta.2   Codex end-to-end
v0.1.0-rc.1     complete release candidate
v0.1.0          Phase 1 stable
```

Rules:

- work-in-progress uses focused Conventional Commits;
- feature completion is tracked by its PR, issue, milestone, and changelog
  entry;
- a SemVer tag is created only when the target commit is independently
  installable, documented, tested, and intentionally released;
- release tags are signed and annotated;
- published tags and artifacts are immutable;
- patch releases contain backward-compatible fixes;
- pre-1.0 breaking changes occur only in a new minor release and include a
  migration path;
- config and remote format compatibility are public API.

### Release artifact contract

```text
reinstate_<version>_darwin_arm64.tar.gz
reinstate_<version>_darwin_amd64.tar.gz
reinstate_<version>_linux_arm64.tar.gz
reinstate_<version>_linux_amd64.tar.gz
reinstate_<version>_windows_amd64.zip
checksums.txt
checksums.txt.sigstore.json
reinstate_<version>.spdx.json
```

Windows arm64 may be build-only/experimental until a real runtime gate exists.

### Release gate

1. Tag matches the documented SemVer prerelease/stable pattern.
2. Tag points to the reviewed release commit on protected `main`.
3. Changelog, release title, binary version, and docs agree.
4. Required CI passes.
5. No unresolved release-blocking security issue exists.
6. Exact draft artifacts install on macOS and native Windows.
7. WSL2 install smoke passes.
8. Claude Windows-to-macOS and macOS-to-Windows resume passes.
9. Codex Windows-to-macOS and macOS-to-Windows resume passes.
10. Wrong-passphrase, tamper, backup, rollback, and conflict tests pass.
11. Installers verify checksums.
12. Archives contain binary, license, notice, and minimum install material.
13. SBOM and build provenance/attestation are attached.
14. Exact draft artifacts—not locally rebuilt approximations—pass smoke tests.
15. Release notes are curated from Keep a Changelog.
16. Published release is immutable.

---

## 12. Planned file map

### CLI

```text
cmd/reinstate/main.go                    process entry only
internal/cli/root.go                     root command construction
internal/cli/version.go                  version command
internal/cli/doctor.go                   diagnostics command
internal/cli/setup.go                    setup preflight
internal/cli/init.go                     interactive initialization
internal/cli/list.go                     session listing
internal/cli/status.go                   sync status
internal/cli/diff.go                     metadata diff
internal/cli/push.go                     push command
internal/cli/pull.go                     pull command
internal/cli/conflicts.go                conflict commands
internal/cli/output.go                   human/JSON output
internal/cli/exit.go                     stable exit codes
internal/cli/*_test.go                   command behavior
```

### Core contracts

```text
internal/schema/config.go                config v1 types/version
internal/schema/state.go                 local state v1
internal/schema/manifest.go              remote manifest v1
internal/schema/envelope.go              encrypted snapshot metadata v1
internal/config/path.go                  REINSTATE_HOME resolution
internal/config/load.go                  load/validate/migrate
internal/config/save.go                  atomic persistence
internal/device/detect.go                OS/arch/native/WSL
internal/doctor/report.go                redacted diagnostic model
internal/fsx/atomic.go                    atomic writes
internal/fsx/permissions.go               POSIX/Windows protection
internal/fsx/backup.go                    timestamped backup
internal/lock/lock.go                    mutation locking
```

### Storage and crypto

```text
internal/backend/backend.go              storage interface and errors
internal/backend/memory/memory.go         deterministic test backend
internal/backend/s3/s3.go                R2/S3 implementation
internal/backend/s3/config.go             endpoint and capability validation
internal/credentials/store.go             credential abstraction
internal/credentials/keyring.go           OS keyring implementation
internal/credentials/environment.go       explicit fallback provider
internal/crypto/envelope.go               streamed age encryption
internal/crypto/passphrase.go             hidden prompt/file-descriptor input
internal/crypto/hash.go                   SHA-256 helpers
```

### Projects, adapters, and sync

```text
internal/project/id.go                    canonical project identity
internal/pathmap/token.go                 portable token model
internal/pathmap/normalize.go             platform to token
internal/pathmap/denormalize.go           token to platform
internal/adapter/types.go                 shared adapter types
internal/adapter/registry.go              adapter registry
internal/adapter/claude/detect.go          Claude roots/version
internal/adapter/claude/discover.go        Claude session discovery
internal/adapter/claude/transform.go       schema-aware JSONL transforms
internal/adapter/claude/restore.go         Claude restore planning
internal/adapter/codex/detect.go           Codex roots/version
internal/adapter/codex/discover.go         rollout discovery
internal/adapter/codex/transform.go        rollout path transforms
internal/adapter/codex/restore.go          rollout/index restore planning
internal/manifest/local.go                 local state operations
internal/manifest/remote.go                encrypted remote manifest
internal/sync/plan.go                      push/pull planning
internal/sync/push.go                      push orchestration
internal/sync/pull.go                      pull orchestration
internal/sync/conflict.go                  divergence/resolution
```

### Tests, installers, CI, and docs

```text
testdata/adapters/claude/{macos,windows,wsl}/
testdata/adapters/codex/{macos,windows,wsl}/
testdata/sync/
scripts/install.sh
scripts/install.ps1
scripts/verify-release.sh
scripts/verify-release.ps1
.github/workflows/ci.yml
.github/workflows/security.yml
.github/workflows/release.yml
.github/ISSUE_TEMPLATE/docs_report.yml
.github/ISSUE_TEMPLATE/compatibility_regression.yml
docs/...                                   tree in Section 10
```

---

# Chunk 1: Phase 0 — Authority, contracts, and diagnostics

## Task 1: Isolate implementation work and capture the baseline

**Files:**

- No product files initially
- Create worktree outside the current dirty tree

- [ ] **Step 1: Audit the current tree**

Run:

```bash
git status --short --branch
git rev-list --left-right --count main...origin/main
git worktree list --porcelain
```

Expected: every user-owned modification is recorded; divergence is understood.

- [ ] **Step 2: Create the approved implementation branch/worktree**

Use a branch such as:

```bash
git worktree add ../reinstate-phase-01 -b feat/phase-0-phase-1 main
```

Expected: a clean isolated worktree at the intended `main` SHA.

- [ ] **Step 3: Re-run the baseline**

Run:

```bash
go test ./... -count=1
go vet ./...
make build
```

Expected: all commands exit `0`; record existing limitations separately.

- [ ] **Step 4: Commit only if the worktree setup required a tracked change**

Normally no commit is expected for this task.

## Task 2: Reconcile roadmap, changelog, support, and version truth

**Files:**

- Modify: `README.md`
- Modify: `ROADMAP.md`
- Modify: `CHANGELOG.md`
- Modify: `RELEASING.md`
- Modify: `CITATION.cff`
- Modify: `SECURITY.md`
- Modify: `SUPPORT.md`
- Modify: `docs/README.md`
- Create: `docs/adr/0001-phase-0-phase-1-scope.md`
- Create: `docs/compatibility.md`

- [ ] **Step 1: Write a docs consistency test**

Create `internal/doctest/version_test.go` or a focused script test that fails
when nonexistent releases are advertised or support status uses implemented
checkmarks for planned adapters.

- [ ] **Step 2: Run the test and verify failure**

Run:

```bash
go test ./internal/doctest -run TestReleaseAndSupportClaims -count=1
```

Expected: FAIL against the current contradictory docs.

- [ ] **Step 3: Normalize Phase 0/1 and version claims**

Apply the authority in Sections 1, 3, and 11. Remove fake `v0.0.0` history and
planned work from changelog release sections.

- [ ] **Step 4: Run docs checks**

Run:

```bash
go test ./internal/doctest -count=1
git diff --check
```

Expected: PASS and no whitespace errors.

- [ ] **Step 5: Commit**

```bash
git add README.md ROADMAP.md CHANGELOG.md RELEASING.md CITATION.cff SECURITY.md SUPPORT.md docs
git commit -m "docs(product): define phase 0 and phase 1 contracts"
```

## Task 3: Introduce CLI routing and stable exit codes

**Files:**

- Modify: `cmd/reinstate/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/exit.go`
- Create: `internal/cli/output.go`
- Create: `internal/cli/root_test.go`
- Modify: `cmd/reinstate/main_test.go`
- Modify: `go.mod`
- Create: `go.sum`

- [ ] **Step 1: Write failing CLI behavior tests**

Cover:

- no arguments shows help and exits `2`;
- `--help` exits `0`;
- unknown command exits `2`;
- `version --json` emits valid JSON;
- errors map to stable exit codes;
- `rein` and `reinstate` names do not change behavior.

- [ ] **Step 2: Run tests and verify failure**

```bash
go test ./cmd/reinstate ./internal/cli -count=1
```

Expected: FAIL because the CLI package does not exist.

- [ ] **Step 3: Add Cobra and implement the minimum routing layer**

Keep `main.go` limited to constructing/executing the root command and mapping
typed errors to process exit codes.

- [ ] **Step 4: Format and run focused tests**

```bash
gofmt -w cmd/reinstate internal/cli
go test ./cmd/reinstate ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 5: Run the binary manually**

```bash
go run ./cmd/reinstate --help
go run ./cmd/reinstate version --json
```

Expected: documented commands and valid version JSON.

- [ ] **Step 6: Commit**

```bash
git add cmd/reinstate internal/cli go.mod go.sum
git commit -m "feat(cli): add command routing and stable exit codes"
```

## Task 4: Implement versioned config and state persistence

**Files:**

- Create: `internal/schema/config.go`
- Create: `internal/schema/state.go`
- Create: `internal/config/path.go`
- Create: `internal/config/load.go`
- Create: `internal/config/save.go`
- Create: `internal/config/config_test.go`
- Create: `internal/fsx/atomic.go`
- Create: `internal/fsx/atomic_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Write failing schema/path/atomic-write tests**

Cover:

- default and overridden Reinstate home;
- native Windows path construction;
- malformed and unknown config versions;
- rejection of secret fields;
- atomic save leaves the previous file intact on failure;
- owner-only permissions where supported;
- state round-trip and version migration errors.

- [ ] **Step 2: Verify tests fail**

```bash
go test ./internal/config ./internal/fsx ./internal/schema -count=1
```

Expected: FAIL with missing implementations.

- [ ] **Step 3: Implement config v1 and state v1**

Use TOML for config and JSON for local state. Do not introduce SQLite in Phase
1.

- [ ] **Step 4: Implement atomic persistence**

Use private temp files in the target directory, sync, rename, and directory sync
where supported.

- [ ] **Step 5: Run focused and race tests**

```bash
gofmt -w internal/config internal/fsx internal/schema
go test ./internal/config ./internal/fsx ./internal/schema -count=1
go test -race ./internal/config ./internal/fsx -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config internal/fsx internal/schema go.mod go.sum
git commit -m "feat(config): add versioned config and atomic state"
```

## Task 5: Detect devices and agent compatibility

**Files:**

- Create: `internal/device/detect.go`
- Create: `internal/device/detect_test.go`
- Create: `internal/adapter/types.go`
- Create: `internal/adapter/registry.go`
- Create: `internal/adapter/registry_test.go`
- Refactor: `internal/adapter/adapter.go`

- [ ] **Step 1: Write failing platform detection tests**

Use injected environment/system probes to cover:

- macOS arm64/amd64;
- native Windows;
- WSL2;
- WSL1 refusal;
- Linux non-WSL;
- stable platform IDs.

- [ ] **Step 2: Write failing compatibility-state tests**

Cover all four states and the restore refusal rules.

- [ ] **Step 3: Run tests and verify failure**

```bash
go test ./internal/device ./internal/adapter -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement the minimum device and registry contracts**

Do not read session contents or credentials during detection.

- [ ] **Step 5: Run and format**

```bash
gofmt -w internal/device internal/adapter
go test ./internal/device ./internal/adapter -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/device internal/adapter
git commit -m "feat(adapter): add device and compatibility detection"
```

## Task 6: Build redacted doctor and setup preflight

**Files:**

- Create: `internal/doctor/report.go`
- Create: `internal/doctor/redact.go`
- Create: `internal/doctor/selftest.go`
- Create: `internal/doctor/report_test.go`
- Create: `internal/cli/doctor.go`
- Create: `internal/cli/setup.go`
- Create: `internal/cli/doctor_test.go`

- [ ] **Step 1: Write failing redaction tests**

Include usernames, home paths, query-string credentials, fake API keys, session
text, and auth filenames. Assert none appear in JSON or human reports.

- [ ] **Step 2: Write failing doctor exit-code tests**

Cover healthy, missing config, unsupported agent, unavailable keyring, failed
self-test, and unavailable storage.

- [ ] **Step 3: Run tests and verify failure**

```bash
go test ./internal/doctor ./internal/cli -run 'Doctor|Setup|Redact' -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement doctor and read-only setup check**

Self-test uses only temporary synthetic data and a memory backend. It never reads
real vendor sessions.

- [ ] **Step 5: Verify**

```bash
gofmt -w internal/doctor internal/cli
go test ./internal/doctor ./internal/cli -count=1
go run ./cmd/reinstate doctor --json
go run ./cmd/reinstate doctor --self-test
```

Expected: valid redacted JSON and successful synthetic self-test.

- [ ] **Step 6: Commit**

```bash
git add internal/doctor internal/cli
git commit -m "feat(doctor): add redacted diagnostics and self-test"
```

---

# Chunk 2: Phase 0 — Installation, prompts, fixtures, and release trust

## Task 7: Create fixture policy and synthetic fixture generators

**Files:**

- Create: `internal/fixture/generate.go`
- Create: `internal/fixture/scan.go`
- Create: `internal/fixture/scan_test.go`
- Create: `testdata/adapters/claude/{macos,windows,wsl}/`
- Create: `testdata/adapters/codex/{macos,windows,wsl}/`
- Create: `docs/contributing/testing.md`
- Create: `docs/adapters/contributing-an-adapter.md`

- [ ] **Step 1: Write a failing secret-fixture scanner test**

Detect common key/token formats, auth filenames, real home paths, emails, and
private repository names.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/fixture -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement scanner and synthetic generators**

Fixtures must be deterministic and contain obvious fake markers.

- [ ] **Step 4: Generate fixtures and scan**

```bash
go test ./internal/fixture -count=1
go run ./cmd/reinstate-internal-fixture generate
go run ./cmd/reinstate-internal-fixture scan testdata
```

If a separate internal command is unnecessary, expose equivalent test helpers
without shipping them in the public binary.

- [ ] **Step 5: Commit**

```bash
git add internal/fixture testdata docs/contributing/testing.md docs/adapters
git commit -m "test(fixtures): add synthetic adapter fixture policy"
```

## Task 8: Harden release builds and CI

**Files:**

- Modify: `.github/workflows/ci.yml`
- Create: `.github/workflows/security.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `.golangci.yml`
- Modify: `Makefile`
- Create: `.goreleaser.yml`
- Create: `scripts/check-docs.sh`
- Create: `scripts/check-docs.ps1`

- [ ] **Step 1: Add local gate targets**

Required targets:

```text
make fmt-check
make lint
make test
make test-race
make vet
make vuln
make docs-check
make verify
```

- [ ] **Step 2: Run the new gate and record expected baseline failures**

```bash
make verify
```

Expected initially: FAIL for known formatting/docs/lint gaps.

- [ ] **Step 3: Make lint and docs checks hard failures**

Pin action dependencies to immutable commit SHAs. Keep the minimum promised Go
version in compatibility CI and use a pinned currently supported Go release for
release artifacts.

- [ ] **Step 4: Add security checks**

Include `govulncheck`, CodeQL, dependency review, secret scanning, and workflow
permission review.

- [ ] **Step 5: Add release validation**

Validate:

- SemVer tag format;
- tag ancestry;
- changelog/version equality;
- clean source archive;
- checksums;
- SBOM;
- provenance/attestation;
- archive contents.

- [ ] **Step 6: Verify locally applicable gates**

```bash
make verify
goreleaser release --snapshot --clean
```

Expected: PASS and complete snapshot artifacts.

- [ ] **Step 7: Commit**

```bash
git add .github .golangci.yml .goreleaser.yml Makefile scripts
git commit -m "ci(release): enforce cross-platform release gates"
```

## Task 9: Implement checksum-verifying installers

**Files:**

- Rewrite: `scripts/install.sh`
- Create: `scripts/install.ps1`
- Create: `scripts/verify-release.sh`
- Create: `scripts/verify-release.ps1`
- Create: `scripts/test-install.sh`
- Create: `scripts/test-install.ps1`
- Create: `docs/install/manual-macos.md`
- Create: `docs/install/manual-windows.md`
- Create: `docs/install/manual-wsl.md`
- Create: `docs/install/verify-installation.md`
- Create: `docs/uninstall.md`

- [ ] **Step 1: Write installer test cases**

Cover:

- exact version selection;
- supported OS/arch mapping;
- user-local installation;
- checksum mismatch refusal;
- missing release asset refusal;
- existing valid installation preservation;
- upgrade/downgrade confirmation;
- PATH instructions;
- Windows `rein.exe` and `reinstate.exe`;
- no surprise elevation.

- [ ] **Step 2: Verify current installers fail requirements**

Run the installer tests against the current script.

Expected: checksum, PowerShell, alias, and elevation tests fail.

- [ ] **Step 3: Implement POSIX installer**

Avoid brittle JSON parsing when release URLs can be derived from an exact
version. Never execute a download before checksum verification.

- [ ] **Step 4: Implement PowerShell installer**

Install under `%LOCALAPPDATA%\Programs\Reinstate\bin` by default and create both
executable names without requiring symlink privileges.

- [ ] **Step 5: Run installer tests against snapshot artifacts**

```bash
sh scripts/test-install.sh dist/
pwsh -File scripts/test-install.ps1 -DistDir dist
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add scripts docs/install docs/uninstall.md
git commit -m "feat(install): add verified macOS Windows and WSL installers"
```

## Task 10: Ship versioned AI-agent setup prompts

**Files:**

- Create: `docs/install/agent-assisted.md`
- Create: `docs/prompts/README.md`
- Create: `docs/prompts/claude-code-setup.md`
- Create: `docs/prompts/codex-setup.md`
- Create: `docs/prompts/contributor-setup.md`
- Modify: `README.md`
- Modify: `AGENTS.md`
- Create or reduce: `CLAUDE.md`
- Create: `internal/doctest/prompts_test.go`

- [ ] **Step 1: Write prompt contract tests**

Assert every public setup prompt:

- contains a version placeholder or exact release;
- uses the official repository;
- requires verification;
- forbids permission bypass;
- forbids secret/auth access;
- delegates hidden input to `rein init`;
- invokes only commands present in CLI help;
- finishes with `rein doctor --self-test`;
- does not clone the repository in end-user mode.

- [ ] **Step 2: Verify tests fail**

```bash
go test ./internal/doctest -run Prompt -count=1
```

Expected: FAIL because prompts do not exist.

- [ ] **Step 3: Write the shared prompt contract and thin variants**

Keep `AGENTS.md` canonical. `CLAUDE.md` imports `@AGENTS.md` plus only
Claude-specific deltas.

- [ ] **Step 4: Run prompt and docs tests**

```bash
go test ./internal/doctest -count=1
make docs-check
```

Expected: PASS.

- [ ] **Step 5: Manually run each prompt in a safe empty bootstrap directory**

Do not use real credentials or publish external changes during the test.

- [ ] **Step 6: Commit**

```bash
git add README.md AGENTS.md CLAUDE.md docs/install/agent-assisted.md docs/prompts internal/doctest
git commit -m "docs(setup): add tested Claude and Codex bootstrap prompts"
```

## Task 11: Cut the final Phase 0 alpha

**Files:**

- Modify: `CHANGELOG.md`
- Modify: `RELEASING.md` if evidence exposes gaps

- [ ] **Step 1: Run the complete Phase 0 gate**

```bash
make verify
goreleaser release --snapshot --clean
sh scripts/test-install.sh dist/
pwsh -File scripts/test-install.ps1 -DistDir dist
```

Expected: all pass.

- [ ] **Step 2: Execute clean-machine manual acceptance**

Verify macOS, native Windows, and WSL2 installation plus doctor/self-test.

- [ ] **Step 3: Execute Claude and Codex prompt acceptance**

Each agent installs and verifies Reinstate from the same exact draft artifacts.

- [ ] **Step 4: Prepare the alpha release commit**

Update changelog with evidence and known limitations.

- [ ] **Step 5: Commit**

```bash
git add CHANGELOG.md RELEASING.md
git commit -m "chore(release): prepare v0.1.0-alpha.4"
```

- [ ] **Step 6: Tag only after all gates pass**

```bash
git tag -s v0.1.0-alpha.4 -m "Reinstate v0.1.0-alpha.4"
```

Do not push/publish the tag without explicit release authorization.

---

# Chunk 3: Phase 1 — Storage, encryption, path mapping, and sync safety

## Task 12: Define and fake the storage backend

**Files:**

- Create: `internal/backend/backend.go`
- Create: `internal/backend/errors.go`
- Create: `internal/backend/memory/memory.go`
- Create: `internal/backend/memory/memory_test.go`
- Create: `internal/backend/contract_test.go`

- [ ] **Step 1: Write the storage conformance suite**

Contract:

```go
type Backend interface {
    Get(ctx context.Context, key string) (Object, error)
    Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error)
    Delete(ctx context.Context, key string) error
    Head(ctx context.Context, key string) (ObjectInfo, error)
}
```

`PutOptions` must support conditional version/ETag updates.

- [ ] **Step 2: Verify tests fail**

```bash
go test ./internal/backend/... -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement deterministic memory backend**

Support injected failures, partial reads, ETag changes, and concurrent update
tests.

- [ ] **Step 4: Run contract tests**

```bash
gofmt -w internal/backend
go test ./internal/backend/... -count=1
go test -race ./internal/backend/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/backend
git commit -m "feat(storage): define backend contract and memory fake"
```

## Task 13: Implement R2/S3 backend

**Files:**

- Create: `internal/backend/s3/config.go`
- Create: `internal/backend/s3/s3.go`
- Create: `internal/backend/s3/s3_test.go`
- Create: `internal/backend/s3/integration_test.go`
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `docs/storage-backends.md`

- [ ] **Step 1: Apply the backend conformance suite to S3**

Use a fake HTTP server or MinIO-compatible local service; unit tests must not
call real R2/AWS.

- [ ] **Step 2: Verify tests fail**

```bash
go test ./internal/backend/s3 -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement AWS SDK v2 client**

Support endpoint, region, bucket, prefix, conditional writes, streamed
get/put, and normalized errors.

- [ ] **Step 4: Test conditional update conflicts**

Ensure stale ETags fail with the typed conflict error.

- [ ] **Step 5: Run tests**

```bash
gofmt -w internal/backend/s3
go test ./internal/backend/... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/backend/s3 docs/storage-backends.md go.mod go.sum
git commit -m "feat(storage): add R2 and S3 compatible backend"
```

## Task 14: Implement credentials and interactive initialization

**Files:**

- Create: `internal/credentials/store.go`
- Create: `internal/credentials/keyring.go`
- Create: `internal/credentials/environment.go`
- Create: `internal/credentials/store_test.go`
- Create: `internal/initflow/init.go`
- Create: `internal/initflow/init_test.go`
- Create: `internal/cli/init.go`
- Create: `internal/cli/init_test.go`
- Create: `docs/configuration.md`

- [ ] **Step 1: Write failing secret-handling tests**

Assert secrets never enter config, logs, command arguments, doctor reports, or
JSON output.

- [ ] **Step 2: Write failing init-flow tests**

Cover first device, additional device, unavailable keyring fallback,
cancelation, storage probe failure, existing config preservation, and invalid
path mapping.

- [ ] **Step 3: Verify failure**

```bash
go test ./internal/credentials ./internal/initflow ./internal/cli -run 'Credential|Init' -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement keyring abstraction**

Use macOS Keychain, Windows Credential Manager, and Linux Secret Service when
available. WSL/unavailable keyring requires an explicit environment/shared
credential-provider fallback; do not silently write plaintext credentials.

- [ ] **Step 5: Implement interactive init**

Secret input is hidden and never accepted through normal flags.

- [ ] **Step 6: Run tests**

```bash
gofmt -w internal/credentials internal/initflow internal/cli
go test ./internal/credentials ./internal/initflow ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/credentials internal/initflow internal/cli docs/configuration.md go.mod go.sum
git commit -m "feat(init): add secure interactive storage setup"
```

## Task 15: Implement streamed encryption envelopes

**Files:**

- Create: `internal/schema/envelope.go`
- Create: `internal/crypto/envelope.go`
- Create: `internal/crypto/passphrase.go`
- Create: `internal/crypto/hash.go`
- Create: `internal/crypto/envelope_test.go`
- Create: `internal/crypto/testdata/`
- Modify: `docs/security-model.md`

- [ ] **Step 1: Write failing crypto tests**

Cover:

- streamed round-trip;
- wrong passphrase;
- tampered header/body;
- truncated ciphertext;
- empty and large streams;
- deterministic plaintext hash but nondeterministic ciphertext;
- no plaintext fragments in ciphertext;
- passphrase zeroing/bounded lifetime where practical.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/crypto -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement age scrypt envelope**

Do not add a contradictory second KDF. Keep envelope version explicit.

- [ ] **Step 4: Run tests and benchmark**

```bash
gofmt -w internal/crypto internal/schema
go test ./internal/crypto -count=1
go test -race ./internal/crypto -count=1
go test ./internal/crypto -run TestLargeStream -count=1
```

Expected: PASS with bounded memory.

- [ ] **Step 5: Commit**

```bash
git add internal/crypto internal/schema docs/security-model.md go.mod go.sum
git commit -m "feat(crypto): add streamed age encrypted snapshots"
```

## Task 16: Implement project identity and cross-OS path mapping

**Files:**

- Create: `internal/project/id.go`
- Create: `internal/project/id_test.go`
- Create: `internal/pathmap/token.go`
- Create: `internal/pathmap/normalize.go`
- Create: `internal/pathmap/denormalize.go`
- Create: `internal/pathmap/pathmap_test.go`
- Create: `internal/pathmap/fuzz_test.go`
- Create: `docs/path-mapping.md`

- [ ] **Step 1: Write table-driven project identity tests**

Cover SSH/HTTPS remotes, `.git` suffixes, case normalization where appropriate,
subdirectories, no remote, aliases, renamed repos, and worktrees.

- [ ] **Step 2: Write path round-trip and fuzz tests**

Cover every path case in Section 8 and assert unrelated strings remain
unchanged.

- [ ] **Step 3: Verify failure**

```bash
go test ./internal/project ./internal/pathmap -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement canonical identity and path tokens**

Reject ambiguous mappings rather than guessing.

- [ ] **Step 5: Run focused tests and fuzz smoke**

```bash
gofmt -w internal/project internal/pathmap
go test ./internal/project ./internal/pathmap -count=1
go test ./internal/pathmap -run=Fuzz -fuzz=FuzzPathRoundTrip -fuzztime=30s
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/project internal/pathmap docs/path-mapping.md
git commit -m "feat(pathmap): add canonical cross-OS project mapping"
```

## Task 17: Implement manifests and sync planning

**Files:**

- Create: `internal/schema/manifest.go`
- Create: `internal/manifest/local.go`
- Create: `internal/manifest/remote.go`
- Create: `internal/manifest/manifest_test.go`
- Create: `internal/sync/plan.go`
- Create: `internal/sync/plan_test.go`

- [ ] **Step 1: Write failing manifest tests**

Cover empty profile, revisions, parents, session heads, malformed manifests,
future schemas, atomic local updates, and encrypted remote round-trip.

- [ ] **Step 2: Write failing sync-plan tests**

Cover no-op, new local, new remote, equal revision, local-ahead, remote-ahead,
diverged, missing adapter, and unsupported schema.

- [ ] **Step 3: Verify failure**

```bash
go test ./internal/manifest ./internal/sync -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement manifest and pure planning logic**

Keep planning side-effect free and exhaustively testable.

- [ ] **Step 5: Run tests**

```bash
gofmt -w internal/manifest internal/sync internal/schema
go test ./internal/manifest ./internal/sync -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/manifest internal/sync internal/schema
git commit -m "feat(sync): add versioned manifests and sync planning"
```

## Task 18: Implement atomic restore, backup, and locking

**Files:**

- Create: `internal/fsx/backup.go`
- Create: `internal/fsx/backup_test.go`
- Create: `internal/lock/lock.go`
- Create: `internal/lock/lock_test.go`
- Create: `internal/restore/restore.go`
- Create: `internal/restore/restore_test.go`
- Create: `docs/backup-and-recovery.md`

- [ ] **Step 1: Write failure-injection tests**

Cover:

- interrupted copy;
- hash mismatch;
- rename failure;
- disk-full simulation;
- active lock;
- stale lock;
- backup creation and rollback;
- target unchanged on every pre-commit failure.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/fsx ./internal/lock ./internal/restore -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement lock, backup, and restore transaction**

Never mutate vendor files before validation and backup are complete.

- [ ] **Step 4: Run tests and race detector**

```bash
gofmt -w internal/fsx internal/lock internal/restore
go test ./internal/fsx ./internal/lock ./internal/restore -count=1
go test -race ./internal/lock ./internal/restore -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/fsx internal/lock internal/restore docs/backup-and-recovery.md
git commit -m "feat(restore): add atomic backup and rollback safety"
```

## Task 19: Implement push, pull, and conflicts against synthetic payloads

**Files:**

- Create: `internal/sync/push.go`
- Create: `internal/sync/pull.go`
- Create: `internal/sync/conflict.go`
- Create: `internal/sync/sync_test.go`
- Create: `testdata/sync/`

- [ ] **Step 1: Write end-to-end fake-backend tests**

Cover:

- first push/pull;
- dry-run no mutation;
- second no-op;
- local-ahead;
- remote-ahead;
- simultaneous divergence;
- stale ETag;
- wrong passphrase;
- tampered object;
- interrupted upload/download;
- keep-local/remote/both resolution.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/sync -run 'Push|Pull|Conflict' -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement orchestration using interfaces**

No command/UI concerns belong in the sync engine.

- [ ] **Step 4: Run tests and race detector**

```bash
gofmt -w internal/sync
go test ./internal/sync -count=1
go test -race ./internal/sync -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sync testdata/sync
git commit -m "feat(sync): add encrypted push pull and conflicts"
```

---

# Chunk 4: Phase 1 — Claude and Codex adapters

## Task 20: Implement Claude Code detection and discovery

**Files:**

- Create: `internal/adapter/claude/detect.go`
- Create: `internal/adapter/claude/discover.go`
- Create: `internal/adapter/claude/exclusions.go`
- Create: `internal/adapter/claude/detect_test.go`
- Create: `internal/adapter/claude/discover_test.go`
- Modify: `testdata/adapters/claude/...`
- Create: `docs/adapters/claude-code.md`

- [ ] **Step 1: Write fixture-backed detection/discovery tests**

Cover macOS, native Windows, WSL2, missing roots, multiple projects, malformed
JSONL, and hard exclusions.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/adapter/claude -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement read-only detection and discovery**

Do not read credential files or traverse excluded directories.

- [ ] **Step 4: Run tests**

```bash
gofmt -w internal/adapter/claude
go test ./internal/adapter/claude -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/claude testdata/adapters/claude docs/adapters/claude-code.md
git commit -m "feat(claude): discover compatible Claude Code sessions"
```

## Task 21: Implement Claude transform/export/restore

**Files:**

- Create: `internal/adapter/claude/transform.go`
- Create: `internal/adapter/claude/export.go`
- Create: `internal/adapter/claude/restore.go`
- Create: `internal/adapter/claude/roundtrip_test.go`
- Modify: `testdata/adapters/claude/...`

- [ ] **Step 1: Write failing golden round-trip tests**

Assert:

- known structural paths transform;
- unknown events/fields survive;
- prose and code blocks do not change;
- project directory identity remaps;
- malformed lines fail safely;
- excluded files cannot enter an export plan;
- restore output is recognized by the target layout.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/adapter/claude -run 'Transform|RoundTrip|Restore' -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement streaming JSONL transformations**

Preserve record ordering and newline semantics.

- [ ] **Step 4: Run tests and a large fixture**

```bash
gofmt -w internal/adapter/claude
go test ./internal/adapter/claude -count=1
go test ./internal/adapter/claude -run TestLargeSession -count=1
```

Expected: PASS with bounded memory.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/claude testdata/adapters/claude
git commit -m "feat(claude): add safe cross-OS session round trips"
```

## Task 22: Validate Claude on real devices

**Files:**

- Modify: `docs/compatibility.md`
- Modify: `docs/adapters/claude-code.md`
- Modify: `CHANGELOG.md`
- Store only redacted evidence summaries; never commit real sessions

- [ ] **Step 1: Install the same draft build on Windows and macOS**

- [ ] **Step 2: Validate Windows-to-macOS**

Push a designated non-sensitive test session, pull with dry run, restore, and
resume using the supported Claude command.

- [ ] **Step 3: Validate macOS-to-Windows**

Repeat in the opposite direction.

- [ ] **Step 4: Validate conflict and rollback**

Create deliberate divergence; verify no silent overwrite and successful
resolution/rollback.

- [ ] **Step 5: Record exact versions and outcomes**

Document agent versions, platforms, Reinstate commit/artifact checksums,
commands, and redacted results.

- [ ] **Step 6: Run complete automated gate**

```bash
make verify
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add docs/compatibility.md docs/adapters/claude-code.md CHANGELOG.md
git commit -m "test(claude): record cross-device resume evidence"
```

- [ ] **Step 8: Prepare `v0.1.0-beta.1` only if independently releasable**

Do not tag if any Claude beta gate is incomplete.

## Task 23: Implement Codex detection and discovery

**Files:**

- Create: `internal/adapter/codex/detect.go`
- Create: `internal/adapter/codex/discover.go`
- Create: `internal/adapter/codex/exclusions.go`
- Create: `internal/adapter/codex/detect_test.go`
- Create: `internal/adapter/codex/discover_test.go`
- Modify: `testdata/adapters/codex/...`
- Create: `docs/adapters/codex.md`

- [ ] **Step 1: Write fixture-backed tests**

Cover date-partitioned rollouts, session metadata, large files, native Windows,
macOS, WSL2, missing roots, malformed lines, state index variants, and hard
exclusions.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/adapter/codex -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement detection/discovery**

Auth and credential material is never opened.

- [ ] **Step 4: Run tests**

```bash
gofmt -w internal/adapter/codex
go test ./internal/adapter/codex -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/codex testdata/adapters/codex docs/adapters/codex.md
git commit -m "feat(codex): discover compatible Codex sessions"
```

## Task 24: Implement Codex transform/export/restore

**Files:**

- Create: `internal/adapter/codex/transform.go`
- Create: `internal/adapter/codex/export.go`
- Create: `internal/adapter/codex/restore.go`
- Create: `internal/adapter/codex/index.go`
- Create: `internal/adapter/codex/roundtrip_test.go`
- Modify: `testdata/adapters/codex/...`

- [ ] **Step 1: Write failing round-trip tests**

Assert:

- session identity survives;
- known path metadata transforms;
- transcript prose/code does not change;
- unknown records survive;
- large rollouts stream with bounded memory;
- current Codex can discover restored rollouts;
- index handling never performs undocumented blind database copying;
- incompatible state variants fail closed.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/adapter/codex -run 'Transform|RoundTrip|Restore|Index' -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement the minimal verified restore strategy**

Prefer vendor-recognized rollout restoration and index reconstruction behavior
proven by tests. Do not mutate SQLite based on guessed schemas.

- [ ] **Step 4: Run tests and large-rollout verification**

```bash
gofmt -w internal/adapter/codex
go test ./internal/adapter/codex -count=1
go test ./internal/adapter/codex -run TestLargeRollout -count=1
```

Expected: PASS with bounded memory.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/codex testdata/adapters/codex
git commit -m "feat(codex): add safe cross-OS rollout round trips"
```

## Task 25: Validate Codex on real devices

**Files:**

- Modify: `docs/compatibility.md`
- Modify: `docs/adapters/codex.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Install the same draft build on Windows and macOS**

- [ ] **Step 2: Validate Windows-to-macOS resume**

- [ ] **Step 3: Validate macOS-to-Windows resume**

- [ ] **Step 4: Validate large rollout behavior**

- [ ] **Step 5: Validate conflict and rollback**

- [ ] **Step 6: Record exact versions, artifact checksum, and redacted evidence**

- [ ] **Step 7: Run complete automated gate**

```bash
make verify
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add docs/compatibility.md docs/adapters/codex.md CHANGELOG.md
git commit -m "test(codex): record cross-device resume evidence"
```

- [ ] **Step 9: Prepare `v0.1.0-beta.2` only if independently releasable**

---

# Chunk 5: Phase 1 — Complete CLI, docs, community, and stable release

## Task 26: Wire complete CLI commands

**Files:**

- Create: `internal/cli/list.go`
- Create: `internal/cli/status.go`
- Create: `internal/cli/diff.go`
- Create: `internal/cli/push.go`
- Create: `internal/cli/pull.go`
- Create: `internal/cli/conflicts.go`
- Create: matching `*_test.go` files
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write command-level behavior tests**

Cover all commands, flags, JSON schemas, exit codes, dry-run guarantees,
interactive refusal in non-TTY mode, and redaction.

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/cli -count=1
```

Expected: FAIL for unwired commands.

- [ ] **Step 3: Wire commands to core interfaces**

Keep orchestration/business logic out of CLI files.

- [ ] **Step 4: Generate and test completions**

Validate Bash, Zsh, Fish, and PowerShell output.

- [ ] **Step 5: Run focused and full tests**

```bash
gofmt -w internal/cli
go test ./internal/cli -count=1
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli
git commit -m "feat(cli): expose complete phase 1 sync workflow"
```

## Task 27: Complete human documentation

**Files:**

- Modify: `README.md`
- Modify: `docs/README.md`
- Create/update every document in Section 10
- Modify: `CONTRIBUTING.md`
- Modify: `SUPPORT.md`
- Modify: `SECURITY.md`
- Modify: `GOVERNANCE.md`
- Modify: `RELEASING.md`
- Modify: `ROADMAP.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Generate CLI reference from actual help**

Use a checked generator or golden test so docs cannot invent commands.

- [ ] **Step 2: Write the complete two-device manual journeys**

Separate macOS, native Windows, and WSL2. Include first device, additional
device, dry-run, resume, conflicts, backup recovery, uninstall, and failure
paths.

- [ ] **Step 3: Update support/security truth**

Support only the latest stable pre-1.0 line; do not claim `main` is a supported
release. State that `v0.1.0` is experimental and not externally audited.

- [ ] **Step 4: Run docs checks**

```bash
make docs-check
git diff --check
```

Expected: PASS.

- [ ] **Step 5: Run every documented command against a temporary profile**

No command example may rely on planned or nonexistent behavior.

- [ ] **Step 6: Commit**

```bash
git add README.md docs CONTRIBUTING.md SUPPORT.md SECURITY.md GOVERNANCE.md RELEASING.md ROADMAP.md CHANGELOG.md
git commit -m "docs(v0.1): publish complete setup and recovery guides"
```

## Task 28: Complete contribution and issue workflows

**Files:**

- Create: `.github/ISSUE_TEMPLATE/docs_report.yml`
- Create: `.github/ISSUE_TEMPLATE/compatibility_regression.yml`
- Modify: `.github/PULL_REQUEST_TEMPLATE.md`
- Modify: `.github/CODEOWNERS`
- Modify: `CONTRIBUTING.md`
- Create: `docs/contributing/development.md`
- Create: `docs/contributing/documentation.md`
- Create: `docs/contributing/release-process.md`

- [ ] **Step 1: Add structured issue forms**

Compatibility reports require source/destination OS, native/WSL, Reinstate
version, agent version, failing stage, and confirmation of redaction.

- [ ] **Step 2: Add PR compatibility/security checklist**

Require tests, docs, changelog, migration notes, fixture declaration, and
secret-handling impact.

- [ ] **Step 3: Document RFC boundaries and contribution licensing**

No mandatory CLA. If DCO is enabled, document sign-off clearly and keep it
low-friction.

- [ ] **Step 4: Validate forms and links**

```bash
make docs-check
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .github CONTRIBUTING.md docs/contributing
git commit -m "docs(contributing): add adapter and compatibility workflows"
```

## Task 29: Execute full release-candidate acceptance

**Files:**

- Modify only evidence/docs/changelog files required by actual results

- [ ] **Step 1: Run complete automated verification**

```bash
make verify
goreleaser release --snapshot --clean
```

Expected: zero failures.

- [ ] **Step 2: Verify exact snapshot archives**

Install and run from archives rather than the repository build directory.

- [ ] **Step 3: Execute the four real-device resume journeys**

1. Claude Windows to macOS
2. Claude macOS to Windows
3. Codex Windows to macOS
4. Codex macOS to Windows

- [ ] **Step 4: Execute WSL2 smoke**

Treat WSL2 as a separate Linux-like device and never reuse native Windows state.

- [ ] **Step 5: Execute both AI-assisted setup prompts**

Claude Code and Codex must each complete the version-pinned setup workflow.

- [ ] **Step 6: Execute safety drills**

Cover wrong passphrase, tampered ciphertext, interrupted restore, concurrent
push conflict, keep-local/remote/both, rollback, and credential-exclusion scan.

- [ ] **Step 7: Measure activation**

Installation-to-first-resume target is under five minutes after host-agent auth
and storage availability.

- [ ] **Step 8: Record redacted evidence**

Include versions, platforms, checksums, commands, pass/fail, and known
limitations. Never commit real sessions or secrets.

- [ ] **Step 9: Re-run full verification after evidence/doc updates**

```bash
make verify
git diff --check
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add docs CHANGELOG.md
git commit -m "test(release): validate phase 1 cross-device workflows"
```

## Task 30: Prepare and release `v0.1.0`

**Files:**

- Modify: `CHANGELOG.md`
- Modify: version/release metadata only as required by the chosen build process

- [ ] **Step 1: Prepare `v0.1.0-rc.1` release PR**

Move `[Unreleased]` entries into a dated RC/stable section as appropriate and
open a fresh empty `[Unreleased]`.

- [ ] **Step 2: Run the immutable release gate**

Use the exact release workflow and draft artifacts.

- [ ] **Step 3: Install exact draft artifacts on required platforms**

- [ ] **Step 4: Verify checksums, SBOM, and attestation**

- [ ] **Step 5: Publish RC only with explicit authorization**

- [ ] **Step 6: Repeat acceptance against RC artifacts**

- [ ] **Step 7: Prepare stable release commit**

```bash
git add CHANGELOG.md
git commit -m "chore(release): prepare v0.1.0"
```

- [ ] **Step 8: Run final fresh verification**

```bash
make verify
```

Expected: PASS with no skipped required gate.

- [ ] **Step 9: Create signed annotated stable tag**

```bash
git tag -s v0.1.0 -m "Reinstate v0.1.0"
```

- [ ] **Step 10: Push/publish only with explicit release authorization**

Verify local/remote SHA and tag ancestry after push. Never move or replace the
published tag/assets.

---

## 13. CI and test matrix

### Required PR checks

- `gofmt` diff check
- `go vet ./...`
- pinned hard-failing linter
- unit tests on Ubuntu, macOS, and Windows
- race tests on Ubuntu
- config/schema migration tests
- Claude/Codex golden fixtures
- path mapping round-trip and fuzz tests
- crypto vectors and tamper/plaintext-leak tests
- atomic restore/rollback/failure-injection tests
- fake-storage sync and concurrency tests
- POSIX and PowerShell installer tests
- CLI help/documentation drift checks
- Markdown links
- secret scanning
- `govulncheck`
- CodeQL
- dependency review

### Scheduled compatibility checks

- current Claude Code fixture compatibility
- current Codex fixture compatibility
- fake S3 integration
- latest supported Go smoke
- release-installer smoke
- upstream format drift issue creation

Scheduled checks may open an issue. They must never auto-edit adapter code or
silently expand the support matrix.

---

## 14. Roadmap after Phase 1

| Phase | Version direction | Gate |
| --- | --- | --- |
| Phase 2: Environment plane | `v0.2.x` | Fresh machine restores safe Claude/Codex MCP, skills, instruction files, and non-secret settings |
| Phase 3: Adapter expansion | `v0.3.x` | Gemini CLI and OpenCode pass the same cross-OS fixture/security contract |
| Phase 4: Habit and trust | `v0.4.x` | Two weeks of unattended hooks/delta sync with zero unresolved conflicts |
| Phase 5: Checkpoints/handoffs | `v0.5.x` | Portable handoff packages help another agent continue without claiming native transcript replay |
| Phase 6: Adapter SDK | `v0.6+` | External adapter passes a public conformance kit without core changes |
| Convenience layer | Later | Optional zero-knowledge relay, local browser UI, selected team sharing |
| Stable contract | `v1.0.0` | Stable CLI/config/remote schemas, migrations, external security review, and sustained adapter compatibility |

---

## 15. Kill and pivot signals

| Signal | Required response |
| --- | --- |
| Vendor format churn repeatedly breaks restore | Freeze adapter expansion and move toward official hooks/app-server surfaces |
| Native vendor sync solves same-agent cross-OS resume | Emphasize universal environment/config portability |
| Any plaintext or credential leak | Stop releases immediately and follow private security response |
| Restore recovery rate is materially non-trivial | Freeze roadmap expansion and fix restore correctness |
| Setup remains over five minutes | Improve bootstrap/pairing before adding adapters |
| S3 credential setup dominates failures | Prioritize safer pairing/credential UX in Phase 2 |

---

## 16. Definition of done

Phase 0 is done only when:

- a clean macOS machine installs and verifies Reinstate;
- a clean native Windows machine installs and verifies Reinstate;
- WSL2 is separately installed/detected/documented;
- Claude Code and Codex both complete the approved agent-assisted setup prompt;
- doctor/self-test is redacted and uses only synthetic data;
- docs, version claims, installers, and release automation describe reality.

Phase 1 is done only when:

- all four Windows/macOS Claude/Codex resume journeys pass using the same exact
  release candidate;
- dry-run, backup, atomic restore, conflicts, wrong-passphrase, tamper, and
  rollback behavior pass;
- remote storage contains no credential files or plaintext session content;
- end-user and contributor prompts pass their contract tests;
- manual setup works without source-code knowledge;
- all required CI/release gates pass;
- `v0.1.0` artifacts, checksums, SBOM, attestation, changelog, and documentation
  agree;
- the signed immutable `v0.1.0` release is published only after explicit release
  authorization.

Anything less is an alpha/beta, not `v0.1.0`.
