package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// TestKeyringRefusalsAllExitSafety pins the exit code every keyring refusal
// carries. `docs/hop.md` and `docs/security-model.md` both name `ExitSafety`
// for these, and before this table the oversized-keyring refusal reached the
// user as the generic `4` (auth_storage) instead, because nothing mapped
// keyring.ErrTooLarge at all. Errors that are not a refusal must stay
// unmapped, so an unrelated storage failure is not dressed up as one.
func TestKeyringRefusalsAllExitSafety(t *testing.T) {
	cases := map[string]struct {
		err  error
		want int
	}{
		"rolled back":                   {&keyringRolledBackError{saw: 1, floor: 2}, ExitSafety},
		"rewritten":                     {&keyringRewrittenError{generation: 1, want: "age1a", found: "age1b"}, ExitSafety},
		"anchor without an account key": {&keyringAnchorBrokenError{missing: "account signing key"}, ExitSafety},
		"unsigned generation":           {fmt.Errorf("wrapped: %w", keyring.ErrUnauthenticatedGeneration), ExitSafety},
		"foreign account key":           {fmt.Errorf("wrapped: %w", keyring.ErrAccountKeyMismatch), ExitSafety},
		"too large to write":            {fmt.Errorf("wrapped: %w", keyring.ErrTooLarge), ExitSafety},
		"not a refusal":                 {errors.New("connection reset"), 0},
		"not found":                     {keyring.ErrNotFound, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			exit := exitForKeyringRefusal(tc.err)
			if tc.want == 0 {
				if exit != nil {
					t.Fatalf("mapped an error that is not a keyring refusal: %v", exit)
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
		})
	}
}

// TestOversizedKeyringRefusalExitsSafety drives the size cliff through the
// real CLI. The keyring only grows — every revocation appends a generation
// holding one wrap per remaining device — and a write that would take it
// past three quarters of the 1 MiB a read accepts is refused, because an
// account that wrote a larger one could never read it back.
//
// `docs/hop.md` documents that refusal as `ExitSafety`. The object here is
// padded with long device ids rather than driven there by two hundred real
// revocations: what is under test is the exit code and the message, not the
// arithmetic, which `internal/keyring` covers directly.
func TestOversizedKeyringRefusalExitsSafety(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000anchor")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	userHome := os.Getenv("HOME")

	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	key, genuine := keyringObject(t, plane)
	ring, err := keyring.Parse(genuine)
	if err != nil {
		t.Fatal(err)
	}
	// Pad generation 1 with device wraps until the object is over the
	// write ceiling but still under the 1 MiB a read accepts, so it loads
	// and only the write is refused. The padding wraps open for nobody,
	// which nothing on this path cares about: no signature covers the
	// device table, and every real wrap is still where it was.
	pad := ring.Generations[0].Devices[0]
	filler := strings.Repeat("p", 2000)
	raw := genuine
	for i := 0; len(raw) < 820_000; i++ {
		clone := pad
		clone.DeviceID = fmt.Sprintf("%s-%04d", filler, i)
		ring.Generations[0].Devices = append(ring.Generations[0].Devices, clone)
		if i%50 != 0 {
			continue
		}
		if raw, err = ring.Marshal(); err != nil {
			t.Fatal(err)
		}
	}
	if raw, err = ring.Marshal(); err != nil {
		t.Fatal(err)
	}
	if len(raw) >= 1<<20 {
		t.Fatalf("the padded keyring is %d bytes, past the size a read accepts", len(raw))
	}
	if _, err := plane.s3.Store.Put(context.Background(), key, bytes.NewReader(raw), int64(len(raw)), backendPutOptions()); err != nil {
		t.Fatal(err)
	}
	// It still loads and still verifies: only the write is refused.
	if _, err := keyring.Parse(raw); err != nil {
		t.Fatalf("the padded keyring no longer parses, so the probe proves nothing: %v", err)
	}

	d := newPairDevice(t, plane, "workstation")
	if _, errb, code := d.run("login"); code != ExitOK {
		t.Fatalf("D login: %d %q", code, errb)
	}
	if _, errb, code := d.run("init", "--hop", "--project", "local/locker="+filepath.Join(userHome, "Projects", "workstation-target")); code != ExitOK {
		t.Fatalf("D init --hop: %d %q", code, errb)
	}
	out, errb, code := d.recover(a.shownCode)
	if code != ExitSafety {
		t.Fatalf("D recover against an oversized keyring: exit=%d (want %d) out=%q err=%q", code, ExitSafety, out, errb)
	}
	for _, want := range []string{"too large to write", "start the account again on a fresh locker"} {
		if !strings.Contains(errb, want) {
			t.Fatalf("the refusal does not say %q: %q", want, errb)
		}
	}
}
