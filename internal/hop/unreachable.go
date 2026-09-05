package hop

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
)

// KindControlPlaneUnreachable is the stable machine-readable classification
// every [UnreachableError] carries under details.kind in --json output,
// whatever the underlying transport cause: the control plane could not be
// reached at all, as opposed to answering with its own refusal.
const KindControlPlaneUnreachable = "control_plane_unreachable"

// UnreachableError reports that URL could not be reached at all — no DNS
// answer, no route (connection refused), or no TLS handshake — as opposed
// to a reachable control plane's own answer, which arrives as *Error. See
// [ClassifyUnreachable].
type UnreachableError struct {
	// URL is the control plane this client tried to reach.
	URL string
	// Phrase is the short, timeless classification printed after the
	// colon in Error(): "no DNS answer for <host>", "connection refused",
	// or "the TLS handshake failed".
	Phrase string
	// cause is the transport error the phrase was classified from. Kept
	// for errors.Is/errors.As rather than shown directly: an *url.Error's
	// own text is often the dialled IP and port, which repeats the URL
	// above without saying anything a person troubleshooting DNS, a
	// firewall, or a certificate needs that Phrase does not already say.
	cause error
}

// Error reads as one sentence naming the control plane and why it could
// not be reached, e.g. "could not reach the Reinstate Hop control plane at
// https://hop.reinstate.dev: connection refused".
func (e *UnreachableError) Error() string {
	return fmt.Sprintf("could not reach the Reinstate Hop control plane at %s: %s", e.URL, e.Phrase)
}

// Unwrap exposes the classified transport failure, so errors.Is and
// errors.As still see it — [Unreachable] included, which every
// UnreachableError still answers true to through this chain.
func (e *UnreachableError) Unwrap() error { return e.cause }

// ClassifyUnreachable inspects err, as returned by a Client method, for one
// of the three transport failures this build classifies as the control
// plane being unreachable at all: DNS failure, connection refused, and a
// failed TLS handshake. It returns (nil, false) for every other error,
// including a reachable control plane's own answer (*Error) and a bare
// timeout — both stay whatever they already were to their caller.
//
// baseURL is recorded on the returned error for display and for the --json
// form; it is the URL the caller resolved and dialled, not something
// parsed back out of err.
func ClassifyUnreachable(baseURL string, err error) (*UnreachableError, bool) {
	if err == nil {
		return nil, false
	}
	var already *UnreachableError
	if errors.As(err, &already) {
		return already, true
	}
	if !Unreachable(err) {
		return nil, false
	}

	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &dnsErr):
		phrase := "no DNS answer"
		if dnsErr.Name != "" {
			phrase = "no DNS answer for " + dnsErr.Name
		}
		return &UnreachableError{URL: baseURL, Phrase: phrase, cause: err}, true
	case isConnectionRefused(err):
		return &UnreachableError{URL: baseURL, Phrase: "connection refused", cause: err}, true
	case isTLSHandshakeFailure(err):
		return &UnreachableError{URL: baseURL, Phrase: "the TLS handshake failed", cause: err}, true
	}
	return nil, false
}

// isTLSHandshakeFailure recognizes the ways crypto/tls and crypto/x509
// report a handshake that never produced a usable connection: an
// unverifiable certificate chain (self-signed, expired, wrong host) or a
// peer that never spoke TLS at all. It does not match every possible TLS
// error — an already-established connection failing later is not "could
// not reach the control plane" — only the ones that leave the handshake
// itself incomplete.
func isTLSHandshakeFailure(err error) bool {
	var certVerify *tls.CertificateVerificationError
	if errors.As(err, &certVerify) {
		return true
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return true
	}
	var hostname x509.HostnameError
	if errors.As(err, &hostname) {
		return true
	}
	var invalid x509.CertificateInvalidError
	if errors.As(err, &invalid) {
		return true
	}
	var recordHeader tls.RecordHeaderError
	if errors.As(err, &recordHeader) {
		return true
	}
	return false
}
