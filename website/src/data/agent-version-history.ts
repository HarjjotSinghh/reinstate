import { releaseHistory } from './releases';

export interface AgentVersionChange {
  version: (typeof releaseHistory)[number]['version'];
  date: (typeof releaseHistory)[number]['date'];
  rangeChange: string;
  compatibilityChange: string;
  source: string;
  implementationSource?: string;
}

const evidenceByVersion: Record<
  AgentVersionChange['version'],
  Pick<
    AgentVersionChange,
    'rangeChange' | 'compatibilityChange' | 'implementationSource'
  >
> = {
  'v0.5.0-rc.6': {
    rangeChange:
      'Unchanged from v0.5.0-rc.5: inclusive Claude Code range 2.1.219-2.1.238 and Codex CLI range 0.133.0-0.149.0.',
    compatibilityChange:
      'Sixth Phase 5 candidate. Carries the single defect physical v0.5.0-rc.5 acceptance found: the agent probe emitted a raw 38-character Git object hash, because the shape normaliser recognised only exactly 32, 40 and 64 characters while OpenCode keeps a Git object store under each snapshot. A committed probe artifact must not carry a content hash of the operator own repository. Claude Code and Codex remain the only T3-T5 surfaces. This candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.6/docs/testing/v0.5.0-rc.6-agent-verification-prompts.md',
  },
  'v0.5.0-rc.5': {
    rangeChange:
      'Raises both inclusive ranges on dual-platform physical evidence: Claude Code to 2.1.219-2.1.238 and Codex CLI to 0.133.0-0.149.0. On macOS and on Windows a session created with the new version was indexed by Reinstate, resumed through the launch plan Reinstate produced, and returned a token that existed only in the original session history.',
    compatibilityChange:
      'Fifth Phase 5 candidate. Fixes an index that never re-read a source after Reinstate itself changed, so a reader fix reached nobody who already had an index. Also recovers Gemini project paths on case-insensitive filesystems, makes an agent root override authoritative when it names a missing path, and skips parsing an unchanged source. Claude Code and Codex remain the only T3-T5 surfaces. This candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.5/docs/testing/v0.5.0-rc.5-agent-verification-prompts.md',
  },
  'v0.5.0-rc.4': {
    rangeChange:
      'Raises both inclusive ranges on dual-platform physical evidence: Claude Code to 2.1.219-2.1.238 and Codex CLI to 0.133.0-0.149.0. On macOS and on Windows a session created with the new version was indexed by Reinstate, resumed through the launch plan Reinstate produced, and returned a token that existed only in the original session history.',
    compatibilityChange:
      'Fourth Phase 5 candidate. Makes the agent probe reproducible, normalizes raw content hashes out of probe output, refuses native actions on a T0 agent with the compatibility exit code, narrows an agent-filtered refresh to that agent, and completes agent keys per flag. Claude Code and Codex remain the only T3-T5 surfaces. This candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.4/docs/testing/v0.5.0-rc.4-agent-verification-prompts.md',
  },
  'v0.5.0-rc.3': {
    rangeChange:
      'Unchanged from v0.5.0-rc.2: inclusive Claude Code range 2.1.219-2.1.229. The Codex CLI range is unchanged at 0.133.0-0.147.0.',
    compatibilityChange:
      'Third Phase 5 candidate. Restores Kimi, Copilot, and OpenCode session discovery after those vendor CLIs changed their on-disk formats. Claude Code and Codex remain the only T3-T5 surfaces. This candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.3/docs/testing/v0.5.0-rc.3-agent-verification-prompts.md',
  },
  'v0.5.0-rc.2': {
    rangeChange:
      'Unchanged from v0.5.0-rc.1: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Second Phase 5 candidate after v0.5.0-rc.1 dual-platform tagged-artifact acceptance FAILED. Structured-handoff projection pairs tool_result with its tool_call. Claude Code and Codex remain the only T3–T5 surfaces. This candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.2/docs/testing/v0.5.0-rc.2-agent-verification-prompts.md',
  },
  'v0.5.0-rc.1': {
    rangeChange:
      'Unchanged from v0.4.0: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'First Phase 5 candidate. Six T1 discover agents and three T2 handoff sources. Claude Code and Codex remain the only T3–T5 surfaces. Dual-platform tagged-artifact acceptance FAILED (macOS 88/150, Windows 93/150); this candidate does not authorize stable v0.5.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.5.0-rc.1/docs/testing/v0.5.0-rc.1-agent-verification-prompts.md',
  },
  'v0.4.0': {
    rangeChange:
      'Unchanged from v0.4.0-rc.11: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Phase 4 stable after dual-platform tagged-artifact acceptance PASS on v0.4.0-rc.11 (Apple Silicon macOS 44/44, native Windows x64 44/44). Structured handoff continues the same task in a new Claude Code or Codex session. Native resume remains same-vendor. Gemini CLI, OpenCode, and Grok Build remain handoff sources only.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0/docs/testing/results/2026-08-15-macos-phase4-V040RC11.md',
  },
  'v0.4.0-rc.11': {
    rangeChange:
      'Unchanged from v0.4.0-rc.10: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Eleventh Phase 4 candidate after v0.4.0-rc.10 physical dual-platform acceptance FAILED (macOS 41/44 A1/A3/A7, Windows 40/44 A1/A2/A3/A5). This candidate requires the five-bullet dest first-reply in bootstrap and Windows one-line argv, records lineage before dest Launch, recovers artifact dirs in handoff list, and Materializes dest-home workspace trust. Dual-platform tagged-artifact acceptance later PASS (macOS 44/44, Windows 44/44); promoted to stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.11/internal/handoff/desttrust.go',
  },
  'v0.4.0-rc.10': {
    rangeChange:
      'Unchanged from v0.4.0-rc.9: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Tenth Phase 4 candidate after v0.4.0-rc.9 physical dual-platform acceptance FAILED (macOS 38/44, Windows 38/44). This candidate falls back to a one-line absolute projection.md pointer whenever dest briefing contains CR/LF so Windows CreateProcess cannot truncate argv. Physical dual-platform acceptance FAILED (macOS 41/44 A1/A3/A7, Windows 40/44 A1/A2/A3/A5; remaining product defects dest first-reply, lineage-after-launch, dest folder-trust); this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.10/internal/handoff/pipeline.go',
  },
  'v0.4.0-rc.9': {
    rangeChange:
      'Unchanged from v0.4.0-rc.8: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Ninth Phase 4 candidate after v0.4.0-rc.8 physical dual-platform acceptance FAILED (macOS 38/44, Windows 38/44). This candidate maps a recognized off-PATH layout to inspect JSON status=supported without claiming a verified version range, and still fail-closes destination launch when the executable is missing. Physical dual-platform acceptance FAILED (macOS 38/44, Windows 38/44; dest-ack A4 PASS, A1/A2/A3/A5/A6/A7 FAIL; remaining product defect Windows dest argv CR/LF truncation); this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.9/internal/agentcheck/agent.go',
  },
  'v0.4.0-rc.8': {
    rangeChange:
      'Unchanged from v0.4.0-rc.7: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Eighth Phase 4 candidate after v0.4.0-rc.7 physical dual-platform acceptance FAILED (macOS 38/44, Windows 34/44). This candidate scans recognized Claude layout on off-PATH Inspect instead of returning before the layout scan (R1) and pins Go 1.25.13 for govulncheck. Physical dual-platform acceptance FAILED (macOS 38/44, Windows 38/44; remaining product defect Windows inspect JSON not_installed); this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.8/internal/agentcheck/agent.go',
  },
  'v0.4.0-rc.7': {
    rangeChange:
      'Unchanged from v0.4.0-rc.6: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Seventh Phase 4 candidate after v0.4.0-rc.6 physical dual-platform acceptance FAILED. This candidate bounds Windows process listing, fails closed on determined out-of-range Claude (2.1.230) during read-only handoff, classifies hung --version as Compatibility UNTESTED instead of Runtime, and does not block source-only Grok preflight on missing native layout. Physical dual-platform acceptance FAILED (macOS 38/44, Windows 34/44; remaining product defect R1 off-PATH Inspect); this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.7/internal/processcheck/process_windows.go',
  },
  'v0.4.0-rc.6': {
    rangeChange:
      'Unchanged from v0.4.0-rc.5: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Sixth Phase 4 candidate after v0.4.0-rc.5 physical dual-platform acceptance FAILED. This candidate remaps foreign/fixture-user workspaces only when the project leaf matches the cwd git repo, refuses a different-repository bind, and refuses non-TTY destination launch before index open. Dual-platform tagged-artifact acceptance is pending; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.6/internal/handoff/workspace.go',
  },
  'v0.4.0-rc.5': {
    rangeChange:
      'Unchanged from v0.4.0-rc.4: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Fifth Phase 4 candidate after v0.4.0-rc.4 physical dual-platform acceptance FAILED. This candidate kills hanging Windows --version process trees, refuses non-TTY destination launch before Plan, remaps foreign-OS and fixture-user workspaces onto the local git checkout, and classifies Codex compaction as summarized. Physical dual-platform acceptance FAILED; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.5/internal/adapter/exec_windows.go',
  },
  'v0.4.0-rc.4': {
    rangeChange:
      'Unchanged from v0.4.0-rc.3: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Fourth Phase 4 candidate after v0.4.0-rc.3 physical dual-platform acceptance FAILED. This candidate treats an explicit empty dest home as supported, bounds hanging --version probes including grandchild-held pipes, persists --no-launch without warning acks, keeps sidecared summarized fidelity, and fixes Windows passphrase-FD / isolation-env unit failures. Physical dual-platform acceptance FAILED; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.4/internal/adapter/version.go',
  },
  'v0.4.0-rc.3': {
    rangeChange:
      'Unchanged from v0.4.0-rc.2: inclusive Claude Code range 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Third Phase 4 candidate after v0.4.0-rc.2 physical dual-platform acceptance FAILED. This candidate carries C4 (wrong-repo cwd refused), F8 (non-TTY fail-closed before spawn), Grok-source busy-check, R4 timed-out probe classified UNTESTED/Compatibility, plus remaining matrix product defects (--no-redact, list isolation, fidelity/projection_events, checkpoint sidecar refs, destination MCP/skill gaps). Physical dual-platform acceptance FAILED; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.3/internal/handoff/projection.go',
  },
  'v0.4.0-rc.2': {
    rangeChange:
      'Widened the inclusive Claude Code range from 2.1.219–2.1.228 to 2.1.219–2.1.229. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'Second Phase 4 candidate. v0.4.0-rc.1 physical acceptance FAILED, and this candidate fixes what it found: Claude-sourced handoffs no longer depend on a version file real installs never create, reader-emitted paths are tokenized before capsule validation, live changed files reach the destination, a timed-out version probe is classified UNTESTED instead of accepted, and capsule validation checks path-typed fields instead of rejecting prose that starts with a slash. Physical dual-platform acceptance FAILED; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.2/internal/handoff/projection.go',
  },
  'v0.4.0-rc.1': {
    rangeChange:
      'Widened the inclusive Claude Code range from 2.1.219–2.1.227 to 2.1.219–2.1.228. The Codex CLI range is unchanged at 0.133.0–0.147.0.',
    compatibilityChange:
      'First Phase 4 candidate: explicit structured handoff of the same task into a new Claude Code or Codex session. Gemini CLI, OpenCode, and Grok Build are handoff sources only. The 2.1.228 patch is source-tested only, and dual-platform tagged-artifact acceptance is pending; this candidate does not authorize stable v0.4.0.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.4.0-rc.1/internal/handoff/projection.go',
  },
  'v0.3.0': {
    rangeChange:
      'Claude Code 2.1.219–2.1.227 and Codex CLI 0.133.0–0.147.0 unchanged from RC7.',
    compatibilityChange:
      'Stable Phase 3 release after dual-platform tagged-artifact acceptance PASS on v0.3.0-rc.7. Fresh stable dual validation is required after tag publication.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0/docs/testing/results/2026-08-11-macos-phase3-V030RC7.md',
  },
  'v0.3.0-rc.7': {
    rangeChange:
      'Claude Code 2.1.219–2.1.227 and Codex CLI 0.133.0–0.147.0 unchanged from RC6.',
    compatibilityChange:
      'Packages the post-RC6 Phase 3 harden stack (non-TTY fail-closed native launch, Windows Ctrl+C safety, capability probe demotion, isolated CLAUDE_CONFIG_DIR/CODEX_HOME roots, expanded local smoke) without widening Claude Code 2.1.219–2.1.227 or Codex CLI 0.133.0–0.147.0. Tagged-artifact acceptance is pending.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.7/scripts/testing/phase3-local-smoke.sh',
  },
  'v0.3.0-rc.6': {
    rangeChange:
      'Expanded the inclusive Claude Code range from 2.1.219–2.1.220 to 2.1.219–2.1.227 and the Codex CLI range from 0.133.0–0.146.0 to 0.133.0–0.147.0.',
    compatibilityChange:
      'Widened fail-closed product ranges so primary-host Claude Code 2.1.225/2.1.227 and Codex CLI 0.147.0 installs were SUPPORTED for dual-platform Phase 3 retest. Dual-platform tagged-artifact acceptance failed on real-launch baseline, TTY/picker, and related host evidence rows.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.6/internal/adapter/claude/claude.go',
  },
  'v0.3.0-rc.5': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Carries the RC4 Windows-first product fixes and makes PowerShell release-artifact verification portable without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Tagged-artifact acceptance is pending.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.5/scripts/check-release-artifacts.ps1',
  },
  'v0.3.0-rc.4': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Fixes the RC3 native-Windows PowerShell 5.1 staging and human-output privacy blockers without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Its release workflow failed before publication.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.4/internal/doctor/redact.go',
  },
  'v0.3.0-rc.3': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Hardens Windows trusted executable resolution and PowerShell snapshot/staging gates after RC2 native Windows FAIL without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Native Windows acceptance still failed on later staging/privacy gates.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.3/internal/executabletrust/resolve.go',
  },
  'v0.3.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Corrects Windows trusted executable resolution for extensionless vendor names via PATHEXT without widening Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0. Tagged-artifact acceptance is pending.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.2/internal/executabletrust/resolve.go',
  },
  'v0.3.0-rc.1': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Adds Phase 3 verified-resume observation and authorization without widening the Claude Code 2.1.219–2.1.220 or Codex CLI 0.133.0–0.146.0 ranges. macOS acceptance passed; native Windows failed.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.3.0-rc.1/internal/preflight/verify.go',
  },
  'v0.2.0': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Ships the RC2-tested runtime adapter tree unchanged. Apple Silicon macOS and native Windows x64 are verified stable platforms; Intel macOS and Linux/WSL2 artifacts remain preview until their deferred physical acceptance completes.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.2.0/internal/adapter/codex/codex.go',
  },
  'v0.2.0-rc.3': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No adapter behavior change from v0.2.0-rc.2. This candidate adds verified package-manager distribution metadata and native package artifacts without widening the tested Claude Code or Codex CLI ranges.',
  },
  'v0.2.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No adapter behavior change from v0.2.0-rc.1. This candidate corrects release-artifact provenance and native-Windows verification without widening the tested Claude Code or Codex CLI ranges.',
  },
  'v0.2.0-rc.1': {
    rangeChange:
      'Expanded the inclusive Codex CLI range from 0.133.0–0.145.0 to 0.133.0–0.146.0; the Claude Code range remains 2.1.219–2.1.220.',
    compatibilityChange:
      'Codex CLI 0.146.0 passed the complete Phase 2 physical matrix on both macOS and native Windows. The adapter still fails closed for stable versions above 0.146.0 and for every prerelease.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.2.0-rc.1/internal/adapter/codex/codex.go',
  },
  'v0.1.0': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'No behavior change from v0.1.0-rc.8. The stable release ships that candidate\u2019s product code unchanged, so the two-device Phase 1 acceptance evidence applies directly to it.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0/internal/processcheck/process.go',
  },
  'v0.1.0-rc.8': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Restore liveness no longer depends on an open file handle. Claude Code closes its session file between appends, so a live session held no handle and RC7 treated it as free. Detection now also matches an agent naming the exact session on its command line, or working inside the session\u2019s mapped project.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.8/internal/processcheck/process.go',
  },
  'v0.1.0-rc.7': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Restore liveness detection became session-scoped. A restore now checks whether the exact target session file is held open, using operating-system file handles, instead of asking whether any Claude Code or Codex process is running. A session that is genuinely in use is restored alongside the live one rather than refused.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.7/internal/processcheck/process.go',
  },
  'v0.1.0-rc.6': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Codex working directories resolve to canonical project IDs. Additional-device init changed from RC5’s metadata-only manifest probe to reading the complete remote ciphertext; init still does not decrypt it or validate the passphrase.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.6/internal/cli/commands_impl.go',
  },
  'v0.1.0-rc.5': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Additional-device init required a metadata-only existence probe for the remote manifest before saving local setup; it did not read or decrypt the ciphertext.',
    implementationSource:
      'https://github.com/HarjjotSinghh/reinstate/blob/v0.1.0-rc.5/internal/cli/commands_impl.go',
  },
  'v0.1.0-rc.4': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Claude restore paths became destination-aware, legacy unmapped snapshots fail closed, and exact native restore locations are verified.',
  },
  'v0.1.0-rc.3': {
    rangeChange:
      'Introduced the current inclusive ranges: Claude Code 2.1.219–2.1.220 and Codex CLI 0.133.0–0.145.0.',
    compatibilityChange:
      'Installed versions outside a tested stable range become UNTESTED and setup check exits with compatibility code 5.',
  },
  'v0.1.0-rc.2': {
    rangeChange: 'No agent-version range change documented.',
    compatibilityChange:
      'Release CI corrected signed-tag verification; no adapter compatibility change is recorded.',
  },
  'v0.1.0-rc.1': {
    rangeChange:
      'Claude Code and Codex adapters launched with fixture-backed checks, but this changelog does not publish an exact historical version range.',
    compatibilityChange:
      'Established fail-closed adapter compatibility states and same-vendor session restore.',
  },
};

export const agentVersionHistory: AgentVersionChange[] = releaseHistory.map(
  ({ version, date }) => ({
    version,
    date,
    ...evidenceByVersion[version],
    source: `https://github.com/HarjjotSinghh/reinstate/blob/${version}/CHANGELOG.md`,
  }),
);
