package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
	"github.com/HarjjotSinghh/reinstate/internal/config"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// TestEnrolmentRecordMustCarryTheAnchor: the local record is what tells this
// account's keyring from a replacement, and it can only do that if it
// carries the account signing key and the recipient of the generation this
// device last unwrapped. A record written by an earlier build carried
// neither and was silently accepted, which skipped the anchor entirely — so
// the schema version is required, every anchor field is required, and
// deleting one refuses the command rather than reaching the unanchored path.
func TestEnrolmentRecordMustCarryTheAnchor(t *testing.T) {
	plane := newFakeControlPlane(t)
	plane.s3 = s3test.NewPlain(t, "lk-0000000000000000000anchor")
	t.Setenv(hopURLEnv, plane.srv.URL)
	for _, env := range []string{"REINSTATE_BACKEND", "REINSTATE_S3_ACCESS_KEY_ID", "REINSTATE_S3_SECRET_ACCESS_KEY", "REINSTATE_PASSPHRASE_FD", "REINSTATE_RECOVERY_CODE_FD", "REINSTATE_PAIRING_CODE_FD", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	project := writeClaudeFixture(t)
	a := newPairDevice(t, plane, "macbook")
	for _, args := range [][]string{{"login"}, {"init", "--hop", "--project", "local/locker=" + project}, {"account", "init"}, {"push", "--all", "--json"}} {
		if out, errb, code := a.run(args...); code != ExitOK {
			t.Fatalf("A %v: exit=%d out=%q err=%q", args, code, out, errb)
		}
	}
	account, err := config.LoadAccount(a.home)
	if err != nil {
		t.Fatal(err)
	}
	if account.SchemaVersion != schema.AccountSchemaVersion || account.AccountKey == "" || account.KeyRecipient == "" {
		t.Fatalf("a fresh enrolment did not record the anchor: %+v", account)
	}
	path := config.AccountPath(a.home)
	sound, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rewrite := func(t *testing.T, f func(map[string]any)) {
		t.Helper()
		var obj map[string]any
		if err := json.Unmarshal(sound, &obj); err != nil {
			t.Fatal(err)
		}
		f(obj)
		out, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]func(map[string]any){
		"an earlier schema version": func(o map[string]any) { o["schema_version"] = float64(1) },
		"no account key":            func(o map[string]any) { delete(o, "account_key") },
		"no key recipient":          func(o map[string]any) { delete(o, "key_recipient") },
		"an empty account key":      func(o map[string]any) { o["account_key"] = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			rewrite(t, mutate)
			defer func() {
				if err := os.WriteFile(path, sound, 0o600); err != nil {
					t.Fatal(err)
				}
			}()
			out, errb, code := a.run("push", "--all", "--json")
			if code == ExitOK {
				t.Fatalf("push accepted an enrolment record with no anchor: out=%q", out)
			}
			// The remedy has to be in the message: nothing else in the CLI
			// removes an enrolment record.
			if !strings.Contains(errb, "rein init --hop --force") {
				t.Fatalf("the refusal does not name the remedy: %q", errb)
			}
		})
	}
	// And the sound record still works, so the refusal is about the anchor
	// and not about the file being touched at all.
	if out, errb, code := a.run("push", "--all", "--json"); code != ExitOK {
		t.Fatalf("push with the record restored: exit=%d out=%q err=%q", code, out, errb)
	}
}
