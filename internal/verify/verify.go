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
type Report struct {
	Version       int    `json:"version"`
	GeneratedAt   string `json:"generated_at"`
	ClientVersion string `json:"client_version"`
	Storage       string `json:"storage"`
	Outcome       Status `json:"outcome"`
	Steps         []Step `json:"steps"`
	// Locker names what was checked; shown locally, never uploaded.
	Locker LockerInfo `json:"locker"`
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

// Passed reports whether every step passed or did not apply.
func (r *Report) Passed() bool { return r.Outcome == Pass }

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
	// credentials. Required when Reference is set.
	OpenReference func(ctx context.Context, ref hop.Reference) (backend.Backend, error)
	// ClientVersion is recorded in the report.
	ClientVersion string
	// Now is injectable for golden output.
	Now func() time.Time
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

	inv := listStep(ctx, o, r)
	raw := ciphertextStep(ctx, o, r, inv)
	decryptStep(ctx, o, r, inv, raw)
	isolationStep(ctx, o, r)

	r.Outcome = Pass
	for _, s := range r.Steps {
		if s.Status == Fail {
			r.Outcome = Fail
		}
	}
	return r
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

func listStep(ctx context.Context, o Options, r *Report) *inventory {
	step := Step{ID: StepList, Name: "List the locker with this device's credentials",
		Did: "Asked the storage endpoint for every object under this account's prefix, signed with the credentials this device uses to push."}
	objects, err := o.Backend.List(ctx, strings.Trim(o.Prefix, "/"))
	if err != nil {
		step.Status = Fail
		step.Observed = "The listing was refused or failed: " + err.Error()
		r.Steps = append(r.Steps, step)
		return nil
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
		step.Status = Fail
		step.Observed = fmt.Sprintf("%d object(s), but no %s: nothing has been pushed from this profile yet. Run rein push first, then verify again.", inv.total, manifestObject)
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
	return inv
}

// fetched is one object body the ciphertext step read.
type fetched struct {
	name string
	body []byte
}

func ciphertextStep(ctx context.Context, o Options, r *Report, inv *inventory) []fetched {
	step := Step{ID: StepCiphertext, Name: "Fetch an object and check it is ciphertext",
		Did: "Downloaded the index object (and one snapshot, when any exists) and looked at the raw bytes for the age encryption header and for any field name that appears in the plaintext."}
	if inv == nil || !inv.manifest {
		step.Status = Fail
		step.Observed = "Not run: there is no object to fetch (see step 1)."
		r.Steps = append(r.Steps, step)
		return nil
	}
	names := []string{manifestObject}
	if n := len(inv.snapshots); n > 0 {
		names = append(names, snapshotPrefix+inv.snapshots[n-1]+".age")
	}
	var got []fetched
	var observed []string
	step.Status = Pass
	for _, name := range names {
		body, err := fetch(ctx, o.Backend, key(o.Prefix, name))
		if err != nil {
			step.Status = Fail
			observed = append(observed, fmt.Sprintf("%s could not be fetched: %v", name, err))
			continue
		}
		got = append(got, fetched{name: name, body: body})
		verdict, ok := inspectCiphertext(body)
		if !ok {
			step.Status = Fail
		}
		observed = append(observed, fmt.Sprintf("%s (%d bytes): %s", name, len(body), verdict))
		if head, _, found := bytes.Cut(body, []byte("\n")); found && len(head) < 200 {
			step.Detail = append(step.Detail, name+" first line: "+string(head))
		}
	}
	step.Observed = strings.Join(observed, " ")
	r.Steps = append(r.Steps, step)
	return got
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
		step.Status = Fail
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
	var observed []string
	for _, f := range got {
		plain, err := o.Codec.DecryptReader(bytes.NewReader(f.body), o.Keys)
		if err != nil {
			step.Status = Fail
			observed = append(observed, fmt.Sprintf("%s did not decrypt with this device's key: %v.", f.name, err))
			continue
		}
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
	// and file are local detail.
	summary := fmt.Sprintf("%s decrypted into a snapshot envelope whose payload sha256 matches the envelope.", name)
	detail := []string{fmt.Sprintf("%s: agent %s, session %s, project %s, created %s on %s, %d file(s), payload %d bytes, file %s", name, env.Agent, env.SessionID, env.ProjectID, env.CreatedAt, env.SourcePlatform, len(env.Files), n, file.Path)}
	return summary, detail, nil
}

func isolationStep(ctx context.Context, o Options, r *Report) {
	step := Step{ID: StepIsolation, Name: "Prove this account's credentials are refused from another bucket",
		Did: "Asked the control plane for its reference locker (a bucket the operator owns, holding one probe object), then tried to list it and read the probe with the same credentials that just listed this account's locker."}
	switch {
	case o.Storage == StorageBYO:
		step.Status = NotApplicable
		step.Observed = "Not applicable: BYO storage has no control plane and no reference locker. Whether your credentials reach other buckets is decided by your own bucket policy."
		r.Steps = append(r.Steps, step)
		return
	case o.Reference == nil && errors.Is(o.ReferenceErr, hop.ErrNoReference):
		step.Status = NotApplicable
		step.Observed = "Not applicable: " + o.ReferenceErr.Error() + "."
		r.Steps = append(r.Steps, step)
		return
	case o.Reference == nil:
		step.Status = Fail
		step.Observed = "The control plane did not say where its reference locker is"
		if o.ReferenceErr != nil {
			step.Observed += ": " + o.ReferenceErr.Error()
		}
		step.Observed += "."
		r.Steps = append(r.Steps, step)
		return
	case o.OpenReference == nil:
		step.Status = Fail
		step.Observed = "No way to open the reference locker was configured."
		r.Steps = append(r.Steps, step)
		return
	}
	ref := *o.Reference
	step.Detail = append(step.Detail, fmt.Sprintf("reference locker %s at %s, probe %s", ref.Bucket, ref.Endpoint, ref.Key))
	b, err := o.OpenReference(ctx, ref)
	if err != nil {
		step.Status = Fail
		step.Observed = "Could not build a client for the reference locker: " + err.Error() + "."
		r.Steps = append(r.Steps, step)
		return
	}
	step.Status = Pass
	var observed []string
	objects, err := b.List(ctx, "")
	switch {
	case errors.Is(err, backend.ErrUnauthorized):
		observed = append(observed, "Listing the reference locker was refused as unauthorized.")
	case err == nil:
		step.Status = Fail
		observed = append(observed, fmt.Sprintf("Listing the reference locker SUCCEEDED and returned %d object(s); this account's credentials reach a bucket that is not its own.", len(objects)))
	default:
		step.Status = Fail
		observed = append(observed, "Listing the reference locker neither succeeded nor was refused as unauthorized: "+err.Error()+".")
	}
	body, err := fetch(ctx, b, ref.Key)
	switch {
	case errors.Is(err, backend.ErrUnauthorized):
		observed = append(observed, "Reading the probe object was refused as unauthorized.")
	case err == nil:
		step.Status = Fail
		observed = append(observed, fmt.Sprintf("Reading the probe object SUCCEEDED (%d bytes); this account's credentials can read another bucket's contents.", len(body)))
	default:
		step.Status = Fail
		observed = append(observed, "Reading the probe object neither succeeded nor was refused as unauthorized: "+err.Error()+".")
	}
	step.Observed = strings.Join(observed, " ")
	r.Steps = append(r.Steps, step)
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
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
	fmt.Fprintln(w, "OUTCOME: "+r.Summary())
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

// Summary is the one-sentence outcome for a non-expert. It claims only
// what the steps observed: the isolation clause appears only when the
// isolation step ran and passed.
func (r *Report) Summary() string {
	if r.Outcome != Pass {
		return "FAIL. At least one step did not hold; read the failed step above. If the locker is a Hop locker, this is worth reporting to security@reinstate.dev."
	}
	if r.IsolationChecked() {
		return "PASS. Everything in the locker is ciphertext sealed on this device, this device can open it, and this account's credentials are refused by a bucket that is not its own."
	}
	return "PASS. Everything in the locker is ciphertext sealed on this device and this device can open it. Whether the credentials reach other buckets was not checked (no reference locker), so nothing is claimed about that."
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
