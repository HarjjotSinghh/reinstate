package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/device"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// hopCommandOptions are the deterministic seams of the hosted-tier sign-in
// commands. Production leaves every field nil.
type hopCommandOptions struct {
	// tokens overrides the OS keyring that holds the device token.
	tokens credentials.DeviceTokenStore
	// openBrowser overrides launching the system browser for the sign-in URL.
	openBrowser func(url string) error
	// sleep overrides the wait between polls so tests never block.
	sleep func(context.Context, time.Duration) error
	// deviceName overrides the hostname reported to the control plane.
	deviceName string
}

func (o hopCommandOptions) tokenStore() credentials.DeviceTokenStore {
	if o.tokens != nil {
		return o.tokens
	}
	return credentials.NewKeyringStore()
}

func (o hopCommandOptions) browser() func(string) error {
	if o.openBrowser != nil {
		return o.openBrowser
	}
	return openSystemBrowser
}

func (o hopCommandOptions) name() string {
	if name := strings.TrimSpace(o.deviceName); name != "" {
		return name
	}
	if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return "unnamed-device"
}

// controlPlaneURL resolves the control plane from the environment, the
// optional [hop] config section, or the production default. A missing or
// unreadable config is not an error here: sign-in precedes `rein init`.
func controlPlaneURL() string {
	home, err := config.Home()
	if err != nil {
		return hop.ResolveURL(nil)
	}
	cfg, err := config.LoadConfig(home)
	if err != nil {
		return hop.ResolveURL(nil)
	}
	return hop.ResolveURL(cfg)
}

func newLoginCmd(o hopCommandOptions) *cobra.Command {
	var (
		addr      string
		asJSON    bool
		noBrowser bool
	)
	cmd := &cobra.Command{
		Use:   "login [--email ADDRESS]",
		Short: "Sign this device in to Reinstate Hop (GitHub or email link)",
		Long: "Sign in to the hosted tier without a password. By default the control plane\n" +
			"starts a GitHub sign-in in your browser; --email sends a one-time link instead.\n" +
			"On approval the device token is stored in the OS keyring, never in a file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd, o, addr, asJSON, noBrowser)
		},
	}
	cmd.Flags().StringVar(&addr, "email", "", "sign in with a one-time link sent to this address instead of GitHub")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "print the sign-in URL instead of opening a browser")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func runLogin(cmd *cobra.Command, o hopCommandOptions, addr string, asJSON, noBrowser bool) error {
	ctx := cmd.Context()
	baseURL := controlPlaneURL()
	client := hop.New(baseURL)
	method := hop.MethodGitHub
	addr = strings.TrimSpace(addr)
	if addr != "" {
		method = hop.MethodEmail
	}
	info := hop.DeviceInfo{Name: o.name(), Platform: device.PlatformID(), LocationHint: hop.LocationHint()}
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if !asJSON {
		if existing, err := o.tokenStore().GetDeviceToken(); err == nil {
			PrintHuman(errOut, "This device is already signed in (device %s at %s); continuing enrols it again as a new device and replaces the stored token.", existing.DeviceID, existing.ControlPlaneURL)
		}
		if plaintextRemote(baseURL) {
			PrintHuman(errOut, "Warning: %s is plain http to a non-loopback host; the device token will travel unencrypted.", baseURL)
		}
	}

	session, err := client.StartLogin(ctx, method, addr, info)
	if err != nil {
		return loginError(err)
	}
	switch method {
	case hop.MethodGitHub:
		if !asJSON {
			PrintHuman(errOut, "Sign in with GitHub at:\n\n  %s\n", session.VerificationURL)
		}
		if !noBrowser {
			if err := o.browser()(session.VerificationURL); err != nil && !asJSON {
				PrintHuman(errOut, "Could not open a browser (%v); open the URL above yourself.", err)
			}
		}
	case hop.MethodEmail:
		if !asJSON {
			PrintHuman(errOut, "A sign-in link was sent to %s. Open it on any device to approve this one.", addr)
		}
	}
	if !asJSON {
		PrintHuman(errOut, "Waiting for approval (expires %s; Ctrl-C to cancel)...", session.ExpiresAt)
	}

	approval, err := client.WaitForApproval(ctx, session, o.sleep)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return NewExitError(ExitRuntime, "login cancelled")
		}
		// A refusal is the browser half saying why it enrolled nothing. It
		// is reported here rather than through loginError because it is not
		// an HTTP failure to be classified by status: the code and the
		// sentence are the answer (see hop_refusal.go).
		var refused *hop.RefusedError
		if errors.As(err, &refused) {
			return loginRefusalError(cmd.Root(), refused)
		}
		return loginError(err)
	}
	tok := credentials.DeviceToken{
		Token:           approval.DeviceToken,
		ControlPlaneURL: baseURL,
		AccountID:       approval.Account.ID,
		DeviceID:        approval.Device.ID,
	}
	if err := o.tokenStore().SetDeviceToken(tok); err != nil {
		return NewExitError(ExitAuthStorage, err.Error())
	}
	if asJSON {
		return WriteJSON(out, whoamiJSON(baseURL, hop.Identity{Account: approval.Account, Device: approval.Device}))
	}
	PrintHuman(out, "Signed in to Reinstate Hop as %s.", accountLabel(approval.Account))
	PrintHuman(out, "This device is enrolled as %q (%s); its token is in the OS keyring.", approval.Device.Name, approval.Device.Platform)
	return nil
}

// plaintextRemote reports whether baseURL is http:// to a host other than
// loopback, where a device token would be sent in the clear.
func plaintextRemote(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

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

func loginError(err error) error {
	var he *hop.Error
	if errors.As(err, &he) {
		switch {
		case he.Status == 400:
			return NewExitError(ExitUsage, err.Error())
		case he.Status == 503:
			return NewExitError(ExitConfig, err.Error())
		}
		return NewExitError(ExitAuthStorage, err.Error())
	}
	return NewExitError(ExitRuntime, err.Error())
}

func openSystemBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	return c.Start()
}
