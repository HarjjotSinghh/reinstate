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
			"v0.1.0-rc.6",
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

func TestRC6AcceptancePromptContracts(t *testing.T) {
	body := read(t, "docs/testing/phase-1-rc6-agent-verification-prompts.md")
	for _, value := range []string{
		"v0.1.0-rc.6",
		"MAC-RC6-M1",
		"WINDOWS-RC6-W1-PASS",
		"local/reinstate-phase1-acceptance-rc6",
		"REINSTATE-PHASE1-RC6-MAC-CLAUDE-A1",
		"REINSTATE-PHASE1-RC6-MAC-CODEX-A1",
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
		"fresh RC6",
		"all 21",
		"test/phase1-rc6-macos-report",
		"test/phase1-rc6-windows-report",
	} {
		if !strings.Contains(body, value) {
			t.Errorf("RC6 acceptance prompts missing %q", value)
		}
	}
	for _, forbidden := range []string{
		"REINSTATE_PASSPHRASE=",
		"--dangerously-skip-permissions",
		"git clone ",
		"Prompt version 5",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("RC6 acceptance prompts contain forbidden instruction %q", forbidden)
		}
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
