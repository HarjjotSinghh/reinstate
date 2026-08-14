//go:build windows

package agentcheck

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const createNoWindow = 0x08000000

func configureVersionCommand(command *exec.Cmd) {
	command.WaitDelay = 200 * time.Millisecond
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		killWindowsProcessTree(command.Process.Pid)
		return command.Process.Kill()
	}
}

func killWindowsProcessTree(pid int) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("SYSTEMROOT")
	}
	taskkill := "taskkill"
	if root != "" {
		taskkill = filepath.Join(root, "System32", "taskkill.exe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	kill := exec.CommandContext(ctx, taskkill, "/T", "/F", "/PID", strconv.Itoa(pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	kill.Stdout = io.Discard
	kill.Stderr = io.Discard
	_ = kill.Run()
}
