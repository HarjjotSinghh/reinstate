//go:build windows

package main

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// updateProcThreadAttributePseudoConsole is UpdateProcThreadAttribute called
// directly rather than through windows.ProcThreadAttributeListContainer.Update,
// which requires the attribute value as unsafe.Pointer.
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE's value is the HPCON handle itself
// (Microsoft's own sample passes the handle, not its address, as lpValue;
// this is the small-value inlining CreateProcessW documents for attributes
// no larger than a pointer), a plain integer with no Go pointer behind it,
// so there is no sound *T to route through unsafe.Pointer for it. Calling
// the syscall directly instead, with the handle as a plain uintptr
// argument, avoids reinterpreting a non-pointer value as one -- the pattern
// go vet's unsafeptr check exists to catch, correctly, in every case except
// this documented Win32 API contract.
var (
	modkernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procUpdateProcThreadAttribute = modkernel32.NewProc("UpdateProcThreadAttribute")
)

func updateProcThreadAttributePseudoConsole(list *windows.ProcThreadAttributeList, hpc windows.Handle) error {
	r1, _, e1 := procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(list)),
		0,
		uintptr(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE),
		uintptr(hpc),
		unsafe.Sizeof(hpc),
		0,
		0,
	)
	if r1 == 0 {
		return e1
	}
	return nil
}

// winConsole is the real Console: a Windows pseudo console (ConPTY)
// attached to a child process via the extended STARTUPINFOEX /
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE mechanism documented at
// https://learn.microsoft.com/en-us/windows/console/creating-a-pseudoconsole-session.
type winConsole struct {
	hpc     windows.Handle
	process windows.Handle
	thread  windows.Handle
	ptyInW  *os.File // our copy of the console's input write end
	ptyOutR *os.File // our copy of the console's output read end

	out       chan []byte
	closeOnce sync.Once
}

// StartConsole allocates a pseudo console of the given size and runs args
// (a Go-style argv; args[0] is the program) inside it, quoted into a Win32
// command line by windows.ComposeCommandLine. dir is the child's working
// directory; empty inherits this process's. The child inherits this
// process's environment; conptydriver has no separate environment-block
// path because setting a variable with os.Setenv before calling StartConsole
// and letting CreateProcess's lpEnvironment stay nil (inherit) is simpler
// than re-encoding the environment and cannot disagree with it.
func StartConsole(args []string, cols, rows int, dir string) (Console, error) {
	cmdLine := windows.ComposeCommandLine(args)
	ptyInR, ptyInW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("allocate console input pipe: %w", err)
	}
	ptyOutR, ptyOutW, err := os.Pipe()
	if err != nil {
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		return nil, fmt.Errorf("allocate console output pipe: %w", err)
	}

	var hpc windows.Handle
	if err := windows.CreatePseudoConsole(
		windows.Coord{X: int16(cols), Y: int16(rows)},
		windows.Handle(ptyInR.Fd()), windows.Handle(ptyOutW.Fd()), 0, &hpc,
	); err != nil {
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		_ = ptyOutR.Close()
		_ = ptyOutW.Close()
		return nil, fmt.Errorf("CreatePseudoConsole: %w", err)
	}

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(hpc)
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		_ = ptyOutR.Close()
		_ = ptyOutW.Close()
		return nil, fmt.Errorf("NewProcThreadAttributeList: %w", err)
	}
	if err := updateProcThreadAttributePseudoConsole(attrList.List(), hpc); err != nil {
		attrList.Delete()
		windows.ClosePseudoConsole(hpc)
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		_ = ptyOutR.Close()
		_ = ptyOutW.Close()
		return nil, fmt.Errorf("attach PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE: %w", err)
	}

	var si windows.StartupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.ProcThreadAttributeList = attrList.List()

	cmdLinePtr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		attrList.Delete()
		windows.ClosePseudoConsole(hpc)
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		_ = ptyOutR.Close()
		_ = ptyOutW.Close()
		return nil, fmt.Errorf("command line %q: %w", cmdLine, err)
	}
	var dirPtr *uint16
	if dir != "" {
		dirPtr, err = windows.UTF16PtrFromString(dir)
		if err != nil {
			attrList.Delete()
			windows.ClosePseudoConsole(hpc)
			_ = ptyInR.Close()
			_ = ptyInW.Close()
			_ = ptyOutR.Close()
			_ = ptyOutW.Close()
			return nil, fmt.Errorf("working directory %q: %w", dir, err)
		}
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		nil, cmdLinePtr, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		nil, dirPtr, &si.StartupInfo, &pi,
	)
	// The attribute list only needs to live through CreateProcess; delete it
	// either way, success or failure.
	attrList.Delete()
	if err != nil {
		windows.ClosePseudoConsole(hpc)
		_ = ptyInR.Close()
		_ = ptyInW.Close()
		_ = ptyOutR.Close()
		_ = ptyOutW.Close()
		return nil, fmt.Errorf("CreateProcess %q: %w", cmdLine, err)
	}

	// The console (via ConPTY's internal conhost) and the child now hold
	// their own copies of the ends we handed them; our copies would
	// otherwise keep the pipe half-open after the child exits, so Read
	// never sees EOF.
	_ = ptyInR.Close()
	_ = ptyOutW.Close()

	c := &winConsole{
		hpc: hpc, process: pi.Process, thread: pi.Thread,
		ptyInW: ptyInW, ptyOutR: ptyOutR, out: make(chan []byte, 64),
	}
	go c.readLoop()
	return c, nil
}

func (c *winConsole) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := c.ptyOutR.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			c.out <- chunk
		}
		if err != nil {
			close(c.out)
			return
		}
	}
}

func (c *winConsole) Write(p []byte) (int, error) { return c.ptyInW.Write(p) }
func (c *winConsole) Output() <-chan []byte       { return c.out }

func (c *winConsole) Resize(cols, rows int) error {
	return windows.ResizePseudoConsole(c.hpc, windows.Coord{X: int16(cols), Y: int16(rows)})
}

func (c *winConsole) Kill() error {
	return windows.TerminateProcess(c.process, 1)
}

func (c *winConsole) Wait() (int, error) {
	if _, err := windows.WaitForSingleObject(c.process, windows.INFINITE); err != nil {
		return 0, fmt.Errorf("WaitForSingleObject: %w", err)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(c.process, &code); err != nil {
		return 0, fmt.Errorf("GetExitCodeProcess: %w", err)
	}
	return int(code), nil
}

func (c *winConsole) Close() error {
	c.closeOnce.Do(func() {
		// ClosePseudoConsole flushes any output still buffered and then
		// closes the console's own handles; doing this before closing our
		// input-write end lets a child that has already exited finish
		// draining rather than racing readLoop's EOF against a still-open
		// pipe.
		windows.ClosePseudoConsole(c.hpc)
		_ = c.ptyInW.Close()
		_ = windows.CloseHandle(c.process)
		_ = windows.CloseHandle(c.thread)
	})
	return nil
}
