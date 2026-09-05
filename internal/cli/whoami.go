package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

func newWhoamiCmd(o hopCommandOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the Reinstate Hop account this device is enrolled under",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := o.tokenStore().GetDeviceToken()
			if errors.Is(err, credentials.ErrNoDeviceToken) {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			if err != nil {
				return NewExitError(ExitAuthStorage, err.Error())
			}
			id, err := hop.New(tok.ControlPlaneURL).Whoami(cmd.Context(), tok.Token)
			if errors.Is(err, hop.ErrUnauthorized) {
				return NewExitError(ExitAuthStorage, "this device's token was rejected by the control plane (revoked or stale); run `rein login` again")
			}
			if unreachable, ok := hop.ClassifyUnreachable(tok.ControlPlaneURL, err); ok {
				return controlPlaneUnreachableError(unreachable)
			}
			if err != nil {
				return NewExitError(ExitRuntime, err.Error())
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), whoamiJSON(tok.ControlPlaneURL, id))
			}
			out := cmd.OutOrStdout()
			PrintHuman(out, "Account: %s", accountLabel(id.Account))
			if id.Account.Plan != "" {
				PrintHuman(out, "Plan:    %s (locker location %s)", id.Account.Plan, id.Account.LocationHint)
			}
			PrintHuman(out, "Device:  %s (%s, enrolled %s)", id.Device.Name, id.Device.Platform, id.Device.CreatedAt)
			PrintHuman(out, "Hop:     %s", tok.ControlPlaneURL)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func whoamiJSON(baseURL string, id hop.Identity) map[string]any {
	return map[string]any{
		"control_plane": baseURL,
		"account":       id.Account,
		"device":        id.Device,
	}
}

func accountLabel(a hop.Account) string {
	switch {
	case a.GitHubLogin != "" && a.Email != "":
		return fmt.Sprintf("%s (GitHub @%s)", a.Email, a.GitHubLogin)
	case a.GitHubLogin != "":
		return "GitHub @" + a.GitHubLogin
	case a.Email != "":
		return a.Email
	}
	return a.ID
}
