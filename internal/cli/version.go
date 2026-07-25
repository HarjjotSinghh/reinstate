package cli

import (
	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/version"
)

func newVersionCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]string{
					"name":    "reinstate",
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				})
			}
			PrintHuman(cmd.OutOrStdout(), "%s", version.String())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}
