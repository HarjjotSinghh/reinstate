package hop

import "strings"

// Sign-in refusals.
//
// Step 2 of the sign-in happens in a browser and step 3 is the CLI polling
// in a terminal (docs/hop.md, "Protocol"). When the browser half ends
// without enrolling a device it records why on the login session, and the
// poll answers `{"status": "refused", "code", "reason"}` with HTTP 403.
// Before that existed the session simply stayed pending: the person read
// what to do in the browser while `rein login` polled on to a timeout and
// then reported an expiry, which was both wrong and unactionable.
//
// Two rules follow from the protocol and are honoured by this package:
//
//   - Switch on the code, never on the HTTP status. Every refusal answers
//     403 today whatever status its browser page carried, and the status a
//     future one answers with is not this client's business.
//   - A code this build has never heard of is still a refusal, and its
//     `reason` is still the sentence a person needs. An unknown code is
//     therefore handled, not rejected — see [RefusedError.Sentence].
const (
	// CodeQuotaDevices: the account already holds the devices its plan
	// allows. The same code POST /v1/locker/credentials answers when the
	// same account is over the same limit, which is why lockerError and a
	// refused sign-in share it rather than spelling it two ways.
	CodeQuotaDevices = "quota_devices"
	// CodeQuotaStorage: the locker holds the bytes the plan allows.
	CodeQuotaStorage = "quota_storage"
	// CodeQuotaPushRate: the account minted the credentials the plan allows
	// this hour.
	CodeQuotaPushRate = "quota_push_rate"

	// CodeLoginExpired: the sign-in link was opened after its deadline.
	// Distinct on the wire from the `expired` status, which is a link that
	// was never opened at all; both are terminal and mean the same thing to
	// a person.
	CodeLoginExpired = "login_expired"
	// CodeGitHubNoCode: GitHub came back with no authorization code, which
	// is what cancelling or denying there looks like from here.
	CodeGitHubNoCode = "github_no_code"
	// CodeGitHubRejected: GitHub refused to exchange the authorization code.
	CodeGitHubRejected = "github_rejected"
	// CodeAccountLinked: the verified address belongs to an account already
	// linked to a different GitHub identity.
	CodeAccountLinked = "account_linked"
	// CodeInternalError: the control plane could not finish the sign-in. It
	// is a refusal rather than a silence for the same reason as the rest:
	// leaving the session pending is a CLI that polls to a timeout while the
	// browser shows an error.
	CodeInternalError = "internal_error"
)

// signInRefusalCodes is every sign-in refusal code this build knows. It is
// the list docs/hop.md is gated against and the list the CLI's guidance
// table is gated against, so a code added here without a documented row or
// a decided exit code fails a test rather than reaching a person as a bare
// string.
//
// It is deliberately not a closed set at runtime: the control plane may be
// newer than the CLI, and a code that is not in this list is still printed
// with its sentence and still stops the poll.
var signInRefusalCodes = []string{
	CodeQuotaDevices,
	CodeLoginExpired,
	CodeGitHubNoCode,
	CodeGitHubRejected,
	CodeAccountLinked,
	CodeInternalError,
}

// SignInRefusalCodes returns the sign-in refusal codes this build knows, in
// a fresh slice.
func SignInRefusalCodes() []string {
	out := make([]string, len(signInRefusalCodes))
	copy(out, signInRefusalCodes)
	return out
}

// GenericRefusalReason is printed for a refusal that arrived carrying no
// sentence at all. The control plane substitutes its own equivalent before
// the answer leaves it, so this is the second of two belts: what it
// protects against is a person being shown an empty line where the
// explanation should be.
const GenericRefusalReason = "This sign-in was refused. Run `rein login` again."

// RefusedError is a sign-in the control plane refused during the browser
// approval: no device was enrolled, and the session is terminal — it is
// never approved afterwards, and the link enrols nothing if it is opened
// again. A caller that receives one stops polling.
type RefusedError struct {
	// Code names the kind of refusal. It is empty only if a control plane
	// sent a refusal without one.
	Code string
	// Reason is the exact sentence the browser page showed. It may contain
	// backticked command text and is meant to be printed verbatim.
	Reason string
}

// Sentence is the reason to show a person: the control plane's own words
// when it sent any, and [GenericRefusalReason] when it did not.
func (e *RefusedError) Sentence() string {
	if r := strings.TrimSpace(e.Reason); r != "" {
		return r
	}
	return GenericRefusalReason
}

func (e *RefusedError) Error() string { return e.Sentence() }
