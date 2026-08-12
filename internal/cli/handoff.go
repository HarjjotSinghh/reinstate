package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/doctor"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const handoffMode = "structured handoff"

type handoffCommandOptions struct {
	local          localCommandOptions
	processChecker AgentProcessChecker
}

type handoffPlanOutput struct {
	Mode                   string                 `json:"mode"`
	DestinationSessionMode string                 `json:"destination_session_mode"`
	HandoffID              string                 `json:"handoff_id"`
	LineageRoot            string                 `json:"lineage_root"`
	Source                 handoffEndpointOutput  `json:"source"`
	Destination            handoffCommandOutput   `json:"destination"`
	Policy                 string                 `json:"policy"`
	Workspace              capsule.Workspace      `json:"workspace"`
	Capabilities           capsule.CapabilityDiff `json:"capabilities"`
	Fidelity               capsule.Fidelity       `json:"fidelity"`
	Parse                  any                    `json:"parse"`
	PlannedFiles           []string               `json:"planned_files"`
	EstimatedBytes         int64                  `json:"estimated_bytes"`
	EstimatedTokens        int                    `json:"estimated_tokens"`
	WarningIDs             []string               `json:"warning_ids"`
	Redactions             map[string]int         `json:"redactions"`
	SourceMayHaveAdvanced  bool                   `json:"source_may_have_advanced"`
}

type handoffEndpointOutput struct {
	Agent     string `json:"agent"`
	SessionID string `json:"session_id"`
}

type handoffCommandOutput struct {
	Agent      string   `json:"agent"`
	SessionID  string   `json:"session_id,omitempty"`
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	CWD        string   `json:"cwd"`
}

type handoffListOutput struct {
	Mode     string                 `json:"mode"`
	Handoffs []handoff.LineageEntry `json:"handoffs"`
}

type handoffInspectOutput struct {
	Mode            string                  `json:"mode"`
	Capsule         capsule.Capsule         `json:"capsule"`
	Lineage         *handoff.LineageEntry   `json:"lineage,omitempty"`
	Acknowledgement handoff.Acknowledgement `json:"acknowledgement"`
	Artifacts       handoffArtifactOutput   `json:"artifacts"`
}

type handoffArtifactOutput struct {
	ProjectionBytes  int    `json:"projection_bytes"`
	ProjectionSHA256 string `json:"projection_sha256"`
	BootstrapBytes   int    `json:"bootstrap_bytes"`
	BootstrapSHA256  string `json:"bootstrap_sha256"`
	SidecarBytes     int    `json:"sidecar_bytes"`
}

func newHandoffCmd(options handoffCommandOptions) *cobra.Command {
	var (
		last            bool
		from            string
		to              string
		policy          string
		dryRun          bool
		asJSON          bool
		noLaunch        bool
		exportPath      string
		allowedWarnings []string
		allowActive     bool
		allowUntested   bool
		showRedactions  bool
	)
	cmd := &cobra.Command{
		Use:   "handoff [SESSION]",
		Short: "Continue the same task in a new destination-agent session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateHandoffSelection(args, last, from, to, dryRun, noLaunch, asJSON, exportPath); err != nil {
				return err
			}
			index, err := openLocalIndex(options.local)
			if err != nil {
				return err
			}
			defer func() { _ = index.Close() }()

			record, err := selectHandoffSource(cmd.Context(), index, args, last, from)
			if err != nil {
				return handoffResolveError(err)
			}
			home, err := config.Home()
			if err != nil {
				return NewExitError(ExitConfig, err.Error())
			}
			pipelineOptions := handoff.Options{
				ToAgent:       strings.ToLower(strings.TrimSpace(to)),
				Policy:        handoff.Policy(strings.ToLower(strings.TrimSpace(policy))),
				Verifier:      options.local.verifier,
				ResolveSource: handoffResolver(index),
				SessionBusy:   handoff.SessionBusyFunc(options.processChecker),
				LaunchRunner:  handoffLaunchRunner(cmd, options.local),
				ReinstateHome: home,
				AllowActive:   allowActive,
				AllowUntested: allowUntested,
				AllowWarnings: append([]string(nil), allowedWarnings...),
				Capability:    handoffCapabilityOptions(options.local.verifier),
				SessionExists: handoffClaudeSessionExists(index),
			}

			var plan handoff.PlanResult
			if dryRun {
				plan, err = handoff.Plan(cmd.Context(), record, pipelineOptions)
				if plan.TempDir != "" {
					defer func() { _ = os.RemoveAll(plan.TempDir) }()
				}
			} else {
				var result handoff.ExecuteResult
				result, err = handoff.Execute(cmd.Context(), record, pipelineOptions, !noLaunch)
				plan = result.Plan
			}
			if plan.HandoffID != "" {
				if writeErr := writeHandoffPlan(cmd, plan, asJSON, showRedactions); writeErr != nil && err == nil {
					err = writeErr
				}
				if exportPath != "" && err == nil {
					if writeErr := exportHandoffProjection(exportPath, plan.Artifacts.ProjectionMD); writeErr != nil {
						err = writeErr
					}
				}
			}
			if err != nil {
				return handoffCLIError(err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&last, "last", false, "resolve the newest matching source session")
	cmd.Flags().StringVar(&from, "from", "", "restrict --last to one source agent")
	cmd.Flags().StringVar(&to, "to", "", "destination agent: claude|codex")
	cmd.Flags().StringVar(&policy, "policy", string(handoff.PolicyBalanced), "projection policy: checkpoint|balanced|full")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without durable writes or launch")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&noLaunch, "no-launch", false, "build the capsule and print the command without spawning")
	cmd.Flags().StringVar(&exportPath, "export", "", "additionally write the projection to PATH")
	cmd.Flags().StringArrayVar(&allowedWarnings, "allow-warning", nil, "acknowledge one exact warning ID (repeatable)")
	cmd.Flags().BoolVar(&allowActive, "allow-active", false, "freeze the last complete record while the source is active")
	cmd.Flags().BoolVar(&allowUntested, "allow-untested", false, "proceed with an untested source or destination layout")
	cmd.Flags().BoolVar(&showRedactions, "show-redactions", false, "show redaction categories and counts, never values")
	cmd.AddCommand(newHandoffListCmd(), newHandoffInspectCmd(), newHandoffExportCmd())
	return cmd
}

func validateHandoffSelection(args []string, last bool, from, to string, dryRun, noLaunch, asJSON bool, exportPath string) error {
	if strings.TrimSpace(to) == "" {
		return NewExitError(ExitUsage, "--to AGENT is required")
	}
	if err := validateNativeAgent(to, false); err != nil {
		return err
	}
	if (len(args) == 0) == !last {
		return NewExitError(ExitUsage, "provide exactly one SESSION or --last")
	}
	if len(args) > 0 && strings.TrimSpace(from) != "" {
		return NewExitError(ExitUsage, "--from requires --last")
	}
	if strings.TrimSpace(from) != "" {
		if err := validateLocalAgent(from, false); err != nil {
			return err
		}
	}
	if dryRun && noLaunch {
		return NewExitError(ExitUsage, "--dry-run and --no-launch are mutually exclusive")
	}
	if asJSON && !dryRun && !noLaunch {
		return NewExitError(ExitUsage, "--json requires --dry-run or --no-launch for structured handoff launches")
	}
	if dryRun && strings.TrimSpace(exportPath) != "" {
		return NewExitError(ExitUsage, "--export cannot be used with --dry-run because dry-run writes only to a temporary directory")
	}
	return nil
}

func selectHandoffSource(ctx context.Context, index *sessionindex.Index, args []string, last bool, from string) (sessionindex.Record, error) {
	if !last {
		record, _, _, err := index.RefreshAndResolve(ctx, args[0])
		return record, err
	}
	filter := sessionindex.Filter{Agent: strings.ToLower(strings.TrimSpace(from)), Limit: 1}
	if filter.Agent == "" {
		filter.Agent = "all"
	}
	if _, err := index.Refresh(ctx); err != nil {
		return sessionindex.Record{}, err
	}
	return index.Last(ctx, filter)
}

func handoffResolver(index *sessionindex.Index) handoff.ResolveSourceFunc {
	return func(ctx context.Context, record sessionindex.Record) (sessionindex.Record, bool, error) {
		resolved, _, fresh, err := index.RefreshAndResolve(ctx, record.Reference())
		return resolved, fresh, err
	}
}

func handoffClaudeSessionExists(index *sessionindex.Index) handoff.ClaudeSessionExists {
	return func(ctx context.Context, sessionID string) (bool, error) {
		if _, err := index.RefreshAgent(ctx, sessionindex.AgentClaude); err != nil {
			return false, err
		}
		_, err := index.Resolve(ctx, sessionindex.CompositeReference(sessionindex.AgentClaude, sessionID))
		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, sessionindex.ErrNotFound):
			return false, nil
		default:
			return false, err
		}
	}
}

func handoffCapabilityOptions(verifier preflight.Verifier) capability.Options {
	switch typed := verifier.(type) {
	case preflight.Service:
		return typed.Options.Capability
	case *preflight.Service:
		if typed != nil {
			return typed.Options.Capability
		}
	}
	return capability.Options{}
}

func handoffLaunchRunner(cmd *cobra.Command, options localCommandOptions) sessionindex.LaunchRunner {
	if options.launchRunner != nil {
		return options.launchRunner
	}
	return sessionindex.ExecLaunchRunner{
		Stdin:  cmd.InOrStdin(),
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	}
}

func handoffResolveError(err error) error {
	switch {
	case errors.Is(err, sessionindex.ErrAmbiguous):
		return NewExitError(ExitConflict, err.Error())
	case errors.Is(err, sessionindex.ErrNotFound):
		return NewExitError(ExitUsage, err.Error())
	default:
		return localRuntimeError("resolve handoff source", err)
	}
}

func handoffCLIError(err error) error {
	var pipelineError *handoff.PipelineError
	if errors.As(err, &pipelineError) {
		return NewExitError(pipelineError.Code, pipelineError.Error())
	}
	var exitError *ExitError
	if errors.As(err, &exitError) {
		return exitError
	}
	return NewExitError(ExitRuntime, err.Error())
}

func writeHandoffPlan(cmd *cobra.Command, plan handoff.PlanResult, asJSON, showRedactions bool) error {
	output := handoffPlanOutput{
		Mode:                   handoffMode,
		DestinationSessionMode: "new",
		HandoffID:              plan.HandoffID,
		LineageRoot:            plan.LineageRoot,
		Source: handoffEndpointOutput{
			Agent: plan.Capsule.RawSource.Agent, SessionID: plan.Capsule.RawSource.SessionID,
		},
		Destination: handoffCommandOutput{
			Agent: plan.Destination.Agent, SessionID: plan.Destination.SessionID,
			Executable: plan.Destination.Executable, Args: append([]string(nil), plan.Destination.Args...), CWD: plan.Destination.Dir,
		},
		Policy: plan.Capsule.Projection.Policy, Workspace: plan.Capsule.Workspace,
		Capabilities: plan.Capsule.Capabilities, Fidelity: plan.Capsule.Fidelity,
		Parse: plan.Parse, PlannedFiles: append([]string(nil), plan.PlannedFiles...),
		EstimatedBytes: plan.EstimatedBytes, EstimatedTokens: plan.EstimatedTokens,
		WarningIDs: append([]string(nil), plan.WarningIDs...), Redactions: plan.RedactionCounts,
		SourceMayHaveAdvanced: plan.SourceMayHaveAdvanced,
	}
	if output.PlannedFiles == nil {
		output.PlannedFiles = []string{}
	}
	if output.WarningIDs == nil {
		output.WarningIDs = []string{}
	}
	if output.Redactions == nil {
		output.Redactions = map[string]int{}
	}
	if asJSON {
		return WriteJSON(cmd.OutOrStdout(), output)
	}
	prefix := handoffHumanPrefix(plan.Destination.Agent)
	PrintHuman(cmd.OutOrStdout(), "%s: handoff %s from %s:%s", prefix, plan.HandoffID, plan.Capsule.RawSource.Agent, plan.Capsule.RawSource.SessionID)
	PrintHuman(cmd.OutOrStdout(), "%s: policy=%s projection=%d bytes estimated_tokens=%d", prefix, plan.Capsule.Projection.Policy, plan.EstimatedBytes, plan.EstimatedTokens)
	PrintHuman(cmd.OutOrStdout(), "%s: command %s", prefix, quoteCommand(plan.Destination.Executable, plan.Destination.Args))
	for _, path := range plan.PlannedFiles {
		PrintHuman(cmd.OutOrStdout(), "%s: file %s", prefix, doctor.RedactPath(path))
	}
	for _, warningID := range plan.WarningIDs {
		PrintHuman(cmd.OutOrStdout(), "%s: warning %s", prefix, warningID)
	}
	if showRedactions {
		categories := make([]string, 0, len(plan.RedactionCounts))
		for category := range plan.RedactionCounts {
			categories = append(categories, category)
		}
		sort.Strings(categories)
		for _, category := range categories {
			PrintHuman(cmd.OutOrStdout(), "%s: redaction %s=%d", prefix, category, plan.RedactionCounts[category])
		}
	}
	return nil
}

func handoffHumanPrefix(agent string) string {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		agent = "destination-agent"
	}
	return fmt.Sprintf("Structured handoff — a new %s session, not native resume", agent)
}

func quoteCommand(executable string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, strconv.Quote(executable))
	for _, arg := range args {
		parts = append(parts, strconv.Quote(arg))
	}
	return strings.Join(parts, " ")
}

func exportHandoffProjection(path string, body []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := fsx.WriteFileAtomic(path, body, fsx.OwnerOnlyFilePerm); err != nil {
		return NewExitError(ExitRuntime, "write handoff projection export: "+err.Error())
	}
	return nil
}

func newHandoffListCmd() *cobra.Command {
	var asJSON bool
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List structured handoff lineage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := openHandoffStore()
			if err != nil {
				return err
			}
			entries, err := store.List(limit)
			if err != nil {
				return handoffCLIError(err)
			}
			if entries == nil {
				entries = []handoff.LineageEntry{}
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), handoffListOutput{Mode: handoffMode, Handoffs: entries})
			}
			if len(entries) == 0 {
				PrintHuman(cmd.OutOrStdout(), "%s: no handoffs", handoffHumanPrefix("destination-agent"))
				return nil
			}
			for _, entry := range entries {
				PrintHuman(cmd.OutOrStdout(), "%s: %s %s:%s -> %s:%s state=%s created=%s", handoffHumanPrefix(entry.Destination.Agent), entry.HandoffID, entry.Source.Agent, entry.Source.SessionID, entry.Destination.Agent, entry.Destination.SessionID, entry.Destination.State, entry.CreatedAt.UTC().Format(time.RFC3339))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().IntVar(&limit, "limit", 100, "maximum handoffs to return")
	return cmd
}

func newHandoffInspectCmd() *cobra.Command {
	var asJSON bool
	var acknowledged bool
	var notAcknowledged bool
	cmd := &cobra.Command{
		Use:   "inspect HANDOFF_ID",
		Short: "Inspect a stored structured handoff",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if acknowledged && notAcknowledged {
				return NewExitError(ExitUsage, "--acknowledged and --not-acknowledged are mutually exclusive")
			}
			store, err := openHandoffStore()
			if err != nil {
				return err
			}
			if acknowledged || notAcknowledged {
				if err := handoff.RecordAcknowledgement(store, args[0], acknowledged); err != nil {
					return handoffStoreError(err)
				}
			}
			c, artifacts, err := store.Get(args[0])
			if err != nil {
				return handoffStoreError(err)
			}
			ack, err := handoff.GetAcknowledgement(store, args[0])
			if err != nil {
				return handoffStoreError(err)
			}
			lineage, err := handoffLineageEntry(store, args[0])
			if err != nil {
				return handoffCLIError(err)
			}
			output := handoffInspectOutput{
				Mode: handoffMode, Capsule: c, Lineage: lineage, Acknowledgement: ack,
				Artifacts: handoffArtifactOutput{
					ProjectionBytes: len(artifacts.ProjectionMD), ProjectionSHA256: bytesSHA256(artifacts.ProjectionMD),
					BootstrapBytes: len(artifacts.Bootstrap), BootstrapSHA256: bytesSHA256(artifacts.Bootstrap),
					SidecarBytes: len(artifacts.SidecarEvents),
				},
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), output)
			}
			agent := capsuleDestinationAgent(c)
			if lineage != nil && lineage.Destination.Agent != "" {
				agent = lineage.Destination.Agent
			}
			prefix := handoffHumanPrefix(agent)
			PrintHuman(cmd.OutOrStdout(), "%s: handoff=%s source=%s:%s policy=%s fidelity=%s", prefix, c.Identity.ID, c.RawSource.Agent, c.RawSource.SessionID, c.Projection.Policy, c.Fidelity.Overall)
			PrintHuman(cmd.OutOrStdout(), "%s: acknowledgement=%s", prefix, acknowledgementText(ack.Confirmed))
			PrintHuman(cmd.OutOrStdout(), "%s: projection=%d bytes bootstrap=%d bytes", prefix, len(artifacts.ProjectionMD), len(artifacts.Bootstrap))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&acknowledged, "acknowledged", false, "record that the destination acknowledgement was confirmed")
	cmd.Flags().BoolVar(&notAcknowledged, "not-acknowledged", false, "record that the destination acknowledgement was not confirmed")
	return cmd
}

func newHandoffExportCmd() *cobra.Command {
	var format string
	var out string
	cmd := &cobra.Command{
		Use:   "export HANDOFF_ID",
		Short: "Export a stored capsule or projection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format = strings.ToLower(strings.TrimSpace(format))
			if format != "json" && format != "markdown" {
				return NewExitError(ExitUsage, "--format must be json or markdown")
			}
			store, err := openHandoffStore()
			if err != nil {
				return err
			}
			c, artifacts, err := store.Get(args[0])
			if err != nil {
				return handoffStoreError(err)
			}
			body := artifacts.ProjectionMD
			if format == "json" {
				body, err = capsule.CanonicalBytes(c)
				if err != nil {
					return handoffCLIError(err)
				}
			}
			if strings.TrimSpace(out) == "" {
				_, err = cmd.OutOrStdout().Write(append(body, '\n'))
				return handoffCLIError(err)
			}
			if err := fsx.WriteFileAtomic(out, body, fsx.OwnerOnlyFilePerm); err != nil {
				return handoffCLIError(err)
			}
			PrintHuman(cmd.OutOrStdout(), "%s: exported %s to %s", handoffHumanPrefix(capsuleDestinationAgent(c)), format, doctor.RedactPath(filepath.Clean(out)))
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "export format: json|markdown")
	cmd.Flags().StringVar(&out, "out", "", "write to PATH instead of stdout")
	return cmd
}

func openHandoffStore() (*handoff.Store, error) {
	home, err := config.Home()
	if err != nil {
		return nil, NewExitError(ExitConfig, err.Error())
	}
	store, err := handoff.OpenStore(home)
	if err != nil {
		return nil, NewExitError(ExitConfig, err.Error())
	}
	return store, nil
}

func handoffStoreError(err error) error {
	if errors.Is(err, handoff.ErrNotFound) {
		return NewExitError(ExitUsage, err.Error())
	}
	return handoffCLIError(err)
}

func handoffLineageEntry(store *handoff.Store, id string) (*handoff.LineageEntry, error) {
	entries, err := store.List(1000)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.HandoffID == id {
			copy := entry
			return &copy, nil
		}
	}
	return nil, nil
}

func capsuleDestinationAgent(c capsule.Capsule) string {
	if agent, ok := c.Capabilities.Destination["agent"].(string); ok {
		return agent
	}
	return "destination-agent"
}

func acknowledgementText(confirmed *bool) string {
	if confirmed == nil {
		return "not recorded"
	}
	if *confirmed {
		return "confirmed"
	}
	return "not confirmed"
}

func bytesSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
