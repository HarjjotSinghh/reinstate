// pair.go drives the real `rein` binary through account pairing, two ways:
//
//   - `pair init`/`pair recover`: `rein account init` on the first device
//     (which generates and must immediately confirm a fresh recovery code)
//     and `rein account recover` on every device after it (which needs
//     that code already known). This is the same command sequence
//     internal/cli/keygeneration_crossplane_test.go (-tags hopacceptance,
//     run against a real hopd) proves works: `login`, `init --hop
//     --project ID=path`, `account init` on the first device; `login`
//     (same email -- one hopd account), `init --hop`, `account recover`
//     with the first device's code, on every device after it.
//   - `pair join`: the live, no-recovery-code path -- the joining device
//     runs `rein account join` (which publishes a pairing request, prints
//     a short code, and blocks), and an already-enrolled device runs `rein
//     devices approve` fed that code. This is
//     internal/cli/pairing_test.go's own two-device journey
//     (startJoin/approve), driven against real compiled binaries instead
//     of the in-process test harness. Use `pair recover` only when no
//     second device is available to approve.
//
// Both flows read their secret from an FD-based automation seam rather
// than a hidden terminal prompt: `account init`/`account recover` from
// REINSTATE_RECOVERY_CODE_FD, `devices approve` from
// REINSTATE_PAIRING_CODE_FD (crypto.ReadSecretFD,
// internal/crypto/passphrase.go -- the product's own documented
// automation path, not something this package invented; account.go's own
// doc comment: "the recovery code is read from a hidden terminal prompt or
// the documented descriptor"). That matters here because hoplab drives the
// compiled binary as a real subprocess, not the in-process test harness
// (hop_first_push_test.go's hopDevice, internal/cli/pairing_test.go's
// pairDevice), so there is no prompt-callback seam to hook, and a caller
// with no real terminal -- an agent driving this through a piped shell,
// exactly the verifier that rejected the previous round of this branch --
// cannot answer a hidden prompt at all. See secretfd_windows.go for the
// Windows handle-passing half of that contract.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// recoveryCodePattern matches `rein account init`'s printed code
// (keyring.RecoveryCodeFormat: 8 groups of 4 Crockford32 characters). It is
// the same pattern internal/cli/account_test.go's recoveryCodePattern uses
// against the in-process harness's captured stderr.
var recoveryCodePattern = regexp.MustCompile(`\b(?:[0-9A-Z]{4}-){7}[0-9A-Z]{4}\b`)

// pairingCodePattern matches `rein account join`'s printed pairing code
// (4 groups of 4 Crockford32 characters -- shorter than a recovery code,
// so the two never collide). The same pattern
// internal/cli/pairing_test.go's pairingCodePattern uses against the
// in-process harness's captured stderr.
var pairingCodePattern = regexp.MustCompile(`\b(?:[0-9A-Z]{4}-){3}[0-9A-Z]{4}\b`)

func pairUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: hoplab pair <init|join|recover> -root <dir> -device <name> [flags]

  init     first device: rein init --hop, then rein account init.
           Prints the recovery code and saves it to hoplab-state.json for
           a later 'pair recover'.
  join     live device approval, no recovery code: -device runs rein init
           --hop then rein account join (publishes a pairing code and
           waits); -approver (an already-enrolled device) runs rein
           devices approve, fed that code non-interactively.
  recover  enrol -device from a recovery code (rein init --hop, then rein
           account recover): -code, or the code 'pair init' saved to
           hoplab-state.json when omitted. Use this only when no second
           device is available to approve live -- prefer 'pair join'.

flags:
  -root <dir>       lab root (matches -root given to 'hoplab homes')
  -device <name>    device to act as (matches a name 'hoplab homes' seeded)
  -approver <name>  join only: the already-enrolled device that approves
  -code <code>      recover only: override the saved recovery code
  -rein <path>      rein/reinstate binary under test (env REINSTATE_REIN_BIN;
                     default: bin/rein.exe or bin/reinstate.exe under the repo root)
  -timeout <dur>    give up waiting on the rein subprocess(es) after this long (default 30s)

Every device must have signed in under the same email first (rein login +
hoplab approve). See scripts/testing/hoplab/README.md for full usage.
`)
}

func cmdPair(argv []string) error {
	if len(argv) == 0 {
		pairUsage(os.Stderr)
		return fmt.Errorf("want init|join|recover")
	}
	if isHelpFlag(argv[0]) {
		pairUsage(os.Stderr)
		return nil
	}
	action, rest := argv[0], argv[1:]
	if action != "init" && action != "join" && action != "recover" {
		pairUsage(os.Stderr)
		return fmt.Errorf("want init|join|recover, got %q", action)
	}
	fs := flag.NewFlagSet("pair "+action, flag.ExitOnError)
	fs.Usage = func() { pairUsage(fs.Output()) }
	root := fs.String("root", "", "lab root directory (matches -root given to `hoplab homes`)")
	device := fs.String("device", "", "device name (matches a name given to `hoplab homes -devices`)")
	approver := fs.String("approver", "", "join only: the already-enrolled device that runs `rein devices approve`")
	reinBin := fs.String("rein", os.Getenv("REINSTATE_REIN_BIN"), "path to the rein/reinstate binary under test (env REINSTATE_REIN_BIN; default: bin/rein.exe or bin/reinstate.exe under the repo root)")
	timeout := fs.Duration("timeout", 30*time.Second, "give up waiting on the rein subprocess(es) after this long")
	code := fs.String("code", "", "recover only: the recovery code from `pair init`; default: read back from hoplab-state.json")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" || strings.TrimSpace(*device) == "" {
		return fmt.Errorf("%s needs -root and -device", action)
	}
	bin, err := resolveReinBin(*reinBin)
	if err != nil {
		return err
	}
	s, err := loadState(*root)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		s = LabState{Root: *root}
	}
	h := BuildDeviceHome(*root, *device)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch action {
	case "init":
		recoveryCode, err := pairInit(ctx, bin, s, h)
		if err != nil {
			return err
		}
		s.Root = *root
		s.PairingRecoveryCode = recoveryCode
		if err := s.save(); err != nil {
			return fmt.Errorf("save the recovery code to %s: %w", statePath(*root), err)
		}
		fmt.Fprintf(os.Stderr, "hoplab: %s initialized the account; recovery code saved to %s for `pair recover`\n", *device, statePath(*root))
		fmt.Println(recoveryCode)
		return nil
	case "join":
		approverName := strings.TrimSpace(*approver)
		if approverName == "" {
			return fmt.Errorf("join needs -approver (the already-enrolled device that will run `rein devices approve`)")
		}
		if approverName == *device {
			return fmt.Errorf("join needs -approver different from -device (a device cannot approve itself)")
		}
		approverHome := BuildDeviceHome(*root, approverName)
		if err := pairJoinLive(ctx, bin, s, h, approverHome); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "hoplab: %s joined the account live, approved by %s\n", *device, approverName)
		return nil
	case "recover":
		joinCode := strings.TrimSpace(*code)
		if joinCode == "" {
			joinCode = strings.TrimSpace(s.PairingRecoveryCode)
		}
		if joinCode == "" {
			return fmt.Errorf("no recovery code: pass -code, or run `hoplab pair init -root %s -device <first-device>` first", *root)
		}
		if err := pairRecover(ctx, bin, s, h, joinCode); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "hoplab: %s enrolled from the recovery code; it now shares the account `pair init` initialized\n", *device)
		return nil
	default:
		return fmt.Errorf("want init|join|recover, got %q", action)
	}
}

// isHelpFlag reports whether s is a request for usage rather than a
// subcommand action -- checked before any argument is treated as an
// action/device/etc., so `hoplab pair -h` and `hoplab keyring -h` print
// usage instead of falling through to "-h needs -root and -device" (a bare
// error, indistinguishable from a real mistake).
func isHelpFlag(s string) bool {
	return s == "-h" || s == "--help" || s == "help" || s == "-help"
}

// resolveReinBin finds the compiled rein/reinstate binary: an explicit
// path (or REINSTATE_REIN_BIN) wins; otherwise `make build`'s own output
// locations under the repository root (bin/rein.exe, bin/reinstate.exe --
// see the Makefile's `build` target).
func resolveReinBin(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("-rein / REINSTATE_REIN_BIN %s: %w", explicit, err)
		}
		return explicit, nil
	}
	repoRoot, err := findRepoRoot("")
	if err != nil {
		return "", err
	}
	for _, name := range []string{"rein.exe", "reinstate.exe", "rein", "reinstate"} {
		candidate := filepath.Join(repoRoot, "bin", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no rein/reinstate binary found under %s\\bin; pass -rein, set REINSTATE_REIN_BIN, or run `make build` first", repoRoot)
}

// pairInit runs `rein init --hop` then `rein account init` for h -- the
// first device of a lab account -- confirming the freshly generated
// recovery code through a live REINSTATE_RECOVERY_CODE_FD pipe (the code
// cannot be known before `account init` prints it) and returning that
// code.
func pairInit(ctx context.Context, bin string, s LabState, h DeviceHome) (string, error) {
	env := reinEnviron(s, h)
	if _, _, err := runRein(ctx, bin, env, "init", "--hop", "--project", projectMapping(h)); err != nil {
		return "", fmt.Errorf("rein init --hop: %w", err)
	}
	code, err := runAccountInit(ctx, bin, env)
	if err != nil {
		return "", fmt.Errorf("rein account init: %w", err)
	}
	return code, nil
}

// pairRecover runs `rein init --hop` then `rein account recover` for h --
// a later device -- against an already-known recovery code from a prior
// pairInit. Use pairJoinLive instead when a second, already-enrolled
// device is available: it needs no recovery code at all.
func pairRecover(ctx context.Context, bin string, s LabState, h DeviceHome, code string) error {
	env := reinEnviron(s, h)
	if _, _, err := runRein(ctx, bin, env, "init", "--hop", "--project", projectMapping(h)); err != nil {
		return fmt.Errorf("rein init --hop: %w", err)
	}
	if _, err := runAccountRecover(ctx, bin, env, code); err != nil {
		return fmt.Errorf("rein account recover: %w", err)
	}
	return nil
}

// pairJoinLive drives the live device-approval flow for h -- a device
// joining an account another device already initialized -- using the real
// rein binary as both participants: h runs `rein init --hop` then `rein
// account join` (which publishes a pairing request, prints a short code,
// and blocks waiting for it to be approved), and approver -- an
// already-enrolled device -- runs `rein devices approve` fed that same
// code through REINSTATE_PAIRING_CODE_FD. This is
// internal/cli/pairing_test.go's own two-device journey
// (startJoin/approveWhilePrompting), driven against real compiled
// binaries instead of the in-process test harness. Both devices must have
// signed in under the same email already (rein login + hoplab approve).
func pairJoinLive(ctx context.Context, bin string, s LabState, h, approver DeviceHome) error {
	joinerEnv := reinEnviron(s, h)
	if _, _, err := runRein(ctx, bin, joinerEnv, "init", "--hop", "--project", projectMapping(h)); err != nil {
		return fmt.Errorf("rein init --hop (%s): %w", h.Name, err)
	}
	code, wait, err := startAccountJoin(ctx, bin, joinerEnv)
	if err != nil {
		return fmt.Errorf("rein account join (%s): %w", h.Name, err)
	}
	approverEnv := reinEnviron(s, approver)
	if _, err := runDevicesApprove(ctx, bin, approverEnv, code); err != nil {
		// The joiner is still blocked in WaitForPairing; give it a chance
		// to notice the request was decided (or let ctx's own deadline
		// stop it) rather than leaving the child running unreaped.
		_ = wait(ctx)
		return fmt.Errorf("rein devices approve (%s): %w", approver.Name, err)
	}
	if err := wait(ctx); err != nil {
		return fmt.Errorf("rein account join (%s) did not finish after approval: %w", h.Name, err)
	}
	return nil
}

// projectMapping is the --project ID=/absolute/local/path value for h
// (internal/cli/commands_impl.go's parseProjectMapping): h's own distinct
// fixture project, so this device's Hop-mode config names the same project
// `hoplab homes` already seeded sessions under.
func projectMapping(h DeviceHome) string {
	return "hoplab-" + h.Name + "=" + h.Project
}

// reinEnviron is the OS environment a `rein` subprocess needs to act as h:
// the real process environment (PATH, SystemRoot, the OS user profile the
// OS keyring/DPAPI actually key off -- unrelated to the sandboxed
// HOME/USERPROFILE below) with every ambientOverrideEnv escape hatch
// stripped first (see env.go: an operator's own shell may carry
// REINSTATE_BACKEND=memory and REINSTATE_MEMORY_BACKEND_DIR from earlier,
// unrelated local testing, which would otherwise route this subprocess
// around the lab's real hopd/fakelocker entirely), then hopLabEnv's
// isolation block overlaid.
func reinEnviron(s LabState, h DeviceHome) []string {
	return mergeEnv(stripEnv(os.Environ(), ambientOverrideEnv), hopLabEnv(s, h))
}

// mergeEnv overlays overrides onto base ("KEY=VALUE" strings, like
// os.Environ()), replacing any existing entry for the same key. An
// override with an empty value is skipped instead of exported empty --
// REINSTATE_HOP_URL when no lab state names one, most notably -- so it is
// left exactly as base already had it (unset, or the caller's own).
func mergeEnv(base []string, overrides []envPair) []string {
	skip := make(map[string]bool, len(overrides))
	for _, o := range overrides {
		if o.Value != "" {
			skip[o.Key] = true
		}
	}
	out := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if skip[key] {
			continue
		}
		out = append(out, kv)
	}
	for _, o := range overrides {
		if o.Value == "" {
			continue
		}
		out = append(out, o.Key+"="+o.Value)
	}
	return out
}

// runRein runs bin with args and env, capturing stdout/stderr, for the
// ordinary (no hidden-prompt) commands in this file.
func runRein(ctx context.Context, bin string, env []string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout, stderr = outBuf.String(), errBuf.String()
	if runErr != nil {
		return stdout, stderr, fmt.Errorf("%w (stdout: %s) (stderr: %s)", runErr, strings.TrimSpace(stdout), strings.TrimSpace(stderr))
	}
	return stdout, stderr, nil
}

// runAccountInit runs `rein account init` with a live secret FD wired to
// REINSTATE_RECOVERY_CODE_FD, watching its stderr as it streams for the
// printed recovery code and feeding that same code back the moment it
// appears -- before the command has necessarily even reached the "Re-enter
// the recovery code" prompt, which is fine: a pipe buffers, so Feed does
// not need to race the child's own read call, only precede its EOF.
func runAccountInit(ctx context.Context, bin string, env []string) (string, error) {
	fd, err := newLiveSecretFD()
	if err != nil {
		return "", err
	}
	defer fd.Close()

	cmd := exec.CommandContext(ctx, bin, "account", "init")
	cmd.Env = append(append([]string{}, env...), "REINSTATE_RECOVERY_CODE_FD="+fd.ChildValue())
	fd.ApplyTo(cmd)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	fd.CloseReadInParent()

	var stderrBuf bytes.Buffer
	var code string
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		buf := make([]byte, 4096)
		fed := false
		for {
			n, readErr := stderrPipe.Read(buf)
			if n > 0 {
				stderrBuf.Write(buf[:n])
				if !fed {
					if m := recoveryCodePattern.FindString(stderrBuf.String()); m != "" {
						code = m
						fed = true
						_ = fd.Feed(code)
					}
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	waitErr := cmd.Wait()
	<-scanDone
	fd.Close() // idempotent: ensures EOF even if the code was never seen
	if waitErr != nil {
		return "", fmt.Errorf("%w (stdout: %s) (stderr: %s)", waitErr, strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()))
	}
	if code == "" {
		return "", fmt.Errorf("no recovery code seen in `rein account init`'s output (stderr: %s)", strings.TrimSpace(stderrBuf.String()))
	}
	return code, nil
}

// runAccountRecover runs `rein account recover` with a fixed secret FD
// carrying the already-known code -- no live streaming needed, since
// nothing here waits on the child to generate anything first.
func runAccountRecover(ctx context.Context, bin string, env []string, code string) (string, error) {
	fd, err := newFixedSecretFD(code)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	cmd := exec.CommandContext(ctx, bin, "account", "recover")
	cmd.Env = append(append([]string{}, env...), "REINSTATE_RECOVERY_CODE_FD="+fd.ChildValue())
	fd.ApplyTo(cmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w (stdout: %s) (stderr: %s)", err, strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), nil
}

// startAccountJoin starts `rein account join` in the background and
// returns once it has published a pairing request and shown the code on
// stderr -- the same moment internal/cli/pairing_test.go's tryStartJoin
// releases at. `account join` needs no secret fed in (unlike `account
// init`'s recovery-code confirmation): it generates the pairing salt and
// code itself and only blocks waiting for an approval, so the caller's
// job is to read the code, hand it to `rein devices approve` elsewhere,
// then call the returned wait function.
//
// wait must be called exactly once, whether or not approval succeeded --
// it is what reaps the child and reports its real exit error, and it
// tolerates being called after a failed approve (the joiner is still
// blocked; wait gives it until ctx to notice the request was decided).
func startAccountJoin(ctx context.Context, bin string, env []string) (code string, wait func(context.Context) error, err error) {
	cmd := exec.CommandContext(ctx, bin, "account", "join")
	cmd.Env = env
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", nil, err
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	if err := cmd.Start(); err != nil {
		return "", nil, err
	}

	codeCh := make(chan string, 1)
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		buf := make([]byte, 4096)
		found := false
		for {
			n, readErr := stderrPipe.Read(buf)
			if n > 0 {
				stderrBuf.Write(buf[:n])
				if !found {
					if m := pairingCodePattern.FindString(stderrBuf.String()); m != "" {
						found = true
						codeCh <- m
					}
				}
			}
			if readErr != nil {
				if !found {
					close(codeCh)
				}
				return
			}
		}
	}()

	reaped := false
	reap := func(waitCtx context.Context) error {
		if reaped {
			return nil
		}
		reaped = true
		waitErrCh := make(chan error, 1)
		go func() { <-scanDone; waitErrCh <- cmd.Wait() }()
		select {
		case werr := <-waitErrCh:
			if werr != nil {
				return fmt.Errorf("%w (stdout: %s) (stderr: %s)", werr, strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()))
			}
			return nil
		case <-waitCtx.Done():
			_ = cmd.Process.Kill()
			<-waitErrCh
			return waitCtx.Err()
		}
	}

	select {
	case c, ok := <-codeCh:
		if !ok {
			// The process ended (its stderr closed) before ever printing a
			// code -- reap it now so the real failure surfaces instead of
			// a generic "no code seen".
			return "", nil, reap(context.Background())
		}
		return c, reap, nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-scanDone
		_ = cmd.Wait()
		reaped = true
		return "", nil, ctx.Err()
	}
}

// runDevicesApprove runs `rein devices approve` with a fixed secret FD
// carrying the already-known pairing code the joining device just showed
// -- no live streaming needed, since nothing here waits on the child to
// generate anything first (compare runAccountInit, which does).
func runDevicesApprove(ctx context.Context, bin string, env []string, code string) (string, error) {
	fd, err := newFixedSecretFD(code)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	cmd := exec.CommandContext(ctx, bin, "devices", "approve")
	cmd.Env = append(append([]string{}, env...), "REINSTATE_PAIRING_CODE_FD="+fd.ChildValue())
	fd.ApplyTo(cmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w (stdout: %s) (stderr: %s)", err, strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), nil
}
