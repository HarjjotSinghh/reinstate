package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	var asJSON bool
	var selfTest bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run redacted diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home := os.Getenv("REINSTATE_HOME")
			rep, err := doctor.Run(context.Background(), doctor.Options{
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
	return cmd
}
