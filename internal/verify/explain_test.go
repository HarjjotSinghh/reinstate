package verify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// TestExplainBackendError: the storage vocabulary a refusal arrives in
// means nothing to the reader this report is written for, and the wrapped
// form the backend hands over ("backend: credential rejected
// (InvalidAccessKeyId)") means even less. Every refusal the isolation step
// can meet has to name a cause in ordinary words, and keep the code.
func TestExplainBackendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"unknown key id", &backend.Refusal{Code: "InvalidAccessKeyId", Credential: true},
			"does not recognise this access key id"},
		{"bad signature", &backend.Refusal{Code: "SignatureDoesNotMatch", Credential: true},
			"rejected the request signature"},
		{"expired token", &backend.Refusal{Code: "ExpiredToken", Credential: true},
			"session token has expired"},
		{"invalid token", &backend.Refusal{Code: "InvalidToken", Credential: true},
			"rejected the credential's session token as invalid or withdrawn"},
		{"other credential refusal", &backend.Refusal{Code: "ExpiredTokenException", Credential: true},
			"rejected the credential itself (ExpiredTokenException)"},
		{"access denied", &backend.Refusal{Code: "AccessDenied"},
			"recognised the credential and refused the request anyway (AccessDenied)"},
		{"bare sentinel", backend.ErrCredentialRejected, "rejected the credential itself"},
		{"missing object", backend.ErrNotFound, "no such object"},
		{"nothing to gloss", errors.New("dial tcp: connection refused"), ""},
		{"no error", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := explainBackendError(tc.err)
			if tc.want == "" {
				if got != "" {
					t.Fatalf("explainBackendError(%v) = %q, want no gloss", tc.err, got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("explainBackendError(%v) = %q, want it to contain %q", tc.err, got, tc.want)
			}
			// The gloss never replaces the error: a maintainer reading the
			// same report needs the exact text.
			if joined := withCause(got, tc.err); !strings.Contains(joined, tc.err.Error()) {
				t.Fatalf("withCause dropped the underlying error: %q", joined)
			}
		})
	}
}

// TestExplainDecryptErrorNamesPlaintextRatherThanEmptiness is the one that
// matters most: age reports plaintext of any length as "file is empty",
// because it found no header to parse. Read literally that says the object
// was truncated, which points the reader away from the finding — that
// something wrote readable bytes into their locker.
func TestExplainDecryptErrorNamesPlaintextRatherThanEmptiness(t *testing.T) {
	body := []byte(`{"schema_version":1,"revision":"r1","sessions":{}}`)
	if len(body) < 40 {
		t.Fatalf("the fixture must be long enough that 'empty' is plainly wrong: %d bytes", len(body))
	}
	got := explainDecryptError(StorageHop, body, errors.New("decrypt: failed to read header: parsing age header: file is empty"))
	if !strings.Contains(got, "not an age envelope at all") || !strings.Contains(got, "unencrypted bytes") {
		t.Fatalf("explainDecryptError(plaintext) = %q", got)
	}
	if strings.Contains(got, "empty") {
		t.Fatalf("the gloss repeats age's misleading word: %q", got)
	}
}

func TestExplainDecryptError(t *testing.T) {
	envelope := []byte(ageHeader + "-> X25519 abc\n")
	tests := []struct {
		name    string
		storage string
		body    []byte
		err     error
		want    string
	}{
		{"hop wrong root key", StorageHop, envelope,
			errors.New("decrypt: identity did not match any of the recipients: incorrect identity for recipient block: incorrect passphrase"),
			"the object was sealed to a different root key"},
		{"byo wrong passphrase", StorageBYO, envelope,
			errors.New("decrypt: identity did not match any of the recipients: incorrect identity for recipient block: incorrect passphrase"),
			"sealed under a different passphrase"},
		{"not a recipient", StorageHop, envelope,
			errors.New("decrypt: identity did not match any of the recipients"),
			"not one of the object's recipients"},
		{"altered header", StorageHop, envelope,
			errors.New("decrypt: bad header MAC"),
			"header has been altered"},
		{"anything else", StorageHop, envelope, errors.New("unexpected EOF"),
			"did not open with the key held on this device"},
		{"no error", StorageHop, envelope, nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := explainDecryptError(tc.storage, tc.body, tc.err)
			if got != "" && tc.want == "" || !strings.Contains(got, tc.want) {
				t.Fatalf("explainDecryptError() = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

// TestRedactLocalPath: the report exists to be shown to somebody else, so
// it must not hand them the layout of the machine it ran on. The form that
// matters is the one the agent harnesses actually store — an absolute path
// flattened into a single directory name — which is not absolute, so
// doctor.RedactPath alone leaves it standing.
func TestRedactLocalPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"windows project id", "C--Users-admin-Projects-reinstate", "[REDACTED_PATH]"},
		{"posix project id", "-Users-admin-Projects-reinstate", "[REDACTED_PATH]"},
		{"archive path", "projects/C--Users-admin-Projects-app/session-1.jsonl", "projects/[REDACTED_PATH]/session-1.jsonl"},
		{"posix archive path", "projects/-Users-admin-code-app/session-1.jsonl", "projects/[REDACTED_PATH]/session-1.jsonl"},
		{"windows separators", `projects\C--Users-admin-app\session-1.jsonl`, `projects\[REDACTED_PATH]\session-1.jsonl`},
		{"an absolute path is still removed", "/Users/admin/code/app.jsonl", "[REDACTED_PATH]"},
		{"an ordinary project id survives", "local/locker", "local/locker"},
		{"an ordinary relative path survives", "sessions/session-1.jsonl", "sessions/session-1.jsonl"},
		{"a hyphenated name is not a flattened path", "my-project/session-1.jsonl", "my-project/session-1.jsonl"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := redactLocalPath(tc.in); got != tc.want {
				t.Fatalf("redactLocalPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestDecryptStepNamesPlaintextEndToEnd drives the gloss through a whole
// run: an object replaced with plaintext must read, in step 3, as
// unencrypted bytes rather than as a truncated file.
func TestDecryptStepNamesPlaintextEndToEnd(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	put(t, store, "manifest.age", []byte(`{"schema_version":1,"revision":"r1","sessions":{}}`))
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	step := r.Steps[2]
	if step.Status != Fail || !strings.Contains(step.Observed, "not an age envelope at all") {
		t.Fatalf("decrypt step %+v", step)
	}
	if !strings.Contains(step.Observed, "file is empty") {
		t.Fatalf("the underlying error was dropped: %q", step.Observed)
	}
}

// TestAccessKeyIDIsGlossedWhereverItIsPrinted: the id is in the report so
// a reader can see steps 1 and 4 name the same credential, but a reader
// who has never signed an S3 request cannot know that half of a credential
// pair is a public identifier — and a report that looks like it leaks a
// secret is not one anyone will show to a third party.
func TestAccessKeyIDIsGlossedWhereverItIsPrinted(t *testing.T) {
	keys := rootKeys(t)
	store := lockerWith(t, keys, "")
	r := Run(context.Background(), Options{Backend: store, Keys: keys, Storage: StorageHop,
		Locker:        LockerInfo{Endpoint: "https://s3.example", Bucket: "lk-1"},
		Reference:     &hop.Reference{Endpoint: "https://s3.example", Bucket: "lk-ref", Key: "reference/probe.txt"},
		OpenReference: openRef(denied, "AKIAHOP1"), CredentialID: credential("AKIAHOP1")})
	for _, i := range []int{0, 3} {
		detail := strings.Join(r.Steps[i].Detail, "\n")
		if !strings.Contains(detail, "AKIAHOP1") {
			t.Fatalf("step %d lost the access key id: %q", i+1, detail)
		}
		for _, want := range []string{"an access key id is a public identifier", "the secret key and session token are never printed"} {
			if !strings.Contains(detail, want) {
				t.Fatalf("step %d prints the access key id without saying %q: %q", i+1, want, detail)
			}
		}
	}
}
