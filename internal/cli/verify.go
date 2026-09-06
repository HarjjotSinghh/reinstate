package cli

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

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
		Short: "Storage-level operations on the configured sync profile",
		Long:  "Commands that inspect or move the configured storage rather than individual sessions. `rein push`, `rein pull`, and `rein status` move sessions.",
	}
	cmd.AddCommand(newSyncVerifyCmd())
	cmd.AddCommand(newSyncMigrateCmd())
	return cmd
}

func newSyncVerifyCmd() *cobra.Command {
	var asJSON, post bool
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Show that the store holds ciphertext this device can open, and what it holds in the clear",
		Long: `Runs the checks behind the zero-knowledge claim and prints a verification
report a non-expert can read and repeat: list the locker with this device's
credentials; fetch an object and show it is ciphertext; decrypt it locally
and show what it contains; and, on a Hop locker, show that the same
credentials are refused from a bucket the operator owns (the reference
locker). BYO storage runs the first three checks; the fourth is reported as
not applicable.

Every session object Reinstate writes to the locker is ciphertext. One
object it writes is not, and the report names it rather than passing over
it: ` + "`" + `keyring.v1.json` + "`" + ` is plaintext by design. It holds no usable key,
and it gives up the account's profile id, every enrolled device's id,
public key and enrolment time, and one entry per key generation with the
time it started — so a locker with more than one generation also shows
which devices stopped being enrolled, and when. See
docs/hop/object-format.md for the full list.

The exit status is 0 when every step passed or did not apply — including a
profile that has pushed nothing yet, where there is nothing to check — and
` + "`" + `7` + "`" + ` (safety) when a step failed. A control plane or a storage endpoint
that could not be reached exits ` + "`" + `1` + "`" + ` after printing the report, because a
step that got no answer is not a step that failed. On a Hop locker the step
results are posted to the control plane for the account console unless
--post=false is given: the step verdicts and their sentences, with the
per-session detail lines stripped, so no object contents, session names or
project paths go with them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, cfg, home, err := engineFromConfig(cmd, "")
			if err != nil {
				// A Hop locker is opened with credentials the control plane
				// mints and a keyring fetched from the bucket, so either one
				// being unreachable stops every check before it starts. That
				// is still worth a report: a bare dial error leaves the
				// reader unable to tell an outage from a finding, which is
				// the one distinction this command exists to make. Print what
				// could and could not run, then exit with the runtime code
				// every other hosted command uses for an outage.
				report, exit := notStartedReport(cmd, cfg, err)
				if report != nil {
					if asJSON {
						if jsonErr := WriteJSON(cmd.OutOrStdout(), map[string]any{"report": report, "posted": false}); jsonErr != nil {
							return jsonErr
						}
					} else {
						report.WriteHuman(cmd.OutOrStdout())
					}
				}
				if exit != nil {
					return exit
				}
				return err
			}
			hosted := hostedFrom(cmd)
			report := runVerification(cmd, eng, cfg, home, hosted)
			posted := false
			// Nothing was checked, so there is no verdict to show in the
			// account console and nothing to post.
			if hosted != nil && post && report.Outcome != verify.NotApplicable {
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
			if report.Failed() {
				return NewExitError(ExitSafety, "verification failed; see the report")
			}
			// The storage endpoint gave no answer, so nothing was checked.
			// That is not a failed verification and must not exit 7, but it
			// is not a clean run either: a script that reads the exit code
			// alone would otherwise take an outage for a pass. It is the
			// same code an unreachable control plane exits with.
			if report.NotVerified() {
				return NewExitError(ExitRuntime, "nothing could be verified; see the report")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&post, "post", true, "post the step results to the control plane (Hop only); --post=false keeps them local")
	return cmd
}

// notStartedReport returns the report to print when the profile could not
// be opened at all, together with the exit error to use in place of the
// command's own.
//
// There are two outages that stop every check before it starts, and both
// have to read as outages rather than as findings: a control plane nobody
// can reach (a Hop locker's credentials are minted there) and a storage
// endpoint that gives no answer (the keyring is fetched from the bucket).
// Both exit `1`. Every other failure — a bad config, a rejected token, a
// quota refusal, a passphrase that does not open the keyring — returns
// nil for both, and the command's own error and exit code stand.
func notStartedReport(cmd *cobra.Command, cfg *schema.Config, err error) (*verify.Report, error) {
	hosted := hostedFrom(cmd)
	opts := verify.Options{Storage: verify.StorageBYO, ClientVersion: version.String()}
	if hosted != nil {
		opts.Storage = verify.StorageHop
	}
	if cfg != nil {
		opts.Locker = verify.LockerInfo{Endpoint: cfg.Storage.Endpoint, Bucket: cfg.Storage.Bucket, Prefix: cfg.Storage.Prefix}
	}
	if ctx := cmd.Context(); ctx != nil {
		if now, ok := ctx.Value(verifyNowContextKey{}).(func() time.Time); ok {
			opts.Now = now
		}
	}
	switch {
	case hosted != nil && hop.Unreachable(hosted.LastError()):
		// The control plane's own error already carries ExitRuntime, so the
		// command's error stands and only the report is added.
		return verify.NotRun(opts, hosted.LastError()), nil
	case verify.Unreachable(err):
		return verify.NotReached(opts, err), NewExitError(ExitRuntime, "nothing could be verified; see the report")
	}
	return nil, nil
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
		if client != nil {
			opts.CredentialID = func(ctx context.Context) (string, error) {
				creds, err := client.CurrentCredentials(ctx)
				return creds.AccessKeyID, err
			}
		}
		opts.OpenReference = func(ctx context.Context, ref hop.Reference) (verify.Probe, error) {
			if client == nil {
				return verify.Probe{}, errors.New("the locker client cannot lend its credentials")
			}
			creds, err := client.CurrentCredentials(ctx)
			if err != nil {
				return verify.Probe{}, err
			}
			// The probe client refuses redirects and records every
			// exchange, so the step can show the locker credential went to
			// the endpoint the control plane pinned and nowhere else.
			httpClient, exchanges := verify.ProbeClient(nil)
			b, err := s3.New(ctx, s3.Config{
				Endpoint: ref.Endpoint, Region: ref.Region, Bucket: ref.Bucket,
				Credentials: s3.StaticCredentials(creds), HTTPClient: httpClient,
				MaxAttempts: 1,
			})
			if err != nil {
				return verify.Probe{}, err
			}
			return verify.Probe{Backend: b, AccessKeyID: creds.AccessKeyID, Exchanges: exchanges}, nil
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
// is reported on stderr and retried after the next push that uploads
// something. The returned
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
	if report.Outcome == verify.NotApplicable {
		// Nothing was checked, so there is nothing to post and nothing to
		// record: the hook must stay armed for the next push that uploads
		// something. A push that uploaded is not expected to land here, so
		// say so rather than pass silently — and say which of the two it
		// was, since a storage endpoint that gave no answer is not the
		// same thing as a locker with nothing in it.
		what := "had nothing to check"
		if report.NotVerified() {
			what = "could not run"
		}
		PrintHuman(cmd.ErrOrStderr(), "note: the verification after the first push %s: %s", what, report.Summary)
		return nil
	}
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
	// The line claims only the objects the checks fetched: a manifest-only
	// locker verified only the index.
	checked, verb := report.CheckedPhrase(), "are"
	if checked == "" {
		checked = "the fetched objects"
	}
	if !strings.Contains(checked, " and ") {
		verb = "is"
	}
	switch {
	case report.Passed() && report.IsolationChecked():
		PrintHuman(cmd.ErrOrStderr(), "First push from this device verified: %s fetched from the locker %s ciphertext this device can open, and this account's credentials are refused by a bucket that is not its own (rein sync verify shows the full report).", checked, verb)
	case report.Passed():
		PrintHuman(cmd.ErrOrStderr(), "First push from this device verified: %s fetched from the locker %s ciphertext this device can open. Whether this account's credentials reach other buckets was not checked (rein sync verify shows the full report and why).", checked, verb)
	default:
		PrintHuman(cmd.ErrOrStderr(), "WARNING: the verification after the first push FAILED. Full report:")
		report.WriteHuman(cmd.ErrOrStderr())
	}
	return summary
}
