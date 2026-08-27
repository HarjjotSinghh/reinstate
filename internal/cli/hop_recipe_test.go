package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
)

// recipeSection is the heading in docs/hop/object-format.md whose shell
// this test runs. The page promises the verification is "reproducible step
// by step"; a recipe nobody has run is not that, and the one this page
// shipped could not work as printed: its first line was
// `eval "$(rein hop credentials | grep '^AWS_')"`, which sets shell
// variables and exports none of them, so every `aws` command after it ran
// with no credentials at all.
const recipeSection = "## Reproducing the checks by hand"

// TestByHandRecipeRunsAsPrinted runs that section's shell, as printed,
// against the in-process fake control plane and fake locker, and then
// makes the same requests the shell's `aws` lines describe.
//
// Two things stand in for parts that cannot run on a test machine, and
// nothing else is changed:
//
//   - `rein` is a shim that replays the output of a real, in-process
//     `rein hop credentials --export` for this journey. The output is the
//     command's own; only the process boundary is faked.
//   - `aws` is a shim that records the arguments and the environment it
//     was given, and serves the locker's real `manifest.age` bytes for the
//     two `s3 cp` lines. The AWS CLI is not a dependency of this
//     repository and is not installed on the bench, so its own argument
//     handling is the one part of the recipe this test does not cover.
//
// What the shim arrangement does cover is exactly what was broken: the
// credential line has to put the values in the *environment*, where a
// child process reads them. The shim prints what it inherited, so a
// recipe whose first line exports nothing fails this test.
//
// It runs twice: once on a locker with no key prefix, which is how Hop
// provisions them, and once on a locker whose record carries one. The
// second case is not hypothetical — `internal/hop.Locker` has the field
// and every client path honours it — and the recipe used to assert a Hop
// locker had no prefix while `--export` printed none, so on a prefixed
// locker the printed commands listed nothing and fetched nothing.
func TestByHandRecipeRunsAsPrinted(t *testing.T) {
	for _, prefix := range []string{"", "team/a"} {
		name := "no prefix"
		if prefix != "" {
			name = "prefix " + prefix
		}
		t.Run(name, func(t *testing.T) { byHandRecipe(t, prefix) })
	}
}

func byHandRecipe(t *testing.T, prefix string) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh on PATH; the recipe is shell and cannot be run here")
	}
	j := newLockerJourney(t)
	j.plane.lockerPrefix = prefix
	j, _ = hostedVerifyJourneyOn(t, j)
	// The form the export prints and the recipe pastes in front of a key:
	// empty, or ending in one slash.
	at := ""
	if prefix != "" {
		at = prefix + "/"
	}
	if _, errb, code := j.run("push", "--all"); code != ExitOK {
		t.Fatalf("push exit=%d err=%q", code, errb)
	}

	// The recipe's first line, run for real. Everything below is driven by
	// what it printed.
	exported, errb, code := j.run("hop", "credentials", "--export")
	if code != ExitOK {
		t.Fatalf("hop credentials --export exit=%d out=%q err=%q", code, exported, errb)
	}
	akid := j.akid()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "aws.log")
	manifestPath := filepath.Join(dir, "manifest.age")
	if err := os.WriteFile(manifestPath, j.object(at+"manifest.age"), 0o600); err != nil {
		t.Fatal(err)
	}
	shimDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeShim(t, filepath.Join(shimDir, "rein"), "#!/bin/sh\ncat "+shellQuote(filepath.Join(dir, "credentials.sh"))+"\n")
	if err := os.WriteFile(filepath.Join(dir, "credentials.sh"), []byte(exported), 0o600); err != nil {
		t.Fatal(err)
	}
	// The shim records one "argv" line and one "env" line per invocation,
	// then serves the object body for the two `s3 cp` lines.
	writeShim(t, filepath.Join(shimDir, "aws"), `#!/bin/sh
{
  printf 'argv'
  for a in "$@"; do printf '\t%s' "$a"; done
  printf '\n'
  printf 'env\t%s\t%s\t%s\t%s\n' "$AWS_ACCESS_KEY_ID" "$AWS_SECRET_ACCESS_KEY" "$AWS_SESSION_TOKEN" "$AWS_ENDPOINT_URL"
} >> "$REIN_TEST_SHIM_LOG"
for a in "$@"; do
  case "$a" in
    *manifest.age) cat "$REIN_TEST_MANIFEST" ;;
  esac
done
`)

	script := recipeShell(t)
	// The two values the page tells the reader to paste in from
	// `rein sync verify` step 4's detail line.
	script = strings.ReplaceAll(script, "paste-the-reference-bucket-here", j.plane.reference.bucket)
	script = strings.ReplaceAll(script, "paste-the-probe-key-here", j.plane.reference.key)

	command := exec.Command(shell, "-c", script)
	command.Dir = dir
	command.Env = append(withoutAWSEnvironment(),
		"PATH="+shimDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"REIN_TEST_SHIM_LOG="+logPath,
		"REIN_TEST_MANIFEST="+manifestPath,
	)
	var stderr strings.Builder
	command.Stderr = &stderr
	stdout, err := command.Output()
	if err != nil {
		// grep -c exits 1 when it finds nothing, which is the documented
		// outcome of the last line, so a non-zero status from the script as
		// a whole is expected; a shell syntax error is not.
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("running the recipe: %v\n%s", err, stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "not found") || strings.Contains(stderr.String(), "syntax error") {
		t.Fatalf("the recipe does not run as printed:\n%s", stderr.String())
	}
	if !strings.Contains(string(stdout), "age-encryption.org/v1") {
		t.Fatalf("the recipe's step 2 did not print the age header:\n%s", stdout)
	}
	if !hasLine(string(stdout), "0") {
		t.Fatalf(`the recipe's plaintext-field count did not print 0:\n%s`, stdout)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("the recipe ran no aws command at all: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(raw), "\r\n", "\n")), "\n")
	invocations, envs := 0, 0
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		switch fields[0] {
		case "argv":
			invocations++
		case "env":
			envs++
			// This is the whole of the bug the recipe had: a child process
			// sees exported variables and nothing else.
			if len(fields) != 5 || fields[1] != akid || fields[2] == "" || fields[3] == "" || fields[4] != j.plane.s3.URL() {
				t.Fatalf("aws invocation %d did not inherit the credentials the recipe printed: %q", envs, line)
			}
		}
	}
	// One listing, two reads of the index, and step 4's two requests
	// against the reference locker.
	if invocations != 5 {
		t.Fatalf("the recipe ran %d aws command(s), want 5:\n%s", invocations, raw)
	}
	log := string(raw)
	for _, want := range []string{j.plane.locker.bucket, j.plane.reference.bucket, j.plane.reference.key, at + "manifest.age"} {
		if !strings.Contains(log, want) {
			t.Fatalf("no aws command named %q:\n%s", want, log)
		}
	}
	// The prefix reaches the commands the way the page says it does: as
	// --prefix on the listing, and in front of the key on both reads. A
	// recipe that dropped it would list an empty bucket and fetch nothing.
	if !strings.Contains(log, "--prefix\t"+at) {
		t.Fatalf("the listing did not carry --prefix %q:\n%s", at, log)
	}

	// The same requests, made for real against the fake locker with the
	// credentials the recipe's first line printed and nothing else. The
	// shell above shows the recipe is followable; this shows the values it
	// hands out do what the page says they do.
	env := shellEnvironment(t, shell, filepath.Join(dir, "credentials.sh"))
	ctx := context.Background()
	locker, err := s3.New(ctx, s3.Config{
		Endpoint: env["AWS_ENDPOINT_URL"], Region: env["AWS_REGION"], Bucket: env["REIN_LOCKER_BUCKET"],
		Credentials: s3.StaticCredentials(s3.Credentials{
			AccessKeyID: env["AWS_ACCESS_KEY_ID"], SecretAccessKey: env["AWS_SECRET_ACCESS_KEY"], SessionToken: env["AWS_SESSION_TOKEN"],
		}),
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	objects, err := locker.List(ctx, strings.TrimSuffix(env["REIN_LOCKER_PREFIX"], "/"))
	if err != nil {
		t.Fatalf("step 1 by hand: %v", err)
	}
	var keys []string
	for _, o := range objects {
		keys = append(keys, o.Key)
	}
	for _, want := range []string{at + "manifest.age", at + "keyring.v1.json"} {
		if !strings.Contains(strings.Join(keys, " "), want) {
			t.Fatalf("step 1 by hand listed %v, without %s", keys, want)
		}
	}
	body := readAll(t, locker, env["REIN_LOCKER_PREFIX"]+"manifest.age")
	if head := string(body[:22]); head != "age-encryption.org/v1\n" {
		t.Fatalf("step 2 by hand: first 22 bytes %q", head)
	}
	if strings.Contains(string(body), `"sessions"`) {
		t.Fatal(`step 2 by hand: the index body contains "sessions"`)
	}

	reference, err := s3.New(ctx, s3.Config{
		Endpoint: env["AWS_ENDPOINT_URL"], Region: env["AWS_REGION"], Bucket: j.plane.reference.bucket,
		Credentials: s3.StaticCredentials(s3.Credentials{
			AccessKeyID: env["AWS_ACCESS_KEY_ID"], SecretAccessKey: env["AWS_SECRET_ACCESS_KEY"], SessionToken: env["AWS_SESSION_TOKEN"],
		}),
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reference.List(ctx, ""); !isAccessDenied(err) {
		t.Fatalf("step 4 by hand: listing the reference locker answered %v, want access denied", err)
	}
	if _, _, err := reference.Get(ctx, j.plane.reference.key); !isAccessDenied(err) {
		t.Fatalf("step 4 by hand: reading the probe answered %v, want access denied", err)
	}
}

// isAccessDenied is the answer step 4 is built on: the endpoint knew the
// credential and refused the bucket, not the credential.
func isAccessDenied(err error) bool {
	return errors.Is(err, backend.ErrAccessDenied) && !errors.Is(err, backend.ErrCredentialRejected)
}

func readAll(t *testing.T, b backend.Backend, key string) []byte {
	t.Helper()
	rc, _, err := b.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get %s: %v", key, err)
	}
	defer func() { _ = rc.Close() }()
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// recipeShell is every fenced shell block under the recipe heading, joined
// in the order the page prints them.
func recipeShell(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "docs", "hop", "object-format.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := strings.ReplaceAll(string(raw), "\r\n", "\n")
	_, section, found := strings.Cut(page, recipeSection)
	if !found {
		t.Fatalf("docs/hop/object-format.md has no %q section", recipeSection)
	}
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}
	var blocks []string
	for rest := section; ; {
		_, after, ok := strings.Cut(rest, "```bash\n")
		if !ok {
			break
		}
		block, remainder, ok := strings.Cut(after, "```")
		if !ok {
			t.Fatal("unterminated shell block in the recipe section")
		}
		blocks = append(blocks, block)
		rest = remainder
	}
	if len(blocks) != 2 {
		t.Fatalf("the recipe section holds %d shell block(s), want 2 (steps 1-2 and step 4)", len(blocks))
	}
	script := strings.Join(blocks, "\n")
	if !strings.Contains(script, `eval "$(rein hop credentials --export)"`) {
		t.Fatalf("the recipe does not export the credentials it prints:\n%s", script)
	}
	return script
}

// shellEnvironment evaluates the exported credentials the way the recipe
// does and returns what a child process would see.
func shellEnvironment(t *testing.T, shell, credentials string) map[string]string {
	t.Helper()
	command := exec.Command(shell, "-c", `eval "$(cat "$1")"; env`, "sh", credentials)
	command.Env = withoutAWSEnvironment()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("evaluating the printed credentials: %v", err)
	}
	env := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if name, value, ok := strings.Cut(line, "="); ok {
			env[name] = value
		}
	}
	// REIN_LOCKER_PREFIX is not here: it is legitimately empty on a locker
	// with no prefix, so it is checked below for presence instead.
	for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_ENDPOINT_URL", "AWS_REGION", "AWS_DEFAULT_REGION", "REIN_LOCKER_BUCKET"} {
		if env[name] == "" {
			t.Fatalf("%s did not reach a child process; the printed credentials are assignments, not exports:\n%s", name, out)
		}
	}
	if _, ok := env["REIN_LOCKER_PREFIX"]; !ok {
		t.Fatalf("REIN_LOCKER_PREFIX did not reach a child process; the recipe pastes it in front of every key:\n%s", out)
	}
	return env
}

// withoutAWSEnvironment is this machine's environment with every name the
// recipe sets removed, so the only credentials in the shell are the ones
// the recipe's first line puts there.
//
// Removing them matters, and blanking them is not the same thing: a shell
// assignment to a name that is *already* exported stays exported, so an
// environment pre-seeded with empty AWS_* values would let a recipe that
// exports nothing pass this test on a machine that happens to have them
// set, and fail on one that does not.
func withoutAWSEnvironment() []string {
	kept := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "AWS_") || strings.HasPrefix(name, "REIN_LOCKER_") {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

func writeShim(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func hasLine(out, want string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}
