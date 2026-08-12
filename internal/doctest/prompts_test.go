package doctest

import (
	"strings"
	"testing"
)

func TestEndUserPromptContracts(t *testing.T) {
	for _, path := range []string{
		"docs/prompts/claude-code-setup.md",
		"docs/prompts/codex-setup.md",
	} {
		body := read(t, path)
		required := []string{
			"Prompt version:** 9",
			"v0.4.0-rc.2",
			"https://reinstate.dev/install.sh",
			"https://reinstate.dev/install.ps1",
			"github.com/HarjjotSinghh/reinstate/releases/download/",
			"checksum",
			"rein init",
			"rein doctor --self-test",
			"rein push --agent",
			"rein pull --agent",
			"--dry-run",
			"hidden",
			"credential",
			"passphrase",
			"approval",
			"visibly",
			"SESSION_ID",
			"REINSTATE_HOME",
			"Never unset",
			"effective home",
		}
		for _, value := range required {
			if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
				t.Errorf("%s is missing prompt contract %q", path, value)
			}
		}
		for _, forbidden := range []string{
			"git clone ",
			"--dangerously-skip-permissions",
			"REINSTATE_PASSPHRASE=",
			"releases/latest",
			"<REINSTATE_VERSION>",
		} {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s contains forbidden end-user instruction %q", path, forbidden)
			}
		}
	}
}

func TestRC4AcceptancePromptContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-1-rc4-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.1.0-rc.4",
		"MAC-RC4-M1",
		"WINDOWS-RC4-W1-PASS",
		"local/reinstate-phase1-acceptance-rc4",
		"REINSTATE-PHASE1-RC4-MAC-CLAUDE-A1",
		"REINSTATE-PHASE1-RC4-MAC-CODEX-A1",
		"claude --resume CLAUDE_SESSION_ID",
		"codex resume CODEX_SESSION_ID",
		"ciphertext",
		"fresh RC4",
		"all 21",
		"test/phase1-rc4-macos-report",
		"test/phase1-rc4-windows-report",
	} {
		if !strings.Contains(body, value) {
			t.Errorf("RC4 acceptance prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"git clone ",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("RC4 acceptance prompts contain forbidden instruction %q", forbidden)
		}
	}
}

func TestRC5AcceptancePromptContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-1-rc5-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.1.0-rc.5",
		"MAC-RC5-M1",
		"WINDOWS-RC5-W1-PASS",
		"local/reinstate-phase1-acceptance-rc5",
		"REINSTATE-PHASE1-RC5-MAC-CLAUDE-A1",
		"REINSTATE-PHASE1-RC5-MAC-CODEX-A1",
		"claude --resume CLAUDE_SESSION_ID",
		"codex resume CODEX_SESSION_ID",
		"f1_default_refusal",
		"f2_missing_manifest_refused",
		"f3_bad_coordinates_refused",
		"remote profile manifest not found",
		"ciphertext",
		"fresh RC5",
		"all 21",
		"test/phase1-rc5-macos-report",
		"test/phase1-rc5-windows-report",
	} {
		if !strings.Contains(body, value) {
			t.Errorf("RC5 acceptance prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"git clone ",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("RC5 acceptance prompts contain forbidden instruction %q", forbidden)
		}
	}
}

func TestRC8AcceptancePromptContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-1-rc8-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.1.0-rc.8",
		"autonomous",
		"R2.txt",
		"REINSTATE_ENCRYPTION_PASSPHRASE",
		"REINSTATE_PASSPHRASE_FD",
		"anonymous pipe",
		"child's environment",
		"MAC-RC8-M1",
		"WINDOWS-RC8-W1-PASS",
		"local/reinstate-phase1-acceptance-rc8",
		"REINSTATE-PHASE1-RC8-MAC-CLAUDE-A1",
		"REINSTATE-PHASE1-RC8-MAC-CODEX-A1",
		"claude --resume CLAUDE_SESSION_ID",
		"codex resume CODEX_SESSION_ID",
		"Prompt version 6",
		"REINSTATE_HOME",
		"would pull",
		"f1_default_refusal",
		"f2_missing_manifest_refused",
		"f3_bad_coordinates_refused",
		"remote profile manifest not found",
		"ciphertext",
		"fresh RC8",
		"all 23",
		"test/phase1-rc8-macos-report",
		"test/phase1-rc8-windows-report",
	} {
		if !strings.Contains(body, value) {
			t.Errorf("RC8 acceptance prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"git clone ",
		"Prompt version 5",
		"I will enter credentials",
		"have me privately",
		"visually confirm",
		"normal R2/S3 UI",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("RC8 acceptance prompts contain forbidden instruction %q", forbidden)
		}
	}
}

func TestPhase2AcceptanceRunbookContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-2-local-index-acceptance.md")
	for _, value := range []string{
		"Current status:",
		"development acceptance passed all 30 required rows",
		"tagged-artifact portions",
		"without:",
		"`rein init`",
		"config.toml",
		"cache/session-index-v1.sqlite",
		"<agent>:<native-session-id>",
		"rein sessions",
		"rein search",
		"rein inspect",
		"rein last",
		"rein resume",
		"rein fork",
		"160 Unicode code points",
		"literal and case-insensitive",
		"Claude Code and Codex",
		"Gemini CLI and OpenCode are",
		"read-only",
		"compatibility exit code `5`",
		"same-vendor",
		"non-TTY",
		"Phase 1 regression",
		"32",
		"PASS",
		"PARTIAL",
		"FAIL",
		"NOT TESTED",
		"parallel",
		"results/phase-2-report-template.md",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("Phase 2 acceptance runbook missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"releases/latest",
		"silently translate",
		"full transcript dump",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("Phase 2 acceptance runbook contains unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase2AutonomousPromptContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-2-agent-verification-prompts.md")
	for _, value := range []string{
		"release-neutral",
		"same exact",
		"independently and run in parallel",
		"Prompt 1 — Claude Code on macOS",
		"Prompt 2 — Codex on native Windows",
		"Never run rein init",
		"Do not create or use R2.txt",
		"cache/session-index-v1.sqlite",
		"rein version --json",
		"rein sessions --json",
		"same-vendor",
		"dry-run JSON",
		"real native PTY",
		"real native console/PTY",
		"PHASE2-DEVICE-REPORT-V1",
		"END-PHASE2-DEVICE-REPORT-V1",
		"PHASE2-FINAL-RECONCILIATION-V1",
		"test/phase2-TEST_ID-macos-report",
		"test/phase2-TEST_ID-windows-report",
		"docs/testing/results/REPORT_DATE-macos-phase2-TEST_ID.md",
		"docs/testing/results/REPORT_DATE-windows-phase2-TEST_ID.md",
		"PASS",
		"PARTIAL",
		"FAIL",
		"NOT TESTED",
		"transcript",
		"credential",
		"passphrase",
		"no product file",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("Phase 2 autonomous prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"git clone ",
		"releases/latest",
		"paste a credential",
		"print the transcript",
		"cat the transcript",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("Phase 2 autonomous prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase2RC1PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.2.0-rc.1-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.2.0-rc.1",
		"Prompt 1 — Claude Code on macOS",
		"Prompt 2 — Codex on native Windows",
		"git verify-tag",
		"five platform archives",
		"five matching",
		"attestation results",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"brand-new isolated INSTALL_DIR",
		"source build is supplemental",
		"test/v0.2.0-rc.1-macos-report",
		"test/v0.2.0-rc.1-windows-report",
		"OpenCode",
		"human-keyboard",
		"Handoff sequence",
		"Promote to stable only if",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.2.0-rc.1 prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"R2.txt",
		"releases/latest",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("v0.2.0-rc.1 prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase2RC2PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.2.0-rc.2-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.2.0-rc.2",
		"Prompt 1 — Claude Code on macOS",
		"Prompt 2 — Codex on native Windows",
		"Prompt 3 — Claude Code on native macOS amd64",
		"Prompt 4 — Codex inside genuine WSL2 amd64",
		"full 40-character TEST_COMMIT",
		"mandatory stop before product-behavior rows",
		"remove any inherited",
		"Makefile owns the per-gate",
		"test/v0.2.0-rc.2-macos-report",
		"test/v0.2.0-rc.2-windows-report",
		"test/v0.2.0-rc.2-macos-amd64-report",
		"test/v0.2.0-rc.2-wsl2-amd64-report",
		"V020RC2-FINAL-RELEASE-RECONCILIATION-V1",
		"raw signer-key fingerprint",
		"corpus/evidence is forbidden",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.2.0-rc.2 prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"R2.txt",
		"releases/latest",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("v0.2.0-rc.2 prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase2RC3PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.2.0-rc.3-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.2.0-rc.3",
		"Prompt 1 — Claude Code on macOS",
		"Prompt 2 — Codex on native Windows",
		"Prompt 3 — Claude Code on native macOS amd64",
		"Prompt 4 — Codex inside genuine WSL2 amd64",
		"full 40-character TEST_COMMIT",
		"mandatory stop before product-behavior rows",
		"remove any inherited",
		"Makefile owns the per-gate",
		"test/v0.2.0-rc.3-macos-report",
		"test/v0.2.0-rc.3-windows-report",
		"test/v0.2.0-rc.3-macos-amd64-report",
		"test/v0.2.0-rc.3-wsl2-amd64-report",
		"V020RC3-FINAL-RELEASE-RECONCILIATION-V1",
		"raw signer-key fingerprint",
		"corpus/evidence is forbidden",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.2.0-rc.3 prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"R2.txt",
		"releases/latest",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("v0.2.0-rc.3 prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase2ReportTemplateContracts(t *testing.T) {
	body := read(t, "docs/testing/results/phase-2-report-template.md")
	for _, value := range []string{
		"cumulative and sanitized",
		"Composite reference",
		"No `rein init` run",
		"No backend/network dependency",
		"32-row table",
		"Release-blocking",
		"Non-blocking",
		"Test-harness deviations",
		"PHASE2-DEVICE-REPORT-V1",
		"END-PHASE2-DEVICE-REPORT-V1",
		"PHASE2-FINAL-RECONCILIATION-V1",
		"END-PHASE2-FINAL-RECONCILIATION-V1",
		"secrets_or_transcripts_committed=false",
		"product_files_changed=0",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("Phase 2 report template missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("Phase 2 report template contains unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase3AcceptanceRunbookContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-3-verified-resume-acceptance.md")
	for _, value := range []string{
		"Required environments for Phase 3 candidates",
		"Apple Silicon macOS",
		"native Windows x64",
		"v0.3.0-rc.1-agent-verification-prompts.md",
		"v0.3.0-rc.2-agent-verification-prompts.md",
		"v0.3.0-rc.3-agent-verification-prompts.md",
		"v0.3.0-rc.4-agent-verification-prompts.md",
		"v0.3.0-rc.5-agent-verification-prompts.md",
		"v0.3.0-rc.6-agent-verification-prompts.md",
		"v0.3.0-rc.7-agent-verification-prompts.md",
		"results/phase-3-report-template.md",
		"Rows 1–32",
		"baseline.unavailable",
		"reinstate_prelaunch_observed",
		"--allow-environment-warning CHECK_ID",
		"full-refresh and large-corpus ceilings",
		"device reports may not relax them",
		"supported mandatory platforms",
		"unsupported/unverified optional evidence",
		"They do not block",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"product_files_changed=0",
		"secrets_or_transcripts_committed=false",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("Phase 3 acceptance runbook missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"releases/latest",
		"silently translate",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("Phase 3 acceptance runbook contains unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase3RC1PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.1-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.1",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"checksummed assets",
		"literal full TEST_COMMIT",
		"mandatory stop before product-behavior rows",
		"source build is supplemental",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"brand-new isolated INSTALL_DIR",
		"test/v0.3.0-rc.1-macos-arm64-report",
		"test/v0.3.0-rc.1-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC1.md",
		"REPORT_DATE-windows-phase3-V030RC1.md",
		"1,000 bounded",
		"256 capability names",
		"maximum `8s`",
		"maximum `12s`",
		"maximum `18s`",
		"greater than 25 percent comparable same-host p95 regression",
		"all five required fuzz-smoke surfaces",
		"staged release assets",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC1-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC1-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
		"Never amend or force-push evidence",
		"unsupported/unverified optional evidence",
		"block RC1 or stable",
		"stable promotion decision",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.1 prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"releases/latest",
		"git clone ",
		"print the transcript",
		"cat the transcript",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("v0.3.0-rc.1 prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase3RC2PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.2-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.2",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"checksummed assets",
		"literal full TEST_COMMIT",
		"mandatory stop before product-behavior rows",
		"source build is supplemental",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"brand-new isolated INSTALL_DIR",
		"test/v0.3.0-rc.2-macos-arm64-report",
		"test/v0.3.0-rc.2-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC2.md",
		"REPORT_DATE-windows-phase3-V030RC2.md",
		"windows-acceptance-host.md",
		"PowerShell-native",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC2-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC2-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
		"Never amend or force-push evidence",
		"unsupported/unverified optional evidence",
		"stable promotion decision",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.2 prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"releases/latest",
		"git clone ",
		"print the transcript",
		"cat the transcript",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("v0.3.0-rc.2 prompts contain unsafe instruction %q", forbidden)
		}
	}
}

func TestPhase3RC3PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.3-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.3",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"literal full TEST_COMMIT",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"test/v0.3.0-rc.3-macos-arm64-report",
		"test/v0.3.0-rc.3-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC3.md",
		"REPORT_DATE-windows-phase3-V030RC3.md",
		"windows-acceptance-host.md",
		"snapshot.ps1",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC3-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC3-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.3 prompts missing %q", value)
		}
	}
}

func TestPhase3RC4PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.4-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.4",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"literal full TEST_COMMIT",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"test/v0.3.0-rc.4-macos-arm64-report",
		"test/v0.3.0-rc.4-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC4.md",
		"REPORT_DATE-windows-phase3-V030RC4.md",
		"windows-acceptance-host.md",
		"snapshot.ps1",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC4-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC4-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.4 prompts missing %q", value)
		}
	}
}

func TestPhase3RC5PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.5-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.5",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"literal full TEST_COMMIT",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"test/v0.3.0-rc.5-macos-arm64-report",
		"test/v0.3.0-rc.5-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC5.md",
		"REPORT_DATE-windows-phase3-V030RC5.md",
		"windows-acceptance-host.md",
		"snapshot.ps1",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC5-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC5-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
		"RC4 draft remained unpublished and unattested",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.5 prompts missing %q", value)
		}
	}
}

func TestPhase3RC6PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.6-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.6",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"literal full TEST_COMMIT",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"test/v0.3.0-rc.6-macos-arm64-report",
		"test/v0.3.0-rc.6-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC6.md",
		"REPORT_DATE-windows-phase3-V030RC6.md",
		"windows-acceptance-host.md",
		"snapshot.ps1",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC6-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC6-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
		"2.1.227",
		"0.147.0",
		"outside the fail-closed ranges",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.6 prompts missing %q", value)
		}
	}
}

func TestPhase3RC7PromptContracts(t *testing.T) {
	body := read(t, "docs/testing/v0.3.0-rc.7-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.3.0-rc.7",
		"Prompt 1 — Claude Code on Apple Silicon macOS",
		"Prompt 2 — Codex on native Windows x64",
		"exact 25-asset set",
		"checksums.txt plus 24",
		"literal full TEST_COMMIT",
		"https://reinstate.dev/install.sh",
		"https://reinstate.dev/install.ps1",
		"test/v0.3.0-rc.7-macos-arm64-report",
		"test/v0.3.0-rc.7-windows-amd64-report",
		"REPORT_DATE-macos-phase3-V030RC7.md",
		"REPORT_DATE-windows-phase3-V030RC7.md",
		"windows-acceptance-host.md",
		"snapshot.ps1",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC7-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC7-FINAL-RECONCILIATION-V1",
		"stable_v0.3.0_authorized=false",
		"2.1.227",
		"0.147.0",
		"non-TTY native launch fail-closed",
		"capability probe diagnostics",
		"CLAUDE_CONFIG_DIR",
		"CODEX_HOME",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("v0.3.0-rc.7 prompts missing %q", value)
		}
	}
}

func TestPhase3ReportTemplateContracts(t *testing.T) {
	body := read(t, "docs/testing/results/phase-3-report-template.md")
	for _, value := range []string{
		"cumulative",
		"sanitized",
		"32 NOT TESTED",
		"Exact 25-asset release set",
		"literal full 40-character tested commit",
		"Required 32-row matrix",
		"Normal cold full refresh",
		"Large cold full refresh",
		"macOS max `8s`; Windows max `12s`",
		"macOS max `12s`; Windows max `18s`",
		"Release-blocking",
		"Non-blocking",
		"Test-harness deviations",
		"PHASE3-DEVICE-REPORT-V1",
		"END-PHASE3-DEVICE-REPORT-V1",
		"PHASE3-RC1-FINAL-RECONCILIATION-V1",
		"END-PHASE3-RC1-FINAL-RECONCILIATION-V1",
		"required counts must sum to 32",
		"product_files_changed=0",
		"secrets_or_transcripts_committed=false",
		"stable_v0.3.0_authorized=false",
		"supported mandatory",
		"unsupported/unverified optional evidence",
		"not block RC1 or stable",
		"separate stable",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("Phase 3 report template missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("Phase 3 report template contains unsafe instruction %q", forbidden)
		}
	}
}

func TestReleaseRunbookStagesAndFreezesReleaseInputs(t *testing.T) {
	body := read(t, "RELEASING.md")
	for _, value := range []string{
		"release commit itself must contain both public bootstrap files",
		"post-tag pin-only edit cannot repair it",
		"v0.3.0-rc.1-agent-verification-prompts.md",
		"v0.3.0-rc.2-agent-verification-prompts.md",
		"v0.3.0-rc.3-agent-verification-prompts.md",
		"v0.3.0-rc.4-agent-verification-prompts.md",
		"v0.3.0-rc.5-agent-verification-prompts.md",
		"v0.3.0-rc.6-agent-verification-prompts.md",
		"v0.3.0-rc.7-agent-verification-prompts.md",
		"windows-acceptance-host.md",
		"two tagged-artifact reports",
		"supported mandatory platforms",
		"unsupported/unverified optional evidence",
		"separate reviewed stable",
		"GOTOOLCHAIN=go1.25.12 go mod tidy -diff",
		"./scripts/stage-release-assets.sh dist",
		"./scripts/check-release-artifacts.sh dist",
		"sh scripts/test-install.sh dist",
		"git diff --exit-code -- go.mod go.sum",
		"test -z \"$(git status --porcelain)\"",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(value)) {
			t.Errorf("release runbook missing %q", value)
		}
	}

	snapshot := strings.Index(body, "make snapshot")
	stage := strings.Index(body, "./scripts/stage-release-assets.sh dist")
	artifactCheck := strings.Index(body, "./scripts/check-release-artifacts.sh dist")
	installerCheck := strings.Index(body, "sh scripts/test-install.sh dist")
	tidyDiff := strings.Index(body, "git diff --exit-code -- go.mod go.sum")
	clean := strings.Index(body, `test -z "$(git status --porcelain)"`)
	if snapshot < 0 || stage < snapshot || artifactCheck < stage || installerCheck < artifactCheck || tidyDiff < installerCheck || clean < tidyDiff {
		t.Fatal("release runbook must snapshot, stage, inspect, test installers, then prove tidy and clean in that order")
	}
	for _, forbidden := range []string{
		"committed four-environment acceptance dispatch",
		"For a stable release, macOS arm64, macOS amd64",
		"normal four-environment",
		"evidence required before stable `v0.3.0`",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("release runbook still contains obsolete platform policy %q", forbidden)
		}
	}
}

func TestPrivateAcceptanceInputIsIgnored(t *testing.T) {
	body := read(t, ".gitignore")
	if !strings.Contains(body, "[Rr]2.txt") {
		t.Fatal(".gitignore must exclude the private acceptance input file")
	}
}

func TestContributorPromptContract(t *testing.T) {
	body := read(t, "docs/prompts/contributor-setup.md")
	for _, value := range []string{
		"github.com/HarjjotSinghh/reinstate",
		"make verify",
		"Never read real Claude/Codex session",
		"Do not push, tag, or publish",
	} {
		if !strings.Contains(body, value) {
			t.Errorf("contributor prompt missing %q", value)
		}
	}
}
