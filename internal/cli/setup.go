package cli

import (
	"context"
	"os"

	"github.com/HarjjotSinghh/reinstate/internal/doctor"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup helpers",
	}
	var asJSON bool
	check := &cobra.Command{
		Use:   "check",
		Short: "Read-only preflight checks",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			rep, err := doctor.Run(context.Background(), doctor.Options{
				Home:     os.Getenv("REINSTATE_HOME"),
				SelfTest: false,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return WriteJSON(c.OutOrStdout(), rep)
			}
			PrintHuman(c.OutOrStdout(), "%s", doctor.FormatHuman(rep))
			code := doctor.ExitCode(rep)
			if code != ExitOK {
				return NewExitError(code, rep.Summary)
			}
			return nil
		},
	}
	check.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.AddCommand(check)
	return cmd
}
