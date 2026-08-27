package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// TestControlPlaneFloorReachesADeviceThatLagsBehind is the probe for the
// one hole the per-device floor cannot cover.
//
// A device that has not yet read a rollover holds nothing locally that a
// rollback would contradict: the pre-revocation keyring is genuine, every
// signature in it verifies, and the generation it names is the generation
// that device last saw. The revoked device, inside the credential window
// its own documentation describes, restores exactly that object — and
// before this floor existed the lagging device accepted it and kept sealing
// to the root key the revoked device holds.
//
// The control plane is the party that saw the rollover the lagging device
// missed. Here A revokes B, which raises the account floor to 2; the
// generation-1 keyring is put back; and C, which has run nothing since
// generation 1, refuses it and names the control plane as the party
// reporting the account has moved on.
//
// Falsified by TestWithoutAControlPlaneFloorALaggingDeviceAcceptsTheOldKeyring,
// which is the same journey against a control plane that carries no floor
// and is the residual the docs state.
func TestControlPlaneFloorReachesADeviceThatLagsBehind(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000000lagging")
	t.Setenv(hopURLEnv, plane.srv.URL)
	clearHopEnv(t)
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	b.enrol(filepath.Join(userHome, "Projects", "desktop-target"), a.shownCode)
	c := newPairDevice(t, plane, "laptop")
	c.enrol(filepath.Join(userHome, "Projects", "laptop-target"), a.shownCode)
	// C reads the keyring once and then goes quiet: from here on it runs
	// nothing until the very end, which is what makes it the lagging device.
	if out, errb, code := c.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("C pull: exit=%d out=%q err=%q", code, out, errb)
	}

	// B keeps a copy of the keyring while it is still enrolled: it holds
	// locker credentials that outlive the revocation by up to an hour.
	key, snapshot := keyringObject(t, plane)
	bID := deviceID(t, b)
	bToken, err := b.tokens.GetDeviceToken()
	if err != nil {
		t.Fatal(err)
	}

	if out, errb, code := a.revoke(bID, a.shownCode); code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}
	plane.mu.Lock()
	floor := plane.keyGeneration
	plane.mu.Unlock()
	if floor != 2 {
		t.Fatalf("control plane floor after the revocation = %d, want 2", floor)
	}

	// The revoked device cannot move the floor in either direction: its
	// token is gone, so the control plane answers 401 to both routes.
	revokedClient := hop.New(plane.srv.URL)
	if _, err := revokedClient.KeyGenerationFloor(ctx, bToken.Token); !errors.Is(err, hop.ErrUnauthorized) {
		t.Fatalf("revoked device read the floor: %v", err)
	}
	if _, err := revokedClient.RaiseKeyGenerationFloor(ctx, bToken.Token, 1); !errors.Is(err, hop.ErrUnauthorized) {
		t.Fatalf("revoked device raised the floor: %v", err)
	}

	// The floor only goes up, so even a token the control plane still
	// accepts cannot talk it back down.
	aToken, err := a.tokens.GetDeviceToken()
	if err != nil {
		t.Fatal(err)
	}
	lowered, err := hop.New(plane.srv.URL).RaiseKeyGenerationFloor(ctx, aToken.Token, 1)
	if err != nil || lowered.Generation != 2 {
		t.Fatalf("floor after being told 1: %+v %v", lowered, err)
	}

	// B puts the genuine generation-1 keyring back.
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(snapshot), int64(len(snapshot)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	account, err := config.LoadAccount(c.home)
	if err != nil {
		t.Fatal(err)
	}
	if account.KeyGeneration != 1 {
		t.Fatalf("C is not the lagging device: it recorded key generation %d", account.KeyGeneration)
	}

	// Nothing C holds locally contradicts the restored keyring. The floor
	// does, and it is what refuses.
	for _, args := range [][]string{{"push", "--all", "--json"}, {"pull", "--all", "--json"}, {"account", "status"}} {
		out, errb, code := c.run(args...)
		combined := out + errb
		if args[0] == "account" {
			// The diagnostic reports the refusal rather than exiting on it.
			if code != ExitOK || !strings.Contains(combined, "refuses it") {
				t.Fatalf("C %v: exit=%d out=%q err=%q", args, code, out, errb)
			}
		} else if code != ExitSafety {
			t.Fatalf("C %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
		if !strings.Contains(combined, string(floorFromControlPlane)) {
			t.Fatalf("C %v did not name the control plane as the floor's source: %q", args, combined)
		}
		if !strings.Contains(combined, "current_generation 1 is below the 2") {
			t.Fatalf("C %v did not name the rollback: %q", args, combined)
		}
	}

	// Having been refused, C has the floor recorded locally, so a control
	// plane that later stops serving the route cannot drop it back to 0.
	account, err = config.LoadAccount(c.home)
	if err != nil {
		t.Fatal(err)
	}
	if account.ControlPlaneKeyGeneration != 2 || account.ControlPlaneConfirmedAt == "" {
		t.Fatalf("C did not record the confirmed floor: %+v", account)
	}
	// A control plane that stops serving the route leaves the last floor it
	// confirmed to this device standing. That is the case the record exists
	// for: a deployment rolled back below the route, or one that never had
	// it, cannot silently drop a floor it has already given out.
	plane.mu.Lock()
	plane.noKeyGenerationFloor = true
	plane.mu.Unlock()
	out, errb, code := c.run("push", "--all", "--json")
	if code != ExitSafety || !strings.Contains(out+errb, string(floorFromLastConfirmed)) {
		t.Fatalf("C push against a control plane with no floor route: exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(out+errb, "current_generation 1 is below the 2") {
		t.Fatalf("C push did not hold the floor it had confirmed: %q", out+errb)
	}

	// A control plane that is answering, and answers a *lower* number, is
	// believed. This is deliberate and it is a change from the first
	// version of this floor, which kept the higher of the two.
	//
	// Keeping the higher number did not defend against a control plane that
	// wants the floor gone -- that one answers 404 and gets the record, or
	// simply never raises it -- and it made a permanent denial of service
	// out of a route any enrolled device may call with a number nobody can
	// verify: one report of a generation no keyring will reach was written
	// into every device's account.json and refused every command on the
	// account for good, with repairing the control plane no help at all.
	// The floor is taken on trust from the control plane; the record is the
	// fallback for one that has stopped answering, not a vote against one
	// that is answering.
	//
	// What does not move is this device's own record of the generation it
	// unwrapped, and that is the half a control plane cannot touch.
	plane.mu.Lock()
	plane.noKeyGenerationFloor = false
	plane.keyGeneration = 0
	plane.mu.Unlock()
	if _, errb, code := c.run("push", "--all", "--json"); code != ExitOK {
		t.Fatalf("C push after the control plane reported 0: exit=%d err=%q", code, errb)
	}

	// The device that did read generation 2 still refuses the rollback, on
	// its own record, with no floor anywhere.
	if _, errb, code := a.run("push", "--all", "--json"); code != ExitSafety || !strings.Contains(errb, string(floorFromLocalRecord)) {
		t.Fatalf("A accepted the rollback once the control plane dropped its floor: exit=%d err=%q", code, errb)
	}
}

// TestAnUnverifiableFloorIsNotAPermanentLockout is the regression the first
// version of this floor introduced and this one closes.
//
// POST /v1/account/key-generation is open to every enrolled device and the
// control plane cannot check what it is told, which the private docs record
// as the price of taking the report on trust. What made that a lockout
// rather than an outage was the client: it wrote the live answer into
// account.json before the keyring was judged and then used the higher of
// the two, so a single report of a generation no keyring would ever hold
// refused every command on every device of the account, permanently, and
// repairing the control plane changed nothing.
//
// The account has to come back when the control plane does.
func TestAnUnverifiableFloorIsNotAPermanentLockout(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-000000000000000000000dosf")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range hopIntegrationEnv {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}

	// Any enrolled device may report anything; here it reports a generation
	// this account will never reach.
	plane.mu.Lock()
	plane.keyGeneration = 1000
	plane.mu.Unlock()
	if _, errb, code := a.run("push", "--all"); code != ExitSafety {
		t.Fatalf("a floor above every keyring did not refuse: exit=%d err=%q", code, errb)
	}

	// The control plane is put right. Every device works again, with no
	// re-enrolment and no recovery code.
	plane.mu.Lock()
	plane.keyGeneration = 0
	plane.mu.Unlock()
	if out, errb, code := a.run("push", "--all"); code != ExitOK {
		t.Fatalf("repairing the control plane did not bring the account back: exit=%d out=%q err=%q", code, out, errb)
	}
	if _, errb, code := a.run("account", "status"); code != ExitOK {
		t.Fatalf("account status after the floor was put right: exit=%d err=%q", code, errb)
	}
}

// TestWithoutAControlPlaneFloorALaggingDeviceAcceptsTheOldKeyring is the
// residual, stated in docs/hop.md and docs/security-model.md rather than
// left for a reader to find: against a control plane that carries no floor
// at all, a device that has never read the rollover has nothing to check a
// restored earlier keyring against, and accepts it.
//
// It is also the falsification of the test above: the same journey, the
// same rollback, the floor route the only difference.
func TestWithoutAControlPlaneFloorALaggingDeviceAcceptsTheOldKeyring(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.noKeyGenerationFloor = true
	plane.s3 = s3test.NewPlain(t, "lk-000000000000000000nofloor")
	t.Setenv(hopURLEnv, plane.srv.URL)
	clearHopEnv(t)
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")
	ctx := context.Background()

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	b := newPairDevice(t, plane, "desktop")
	b.enrol(filepath.Join(userHome, "Projects", "desktop-target"), a.shownCode)
	c := newPairDevice(t, plane, "laptop")
	c.enrol(filepath.Join(userHome, "Projects", "laptop-target"), a.shownCode)
	if out, errb, code := c.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("C pull: exit=%d out=%q err=%q", code, out, errb)
	}

	key, snapshot := keyringObject(t, plane)
	bID := deviceID(t, b)
	out, errb, code := a.revoke(bID, a.shownCode)
	if code != ExitOK {
		t.Fatalf("A revokes B: exit=%d out=%q err=%q", code, out, errb)
	}
	// The revoking operator is told, on stderr, exactly what is not covered.
	if !strings.Contains(errb, "does not carry an account key generation floor") {
		t.Fatalf("revoke did not name the missing floor: out=%q err=%q", out, errb)
	}
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(snapshot), int64(len(snapshot)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	if out, errb, code := c.run("pull", "--all", "--json"); code != ExitOK {
		t.Fatalf("C pull on the restored keyring: exit=%d out=%q err=%q (the residual this test records has changed; update the docs with it)", code, out, errb)
	}
	// A, which did see the rollover, still refuses: the per-device floor
	// covers the devices it can.
	if out, errb, code := a.run("pull", "--all", "--json"); code != ExitSafety || !strings.Contains(errb, string(floorFromLocalRecord)) {
		t.Fatalf("A pull on the restored keyring: exit=%d out=%q err=%q", code, out, errb)
	}
}

// newSignedKeyringForTest builds a one-generation keyring and returns it
// with the recovery code that signed it.
func newSignedKeyringForTest(t *testing.T) (*keyring.Keyring, string) {
	t.Helper()
	rootKey, err := crypto.NewRootKey()
	if err != nil {
		t.Fatal(err)
	}
	defer crypto.Zero(rootKey)
	code, err := keyring.GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	ring, err := keyring.New("profile-floor", rootKey, code, "dev-floor", identity, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return ring, code
}

// TestKeyringAnchorRefusesAnUndecidedFloor pins the fail-closed default: an
// anchor built without deciding about the account-wide floor refuses every
// keyring, however good the signatures are. It is the runtime half of
// TestEveryKeyringAnchorDecidesTheFloor.
func TestKeyringAnchorRefusesAnUndecidedFloor(t *testing.T) {
	ring, code := newSignedKeyringForTest(t)
	accountKey, err := keyring.DeriveAccountKey(ring.ProfileID, code)
	if err != nil {
		t.Fatal(err)
	}
	defer accountKey.Zero()

	cases := []struct {
		name    string
		anchor  keyringAnchor
		wantErr bool
	}{
		{name: "the zero value", anchor: keyringAnchor{}, wantErr: true},
		{name: "undecided", anchor: keyringAnchor{accountKey: accountKey.Public()}, wantErr: true},
		{name: "undecided with a full local record", anchor: keyringAnchor{
			generation: ring.CurrentGeneration, recipient: ring.CurrentRecipient(), accountKey: accountKey.Public(),
		}, wantErr: true},
		{name: "decided, no control plane to ask", anchor: keyringAnchor{
			accountKey: accountKey.Public(), floor: noKeyGenerationFloor(floorFromNoControlPlane),
		}},
		{name: "decided at the generation held", anchor: keyringAnchor{
			accountKey: accountKey.Public(),
			floor:      keyringFloor{decided: true, generation: ring.CurrentGeneration, source: floorFromControlPlane},
		}},
		{name: "decided above the generation held", anchor: keyringAnchor{
			accountKey: accountKey.Public(),
			floor:      keyringFloor{decided: true, generation: ring.CurrentGeneration + 1, source: floorFromControlPlane},
		}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.anchor.check(ring)
			if (err != nil) != tc.wantErr {
				t.Fatalf("check = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil {
				return
			}
			// Every refusal here is a safety exit, not a storage one.
			if exit := exitForKeyringRefusal(err); exit == nil {
				t.Fatalf("refusal %v is not mapped to a safety exit", err)
			}
		})
	}
}

// TestEveryKeyringAnchorDecidesTheFloor is the guard against the class of
// bug this round exists to close: a check added to one route and left off
// the others on the same surface.
//
// Every keyringAnchor built anywhere in internal/cli must set `floor`
// explicitly, whether the answer is "the control plane says N" or "there is
// no control plane to ask". A route added later that constructs one without
// deciding fails here, and would fail closed at run time too
// (TestKeyringAnchorRefusesAnUndecidedFloor) — this test is what makes the
// omission visible before it ships rather than after.
//
// The one form allowed to say nothing is the bare `keyringAnchor{}`: it is
// the zero value returned beside an error, it is never used to check
// anything, and check refuses it if it ever is.
func TestEveryKeyringAnchorDecidesTheFloor(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var undecided []string
	found := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				name, ok := lit.Type.(*ast.Ident)
				if !ok || name.Name != "keyringAnchor" {
					return true
				}
				if len(lit.Elts) == 0 {
					return true
				}
				found++
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "floor" {
						return true
					}
				}
				undecided = append(undecided, fset.Position(lit.Pos()).String())
				return true
			})
		}
	}
	if found == 0 {
		t.Fatal("no keyringAnchor literal found; this guard is looking at the wrong package")
	}
	if len(undecided) != 0 {
		sort.Strings(undecided)
		t.Fatalf("keyringAnchor built without deciding the account key generation floor at:\n  %s\n"+
			"Every route that reads the keyring must establish the floor (confirmKeyGenerationFloor) or record that there is none to ask for (noKeyGenerationFloor).",
			strings.Join(undecided, "\n  "))
	}
}

// TestEveryKeyringUnwrapGoesThroughTheAnchor is the other half of the same
// guard, and the one that catches a route that skips keyringAnchor
// altogether rather than building one badly.
//
// Every call in internal/cli that unwraps a root key out of a keyring must
// sit lexically inside a trustKeyring(...) call — which checks the
// signatures, the account key, the account-wide floor and the local anchor
// before it hands anything back. Two calls deliberately do not, and both are
// listed below with the reason; a third, or a new one anywhere else, fails
// here and has to be argued for in this table rather than merged quietly.
func TestEveryKeyringUnwrapGoesThroughTheAnchor(t *testing.T) {
	// enclosing function -> selector -> how many unguarded calls are
	// expected there, and why.
	//
	//	newAccountRecoverCmd / UnwrapGenerationsWithRecoveryCode:
	//	  the pre-check that lets a mistyped code be reported as a mistyped
	//	  code rather than as a tampered keyring. The keys it returns are
	//	  zeroed immediately and nothing is written or sealed from them; the
	//	  enrolment itself goes through trustKeyring a few lines later.
	//	newAccountRecoverCmd / UnwrapForDevice:
	//	  the re-attach branch, reached only after codeAnchor.check(ring) has
	//	  already run the full anchor — signatures, account key and floor —
	//	  against the same object. It proves this machine's stored device key
	//	  still opens its wrap; the keys are zeroed immediately.
	allowed := map[string]int{
		"newAccountRecoverCmd.UnwrapGenerationsWithRecoveryCode": 1,
		"newAccountRecoverCmd.UnwrapForDevice":                   1,
	}
	unwraps := map[string]bool{
		"UnwrapGenerations":                 true,
		"UnwrapForDevice":                   true,
		"UnwrapWithRecoveryCode":            true,
		"UnwrapGenerationsWithRecoveryCode": true,
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	var reported []string
	total := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Every span covered by a trustKeyring(...) call, including the
			// closure it is handed.
			var guarded [][2]token.Pos
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "trustKeyring" {
					guarded = append(guarded, [2]token.Pos{call.Pos(), call.End()})
				}
				return true
			})
			inside := func(p token.Pos) bool {
				for _, span := range guarded {
					if p >= span[0] && p < span[1] {
						return true
					}
				}
				return false
			}
			var enclosing string
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					enclosing = fn.Name.Name
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !unwraps[sel.Sel.Name] {
					return true
				}
				total++
				if inside(call.Pos()) {
					return true
				}
				key := enclosing + "." + sel.Sel.Name
				found[key]++
				if found[key] > allowed[key] {
					reported = append(reported, fmt.Sprintf("%s at %s", key, fset.Position(call.Pos())))
				}
				return true
			})
		}
	}
	if total == 0 {
		t.Fatal("no keyring unwrap found; this guard is looking at the wrong package")
	}
	if len(reported) != 0 {
		sort.Strings(reported)
		t.Fatalf("keyring unwrapped outside trustKeyring at:\n  %s\n"+
			"Route it through trustKeyring, which checks the signatures, the account key, the account key generation floor and the local anchor first — or add it to this test's table with the reason it is safe.",
			strings.Join(reported, "\n  "))
	}
	for key, want := range allowed {
		if found[key] != want {
			t.Fatalf("%s: %d unguarded unwrap(s), the table expects %d. If one was removed, remove it from the table too; the table is the record of what was argued for.", key, found[key], want)
		}
	}
}
