package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// TestRecoveryWrapExitTable pins which of the two answers each error gets.
// They are different exits on purpose: a code to re-type is an authentication problem the
// person can fix (4), a damaged keyring is a safety refusal (7).
func TestRecoveryWrapExitTable(t *testing.T) {
	cases := map[string]struct {
		err   error
		want  int
		blame string
	}{
		"mistyped code":        {fmt.Errorf("wrapped: %w", keyring.ErrRecoveryMismatch), ExitAuthStorage, "code as typed"},
		"malformed wrap":       {fmt.Errorf("wrapped: %w", keyring.ErrRecoveryWrapMalformed), ExitSafety, "not a wrong recovery code"},
		"neither":              {errors.New("connection reset"), 0, ""},
		"an unsigned keyring":  {fmt.Errorf("wrapped: %w", keyring.ErrUnauthenticatedGeneration), 0, ""},
		"a missing enrolment":  {keyring.ErrDeviceNotEnrolled, 0, ""},
		"a keyring not found":  {keyring.ErrNotFound, 0, ""},
		"an oversized keyring": {keyring.ErrTooLarge, 0, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			exit := exitForRecoveryWrap(tc.err, "written")
			if tc.want == 0 {
				if exit != nil {
					t.Fatalf("mapped an error that is not a recovery-wrap answer: %v", exit)
				}
				return
			}
			var e *ExitError
			if !errors.As(exit, &e) {
				t.Fatalf("got %v, want an exit error", exit)
			}
			if e.Code != tc.want {
				t.Fatalf("exit %d, want %d", e.Code, tc.want)
			}
			if !strings.Contains(e.Error(), tc.blame) {
				t.Fatalf("message does not say what to do: %q", e.Error())
			}
		})
	}
}

// TestADamagedRecoveryWrapDoesNotReadAsAMistypedCode drives the pair
// through the real CLI, because the message is the whole point of the fix.
//
// A party with write access flips one byte of the recovery wrap's
// ciphertext. Before keyring format 5 that reached the person as "recovery
// code does not match this keyring" — at the one moment where the only
// thing they can act on is the code, and the code was right. Now the wrap is
// inside the generation signature, so the object is refused as tampered
// with, on every route, before a code is asked for at all; and a genuinely
// wrong code still says so, and says what to do about it.
func TestADamagedRecoveryWrapDoesNotReadAsAMistypedCode(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-00000000000000000recovwrap")
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
	bID := deviceID(t, b)

	key, sound := keyringObject(t, plane)
	damaged := flipRecoveryCiphertext(t, sound)
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(damaged), int64(len(damaged)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}

	// Every route refuses, and none of them blames the code.
	c := newPairDevice(t, plane, "laptop")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + filepath.Join(userHome, "Projects", "laptop-target")}} {
		if out, errb, code := c.run(args...); code != ExitOK {
			t.Fatalf("C %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	type probe struct {
		name string
		run  func() (string, string, int)
		want int
	}
	probes := []probe{
		{"A push", func() (string, string, int) { return a.run("push", "--all", "--json") }, ExitSafety},
		{"A revoke", func() (string, string, int) { return a.revoke(bID, a.shownCode) }, ExitSafety},
		{"C recover", func() (string, string, int) { return c.recover(a.shownCode) }, ExitSafety},
	}
	for _, p := range probes {
		out, errb, code := p.run()
		if code != p.want {
			t.Fatalf("%s on a damaged recovery wrap: exit=%d out=%q err=%q", p.name, code, out, errb)
		}
		if strings.Contains(out+errb, "recovery code does not match") {
			t.Fatalf("%s blamed a correct recovery code: %q", p.name, out+errb)
		}
		if !strings.Contains(out+errb, "write access") && !strings.Contains(out+errb, "does not verify") {
			t.Fatalf("%s did not say the keyring was written by someone else: %q", p.name, out+errb)
		}
	}
	// The diagnostics report it rather than exiting on it, and they are held
	// to the same rule about what they blame.
	for _, args := range [][]string{{"account", "status"}, {"devices"}} {
		out, errb, code := a.run(args...)
		if code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
		if !strings.Contains(out+errb, "refuses it") {
			t.Fatalf("A %v did not report the refusal: out=%q err=%q", args, out, errb)
		}
		if strings.Contains(out+errb, "recovery code does not match") {
			t.Fatalf("A %v blamed a correct recovery code: %q", args, out+errb)
		}
	}

	// With the sound keyring back, a genuinely wrong code says so — and
	// says what to do, which is re-enter it.
	if _, err := plane.s3.Store.Put(ctx, key, bytes.NewReader(sound), int64(len(sound)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	wrong, err := keyring.GenerateRecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	out, errb, code := c.recover(wrong)
	if code != ExitAuthStorage {
		t.Fatalf("C recover with the wrong code: exit=%d out=%q err=%q", code, out, errb)
	}
	if !strings.Contains(errb, "recovery code does not match") || !strings.Contains(errb, "Re-enter the code") {
		t.Fatalf("C recover with the wrong code did not name the code: %q", errb)
	}
	// And the right code still works, so the refusal above was about the
	// code and not about the keyring having been touched at all.
	if out, errb, code := c.recover(a.shownCode); code != ExitOK {
		t.Fatalf("C recover with the right code: exit=%d out=%q err=%q", code, out, errb)
	}
}

// flipRecoveryCiphertext changes one byte of generation 1's recovery wrap,
// which is what a party with bucket write access and no recovery code can
// still do to the object.
func flipRecoveryCiphertext(t *testing.T, raw []byte) []byte {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	gens, ok := obj["generations"].([]any)
	if !ok || len(gens) == 0 {
		t.Fatalf("generations is %T", obj["generations"])
	}
	g, ok := gens[0].(map[string]any)
	if !ok {
		t.Fatalf("generation is %T", gens[0])
	}
	wrap, ok := g["recovery"].(map[string]any)
	if !ok {
		t.Fatalf("recovery is %T", g["recovery"])
	}
	cipher, err := base64.StdEncoding.DecodeString(wrap["wrap"].(string))
	if err != nil {
		t.Fatal(err)
	}
	cipher[len(cipher)-1] ^= 0x01
	wrap["wrap"] = base64.StdEncoding.EncodeToString(cipher)
	out, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
