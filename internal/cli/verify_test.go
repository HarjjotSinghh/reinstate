package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/processcheck"
	"github.com/HarjjotSinghh/reinstate/internal/verify"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files")

// verifyJSON is `rein sync verify --json` as a test reads it.
type verifyJSON struct {
	Report verify.Report `json:"report"`
	Posted bool          `json:"posted"`
}

func decodeVerify(t *testing.T, out string) verifyJSON {
	t.Helper()
	var v verifyJSON
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("verify --json output %q: %v", out, err)
	}
	return v
}

func stepByID(t *testing.T, r verify.Report, id string) verify.Step {
	t.Helper()
	for _, s := range r.Steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("report has no step %q: %+v", id, r.Steps)
	return verify.Step{}
}

// hostedVerifyJourney signs in, inits for Hop, creates the root key, and
// pushes once, with the fake control plane advertising a reference locker.
func hostedVerifyJourney(t *testing.T) (*lockerJourney, string) {
	t.Helper()
	j := newLockerJourney(t)
	j.plane.reference = &fakeReference{bucket: "lk-0000000000000000000000refr", key: "reference/probe.txt"}
	project := writeClaudeFixture(t)
	if _, errb, code := j.run("login"); code != ExitOK {
		t.Fatalf("login exit=%d err=%q", code, errb)
	}
	if _, errb, code := j.run("init", "--hop", "--project", "local/locker="+project); code != ExitOK {
		t.Fatalf("init --hop exit=%d err=%q", code, errb)
	}
	if _, errb, code := j.run("account", "init"); code != ExitOK {
		t.Fatalf("account init exit=%d err=%q", code, errb)
	}
	return j, project
}

// akid is the newest credential the fake control plane minted, the one
// every request of the latest run was signed with.
func (j *lockerJourney) akid() string {
	j.t.Helper()
	j.plane.mu.Lock()
	defer j.plane.mu.Unlock()
	if len(j.plane.mints) == 0 {
		j.t.Fatal("no credential minted")
	}
	return j.plane.mints[len(j.plane.mints)-1]
}

func (j *lockerJourney) object(key string) []byte {
	j.t.Helper()
	rc, _, err := j.plane.s3.Store.Get(context.Background(), key)
	if err != nil {
		j.t.Fatalf("locker object %s: %v", key, err)
	}
	defer rc.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(rc)
	return buf.Bytes()
}

func (j *lockerJourney) overwrite(key string, body []byte) {
	j.t.Helper()
	if err := j.plane.s3.Store.Delete(context.Background(), key); err != nil && err != backend.ErrNotFound {
		j.t.Fatal(err)
	}
	if _, err := j.plane.s3.Store.Put(context.Background(), key, bytes.NewReader(body), int64(len(body)), backend.PutOptions{}); err != nil {
		j.t.Fatal(err)
	}
}

// TestSyncVerifyJourneyHosted: the first push runs the verification once
// and posts it; rein sync verify passes, reads as a report, posts again on
// demand, and never uploads a session name.
func TestSyncVerifyJourneyHosted(t *testing.T) {
	j, project := hostedVerifyJourney(t)

	out, errb, code := j.run("push", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
	}
	var pushed struct {
		Verification struct {
			Outcome string `json:"outcome"`
			Posted  bool   `json:"posted"`
		} `json:"verification"`
	}
	if err := json.Unmarshal([]byte(out), &pushed); err != nil || pushed.Verification.Outcome != "pass" || !pushed.Verification.Posted {
		t.Fatalf("push output %q err=%v", out, err)
	}
	if !strings.Contains(errb, "First push from this device verified: the index and the newest snapshot fetched from the locker are ciphertext this device can open, and this account's credentials are refused by a bucket that is not its own") {
		t.Fatalf("push stderr %q", errb)
	}
	if len(j.plane.reports) != 1 {
		t.Fatalf("reports after first push: %d", len(j.plane.reports))
	}
	state, err := config.LoadState(j.home)
	if err != nil || state.VerifyReportedAt == "" {
		t.Fatalf("state after first push: %+v err=%v", state, err)
	}

	// A second push (nothing to upload) and a third with a change post no
	// further report: once per device.
	if _, _, code := j.run("push", "--all"); code != ExitOK {
		t.Fatal("second push failed")
	}
	sessionFile := filepath.Join(os.Getenv("HOME"), ".claude", "projects", claudeProjectDirectoryForTest(project), "session-locker.jsonl")
	raw, _ := os.ReadFile(sessionFile)
	if err := os.WriteFile(sessionFile, append(raw, []byte(`{"type":"user","message":{"content":"more"}}`+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, code = j.run("push", "--all", "--json")
	if code != ExitOK || strings.Contains(out, "verification") || strings.Contains(errb, "verified") {
		t.Fatalf("third push re-verified: exit=%d out=%q err=%q", code, out, errb)
	}
	if len(j.plane.reports) != 1 {
		t.Fatalf("reports after later pushes: %d", len(j.plane.reports))
	}

	// On demand: human report, posted.
	out, errb, code = j.run("sync", "verify")
	if code != ExitOK {
		t.Fatalf("sync verify exit=%d out=%q err=%q", code, out, errb)
	}
	for _, want := range []string{
		"VERIFICATION REPORT",
		"Checked: lk-0000000000000000000000test at " + j.plane.s3.URL(),
		"Step 1: List the locker with this device's credentials",
		"manifest.age (the encrypted index)",
		"keyring.v1.json (the root key, wrapped per device",
		"2 snapshot(s) under snapshots/ named by opaque ids",
		"Step 2: Fetch an object and check it is ciphertext",
		"begins with the age v1 header (recipient X25519 (root key)); no plaintext field name appears anywhere in the body",
		"Step 3: Decrypt the object locally",
		"manifest.age decrypted into a schema v1 index.",
		"- index revision ",
		"1 session(s) (claude 1)",
		"index entry claude:session-locker -> snapshots/",
		"decrypted into a snapshot envelope whose payload sha256 matches the envelope",
		": agent claude, session session-locker, project ",
		"Step 4: Prove this account's credentials are refused from another bucket",
		"reference locker lk-0000000000000000000000refr at " + j.plane.s3.URL() + ", probe reference/probe.txt",
		"Listing the reference locker was refused as access denied. Reading the probe object was refused as access denied.",
		"Result:         PASS",
		"OUTCOME: PASS. The objects checked (the index and the newest snapshot) are ciphertext this device can open. Not opened and judged by name only: 1 older age-named snapshot(s), the wrapped keyring. This account's credentials are refused by a bucket that is not its own.",
		"Step results posted to the control plane",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sync verify output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "FAIL") || strings.Contains(out, "Everything in the locker") || strings.Contains(out, "sealed on this device") {
		t.Fatalf("a passing report mentions FAIL or over-claims:\n%s", out)
	}
	// Steps 1 and 4 name the same access key id: the credential the locker
	// accepted is the one the reference refused. The id stays local.
	if n := strings.Count(out, "- signed with access key id "+j.akid()); n != 2 {
		t.Fatalf("access key id recorded %d time(s), want 2 (steps 1 and 4):\n%s", n, out)
	}
	if bytes.Contains(j.plane.reports[1].raw, []byte(j.akid())) {
		t.Fatalf("posted report carries the access key id: %s", j.plane.reports[1].raw)
	}
	if len(j.plane.reports) != 2 {
		t.Fatalf("reports after sync verify: %d", len(j.plane.reports))
	}
	// The reference probe was signed with the credential the locker had,
	// not a freshly minted one.
	log := j.plane.s3.RequestLog()
	var foreign []string
	for _, l := range log {
		if strings.Contains(l, "(foreign bucket)") {
			foreign = append(foreign, l)
		}
	}
	if len(foreign) < 2 || !strings.Contains(foreign[0], "lk-0000000000000000000000refr") {
		t.Fatalf("reference probes:\n%s", strings.Join(log, "\n"))
	}

	// The upload carries step results only: opaque keys, no session or
	// project name; the local JSON keeps the detail.
	posted := j.plane.reports[1]
	for _, secret := range []string{"session-locker", "locker-source", "synthetic locker journey", "detail"} {
		if bytes.Contains(posted.raw, []byte(secret)) {
			t.Fatalf("posted report contains %q:\n%s", secret, posted.raw)
		}
	}
	if posted.report.Storage != "hop" || posted.report.Outcome != "pass" || len(posted.report.Steps) != 4 || posted.token == "" {
		t.Fatalf("posted report %+v", posted.report)
	}

	out, _, code = j.run("sync", "verify", "--json", "--post=false")
	if code != ExitOK {
		t.Fatalf("sync verify --json exit=%d out=%q", code, out)
	}
	v := decodeVerify(t, out)
	if v.Posted || len(j.plane.reports) != 2 {
		t.Fatalf("--post=false still posted: %+v reports=%d", v, len(j.plane.reports))
	}
	if v.Report.Storage != "hop" || v.Report.Outcome != "pass" || v.Report.Locker.Bucket != "lk-0000000000000000000000test" {
		t.Fatalf("report %+v", v.Report)
	}
	if d := stepByID(t, v.Report, "decrypt").Detail; len(d) == 0 || !strings.Contains(strings.Join(d, "\n"), "session session-locker") {
		t.Fatalf("local detail lost: %v", d)
	}
}

// TestSyncVerifyJourneyTamperedObjects: an object replaced with plaintext
// fails the ciphertext step; a ciphertext with a flipped byte still looks
// like ciphertext but fails to decrypt; both exit with the safety code.
func TestSyncVerifyJourneyTamperedObjects(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	original := j.object("manifest.age")

	t.Run("plaintext in place of ciphertext", func(t *testing.T) {
		j.overwrite("manifest.age", []byte(`{"schema_version":1,"revision":"r1","sessions":{}}`))
		out, errb, code := j.run("sync", "verify", "--json", "--post=false")
		if code != ExitSafety || !strings.Contains(errb, "verification failed") {
			t.Fatalf("exit=%d err=%q", code, errb)
		}
		v := decodeVerify(t, out)
		if v.Report.Outcome != "fail" {
			t.Fatalf("outcome %s", v.Report.Outcome)
		}
		if s := stepByID(t, v.Report, "ciphertext"); s.Status != "fail" || !strings.Contains(s.Observed, "does NOT begin with the age v1 header") {
			t.Fatalf("ciphertext step %+v", s)
		}
		if s := stepByID(t, v.Report, "decrypt"); s.Status != "fail" || !strings.Contains(s.Observed, "did not decrypt") {
			t.Fatalf("decrypt step %+v", s)
		}
		if s := stepByID(t, v.Report, "isolation"); s.Status != "pass" {
			t.Fatalf("isolation step still runs: %+v", s)
		}
	})

	t.Run("flipped byte in the ciphertext", func(t *testing.T) {
		corrupt := append([]byte(nil), original...)
		corrupt[len(corrupt)-8] ^= 0xff
		j.overwrite("manifest.age", corrupt)
		out, _, code := j.run("sync", "verify", "--json", "--post=false")
		if code != ExitSafety {
			t.Fatalf("exit=%d", code)
		}
		v := decodeVerify(t, out)
		if s := stepByID(t, v.Report, "ciphertext"); s.Status != "pass" {
			t.Fatalf("ciphertext step %+v", s)
		}
		if s := stepByID(t, v.Report, "decrypt"); s.Status != "fail" || !strings.Contains(s.Observed, "manifest.age did not decrypt") {
			t.Fatalf("decrypt step %+v", s)
		}
	})

	j.overwrite("manifest.age", original)
	if _, _, code := j.run("sync", "verify", "--post=false"); code != ExitOK {
		t.Fatal("restored manifest does not verify")
	}
}

// TestSyncVerifyJourneyReferenceReachable: a storage endpoint that lets
// the account's credentials into another bucket is a failed verification,
// and the failing report is still posted.
func TestSyncVerifyJourneyReferenceReachable(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	// Serve every bucket from the one store, as a misconfigured endpoint
	// would, with the probe present.
	j.plane.s3.Mu.Lock()
	j.plane.s3.AnyBucket = true
	j.plane.s3.SharedStore = true
	j.plane.s3.Mu.Unlock()
	j.overwrite("reference/probe.txt", []byte("Reinstate Hop reference locker probe.\n"))

	out, errb, code := j.run("sync", "verify")
	if code != ExitSafety {
		t.Fatalf("exit=%d out=%q err=%q", code, out, errb)
	}
	for _, want := range []string{"Listing the reference locker SUCCEEDED", "Reading the probe object SUCCEEDED (", "Result:         FAIL", "OUTCOME: FAIL.", "security@reinstate.dev"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if n := len(j.plane.reports); n != 2 || j.plane.reports[1].report.Outcome != "fail" {
		t.Fatalf("failing report not posted: %d reports", n)
	}
}

// TestSyncVerifyJourneyReferenceRejectsTheCredential: a 403 that says the
// credential itself is bad (unknown key id, bad signature, expired token)
// is what every bucket answers a dead credential, so it proves nothing
// about scope: step 4 fails and says so, through the real S3 client.
func TestSyncVerifyJourneyReferenceRejectsTheCredential(t *testing.T) {
	for _, code := range []string{"InvalidAccessKeyId", "SignatureDoesNotMatch", "ExpiredToken", "InvalidToken"} {
		t.Run(code, func(t *testing.T) {
			j, _ := hostedVerifyJourney(t)
			if _, errb, code := j.run("push", "--all"); code != ExitOK {
				t.Fatalf("push exit=%d err=%q", code, errb)
			}
			j.plane.s3.Mu.Lock()
			j.plane.s3.ForeignBucketAs = code
			j.plane.s3.Mu.Unlock()
			out, _, exit := j.run("sync", "verify", "--post=false")
			if exit != ExitSafety {
				t.Fatalf("exit=%d:\n%s", exit, out)
			}
			for _, want := range []string{
				"Listing the reference locker failed because the credential itself was rejected (backend: credential rejected (" + code + ")), so nothing about bucket scope was shown.",
				"Reading the probe object failed because the credential itself was rejected (",
				"OUTCOME: FAIL.",
			} {
				if !strings.Contains(out, want) {
					t.Fatalf("output missing %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "refused as access denied") || strings.Contains(out, "refused by a bucket that is not its own") {
				t.Fatalf("a rejected credential was read as a scope refusal:\n%s", out)
			}
		})
	}
}

// TestSyncVerifyJourneyNoReferenceLocker: a control plane without a
// reference locker makes the isolation step not applicable, not a failure.
func TestSyncVerifyJourneyNoReferenceLocker(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	j.plane.reference = nil
	_, errb, code := j.run("push", "--all")
	if code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}
	// The push-hook sentence must not claim isolation that was not checked.
	if !strings.Contains(errb, "First push from this device verified") || !strings.Contains(errb, "Isolation was not checked (no reference locker)") || strings.Contains(errb, "refused by a bucket") {
		t.Fatalf("push stderr %q", errb)
	}
	out, _, code := j.run("sync", "verify", "--json", "--post=false")
	if code != ExitOK {
		t.Fatalf("exit=%d out=%q", code, out)
	}
	v := decodeVerify(t, out)
	s := stepByID(t, v.Report, "isolation")
	if s.Status != "not-applicable" || !strings.Contains(s.Observed, "no reference locker") || v.Report.Outcome != "pass" {
		t.Fatalf("isolation %+v outcome %s", s, v.Report.Outcome)
	}
	human, _, code := j.run("sync", "verify", "--post=false")
	if code != ExitOK || !strings.Contains(human, "OUTCOME: PASS. The objects checked (the index and the newest snapshot) are ciphertext this device can open. Not opened and judged by name only: the wrapped keyring. Whether the credentials reach other buckets was not checked (no reference locker)") || strings.Contains(human, "refused by a bucket") {
		t.Fatalf("human summary claims isolation:\n%s", human)
	}
}

// TestSyncVerifyBeforeAnyPush: nothing pushed yet is a failed listing with
// the next step spelled out, and nothing is posted after a push that
// uploaded nothing.
func TestSyncVerifyBeforeAnyPush(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	out, _, code := j.run("sync", "verify", "--post=false")
	if code != ExitSafety || !strings.Contains(out, "no manifest.age: nothing has been pushed from this profile yet. Run rein push first") {
		t.Fatalf("exit=%d out=%q", code, out)
	}
	if len(j.plane.reports) != 0 {
		t.Fatal("a report was posted with --post=false")
	}
}

// TestSyncVerifyJourneyBYO: BYO storage runs the first three checks with
// the passphrase and reports the isolation step as not applicable; there
// is no control plane to post to. The JSON output is pinned by a golden
// file so the shape a script reads never drifts by accident.
func TestSyncVerifyJourneyBYO(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "memory")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")
	t.Setenv("REINSTATE_HOP_URL", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	project := writeClaudeFixture(t)

	codec := &fastAgeEnvelopeCodec{}
	fixed := time.Date(2026, 8, 23, 12, 10, 0, 0, time.UTC)
	run := func(args ...string) (string, string, int) {
		passphraseFile, err := os.CreateTemp(t.TempDir(), "passphrase-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = passphraseFile.Close() }()
		if _, err := passphraseFile.WriteString("verify-test-passphrase-not-real\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := passphraseFile.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		t.Setenv("REINSTATE_PASSPHRASE_FD", strconv.FormatUint(uint64(passphraseFile.Fd()), 10))
		ctx := context.WithValue(context.Background(), verifyNowContextKey{}, func() time.Time { return fixed })
		var out, errb bytes.Buffer
		code := Execute(Options{
			Name: "rein", Stdout: &out, Stderr: &errb, Args: args, Context: ctx,
			AgentProcessChecker: func(_ context.Context, _ string, _ processcheck.Target) (bool, bool, error) { return false, true, nil },
			EnvelopeCodec:       codec,
		})
		return out.String(), errb.String(), code
	}

	if out, errb, code := run("init", "--endpoint", "https://example.r2.cloudflarestorage.com", "--bucket", "reinstate-test", "--prefix", "team/a", "--project", "local/locker="+project, "--yes"); code != ExitOK {
		t.Fatalf("init exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := run("push", "--all"); code != ExitOK || strings.Contains(errb, "verified") {
		t.Fatalf("push exit=%d out=%q err=%q", code, out, errb)
	}

	out, errb, code := run("sync", "verify")
	if code != ExitOK {
		t.Fatalf("sync verify exit=%d out=%q err=%q", code, out, errb)
	}
	for _, want := range []string{
		"storage: bring-your-own bucket",
		"Checked: memory:",
		"recipient scrypt (passphrase)",
		"Result:         NOT APPLICABLE",
		"Not applicable: BYO storage has no control plane and no reference locker",
		"OUTCOME: PASS. The objects checked (the index and the newest snapshot) are ciphertext this device can open. No other object is in the locker. Whether the credentials reach other buckets was not checked (no reference locker)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "posted") {
		t.Fatalf("BYO verify claims to post:\n%s", out)
	}

	out, _, code = run("sync", "verify", "--json")
	if code != ExitOK {
		t.Fatalf("sync verify --json exit=%d out=%q", code, out)
	}
	got := normalizeVerifyJSON(t, out, home)
	golden := filepath.Join("testdata", "verify", "byo-report.golden.json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden: %v (run with -update to create it)", err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		t.Fatalf("verify --json drifted from %s:\n--- want\n%s\n--- got\n%s", golden, want, got)
	}
}

// normalizeVerifyJSON replaces the values that differ run to run (snapshot
// ids, timestamps, sizes, the temp home) with fixed tokens and re-indents.
func normalizeVerifyJSON(t *testing.T, out, home string) []byte {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("json %q: %v", out, err)
	}
	report := v["report"].(map[string]any)
	report["client_version"] = "reinstate <version>"
	locker := report["locker"].(map[string]any)
	locker["bucket"] = "memory:<home>/cache/memory-backend"
	m := regexp.MustCompile(`snapshots/([0-9a-f-]{36})\.age`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no snapshot id in %s", out)
	}
	snapshot := m[1]
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	raw := bytes.TrimSpace(buf.Bytes())
	s := string(raw)
	s = strings.ReplaceAll(s, snapshot, "<snapshot-id>")
	s = strings.ReplaceAll(s, home, "<home>")
	s = replaceAllRegexp(s, `"updated \d{4}-[^)]+\)`, `"updated <time>)`)
	s = replaceAllRegexp(s, `updated \d{4}-\d{2}-\d{2}T[0-9:]+Z`, `updated <time>`)
	s = replaceAllRegexp(s, `created \d{4}-\d{2}-\d{2}T[0-9:]+Z on [a-z0-9-]+`, `created <time> on <platform>`)
	s = replaceAllRegexp(s, `manifest\.age \(\d+ bytes\)`, `manifest.age (<n> bytes)`)
	s = replaceAllRegexp(s, `snapshots/<snapshot-id>\.age \(\d+ bytes\)`, `snapshots/<snapshot-id>.age (<n> bytes)`)
	s = replaceAllRegexp(s, `with 1 file \(\d+ bytes\)`, `with 1 file (<n> bytes)`)
	s = replaceAllRegexp(s, `file projects/[^/"]+/`, `file projects/<project-dir>/`)
	return []byte(s + "\n")
}

func replaceAllRegexp(s, pattern, repl string) string {
	return regexp.MustCompile(pattern).ReplaceAllString(s, repl)
}
