//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// handleFlagInherit is WinAPI's HANDLE_FLAG_INHERIT (winbase.h). The
// vendored golang.org/x/sys/windows used here does not export it.
const handleFlagInherit = 0x00000001

// applyInheritedHandle wires h into cmd so the child actually inherits it.
// Marking a handle inheritable (SetHandleInformation, below) is necessary
// but not sufficient: since Go added PROC_THREAD_ATTRIBUTE_HANDLE_LIST
// support (syscall.SysProcAttr.AdditionalInheritedHandles,
// $GOROOT/src/syscall/exec_windows.go, guarded by the comment "Do not
// accidentally inherit more than these handles"), CreateProcess only
// inherits the handles *named* in that list once any are given -- an
// otherwise-inheritable handle the child never asked for is no longer
// inherited by default the way a bare os/exec call used to. Confirmed the
// hard way: an earlier version of this file, without this call, produced
// "REINSTATE_RECOVERY_CODE_FD is unavailable" in the child even though
// SetHandleInformation alone looked correct and the parent-side plumbing
// was otherwise sound -- see secretfd_windows_test.go, which exercises
// this against a real child process specifically to catch a regression
// here again.
func applyInheritedHandle(cmd *exec.Cmd, h syscall.Handle) {
	cmd.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: []syscall.Handle{h}}
}

// liveSecretFD is a live, one-shot anonymous pipe wired for
// REINSTATE_RECOVERY_CODE_FD/REINSTATE_PAIRING_CODE_FD automation
// (internal/crypto/passphrase.go's ReadSecretFD): a real Windows HANDLE
// value the *child* inherits and reads from, blocking until EOF -- a pipe,
// unlike a file, has none until every write end closes -- so the value can
// be produced by the parent *after* the child has already started and
// asked for it. `rein account init` generates its recovery code and prints
// it before reading this FD to confirm it back; the code cannot be known
// before the child starts, which is why `pair init` (pair.go) needs this
// and `pair join` (already knowing the code from a prior `pair init`) can
// use the simpler fixedSecretFD below instead.
//
// os.Pipe on Windows creates both ends already inheritable
// (syscall.Pipe -> makeInheritSa, InheritHandle=1 -- see
// $GOROOT/src/syscall/syscall_windows.go); os/exec's own CreateProcess
// call on this OS always requests handle inheritance for any child that
// has stdin/stdout/stderr wired at all
// (syscall.StartProcess's willInheritHandles, exec_windows.go), which
// every exec.Cmd does. So the *write* end must have its inherit flag
// cleared explicitly here, or the child would hold its own copy open and
// the parent's Close would never produce EOF.
type liveSecretFD struct {
	read, write *os.File
}

func newLiveSecretFD() (*liveSecretFD, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("open the recovery-code pipe: %w", err)
	}
	if err := windows.SetHandleInformation(windows.Handle(w.Fd()), handleFlagInherit, 0); err != nil {
		_ = r.Close()
		_ = w.Close()
		return nil, fmt.Errorf("mark the recovery-code pipe's write end non-inheritable: %w", err)
	}
	return &liveSecretFD{read: r, write: w}, nil
}

// ChildValue is what REINSTATE_RECOVERY_CODE_FD must be set to before the
// child starts: the read end's own Windows HANDLE value. os/exec's
// inheritance carries it into the child unchanged -- an inherited handle
// keeps the same numeric value in the child that it had in the parent.
func (f *liveSecretFD) ChildValue() string {
	return fmt.Sprintf("%d", f.read.Fd())
}

// ApplyTo wires cmd so its child actually inherits f's read end -- call
// this before cmd.Start()/cmd.Run(). See applyInheritedHandle.
func (f *liveSecretFD) ApplyTo(cmd *exec.Cmd) {
	applyInheritedHandle(cmd, syscall.Handle(f.read.Fd()))
}

// CloseReadInParent closes the parent's own copy of the read end once the
// child has been started; the child's inherited copy is independent and
// keeps working. Not required for correctness -- a second open reader does
// not stop EOF once every write end closes -- but leaves nothing open the
// parent has no further use for.
func (f *liveSecretFD) CloseReadInParent() {
	_ = f.read.Close()
}

// Feed writes code and immediately closes the write end, which is what
// lets the child's blocking read return: readBoundedSecret
// (internal/crypto/passphrase.go) reads until EOF, and a pipe's EOF is
// "every write end is closed", not "the writer sent a terminator byte".
func (f *liveSecretFD) Feed(code string) error {
	if _, err := fmt.Fprintln(f.write, code); err != nil {
		_ = f.write.Close()
		return err
	}
	return f.write.Close()
}

// Close releases both ends. Safe to call more than once, and safe to call
// after Feed already closed the write end.
func (f *liveSecretFD) Close() {
	_ = f.read.Close()
	_ = f.write.Close()
}

// fixedSecretFD is a plain temp file carrying an already-known secret (the
// recovery code a prior `rein account init` already showed), for `rein
// account recover`'s single, unblocking read: readBoundedSecret hits a
// real end-of-file at the end of the file's content, so no pipe or live
// feed is needed here. os.CreateTemp opens the file non-inheritable by
// default (os.OpenFile always ORs in syscall.O_CLOEXEC on Windows -- see
// $GOROOT/src/os/file_windows.go), so the inheritable flag is set here
// explicitly, the same as the pipe's read end above.
type fixedSecretFD struct {
	file *os.File
	path string
}

func newFixedSecretFD(secret string) (*fixedSecretFD, error) {
	f, err := os.CreateTemp("", "reinstate-recovery-code-*")
	if err != nil {
		return nil, fmt.Errorf("create the recovery-code file: %w", err)
	}
	path := f.Name()
	if _, err := fmt.Fprintln(f, secret); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write the recovery-code file: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("rewind the recovery-code file: %w", err)
	}
	if err := windows.SetHandleInformation(windows.Handle(f.Fd()), handleFlagInherit, handleFlagInherit); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("mark the recovery-code file inheritable: %w", err)
	}
	return &fixedSecretFD{file: f, path: path}, nil
}

// ChildValue is what REINSTATE_RECOVERY_CODE_FD must be set to.
func (f *fixedSecretFD) ChildValue() string {
	return fmt.Sprintf("%d", f.file.Fd())
}

// ApplyTo wires cmd so its child actually inherits f's file handle -- call
// this before cmd.Start()/cmd.Run(). See applyInheritedHandle.
func (f *fixedSecretFD) ApplyTo(cmd *exec.Cmd) {
	applyInheritedHandle(cmd, syscall.Handle(f.file.Fd()))
}

// Close removes the temp file. It lives in the per-user temp directory,
// never under a lab root or this repository, and is removed as soon as the
// subprocess that read it has exited -- the same lifetime
// internal/cli/e2e_test.go's own REINSTATE_PASSPHRASE_FD temp file already
// gets, in-process.
func (f *fixedSecretFD) Close() {
	_ = f.file.Close()
	_ = os.Remove(f.path)
}
