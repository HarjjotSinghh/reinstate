package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/probe"
	"github.com/HarjjotSinghh/reinstate/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	var asJSON bool
	var selfTest bool
	var agentsMode bool
	var acceptanceMatrix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run redacted diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if acceptanceMatrix {
				return runDoctorAcceptanceMatrix(cmd, asJSON)
			}
			if agentsMode && asJSON {
				return runDoctorAgentsJSON(cmd)
			}
			if agentsMode {
				return runDoctorAgentsHuman(cmd)
			}
			home := os.Getenv("REINSTATE_HOME")
			rep, err := doctor.Run(cmd.Context(), doctor.Options{
				Home:     home,
				SelfTest: selfTest,
			})
			if err != nil {
				if ee, ok := err.(*ExitError); ok {
					return ee
				}
				return err
			}
			code := doctor.ExitCode(rep)
			if asJSON {
				if err := WriteJSON(cmd.OutOrStdout(), rep); err != nil {
					return err
				}
			} else {
				PrintHuman(cmd.OutOrStdout(), "%s", doctor.FormatHuman(rep))
			}
			if code != ExitOK {
				return NewExitError(code, rep.Summary)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&selfTest, "self-test", false, "run synthetic encryption/storage self-test")
	cmd.Flags().BoolVar(&agentsMode, "agents", false, "list catalog agents, tiers, and local inventory")
	cmd.Flags().BoolVar(&acceptanceMatrix, "acceptance-matrix", false, "emit the generated Phase 5 acceptance row count")
	return cmd
}

func runDoctorAgentsJSON(cmd *cobra.Command) error {
	art, err := collectAgentProbe(cmd.Context())
	if err != nil {
		return err
	}
	if err := probe.Validate(art); err != nil {
		return err
	}
	return WriteJSON(cmd.OutOrStdout(), art)
}

func runDoctorAgentsHuman(cmd *cobra.Command) error {
	art, err := collectAgentProbe(cmd.Context())
	if err != nil {
		return err
	}
	counts := sessionCounts(cmd.Context())
	PrintHuman(cmd.OutOrStdout(), "key\ttier\tinstalled\troot\tsessions\tnotes")
	for _, rec := range art.Agents {
		installed := "no"
		if rec.ExecutableOnPath {
			installed = "yes"
		}
		root := "no"
		if rec.ResolvedRoot != nil {
			root = "yes"
		}
		sessions := "-"
		if n, ok := counts[rec.Key]; ok {
			sessions = strconv.Itoa(n)
		}
		notes := ""
		if rec.DeclaredTier == "T0" {
			if d, ok := agents.Get(rec.Key); ok && d.T0Reason != "" {
				notes = "t0_reason=" + string(d.T0Reason)
			}
		}
		PrintHuman(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\t%s", rec.Key, rec.DeclaredTier, installed, root, sessions, notes)
	}
	return nil
}

func collectAgentProbe(ctx context.Context) (probe.Artifact, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	env := agents.Env{Home: home, LookupEnv: os.Getenv}
	return probe.Collect(ctx, env, agents.All(), probe.Options{})
}

func sessionCounts(ctx context.Context) map[string]int {
	out := map[string]int{}
	home, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	env := agents.Env{Home: home, LookupEnv: os.Getenv}
	for _, d := range agents.Capable(agents.CapabilityIndex) {
		if d.NewIndexSource == nil {
			continue
		}
		source, err := d.NewIndexSource(env)
		if err != nil || source == nil {
			continue
		}
		result, err := source.Scan(ctx)
		if err != nil {
			continue
		}
		out[d.Key] = len(result.Records)
	}
	return out
}

const (
	matrixARows = 10
	matrixBRows = 9
	matrixGRows = 8
	matrixHRows = 6
	matrixCRows = 6
	matrixDRows = 5
	matrixERows = 6
	matrixFRows = 2
)

type acceptanceMatrix struct {
	Schema       string                `json:"schema"`
	RowCount     int                   `json:"row_count"`
	CoreRowCount int                   `json:"core_row_count"`
	Core         map[string]int        `json:"core"`
	Agents       []acceptanceAgentRows `json:"agents"`
}

type acceptanceAgentRows struct {
	Key      string   `json:"key"`
	Tier     string   `json:"tier"`
	RowCount int      `json:"row_count"`
	Rows     []string `json:"rows"`
}

func buildAcceptanceMatrix() acceptanceMatrix {
	core := map[string]int{"A": matrixARows, "B": matrixBRows, "G": matrixGRows, "H": matrixHRows}
	coreCount := matrixARows + matrixBRows + matrixGRows + matrixHRows
	var agentsRows []acceptanceAgentRows
	total := coreCount
	for _, d := range agents.All() {
		row := agentMatrixRows(d)
		total += row.RowCount
		agentsRows = append(agentsRows, row)
	}
	return acceptanceMatrix{
		Schema:       "PHASE5-ACCEPTANCE-MATRIX-V1",
		RowCount:     total,
		CoreRowCount: coreCount,
		Core:         core,
		Agents:       agentsRows,
	}
}

func agentMatrixRows(d agents.Descriptor) acceptanceAgentRows {
	var rows []string
	switch d.Tier {
	case agents.TierKnown:
		rows = numbered("F", matrixFRows)
	default:
		if d.Tier >= agents.TierDiscover {
			rows = append(rows, numbered("C", matrixCRows)...)
		}
		if d.Tier >= agents.TierHandoffFrom {
			rows = append(rows, numbered("D", matrixDRows)...)
		}
		if d.Tier >= agents.TierResume {
			rows = append(rows, numbered("E", matrixERows)...)
		}
	}
	return acceptanceAgentRows{
		Key:      d.Key,
		Tier:     d.Tier.String(),
		RowCount: len(rows),
		Rows:     rows,
	}
}

func numbered(prefix string, n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = fmt.Sprintf("%s%d", prefix, i+1)
	}
	return out
}

func runDoctorAcceptanceMatrix(cmd *cobra.Command, asJSON bool) error {
	matrix := buildAcceptanceMatrix()
	if asJSON {
		return WriteJSON(cmd.OutOrStdout(), matrix)
	}
	PrintHuman(cmd.OutOrStdout(), "phase5_acceptance_matrix")
	PrintHuman(cmd.OutOrStdout(), "row_count=%d", matrix.RowCount)
	PrintHuman(cmd.OutOrStdout(), "core A=%d B=%d G=%d H=%d core_row_count=%d",
		matrix.Core["A"], matrix.Core["B"], matrix.Core["G"], matrix.Core["H"], matrix.CoreRowCount)
	for _, agent := range matrix.Agents {
		PrintHuman(cmd.OutOrStdout(), "%s %s row_count=%d rows=%s",
			agent.Key, agent.Tier, agent.RowCount, strings.Join(agent.Rows, ","))
	}
	return nil
}
