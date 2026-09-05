// pair.go drives the real `rein` binary through the two account-pairing
// commands the Hop lab's "two homes" promise depends on: `rein account
// init` on the first device (which generates and must immediately confirm
// a fresh recovery code) and `rein account recover` on every device after
// it (which needs that code already known). This is the same command
// sequence internal/cli/keygeneration_crossplane_test.go
// (-tags hopacceptance, run against a real hopd) proves works: `login`,
// `init --hop --project ID=path`, `account init` on the first device;
// `login` (same email -- one hopd account), `init --hop`, `account
// recover` with the first device's code, on every device after it.
//
// Both account commands read their secret from REINSTATE_RECOVERY_CODE_FD
// when it is set (crypto.ReadSecretFD, internal/crypto/passphrase.go) --
// automation's documented alternative to a hidden terminal prompt
// (account.go's own doc comment: "the recovery code is read from a hidden
// terminal prompt or the documented descriptor"). That matters here
// because hoplab drives the compiled binary as a real subprocess, not the
// in-process test harness (hop_first_push_test.go's hopDevice), so there
// is no RecoveryCodePrompt seam to hook, and a caller with no real
// terminal -- an agent driving this through a piped shell, exactly the
// verifier that rejected the previous round of this branch -- cannot
// answer a hidden prompt at all. See secretfd_windows.go for the Windows
// handle-passing half of that contract.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
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

func cmdPair(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("want init|join")
	}
	action, rest := argv[0], argv[1:]
	fs := flag.NewFlagSet("pair "+action, flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (matches -root given to `hoplab homes`)")
	device := fs.String("device", "", "device name (matches a name given to `hoplab homes -devices`)")
	reinBin := fs.String("rein", os.Getenv("REINSTATE_REIN_BIN"), "path to the rein/reinstate binary under test (env REINSTATE_REIN_BIN; default: bin/rein.exe or bin/reinstate.exe under the repo root)")
	timeout := fs.Duration("timeout", 30*time.Second, "give up waiting on the rein subprocess after this long")
	code := fs.String("code", "", "join only: the recovery code from `pair init`; default: read back from hoplab-state.json")
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
		fmt.Fprintf(os.Stderr, "hoplab: %s initialized the account; recovery code saved to %s for `pair join`\n", *device, statePath(*root))
		fmt.Println(recoveryCode)
		return nil
	case "join":
		joinCode := strings.TrimSpace(*code)
		if joinCode == "" {
			joinCode = strings.TrimSpace(s.PairingRecoveryCode)
		}
		if joinCode == "" {
			return fmt.Errorf("no recovery code: pass -code, or run `hoplab pair init -root %s -device <first-device>` first", *root)
		}
		if err := pairJoin(ctx, bin, s, h, joinCode); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "hoplab: %s enrolled from the recovery code; it now shares the account `pair init` initialized\n", *device)
		return nil
	default:
		return fmt.Errorf("want init|join, got %q", action)
	}
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

// pairJoin runs `rein init --hop` then `rein account recover` for h -- a
// later device -- against an already-known recovery code from a prior
// pairInit.
func pairJoin(ctx context.Context, bin string, s LabState, h DeviceHome, code string) error {
	env := reinEnviron(s, h)
	if _, _, err := runRein(ctx, bin, env, "init", "--hop", "--project", projectMapping(h)); err != nil {
		return fmt.Errorf("rein init --hop: %w", err)
	}
	if _, err := runAccountRecover(ctx, bin, env, code); err != nil {
		return fmt.Errorf("rein account recover: %w", err)
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
// HOME/USERPROFILE below), with hopLabEnv's isolation block overlaid.
func reinEnviron(s LabState, h DeviceHome) []string {
	return mergeEnv(os.Environ(), hopLabEnv(s, h))
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
