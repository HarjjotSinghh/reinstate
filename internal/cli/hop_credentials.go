package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// newHopCredentialsCmd prints the locker credentials so a customer can
// repeat the verification by hand.
//
// `rein sync verify` is the command that asks people to trust the product
// less and check it more, and `docs/hop/object-format.md` ships an S3
// recipe for repeating its steps. On a Hop locker none of it could be
// run: the only credentials that reach the locker are the hourly ones the
// control plane mints, and nothing exposed them. The recipe was
// unfollowable, which is a worse position than not shipping one.
//
// What this prints is deliberately the safe half of that problem. These
// are the same credentials `rein push` already uses: minted for this
// account, scoped by the provider to this account's bucket and no other
// (which is exactly what step 4 of the verification tests), and dead
// within the hour. They open nothing that this device could not already
// open.
//
// They do not read "nothing but ciphertext", which is what this command
// and four other places used to say. `keyring.v1.json` is plaintext by
// design and sits in the same bucket, and anyone holding these credentials
// reads it: the account's profile id, every enrolled device's id, public
// key and enrolment time, and the key generations. It holds no usable key
// — every copy of the root key in it is wrapped to something the bucket
// does not contain — but it is metadata about the account's devices, and a
// command that hands out a credential has to say so rather than round it
// off to "ciphertext".
//
// The other half is not printed and will not be. Steps 1, 2 and 4 need a
// credential; step 3 needs the account's root key, and a command that
// wrote that key to a file would hand over every object the account has
// ever written, past and future, in one command — a far larger exposure
// than the documentation gap it would close. `crypto.RootKeyIdentity`
// stays package-internal, and the object format now says plainly that
// step 3 is reproducible by hand only on BYO storage, where the key is a
// passphrase the person already holds.
func newHopCredentialsCmd() *cobra.Command {
	var asJSON, asExport bool
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Print short-lived, bucket-scoped credentials for this account's locker",
		Long: `Mints one credential set for this account's locker and prints it, so the
first, second and fourth checks of ` + "`" + `rein sync verify` + "`" + ` can be repeated by
hand with any S3 client (see docs/hop/object-format.md, "Reproducing the
checks by hand").

These are the same credentials rein push uses: valid for at most an hour,
scoped by the storage provider to this account's bucket and no other, and
able to reach nothing this device cannot already reach. Every session
object they can read is ciphertext; ` + "`" + `keyring.v1.json` + "`" + ` is not, and never
was — it is plaintext by design and holds no usable key, but it names the
account's profile id and every enrolled device's id, public key and
enrolment time. Each run mints a fresh set and counts against the hourly
mint quota that rein hop status shows.

--export prints the same values as shell ` + "`" + `export` + "`" + ` statements and nothing
else, for ` + "`" + `eval "$(rein hop credentials --export)"` + "`" + `; it adds
REIN_LOCKER_BUCKET, which the documented recipe uses.

The third check needs the account's root key, which never leaves the
device and which no command exports.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, client, err := hostedSession(cmd)
			if err != nil {
				return err
			}
			creds, err := client.MintCredentials(cmd.Context(), tok.Token)
			if err != nil {
				return hopExitError(err)
			}
			out := cmd.OutOrStdout()
			switch {
			case asJSON:
				return WriteJSON(out, creds)
			case asExport:
				// Only export statements, so the whole of stdout can be
				// eval'd. The previous recipe piped this output through
				// `grep '^AWS_'` and eval'd that, which sets shell
				// variables the aws client never sees: they are not
				// exported, so every command after the first line ran
				// without credentials.
				for _, kv := range [][2]string{
					{"AWS_ACCESS_KEY_ID", creds.AccessKeyID},
					{"AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey},
					{"AWS_SESSION_TOKEN", creds.SessionToken},
					{"AWS_ENDPOINT_URL", creds.Endpoint},
					{"AWS_REGION", creds.Region},
					{"AWS_DEFAULT_REGION", creds.Region},
					{"REIN_LOCKER_BUCKET", creds.Bucket},
				} {
					PrintHuman(out, "export %s=%s", kv[0], shellQuote(kv[1]))
				}
			default:
				PrintHuman(out, "Locker:  %s at %s (region %s)", creds.Bucket, creds.Endpoint, creds.Region)
				PrintHuman(out, "Expires: %s", orNever(creds.ExpiresAt))
				PrintHuman(out, "")
				PrintHuman(out, "AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID)
				PrintHuman(out, "AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey)
				PrintHuman(out, "AWS_SESSION_TOKEN=%s", creds.SessionToken)
				PrintHuman(out, "AWS_ENDPOINT_URL=%s", creds.Endpoint)
			}
			// The caution goes to stderr so the values above can be piped,
			// redirected or eval'd without it, and is printed every time: a
			// credential is worth less caution than a key, not none.
			expiry := "within the hour"
			if creds.ExpiresAt != "" {
				expiry = "at " + creds.ExpiresAt
			}
			PrintHuman(cmd.ErrOrStderr(), "note: these reach this account's own bucket and no other (rein sync verify step 4 checks that), and expire %s. Every session object in that bucket is ciphertext; keyring.v1.json is plaintext and names the account and its enrolled devices. Treat the secret key and session token as secrets until they expire; the access key id is a public identifier.", expiry)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&asExport, "export", false, "print shell export statements and nothing else, for eval \"$(rein hop credentials --export)\"")
	cmd.MarkFlagsMutuallyExclusive("json", "export")
	return cmd
}

// shellQuote wraps a value in single quotes for POSIX `sh`, so an eval of
// the export statements can never run part of a credential as a command.
// A single quote inside the value ends the quoted run, escapes one quote,
// and opens the next.
func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}
