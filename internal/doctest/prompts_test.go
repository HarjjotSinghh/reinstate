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
			"v0.1.0-rc.3",
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
