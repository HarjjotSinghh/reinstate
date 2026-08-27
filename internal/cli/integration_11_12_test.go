package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// hopIntegrationEnv is the environment both journeys below start from: the
// lab locker variables cleared, so the CLI reads nothing the bench left
// behind.
var hopIntegrationEnv = []string{
	"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY",
	"REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD",
	"REINSTATE_HOP_LOCATION", "CLAUDE_CONFIG_DIR", "CODEX_HOME",
}

// TestVerifyReadsASignedKeyring is the cross-ticket seam neither branch can
// run on its own: #11's signed key generations and #12's `rein sync verify`
// in one tree.
//
// Four things are asserted that only a merged tree can answer.
//
//  1. The once-per-device verification inside `push --all` fires on a first
//     push that also had to read, verify and anchor a signed keyring. Both
//     tickets put work on that path and neither had seen the other's.
//  2. `rein sync verify` judges `keyring.v1.json` by name only, so the
//     schema-4 object — an ed25519 signature on every generation, a
//     published account key, a revocation record, bound wraps — passes
//     through it untouched, and the report says so rather than claiming to
//     have opened it.
//  3. A device enrolled AFTER a rollover still gets a first-push
//     verification, against a keyring with two generations rather than one.
//  4. Nothing the client posts to the control plane carries any of it.
func TestVerifyReadsASignedKeyring(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000000000x1112")
	plane.reference = &fakeReference{bucket: "lk-0000000000000000000000refr", key: "reference/probe.txt"}
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range hopIntegrationEnv {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")

	// A enrols and pushes. The push uploads, so the once-per-device
	// verification runs inside it, on a tree that also anchors the keyring.
	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	out, errb, code := a.run("push", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("A push: exit=%d out=%q err=%q", code, out, errb)
	}
	var pushed struct {
		Snapshots    []string       `json:"snapshots"`
		Verification map[string]any `json:"verification"`
	}
	if err := json.Unmarshal([]byte(out), &pushed); err != nil {
		t.Fatalf("push --json %q: %v", out, err)
	}
	if pushed.Verification == nil {
		t.Fatalf("the first push ran no verification: %q (stderr %q)", out, errb)
	}
	if pushed.Verification["outcome"] != "pass" || pushed.Verification["posted"] != true {
		t.Fatalf("first-push verification: %+v (stderr %q)", pushed.Verification, errb)
	}
	if !strings.Contains(errb, "First push from this device verified") ||
		!strings.Contains(errb, "refused by a bucket that is not its own") {
		t.Fatalf("first-push line did not claim the isolation check: %q", errb)
	}
	if len(plane.reports) != 1 {
		t.Fatalf("reports posted after A's first push: %d", len(plane.reports))
	}

	// B joins by approval, and A revokes it. That is the rollover: the
	// stored keyring now holds two generations, each signed under the
	// account key the recovery code derives, with a revocation record on
	// the second.
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	join := b.startJoin()
	if out, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("A approves B: exit=%d out=%q err=%q", code, out, errb)
	}
	if out, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: exit=%d out=%q err=%q", code, out, errb)
	}
	bID := deviceID(t, b)
	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}
	ring, _ := a.keyringState(t, plane)
	if ring.SchemaVersion != keyring.SchemaVersion || ring.CurrentGeneration != 2 {
		t.Fatalf("keyring after the rollover: schema=%d current=%d", ring.SchemaVersion, ring.CurrentGeneration)
	}
	account, err := config.LoadAccount(a.home)
	if err != nil || account.AccountKey == "" {
		t.Fatalf("A's enrolment record carries no account key: %+v %v", account, err)
	}
	// Verification needs no key at all, which is what lets step 1 leave the
	// object alone: it is checked against the key this device pinned.
	if err := ring.VerifyGenerations(account.AccountKey); err != nil {
		t.Fatalf("the stored keyring does not verify against the pinned account key: %v", err)
	}
	if len(ring.Generations) != 2 || len(ring.Generations[1].Revoked) != 1 ||
		ring.Generations[1].Revoked[0].DeviceID != bID {
		t.Fatalf("the second generation does not record the revocation: %+v", ring.Generations)
	}

	// #12's step 1 must still read that object as one it judged by name.
	out, errb, code = a.run("sync", "verify", "--json")
	if code != ExitOK {
		t.Fatalf("A sync verify after the rollover: exit=%d out=%q err=%q", code, out, errb)
	}
	v := decodeVerify(t, out)
	if !v.Report.Passed() || !v.Report.IsolationChecked() {
		t.Fatalf("verify against a signed keyring: %s", v.Report.Summary)
	}
	list := stepByID(t, v.Report, "list")
	if !strings.Contains(list.Observed, keyring.ObjectName) ||
		!strings.Contains(list.Observed, "holds no usable key on its own") {
		t.Fatalf("step 1 did not name the keyring as judged by name: %q", list.Observed)
	}
	if !strings.Contains(v.Report.Unopened, "keyring") {
		t.Fatalf("the report does not say the keyring was left unopened: %q", v.Report.Unopened)
	}
	if strings.Contains(v.Report.Summary, "security@reinstate.dev") {
		t.Fatalf("a passing run asked for a security report: %s", v.Report.Summary)
	}

	// A device enrolled after the rollover holds two generations, and its
	// own first push must verify against the two-generation keyring.
	c := newPairDevice(t, plane, "workstation")
	cProject := writeSecondClaudeFixture(t, userHome, "workstation-source", "session-workstation")
	c.enrol(cProject, a.shownCode)
	if _, keys := c.keyringState(t, plane); len(keys) != 2 {
		t.Fatalf("C holds %d generations, want 2", len(keys))
	}
	out, errb, code = c.run("push", "--all", "--json")
	if code != ExitOK {
		t.Fatalf("C push: exit=%d out=%q err=%q", code, out, errb)
	}
	pushed.Verification = nil
	if err := json.Unmarshal([]byte(out), &pushed); err != nil {
		t.Fatalf("C push --json %q: %v", out, err)
	}
	if pushed.Verification == nil || pushed.Verification["outcome"] != "pass" || pushed.Verification["posted"] != true {
		t.Fatalf("C's first-push verification: %+v (stderr %q)", pushed.Verification, errb)
	}
	// Three in all: A's first push, A's explicit `sync verify` (which posts
	// unless --post=false), and C's first push.
	if len(plane.reports) != 3 {
		t.Fatalf("reports posted in total: %d, want 3", len(plane.reports))
	}
	// The posted body carries only step results, and never anything out of
	// the keyring: not the profile id, not the account key, not a
	// signature, not a revoked device's id.
	for i, rep := range plane.reports {
		for _, detail := range []string{ring.ProfileID, ring.AccountKey, ring.Generations[1].Signature, bID} {
			if detail != "" && strings.Contains(string(rep.raw), detail) {
				t.Fatalf("report %d carried keyring detail %q: %s", i, detail, rep.raw)
			}
		}
	}
}

// TestVerifyOnARevokedDeviceIsNotASecurityAlarm is the other half of the
// cross-ticket seam. #12's readability work turned on telling a real
// failure apart from a non-event, and enumerated the non-events it knew
// about; #11 introduces one it could not have known about, a device whose
// token was revoked while it still has a home pointing at the locker. It
// must be told what happened, and must not be told to report a security
// incident, while the device that did the revoking still passes.
func TestVerifyOnARevokedDeviceIsNotASecurityAlarm(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000000revok")
	plane.reference = &fakeReference{bucket: "lk-0000000000000000000000refr", key: "reference/probe.txt"}
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range hopIntegrationEnv {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	if _, errb, code := b.run("login"); code != ExitOK {
		t.Fatalf("B login: %d %q", code, errb)
	}
	if _, errb, code := b.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "desktop-target")); code != ExitOK {
		t.Fatalf("B init --hop: %d %q", code, errb)
	}
	join := b.startJoin()
	if _, errb, code := a.approve(join.code, false); code != ExitOK {
		t.Fatalf("A approves B: %d %q", code, errb)
	}
	if _, errb, code := join.finish(t); code != ExitOK {
		t.Fatalf("B join: %d %q", code, errb)
	}
	if _, errb, code := b.run("pull", "--all"); code != ExitOK {
		t.Fatalf("B pull: %d %q", code, errb)
	}
	if _, errb, code := a.revoke(deviceID(t, b), a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: %d %q", code, errb)
	}

	// A's own first push already posted one; a revoked device must add none.
	before := len(plane.reports)
	out, errb, code := b.run("sync", "verify")
	if code != ExitAuthStorage {
		t.Fatalf("a revoked device's sync verify exited %d, want %d: out=%q err=%q", code, ExitAuthStorage, out, errb)
	}
	if !strings.Contains(errb, "revoked or stale") || !strings.Contains(errb, "rein login") {
		t.Fatalf("a revoked device was not told what happened: %q", errb)
	}
	for _, alarm := range []string{"security@reinstate.dev", "OUTCOME: FAIL"} {
		if strings.Contains(out+errb, alarm) {
			t.Fatalf("a revoked device was shown %q: out=%q err=%q", alarm, out, errb)
		}
	}
	if len(plane.reports) != before {
		t.Fatalf("a revoked device posted %d report(s)", len(plane.reports)-before)
	}

	// The device that did the revoking is unaffected.
	out, errb, code = a.run("sync", "verify", "--json")
	if code != ExitOK {
		t.Fatalf("A sync verify after revoking: exit=%d out=%q err=%q", code, out, errb)
	}
	if v := decodeVerify(t, out); !v.Report.Passed() {
		t.Fatalf("A's verification after the rollover: %s", v.Report.Summary)
	}
}

// TestAPlantedKeyringStopsVerifyAndPushAlike settles, on the tree that
// holds both tickets, what `rein sync verify` does about a keyring a party
// with bucket write access replaced. It is a cross-ticket question because
// each branch answers half of it and the halves point in opposite
// directions: #12 proves the four checks never fetch `keyring.v1.json`
// (internal/verify TestVerifyNeverOpensTheKeyring) and reports it as
// judged by name only, while #11 makes every read path verify the object's
// signatures before it yields a key.
//
// Both together mean something neither says alone. On a Hop profile the
// command has to open the account's root key to decrypt locally in step 3,
// and that load is a read path, so it verifies. A planted keyring
// therefore does not produce a passing report: it produces no report at
// all. The command exits `ExitSafety` before the first check runs, posts
// nothing, and the push that would have used the same keyring refuses it
// and writes nothing.
//
// The distinction the report keeps is still real and is asserted here on
// the genuine object: the checks say the keyring was not opened, so a
// passing report never claims to have examined it. What is refuted is the
// stronger reading — that a planted keyring would sail through the
// trust-establishing command unremarked.
func TestAPlantedKeyringStopsVerifyAndPushAlike(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000000plant")
	plane.reference = &fakeReference{bucket: "lk-0000000000000000000000refr", key: "reference/probe.txt"}
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range hopIntegrationEnv {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}

	// On the genuine object the report names the keyring as one it did not
	// open, and no step claims to have checked a signature.
	out, errb, code := a.run("sync", "verify", "--json")
	if code != ExitOK {
		t.Fatalf("sync verify on the genuine keyring: exit=%d out=%q err=%q", code, out, errb)
	}
	v := decodeVerify(t, out)
	if !strings.Contains(v.Report.Unopened, "keyring") {
		t.Fatalf("the report does not name the keyring as unopened: %q", v.Report.Unopened)
	}
	for _, step := range v.Report.Steps {
		if strings.Contains(step.Did, "signature") || strings.Contains(step.Observed, "signature") {
			t.Fatalf("step %s claims to have checked a signature: did=%q observed=%q", step.ID, step.Did, step.Observed)
		}
	}
	key, genuine := keyringObject(t, plane)
	before := objectKeys(t, plane)
	posted := len(plane.reports)

	// A party with bucket write access replaces the signature on the
	// current generation. Everything else about the object is the
	// account's own.
	var doc map[string]any
	if err := json.Unmarshal(genuine, &doc); err != nil {
		t.Fatal(err)
	}
	gens, ok := doc["generations"].([]any)
	if !ok || len(gens) == 0 {
		t.Fatalf("keyring has no generations: %s", genuine)
	}
	gens[len(gens)-1].(map[string]any)["signature"] = strings.Repeat("A", 86) + "=="
	forged, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(forged), int64(len(forged)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}

	// No report, no verdict, no security-incident advice: the command
	// refuses before it can produce one.
	out, errb, code = a.run("sync", "verify", "--json")
	if code != ExitSafety {
		t.Fatalf("sync verify over a planted keyring: exit=%d, want %d: out=%q err=%q", code, ExitSafety, out, errb)
	}
	if !strings.Contains(errb, "does not verify under account key") {
		t.Fatalf("the refusal does not name the signature: %q", errb)
	}
	for _, absent := range []string{"OUTCOME", "security@reinstate.dev"} {
		if strings.Contains(out+errb, absent) {
			t.Fatalf("a refused verification printed %q: out=%q err=%q", absent, out, errb)
		}
	}
	if len(plane.reports) != posted {
		t.Fatalf("a refused verification posted %d report(s)", len(plane.reports)-posted)
	}

	// The push that would have used that keyring refuses it too, and
	// writes nothing.
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, '\n')
	content = append(content, []byte(`{"type":"user","message":{"content":"written after the forgery"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, "session-planted.jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, code = a.run("push", "--all")
	if code != ExitSafety {
		t.Fatalf("push over a planted keyring: exit=%d, want %d: out=%q err=%q", code, ExitSafety, out, errb)
	}
	if after := objectKeys(t, plane); len(after) != len(before) {
		t.Fatalf("the refused push wrote objects: before=%v after=%v", before, after)
	}
}

// TestTheFloorReachesALaggingDeviceOnBothTicketsSurfaces is the claim that
// changed between rounds, and it changed in the direction the earlier test
// said it had to be re-examined in.
//
// Round two recorded the opposite property here: the generation floor and
// the recipient anchor were recorded per device, so they protected a device
// that had already observed a rollover and nothing protected one that had
// not. A device revoked at the N -> N+1 rollover, inside the credential
// window the product documents, could restore the genuine pre-revocation
// keyring -- every signature in it verifies, because the account really did
// write it -- and a device still at generation N accepted it, correctly by
// every rule it had, and kept sealing to the key the revoked device holds.
//
// Round three closes it with a floor the control plane carries, so this
// test now asserts the closure. It is still a cross-ticket test and belongs
// here rather than beside the floor, because the route class spans both
// tickets: `rein sync verify` is #12's command and reads the keyring
// through the same key resolution `push` and `pull` use, so it is in the
// class whether or not #11 thought about it. A floor that reached push and
// not the trust-establishing command would be the worst of both.
//
// What the floor does not reach is stated in docs/hop.md and exercised
// against the real control plane in keygeneration_crossplane_test.go: a
// deployment that carries no floor, and an operator holding both the
// control plane and the bucket.
func TestTheFloorReachesALaggingDeviceOnBothTicketsSurfaces(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-000000000000000000000lagg")
	plane.reference = &fakeReference{bucket: "lk-0000000000000000000000refr", key: "reference/probe.txt"}
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range hopIntegrationEnv {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	joinDevice := func(name, dir string) *pairDevice {
		t.Helper()
		d := newPairDevice(t, plane, name)
		if _, errb, code := d.run("login"); code != ExitOK {
			t.Fatalf("%s login: %d %q", name, code, errb)
		}
		if _, errb, code := d.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", dir)); code != ExitOK {
			t.Fatalf("%s init --hop: %d %q", name, code, errb)
		}
		j := d.startJoin()
		if _, errb, code := a.approve(j.code, false); code != ExitOK {
			t.Fatalf("A approves %s: %d %q", name, code, errb)
		}
		if _, errb, code := j.finish(t); code != ExitOK {
			t.Fatalf("%s join: %d %q", name, code, errb)
		}
		return d
	}
	b := joinDevice("desktop", "desktop-target")
	c := joinDevice("workstation", "workstation-target")

	// The bytes the revoked device keeps a copy of.
	key, preRevocation := keyringObject(t, plane)
	if _, errb, code := a.revoke(deviceID(t, b), a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: %d %q", code, errb)
	}
	if ring, _ := a.keyringState(t, plane); ring.CurrentGeneration != 2 {
		t.Fatalf("the revocation did not roll the key over: current=%d", ring.CurrentGeneration)
	}
	if plane.keyGeneration != 2 {
		t.Fatalf("the control plane's floor is %d after a rollover to generation 2", plane.keyGeneration)
	}

	// Inside its credential window, B puts the genuine pre-revocation
	// object back. Nothing about it is forged.
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(preRevocation), int64(len(preRevocation)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Parse(preRevocation); err != nil {
		t.Fatalf("the restored object is not the account's own: %v", err)
	}

	// A saw generation 2 itself, so its own record refuses the rollback
	// whether or not a control plane carries a floor.
	out, errb, code := a.run("push", "--all")
	if code != ExitSafety || !strings.Contains(errb, "rolled back") {
		t.Fatalf("A did not refuse the rollback: exit=%d out=%q err=%q", code, out, errb)
	}

	// C has run nothing since generation 1: its own record says generation
	// 1 and the restored object matches it. The only thing that can refuse
	// on C is the floor the control plane carries, and it has to reach
	// both tickets' commands.
	writeSecondClaudeFixture(t, userHome, "workstation-target", "session-lagging")
	before := objectKeys(t, plane)
	for _, tc := range []struct {
		step string
		args []string
	}{
		{step: "push", args: []string{"push", "--all"}},
		{step: "pull", args: []string{"pull", "--all"}},
		{step: "sync verify", args: []string{"sync", "verify"}},
	} {
		out, errb, code := c.run(tc.args...)
		if code != ExitSafety {
			t.Errorf("%s on the lagging device exited %d, want %d: out=%q err=%q", tc.step, code, ExitSafety, out, errb)
		}
		if !strings.Contains(errb, "control plane") {
			t.Errorf("%s on the lagging device said %q; the refusal has to name where the floor came from", tc.step, errb)
		}
	}
	for name := range objectKeys(t, plane) {
		if !before[name] {
			t.Errorf("the lagging device wrote %s after refusing the keyring", name)
		}
	}

	// The revoked device still holds generation 1's root key, which is what
	// made the attack worth making; what it no longer gets is anything C
	// writes afterwards, because C writes nothing.
	_, keys := b.keyringState(t, plane)
	if _, ok := keys[1]; !ok {
		t.Fatalf("the revoked device does not hold generation 1's root key: %v; the attack this closes would not have worked either", keys)
	}
	if _, err := crypto.RootKeyIdentity(keys[1]); err != nil {
		t.Fatal(err)
	}
}

// writeSecondClaudeFixture adds another Claude session under the HOME the
// journey already set up, so a second device has something of its own to
// push. writeClaudeFixture makes its own HOME and can only be called once.
func writeSecondClaudeFixture(t *testing.T, userHome, projectName, session string) string {
	t.Helper()
	project := filepath.Join(userHome, "Projects", projectName)
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(userHome, ".claude", "projects", claudeProjectDirectoryForTest(project))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"type": "meta", "cwd": project})
	content := append(meta, byte('\n'))
	content = append(content, []byte(`{"type":"user","message":{"content":"synthetic integration journey"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(root, session+".jsonl"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return project
}
