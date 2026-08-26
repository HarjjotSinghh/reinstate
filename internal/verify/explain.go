package verify

import (
	"bytes"
	"errors"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/doctor"
)

// This file holds the prose the report shows when something went wrong.
// The command exists so that someone who has not read the source can
// satisfy themselves, so every error it prints has to name a cause in
// ordinary words. The underlying error is always kept after the gloss:
// the person who does read the source still needs it, and a support
// thread needs the exact text.

// explainBackendError puts a plain-English cause in front of a storage
// refusal. The S3 API's own vocabulary (`InvalidAccessKeyId`,
// `SignatureDoesNotMatch`) says nothing to a reader who has never signed
// an S3 request, and the wrapped form the backend returns
// ("backend: credential rejected (InvalidAccessKeyId)") says even less.
//
// It returns the sentence to show; the caller keeps err for the detail.
func explainBackendError(err error) string {
	if err == nil {
		return ""
	}
	var refusal *backend.Refusal
	if errors.As(err, &refusal) {
		switch refusal.Code {
		case "InvalidAccessKeyId":
			return "the storage endpoint does not recognise this access key id, so the credential has been withdrawn or has expired"
		case "SignatureDoesNotMatch":
			return "the storage endpoint rejected the request signature, so the secret half of the credential is wrong or this machine's clock is far off"
		case "ExpiredToken", "TokenRefreshRequired":
			return "the credential's session token has expired; locker credentials last at most an hour"
		case "InvalidToken":
			return "the storage endpoint rejected the credential's session token as invalid or withdrawn"
		}
		if refusal.Credential {
			return "the storage endpoint rejected the credential itself (" + refusal.Code + ") rather than the request"
		}
		return "the storage endpoint recognised the credential and refused the request anyway (" + refusal.Code + ")"
	}
	switch {
	case errors.Is(err, backend.ErrCredentialRejected):
		return "the storage endpoint rejected the credential itself rather than the request"
	case errors.Is(err, backend.ErrAccessDenied):
		return "the storage endpoint recognised the credential and refused the request anyway"
	case errors.Is(err, backend.ErrNotFound):
		return "the storage endpoint says there is no such object"
	}
	return ""
}

// withCause joins a plain-English cause to the error it came from, so the
// sentence a person acts on comes first and the exact text a maintainer
// needs is still there. A cause it has no gloss for degrades to the raw
// error rather than to silence.
func withCause(cause string, err error) string {
	if err == nil {
		return cause
	}
	if cause == "" {
		return err.Error()
	}
	return cause + " (" + err.Error() + ")"
}

// explainDecryptError says, in ordinary words, why an object did not open.
//
// The two failures a reader is most likely to meet both arrive as age's
// internal vocabulary and both mislead:
//
//   - Plaintext in the locker reports "parsing age header: file is empty"
//     however many bytes it holds, because age found no header to parse.
//     Read literally that says the object is truncated, which points the
//     reader away from the finding that matters: something wrote
//     unencrypted bytes into the locker.
//   - A key that does not match reports "identity did not match any of the
//     recipients: incorrect identity for recipient block: incorrect
//     passphrase" — three layers over one useful clause.
//
// storage is StorageHop or StorageBYO; body is the object as fetched.
func explainDecryptError(storage string, body []byte, err error) string {
	if err == nil {
		return ""
	}
	if !bytes.HasPrefix(body, []byte(ageHeader)) {
		return "it is not an age envelope at all — it does not begin with the age v1 header, so these are unencrypted bytes and there is nothing to decrypt"
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "incorrect passphrase"):
		if storage == StorageHop {
			return "the key held on this device does not open it: the object was sealed to a different root key"
		}
		return "the passphrase held on this device does not open it: the object was sealed under a different passphrase"
	case strings.Contains(text, "no identity matched"), strings.Contains(text, "identity did not match"):
		return "the key held on this device is not one of the object's recipients: it was sealed to a different key"
	case strings.Contains(text, "header MAC"), strings.Contains(text, "bad header MAC"):
		return "the envelope's header has been altered since it was written"
	}
	return "the object did not open with the key held on this device"
}

// redactLocalPath removes from a report line what a local path would tell
// a reader it was never meant to tell. `rein sync verify` exists to be
// shown to somebody else — a colleague, a reviewer, an auditor — so its
// output must not hand them the layout of the machine it ran on.
//
// doctor.RedactPath already covers an absolute path and the home
// directory. It cannot see the form the agent harnesses actually store,
// where an absolute path is flattened into a single directory name by
// replacing every character that is not alphanumeric with a dash
// ("C--Users-admin-Projects-app", "-Users-admin-Projects-app"). That form
// is not absolute and survives RedactPath untouched, and reversing it
// takes no effort at all, so each such component is removed here first.
func redactLocalPath(p string) string {
	if p == "" {
		return p
	}
	var out strings.Builder
	start := 0
	for i := 0; i <= len(p); i++ {
		if i < len(p) && p[i] != '/' && p[i] != '\\' {
			continue
		}
		segment := p[start:i]
		if looksFlattened(segment) {
			out.WriteString(doctor.RedactedPathToken)
		} else {
			out.WriteString(segment)
		}
		if i < len(p) {
			out.WriteByte(p[i])
		}
		start = i + 1
	}
	return doctor.RedactPath(out.String())
}

// looksFlattened reports whether one path component is an absolute path
// that a harness flattened into a directory name: a leading dash is a
// POSIX root ("/Users/…" → "-Users-…"), and a single letter followed by
// two dashes is a Windows drive ("C:\Users\…" → "C--Users-…").
func looksFlattened(segment string) bool {
	if len(segment) > 1 && segment[0] == '-' {
		return true
	}
	return len(segment) > 3 && isASCIILetter(segment[0]) && segment[1] == '-' && segment[2] == '-'
}

func isASCIILetter(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
