// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package cli

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/handoffui"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// runHandoffStudio opens the interactive handoff studio for one source session
// and then performs whatever the user chose.
//
// The studio's previews are dry-run plans: handoff.Plan writes nothing outside
// a temporary directory, so cycling through policies to compare them has no
// durable effect. Only the final confirmation runs the real command.
func runHandoffStudio(
	cmd *cobra.Command,
	options localCommandOptions,
	capability ui.Capability,
	reference string,
	sourceAgent string,
) error {
	destinations := withoutSourceAgent(catalogKeysAtLeast(agents.TierHandoffTo), sourceAgent)
	if len(destinations) == 0 {
		return NewExitError(
			ExitCompatibility,
			"no other agent supports structured handoff from "+sourceAgent,
		)
	}

	planner := handoffui.NewPlanner(studioPlanFunc(cmd, options, reference))
	studio := handoffui.New(handoffui.Options{
		Theme:        ui.NewTheme(capability),
		Capability:   capability,
		Planner:      planner,
		Clipboard:    tui.OSC52Clipboard(cmd.OutOrStdout()),
		Context:      cmd.Context(),
		Reference:    reference,
		SourceAgent:  sourceAgent,
		Destinations: destinations,
		Policy:       string(handoff.PolicyBalanced),
	})

	intent, err := tui.Run(cmd.Context(), studio, tui.RunOptions{
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		Capability: capability,
		AltScreen:  true,
	})
	if err != nil {
		return localRuntimeError("run the handoff studio", err)
	}
	if !intent.Chosen() {
		return nil
	}
	PrintHuman(cmd.ErrOrStderr(), "%s.", handoffHumanPrefix(intent.Destination))
	return runHandoffWith(cmd, handoffAliasOptions{
		Session:         intent.Reference,
		Agent:           intent.Destination,
		Policy:          intent.Policy,
		NoLaunch:        studio.ExportRequested(),
		AllowedWarnings: intent.AcknowledgedWarnings,
	})
}

// studioPlanFunc builds dry-run plans for the studio's live preview.
//
// Each plan runs against a throwaway index home so a preview can never mutate
// durable state, matching what `rein handoff --dry-run` does. The temporary
// directories are removed as soon as the plan has been measured.
func studioPlanFunc(
	cmd *cobra.Command,
	options localCommandOptions,
	reference string,
) handoffui.PlanFunc {
	return func(ctx context.Context, destination, policy string) (handoff.PlanResult, error) {
		home, err := config.Home()
		if err != nil {
			return handoff.PlanResult{}, err
		}
		indexHome, err := os.MkdirTemp("", "reinstate-handoff-preview-*")
		if err != nil {
			return handoff.PlanResult{}, err
		}
		defer func() { _ = os.RemoveAll(indexHome) }()

		index, err := openHandoffIndex(options, indexHome)
		if err != nil {
			return handoff.PlanResult{}, err
		}
		defer func() { _ = index.Close() }()

		// The preview index is a throwaway, so it starts empty. Resolving
		// without refreshing first would report every session as missing.
		record, _, _, err := index.RefreshAndResolve(ctx, reference)
		if err != nil {
			return handoff.PlanResult{}, err
		}
		pipelineOptions := handoff.Options{
			ToAgent:       strings.ToLower(strings.TrimSpace(destination)),
			Policy:        handoff.Policy(strings.ToLower(strings.TrimSpace(policy))),
			Verifier:      options.verifier,
			ResolveSource: handoffResolver(index),
			ReinstateHome: home,
			Capability:    handoffCapabilityOptions(options.verifier),
			SessionExists: handoffClaudeSessionExists(index),
		}
		if options.processChecker != nil {
			pipelineOptions.SessionBusy = handoff.SessionBusyFunc(options.processChecker)
		}
		if working, wdErr := os.Getwd(); wdErr == nil {
			pipelineOptions.WorkingDir = working
		}
		plan, err := handoff.Plan(ctx, record, pipelineOptions)
		if plan.TempDir != "" {
			_ = os.RemoveAll(plan.TempDir)
		}
		return plan, err
	}
}

// withoutSourceAgent removes the source agent from the destination list.
//
// A handoff to the agent the session already belongs to is refused by the
// pipeline as a usage error — "source and destination agents must differ"
// (internal/handoff/pipeline.go). Offering it as a selectable destination would
// let the user commit to a choice that can only fail, so it is not offered.
func withoutSourceAgent(destinations []string, sourceAgent string) []string {
	if sourceAgent == "" {
		return destinations
	}
	ordered := make([]string, 0, len(destinations))
	for _, destination := range destinations {
		if destination != sourceAgent {
			ordered = append(ordered, destination)
		}
	}
	return ordered
}

// handoffSourceAgent extracts the agent from a canonical reference for display.
func handoffSourceAgent(reference string) string {
	agent, _, ok := sessionindex.ParseCompositeReference(reference)
	if !ok {
		return ""
	}
	return agent
}
