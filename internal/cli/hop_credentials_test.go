package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestHopCredentialsMakesTheByHandRecipeReal: `docs/hop.md` promises that
// every step of `rein sync verify` can be repeated by hand, and
// `docs/hop/object-format.md` ships the S3 recipe. On a Hop locker none
// of it could be run, because the only credentials that reach the locker
// are the hourly ones the control plane mints and no command exposed
// them. This is the command that makes steps 1, 2 and 4 followable.
func TestHopCredentialsMakesTheByHandRecipeReal(t *testing.T) {
	j, _ := hostedVerifyJourney(t)
	before := len(j.plane.mints)

	out, errb, code := j.run("hop", "credentials")
	if code != ExitOK {
		t.Fatalf("hop credentials exit=%d out=%q err=%q", code, out, errb)
	}
	minted := j.akid()
	if len(j.plane.mints) != before+1 {
		t.Fatalf("mints %d, want %d: the command must mint rather than reuse", len(j.plane.mints), before+1)
	}
	for _, want := range []string{
		"Locker:  lk-0000000000000000000000test at " + j.plane.s3.URL() + " (region auto)",
		"AWS_ACCESS_KEY_ID=" + minted,
		"AWS_SECRET_ACCESS_KEY=secret-" + minted,
		"AWS_SESSION_TOKEN=session-" + minted,
		"AWS_ENDPOINT_URL=" + j.plane.s3.URL(),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	// The caution is on stderr, so the values can be redirected without it.
	if strings.Contains(out, "note:") || !strings.Contains(errb, "reach this account's own bucket and no other") {
		t.Fatalf("the caution is in the wrong stream: out=%q err=%q", out, errb)
	}

	out, _, code = j.run("hop", "credentials", "--json")
	if code != ExitOK {
		t.Fatalf("hop credentials --json exit=%d out=%q", code, out)
	}
	var creds struct {
		AccessKeyID     string `json:"access_key_id"`
		SecretAccessKey string `json:"secret_access_key"`
		SessionToken    string `json:"session_token"`
		ExpiresAt       string `json:"expires_at"`
		Endpoint        string `json:"endpoint"`
		Bucket          string `json:"bucket"`
		Region          string `json:"region"`
	}
	if err := json.Unmarshal([]byte(out), &creds); err != nil {
		t.Fatalf("json %q: %v", out, err)
	}
	if creds.AccessKeyID != j.akid() || creds.SecretAccessKey == "" || creds.SessionToken == "" {
		t.Fatalf("credentials %+v", creds)
	}
	if creds.Bucket != "lk-0000000000000000000000test" || creds.Endpoint != j.plane.s3.URL() || creds.Region != "auto" {
		t.Fatalf("the coordinates an S3 client needs are missing: %+v", creds)
	}
	if creds.ExpiresAt == "" {
		t.Fatal("nothing says when the credential dies")
	}

	// The credential is real: the objects the recipe lists can be listed
	// with it, and the fake S3 only ever accepts the newest mint.
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push after minting by hand exit=%d err=%q", code, errb)
	}
}

// TestHopCredentialsNeedsASignedInDevice: the command reaches the control
// plane for a fresh mint, so it fails the way every other hosted command
// does rather than printing an empty credential.
func TestHopCredentialsNeedsASignedInDevice(t *testing.T) {
	j := newLockerJourney(t)
	out, errb, code := j.run("hop", "credentials")
	if code != ExitAuthStorage || !strings.Contains(errb, "not signed in") {
		t.Fatalf("exit=%d out=%q err=%q", code, out, errb)
	}
	if strings.Contains(out, "AWS_") {
		t.Fatalf("credentials printed without a sign-in:\n%s", out)
	}
}
