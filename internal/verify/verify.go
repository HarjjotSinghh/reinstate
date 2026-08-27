// Package verify runs the checks behind `rein sync verify`: it lists the
// locker with this device's credentials, fetches one object and shows it is
// ciphertext, decrypts it locally, and proves that the same credentials are
// refused from a bucket the operator owns (the reference locker). The
// result is a verification report a non-expert can read and reproduce step
// by step; the upload form carries step results only.
package verify

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// ReportVersion is the report shape the control plane accepts.
const ReportVersion = 1

// Status of one step.
type Status string

// Step statuses.
const (
	Pass          Status = "pass"
	Fail          Status = "fail"
	NotApplicable Status = "not-applicable"
)

// Storage kinds named in a report.
const (
	StorageHop = "hop"
	StorageBYO = "byo"
)

// Step ids, in the order they run.
const (
	StepList       = "list"
	StepCiphertext = "ciphertext"
	StepDecrypt    = "decrypt"
	StepIsolation  = "isolation"
)

// Object names the checks look for; snapshots live under snapshots/.
const (
	manifestObject = "manifest.age"
	snapshotPrefix = "snapshots/"
	ageHeader      = "age-encryption.org/v1\n"
	maxObjectBytes = 64 << 20
	maxHeaderBytes = 4096
)

// accessKeyIDNote follows every access key id the report prints. The id
// is there so a reader can see that the credential step 1's locker
// accepted is the one step 4's reference locker refused, which is the
// whole of what step 4 proves — but a reader who has never signed an S3
// request has no way to know that half of a credential pair is a public
// identifier, and a report that looks like it is leaking a secret is not
// a report anyone will show to a third party.
const accessKeyIDNote = "(shown so steps 1 and 4 can be seen to name the same credential; an access key id is a public identifier, and the secret key and session token are never printed)"

// plaintextMarkers are field names every manifest and snapshot envelope
// contains in the clear before encryption. Finding one in an object body
// means the object is not ciphertext.
var plaintextMarkers = []string{`"schema_version"`, `"sessions"`, `"snapshot_id"`, `"session_id"`, `"revision"`, `"files"`}

// Step is one check: what was done, what was observed, and the verdict.
// Detail lines are shown locally only and may name sessions; they are
// never part of the upload form.
type Step struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Did      string   `json:"did"`
	Observed string   `json:"observed"`
	Status   Status   `json:"status"`
	Detail   []string `json:"detail,omitempty"`
}

// Report is the verification report as printed and as `--json` emits it.
//
// Summary, CheckedObjects and Unopened are serialised because a consumer
// that decodes the document has to be able to rebuild the same sentence
// the human output ends with. Without them a decoder sees `outcome: pass`
// and nothing else, and the honest qualifications the human text was
// written to carry — which objects were opened, which were judged by
// name — are lost exactly where an over-claim is easiest to make.
type Report struct {
	Version       int    `json:"version"`
	GeneratedAt   string `json:"generated_at"`
	ClientVersion string `json:"client_version"`
	Storage       string `json:"storage"`
	Outcome       Status `json:"outcome"`
	Steps         []Step `json:"steps"`
	// Locker names what was checked; shown locally, never uploaded.
	Locker LockerInfo `json:"locker"`
	// Summary is the outcome sentence, the one line the whole report comes
	// down to. It is the text after "OUTCOME: " in the human output.
	Summary string `json:"summary"`
	// CheckedObjects names what step 2 actually fetched ("the index", "the
	// newest snapshot in the index"), so nothing claims an object no step
	// opened.
	CheckedObjects []string `json:"checked_objects,omitempty"`
	// Unopened describes the objects step 1 saw and steps 2–3 did not
	// fetch, so a passing report never reads as "everything is verified".
	Unopened string `json:"unopened,omitempty"`
	// plaintext records that step 2 found an object that is not an age
	// envelope, or an envelope with plaintext in its body. It decides
	// whether the summary asks for a security report.
	plaintext bool
	// foreignBucket records that step 4 reached a bucket that is not this
	// account's. It decides the same thing.
	foreignBucket bool
	// wrongKey records that every object step 2 proved to be ciphertext
	// failed to open with the key held here — the mistyped-passphrase
	// shape, not a security incident.
	wrongKey bool
	// unreached is why the storage endpoint gave no answer, when steps 1
	// or 2 got none: a timeout, a dropped connection, a name that did not
	// resolve. It is what separates "nothing could be checked" from
	// "nothing has been pushed yet", which are the two ways this report
	// ends up with no verdict at all.
	unreached string
}

// LockerInfo names the bucket the checks ran against.
type LockerInfo struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

// Upload is the report as posted to the control plane: step results only.
type Upload struct {
	Version       int          `json:"version"`
	GeneratedAt   string       `json:"generated_at"`
	ClientVersion string       `json:"client_version"`
	Storage       string       `json:"storage"`
	Outcome       Status       `json:"outcome"`
	Steps         []UploadStep `json:"steps"`
}

// UploadStep is a Step without its local detail.
type UploadStep struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Did      string `json:"did"`
	Observed string `json:"observed"`
	Status   Status `json:"status"`
}

// ForUpload strips everything but the step results.
func (r *Report) ForUpload() Upload {
	u := Upload{Version: r.Version, GeneratedAt: r.GeneratedAt, ClientVersion: r.ClientVersion, Storage: r.Storage, Outcome: r.Outcome}
	for _, s := range r.Steps {
		u.Steps = append(u.Steps, UploadStep{ID: s.ID, Name: s.Name, Did: s.Did, Observed: s.Observed, Status: s.Status})
	}
	return u
}

// Passed reports whether every step that could run passed.
func (r *Report) Passed() bool { return r.Outcome == Pass }

// Failed reports whether a step did not hold. It is distinct from
// !Passed(): a run with nothing to check yet is neither.
func (r *Report) Failed() bool { return r.Outcome == Fail }

// CheckedPhrase names, as one phrase ("the index and the newest snapshot
// in the index"), the objects step 2 actually fetched, so sentences
// outside the report claim only those.
func (r *Report) CheckedPhrase() string { return strings.Join(r.CheckedObjects, " and ") }

// Codec decrypts an envelope; it matches sync.EnvelopeCodec's read half so
// the CLI can pass the engine's codec through.
type Codec interface {
	DecryptReader(source io.Reader, keys crypto.KeyProvider) (io.Reader, error)
}

type ageCodec struct{}

func (ageCodec) DecryptReader(source io.Reader, keys crypto.KeyProvider) (io.Reader, error) {
	return crypto.OpenReader(source, keys)
}

// Options configure one run.
type Options struct {
	// Backend is the locker opened with this device's credentials.
	Backend backend.Backend
	// Prefix is the engine-side key prefix (empty when the client scopes keys).
	Prefix string
	// Keys decrypts objects; the device's root key or the passphrase.
	Keys crypto.KeyProvider
	// Codec overrides age decryption in tests; nil means age.
	Codec Codec
	// Storage is StorageHop or StorageBYO.
	Storage string
	// Locker names what is being checked (shown locally only).
	Locker LockerInfo
	// Reference is where the control plane's probe lives; nil for BYO
	// storage or a control plane without one (ReferenceErr then says why).
	Reference    *hop.Reference
	ReferenceErr error
	// OpenReference opens the reference locker with this device's locker
	// credentials. Required when Reference is set. The client it returns
	// must refuse redirects and record its exchanges (see ProbeClient), so
	// the isolation step can pin its verdict to the response rather than
	// to the endpoint the control plane named.
	OpenReference func(ctx context.Context, ref hop.Reference) (Probe, error)
	// CredentialID returns the access key id Backend is signing with right
	// now; optional. It is recorded in step 1 so the report shows the
	// credential the locker accepted is the one the reference refused.
	CredentialID func(ctx context.Context) (string, error)
	// ClientVersion is recorded in the report.
	ClientVersion string
	// Now is injectable for golden output.
	Now func() time.Time
}

// Probe is the reference locker opened with this device's locker
// credentials, together with the record of what its transport saw. The
// isolation step needs both: a backend to make the request with, and a
// record proving the request carried the credential to the host the
// control plane pinned and came back as a signed S3 refusal.
type Probe struct {
	// Backend talks to the reference locker.
	Backend backend.Backend
	// AccessKeyID is the access key id Backend signs with; empty when the
	// client has no notion of one.
	AccessKeyID string
	// Exchanges returns, in order, what the transport observed for every
	// request Backend has made so far. A nil Exchanges means the client
	// was not instrumented: the step then has nothing to pin its verdict
	// to and reports not-applicable rather than passing.
	Exchanges func() []Exchange
}

// Run executes the checks and returns the report. It never returns an
// error: a check that cannot run is a failed or not-applicable step.
func Run(ctx context.Context, o Options) *Report {
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	if o.Codec == nil {
		o.Codec = ageCodec{}
	}
	r := &Report{Version: ReportVersion, GeneratedAt: now().UTC().Format(time.RFC3339), ClientVersion: o.ClientVersion, Storage: o.Storage, Locker: o.Locker}

	inv, akid, listed := listStep(ctx, o, r)
	raw := ciphertextStep(ctx, o, r, inv)
	r.Unopened = describeUnopened(inv, raw)
	r.CheckedObjects = describeChecked(raw)
	decryptStep(ctx, o, r, inv, raw)
	isolationStep(ctx, o, r, akid, listed)

	r.Outcome = outcomeOf(r.Steps)
	// A run that listed the locker but opened nothing has shown nothing
	// about what the locker holds, and "pass" is the one word it must not
	// be reduced to. This is only reachable when a fetch got no answer at
	// all — a locker with an index always has an object to fetch — so the
	// report says so instead of passing on the strength of step 1.
	if r.Outcome == Pass && len(r.CheckedObjects) == 0 {
		r.Outcome = NotApplicable
	}
	r.Summary = r.buildSummary()
	return r
}

// NotRun returns the report for a run that never started because the
// control plane could not be reached. On a Hop locker the credentials the
// first three checks need are minted by the control plane, so an outage
// stops all four; a reader still deserves to see which four, and to be
// told the difference between a service being down and a claim not
// holding. cause is the transport failure, kept verbatim.
func NotRun(o Options, cause error) *Report {
	because := withReason("the control plane could not be reached", cause)
	return notStarted(o, because,
		"Could not run: "+because+", and a Hop locker is listed with credentials the control plane mints for this device, so there were no credentials to list it with.",
		"Could not run: "+because+", so it could not say where its reference locker is.",
		"Run rein sync verify again when the control plane is reachable.")
}

// NotReached returns the report for a run that never started because the
// storage endpoint gave no answer — a request that timed out, a
// connection refused, a name that did not resolve. It is the storage half
// of NotRun, and it exists for the same reason: the profile could not
// even be opened, and the reader has to be able to tell an outage from a
// finding. Use Unreachable to decide whether an error belongs here.
func NotReached(o Options, cause error) *Report {
	because := withReason("the storage endpoint gave no answer", cause)
	return notStarted(o, because,
		"Could not run: "+because+", so the locker could not be opened or listed.",
		"Could not run: step 1 could not list this account's locker, so a refusal from another bucket would show nothing about bucket scope.",
		"Run rein sync verify again when the storage endpoint answers.")
}

// withReason appends the underlying failure, kept verbatim, to the plain
// sentence a reader acts on.
func withReason(sentence string, cause error) string {
	if cause == nil {
		return sentence
	}
	return sentence + " (" + cause.Error() + ")"
}

// notStarted builds the four not-applicable steps for a run that never
// began, so the reader still sees which four checks exist and which of
// them said nothing. It is never a failure and never a pass.
func notStarted(o Options, because, listed, isolation, retry string) *Report {
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	r := &Report{Version: ReportVersion, GeneratedAt: now().UTC().Format(time.RFC3339), ClientVersion: o.ClientVersion, Storage: o.Storage, Outcome: NotApplicable, Locker: o.Locker, unreached: because}
	for _, s := range []struct{ id, name, observed string }{
		{StepList, "List the locker with this device's credentials", listed},
		{StepCiphertext, "Fetch an object and check it is ciphertext",
			"Could not run: no object could be fetched, because the locker could not be opened (see step 1)."},
		{StepDecrypt, "Decrypt the object locally",
			"Could not run: no object was fetched (see step 2). The key this step would have used never leaves this device and was never involved."},
		{StepIsolation, "Prove this account's credentials are refused from another bucket", isolation},
	} {
		r.Steps = append(r.Steps, Step{ID: s.id, Name: s.name, Did: "Nothing: the check did not start.", Observed: s.observed, Status: NotApplicable})
	}
	r.Summary = "NOT VERIFIED. Nothing was checked, because " + because + ". No step failed and nothing here says anything about what the locker holds. " + retry
	return r
}

// Unreachable reports an error that is a failure to reach an endpoint at
// all, rather than anything the endpoint said. A dial that was refused, a
// name that did not resolve, a handshake that never finished, a request
// that timed out: none of them says a word about the locker, and the
// command that establishes trust must not present one as a finding.
//
// It is deliberately a positive test rather than "not a refusal": the
// caller uses it on errors from opening a whole profile, where a bad
// config file and an unreachable bucket both arrive as errors and only
// one of them is an outage.
func Unreachable(err error) bool {
	if err == nil || answered(err) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}

// outcomeOf folds the step verdicts into the report's own. A single
// failed step fails the report; a run where no step could reach a verdict
// at all — a profile that has never pushed — is neither a pass nor a
// failure, and saying "pass" there would be the report's largest possible
// over-claim.
func outcomeOf(steps []Step) Status {
	ran := false
	for _, s := range steps {
		switch s.Status {
		case Fail:
			return Fail
		case NotApplicable:
		default:
			ran = true
		}
	}
	if !ran {
		return NotApplicable
	}
	return Pass
}

// describeUnopened says what step 1 listed that no later step fetched, so
// the summary does not call objects ciphertext that were judged by name.
// The snapshots are counted against what step 2 actually read rather than
// assumed to be "all but one": a fetch that failed leaves nothing opened.
func describeUnopened(inv *inventory, got []fetched) string {
	if inv == nil {
		return ""
	}
	opened := map[string]bool{}
	for _, f := range got {
		opened[f.name] = true
	}
	var parts []string
	n := 0
	for _, id := range inv.snapshots {
		if !opened[snapshotPrefix+id+".age"] {
			n++
		}
	}
	if n > 0 {
		parts = append(parts, fmt.Sprintf("%d other age-named snapshot(s)", n))
	}
	if inv.keyring {
		parts = append(parts, "the wrapped keyring")
	}
	if n := len(inv.other); n > 0 {
		parts = append(parts, fmt.Sprintf("%d unrecognised object(s)", n))
	}
	if len(parts) == 0 {
		return "No other object is in the locker."
	}
	return "Not opened and judged by name only: " + strings.Join(parts, ", ") + "."
}

// describeChecked names the objects step 2 fetched, in order, so the
// summary never claims an object that was not fetched (a manifest-only
// locker fetches only the index). The label is the one step 2 earned: a
// snapshot is only called the newest when the index said which one that
// is.
func describeChecked(got []fetched) []string {
	var names []string
	for _, f := range got {
		names = append(names, f.label)
	}
	return names
}

// inventory is what the listing found.
type inventory struct {
	manifest  bool
	keyring   bool
	snapshots []string
	other     []string
	total     int
}

func key(prefix, relative string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return relative
	}
	return prefix + "/" + relative
}

// listStep lists the locker and returns what it found, the access key id
// the listing was signed with (empty when the backend has no notion of
// one, such as BYO keys from the SDK chain or the memory backend), and the
// step's own verdict, which step 4 needs: a credential no locker was shown
// to accept proves nothing when another bucket refuses it.
func listStep(ctx context.Context, o Options, r *Report) (*inventory, string, Status) {
	step := Step{ID: StepList, Name: "List the locker with this device's credentials",
		Did: "Asked the storage endpoint for every object under this account's prefix (following every listing page), signed with the credentials this device uses to push."}
	objects, err := o.Backend.List(ctx, strings.Trim(o.Prefix, "/"))
	akid := ""
	if o.CredentialID != nil {
		if id, idErr := o.CredentialID(ctx); idErr == nil && id != "" {
			akid = id
			step.Detail = append(step.Detail, "signed with access key id "+akid+" "+accessKeyIDNote)
		}
	}
	if err != nil {
		// A listing the endpoint refused is an answer, and the answer
		// contradicts the claim that this device's credentials open this
		// locker. A listing that got no answer — a timeout, a dropped
		// connection, a name that did not resolve — is not an answer about
		// anything, and reporting it as a failed check would tell the
		// reader their locker failed a security check because their
		// network dropped.
		if !answered(err) {
			step.Status = NotApplicable
			step.Observed = "Could not run: the storage endpoint gave no answer to the listing — " + withCause(explainBackendError(err), err) + " — so nothing about this locker was shown either way."
			r.unreached = "The locker could not be listed: " + withCause(explainBackendError(err), err) + "."
		} else {
			step.Status = Fail
			step.Observed = "The listing was refused or failed: " + withCause(explainBackendError(err), err) + "."
		}
		r.Steps = append(r.Steps, step)
		return nil, akid, step.Status
	}
	inv := &inventory{total: len(objects)}
	for _, obj := range objects {
		k := strings.TrimPrefix(obj.Key, strings.Trim(o.Prefix, "/")+"/")
		switch {
		case k == manifestObject:
			inv.manifest = true
		case k == keyring.ObjectName:
			inv.keyring = true
		case strings.HasPrefix(k, snapshotPrefix) && strings.HasSuffix(k, ".age"):
			inv.snapshots = append(inv.snapshots, strings.TrimSuffix(strings.TrimPrefix(k, snapshotPrefix), ".age"))
		default:
			inv.other = append(inv.other, k)
		}
	}
	sort.Strings(inv.snapshots)
	sort.Strings(inv.other)
	var parts []string
	if inv.manifest {
		parts = append(parts, manifestObject+" (the encrypted index)")
	}
	if inv.keyring {
		parts = append(parts, keyring.ObjectName+" (the root key, wrapped per device; holds no usable key on its own)")
	}
	if n := len(inv.snapshots); n > 0 {
		parts = append(parts, fmt.Sprintf("%d snapshot(s) under %s named by opaque ids", n, snapshotPrefix))
	}
	if n := len(inv.other); n > 0 {
		parts = append(parts, fmt.Sprintf("%d other object(s)", n))
	}
	if !inv.manifest {
		// Nothing to check is not a failed check. A device that has not
		// pushed yet is the ordinary state of a new install, and reporting
		// it as a failure is how the command ended up telling first-time
		// users to report a security incident.
		step.Status = NotApplicable
		step.Observed = fmt.Sprintf("%d object(s) under this prefix, and no %s: nothing has been pushed from this profile yet, so there is nothing to check. Run rein push first, then rein sync verify again.", inv.total, manifestObject)
	} else {
		step.Status = Pass
		step.Observed = fmt.Sprintf("%d object(s): %s.", inv.total, strings.Join(parts, "; "))
	}
	for _, id := range inv.snapshots {
		step.Detail = append(step.Detail, snapshotPrefix+id+".age")
	}
	for _, k := range inv.other {
		step.Detail = append(step.Detail, k)
	}
	r.Steps = append(r.Steps, step)
	return inv, akid, step.Status
}

// fetched is one object body the ciphertext step read. label is how the
// report names it to a reader ("the index", "the newest snapshot in the
// index"); it is what the summary is allowed to claim.
type fetched struct {
	name  string
	label string
	body  []byte
}

func ciphertextStep(ctx context.Context, o Options, r *Report, inv *inventory) []fetched {
	step := Step{ID: StepCiphertext, Name: "Fetch an object and check it is ciphertext",
		Did: "Downloaded the index object and, when the index names one, the snapshot it records as updated last, and looked at the raw bytes for the age encryption header and for any field name that appears in the plaintext."}
	// A step that never ran is not a step that failed. Which of the two it
	// was is the difference between "you have not pushed yet" and "your
	// locker is not what we said it is", and the report has to keep them
	// apart: step 1 has already said which, so this step points at it.
	if inv == nil {
		step.Status = NotApplicable
		step.Observed = "Could not run: the locker could not be listed, so there was nothing to fetch (see step 1)."
		r.Steps = append(r.Steps, step)
		return nil
	}
	if !inv.manifest {
		step.Status = NotApplicable
		step.Observed = "Not run: the locker holds no index yet, so there is no object to fetch (see step 1)."
		r.Steps = append(r.Steps, step)
		return nil
	}
	var got []fetched
	var observed []string
	step.Status = Pass
	read := func(name, label string) []byte {
		body, err := fetch(ctx, o.Backend, key(o.Prefix, name))
		if err != nil {
			// The same line step 1 draws: a refusal is an answer and
			// contradicts the claim; a request that got no answer at all
			// shows nothing about the object and must not be reported as a
			// check that failed.
			if !answered(err) {
				degrade(&step, NotApplicable)
				observed = append(observed, fmt.Sprintf("Could not run: the storage endpoint gave no answer for %s — %s — so nothing about that object was shown.", name, withCause(explainBackendError(err), err)))
				if r.unreached == "" {
					r.unreached = fmt.Sprintf("%s could not be fetched: %s.", name, withCause(explainBackendError(err), err))
				}
				return nil
			}
			degrade(&step, Fail)
			observed = append(observed, fmt.Sprintf("%s could not be fetched: %s.", name, withCause(explainBackendError(err), err)))
			return nil
		}
		got = append(got, fetched{name: name, label: label, body: body})
		verdict, ok := inspectCiphertext(body)
		if !ok {
			degrade(&step, Fail)
			r.plaintext = true
		}
		observed = append(observed, fmt.Sprintf("%s (%d bytes): %s", name, len(body), verdict))
		if head, _, found := bytes.Cut(body, []byte("\n")); found && len(head) < 200 {
			step.Detail = append(step.Detail, name+" first line: "+string(head))
		}
		return body
	}
	// The index is read first because it is the only thing that knows which
	// snapshot is the newest one; the ids themselves are random.
	manifest := read(manifestObject, "the index")
	if len(inv.snapshots) > 0 {
		id, fromIndex := newestSnapshot(o, inv, manifest)
		label := "one snapshot"
		if fromIndex {
			label = "the newest snapshot in the index"
		}
		read(snapshotPrefix+id+".age", label)
	}
	step.Observed = strings.Join(observed, " ")
	r.Steps = append(r.Steps, step)
	return got
}

// newestSnapshot names the snapshot step 2 fetches. Snapshot ids are random
// uuids, so sort order says nothing about age; the index does, because it
// records when each session was last updated. The index is opened here with
// the key held on this device — the same open step 3 reports on — and the
// snapshot its newest entry points at is the one fetched.
//
// It reports false when the index cannot be opened or read here, or names
// none of the snapshots the listing found. The last id in sort order is
// then used and the report calls it "one snapshot" rather than the newest,
// because nothing observed its age.
func newestSnapshot(o Options, inv *inventory, manifest []byte) (string, bool) {
	fallback := inv.snapshots[len(inv.snapshots)-1]
	if len(manifest) == 0 || o.Keys == nil || o.Codec == nil {
		return fallback, false
	}
	plain, err := o.Codec.DecryptReader(bytes.NewReader(manifest), o.Keys)
	if err != nil {
		return fallback, false
	}
	raw, err := io.ReadAll(io.LimitReader(plain, maxObjectBytes))
	if err != nil {
		return fallback, false
	}
	var m schema.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return fallback, false
	}
	listed := map[string]bool{}
	for _, id := range inv.snapshots {
		listed[id] = true
	}
	keys := make([]string, 0, len(m.Sessions))
	for k := range m.Sessions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best, bestAt := "", time.Time{}
	// Entries are walked in key order and the first of any tie wins, so two
	// runs against the same index fetch the same object.
	for _, k := range keys {
		s := m.Sessions[k]
		if !listed[s.SnapshotID] {
			continue
		}
		at, err := time.Parse(time.RFC3339, s.UpdatedAt)
		if err != nil {
			continue
		}
		if best == "" || at.After(bestAt) {
			best, bestAt = s.SnapshotID, at
		}
	}
	if best == "" {
		return fallback, false
	}
	return best, true
}

// inspectCiphertext says what the bytes look like and whether they pass.
func inspectCiphertext(body []byte) (string, bool) {
	if !bytes.HasPrefix(body, []byte(ageHeader)) {
		return "does NOT begin with the age v1 header; this object is not an age envelope.", false
	}
	head := body
	if len(head) > maxHeaderBytes {
		head = head[:maxHeaderBytes]
	}
	recipient := "unknown"
	switch {
	case bytes.Contains(head, []byte("-> X25519 ")):
		recipient = "X25519 (root key)"
	case bytes.Contains(head, []byte("-> scrypt ")):
		recipient = "scrypt (passphrase)"
	}
	for _, m := range plaintextMarkers {
		if bytes.Contains(body, []byte(m)) {
			return fmt.Sprintf("begins with the age v1 header (recipient %s) but the plaintext field %s appears in the body; this object is NOT fully encrypted.", recipient, m), false
		}
	}
	return fmt.Sprintf("begins with the age v1 header (recipient %s); no plaintext field name appears anywhere in the body.", recipient), true
}

func fetch(ctx context.Context, b backend.Backend, k string) ([]byte, error) {
	rc, _, err := b.Get(ctx, k)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	body, err := io.ReadAll(io.LimitReader(rc, maxObjectBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxObjectBytes {
		return nil, errors.New("object larger than the 64 MiB the check reads")
	}
	return body, nil
}

func decryptStep(_ context.Context, o Options, r *Report, inv *inventory, got []fetched) {
	step := Step{ID: StepDecrypt, Name: "Decrypt the object locally",
		Did: "Opened the same bytes on this device with the key held here (the root key for Hop, the passphrase for BYO storage) and read what they contain. Nothing was sent anywhere."}
	if inv == nil || len(got) == 0 {
		step.Status = NotApplicable
		step.Observed = "Not run: no object was fetched (see step 2)."
		r.Steps = append(r.Steps, step)
		return
	}
	if o.Keys == nil {
		step.Status = Fail
		step.Observed = "No key is available on this device to decrypt with."
		r.Steps = append(r.Steps, step)
		return
	}
	step.Status = Pass
	opened, refused := 0, 0
	var observed []string
	for _, f := range got {
		plain, err := o.Codec.DecryptReader(bytes.NewReader(f.body), o.Keys)
		if err != nil {
			step.Status = Fail
			refused++
			observed = append(observed, fmt.Sprintf("%s did not decrypt: %s.", f.name, withCause(explainDecryptError(o.Storage, f.body, err), err)))
			continue
		}
		opened++
		if f.name == manifestObject {
			summary, detail, err := describeManifest(plain)
			if err != nil {
				step.Status = Fail
				observed = append(observed, fmt.Sprintf("%s did not decrypt cleanly: %v.", f.name, err))
				continue
			}
			observed = append(observed, summary)
			step.Detail = append(step.Detail, detail...)
			continue
		}
		summary, detail, err := describeSnapshot(f.name, plain)
		if err != nil {
			step.Status = Fail
			observed = append(observed, fmt.Sprintf("%s did not decrypt cleanly: %v.", f.name, err))
			continue
		}
		observed = append(observed, summary)
		step.Detail = append(step.Detail, detail...)
	}
	// Every envelope step 2 proved to be ciphertext refused this device's
	// key, and none opened. That is the shape of a key that does not
	// belong to these objects — a second passphrase, a device enrolled
	// against another account — not the shape of an operator holding
	// plaintext, and the summary must not confuse the two.
	r.wrongKey = refused > 0 && opened == 0
	step.Observed = strings.Join(observed, " ")
	r.Steps = append(r.Steps, step)
}

func describeManifest(plain io.Reader) (string, []string, error) {
	raw, err := io.ReadAll(io.LimitReader(plain, maxObjectBytes))
	if err != nil {
		return "", nil, err
	}
	var m schema.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", nil, fmt.Errorf("the plaintext is not a manifest: %w", err)
	}
	agents := map[string]int{}
	for _, s := range m.Sessions {
		agents[s.Agent]++
	}
	names := make([]string, 0, len(agents))
	for a := range agents {
		names = append(names, a)
	}
	sort.Strings(names)
	var byAgent []string
	for _, a := range names {
		byAgent = append(byAgent, fmt.Sprintf("%s %d", a, agents[a]))
	}
	// The summary is uploaded with the report, so it names only the
	// object and the schema; the revision, counts and agent mix are
	// local detail.
	summary := fmt.Sprintf("%s decrypted into a schema v%d index.", manifestObject, m.SchemaVersion)
	detail := []string{fmt.Sprintf("index revision %s, %d session(s)", orNone(m.Revision), len(m.Sessions))}
	if len(byAgent) > 0 {
		detail[0] += " (" + strings.Join(byAgent, ", ") + ")"
	}
	keys := make([]string, 0, len(m.Sessions))
	for k := range m.Sessions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s := m.Sessions[k]
		detail = append(detail, fmt.Sprintf("index entry %s -> %s%s.age (updated %s)", k, snapshotPrefix, s.SnapshotID, s.UpdatedAt))
	}
	return summary, detail, nil
}

func describeSnapshot(name string, plain io.Reader) (string, []string, error) {
	reader := bufio.NewReaderSize(plain, 1<<20)
	line, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, err
	}
	var env schema.Envelope
	if err := json.Unmarshal(bytes.TrimSuffix(line, []byte{'\n'}), &env); err != nil {
		return "", nil, fmt.Errorf("the plaintext does not start with a snapshot envelope: %w", err)
	}
	if len(env.Files) == 0 {
		return "", nil, errors.New("the snapshot envelope lists no file")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(reader, maxObjectBytes))
	if err != nil {
		return "", nil, err
	}
	file := env.Files[0]
	sum := hex.EncodeToString(h.Sum(nil))
	if n != file.Size || sum != file.SHA256 {
		return "", nil, fmt.Errorf("the payload (%d bytes, sha256 %s…) does not match the envelope (%d bytes, sha256 %s…)", n, sum[:12], file.Size, short(file.SHA256))
	}
	// The summary is uploaded with the report; the agent, session, size
	// and file are local detail. The project id and the archive path are
	// redacted even so: this report is written to be handed to somebody
	// else, and both carry the local project directory, which the agent
	// harnesses store as a flattened absolute path.
	summary := fmt.Sprintf("%s decrypted into a snapshot envelope whose payload sha256 matches the envelope.", name)
	detail := []string{fmt.Sprintf("%s: agent %s, session %s, project %s, created %s on %s, %d file(s), payload %d bytes, file %s",
		name, env.Agent, env.SessionID, redactLocalPath(env.ProjectID), env.CreatedAt, env.SourcePlatform, len(env.Files), n, redactLocalPath(file.Path))}
	return summary, detail, nil
}

// isolationStep asks whether this account's credentials are refused by a
// bucket that is not this account's.
//
// It reports Fail only for something it observed that contradicts the
// claim: a reference locker that accepted the credential, a request that
// landed somewhere other than the endpoint step 1 listed, a redirect
// offered in place of an answer, or a control plane asking for the
// credential to be sent over an unencrypted connection to anything but a
// loopback address (see loopback, and every page that documents this
// refusal states that exemption too). Everything else
// that stops the check — an outage, a control plane that answered an
// error, a reference bucket that has been deleted, a dropped connection,
// a credential that died between step 1 and here — is a check that could
// not run, and is reported not-applicable with its reason. The two are
// not the same thing and must not read as the same thing: a customer told
// `OUTCOME: FAIL` by their trust-establishing command because the
// operator misconfigured one row has been told a falsehood about their
// own locker, and the alarm that matters is the one they will not believe
// the second time.
func isolationStep(ctx context.Context, o Options, r *Report, lockerAKID string, listed Status) {
	step := Step{ID: StepIsolation, Name: "Prove this account's credentials are refused from another bucket",
		Did: "Asked the control plane for its reference locker (a bucket the operator owns, holding one probe object), checked that it names a different bucket at the same storage endpoint step 1 listed and that reaching it would not put this device's credentials on the wire unencrypted, then tried to list it and read the probe with the same credentials that just listed this account's locker, over a client that refuses to follow a redirect anywhere else."}
	switch {
	case o.Storage == StorageBYO:
		// "What was done" has to be what was done on the profile the
		// reader is on. The sentence above describes asking a control
		// plane; this profile has none, and printing it here would tell a
		// BYO reader about requests that were never made.
		step.Did = "Nothing. This profile stores to a bucket you configured yourself, so there is no control plane to name a reference locker and no operator-owned bucket to offer these credentials to. No request was made."
		step.Status = NotApplicable
		step.Observed = "Not applicable: BYO storage has no control plane and no reference locker. Whether your credentials reach other buckets is decided by your own bucket policy."
		r.Steps = append(r.Steps, step)
		return
	case o.Reference == nil && errors.Is(o.ReferenceErr, hop.ErrNoReference):
		step.Status = NotApplicable
		step.Observed = "Not applicable: " + o.ReferenceErr.Error() + "."
		r.Steps = append(r.Steps, step)
		return
	case o.Reference == nil && hop.Unreachable(o.ReferenceErr):
		// A control plane nobody could reach is a check that did not run,
		// not a check that failed. It says nothing either way about where
		// this account's credentials reach, and the summary says so.
		step.Status = NotApplicable
		step.Observed = "Could not run: the control plane could not be reached, so it could not say where its reference locker is and nothing about bucket scope was shown (" + o.ReferenceErr.Error() + "). " + stepsStandAlone
		r.Steps = append(r.Steps, step)
		return
	case o.Reference == nil:
		// The control plane answered, and what it answered was not a
		// reference locker: an error status, or a row missing the bucket or
		// the probe key. That is a fault on the operator's side of the
		// service and says nothing about this account's credentials, so it
		// is a check that could not run.
		step.Status = NotApplicable
		step.Observed = "Could not run: the control plane did not say where its reference locker is"
		if o.ReferenceErr != nil {
			step.Observed += " (" + o.ReferenceErr.Error() + ")"
		}
		step.Observed += ", so nothing about bucket scope was shown. That is a fault on the control plane's side, not a finding about this locker. " + stepsStandAlone
		r.Steps = append(r.Steps, step)
		return
	case o.OpenReference == nil:
		step.Status = NotApplicable
		step.Observed = "Could not run: no way to open the reference locker was configured on this device, so nothing was asked of it."
		r.Steps = append(r.Steps, step)
		return
	case listed == NotApplicable:
		// Step 1 found nothing to check. There is a locker and it accepted
		// the credentials, but with no push behind it this step would be
		// checking the scope of a credential on an empty bucket, which is
		// not what the report is for.
		step.Status = NotApplicable
		step.Observed = "Not applicable: nothing has been pushed from this profile yet (see step 1), so there is nothing to verify. Run rein push, then rein sync verify again."
		r.Steps = append(r.Steps, step)
		return
	case listed != Pass:
		// A refusal is only evidence of bucket scope when the credential
		// refused is one some locker was just shown to accept. Step 1 did
		// not show that, so a 403 here would be the answer every host gives
		// a credential it does not know.
		step.Status = NotApplicable
		step.Observed = "Not applicable: step 1 did not list this account's locker, so no locker was shown to accept these credentials and a refusal here would show nothing about bucket scope. Fix step 1 and verify again."
		r.Steps = append(r.Steps, step)
		return
	}
	ref := *o.Reference
	step.Detail = append(step.Detail, fmt.Sprintf("reference locker %s at %s, probe %s", ref.Bucket, ref.Endpoint, ref.Key))
	locker, lockerOK := parseEndpoint(o.Locker.Endpoint)
	if !lockerOK {
		// Without the locker's own endpoint there is nothing to pin the
		// reference against, and an unpinned refusal is worth nothing: any
		// host answers a foreign credential with 403. Unverifiable is
		// not-applicable, never a pass.
		step.Status = NotApplicable
		step.Observed = "Could not run: the storage endpoint this account's locker was listed at is not known here" + orInvalid(o.Locker.Endpoint) + ", so the reference locker cannot be pinned to it and a refusal from it would show nothing about bucket scope."
		r.Steps = append(r.Steps, step)
		return
	}
	if o.Locker.Bucket == "" {
		// The step's whole sentence is "a bucket that is not this
		// account's". With no name for this account's bucket, nothing here
		// can tell the two apart.
		step.Status = NotApplicable
		step.Observed = "Could not run: the bucket step 1 listed is not known here, so nothing could show that the reference locker is a different bucket from this account's."
		r.Steps = append(r.Steps, step)
		return
	}
	if sameBucket(ref.Bucket, o.Locker.Bucket) {
		// A reference locker that is this account's own bucket tests
		// nothing: these credentials are meant to reach it, so it answers
		// them, and the answer would be reported below as credentials
		// reaching a bucket that is not their own — which would be exactly
		// backwards. Refuse the probe and say why.
		step.Status = NotApplicable
		step.Observed = fmt.Sprintf("Could not run: the control plane named this account's own bucket (%s) as its reference locker. These credentials are supposed to reach that bucket, so nothing it answers says anything about other buckets. That is a fault on the control plane's side, not a finding about this locker. "+stepsStandAlone, o.Locker.Bucket)
		r.Steps = append(r.Steps, step)
		return
	}
	// A refusal only proves bucket scope when it comes from the same
	// storage endpoint that accepted the credentials in step 1 — scheme,
	// host and port. A control plane pointing this step at some other host
	// — any host answers a foreign credential with 403 — would otherwise
	// buy a passing report.
	pinned, pinnedOK := parseEndpoint(ref.Endpoint)
	if !pinnedOK || !pinned.equal(locker) {
		step.Status = Fail
		step.Observed = fmt.Sprintf("The control plane pointed this step at %s, but step 1 listed this account's locker at %s. A refusal from a different endpoint proves nothing about this locker's credentials, so nothing about bucket scope was shown.", orNone(ref.Endpoint), o.Locker.Endpoint)
		r.Steps = append(r.Steps, step)
		return
	}
	// The pin now agrees, which on a plaintext endpoint means both sides
	// agree on sending a live secret key and session token where anything
	// on the path can read them. No pin makes that the right thing to do,
	// so the probe is not made at all. One address is exempt and it is
	// stated wherever this refusal is documented: a loopback address,
	// where the request does not leave the machine. That is the fake
	// locker the tests and a locally run control plane use, and no Hop
	// endpoint is one.
	if pinned.plaintext() && !pinned.loopback() {
		step.Status = Fail
		step.Observed = fmt.Sprintf("The control plane pointed this step at %s, which is plaintext http. This step signs its request with the same temporary credentials this device pushes with — a secret key and a session token — and it sends those over an unencrypted connection to nothing but this machine's own loopback address, so no request was made and nothing about bucket scope was shown. Step 1 listed this account's locker at %s, so those credentials are already travelling unencrypted on every push: report this to the operator.", ref.Endpoint, o.Locker.Endpoint)
		r.Steps = append(r.Steps, step)
		return
	}
	probe, err := o.OpenReference(ctx, ref)
	if err != nil {
		step.Status = NotApplicable
		step.Observed = "Could not run: no client for the reference locker could be built (" + err.Error() + "), so nothing was asked of it."
		r.Steps = append(r.Steps, step)
		return
	}
	b, akid := probe.Backend, probe.AccessKeyID
	if b == nil {
		step.Status = NotApplicable
		step.Observed = "Could not run: no client for the reference locker was returned, so nothing was asked of it."
		r.Steps = append(r.Steps, step)
		return
	}
	if akid != "" {
		step.Detail = append(step.Detail, "signed with access key id "+akid+" "+accessKeyIDNote)
	}
	step.Status = Pass
	var observed []string
	// The step holds only when the credential that step 1 proved the
	// locker accepts is the one the reference refuses. A credential that
	// was rotated between the steps proves nothing about scope — and
	// proves nothing against the account either: locker credentials last
	// an hour and a push may mint a new one at any time.
	if lockerAKID != "" && akid != "" && lockerAKID != akid {
		degrade(&step, NotApplicable)
		observed = append(observed, fmt.Sprintf("Could not run: the locker credential changed between step 1 (%s) and this step (%s), so a refusal here would not be about the credential the locker accepted. Run rein sync verify again.", lockerAKID, akid))
	}
	objects, listErr := b.List(ctx, "")
	body, fetchErr := fetch(ctx, b, ref.Key)
	// The pin is read before either answer is turned into a verdict. The
	// two endpoint strings compared above both came from the control
	// plane; only the transport says where the request landed, and an
	// answer it cannot place is not evidence about buckets. Deciding the
	// other way round is how a report came to assert that credentials
	// reached a foreign bucket on the strength of an observation the pin
	// then invalidated — degrade cannot lift a Fail, so the alarm could
	// not be taken back.
	pin := pinToResponse(pinned, probe.Exchanges)
	// concluded is the clause that names what an answer means for bucket
	// scope. It is added only when the transport placed the request at the
	// pinned endpoint, because that is what makes "another bucket" a fact
	// rather than a name the control plane supplied.
	concluded := func(clause string) string {
		if pin.placed {
			return " " + clause
		}
		return " Where that request went could not be established (see below), so nothing is concluded from it about bucket scope."
	}
	switch {
	case errors.Is(listErr, backend.ErrAccessDenied):
		observed = append(observed, "Listing the reference locker was refused as access denied.")
	case errors.Is(listErr, backend.ErrCredentialRejected):
		degrade(&step, NotApplicable)
		observed = append(observed, "Could not run: listing the reference locker failed because the credential itself was rejected — "+withCause(explainBackendError(listErr), listErr)+" — and a credential no bucket accepts is refused everywhere, so nothing about bucket scope was shown.")
	case listErr == nil:
		degrade(&step, Fail)
		r.foreignBucket = r.foreignBucket || pin.placed
		observed = append(observed, fmt.Sprintf("Listing the reference locker SUCCEEDED and returned %d object(s).%s", len(objects), concluded("This account's credentials reach a bucket that is not its own.")))
	default:
		// Neither a refusal nor a success: a bucket that has been deleted,
		// an endpoint answering 500, a connection that dropped, a request
		// that timed out. None of it says anything about bucket scope, and
		// none of it is this account's doing.
		degrade(&step, NotApplicable)
		observed = append(observed, "Could not run: listing the reference locker neither succeeded nor was refused as access denied: "+withCause(explainBackendError(listErr), listErr)+".")
	}
	switch {
	case errors.Is(fetchErr, backend.ErrAccessDenied):
		observed = append(observed, "Reading the probe object was refused as access denied.")
	case errors.Is(fetchErr, backend.ErrCredentialRejected):
		degrade(&step, NotApplicable)
		observed = append(observed, "Could not run: reading the probe object failed because the credential itself was rejected — "+withCause(explainBackendError(fetchErr), fetchErr)+" — and a credential no bucket accepts is refused everywhere, so nothing about bucket scope was shown.")
	case fetchErr == nil:
		degrade(&step, Fail)
		r.foreignBucket = r.foreignBucket || pin.placed
		observed = append(observed, fmt.Sprintf("Reading the probe object SUCCEEDED (%d bytes).%s", len(body), concluded("This account's credentials can read another bucket's contents.")))
	default:
		degrade(&step, NotApplicable)
		observed = append(observed, "Could not run: reading the probe object neither succeeded nor was refused as access denied: "+withCause(explainBackendError(fetchErr), fetchErr)+".")
	}
	// The pin's own verdict, read above, is applied last only so its
	// sentence closes the paragraph; it was already decided before any
	// answer was read, and nothing above it drew a conclusion it did not
	// support.
	if pin.verdict != Pass {
		degrade(&step, pin.verdict)
		observed = append(observed, pin.note)
	}
	step.Observed = strings.Join(observed, " ")
	r.Steps = append(r.Steps, step)
}

// stepsStandAlone closes an isolation step that could not run for a reason
// on the operator's side of the service. The reader has three steps that
// did run and a fourth that says nothing; the sentence has to leave them
// knowing which is which.
const stepsStandAlone = "The first three steps above ran entirely against storage and the key on this device and stand on their own; run rein sync verify again later."

// degrade moves a step's verdict, never upward: a Fail sticks, and a
// NotApplicable only replaces a Pass. Two observations in one step must
// not let the second one talk the first out of a failure.
func degrade(step *Step, verdict Status) {
	if verdict == Fail || (verdict == NotApplicable && step.Status == Pass) {
		step.Status = verdict
	}
}

// sameBucket reports whether two bucket names name the same bucket. S3
// bucket names are lowercase, but the two strings here come from different
// places — one from the control plane's reference row, one from the
// locker record — and a comparison this step's honesty rests on should not
// turn on the case somebody typed.
func sameBucket(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// pinResult is what the probe's transport says about the requests the
// isolation step made.
type pinResult struct {
	// placed is true when every request the probe made was observed and
	// went to the pinned endpoint without a redirect. Only then does an
	// answer say which bucket answered it, so only then may the step
	// conclude anything about bucket scope from one.
	placed bool
	// note is the sentence to add to the step, empty when the transport
	// has nothing to add.
	note string
	// verdict is Pass only when the transport confirms a signed S3 refusal
	// from the pinned endpoint.
	verdict Status
}

// pinToResponse judges the isolation step against what the probe's
// transport actually saw, so the step passes only on a refusal that
// carried this account's credential to the pinned host and came back as a
// signed S3 error.
//
// It never turns a failure into a pass: the caller applies a Fail always
// and a NotApplicable only over an otherwise-passing step. NotApplicable
// is the verdict whenever the pin cannot be made, because an unverifiable
// pin shows nothing and must not read as proof.
func pinToResponse(pinned endpointURL, exchanges func() []Exchange) pinResult {
	if exchanges == nil {
		return pinResult{note: "The reference probe's client was not instrumented, so where the request landed and what the endpoint answered could not be checked; nothing about bucket scope is claimed.", verdict: NotApplicable}
	}
	seen := exchanges()
	if len(seen) == 0 {
		return pinResult{note: "The reference probe made no request at all, so this account's credentials were never offered to another bucket and nothing about bucket scope was shown.", verdict: NotApplicable}
	}
	for _, ex := range seen {
		if ex.RedirectedTo != "" {
			return pinResult{note: fmt.Sprintf("The reference locker answered with a redirect to %s, which the probe refused to follow: a credential is only refused by a bucket if it was sent to that bucket, and this one was not, so nothing about bucket scope was shown.", ex.RedirectedTo), verdict: Fail}
		}
		landed, ok := parseEndpoint(ex.Scheme + "://" + ex.Host)
		if !ok || !landed.equal(pinned) {
			return pinResult{note: fmt.Sprintf("The reference probe's request went to %s, not to the pinned endpoint %s, so nothing about this locker's credentials was shown.", orNone(strings.TrimPrefix(ex.Scheme+"://"+ex.Host, "://")), pinned), verdict: Fail}
		}
	}
	refusals := 0
	for _, ex := range seen {
		if ex.Status != http.StatusForbidden {
			continue
		}
		if ex.ErrorCode == "" {
			return pinResult{placed: true, note: fmt.Sprintf("%s answered 403 with no S3 error body. Any web server answers 403; only an S3 refusal naming its code shows a bucket refused this credential, so nothing about bucket scope was shown.", pinned), verdict: NotApplicable}
		}
		refusals++
	}
	if refusals == 0 {
		return pinResult{placed: true, note: fmt.Sprintf("%s never answered 403 to the probe, so no bucket was observed refusing this account's credentials.", pinned), verdict: NotApplicable}
	}
	return pinResult{placed: true, verdict: Pass}
}

// endpointURL is an endpoint reduced to what the isolation step compares:
// scheme, host and port.
//
// The scheme is part of it. http://host and https://host reach the same
// machine, but this step's requests carry a live secret key and session
// token, and one of the two hands them to anything on the path; a pin that
// ignored the scheme would let a control plane downgrade the probe and
// leak the credential while every string still matched. Case, a trailing
// slash, a trailing dot on the host and an implicit default port are not
// differences.
type endpointURL struct {
	scheme string // "http" or "https", lowercased
	host   string // lowercased, no brackets, no trailing dot
	port   string // explicit, or the scheme's default
}

// parseEndpoint reads an endpoint URL, reporting false for anything that
// is not an absolute http or https URL with a host. The same string is
// handed to the S3 client as its base endpoint, which needs a scheme too,
// so a bare host is not an endpoint this step can either use or compare.
func parseEndpoint(raw string) (endpointURL, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return endpointURL{}, false
	}
	e := endpointURL{
		scheme: strings.ToLower(u.Scheme),
		host:   strings.TrimSuffix(strings.ToLower(u.Hostname()), "."),
		port:   u.Port(),
	}
	switch {
	case e.host == "":
		return endpointURL{}, false
	case e.scheme == "https":
		if e.port == "" {
			e.port = "443"
		}
	case e.scheme == "http":
		if e.port == "" {
			e.port = "80"
		}
	default:
		return endpointURL{}, false
	}
	return e, true
}

// String is the canonical form, printed wherever the report names the
// endpoint it compared rather than the string it was handed.
func (e endpointURL) String() string {
	host := e.host
	if strings.Contains(host, ":") {
		host = "[" + host + "]" // an IPv6 literal
	}
	return e.scheme + "://" + host + ":" + e.port
}

func (e endpointURL) equal(other endpointURL) bool { return e == other }

// plaintext reports an endpoint whose requests, and the credentials
// signing them, are readable by anything on the path.
func (e endpointURL) plaintext() bool { return e.scheme == "http" }

// loopback reports an address whose traffic does not leave the machine.
// It is the one exemption from the plaintext refusal above, and it is
// what the test fakes and a locally run control plane use.
//
// The literals (127.0.0.0/8, ::1) are decided here. The name "localhost"
// is taken at its word rather than resolved: RFC 6761 reserves it for the
// loopback interface, but this does not check the resolver, so a machine
// whose hosts file points "localhost" somewhere else would have a
// plaintext probe made to that host. Every other name — including one
// that resolves to a loopback address — is not exempt, which is the side
// to err on.
func (e endpointURL) loopback() bool {
	if e.host == "localhost" {
		return true
	}
	ip := net.ParseIP(e.host)
	return ip != nil && ip.IsLoopback()
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// orInvalid quotes an endpoint the report could not read, so a reader can
// see what was wrong with it, and says nothing when there was none.
func orInvalid(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return fmt.Sprintf(" (%q is not an absolute http or https URL)", s)
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// WriteHuman prints the report for a person: one block per step with what
// was done, what was observed, and the verdict, then the overall outcome.
func (r *Report) WriteHuman(w io.Writer) {
	fmt.Fprintln(w, "VERIFICATION REPORT")
	fmt.Fprintf(w, "Generated %s by %s; storage: %s.\n", r.GeneratedAt, r.ClientVersion, storageLabel(r.Storage))
	if r.Locker.Bucket != "" {
		where := r.Locker.Bucket
		if r.Locker.Prefix != "" {
			where += "/" + r.Locker.Prefix
		}
		if r.Locker.Endpoint != "" {
			where += " at " + r.Locker.Endpoint
		}
		fmt.Fprintf(w, "Checked: %s.\n", where)
	}
	fmt.Fprintln(w)
	for i, s := range r.Steps {
		fmt.Fprintf(w, "Step %d: %s\n", i+1, s.Name)
		fmt.Fprintf(w, "  What was done:  %s\n", s.Did)
		fmt.Fprintf(w, "  What was seen:  %s\n", s.Observed)
		for _, d := range s.Detail {
			fmt.Fprintf(w, "                  - %s\n", d)
		}
		fmt.Fprintf(w, "  Result:         %s\n\n", verdictLabel(s.Status))
	}
	fmt.Fprintln(w, "OUTCOME: "+r.Summary)
}

// NotVerified reports a run that reached no verdict because the storage
// endpoint gave no answer, as opposed to one that reached none because
// there was nothing to check. Both end with Outcome NotApplicable and
// neither is a failure, but only this one is worth a non-zero exit: the
// caller asked for the locker to be checked and it was not.
//
// It is computed while the run happens, so it is false on a report
// decoded from JSON; a decoder reads Summary, which says the same thing
// in words.
func (r *Report) NotVerified() bool {
	return r.Outcome == NotApplicable && r.unreached != ""
}

// IsolationChecked reports whether the isolation step ran and passed, as
// opposed to being not applicable (BYO storage, or a control plane without
// a reference locker).
func (r *Report) IsolationChecked() bool {
	for _, s := range r.Steps {
		if s.ID == StepIsolation {
			return s.Status == Pass
		}
	}
	return false
}

// buildSummary writes the short outcome for a non-expert. It claims only
// what the steps observed: only the fetched objects are called ciphertext
// (the rest were judged by name in step 1), nothing is said about which
// device sealed them, and the isolation clause appears only when the
// isolation step ran and passed.
//
// A failure is not one thing, and the three it can be want three
// different next steps. Telling a new user who has pushed nothing, or
// someone who mistyped a passphrase, to report a security incident is
// worse than saying nothing: it teaches them that this command's alarm
// means nothing, which is precisely the alarm that has to be believed the
// one time it fires for real.
func (r *Report) buildSummary() string {
	switch {
	case r.Outcome == NotApplicable && r.unreached != "":
		// No verdict, and not because there was nothing to check: the
		// storage endpoint gave no answer. Saying "nothing has been pushed
		// yet" here would send the reader to `rein push`, which is not the
		// problem, and would read as a clean bill of health besides.
		return "NOT VERIFIED. " + r.unreached + " Nothing was checked, so nothing here says anything about what the locker holds or where these credentials reach. Run rein sync verify again when the storage endpoint answers."
	case r.Outcome == NotApplicable:
		return "NOT YET VERIFIABLE. Nothing has been pushed from this profile yet, so the locker holds nothing to check. Run rein push, then rein sync verify again."
	case r.Outcome == Fail && (r.plaintext || r.foreignBucket):
		// The two findings this command exists to catch: something other
		// than an age envelope in the locker, or a credential that reached
		// a bucket that is not this account's.
		return "FAIL. " + r.alarm() + " This is what these checks exist to catch. Keep this report; if the locker is a Hop locker, send it to security@reinstate.dev."
	case r.Outcome == Fail && r.wrongKey:
		return "FAIL. The objects in the locker are ciphertext, but " + r.keyAdvice() + " Nothing here says the locker holds anything it should not; it says this device cannot read it."
	case r.Outcome == Fail:
		return "FAIL. At least one step did not hold. The failed step above names what was seen and what to do about it."
	}
	names := r.CheckedObjects
	if len(names) == 0 {
		// A report decoded from an older document carries no record of what
		// step 2 fetched; claim nothing more specific than "fetched".
		names = []string{"the fetched objects"}
	}
	checked := fmt.Sprintf("The objects checked (%s) are ciphertext this device can open.", strings.Join(names, " and "))
	if len(names) == 1 {
		checked = fmt.Sprintf("The object checked (%s) is ciphertext this device can open.", names[0])
	}
	if r.Unopened != "" {
		checked += " " + r.Unopened
	}
	if r.unreached != "" {
		// Something the report set out to fetch gave no answer, and the
		// rest of the run passed. The sentence keeps the two apart rather
		// than letting the pass cover both: a step that could not run is
		// not a step that failed, and it is not a step that passed either.
		checked += " " + r.unreached + " That object was not checked; the steps above say why."
	}
	if r.IsolationChecked() {
		return "PASS. " + checked + " This account's credentials are refused by a bucket that is not its own."
	}
	// The isolation step can end up not-applicable for several reasons —
	// no reference locker, an endpoint that cannot be pinned, a refusal
	// that decided nothing — and it states its own above, so the summary
	// points at it rather than guessing which one it was.
	return "PASS. " + checked + " Whether the credentials reach other buckets was not checked (step 4 above says why), so nothing is claimed about that."
}

// alarm names, in one clause, the finding that warrants a security report.
func (r *Report) alarm() string {
	switch {
	case r.plaintext && r.foreignBucket:
		return "An object in the locker is not encrypted, and this account's credentials reached a bucket that is not its own."
	case r.plaintext:
		return "An object in the locker is not encrypted: something wrote readable bytes where only an age envelope belongs."
	default:
		return "This account's credentials reached a bucket that is not its own, so they are not scoped to this locker."
	}
}

// keyAdvice names the likeliest cause of a key that does not open the
// account's own objects, and what to do about it. It is storage-specific
// because the key is: a passphrase the person types on BYO storage, a
// root key the account holds on Hop.
func (r *Report) keyAdvice() string {
	if r.Storage == StorageHop {
		return "the root key this device holds did not open them. The usual cause is a device enrolled against a different account, or one that never received the current key generation: run rein devices to see whether the keyring holds a wrap for this device, then rein account recover with your recovery code, or rein account join and approve it from an enrolled device. If this device has opened these objects before and nothing about the account has changed, keep this report and send it to security@reinstate.dev."
	}
	return "the passphrase this device used did not open them. The usual cause is a different passphrase than the one given at rein init — it is never stored anywhere, so a typo or a second passphrase produces exactly this. Try again with the passphrase this profile was created with. If you are certain it is the right one, the objects may have been altered in the bucket; keep this report and send it to security@reinstate.dev."
}

func storageLabel(s string) string {
	switch s {
	case StorageHop:
		return "Reinstate Hop locker"
	case StorageBYO:
		return "bring-your-own bucket"
	}
	return s
}

func verdictLabel(s Status) string {
	switch s {
	case Pass:
		return "PASS"
	case Fail:
		return "FAIL"
	case NotApplicable:
		return "NOT APPLICABLE"
	}
	return string(s)
}
