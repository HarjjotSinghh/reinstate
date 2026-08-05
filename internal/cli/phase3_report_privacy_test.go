package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/runtimecheck"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func TestEnvironmentHumanRendererUsesPrivacySafeFieldAllowlist(t *testing.T) {
	const hidden = "PHASE3-HUMAN-HIDDEN-SENTINEL"
	report := preflight.Report{
		Decision: preflight.DecisionConfirmationRequired,
		Checks: []preflight.Check{{
			ID: "git.branch", Status: preflight.StatusChanged,
			Severity: preflight.SeverityWarning,
			Expected: "main", Actual: "release-candidate",
			Provenance: workspace.ProvenanceVendorRecorded,
			Message:    "the current branch differs from the recorded branch",
			Repair:     "switch to the expected branch or review this exact warning",
		}},
		Workspace: workspace.Fingerprint{
			Workspace: workspace.WorkspaceFingerprint{Path: hidden},
			Git:       workspace.GitFingerprint{Root: hidden, RepositoryID: hidden},
		},
		Agent: agentcheck.Result{Version: hidden},
		Capabilities: capability.Inventory{Items: []capability.Item{{
			Agent: capability.AgentClaude, Kind: capability.KindSkill, Name: hidden,
		}}},
		Runtimes: []runtimecheck.Result{{Name: "node", Declared: hidden, Actual: hidden}},
	}

	var output bytes.Buffer
	writeEnvironmentReportHuman(&output, report)
	rendered := output.String()
	if strings.Contains(rendered, hidden) {
		t.Fatalf("human renderer widened its field allowlist: %s", rendered)
	}
	for _, required := range []string{
		"Environment decision: confirmation_required",
		"Environment check: git.branch",
		"status=changed",
		"severity=warning",
		"provenance=vendor_recorded",
		"the current branch differs from the recorded branch",
		"Expected: \"main\"",
		"Actual: \"release-candidate\"",
		"Repair: switch to the expected branch or review this exact warning",
	} {
		if !strings.Contains(rendered, required) {
			t.Fatalf("human environment report omitted %q: %s", required, rendered)
		}
	}
}
