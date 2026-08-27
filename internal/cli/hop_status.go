package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

func newHopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hop",
		Short: "Reinstate Hop: the account's locker and its usage",
		Long: "Commands for the hosted tier beyond sign-in. The locker is the storage bucket\n" +
			"provisioned for exactly one account, reached with hourly credentials the control\n" +
			"plane mints for this device. Every session object Reinstate writes to it is\n" +
			"ciphertext; one object it writes is not. `keyring.v1.json` is plaintext by\n" +
			"design: it holds no usable key, and it names the account's profile id, every\n" +
			"enrolled device's id, public key and enrolment time, and one entry per key\n" +
			"generation with the time it started — so a locker with more than one\n" +
			"generation also shows which devices stopped being enrolled, and when.\n" +
			"docs/hop/object-format.md lists it in full.",
	}
	cmd.AddCommand(newHopStatusCmd())
	cmd.AddCommand(newHopCredentialsCmd())
	return cmd
}

func newHopStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the locker's endpoint, bucket, location, usage, and plan limits",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, client, err := hostedSession(cmd)
			if err != nil {
				return err
			}
			locker, err := client.LockerStatus(cmd.Context(), tok.Token)
			if errors.Is(err, hop.ErrNoLocker) {
				if asJSON {
					return WriteJSON(cmd.OutOrStdout(), map[string]any{"control_plane": tok.ControlPlaneURL, "locker": nil})
				}
				PrintHuman(cmd.OutOrStdout(), "Hop:    %s", tok.ControlPlaneURL)
				PrintHuman(cmd.OutOrStdout(), "Locker: not provisioned yet; the first push creates it")
				return nil
			}
			if err != nil {
				return hopExitError(err)
			}
			if asJSON {
				return WriteJSON(cmd.OutOrStdout(), map[string]any{"control_plane": tok.ControlPlaneURL, "locker": locker})
			}
			out := cmd.OutOrStdout()
			PrintHuman(out, "Hop:      %s", tok.ControlPlaneURL)
			PrintHuman(out, "Locker:   %s at %s (region %s, location %s)", locker.Bucket, locker.Endpoint, locker.Region, locker.LocationHint)
			PrintHuman(out, "Plan:     %s", locker.Plan)
			PrintHuman(out, "Usage:    %s in %d object(s) of %s (measured %s)", humanBytes(locker.Usage.Bytes), locker.Usage.Objects, quotaBytes(locker.Quota.StorageBytes), orNever(locker.Usage.ObservedAt))
			PrintHuman(out, "Devices:  %d of %s", locker.Devices, quotaCount(locker.Quota.Devices))
			PrintHuman(out, "Pushes:   up to %s credential mints per hour", quotaCount(locker.Quota.MintsPerHour))
			PrintHuman(out, "Created:  %s; first push: %s", locker.CreatedAt, orNever(locker.FirstPushAt))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func orNever(s string) string {
	if s == "" {
		return "never"
	}
	return s
}

func quotaBytes(n int64) string {
	if n <= 0 {
		return "unlimited"
	}
	return humanBytes(n)
}

func quotaCount(n int) string {
	if n <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", n)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
