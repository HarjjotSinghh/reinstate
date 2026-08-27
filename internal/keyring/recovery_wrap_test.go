package keyring

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// mutateRecoveryWrap rewrites one field of one generation's recovery wrap
// in the marshalled object, leaving the signature as the account wrote it.
func mutateRecoveryWrap(t *testing.T, k *Keyring, generation int, f func(map[string]any)) []byte {
	t.Helper()
	raw, err := k.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	gens, ok := obj["generations"].([]any)
	if !ok {
		t.Fatalf("generations is %T", obj["generations"])
	}
	for _, entry := range gens {
		g, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if int(g["number"].(float64)) != generation {
			continue
		}
		wrap, ok := g["recovery"].(map[string]any)
		if !ok {
			t.Fatalf("recovery is %T", g["recovery"])
		}
		f(wrap)
	}
	out, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// flipLastByte changes one byte of a base64 field, which is what a party
// with write access to the bucket does when it cannot forge a signature but
// can still damage the object.
func flipLastByte(t *testing.T, encoded string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("nothing to flip")
	}
	raw[len(raw)-1] ^= 0x01
	return base64.StdEncoding.EncodeToString(raw)
}

// TestATamperedRecoveryWrapIsNotReportedAsAMistypedCode is the fix for the
// finding this format version exists for.
//
// Under keyring format 4 the recovery wrap was outside the generation
// signature. A party with write access flipped one byte of the ciphertext
// and every later `rein account recover` told the person their recovery
// code was wrong — the single worst thing to say at the moment whose whole
// job is getting the code right. Under format 5 the wrap's parameters, salt
// and ciphertext are inside the signature, so the same edit is refused as
// what it is, by Parse, before any code is asked for.
//
// Falsified by removing the recovery fields from generationMessage: every
// case here then parses, and the ciphertext case reports ErrRecoveryMismatch
// against a correct code.
func TestATamperedRecoveryWrapIsNotReportedAsAMistypedCode(t *testing.T) {
	cases := map[string]struct {
		mutate func(map[string]any)
		// bySignature marks the fields only the signature covers. The wrap
		// format has a structural rule of its own in Parse and is refused
		// by that first; it is in the table so the field cannot lose both
		// guards at once without a failure here.
		bySignature bool
	}{
		"the ciphertext": {func(w map[string]any) { w["wrap"] = flipLastByteOf(t, w, "wrap") }, true},
		"the salt":       {func(w map[string]any) { w["salt"] = flipLastByteOf(t, w, "salt") }, true},
		"the kdf name":   {func(w map[string]any) { w["kdf"] = "argon2i" }, true},
		"the time cost":  {func(w map[string]any) { w["time"] = w["time"].(float64) + 1 }, true},
		"the memory":     {func(w map[string]any) { w["memory_kib"] = w["memory_kib"].(float64) * 2 }, true},
		"the threads":    {func(w map[string]any) { w["threads"] = w["threads"].(float64) + 1 }, true},
		"the format":     {func(w map[string]any) { w["format"] = float64(1) }, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			k, next := rolledOverPair(t)
			defer crypto.Zero(next)
			// Sound to start with, under both the code and Parse.
			sound, err := k.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(sound); err != nil {
				t.Fatalf("the unmodified keyring does not parse: %v", err)
			}
			for _, generation := range k.GenerationNumbers() {
				raw := mutateRecoveryWrap(t, k, generation, tc.mutate)
				parsed, err := Parse(raw)
				if err == nil {
					t.Fatalf("generation %d: Parse accepted an edited recovery wrap", generation)
				}
				if parsed != nil {
					t.Fatalf("generation %d: Parse returned a keyring alongside its refusal", generation)
				}
				if tc.bySignature && !errors.Is(err, ErrUnauthenticatedGeneration) {
					t.Fatalf("generation %d: Parse = %v, want an unauthenticated-generation refusal", generation, err)
				}
				if strings.Contains(err.Error(), "recovery code") {
					t.Fatalf("generation %d: the refusal blames the recovery code: %v", generation, err)
				}
			}
		})
	}
}

// flipLastByteOf is flipLastByte for a field inside the decoded wrap map.
func flipLastByteOf(t *testing.T, wrap map[string]any, field string) string {
	t.Helper()
	s, ok := wrap[field].(string)
	if !ok {
		t.Fatalf("%s is %T", field, wrap[field])
	}
	return flipLastByte(t, s)
}

// TestRecoveryWrapShapesAreNotACodeVerdict: a caller that meets a damaged
// wrap without going through Parse — the shapes below are refused there
// now, but the unwrap is a public entry point — still gets an error that
// says the wrap is malformed rather than one that says the code is wrong.
// The distinction is what `rein account recover` prints, so it is pinned
// here rather than left to the message text.
func TestRecoveryWrapShapesAreNotACodeVerdict(t *testing.T) {
	sound, err := wrapUnderRecoveryCode(goldenRootKey(), goldenRecoveryCode, binding{profileID: goldenProfileID, generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	bind := binding{profileID: goldenProfileID, generation: 1}
	if _, err := unwrapWithRecoveryCode(sound, goldenRecoveryCode, bind); err != nil {
		t.Fatalf("the sound wrap does not open: %v", err)
	}

	cases := map[string]struct {
		mutate func(RecoveryWrap) RecoveryWrap
		want   error
	}{
		"unknown kdf": {func(w RecoveryWrap) RecoveryWrap { w.KDF = "scrypt"; return w }, ErrRecoveryWrapMalformed},
		"unknown format": {func(w RecoveryWrap) RecoveryWrap {
			w.Format = WrapFormatBound + 1
			return w
		}, ErrRecoveryWrapMalformed},
		"salt not base64": {func(w RecoveryWrap) RecoveryWrap { w.Salt = "not base64!"; return w }, ErrRecoveryWrapMalformed},
		"wrap not base64": {func(w RecoveryWrap) RecoveryWrap { w.Wrap = "not base64!"; return w }, ErrRecoveryWrapMalformed},
		"wrap truncated": {func(w RecoveryWrap) RecoveryWrap {
			w.Wrap = base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
			return w
		}, ErrRecoveryWrapMalformed},
		"ciphertext flip":  {func(w RecoveryWrap) RecoveryWrap { w.Wrap = flipLastByte(t, w.Wrap); return w }, ErrRecoveryMismatch},
		"a different code": {func(w RecoveryWrap) RecoveryWrap { return w }, ErrRecoveryMismatch},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code := goldenRecoveryCode
			if name == "a different code" {
				other, err := GenerateRecoveryCode()
				if err != nil {
					t.Fatal(err)
				}
				code = other
			}
			_, err := unwrapWithRecoveryCode(tc.mutate(sound), code, bind)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			// The two must stay distinguishable: a malformed wrap is never
			// also a code verdict, and vice versa.
			if errors.Is(err, ErrRecoveryWrapMalformed) && errors.Is(err, ErrRecoveryMismatch) {
				t.Fatalf("the two answers are not separable: %v", err)
			}
		})
	}
}

// TestRolloverSignsTheRecoveryWrapItWrites pins the ordering inside
// Rollover: the new generation's recovery wrap must be in place before the
// signature is produced, or every rollover would ship an object that does
// not parse.
func TestRolloverSignsTheRecoveryWrapItWrites(t *testing.T) {
	a, b := goldenIdentities(t)
	k, err := New(goldenProfileID, goldenRootKey(), goldenRecoveryCode, goldenDeviceID, a, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Enrol(goldenRootKey(), goldenDeviceBID, b.Recipient(), time.Now()); err != nil {
		t.Fatal(err)
	}
	for round := 0; round < 3; round++ {
		next, err := k.Rollover(goldenRootKey(), goldenRecoveryCode, []string{goldenDeviceBID}, goldenDeviceID, time.Now())
		if round == 0 {
			if err != nil {
				t.Fatal(err)
			}
			crypto.Zero(next)
			continue
		}
		// Later rounds have nothing left to revoke; the first is what this
		// test is about.
		if err == nil {
			crypto.Zero(next)
		}
		break
	}
	raw, err := k.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("a rolled-over keyring does not parse: %v", err)
	}
	if parsed.CurrentGeneration != 2 {
		t.Fatalf("current generation %d", parsed.CurrentGeneration)
	}
	for _, n := range parsed.GenerationNumbers() {
		key, err := unwrapWithRecoveryCode(parsed.generation(n).Recovery, goldenRecoveryCode, parsed.bindingFor(parsed.generation(n)))
		if err != nil {
			t.Fatalf("generation %d does not open under the recovery code: %v", n, err)
		}
		crypto.Zero(key)
	}
}
