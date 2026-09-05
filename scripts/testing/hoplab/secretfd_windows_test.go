//go:build windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/crypto"
)

// helperProcessEnv marks a re-exec of this test binary as the helper
// process for the two tests below, the same "re-exec myself" pattern
// $GOROOT/src/os/exec/exec_test.go's TestHelperProcess uses: running the
// suite normally never sets it, so TestHelperProcessReadsRecoveryCodeFD
// below just skips.
const helperProcessEnv = "HOPLAB_WANT_HELPER_PROCESS"

// TestHelperProcessReadsRecoveryCodeFD is not a real test of this package:
// it is the child process pair.go's runAccountInit/runAccountRecover
// spawn, played by the *real* product function
// (crypto.ReadSecretFD(crypto.RecoveryCodeFDEnv), the exact call
// internal/cli/account.go's accountSeams.readRecoveryCode makes) instead
// of a hand-rolled stand-in, so the two tests below prove the actual
// contract: what internal/crypto's Windows FD duplication
// (passphrase_fd_windows.go) reads back is exactly what
// secretfd_windows.go's parent-side handle-passing sent, across a real
// process boundary, not just within one.
func TestHelperProcessReadsRecoveryCodeFD(t *testing.T) {
	if os.Getenv(helperProcessEnv) != "1" {
		t.Skip("only runs when re-exec'd as a helper process")
	}
	secret, configured, err := crypto.ReadSecretFD(crypto.RecoveryCodeFDEnv)
	if !configured {
		os.Stderr.WriteString("REINSTATE_RECOVERY_CODE_FD was not configured\n")
		os.Exit(2)
	}
	if err != nil {
		os.Stderr.WriteString("ReadSecretFD: " + err.Error() + "\n")
		os.Exit(3)
	}
	os.Stdout.Write(secret)
	os.Exit(0)
}

// applier is the subset of liveSecretFD/fixedSecretFD runHelper needs: the
// env value and the SysProcAttr wiring that actually makes it inherited
// (see secretfd_windows.go's applyInheritedHandle doc comment).
type applier interface {
	ChildValue() string
	ApplyTo(*exec.Cmd)
}

func runHelper(t *testing.T, fd applier) (stdout, stderr string, err error) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^TestHelperProcessReadsRecoveryCodeFD$")
	cmd.Env = append(os.Environ(),
		helperProcessEnv+"=1",
		crypto.RecoveryCodeFDEnv+"="+fd.ChildValue(),
	)
	fd.ApplyTo(cmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// TestLiveSecretFDReachesAChildProcessAfterItStarts proves the harder
// half of the pipe contract pair.go's runAccountInit relies on: the value
// fed into the pipe *after* the child has already started (as it must be
// -- `rein account init` generates its own code, pair.go cannot know it in
// advance) still reaches the child's blocking read.
func TestLiveSecretFDReachesAChildProcessAfterItStarts(t *testing.T) {
	fd, err := newLiveSecretFD()
	if err != nil {
		t.Fatalf("newLiveSecretFD: %v", err)
	}
	defer fd.Close()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^TestHelperProcessReadsRecoveryCodeFD$")
	cmd.Env = append(os.Environ(),
		helperProcessEnv+"=1",
		crypto.RecoveryCodeFDEnv+"="+fd.ChildValue(),
	)
	fd.ApplyTo(cmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	fd.CloseReadInParent()

	const want = "LIVE-TEST-CODE-0001-EXTRA"
	if err := fd.Feed(want); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper process failed: %v (stdout: %q) (stderr: %s)", err, outBuf.String(), errBuf.String())
	}
	if got := outBuf.String(); got != want {
		t.Fatalf("helper read %q, want %q (stderr: %s)", got, want, errBuf.String())
	}
}

// TestFixedSecretFDReachesAChildProcess proves the simpler half:
// pair.go's runAccountRecover, which already knows the code before it
// spawns anything.
func TestFixedSecretFDReachesAChildProcess(t *testing.T) {
	const want = "FIXED-TEST-CODE-0002"
	fd, err := newFixedSecretFD(want)
	if err != nil {
		t.Fatalf("newFixedSecretFD: %v", err)
	}
	defer fd.Close()

	stdout, stderr, err := runHelper(t, fd)
	if err != nil {
		t.Fatalf("helper process failed: %v (stdout: %q) (stderr: %s)", err, stdout, stderr)
	}
	if stdout != want {
		t.Fatalf("helper read %q, want %q (stderr: %s)", stdout, want, stderr)
	}
}

// TestFixedSecretFDCleansUpItsFile pins that Close removes the temp file
// (never left behind in the OS temp directory across lab runs).
func TestFixedSecretFDCleansUpItsFile(t *testing.T) {
	fd, err := newFixedSecretFD("cleanup-check")
	if err != nil {
		t.Fatalf("newFixedSecretFD: %v", err)
	}
	path := fd.path
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temp file missing before Close: %v", err)
	}
	fd.Close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temp file survived Close: err=%v", err)
	}
}
