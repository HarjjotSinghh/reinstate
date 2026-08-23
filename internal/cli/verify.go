package cli

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
	"github.com/HarjjotSinghh/reinstate/internal/sync"
	"github.com/HarjjotSinghh/reinstate/internal/verify"
	"github.com/HarjjotSinghh/reinstate/internal/version"
)

// verifyNowContextKey lets tests pin the report's timestamp for golden
// output.
type verifyNowContextKey struct{}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Checks on the encrypted sync store",
		Long:  "Commands that inspect the configured storage rather than move sessions. `rein push`, `rein pull`, and `rein status` do the moving.",
	}
	cmd.AddCommand(newSyncVerifyCmd())
	return cmd
}

func newSyncVerifyCmd() *cobra.Command {
	var asJSON, post bool
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Show that the store holds only ciphertext this device can open",
		Long: `Runs the checks behind the zero-knowledge claim and prints a verification
report a non-expert can read and repeat: list the locker with this device's
credentials; fetch an object and show it is ciphertext; decrypt it locally
and show what it contains; and, on a Hop locker, show that the same
credentials are refused from a bucket the operator owns (the reference
locker). BYO storage runs the first three checks; the fourth is reported as
not applicable.

The exit status is 0 when every step passed or did not apply and ` + "`" + `4` + "`" + ` (safety)
when a step failed. On a Hop locker the step results (never object contents
or session names) are posted to the control plane for the account console
unless --post=false is given.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, cfg, home, err := engineFromConfig(cmd, "")
			if err != nil {
				return err
			}
			hosted := hostedFrom(cmd)
			report := runVerification(cmd, eng, cfg, home, hosted)
			posted := false
			if hosted != nil && post {
				if err := postVerification(cmd, report); err != nil {
					PrintHuman(cmd.ErrOrStderr(), "note: could not post the report to the control plane: %v", err)
				} else {
					posted = true
				}
			}
			if asJSON {
				if err := WriteJSON(cmd.OutOrStdout(), map[string]any{"report": report, "posted": posted}); err != nil {
					return err
				}
			} else {
				report.WriteHuman(cmd.OutOrStdout())
				if posted {
					PrintHuman(cmd.OutOrStdout(), "Step results posted to the control plane for the account console.")
				}
			}
			if !report.Passed() {
				return NewExitError(ExitSafety, "verification failed; see the report")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&post, "post", true, "post the step results to the control plane (Hop only); --post=false keeps them local")
	return cmd
}

// runVerification runs the checks against the engine's backend and keys.
// It is the one entry point for both the command and the post-first-push
// hook so a daemon can call it the same way.
func runVerification(cmd *cobra.Command, eng *sync.Engine, cfg *schema.Config, home string, hosted *hop.Source) *verify.Report {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	opts := verify.Options{
		Backend:       eng.Backend,
		Prefix:        eng.Prefix,
		Keys:          eng.Keys,
		Storage:       verify.StorageBYO,
		ClientVersion: version.String(),
	}
	if eng.Codec != nil {
		opts.Codec = eng.Codec
	}
	if now, ok := ctx.Value(verifyNowContextKey{}).(func() time.Time); ok {
		opts.Now = now
	}
	switch {
	case os.Getenv("REINSTATE_BACKEND") == "memory":
		opts.Locker = verify.LockerInfo{Bucket: "memory:" + memoryBackendRoot(home), Prefix: cfg.Storage.Prefix}
	case hosted != nil:
		opts.Storage = verify.StorageHop
		if locker, err := hosted.Locker(ctx); err == nil {
			opts.Locker = verify.LockerInfo{Endpoint: locker.Endpoint, Bucket: locker.Bucket, Prefix: locker.Prefix}
		}
		ref, err := referenceLocker(cmd)
		if err != nil {
			opts.ReferenceErr = err
		} else {
			opts.Reference = &ref
		}
		client, _ := eng.Backend.(*s3.Client)
		opts.OpenReference = func(ctx context.Context, ref hop.Reference) (backend.Backend, error) {
			if client == nil {
				return nil, errors.New("the locker client cannot lend its credentials")
			}
			creds, err := client.CurrentCredentials(ctx)
			if err != nil {
				return nil, err
			}
			return s3.New(ctx, s3.Config{
				Endpoint: ref.Endpoint, Region: ref.Region, Bucket: ref.Bucket,
				Credentials: s3.StaticCredentials(creds),
			})
		}
	default:
		opts.Locker = verify.LockerInfo{Endpoint: cfg.Storage.Endpoint, Bucket: cfg.Storage.Bucket, Prefix: cfg.Storage.Prefix}
	}
	return verify.Run(ctx, opts)
}

// referenceLocker asks the control plane where its probe lives.
func referenceLocker(cmd *cobra.Command) (hop.Reference, error) {
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return hop.Reference{}, err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return client.VerifyReference(ctx, tok.Token)
}

// postVerification sends the step results to the control plane.
func postVerification(cmd *cobra.Command, report *verify.Report) error {
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	_, err = client.PostVerifyReport(ctx, tok.Token, report.ForUpload())
	return err
}

// verifyAfterFirstPush runs the verification once per device, after the
// first push that uploaded something to a Hop locker, and posts the step
// results. It never fails the push: a verification that cannot run or post
// is reported on stderr and retried after the next push. The returned
// summary is added to the push's JSON output when the checks ran.
func verifyAfterFirstPush(cmd *cobra.Command, eng *sync.Engine, cfg *schema.Config, home string, hosted *hop.Source) map[string]any {
	if hosted == nil {
		return nil
	}
	state, err := config.LoadState(home)
	if err != nil || state.VerifyReportedAt != "" {
		return nil
	}
	report := runVerification(cmd, eng, cfg, home, hosted)
	summary := map[string]any{"outcome": report.Outcome, "posted": false}
	if err := postVerification(cmd, report); err != nil {
		PrintHuman(cmd.ErrOrStderr(), "note: verification ran (%s) but the report could not be posted: %v", report.Outcome, err)
	} else {
		summary["posted"] = true
		state.VerifyReportedAt = time.Now().UTC().Format(time.RFC3339)
		if err := config.SaveState(home, state); err != nil {
			PrintHuman(cmd.ErrOrStderr(), "note: could not record the verification: %v", err)
		}
	}
	if report.Passed() {
		PrintHuman(cmd.ErrOrStderr(), "First push from this device verified: the locker holds only ciphertext this device can open, and this account's credentials are refused elsewhere (rein sync verify shows the full report).")
	} else {
		PrintHuman(cmd.ErrOrStderr(), "WARNING: the verification after the first push FAILED. Full report:")
		report.WriteHuman(cmd.ErrOrStderr())
	}
	return summary
}
