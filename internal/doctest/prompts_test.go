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
			"Prompt version:** 6",
			"v0.1.0",
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
		"Physical macOS and native-Windows acceptance remains pending",
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
